package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
)

var (
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists")
	ErrSubscriptionNotFound      = errors.New("subscription not found")
)

type CreateSubscriptionProductInput struct {
	ProductID               string
	Quantity                int
	SelectedVariantValueIDs []string
}

type CreateSubscriptionOtherInfoInput struct {
	SalesPerson   string
	StartDate     *time.Time
	PaymentMethod string
	IsPaymentMode *bool
}

type CreateSubscriptionInput struct {
	CustomerID      string
	NextInvoiceDate time.Time
	RecurringPlanID string
	PaymentTermID   string
	QuotationID     string
	Status          string
	Products        []CreateSubscriptionProductInput
	OtherInfo       CreateSubscriptionOtherInfoInput
}

// SubscriptionService encapsulates subscription operations.
type SubscriptionService struct {
	db            *pgxpool.Pool
	quoteNotifier *SubscriptionQuoteNotifier
}

func NewSubscriptionService(db *pgxpool.Pool, quoteNotifier *SubscriptionQuoteNotifier) *SubscriptionService {
	return &SubscriptionService{
		db:            db,
		quoteNotifier: quoteNotifier,
	}
}

func normalizeSubscriptionStatus(status string) (models.SubscriptionStatus, bool) {
	normalizedStatus := strings.ToLower(strings.TrimSpace(status))
	switch normalizedStatus {
	case "draft":
		return models.SubscriptionStatusDraft, true
	case "", "quotation sent", "quotation_sent", "quotationsent":
		return models.SubscriptionStatusQuotationSent, true
	case "active":
		return models.SubscriptionStatusActive, true
	case "confirmed":
		return models.SubscriptionStatusConfirmed, true
	default:
		return "", false
	}
}

func roundToTwo(value float64) float64 {
	return math.Round(value*100) / 100
}

func validateSubscriptionInput(input CreateSubscriptionInput) (CreateSubscriptionInput, error) {
	normalizedCustomerID := strings.TrimSpace(input.CustomerID)
	if normalizedCustomerID == "" {
		return CreateSubscriptionInput{}, ValidationError{Message: "Customer is required."}
	}

	if input.NextInvoiceDate.IsZero() {
		return CreateSubscriptionInput{}, ValidationError{Message: "Next invoice date is required."}
	}

	normalizedRecurringPlanID := strings.TrimSpace(input.RecurringPlanID)
	if normalizedRecurringPlanID == "" {
		return CreateSubscriptionInput{}, ValidationError{Message: "Recurring plan is required."}
	}

	normalizedPaymentTermID := strings.TrimSpace(input.PaymentTermID)

	normalizedQuotationID := strings.TrimSpace(input.QuotationID)
	if normalizedQuotationID == "" {
		return CreateSubscriptionInput{}, ValidationError{Message: "Quotation template is required."}
	}

	normalizedStatus, validStatus := normalizeSubscriptionStatus(input.Status)
	if !validStatus {
		return CreateSubscriptionInput{}, ValidationError{Message: "Status must be Draft, Quotation Sent, Active, or Confirmed."}
	}

	normalizedSalesPerson := strings.TrimSpace(input.OtherInfo.SalesPerson)
	normalizedPaymentMethod := strings.TrimSpace(input.OtherInfo.PaymentMethod)
	var normalizedStartDate *time.Time
	if input.OtherInfo.StartDate != nil {
		dateValue := input.OtherInfo.StartDate.UTC()
		normalizedDateValue := time.Date(dateValue.Year(), dateValue.Month(), dateValue.Day(), 0, 0, 0, 0, time.UTC)
		normalizedStartDate = &normalizedDateValue
	}

	normalizedIsPaymentMode := input.OtherInfo.IsPaymentMode
	if normalizedStatus == models.SubscriptionStatusDraft {
		normalizedPaymentTermID = ""
	}

	if normalizedStatus != models.SubscriptionStatusConfirmed {
		normalizedStartDate = nil
		normalizedPaymentMethod = ""
		normalizedIsPaymentMode = nil
	} else if normalizedStartDate == nil {
		now := time.Now().UTC()
		defaultStartDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		normalizedStartDate = &defaultStartDate
	}

	normalizedProducts := make([]CreateSubscriptionProductInput, 0, len(input.Products))
	type aggregatedProductInput struct {
		quantity          int
		variantValueIDSet map[string]struct{}
	}
	aggregatedProducts := make(map[string]*aggregatedProductInput, len(input.Products))
	productOrder := make([]string, 0, len(input.Products))

	for _, product := range input.Products {
		normalizedProductID := strings.TrimSpace(product.ProductID)
		if normalizedProductID == "" {
			return CreateSubscriptionInput{}, ValidationError{Message: "Product ID cannot be empty."}
		}
		if product.Quantity < 1 {
			return CreateSubscriptionInput{}, ValidationError{Message: "Quantity must be at least 1."}
		}

		normalizedVariantValueIDs := make([]string, 0, len(product.SelectedVariantValueIDs))
		seenVariantValueIDs := make(map[string]struct{}, len(product.SelectedVariantValueIDs))
		for _, selectedVariantValueID := range product.SelectedVariantValueIDs {
			normalizedVariantValueID := strings.TrimSpace(selectedVariantValueID)
			if normalizedVariantValueID == "" {
				return CreateSubscriptionInput{}, ValidationError{Message: "Variant value ID cannot be empty."}
			}
			if _, exists := seenVariantValueIDs[normalizedVariantValueID]; exists {
				continue
			}
			seenVariantValueIDs[normalizedVariantValueID] = struct{}{}
			normalizedVariantValueIDs = append(normalizedVariantValueIDs, normalizedVariantValueID)
		}

		aggregatedProduct, exists := aggregatedProducts[normalizedProductID]
		if !exists {
			aggregatedProduct = &aggregatedProductInput{
				quantity:          0,
				variantValueIDSet: make(map[string]struct{}),
			}
			aggregatedProducts[normalizedProductID] = aggregatedProduct
			productOrder = append(productOrder, normalizedProductID)
		}

		aggregatedProduct.quantity += product.Quantity
		for _, normalizedVariantValueID := range normalizedVariantValueIDs {
			aggregatedProduct.variantValueIDSet[normalizedVariantValueID] = struct{}{}
		}
	}

	if len(aggregatedProducts) == 0 {
		return CreateSubscriptionInput{}, ValidationError{Message: "At least one product is required."}
	}

	for _, productID := range productOrder {
		aggregatedProduct := aggregatedProducts[productID]
		normalizedVariantValueIDs := make([]string, 0, len(aggregatedProduct.variantValueIDSet))
		for variantValueID := range aggregatedProduct.variantValueIDSet {
			normalizedVariantValueIDs = append(normalizedVariantValueIDs, variantValueID)
		}
		sort.Strings(normalizedVariantValueIDs)

		normalizedProducts = append(normalizedProducts, CreateSubscriptionProductInput{
			ProductID:               productID,
			Quantity:                aggregatedProduct.quantity,
			SelectedVariantValueIDs: normalizedVariantValueIDs,
		})
	}

	return CreateSubscriptionInput{
		CustomerID:      normalizedCustomerID,
		NextInvoiceDate: input.NextInvoiceDate,
		RecurringPlanID: normalizedRecurringPlanID,
		PaymentTermID:   normalizedPaymentTermID,
		QuotationID:     normalizedQuotationID,
		Status:          string(normalizedStatus),
		Products:        normalizedProducts,
		OtherInfo: CreateSubscriptionOtherInfoInput{
			SalesPerson:   normalizedSalesPerson,
			StartDate:     normalizedStartDate,
			PaymentMethod: normalizedPaymentMethod,
			IsPaymentMode: normalizedIsPaymentMode,
		},
	}, nil
}

func isSubscriptionNumberUniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23505" && strings.Contains(pgError.ConstraintName, "subscription_number")
	}

	return false
}

func isSubscriptionRecurringPlanForeignKeyViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23503" && strings.Contains(pgError.ConstraintName, "recurring_plan")
	}

	return false
}

func isSubscriptionQuotationForeignKeyViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23503" && strings.Contains(pgError.ConstraintName, "quotation")
	}

	return false
}

func isSubscriptionPaymentTermForeignKeyViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23503" && strings.Contains(pgError.ConstraintName, "payment_term")
	}

	return false
}

type subscriptionQuerier interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

type productTaxRate struct {
	TaxComputationUnit  string
	TaxComputationValue float64
}

func scanSubscriptionRow(row pgx.Row) (models.Subscription, error) {
	var subscription models.Subscription
	var recurring *string
	var plan *string
	var recurringPlanID *string
	var paymentTermID *string
	var paymentTermName *string
	var quotationID *string
	var status string
	if err := row.Scan(
		&subscription.SubscriptionID,
		&subscription.SubscriptionNumber,
		&subscription.CustomerID,
		&subscription.CustomerName,
		&subscription.NextInvoiceDate,
		&recurring,
		&plan,
		&recurringPlanID,
		&paymentTermID,
		&paymentTermName,
		&quotationID,
		&status,
		&subscription.CreatedAt,
		&subscription.UpdatedAt,
	); err != nil {
		return models.Subscription{}, err
	}

	subscription.Recurring = recurring
	subscription.Plan = plan
	subscription.RecurringPlanID = recurringPlanID
	subscription.PaymentTermID = paymentTermID
	subscription.PaymentTermName = paymentTermName
	subscription.QuotationID = quotationID
	subscription.Status = models.SubscriptionStatus(status)
	subscription.Products = []models.SubscriptionProduct{}
	return subscription, nil
}

func (service *SubscriptionService) getCustomerContactByID(ctx context.Context, customerID string) (string, string, error) {
	const query = `
		SELECT name, email
		FROM users."user"
		WHERE id = $1 AND role = 'User'`

	var customerName string
	var customerEmail string
	if err := service.db.QueryRow(ctx, query, customerID).Scan(&customerName, &customerEmail); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ValidationError{Message: "Selected customer is invalid. Please choose a user with role User."}
		}
		return "", "", fmt.Errorf("failed to validate customer: %w", err)
	}

	return customerName, customerEmail, nil
}

func (service *SubscriptionService) getSubscriptionStatusByID(ctx context.Context, querier subscriptionQuerier, subscriptionID string) (models.SubscriptionStatus, error) {
	const query = `
		SELECT status
		FROM subscription.subscriptions
		WHERE subscription_id = $1`

	var status string
	if err := querier.QueryRow(ctx, query, subscriptionID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrSubscriptionNotFound
		}
		return "", fmt.Errorf("failed to fetch subscription status: %w", err)
	}

	return models.SubscriptionStatus(status), nil
}

