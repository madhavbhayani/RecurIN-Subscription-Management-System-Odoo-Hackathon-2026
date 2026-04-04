package services

import (
	"context"
	"fmt"
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
) error {
	if service.cartService == nil || service.cartService.db == nil {
		return fmt.Errorf("paypal service is not initialized correctly")
	}

	normalizedUserID := strings.TrimSpace(userID)
	normalizedPaymentID := strings.TrimSpace(paymentID)
	normalizedCurrency := strings.ToUpper(strings.TrimSpace(currency))
	if normalizedCurrency == "" {
		normalizedCurrency = service.currencyCode
	}
	if normalizedUserID == "" || normalizedPaymentID == "" {
		return ValidationError{Message: "User ID and payment ID are required for payment persistence."}
	}

	alreadyRecorded, err := service.isPaymentAlreadyRecorded(ctx, normalizedUserID, normalizedPaymentID)
	if err != nil {
		return err
	}
	if alreadyRecorded {
		return nil
	}

	cartItems, err := service.cartService.ListCartItems(ctx, normalizedUserID)
	if err != nil {
		return err
	}

	tx, err := service.cartService.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start payment transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	customerName, err := service.fetchCustomerNameTx(ctx, tx, normalizedUserID)
	if err != nil {
		return err
	}

	groups, err := service.buildRecurringCartGroupsTx(ctx, tx, cartItems)
	if err != nil {
		return err
	}

	createdSubscriptionIDs, err := service.createConfirmedSubscriptionsTx(ctx, tx, normalizedUserID, customerName, groups)
	if err != nil {
		return err
	}

	paymentAmount := roundCurrency(amount)
	if paymentAmount < 0 {
		paymentAmount = 0
	}

	if len(createdSubscriptionIDs) == 0 {
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
			return err
		}
	} else {
		for index, subscriptionID := range createdSubscriptionIDs {
			subscriptionIDCopy := subscriptionID
			invoiceNumber := buildInvoiceNumber(normalizedPaymentID, index+1)
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
				return err
			}
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM users.cart WHERE user_id = $1`, normalizedUserID); err != nil {
		return fmt.Errorf("failed to clear cart after successful payment: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit payment transaction: %w", err)
	}

	return nil
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

	const query = `
		SELECT
			COALESCE(p.recurring_plan_id::text, ''),
			COALESCE(rp.billing_period, '')
		FROM products.product_data p
		LEFT JOIN recurring_plans.recurring_plan_data rp ON rp.recurring_plan_id = p.recurring_plan_id
		WHERE p.product_id = $1`

	for _, item := range cartItems {
		productID := strings.TrimSpace(item.ProductID)
		if productID == "" {
			continue
		}

		var recurringPlanID string
		var billingPeriod string
		if err := tx.QueryRow(ctx, query, productID).Scan(&recurringPlanID, &billingPeriod); err != nil {
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
	const query = `
		INSERT INTO users.payments (
			user_id,
			subscription_id,
			invoice_number,
			paypal_payment_id,
			paypal_payer_id,
			paypal_capture_id,
			paypal_status,
			payment_amount,
			payment_currency,
			payment_method,
			payment_date,
			raw_payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'PayPal', NOW(), $10::jsonb)`

	payload := rawPayload
	if len(payload) == 0 {
		payload = []byte("{}")
	}

	if _, err := tx.Exec(
		ctx,
		query,
		userID,
		subscriptionID,
		invoiceNumber,
		paypalPaymentID,
		nullableString(paypalPayerID),
		nullableString(paypalCaptureID),
		paypalStatus,
		paymentAmount,
		paymentCurrency,
		payload,
	); err != nil {
		return fmt.Errorf("failed to insert payment record: %w", err)
	}

	return nil
}
