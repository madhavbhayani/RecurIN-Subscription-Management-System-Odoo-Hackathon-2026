package services

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
)

type recurringCartGroup struct {
	RecurringPlanID string
	BillingPeriod   string
	Items           []models.CartItem
}

type subscriptionProductAggregate struct {
	Quantity    int
	TotalAmount float64
}

func (service *PayPalService) persistCapturedPayment(
	ctx context.Context,
	userID string,
	paymentID string,
	payerID string,
	captureID string,
	status string,
	amount float64,
	currency string,
	rawPayload []byte,
	targetSubscriptionID string,
) ([]string, error) {
	log.Printf("[PERSIST] persistCapturedPayment called: userID=%q paymentID=%q status=%q amount=%.2f", userID, paymentID, status, amount)

	if service.cartService == nil || service.cartService.db == nil {
		log.Printf("[PERSIST] ERROR: paypal service not initialized correctly")
		return nil, fmt.Errorf("paypal service is not initialized correctly")
	}

	normalizedUserID := strings.TrimSpace(userID)
	normalizedPaymentID := strings.TrimSpace(paymentID)
	normalizedCurrency := strings.ToUpper(strings.TrimSpace(currency))
	normalizedTargetSubscriptionID := strings.TrimSpace(targetSubscriptionID)
	if normalizedCurrency == "" {
		normalizedCurrency = service.currencyCode
	}
	if normalizedUserID == "" || normalizedPaymentID == "" {
		log.Printf("[PERSIST] ERROR: missing userID or paymentID")
		return nil, ValidationError{Message: "User ID and payment ID are required for payment persistence."}
	}

	alreadyRecorded, err := service.isPaymentAlreadyRecorded(ctx, normalizedUserID, normalizedPaymentID)
	if err != nil {
		log.Printf("[PERSIST] ERROR checking if payment already recorded: %v", err)
		return nil, err
	}
	if alreadyRecorded {
		log.Printf("[PERSIST] Payment %s already recorded for user %s, skipping", normalizedPaymentID, normalizedUserID)
		return service.listSubscriptionIDsByPaymentID(ctx, normalizedUserID, normalizedPaymentID)
	}

	var cartItems []models.CartItem
	if normalizedTargetSubscriptionID == "" {
		// Get cart items BEFORE starting transaction to keep behavior unchanged for cart checkout.
		cartItems, err = service.cartService.ListCartItems(ctx, normalizedUserID)
		if err != nil {
			log.Printf("[PERSIST] ERROR fetching cart items: %v", err)
			return nil, err
		}
		log.Printf("[PERSIST] Found %d cart items for user %s", len(cartItems), normalizedUserID)
		for i, item := range cartItems {
			log.Printf("[PERSIST]   cart[%d]: productID=%s productName=%s qty=%d lineTotal=%.2f", i, item.ProductID, item.ProductName, item.Quantity, item.LineTotal)
		}
	}

	tx, err := service.cartService.db.Begin(ctx)
	if err != nil {
		log.Printf("[PERSIST] ERROR starting transaction: %v", err)
		return nil, fmt.Errorf("failed to start payment transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if normalizedTargetSubscriptionID != "" {
		log.Printf("[PERSIST] Persisting payment for existing quotation subscription %s", normalizedTargetSubscriptionID)
		subscriptionIDs, quotationPersistErr := service.persistCapturedQuotationPaymentTx(
			ctx,
			tx,
			normalizedUserID,
			normalizedTargetSubscriptionID,
			normalizedPaymentID,
			strings.TrimSpace(payerID),
			strings.TrimSpace(captureID),
			strings.TrimSpace(status),
			amount,
			normalizedCurrency,
			rawPayload,
		)
		if quotationPersistErr != nil {
			log.Printf("[PERSIST] ERROR persisting quotation payment: %v", quotationPersistErr)
			return nil, quotationPersistErr
		}

		if err := tx.Commit(ctx); err != nil {
			log.Printf("[PERSIST] ERROR committing quotation payment transaction: %v", err)
			return nil, fmt.Errorf("failed to commit quotation payment transaction: %w", err)
		}

		log.Printf("[PERSIST] ✅ Quotation payment persisted successfully: user=%s paymentID=%s subscription=%s", normalizedUserID, normalizedPaymentID, normalizedTargetSubscriptionID)
		return subscriptionIDs, nil
	}

	customerName, err := service.fetchCustomerNameTx(ctx, tx, normalizedUserID)
	if err != nil {
		log.Printf("[PERSIST] ERROR fetching customer name: %v", err)
		return nil, err
	}
	log.Printf("[PERSIST] Customer name: %q", customerName)

	// Build groups from cart items with recurring plans
	groups, err := service.buildRecurringCartGroupsTx(ctx, tx, cartItems)
	if err != nil {
		log.Printf("[PERSIST] ERROR building recurring cart groups: %v", err)
		return nil, err
	}
	log.Printf("[PERSIST] Built %d recurring cart groups", len(groups))

	// Create Active subscriptions from the groups
	createdSubscriptionIDs, err := service.createConfirmedSubscriptionsTx(ctx, tx, normalizedUserID, customerName, groups)
	if err != nil {
		log.Printf("[PERSIST] ERROR creating confirmed subscriptions: %v", err)
		return nil, err
	}
	log.Printf("[PERSIST] Created %d subscriptions from recurring groups: %v", len(createdSubscriptionIDs), createdSubscriptionIDs)

	// If no subscriptions were created (no recurring plans linked), create simple subscriptions directly from cart
	if len(createdSubscriptionIDs) == 0 && len(cartItems) > 0 {
		log.Printf("[PERSIST] No recurring plan subscriptions created, falling back to simple subscription for %d cart items", len(cartItems))
		createdSubscriptionIDs, err = service.createSimpleSubscriptionsFromCartTx(ctx, tx, normalizedUserID, customerName, cartItems)
		if err != nil {
			log.Printf("[PERSIST] ERROR creating simple subscriptions: %v", err)
			return nil, err
		}
		log.Printf("[PERSIST] Created %d simple subscriptions: %v", len(createdSubscriptionIDs), createdSubscriptionIDs)
	}

	paymentAmount := roundCurrency(amount)
	if paymentAmount < 0 {
		paymentAmount = 0
	}

	// Insert payment records
	if len(createdSubscriptionIDs) == 0 {
		log.Printf("[PERSIST] No subscriptions created, inserting standalone payment record")
		invoiceNumber := buildInvoiceNumber(normalizedPaymentID, 1)
		if err := service.insertPaymentRecordTx(
			ctx,
			tx,
			normalizedUserID,
			nil,
			invoiceNumber,
			normalizedPaymentID,
			strings.TrimSpace(payerID),
			strings.TrimSpace(captureID),
			strings.TrimSpace(status),
			paymentAmount,
			normalizedCurrency,
			rawPayload,
		); err != nil {
			log.Printf("[PERSIST] ERROR inserting standalone payment record: %v", err)
			return nil, err
		}
	} else {
		// Insert a payment record for each created subscription
		for index, subscriptionID := range createdSubscriptionIDs {
			subscriptionIDCopy := subscriptionID
			invoiceNumber := buildInvoiceNumber(normalizedPaymentID, index+1)
			log.Printf("[PERSIST] Inserting payment record for subscription %s (invoice: %s)", subscriptionIDCopy, invoiceNumber)
			if err := service.insertPaymentRecordTx(
				ctx,
				tx,
				normalizedUserID,
				&subscriptionIDCopy,
				invoiceNumber,
				normalizedPaymentID,
				strings.TrimSpace(payerID),
				strings.TrimSpace(captureID),
				strings.TrimSpace(status),
				paymentAmount,
				normalizedCurrency,
				rawPayload,
			); err != nil {
				log.Printf("[PERSIST] ERROR inserting payment record for subscription %s: %v", subscriptionIDCopy, err)
				return nil, err
			}
		}
	}

	// Clear the cart after successful payment and subscription creation
	log.Printf("[PERSIST] Clearing cart for user %s", normalizedUserID)
	if _, err := tx.Exec(ctx, `DELETE FROM users.cart WHERE user_id = $1`, normalizedUserID); err != nil {
		log.Printf("[PERSIST] ERROR clearing cart: %v", err)
		return nil, fmt.Errorf("failed to clear cart after successful payment: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[PERSIST] ERROR committing transaction: %v", err)
		return nil, fmt.Errorf("failed to commit payment transaction: %w", err)
	}

	log.Printf("[PERSIST] ✅ Payment persisted successfully: user=%s paymentID=%s subscriptions=%v", normalizedUserID, normalizedPaymentID, createdSubscriptionIDs)
	return createdSubscriptionIDs, nil
}

func (service *PayPalService) persistCapturedQuotationPaymentTx(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	subscriptionID string,
	paymentID string,
	payerID string,
	captureID string,
	status string,
	amount float64,
	currency string,
	rawPayload []byte,
) ([]string, error) {
	normalizedSubscriptionID := strings.TrimSpace(subscriptionID)
	if normalizedSubscriptionID == "" {
		return nil, ValidationError{Message: "Subscription ID is required for quotation payment."}
	}

	const lockSubscriptionQuery = `
		SELECT s.status
		FROM subscription.subscriptions s
		WHERE s.subscription_id = $1
		  AND s.customer_id = $2
		FOR UPDATE`

	const quotationTotalAmountQuery = `
		SELECT COALESCE(SUM(
			CASE
				WHEN sp.total_amount > 0 THEN sp.total_amount
				ELSE sp.unit_price * GREATEST(sp.quantity, 1)
			END
		), 0)::float8
		FROM subscription.subscription_products sp
		WHERE sp.subscription_id = $1`

	var currentStatus string
	var quotationTotalAmount float64
	if err := tx.QueryRow(ctx, lockSubscriptionQuery, normalizedSubscriptionID, userID).Scan(&currentStatus); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ValidationError{Message: "Selected quotation was not found."}
		}
		return nil, fmt.Errorf("failed to load quotation payment details: %w", err)
	}

	if err := tx.QueryRow(ctx, quotationTotalAmountQuery, normalizedSubscriptionID).Scan(&quotationTotalAmount); err != nil {
		return nil, fmt.Errorf("failed to calculate quotation total amount: %w", err)
	}

	normalizedCurrentStatus := strings.TrimSpace(currentStatus)
	if !strings.EqualFold(normalizedCurrentStatus, string(models.SubscriptionStatusQuotationSent)) {
		if strings.EqualFold(normalizedCurrentStatus, string(models.SubscriptionStatusConfirmed)) {
			return nil, ValidationError{Message: "This quotation is already confirmed."}
		}
		return nil, ValidationError{Message: fmt.Sprintf("Payment is only allowed when status is Quotation Sent (current: %s).", normalizedCurrentStatus)}
	}

	paymentAmount := roundCurrency(amount)
	if paymentAmount <= 0 {
		paymentAmount = roundCurrency(quotationTotalAmount)
	}
	if paymentAmount <= 0 {
		return nil, ValidationError{Message: "Quotation total must be greater than zero."}
	}

	normalizedCurrency := strings.ToUpper(strings.TrimSpace(currency))
	if normalizedCurrency == "" {
		normalizedCurrency = service.currencyCode
	}

	normalizedPayPalStatus := strings.TrimSpace(status)
	if normalizedPayPalStatus == "" {
		normalizedPayPalStatus = "completed"
	}

	const updateSubscriptionStatusQuery = `
		UPDATE subscription.subscriptions
		SET
			status = $1,
			updated_at = NOW()
		WHERE subscription_id = $2
		  AND customer_id = $3`

	updateResult, err := tx.Exec(ctx, updateSubscriptionStatusQuery, string(models.SubscriptionStatusConfirmed), normalizedSubscriptionID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to update quotation subscription status: %w", err)
	}
	if updateResult.RowsAffected() == 0 {
		return nil, ValidationError{Message: "Selected quotation was not found."}
	}

	startDate := time.Now().UTC()
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)

	const upsertOtherInfoQuery = `
		INSERT INTO subscription.subscription_other_info (
			subscription_id,
			start_date,
			payment_method,
			is_payment_mode
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (subscription_id)
		DO UPDATE SET
			start_date = COALESCE(subscription.subscription_other_info.start_date, EXCLUDED.start_date),
			payment_method = EXCLUDED.payment_method,
			is_payment_mode = EXCLUDED.is_payment_mode,
			updated_at = NOW()`

	if _, err := tx.Exec(ctx, upsertOtherInfoQuery, normalizedSubscriptionID, startDate, "PayPal", true); err != nil {
		return nil, fmt.Errorf("failed to update subscription payment metadata: %w", err)
	}

	invoiceNumber := buildInvoiceNumber(paymentID, 1)
	subscriptionIDCopy := normalizedSubscriptionID
	if err := service.insertPaymentRecordTx(
		ctx,
		tx,
		userID,
		&subscriptionIDCopy,
		invoiceNumber,
		paymentID,
		payerID,
		captureID,
		normalizedPayPalStatus,
		paymentAmount,
		normalizedCurrency,
		rawPayload,
	); err != nil {
		return nil, err
	}

	return []string{normalizedSubscriptionID}, nil
}