func (service *SubscriptionService) sendQuotationNotification(ctx context.Context, previousStatus models.SubscriptionStatus, subscription models.Subscription, customerEmail string) {
	if service.quoteNotifier == nil || !service.quoteNotifier.IsEnabled() {
		return
	}
	if subscription.Status != models.SubscriptionStatusQuotationSent {
		return
	}
	if previousStatus == models.SubscriptionStatusQuotationSent {
		return
	}

	recipientName := strings.TrimSpace(subscription.CustomerName)
	if recipientName == "" {
		recipientName = "Customer"
	}

	if err := service.quoteNotifier.SendQuotationEmail(ctx, customerEmail, recipientName, subscription); err != nil {
		log.Printf("quotation notification send failed for subscription %s: %v", subscription.SubscriptionID, err)
	}
}

func (service *SubscriptionService) ensureQuotationExists(ctx context.Context, quotationID string) error {
	const query = `
		SELECT 1
		FROM quotations.quotation
		WHERE quotation_id = $1`

	var exists int
	if err := service.db.QueryRow(ctx, query, quotationID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ValidationError{Message: "Selected quotation template is invalid."}
		}
		return fmt.Errorf("failed to validate quotation template: %w", err)
	}

	return nil
}

func (service *SubscriptionService) ensurePaymentTermExists(ctx context.Context, paymentTermID string) error {
	normalizedPaymentTermID := strings.TrimSpace(paymentTermID)
	if normalizedPaymentTermID == "" {
		return nil
	}

	const query = `
		SELECT 1
		FROM payment_term.payment_term_data
		WHERE payment_term_id = $1`

	var exists int
	if err := service.db.QueryRow(ctx, query, normalizedPaymentTermID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ValidationError{Message: "Selected payment term is invalid."}
		}
		return fmt.Errorf("failed to validate payment term: %w", err)
	}

	return nil
}

func nullableString(value string) *string {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return nil
	}

	return &trimmedValue
}

func insertSubscriptionOtherInfo(
	ctx context.Context,
	tx pgx.Tx,
	subscriptionID string,
	input CreateSubscriptionOtherInfoInput,
) (models.SubscriptionOtherInfo, error) {
	const query = `
		INSERT INTO subscription.subscription_other_info (
			subscription_id,
			sales_person,
			start_date,
			payment_method,
			is_payment_mode
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING
			subscription_other_info_id,
			subscription_id,
			sales_person,
			start_date,
			payment_method,
			is_payment_mode,
			created_at,
			updated_at`

	var otherInfo models.SubscriptionOtherInfo
	if err := tx.QueryRow(
		ctx,
		query,
		subscriptionID,
		nullableString(input.SalesPerson),
		input.StartDate,
		nullableString(input.PaymentMethod),
		input.IsPaymentMode,
	).Scan(
		&otherInfo.SubscriptionOtherInfoID,
		&otherInfo.SubscriptionID,
		&otherInfo.SalesPerson,
		&otherInfo.StartDate,
		&otherInfo.PaymentMethod,
		&otherInfo.IsPaymentMode,
		&otherInfo.CreatedAt,
		&otherInfo.UpdatedAt,
	); err != nil {
		return models.SubscriptionOtherInfo{}, fmt.Errorf("failed to save subscription other info: %w", err)
	}

	return otherInfo, nil
}

func fetchSubscriptionProductProfile(ctx context.Context, querier subscriptionQuerier, productID string, invoiceDate time.Time) (string, float64, float64, []productTaxRate, error) {
	const productQuery = `
		SELECT product_name, sales_price::float8
		FROM products.product_data
		WHERE product_id = $1`

	var productName string
	var unitPrice float64
	if err := querier.QueryRow(ctx, productQuery, productID).Scan(&productName, &unitPrice); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", 0, 0, nil, ValidationError{Message: "One or more selected products are invalid."}
		}
		return "", 0, 0, nil, fmt.Errorf("failed to fetch product profile: %w", err)
	}

	const discountQuery = `
		SELECT d.discount_unit, d.discount_value::float8
		FROM discount.product_discount pd
		JOIN discount.discount_data d ON d.discount_id = pd.discount_id
		WHERE pd.product_id = $1
		  AND d.is_active = TRUE
		  AND $2::date BETWEEN d.start_date AND d.end_date`

	discountRows, err := querier.Query(ctx, discountQuery, productID, invoiceDate)
	if err != nil {
		return "", 0, 0, nil, fmt.Errorf("failed to fetch product discounts: %w", err)
	}
	defer discountRows.Close()

	perUnitDiscount := 0.0
	for discountRows.Next() {
		var discountUnit string
		var discountValue float64
		if err := discountRows.Scan(&discountUnit, &discountValue); err != nil {
			return "", 0, 0, nil, fmt.Errorf("failed to scan product discount row: %w", err)
		}

		switch discountUnit {
		case "Percentage":
			perUnitDiscount += unitPrice * (discountValue / 100)
		case "Fixed Price":
			perUnitDiscount += discountValue
		}
	}
	if err := discountRows.Err(); err != nil {
		return "", 0, 0, nil, fmt.Errorf("failed while iterating product discount rows: %w", err)
	}

	if perUnitDiscount > unitPrice {
		perUnitDiscount = unitPrice
	}

	const taxQuery = `
		SELECT t.tax_computation_unit, t.tax_computation_value::float8
		FROM taxes.product_tax pt
		JOIN taxes.tax_data t ON t.tax_id = pt.tax_id
		WHERE pt.product_id = $1`

	taxRows, err := querier.Query(ctx, taxQuery, productID)
	if err != nil {
		return "", 0, 0, nil, fmt.Errorf("failed to fetch product taxes: %w", err)
	}
	defer taxRows.Close()

	taxes := make([]productTaxRate, 0)
	for taxRows.Next() {
		var taxRate productTaxRate
		if err := taxRows.Scan(&taxRate.TaxComputationUnit, &taxRate.TaxComputationValue); err != nil {
			return "", 0, 0, nil, fmt.Errorf("failed to scan product tax row: %w", err)
		}
		taxes = append(taxes, taxRate)
	}
	if err := taxRows.Err(); err != nil {
		return "", 0, 0, nil, fmt.Errorf("failed while iterating product tax rows: %w", err)
	}

	return productName, unitPrice, perUnitDiscount, taxes, nil
}

