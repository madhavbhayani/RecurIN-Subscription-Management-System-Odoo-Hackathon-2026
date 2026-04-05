package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
)

var (
	ErrDiscountAlreadyExists = errors.New("discount already exists")
	ErrDiscountNotFound      = errors.New("discount not found")
)

type CreateDiscountInput struct {
	DiscountName    string
	DiscountUnit    string
	DiscountValue   float64
	MinimumPurchase float64
	MaximumPurchase float64
	StartDate       time.Time
	EndDate         time.Time
	IsLimit         bool
	LimitUsers      *int
	IsActive        bool
	ProductIDs      []string
}

// DiscountService encapsulates discount operations.
type DiscountService struct {
	db *pgxpool.Pool
}

func NewDiscountService(db *pgxpool.Pool) *DiscountService {
	return &DiscountService{db: db}
}

func normalizeDiscountUnit(unit string) (string, bool) {
	normalizedUnit := strings.ToLower(strings.TrimSpace(unit))
	switch {
	case strings.Contains(normalizedUnit, "fixed") && strings.Contains(normalizedUnit, "price"):
		return models.DiscountUnitFixedPrice, true
	case strings.Contains(normalizedUnit, "percentage"):
		return models.DiscountUnitPercentage, true
	default:
		return "", false
	}
}

func validateDiscountInput(input CreateDiscountInput) (CreateDiscountInput, error) {
	discountName := strings.TrimSpace(input.DiscountName)
	if discountName == "" {
		return CreateDiscountInput{}, ValidationError{Message: "Discount name is required."}
	}

	normalizedDiscountUnit, validDiscountUnit := normalizeDiscountUnit(input.DiscountUnit)
	if !validDiscountUnit {
		return CreateDiscountInput{}, ValidationError{Message: "Discount unit must be either Percentage or Fixed Price."}
	}

	if input.DiscountValue <= 0 {
		return CreateDiscountInput{}, ValidationError{Message: "Discount value must be greater than zero."}
	}
	if normalizedDiscountUnit == models.DiscountUnitPercentage && input.DiscountValue > 100 {
		return CreateDiscountInput{}, ValidationError{Message: "Percentage discount value cannot be greater than 100."}
	}

	if input.MinimumPurchase < 0 {
		return CreateDiscountInput{}, ValidationError{Message: "Minimum purchase cannot be negative."}
	}
	if input.MaximumPurchase < 0 {
		return CreateDiscountInput{}, ValidationError{Message: "Maximum purchase cannot be negative."}
	}
	if input.MaximumPurchase < input.MinimumPurchase {
		return CreateDiscountInput{}, ValidationError{Message: "Maximum purchase must be greater than or equal to minimum purchase."}
	}

	if input.StartDate.IsZero() {
		return CreateDiscountInput{}, ValidationError{Message: "Start date is required."}
	}
	if input.EndDate.IsZero() {
		return CreateDiscountInput{}, ValidationError{Message: "End date is required."}
	}
	if input.EndDate.Before(input.StartDate) {
		return CreateDiscountInput{}, ValidationError{Message: "End date cannot be before start date."}
	}

	var normalizedLimitUsers *int
	if input.IsLimit {
		if input.LimitUsers == nil {
			return CreateDiscountInput{}, ValidationError{Message: "Limit users is required when limit is enabled."}
		}
		limitUsers := *input.LimitUsers
		if limitUsers <= 0 {
			return CreateDiscountInput{}, ValidationError{Message: "Limit users must be greater than zero."}
		}
		normalizedLimitUsers = &limitUsers
	}

	normalizedProductIDs := make([]string, 0, len(input.ProductIDs))
	seenProductIDs := make(map[string]struct{}, len(input.ProductIDs))
	for _, productID := range input.ProductIDs {
		normalizedProductID := strings.TrimSpace(productID)
		if normalizedProductID == "" {
			return CreateDiscountInput{}, ValidationError{Message: "Product ID cannot be empty."}
		}
		if _, exists := seenProductIDs[normalizedProductID]; exists {
			continue
		}
		seenProductIDs[normalizedProductID] = struct{}{}
		normalizedProductIDs = append(normalizedProductIDs, normalizedProductID)
	}

	if len(normalizedProductIDs) == 0 {
		return CreateDiscountInput{}, ValidationError{Message: "At least one product is required."}
	}

	return CreateDiscountInput{
		DiscountName:    discountName,
		DiscountUnit:    normalizedDiscountUnit,
		DiscountValue:   input.DiscountValue,
		MinimumPurchase: input.MinimumPurchase,
		MaximumPurchase: input.MaximumPurchase,
		StartDate:       input.StartDate,
		EndDate:         input.EndDate,
		IsLimit:         input.IsLimit,
		LimitUsers:      normalizedLimitUsers,
		IsActive:        input.IsActive,
		ProductIDs:      normalizedProductIDs,
	}, nil
}

func isDiscountNameUniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23505" && strings.Contains(pgError.ConstraintName, "discount_name")
	}

	return false
}

func isDiscountForeignKeyViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23503"
	}

	return false
}

func scanDiscountBaseRow(row pgx.Row) (models.Discount, error) {
	var discount models.Discount
	if err := row.Scan(
		&discount.DiscountID,
		&discount.DiscountName,
		&discount.DiscountUnit,
		&discount.DiscountValue,
		&discount.MinimumPurchase,
		&discount.MaximumPurchase,
		&discount.StartDate,
		&discount.EndDate,
		&discount.IsLimit,
		&discount.LimitUsers,
		&discount.AppliedUserCount,
		&discount.IsActive,
		&discount.CreatedAt,
		&discount.UpdatedAt,
	); err != nil {
		return models.Discount{}, err
	}

	return discount, nil
}

type discountQuerier interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
}

func fetchDiscountProducts(ctx context.Context, querier discountQuerier, discountID string) ([]models.DiscountProduct, error) {
	const query = `
		SELECT
			p.product_id,
			p.product_name,
			p.product_type,
			p.sales_price::float8
		FROM discount.discount_products dp
		JOIN products.product_data p ON p.product_id = dp.product_id
		WHERE dp.discount_id = $1
		ORDER BY p.product_name ASC`

	rows, err := querier.Query(ctx, query, discountID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch discount products: %w", err)
	}
	defer rows.Close()

	products := make([]models.DiscountProduct, 0)
	for rows.Next() {
		var product models.DiscountProduct
		if err := rows.Scan(
			&product.ProductID,
			&product.ProductName,
			&product.ProductType,
			&product.SalesPrice,
		); err != nil {
			return nil, fmt.Errorf("failed to scan discount product row: %w", err)
		}
		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating discount product rows: %w", err)
	}

	return products, nil
}

func insertDiscountProducts(ctx context.Context, tx pgx.Tx, discountID string, productIDs []string) error {
	const query = `
		INSERT INTO discount.discount_products (discount_id, product_id)
		VALUES ($1, $2)`

	for _, productID := range productIDs {
		if _, err := tx.Exec(ctx, query, discountID, productID); err != nil {
			if isDiscountForeignKeyViolation(err) {
				return ValidationError{Message: "One or more selected products are invalid."}
			}
			return fmt.Errorf("failed to assign products to discount: %w", err)
		}
	}

	return nil
}

func applyDiscountAutoInactive(discount models.Discount) models.Discount {
	if discount.IsLimit && discount.LimitUsers != nil && discount.AppliedUserCount >= *discount.LimitUsers {
		discount.IsActive = false
	}

	return discount
}

