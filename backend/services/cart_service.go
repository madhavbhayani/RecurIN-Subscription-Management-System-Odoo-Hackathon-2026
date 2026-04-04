package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
)

var (
	ErrCartItemNotFound = errors.New("cart item not found")
)

type AddCartItemInput struct {
	UserID                     string
	ProductID                  string
	Quantity                   int
	SelectedVariantAttributeID string
}

type cartProductProfile struct {
	ProductName   string
	ProductType   string
	RecurringName string
	BillingPeriod string
	UnitPrice     float64
}

type cartDiscountRule struct {
	DiscountUnit     string
	DiscountValue    float64
	MinimumPurchase  float64
	MaximumPurchase  float64
	IsLimit          bool
	LimitUsers       *int
	AppliedUserCount int
}

// CartService encapsulates user cart operations.
type CartService struct {
	db *pgxpool.Pool
}

func NewCartService(db *pgxpool.Pool) *CartService {
	return &CartService{db: db}
}

type cartQuerier interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

func validateAddCartItemInput(input AddCartItemInput) (AddCartItemInput, error) {
	normalizedUserID := strings.TrimSpace(input.UserID)
	if normalizedUserID == "" {
		return AddCartItemInput{}, ValidationError{Message: "User ID is required."}
	}

	normalizedProductID := strings.TrimSpace(input.ProductID)
	if normalizedProductID == "" {
		return AddCartItemInput{}, ValidationError{Message: "Product ID is required."}
	}

	normalizedVariantAttributeID := strings.TrimSpace(input.SelectedVariantAttributeID)

	if input.Quantity < 1 {
		return AddCartItemInput{}, ValidationError{Message: "Quantity must be at least 1."}
	}

	return AddCartItemInput{
		UserID:                     normalizedUserID,
		ProductID:                  normalizedProductID,
		Quantity:                   input.Quantity,
		SelectedVariantAttributeID: normalizedVariantAttributeID,
	}, nil
}

func roundCurrency(value float64) float64 {
	return math.Round(value*100) / 100
}

func nullableUUIDText(value string) interface{} {
	normalizedValue := strings.TrimSpace(value)
	if normalizedValue == "" {
		return nil
	}

	return normalizedValue
}

func fetchCartProductProfile(ctx context.Context, querier cartQuerier, productID string) (cartProductProfile, error) {
	const query = `
		SELECT
			p.product_name,
			p.product_type,
			COALESCE(rp.recurring_name, ''),
			COALESCE(rp.billing_period, ''),
			p.sales_price::float8
		FROM products.product_data p
		LEFT JOIN recurring_plans.recurring_plan_data rp ON rp.recurring_plan_id = p.recurring_plan_id
		WHERE p.product_id = $1`

	var profile cartProductProfile
	if err := querier.QueryRow(ctx, query, productID).Scan(
		&profile.ProductName,
		&profile.ProductType,
		&profile.RecurringName,
		&profile.BillingPeriod,
		&profile.UnitPrice,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return cartProductProfile{}, ValidationError{Message: "Selected product is invalid."}
		}
		return cartProductProfile{}, fmt.Errorf("failed to fetch product profile: %w", err)
	}

	return profile, nil
}

func validateVariantForProduct(ctx context.Context, querier cartQuerier, productID, attributeID string) (*string, error) {
	normalizedAttributeID := strings.TrimSpace(attributeID)
	if normalizedAttributeID == "" {
		return nil, nil
	}

	const query = `
		SELECT a.attribute_name
		FROM attributes.product_variants pv
		JOIN attributes.attribute a ON a.attribute_id = pv.attribute_id
		WHERE pv.product_id = $1
		  AND pv.attribute_id = $2`

	var attributeName string
	if err := querier.QueryRow(ctx, query, productID, normalizedAttributeID).Scan(&attributeName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ValidationError{Message: "Selected variant is not valid for this product."}
		}
		return nil, fmt.Errorf("failed to validate product variant: %w", err)
	}

	return &attributeName, nil
}

func fetchVariantExtraPriceForProduct(ctx context.Context, querier cartQuerier, productID, attributeID string) (float64, error) {
	normalizedAttributeID := strings.TrimSpace(attributeID)
	if normalizedAttributeID == "" {
		return 0, nil
	}

	const query = `
		SELECT COALESCE(MIN(av.default_extra_price), 0)::float8
		FROM attributes.product_variants pv
		LEFT JOIN attributes.attribute_values av ON av.attribute_id = pv.attribute_id
		WHERE pv.product_id = $1
		  AND pv.attribute_id = $2`

	var extraPrice float64
	if err := querier.QueryRow(ctx, query, productID, normalizedAttributeID).Scan(&extraPrice); err != nil {
		return 0, fmt.Errorf("failed to fetch variant extra price: %w", err)
	}

	if extraPrice < 0 {
		extraPrice = 0
	}

	return roundCurrency(extraPrice), nil
}