func (service *PayPalService) isPaymentAlreadyRecorded(ctx context.Context, userID, paymentID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM users.payments
			WHERE user_id = $1
			  AND paypal_payment_id = $2
		)`

	var exists bool
	if err := service.cartService.db.QueryRow(ctx, query, userID, paymentID).Scan(&exists); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), `relation "users.payments" does not exist`) {
			return false, ValidationError{Message: "users.payments schema is missing. Run backend migrations first."}
		}
		return false, fmt.Errorf("failed to check existing payment record: %w", err)
	}

	return exists, nil
}

func (service *PayPalService) listSubscriptionIDsByPaymentID(ctx context.Context, userID, paymentID string) ([]string, error) {
	const query = `
		SELECT subscription_id::text
		FROM users.payments
		WHERE user_id = $1
		  AND paypal_payment_id = $2
		  AND subscription_id IS NOT NULL
		ORDER BY payment_date DESC`

	rows, err := service.cartService.db.Query(ctx, query, userID, paymentID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), `relation "users.payments" does not exist`) {
			return nil, ValidationError{Message: "users.payments schema is missing. Run backend migrations first."}
		}
		return nil, fmt.Errorf("failed to list subscriptions by payment id: %w", err)
	}
	defer rows.Close()

	ids := make([]string, 0)
	seen := make(map[string]struct{})
	for rows.Next() {
		var subscriptionID string
		if scanErr := rows.Scan(&subscriptionID); scanErr != nil {
			return nil, fmt.Errorf("failed to scan subscription id row: %w", scanErr)
		}

		normalizedID := strings.TrimSpace(subscriptionID)
		if normalizedID == "" {
			continue
		}
		if _, exists := seen[normalizedID]; exists {
			continue
		}
		seen[normalizedID] = struct{}{}
		ids = append(ids, normalizedID)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("failed while iterating subscription id rows: %w", rowsErr)
	}

	return ids, nil
}

func (service *PayPalService) fetchCustomerNameTx(ctx context.Context, tx pgx.Tx, userID string) (string, error) {
	const query = `
		SELECT name
		FROM users."user"
		WHERE id = $1`

	var customerName string
	if err := tx.QueryRow(ctx, query, userID).Scan(&customerName); err != nil {
		if err == pgx.ErrNoRows {
			return userID, nil
		}
		return "", fmt.Errorf("failed to fetch customer details: %w", err)
	}

	trimmedName := strings.TrimSpace(customerName)
	if trimmedName == "" {
		return userID, nil
	}

	return trimmedName, nil
}

func (service *PayPalService) buildRecurringCartGroupsTx(ctx context.Context, tx pgx.Tx, cartItems []models.CartItem) (map[string]*recurringCartGroup, error) {
	groups := make(map[string]*recurringCartGroup)

	// Check if the junction table exists (it was dropped in migration 000010)
	const tableExistsQuery = `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'recurring_plans' AND table_name = 'subscription_products'
		)`

	var junctionTableExists bool
	if err := tx.QueryRow(ctx, tableExistsQuery).Scan(&junctionTableExists); err != nil {
		log.Printf("[PERSIST] Could not check junction table existence: %v", err)
		junctionTableExists = false
	}

	var lookupQuery string
	if junctionTableExists {
		lookupQuery = `
			SELECT
				COALESCE(sp.recurring_plan_id::text, ''),
				COALESCE(rp.billing_period, '')
			FROM recurring_plans.subscription_products sp
			JOIN recurring_plans.recurring_plan_data rp ON rp.recurring_plan_id = sp.recurring_plan_id
			WHERE sp.product_id = $1`
		log.Printf("[PERSIST] Using junction table for recurring plan lookup")
	} else {
		lookupQuery = `
			SELECT
				COALESCE(p.recurring_plan_id::text, ''),
				COALESCE(rp.billing_period, '')
			FROM products.product_data p
			JOIN recurring_plans.recurring_plan_data rp ON rp.recurring_plan_id = p.recurring_plan_id
			WHERE p.product_id = $1
			  AND p.recurring_plan_id IS NOT NULL`
		log.Printf("[PERSIST] Junction table not found, using products.product_data for recurring plan lookup")
	}

	for _, item := range cartItems {
		productID := strings.TrimSpace(item.ProductID)
		if productID == "" {
			continue
		}

		var recurringPlanID string
		var billingPeriod string
		if err := tx.QueryRow(ctx, lookupQuery, productID).Scan(&recurringPlanID, &billingPeriod); err != nil {
			if err == pgx.ErrNoRows {
				continue
			}
			return nil, fmt.Errorf("failed to resolve recurring metadata for product %s: %w", productID, err)
		}

		recurringPlanID = strings.TrimSpace(recurringPlanID)
		if recurringPlanID == "" {
			continue
		}

		group, exists := groups[recurringPlanID]
		if !exists {
			group = &recurringCartGroup{
				RecurringPlanID: recurringPlanID,
				BillingPeriod:   strings.TrimSpace(billingPeriod),
				Items:           make([]models.CartItem, 0, 1),
			}
			groups[recurringPlanID] = group
		}

		if group.BillingPeriod == "" {
			group.BillingPeriod = strings.TrimSpace(billingPeriod)
		}
		if group.BillingPeriod == "" {
			group.BillingPeriod = strings.TrimSpace(item.BillingPeriod)
		}

		group.Items = append(group.Items, item)
	}

	return groups, nil
}

func (service *PayPalService) createConfirmedSubscriptionsTx(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	customerName string,
	groups map[string]*recurringCartGroup,
) ([]string, error) {
	if len(groups) == 0 {
		return nil, nil
	}

	planIDs := make([]string, 0, len(groups))
	for recurringPlanID := range groups {
		planIDs = append(planIDs, recurringPlanID)
	}
	sort.Strings(planIDs)

	const insertSubscriptionQuery = `
		INSERT INTO subscription.subscriptions (
			subscription_number,
			customer_id,
			customer_name,
			next_invoice_date,
			recurring_plan_id,
			status
		)
		VALUES (DEFAULT, $1, $2, $3, $4, 'Active')
		RETURNING subscription_id`

	const insertProductQuery = `
		INSERT INTO subscription.subscription_products (
			subscription_id,
			product_id,
			quantity,
			unit_price,
			discount_amount,
			tax_amount,
			total_amount
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	const insertOtherInfoQuery = `
		INSERT INTO subscription.subscription_other_info (
			subscription_id,
			start_date,
			payment_method,
			is_payment_mode
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (subscription_id)
		DO UPDATE SET
			start_date = EXCLUDED.start_date,
			payment_method = EXCLUDED.payment_method,
			is_payment_mode = EXCLUDED.is_payment_mode,
			updated_at = NOW()`

	startDate := time.Now().UTC()
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)

	createdSubscriptionIDs := make([]string, 0, len(planIDs))
	for _, planID := range planIDs {
		group := groups[planID]
		if group == nil || len(group.Items) == 0 {
			continue
		}

		nextInvoiceDate := calculateNextInvoiceDateByBillingPeriod(startDate, group.BillingPeriod)

		var subscriptionID string
		if err := tx.QueryRow(ctx, insertSubscriptionQuery, userID, customerName, nextInvoiceDate, group.RecurringPlanID).Scan(&subscriptionID); err != nil {
			return nil, fmt.Errorf("failed to create confirmed subscription: %w", err)
		}

		productAggregates := make(map[string]*subscriptionProductAggregate)
		for _, item := range group.Items {
			productID := strings.TrimSpace(item.ProductID)
			if productID == "" {
				continue
			}

			normalizedQuantity := item.Quantity
			if normalizedQuantity < 1 {
				normalizedQuantity = 1
			}

			aggregate, exists := productAggregates[productID]
			if !exists {
				aggregate = &subscriptionProductAggregate{}
				productAggregates[productID] = aggregate
			}

			aggregate.Quantity += normalizedQuantity
			lineTotal := item.LineTotal
			if lineTotal <= 0 {
				lineTotal = item.EffectiveUnitPrice * float64(normalizedQuantity)
			}
			aggregate.TotalAmount += lineTotal
		}

		productIDs := make([]string, 0, len(productAggregates))
		for productID := range productAggregates {
			productIDs = append(productIDs, productID)
		}
		sort.Strings(productIDs)

		for _, productID := range productIDs {
			aggregate := productAggregates[productID]
			if aggregate == nil || aggregate.Quantity < 1 {
				continue
			}

			totalAmount := roundCurrency(aggregate.TotalAmount)
			unitPrice := roundCurrency(totalAmount / float64(aggregate.Quantity))
			if _, err := tx.Exec(ctx, insertProductQuery, subscriptionID, productID, aggregate.Quantity, unitPrice, 0, 0, totalAmount); err != nil {
				return nil, fmt.Errorf("failed to insert confirmed subscription product: %w", err)
			}
		}

		if _, err := tx.Exec(ctx, insertOtherInfoQuery, subscriptionID, startDate, "PayPal", true); err != nil {
			return nil, fmt.Errorf("failed to insert confirmed subscription other info: %w", err)
		}

		createdSubscriptionIDs = append(createdSubscriptionIDs, subscriptionID)
	}

	return createdSubscriptionIDs, nil
}