func (service *DiscountService) CreateDiscount(ctx context.Context, input CreateDiscountInput) (models.Discount, error) {
	validatedInput, err := validateDiscountInput(input)
	if err != nil {
		return models.Discount{}, err
	}

	tx, err := service.db.Begin(ctx)
	if err != nil {
		return models.Discount{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const query = `
		INSERT INTO discount.discount_data (
			discount_name,
			discount_unit,
			discount_value,
			minimum_purchase,
			maximum_purchase,
			start_date,
			end_date,
			is_limit,
			limit_users,
			is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING discount_id, discount_name, discount_unit, discount_value::float8, minimum_purchase::float8, maximum_purchase::float8, start_date, end_date, is_limit, limit_users, applied_user_count, is_active, created_at, updated_at`

	createdDiscount, err := scanDiscountBaseRow(tx.QueryRow(
		ctx,
		query,
		validatedInput.DiscountName,
		validatedInput.DiscountUnit,
		validatedInput.DiscountValue,
		validatedInput.MinimumPurchase,
		validatedInput.MaximumPurchase,
		validatedInput.StartDate,
		validatedInput.EndDate,
		validatedInput.IsLimit,
		validatedInput.LimitUsers,
		validatedInput.IsActive,
	))
	if err != nil {
		if isDiscountNameUniqueViolation(err) {
			return models.Discount{}, ErrDiscountAlreadyExists
		}
		return models.Discount{}, fmt.Errorf("failed to create discount: %w", err)
	}

	if err := insertDiscountProducts(ctx, tx, createdDiscount.DiscountID, validatedInput.ProductIDs); err != nil {
		return models.Discount{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Discount{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return service.GetDiscountByID(ctx, createdDiscount.DiscountID)
}

func (service *DiscountService) ListDiscounts(ctx context.Context, search string, page int, limit int) ([]models.Discount, int, error) {
	if limit <= 0 || limit > 30 {
		limit = 30
	}

	normalizedSearch := strings.TrimSpace(search)

	const baseQuery = `
		SELECT
			d.discount_id,
			d.discount_name,
			d.discount_unit,
			d.discount_value::float8,
			d.minimum_purchase::float8,
			d.maximum_purchase::float8,
			d.start_date,
			d.end_date,
			d.is_limit,
			d.limit_users,
			d.applied_user_count,
			d.is_active,
			d.created_at,
			d.updated_at,
			COALESCE(dp.product_count, 0)::int AS product_count
		FROM discount.discount_data d
		LEFT JOIN (
			SELECT discount_id, COUNT(*)::int AS product_count
			FROM discount.discount_products
			GROUP BY discount_id
		) dp ON dp.discount_id = d.discount_id
		WHERE (
			$1 = ''
			OR d.discount_name ILIKE '%' || $1 || '%'
			OR d.discount_unit ILIKE '%' || $1 || '%'
		)
		ORDER BY d.created_at DESC`

	totalRecords := 0
	query := baseQuery
	queryArgs := []interface{}{normalizedSearch}

	if page > 0 {
		offset := (page - 1) * limit

		const countQuery = `
			SELECT COUNT(*)
			FROM discount.discount_data d
			WHERE (
				$1 = ''
				OR d.discount_name ILIKE '%' || $1 || '%'
				OR d.discount_unit ILIKE '%' || $1 || '%'
			)`

		if err := service.db.QueryRow(ctx, countQuery, normalizedSearch).Scan(&totalRecords); err != nil {
			return nil, 0, fmt.Errorf("failed to count discounts: %w", err)
		}

		query = query + "\n\t\tLIMIT $2 OFFSET $3"
		queryArgs = append(queryArgs, limit, offset)
	}

	rows, err := service.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list discounts: %w", err)
	}
	defer rows.Close()

	discounts := make([]models.Discount, 0)
	for rows.Next() {
		var discount models.Discount
		if err := rows.Scan(
			&discount.DiscountID,
			&discount.DiscountName,
			&discount.DiscountUnit,
			&discount.DiscountValue,
			&discount.MinimumPurchase,
			&discount.MaximumPurchase,
			&discount.StartDate,
			&discount.EndDate,
			&discount.IsLimit,
			&discount.LimitUsers,
			&discount.AppliedUserCount,
			&discount.IsActive,
			&discount.CreatedAt,
			&discount.UpdatedAt,
			&discount.ProductCount,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan discount row: %w", err)
		}

		discount.Products = []models.DiscountProduct{}
		discount = applyDiscountAutoInactive(discount)
		discounts = append(discounts, discount)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed while iterating discount rows: %w", err)
	}

	if page <= 0 {
		totalRecords = len(discounts)
	}

	return discounts, totalRecords, nil
}

func (service *DiscountService) GetDiscountByID(ctx context.Context, discountID string) (models.Discount, error) {
	normalizedDiscountID := strings.TrimSpace(discountID)
	if normalizedDiscountID == "" {
		return models.Discount{}, ValidationError{Message: "Discount ID is required."}
	}

	const query = `
		SELECT
			discount_id,
			discount_name,
			discount_unit,
			discount_value::float8,
			minimum_purchase::float8,
			maximum_purchase::float8,
			start_date,
			end_date,
			is_limit,
			limit_users,
			applied_user_count,
			is_active,
			created_at,
			updated_at
		FROM discount.discount_data
		WHERE discount_id = $1`

	discount, err := scanDiscountBaseRow(service.db.QueryRow(ctx, query, normalizedDiscountID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Discount{}, ErrDiscountNotFound
		}
		return models.Discount{}, fmt.Errorf("failed to fetch discount: %w", err)
	}

	products, err := fetchDiscountProducts(ctx, service.db, normalizedDiscountID)
	if err != nil {
		return models.Discount{}, err
	}

	discount.Products = products
	discount.ProductCount = len(products)
	discount = applyDiscountAutoInactive(discount)

	return discount, nil
}

func (service *DiscountService) UpdateDiscount(ctx context.Context, discountID string, input CreateDiscountInput) (models.Discount, error) {
	normalizedDiscountID := strings.TrimSpace(discountID)
	if normalizedDiscountID == "" {
		return models.Discount{}, ValidationError{Message: "Discount ID is required."}
	}

	validatedInput, err := validateDiscountInput(input)
	if err != nil {
		return models.Discount{}, err
	}

	tx, err := service.db.Begin(ctx)
	if err != nil {
		return models.Discount{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const query = `
		UPDATE discount.discount_data
		SET
			discount_name = $1,
			discount_unit = $2,
			discount_value = $3,
			minimum_purchase = $4,
			maximum_purchase = $5,
			start_date = $6,
			end_date = $7,
			is_limit = $8,
			limit_users = $9,
			is_active = $10,
			updated_at = NOW()
		WHERE discount_id = $11
		RETURNING discount_id, discount_name, discount_unit, discount_value::float8, minimum_purchase::float8, maximum_purchase::float8, start_date, end_date, is_limit, limit_users, applied_user_count, is_active, created_at, updated_at`

	if _, err := scanDiscountBaseRow(tx.QueryRow(
		ctx,
		query,
		validatedInput.DiscountName,
		validatedInput.DiscountUnit,
		validatedInput.DiscountValue,
		validatedInput.MinimumPurchase,
		validatedInput.MaximumPurchase,
		validatedInput.StartDate,
		validatedInput.EndDate,
		validatedInput.IsLimit,
		validatedInput.LimitUsers,
		validatedInput.IsActive,
		normalizedDiscountID,
	)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Discount{}, ErrDiscountNotFound
		}
		if isDiscountNameUniqueViolation(err) {
			return models.Discount{}, ErrDiscountAlreadyExists
		}
		return models.Discount{}, fmt.Errorf("failed to update discount: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM discount.discount_products WHERE discount_id = $1`, normalizedDiscountID); err != nil {
		return models.Discount{}, fmt.Errorf("failed to refresh discount products: %w", err)
	}

	if err := insertDiscountProducts(ctx, tx, normalizedDiscountID, validatedInput.ProductIDs); err != nil {
		return models.Discount{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Discount{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return service.GetDiscountByID(ctx, normalizedDiscountID)
}

func (service *DiscountService) DeleteDiscount(ctx context.Context, discountID string) error {
	normalizedDiscountID := strings.TrimSpace(discountID)
	if normalizedDiscountID == "" {
		return ValidationError{Message: "Discount ID is required."}
	}

	result, err := service.db.Exec(ctx, `DELETE FROM discount.discount_data WHERE discount_id = $1`, normalizedDiscountID)
	if err != nil {
		return fmt.Errorf("failed to delete discount: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrDiscountNotFound
	}

	return nil
}