func fetchApplicableDiscountRules(ctx context.Context, querier cartQuerier, productID string, asOfDate time.Time) ([]cartDiscountRule, error) {
	const query = `
		SELECT
			d.discount_unit,
			d.discount_value::float8,
			d.minimum_purchase::float8,
			d.maximum_purchase::float8,
			d.is_limit,
			d.limit_users,
			d.applied_user_count
		FROM discount.product_discount pd
		JOIN discount.discount_data d ON d.discount_id = pd.discount_id
		WHERE pd.product_id = $1
		  AND d.is_active = TRUE
		  AND $2::date BETWEEN d.start_date AND d.end_date`

	rows, err := querier.Query(ctx, query, productID, asOfDate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch product discounts: %w", err)
	}
	defer rows.Close()

	rules := make([]cartDiscountRule, 0)
	for rows.Next() {
		var rule cartDiscountRule
		if err := rows.Scan(
			&rule.DiscountUnit,
			&rule.DiscountValue,
			&rule.MinimumPurchase,
			&rule.MaximumPurchase,
			&rule.IsLimit,
			&rule.LimitUsers,
			&rule.AppliedUserCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan product discount row: %w", err)
		}
		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating product discount rows: %w", err)
	}

	return rules, nil
}

func calculateDiscountAmount(baseAmount float64, rules []cartDiscountRule) float64 {
	if baseAmount <= 0 {
		return 0
	}

	discountAmount := 0.0
	for _, rule := range rules {
		if baseAmount < rule.MinimumPurchase || baseAmount > rule.MaximumPurchase {
			continue
		}

		if rule.IsLimit && rule.LimitUsers != nil && *rule.LimitUsers > 0 && rule.AppliedUserCount >= *rule.LimitUsers {
			continue
		}

		switch strings.TrimSpace(rule.DiscountUnit) {
		case models.DiscountUnitPercentage:
			discountAmount += baseAmount * (rule.DiscountValue / 100)
		case models.DiscountUnitFixedPrice:
			discountAmount += rule.DiscountValue
		}
	}

	if discountAmount > baseAmount {
		discountAmount = baseAmount
	}

	return roundCurrency(discountAmount)
}

func scanCartItemRow(row pgx.Row) (models.CartItem, error) {
	var item models.CartItem
	var selectedVariantAttributeID sql.NullString
	var selectedVariantAttributeName sql.NullString
	if err := row.Scan(
		&item.CartItemID,
		&item.UserID,
		&item.ProductID,
		&item.ProductName,
		&item.ProductType,
		&item.RecurringName,
		&item.BillingPeriod,
		&item.Quantity,
		&item.UnitPrice,
		&selectedVariantAttributeID,
		&selectedVariantAttributeName,
		&item.SelectedVariantPrice,
		&item.DiscountAmount,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return models.CartItem{}, err
	}

	if selectedVariantAttributeID.Valid {
		item.SelectedVariantAttributeID = &selectedVariantAttributeID.String
	}
	if selectedVariantAttributeName.Valid {
		item.SelectedVariantAttributeName = &selectedVariantAttributeName.String
	}

	item.UnitPrice = roundCurrency(item.UnitPrice)
	item.SelectedVariantPrice = roundCurrency(item.SelectedVariantPrice)
	item.DiscountAmount = roundCurrency(item.DiscountAmount)

	effectiveUnitPrice := item.UnitPrice + item.SelectedVariantPrice - item.DiscountAmount
	if effectiveUnitPrice < 0 {
		effectiveUnitPrice = 0
	}
	item.EffectiveUnitPrice = roundCurrency(effectiveUnitPrice)
	item.LineTotal = roundCurrency(item.EffectiveUnitPrice * float64(item.Quantity))

	return item, nil
}