func resolveSubscriptionProductVariants(
	ctx context.Context,
	querier subscriptionQuerier,
	productID string,
	selectedVariantValueIDs []string,
) ([]models.SubscriptionProductVariant, float64, error) {
	if len(selectedVariantValueIDs) == 0 {
		return []models.SubscriptionProductVariant{}, 0, nil
	}

	const query = `
		SELECT
			av.attribute_value_id,
			av.attribute_id,
			a.attribute_name,
			av.attribute_value,
			av.default_extra_price::float8
		FROM attributes.attribute_values av
		JOIN attributes.attribute a ON a.attribute_id = av.attribute_id
		JOIN attributes.product_variants pv
			ON pv.attribute_id = av.attribute_id
			AND pv.product_id = $1
		WHERE av.attribute_value_id = ANY($2::uuid[])
		ORDER BY a.attribute_name ASC, av.attribute_value ASC`

	rows, err := querier.Query(ctx, query, productID, selectedVariantValueIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch selected product variants: %w", err)
	}
	defer rows.Close()

	selectedVariants := make([]models.SubscriptionProductVariant, 0, len(selectedVariantValueIDs))
	requestedValueIDSet := make(map[string]struct{}, len(selectedVariantValueIDs))
	foundValueIDSet := make(map[string]struct{}, len(selectedVariantValueIDs))
	selectedAttributeIDSet := make(map[string]struct{}, len(selectedVariantValueIDs))
	perUnitVariantExtra := 0.0

	for _, selectedVariantValueID := range selectedVariantValueIDs {
		requestedValueIDSet[selectedVariantValueID] = struct{}{}
	}

	for rows.Next() {
		var selectedVariant models.SubscriptionProductVariant
		if err := rows.Scan(
			&selectedVariant.AttributeValueID,
			&selectedVariant.AttributeID,
			&selectedVariant.AttributeName,
			&selectedVariant.AttributeValue,
			&selectedVariant.ExtraPrice,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan selected product variant row: %w", err)
		}

		if _, exists := selectedAttributeIDSet[selectedVariant.AttributeID]; exists {
			return nil, 0, ValidationError{Message: "Only one variant value can be selected per attribute for each product."}
		}

		selectedVariant.ProductID = productID
		selectedVariants = append(selectedVariants, selectedVariant)
		selectedAttributeIDSet[selectedVariant.AttributeID] = struct{}{}
		foundValueIDSet[selectedVariant.AttributeValueID] = struct{}{}
		perUnitVariantExtra += selectedVariant.ExtraPrice
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed while iterating selected product variant rows: %w", err)
	}

	if len(foundValueIDSet) != len(requestedValueIDSet) {
		return nil, 0, ValidationError{Message: "One or more selected product variants are invalid for the selected product."}
	}

	return selectedVariants, roundToTwo(perUnitVariantExtra), nil
}

func insertSubscriptionProductVariants(
	ctx context.Context,
	tx pgx.Tx,
	subscriptionProductID string,
	productID string,
	selectedVariants []models.SubscriptionProductVariant,
) ([]models.SubscriptionProductVariant, error) {
	if len(selectedVariants) == 0 {
		return []models.SubscriptionProductVariant{}, nil
	}

	const query = `
		INSERT INTO subscription.subscription_product_variants (
			subscription_product_id,
			product_id,
			attribute_id,
			attribute_value_id,
			extra_price
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING subscription_product_variant_id`

	insertedVariants := make([]models.SubscriptionProductVariant, 0, len(selectedVariants))
	for _, selectedVariant := range selectedVariants {
		variant := selectedVariant
		variant.SubscriptionProductID = subscriptionProductID

		if err := tx.QueryRow(
			ctx,
			query,
			subscriptionProductID,
			productID,
			variant.AttributeID,
			variant.AttributeValueID,
			roundToTwo(variant.ExtraPrice),
		).Scan(&variant.SubscriptionProductVariantID); err != nil {
			if isForeignKeyViolation(err) {
				return nil, ValidationError{Message: "One or more selected product variants are invalid."}
			}
			return nil, fmt.Errorf("failed to insert subscription product variants: %w", err)
		}

		insertedVariants = append(insertedVariants, variant)
	}

	return insertedVariants, nil
}