func (service *PayPalService) createSimpleSubscriptionsFromCartTx(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	customerName string,
	cartItems []models.CartItem,
) ([]string, error) {
	if len(cartItems) == 0 {
		return nil, nil
	}

	const insertSubscriptionQuery = `
		INSERT INTO subscription.subscriptions (
			subscription_number,
			customer_id,
			customer_name,
			next_invoice_date,
			recurring_plan_id,
			status
		)
		VALUES (DEFAULT, $1, $2, $3, $4, 'Active')
		RETURNING subscription_id`

	const insertProductQuery = `
		INSERT INTO subscription.subscription_products (
			subscription_id,
			product_id,
			quantity,
			unit_price,
			discount_amount,
			tax_amount,
			total_amount
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	const insertOtherInfoQuery = `
		INSERT INTO subscription.subscription_other_info (
			subscription_id,
			start_date,
			payment_method,
			is_payment_mode
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (subscription_id)
		DO UPDATE SET
			start_date = EXCLUDED.start_date,
			payment_method = EXCLUDED.payment_method,
			is_payment_mode = EXCLUDED.is_payment_mode,
			updated_at = NOW()`

	// Try to find recurring_plan_id from the first cart item's product
	const productRecurringPlanQuery = `
		SELECT recurring_plan_id
		FROM products.product_data
		WHERE product_id = $1
		  AND recurring_plan_id IS NOT NULL`

	startDate := time.Now().UTC()
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	nextInvoiceDate := startDate.AddDate(0, 1, 0) // Default: monthly

	// Try to resolve a recurring plan from any cart item's product
	var recurringPlanID *string
	for _, item := range cartItems {
		productID := strings.TrimSpace(item.ProductID)
		if productID == "" {
			continue
		}
		var planID string
		if err := tx.QueryRow(ctx, productRecurringPlanQuery, productID).Scan(&planID); err == nil {
			trimmedPlanID := strings.TrimSpace(planID)
			if trimmedPlanID != "" {
				recurringPlanID = &trimmedPlanID
				break
			}
		}
	}

	createdSubscriptionIDs := make([]string, 0)

	// Create one subscription with all cart items as products
	var subscriptionID string
	if err := tx.QueryRow(ctx, insertSubscriptionQuery, userID, customerName, nextInvoiceDate, recurringPlanID).Scan(&subscriptionID); err != nil {
		return nil, fmt.Errorf("failed to create subscription from cart: %w", err)
	}

	// Add all cart items as subscription products
	for _, item := range cartItems {
		productID := strings.TrimSpace(item.ProductID)
		if productID == "" {
			continue
		}

		totalAmount := roundCurrency(item.LineTotal)
		unitPrice := roundCurrency(item.EffectiveUnitPrice)

		if _, err := tx.Exec(ctx, insertProductQuery, subscriptionID, productID, item.Quantity, unitPrice, 0, 0, totalAmount); err != nil {
			return nil, fmt.Errorf("failed to insert subscription product: %w", err)
		}
	}

	// Add subscription other info
	if _, err := tx.Exec(ctx, insertOtherInfoQuery, subscriptionID, startDate, "PayPal", true); err != nil {
		return nil, fmt.Errorf("failed to insert subscription other info: %w", err)
	}

	createdSubscriptionIDs = append(createdSubscriptionIDs, subscriptionID)
	return createdSubscriptionIDs, nil
}

func calculateNextInvoiceDateByBillingPeriod(startDate time.Time, billingPeriod string) time.Time {
	normalizedPeriod := strings.ToLower(strings.TrimSpace(billingPeriod))

	switch normalizedPeriod {
	case "daily", "day":
		return startDate.AddDate(0, 0, 1)
	case "weekly", "week":
		return startDate.AddDate(0, 0, 7)
	case "monthly", "month":
		return startDate.AddDate(0, 1, 0)
	case "yearly", "year", "annual", "annually":
		return startDate.AddDate(1, 0, 0)
	default:
		return startDate.AddDate(0, 1, 0)
	}
}

func buildInvoiceNumber(paypalPaymentID string, sequence int) string {
	normalizedPaymentID := strings.ToUpper(strings.TrimSpace(paypalPaymentID))
	if normalizedPaymentID == "" {
		normalizedPaymentID = "PAYMENT"
	}
	if len(normalizedPaymentID) > 12 {
		normalizedPaymentID = normalizedPaymentID[len(normalizedPaymentID)-12:]
	}
	if sequence < 1 {
		sequence = 1
	}

	return fmt.Sprintf("INV-%s-%s-%02d", time.Now().UTC().Format("20060102"), normalizedPaymentID, sequence)
}

func (service *PayPalService) insertPaymentRecordTx(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	subscriptionID *string,
	invoiceNumber string,
	paypalPaymentID string,
	paypalPayerID string,
	paypalCaptureID string,
	paypalStatus string,
	paymentAmount float64,
	paymentCurrency string,
	rawPayload []byte,
) error {
	payload := rawPayload
	if len(payload) == 0 {
		payload = []byte("{}")
	}

	// The DB schema retains legacy amount_inr/amount_usd column names.
	// Values are stored from the captured payment amount and normalized currency metadata.
	amountINR := paymentAmount
	amountUSD := paymentAmount
	currencyFrom := "USD"
	currencyTo := "USD"

	normalizedPaymentCurrency := strings.ToUpper(strings.TrimSpace(paymentCurrency))
	if normalizedPaymentCurrency != "" {
		currencyFrom = normalizedPaymentCurrency
		currencyTo = normalizedPaymentCurrency
	}

	const query = `
		INSERT INTO users.payments (
			user_id,
			subscription_id,
			paypal_payment_id,
			paypal_payer_id,
			paypal_capture_id,
			paypal_status,
			amount_inr,
			amount_usd,
			currency_from,
			currency_to,
			payment_method,
			payment_date,
			raw_payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'PayPal', NOW(), $11::jsonb)`

	if _, err := tx.Exec(
		ctx,
		query,
		userID,
		subscriptionID,
		paypalPaymentID,
		nullableString(paypalPayerID),
		nullableString(paypalCaptureID),
		paypalStatus,
		amountINR,
		amountUSD,
		currencyFrom,
		currencyTo,
		payload,
	); err != nil {
		return fmt.Errorf("failed to insert payment record: %w", err)
	}

	return nil
}