func (service *CartService) AddCartItem(ctx context.Context, input AddCartItemInput) (models.CartItem, error) {
	validatedInput, err := validateAddCartItemInput(input)
	if err != nil {
		return models.CartItem{}, err
	}

	tx, err := service.db.Begin(ctx)
	if err != nil {
		return models.CartItem{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	profile, err := fetchCartProductProfile(ctx, tx, validatedInput.ProductID)
	if err != nil {
		return models.CartItem{}, err
	}

	if _, err := validateVariantForProduct(ctx, tx, validatedInput.ProductID, validatedInput.SelectedVariantAttributeID); err != nil {
		return models.CartItem{}, err
	}

	variantExtraPrice, err := fetchVariantExtraPriceForProduct(ctx, tx, validatedInput.ProductID, validatedInput.SelectedVariantAttributeID)
	if err != nil {
		return models.CartItem{}, err
	}

	baseAmount := profile.UnitPrice + variantExtraPrice
	discountRules, err := fetchApplicableDiscountRules(ctx, tx, validatedInput.ProductID, time.Now().UTC())
	if err != nil {
		return models.CartItem{}, err
	}
	discountAmount := calculateDiscountAmount(baseAmount, discountRules)

	const existingQuery = `
		SELECT cart_item_id, quantity
		FROM users.cart
		WHERE user_id = $1
		  AND product_id = $2
		  AND (
				($3::uuid IS NULL AND selected_variant_attribute_id IS NULL)
				OR selected_variant_attribute_id = $3::uuid
		  )
		LIMIT 1`

	var cartItemID string
	var existingQuantity int
	err = tx.QueryRow(
		ctx,
		existingQuery,
		validatedInput.UserID,
		validatedInput.ProductID,
		nullableUUIDText(validatedInput.SelectedVariantAttributeID),
	).Scan(&cartItemID, &existingQuantity)

	switch {
	case err == nil:
		const updateQuery = `
			UPDATE users.cart
			SET
				quantity = $1,
				unit_price = $2,
				selected_variant_price = $3,
				discount_amount = $4,
				billing_period = $5,
				updated_at = NOW()
			WHERE cart_item_id = $6
			RETURNING cart_item_id`

		if err := tx.QueryRow(
			ctx,
			updateQuery,
			existingQuantity+validatedInput.Quantity,
			roundCurrency(profile.UnitPrice),
			variantExtraPrice,
			discountAmount,
			strings.TrimSpace(profile.BillingPeriod),
			cartItemID,
		).Scan(&cartItemID); err != nil {
			return models.CartItem{}, fmt.Errorf("failed to update cart item: %w", err)
		}

	case errors.Is(err, pgx.ErrNoRows):
		const insertQuery = `
			INSERT INTO users.cart (
				user_id,
				product_id,
				selected_variant_attribute_id,
				quantity,
				unit_price,
				selected_variant_price,
				discount_amount,
				billing_period
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING cart_item_id`

		if err := tx.QueryRow(
			ctx,
			insertQuery,
			validatedInput.UserID,
			validatedInput.ProductID,
			nullableUUIDText(validatedInput.SelectedVariantAttributeID),
			validatedInput.Quantity,
			roundCurrency(profile.UnitPrice),
			variantExtraPrice,
			discountAmount,
			strings.TrimSpace(profile.BillingPeriod),
		).Scan(&cartItemID); err != nil {
			return models.CartItem{}, fmt.Errorf("failed to create cart item: %w", err)
		}

	default:
		return models.CartItem{}, fmt.Errorf("failed to check existing cart item: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.CartItem{}, fmt.Errorf("failed to commit cart transaction: %w", err)
	}

	return service.GetCartItemByID(ctx, validatedInput.UserID, cartItemID)
}

func (service *CartService) ListCartItems(ctx context.Context, userID string) ([]models.CartItem, error) {
	normalizedUserID := strings.TrimSpace(userID)
	if normalizedUserID == "" {
		return nil, ValidationError{Message: "User ID is required."}
	}

	const query = `
		SELECT
			c.cart_item_id,
			c.user_id,
			c.product_id,
			p.product_name,
			p.product_type,
			COALESCE(rp.recurring_name, ''),
			COALESCE(c.billing_period, rp.billing_period, ''),
			c.quantity,
			c.unit_price::float8,
			c.selected_variant_attribute_id::text,
			a.attribute_name,
			c.selected_variant_price::float8,
			c.discount_amount::float8,
			c.created_at,
			c.updated_at
		FROM users.cart c
		JOIN products.product_data p ON p.product_id = c.product_id
		LEFT JOIN recurring_plans.recurring_plan_data rp ON rp.recurring_plan_id = p.recurring_plan_id
		LEFT JOIN attributes.attribute a ON a.attribute_id = c.selected_variant_attribute_id
		WHERE c.user_id = $1
		ORDER BY c.created_at DESC`

	rows, err := service.db.Query(ctx, query, normalizedUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cart items: %w", err)
	}
	defer rows.Close()

	items := make([]models.CartItem, 0)
	for rows.Next() {
		item, scanErr := scanCartItemRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan cart item row: %w", scanErr)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating cart item rows: %w", err)
	}

	return items, nil
}

func (service *CartService) GetCartItemByID(ctx context.Context, userID, cartItemID string) (models.CartItem, error) {
	normalizedUserID := strings.TrimSpace(userID)
	normalizedCartItemID := strings.TrimSpace(cartItemID)
	if normalizedUserID == "" {
		return models.CartItem{}, ValidationError{Message: "User ID is required."}
	}
	if normalizedCartItemID == "" {
		return models.CartItem{}, ValidationError{Message: "Cart item ID is required."}
	}

	const query = `
		SELECT
			c.cart_item_id,
			c.user_id,
			c.product_id,
			p.product_name,
			p.product_type,
			COALESCE(rp.recurring_name, ''),
			COALESCE(c.billing_period, rp.billing_period, ''),
			c.quantity,
			c.unit_price::float8,
			c.selected_variant_attribute_id::text,
			a.attribute_name,
			c.selected_variant_price::float8,
			c.discount_amount::float8,
			c.created_at,
			c.updated_at
		FROM users.cart c
		JOIN products.product_data p ON p.product_id = c.product_id
		LEFT JOIN recurring_plans.recurring_plan_data rp ON rp.recurring_plan_id = p.recurring_plan_id
		LEFT JOIN attributes.attribute a ON a.attribute_id = c.selected_variant_attribute_id
		WHERE c.user_id = $1
		  AND c.cart_item_id = $2`

	item, err := scanCartItemRow(service.db.QueryRow(ctx, query, normalizedUserID, normalizedCartItemID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.CartItem{}, ErrCartItemNotFound
		}
		return models.CartItem{}, fmt.Errorf("failed to fetch cart item: %w", err)
	}

	return item, nil
}

func (service *CartService) UpdateCartItemQuantity(ctx context.Context, userID, cartItemID string, quantity int) (models.CartItem, error) {
	normalizedUserID := strings.TrimSpace(userID)
	normalizedCartItemID := strings.TrimSpace(cartItemID)
	if normalizedUserID == "" {
		return models.CartItem{}, ValidationError{Message: "User ID is required."}
	}
	if normalizedCartItemID == "" {
		return models.CartItem{}, ValidationError{Message: "Cart item ID is required."}
	}
	if quantity < 1 {
		return models.CartItem{}, ValidationError{Message: "Quantity must be at least 1."}
	}

	const query = `
		UPDATE users.cart
		SET
			quantity = $1,
			updated_at = NOW()
		WHERE cart_item_id = $2
		  AND user_id = $3
		RETURNING cart_item_id`

	var returnedCartItemID string
	if err := service.db.QueryRow(ctx, query, quantity, normalizedCartItemID, normalizedUserID).Scan(&returnedCartItemID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.CartItem{}, ErrCartItemNotFound
		}
		return models.CartItem{}, fmt.Errorf("failed to update cart item quantity: %w", err)
	}

	return service.GetCartItemByID(ctx, normalizedUserID, returnedCartItemID)
}

func (service *CartService) DeleteCartItem(ctx context.Context, userID, cartItemID string) error {
	normalizedUserID := strings.TrimSpace(userID)
	normalizedCartItemID := strings.TrimSpace(cartItemID)
	if normalizedUserID == "" {
		return ValidationError{Message: "User ID is required."}
	}
	if normalizedCartItemID == "" {
		return ValidationError{Message: "Cart item ID is required."}
	}

	result, err := service.db.Exec(ctx, `DELETE FROM users.cart WHERE cart_item_id = $1 AND user_id = $2`, normalizedCartItemID, normalizedUserID)
	if err != nil {
		return fmt.Errorf("failed to delete cart item: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrCartItemNotFound
	}

	return nil
}

func (service *CartService) ClearCart(ctx context.Context, userID string) error {
	normalizedUserID := strings.TrimSpace(userID)
	if normalizedUserID == "" {
		return ValidationError{Message: "User ID is required."}
	}

	if _, err := service.db.Exec(ctx, `DELETE FROM users.cart WHERE user_id = $1`, normalizedUserID); err != nil {
		return fmt.Errorf("failed to clear cart: %w", err)
	}

	return nil
}