func calculateSubscriptionLineAmounts(unitPrice float64, quantity int, perUnitDiscount float64, taxes []productTaxRate) (float64, float64, float64) {
	lineSubtotal := unitPrice * float64(quantity)
	lineDiscount := perUnitDiscount * float64(quantity)
	if lineDiscount > lineSubtotal {
		lineDiscount = lineSubtotal
	}

	taxableAmount := lineSubtotal - lineDiscount
	lineTax := 0.0
	for _, tax := range taxes {
		switch tax.TaxComputationUnit {
		case "Percentage":
			lineTax += taxableAmount * (tax.TaxComputationValue / 100)
		case "Fixed Price":
			lineTax += tax.TaxComputationValue * float64(quantity)
		}
	}

	lineTotal := taxableAmount + lineTax

	return roundToTwo(lineDiscount), roundToTwo(lineTax), roundToTwo(lineTotal)
}

func insertSubscriptionProducts(
	ctx context.Context,
	tx pgx.Tx,
	subscriptionID string,
	nextInvoiceDate time.Time,
	products []CreateSubscriptionProductInput,
) ([]models.SubscriptionProduct, error) {
	const insertQuery = `
		INSERT INTO subscription.subscription_products (
			subscription_id,
			product_id,
			quantity,
			unit_price,
			discount_amount,
			tax_amount,
			total_amount
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING subscription_product_id`

	insertedProducts := make([]models.SubscriptionProduct, 0, len(products))
	for _, productInput := range products {
		productName, unitPrice, perUnitDiscount, taxRates, err := fetchSubscriptionProductProfile(ctx, tx, productInput.ProductID, nextInvoiceDate)
		if err != nil {
			return nil, err
		}

		selectedVariants, perUnitVariantExtra, err := resolveSubscriptionProductVariants(
			ctx,
			tx,
			productInput.ProductID,
			productInput.SelectedVariantValueIDs,
		)
		if err != nil {
			return nil, err
		}

		effectiveUnitPrice := roundToTwo(unitPrice + perUnitVariantExtra)
		discountAmount, taxAmount, totalAmount := calculateSubscriptionLineAmounts(effectiveUnitPrice, productInput.Quantity, perUnitDiscount, taxRates)

		var subscriptionProductID string
		if err := tx.QueryRow(
			ctx,
			insertQuery,
			subscriptionID,
			productInput.ProductID,
			productInput.Quantity,
			effectiveUnitPrice,
			discountAmount,
			taxAmount,
			totalAmount,
		).Scan(&subscriptionProductID); err != nil {
			if isForeignKeyViolation(err) {
				return nil, ValidationError{Message: "One or more selected products are invalid."}
			}
			return nil, fmt.Errorf("failed to add subscription products: %w", err)
		}

		insertedVariants, err := insertSubscriptionProductVariants(
			ctx,
			tx,
			subscriptionProductID,
			productInput.ProductID,
			selectedVariants,
		)
		if err != nil {
			return nil, err
		}

		variantExtraAmount := roundToTwo(perUnitVariantExtra * float64(productInput.Quantity))

		insertedProducts = append(insertedProducts, models.SubscriptionProduct{
			SubscriptionProductID: subscriptionProductID,
			ProductID:             productInput.ProductID,
			ProductName:           productName,
			Quantity:              productInput.Quantity,
			UnitPrice:             effectiveUnitPrice,
			VariantExtraAmount:    variantExtraAmount,
			DiscountAmount:        discountAmount,
			TaxAmount:             taxAmount,
			TotalAmount:           totalAmount,
			SelectedVariants:      insertedVariants,
		})
	}

	return insertedProducts, nil
}

func (service *SubscriptionService) CreateSubscription(ctx context.Context, input CreateSubscriptionInput) (models.Subscription, error) {
	validatedInput, err := validateSubscriptionInput(input)
	if err != nil {
		return models.Subscription{}, err
	}

	customerName, customerEmail, err := service.getCustomerContactByID(ctx, validatedInput.CustomerID)
	if err != nil {
		return models.Subscription{}, err
	}

	if err := service.ensureQuotationExists(ctx, validatedInput.QuotationID); err != nil {
		return models.Subscription{}, err
	}

	if err := service.ensurePaymentTermExists(ctx, validatedInput.PaymentTermID); err != nil {
		return models.Subscription{}, err
	}

	tx, err := service.db.Begin(ctx)
	if err != nil {
		return models.Subscription{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const query = `
		WITH inserted AS (
			INSERT INTO subscription.subscriptions (
				subscription_number,
				customer_id,
				customer_name,
				next_invoice_date,
				recurring_plan_id,
				payment_term_id,
				quotation_id,
				status
			)
			VALUES (DEFAULT, $1, $2, $3, $4, $5, $6, $7)
			RETURNING
				subscription_id,
				subscription_number,
				customer_id,
				customer_name,
				next_invoice_date,
				recurring_plan_id,
				payment_term_id,
				quotation_id,
				status,
				created_at,
				updated_at
		)
		SELECT
			i.subscription_id,
			i.subscription_number,
			i.customer_id,
			i.customer_name,
			i.next_invoice_date,
			rp.billing_period AS recurring,
			rp.recurring_name AS plan,
			i.recurring_plan_id,
			i.payment_term_id,
			pt.payment_term_name,
			i.quotation_id,
			i.status,
			i.created_at,
			i.updated_at
		FROM inserted i
		LEFT JOIN recurring_plans.recurring_plan_data rp ON rp.recurring_plan_id = i.recurring_plan_id
		LEFT JOIN payment_term.payment_term_data pt ON pt.payment_term_id = i.payment_term_id`

	subscription, err := scanSubscriptionRow(tx.QueryRow(
		ctx,
		query,
		validatedInput.CustomerID,
		customerName,
		validatedInput.NextInvoiceDate,
		validatedInput.RecurringPlanID,
		nullableString(validatedInput.PaymentTermID),
		validatedInput.QuotationID,
		validatedInput.Status,
	))
	if err != nil {
		if isSubscriptionNumberUniqueViolation(err) {
			return models.Subscription{}, ErrSubscriptionAlreadyExists
		}
		if isSubscriptionRecurringPlanForeignKeyViolation(err) {
			return models.Subscription{}, ValidationError{Message: "Selected recurring plan is invalid."}
		}
		if isSubscriptionQuotationForeignKeyViolation(err) {
			return models.Subscription{}, ValidationError{Message: "Selected quotation template is invalid."}
		}
		if isSubscriptionPaymentTermForeignKeyViolation(err) {
			return models.Subscription{}, ValidationError{Message: "Selected payment term is invalid."}
		}
		return models.Subscription{}, fmt.Errorf("failed to create subscription: %w", err)
	}

	insertedProducts, err := insertSubscriptionProducts(
		ctx,
		tx,
		subscription.SubscriptionID,
		validatedInput.NextInvoiceDate,
		validatedInput.Products,
	)
	if err != nil {
		return models.Subscription{}, err
	}

	subscription.Products = insertedProducts

	otherInfo, err := insertSubscriptionOtherInfo(ctx, tx, subscription.SubscriptionID, validatedInput.OtherInfo)
	if err != nil {
		return models.Subscription{}, err
	}
	subscription.OtherInfo = &otherInfo

	if err := tx.Commit(ctx); err != nil {
		return models.Subscription{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	service.sendQuotationNotification(ctx, "", subscription, customerEmail)

	return subscription, nil
}

func (service *SubscriptionService) ListSubscriptions(ctx context.Context, search string) ([]models.Subscription, error) {
	normalizedSearch := strings.TrimSpace(search)

	const query = `
		SELECT
			s.subscription_id,
			s.subscription_number,
			s.customer_id,
			s.customer_name,
			s.next_invoice_date,
			rp.billing_period AS recurring,
			rp.recurring_name AS plan,
			s.recurring_plan_id,
			s.payment_term_id,
			pt.payment_term_name,
			s.quotation_id,
			s.status,
			s.created_at,
			s.updated_at
		FROM subscription.subscriptions s
		LEFT JOIN recurring_plans.recurring_plan_data rp ON rp.recurring_plan_id = s.recurring_plan_id
		LEFT JOIN payment_term.payment_term_data pt ON pt.payment_term_id = s.payment_term_id
		WHERE (
			$1 = ''
			OR s.subscription_number ILIKE '%' || $1 || '%'
			OR s.customer_name ILIKE '%' || $1 || '%'
			OR rp.recurring_name ILIKE '%' || $1 || '%'
			OR rp.billing_period ILIKE '%' || $1 || '%'
			OR s.status ILIKE '%' || $1 || '%'
		)
		ORDER BY s.created_at DESC`

	rows, err := service.db.Query(ctx, query, normalizedSearch)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}
	defer rows.Close()

	subscriptions := make([]models.Subscription, 0)
	for rows.Next() {
		subscription, scanErr := scanSubscriptionRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan subscription row: %w", scanErr)
		}

		payment, paymentErr := fetchLatestSubscriptionPayment(ctx, service.db, subscription.SubscriptionID)
		if paymentErr != nil {
			return nil, paymentErr
		}
		subscription.Payment = payment

		subscriptions = append(subscriptions, subscription)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating subscription rows: %w", err)
	}

	return subscriptions, nil
}

func fetchSubscriptionProductVariants(
	ctx context.Context,
	querier subscriptionQuerier,
	subscriptionProductID string,
) ([]models.SubscriptionProductVariant, error) {
	const query = `
		SELECT
			spv.subscription_product_variant_id,
			spv.subscription_product_id,
			spv.product_id,
			spv.attribute_id,
			a.attribute_name,
			spv.attribute_value_id,
			av.attribute_value,
			spv.extra_price::float8
		FROM subscription.subscription_product_variants spv
		JOIN attributes.attribute a ON a.attribute_id = spv.attribute_id
		JOIN attributes.attribute_values av ON av.attribute_value_id = spv.attribute_value_id
		WHERE spv.subscription_product_id = $1
		ORDER BY a.attribute_name ASC, av.attribute_value ASC`

	rows, err := querier.Query(ctx, query, subscriptionProductID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscription product variants: %w", err)
	}
	defer rows.Close()

	variants := make([]models.SubscriptionProductVariant, 0)
	for rows.Next() {
		var variant models.SubscriptionProductVariant
		if err := rows.Scan(
			&variant.SubscriptionProductVariantID,
			&variant.SubscriptionProductID,
			&variant.ProductID,
			&variant.AttributeID,
			&variant.AttributeName,
			&variant.AttributeValueID,
			&variant.AttributeValue,
			&variant.ExtraPrice,
		); err != nil {
			return nil, fmt.Errorf("failed to scan subscription product variant row: %w", err)
		}

		variants = append(variants, variant)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating subscription product variant rows: %w", err)
	}

	return variants, nil
}

func fetchSubscriptionProducts(
	ctx context.Context,
	querier subscriptionQuerier,
	subscriptionID string,
) ([]models.SubscriptionProduct, error) {
	const query = `
		SELECT
			sp.subscription_product_id,
			sp.product_id,
			p.product_name,
			sp.quantity,
			sp.unit_price::float8,
			sp.discount_amount::float8,
			sp.tax_amount::float8,
			sp.total_amount::float8
		FROM subscription.subscription_products sp
		JOIN products.product_data p ON p.product_id = sp.product_id
		WHERE sp.subscription_id = $1
		ORDER BY p.product_name ASC`

	rows, err := querier.Query(ctx, query, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscription products: %w", err)
	}
	defer rows.Close()

	products := make([]models.SubscriptionProduct, 0)
	for rows.Next() {
		var product models.SubscriptionProduct
		if err := rows.Scan(
			&product.SubscriptionProductID,
			&product.ProductID,
			&product.ProductName,
			&product.Quantity,
			&product.UnitPrice,
			&product.DiscountAmount,
			&product.TaxAmount,
			&product.TotalAmount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan subscription product row: %w", err)
		}

		selectedVariants, err := fetchSubscriptionProductVariants(ctx, querier, product.SubscriptionProductID)
		if err != nil {
			return nil, err
		}

		variantExtraPerUnit := 0.0
		for _, selectedVariant := range selectedVariants {
			variantExtraPerUnit += selectedVariant.ExtraPrice
		}

		product.VariantExtraAmount = roundToTwo(variantExtraPerUnit * float64(product.Quantity))
		product.SelectedVariants = selectedVariants
		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating subscription product rows: %w", err)
	}

	return products, nil
}

func fetchSubscriptionOtherInfo(
	ctx context.Context,
	querier subscriptionQuerier,
	subscriptionID string,
) (*models.SubscriptionOtherInfo, error) {
	const query = `
		SELECT
			subscription_other_info_id,
			subscription_id,
			sales_person,
			start_date,
			payment_method,
			is_payment_mode,
			created_at,
			updated_at
		FROM subscription.subscription_other_info
		WHERE subscription_id = $1`

	var otherInfo models.SubscriptionOtherInfo
	if err := querier.QueryRow(ctx, query, subscriptionID).Scan(
		&otherInfo.SubscriptionOtherInfoID,
		&otherInfo.SubscriptionID,
		&otherInfo.SalesPerson,
		&otherInfo.StartDate,
		&otherInfo.PaymentMethod,
		&otherInfo.IsPaymentMode,
		&otherInfo.CreatedAt,
		&otherInfo.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch subscription other info: %w", err)
	}

	return &otherInfo, nil
}

func fetchLatestSubscriptionPayment(
	ctx context.Context,
	querier subscriptionQuerier,
	subscriptionID string,
) (*models.SubscriptionPayment, error) {
	const query = `
		SELECT
			payment_id,
			invoice_number,
			paypal_payment_id,
			paypal_payer_id,
			paypal_capture_id,
			paypal_status,
			payment_amount::float8,
			payment_currency,
			payment_method,
			payment_date,
			raw_payload
		FROM users.payments
		WHERE subscription_id = $1
		ORDER BY payment_date DESC
		LIMIT 1`

	var payment models.SubscriptionPayment
	var rawPayload []byte
	if err := querier.QueryRow(ctx, query, subscriptionID).Scan(
		&payment.PaymentID,
		&payment.InvoiceNumber,
		&payment.PayPalPaymentID,
		&payment.PayPalPayerID,
		&payment.PayPalCaptureID,
		&payment.PayPalStatus,
		&payment.PaymentAmount,
		&payment.PaymentCurrency,
		&payment.PaymentMethod,
		&payment.PaymentDate,
		&rawPayload,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		if strings.Contains(strings.ToLower(err.Error()), `relation "users.payments" does not exist`) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch latest subscription payment: %w", err)
	}

	if len(rawPayload) > 0 {
		payload := make(map[string]interface{})
		if err := json.Unmarshal(rawPayload, &payload); err == nil {
			payment.RawPayload = payload
		}
	}

	return &payment, nil
}

func (service *SubscriptionService) GetSubscriptionByID(ctx context.Context, subscriptionID string) (models.Subscription, error) {
	normalizedSubscriptionID := strings.TrimSpace(subscriptionID)
	if normalizedSubscriptionID == "" {
		return models.Subscription{}, ValidationError{Message: "Subscription ID is required."}
	}

	const query = `
		SELECT
			s.subscription_id,
			s.subscription_number,
			s.customer_id,
			s.customer_name,
			s.next_invoice_date,
			rp.billing_period AS recurring,
			rp.recurring_name AS plan,
			s.recurring_plan_id,
			s.payment_term_id,
			pt.payment_term_name,
			s.quotation_id,
			s.status,
			s.created_at,
			s.updated_at
		FROM subscription.subscriptions s
		LEFT JOIN recurring_plans.recurring_plan_data rp ON rp.recurring_plan_id = s.recurring_plan_id
		LEFT JOIN payment_term.payment_term_data pt ON pt.payment_term_id = s.payment_term_id
		WHERE s.subscription_id = $1`

	subscription, err := scanSubscriptionRow(service.db.QueryRow(ctx, query, normalizedSubscriptionID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Subscription{}, ErrSubscriptionNotFound
		}
		return models.Subscription{}, fmt.Errorf("failed to fetch subscription: %w", err)
	}

	products, err := fetchSubscriptionProducts(ctx, service.db, normalizedSubscriptionID)
	if err != nil {
		return models.Subscription{}, err
	}

	otherInfo, err := fetchSubscriptionOtherInfo(ctx, service.db, normalizedSubscriptionID)
	if err != nil {
		return models.Subscription{}, err
	}

	payment, err := fetchLatestSubscriptionPayment(ctx, service.db, normalizedSubscriptionID)
	if err != nil {
		return models.Subscription{}, err
	}

	subscription.Products = products
	subscription.OtherInfo = otherInfo
	subscription.Payment = payment

	return subscription, nil
}

func (service *SubscriptionService) UpdateSubscription(ctx context.Context, subscriptionID string, input CreateSubscriptionInput) (models.Subscription, error) {
	normalizedSubscriptionID := strings.TrimSpace(subscriptionID)
	if normalizedSubscriptionID == "" {
		return models.Subscription{}, ValidationError{Message: "Subscription ID is required."}
	}

	validatedInput, err := validateSubscriptionInput(input)
	if err != nil {
		return models.Subscription{}, err
	}

	customerName, customerEmail, err := service.getCustomerContactByID(ctx, validatedInput.CustomerID)
	if err != nil {
		return models.Subscription{}, err
	}

	if err := service.ensureQuotationExists(ctx, validatedInput.QuotationID); err != nil {
		return models.Subscription{}, err
	}

	if err := service.ensurePaymentTermExists(ctx, validatedInput.PaymentTermID); err != nil {
		return models.Subscription{}, err
	}

	tx, err := service.db.Begin(ctx)
	if err != nil {
		return models.Subscription{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	previousStatus, err := service.getSubscriptionStatusByID(ctx, tx, normalizedSubscriptionID)
	if err != nil {
		return models.Subscription{}, err
	}

	const query = `
		UPDATE subscription.subscriptions
		SET
			customer_id = $1,
			customer_name = $2,
			next_invoice_date = $3,
			recurring_plan_id = $4,
			payment_term_id = $5,
			quotation_id = $6,
			status = $7,
			updated_at = NOW()
		WHERE subscription_id = $8
		RETURNING subscription_id`

	var updatedSubscriptionID string
	if err := tx.QueryRow(
		ctx,
		query,
		validatedInput.CustomerID,
		customerName,
		validatedInput.NextInvoiceDate,
		validatedInput.RecurringPlanID,
		nullableString(validatedInput.PaymentTermID),
		validatedInput.QuotationID,
		validatedInput.Status,
		normalizedSubscriptionID,
	).Scan(&updatedSubscriptionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Subscription{}, ErrSubscriptionNotFound
		}
		if isSubscriptionRecurringPlanForeignKeyViolation(err) {
			return models.Subscription{}, ValidationError{Message: "Selected recurring plan is invalid."}
		}
		if isSubscriptionQuotationForeignKeyViolation(err) {
			return models.Subscription{}, ValidationError{Message: "Selected quotation template is invalid."}
		}
		if isSubscriptionPaymentTermForeignKeyViolation(err) {
			return models.Subscription{}, ValidationError{Message: "Selected payment term is invalid."}
		}
		return models.Subscription{}, fmt.Errorf("failed to update subscription: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM subscription.subscription_products WHERE subscription_id = $1`, normalizedSubscriptionID); err != nil {
		return models.Subscription{}, fmt.Errorf("failed to refresh subscription products: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM subscription.subscription_other_info WHERE subscription_id = $1`, normalizedSubscriptionID); err != nil {
		return models.Subscription{}, fmt.Errorf("failed to refresh subscription other info: %w", err)
	}

	if _, err := insertSubscriptionProducts(
		ctx,
		tx,
		normalizedSubscriptionID,
		validatedInput.NextInvoiceDate,
		validatedInput.Products,
	); err != nil {
		return models.Subscription{}, err
	}

	if _, err := insertSubscriptionOtherInfo(ctx, tx, normalizedSubscriptionID, validatedInput.OtherInfo); err != nil {
		return models.Subscription{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Subscription{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	updatedSubscription, err := service.GetSubscriptionByID(ctx, updatedSubscriptionID)
	if err != nil {
		return models.Subscription{}, err
	}

	service.sendQuotationNotification(ctx, previousStatus, updatedSubscription, customerEmail)

	return updatedSubscription, nil
}

func (service *SubscriptionService) DeleteSubscription(ctx context.Context, subscriptionID string) error {
	normalizedSubscriptionID := strings.TrimSpace(subscriptionID)
	if normalizedSubscriptionID == "" {
		return ValidationError{Message: "Subscription ID is required."}
	}

	result, err := service.db.Exec(ctx, `DELETE FROM subscription.subscriptions WHERE subscription_id = $1`, normalizedSubscriptionID)
	if err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSubscriptionNotFound
	}

	return nil
}
