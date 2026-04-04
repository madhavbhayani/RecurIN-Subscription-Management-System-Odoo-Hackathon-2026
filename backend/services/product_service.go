package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
)

var (
	ErrProductAlreadyExists = errors.New("product already exists")
	ErrProductNotFound      = errors.New("product not found")
)

type CreateProductVariantInput struct {
	AttributeID string
}

type CreateProductInput struct {
	ProductName     string
	ProductType     string
	SalesPrice      float64
	CostPrice       float64
	RecurringPlanID string
	TaxIDs          []string
	DiscountIDs     []string
	Variants        []CreateProductVariantInput
}

// ProductService encapsulates product operations.
type ProductService struct {
	db *pgxpool.Pool
}

func NewProductService(db *pgxpool.Pool) *ProductService {
	return &ProductService{db: db}
}

type productQuerier interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

func normalizeProductType(productType string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(productType))
	switch normalized {
	case "service":
		return "Service", true
	case "goods", "good":
		return "Goods", true
	default:
		return "", false
	}
}

func validateProductInput(input CreateProductInput) (CreateProductInput, error) {
	productName := strings.TrimSpace(input.ProductName)
	if productName == "" {
		return CreateProductInput{}, ValidationError{Message: "Product name is required."}
	}

	normalizedType, validType := normalizeProductType(input.ProductType)
	if !validType {
		return CreateProductInput{}, ValidationError{Message: "Product type must be either Service or Goods."}
	}

	if input.SalesPrice < 0 {
		return CreateProductInput{}, ValidationError{Message: "Sales price cannot be negative."}
	}
	if input.CostPrice < 0 {
		return CreateProductInput{}, ValidationError{Message: "Cost price cannot be negative."}
	}

	normalizedRecurringPlanID := strings.TrimSpace(input.RecurringPlanID)

	normalizedTaxIDs := make([]string, 0, len(input.TaxIDs))
	seenTaxIDs := make(map[string]struct{}, len(input.TaxIDs))
	for _, taxID := range input.TaxIDs {
		normalizedTaxID := strings.TrimSpace(taxID)
		if normalizedTaxID == "" {
			return CreateProductInput{}, ValidationError{Message: "Tax ID cannot be empty."}
		}
		if _, exists := seenTaxIDs[normalizedTaxID]; exists {
			continue
		}
		seenTaxIDs[normalizedTaxID] = struct{}{}
		normalizedTaxIDs = append(normalizedTaxIDs, normalizedTaxID)
	}

	normalizedDiscountIDs := make([]string, 0, len(input.DiscountIDs))
	seenDiscountIDs := make(map[string]struct{}, len(input.DiscountIDs))
	for _, discountID := range input.DiscountIDs {
		normalizedDiscountID := strings.TrimSpace(discountID)
		if normalizedDiscountID == "" {
			return CreateProductInput{}, ValidationError{Message: "Discount ID cannot be empty."}
		}
		if _, exists := seenDiscountIDs[normalizedDiscountID]; exists {
			continue
		}
		seenDiscountIDs[normalizedDiscountID] = struct{}{}
		normalizedDiscountIDs = append(normalizedDiscountIDs, normalizedDiscountID)
	}

	normalizedVariants := make([]CreateProductVariantInput, 0, len(input.Variants))
	seenAttributeIDs := make(map[string]struct{}, len(input.Variants))
	for _, variant := range input.Variants {
		attributeID := strings.TrimSpace(variant.AttributeID)
		if attributeID == "" {
			return CreateProductInput{}, ValidationError{Message: "Variant attribute is required."}
		}

		if _, exists := seenAttributeIDs[attributeID]; exists {
			continue
		}
		seenAttributeIDs[attributeID] = struct{}{}

		normalizedVariants = append(normalizedVariants, CreateProductVariantInput{
			AttributeID: attributeID,
		})
	}

	return CreateProductInput{
		ProductName:     productName,
		ProductType:     normalizedType,
		SalesPrice:      input.SalesPrice,
		CostPrice:       input.CostPrice,
		RecurringPlanID: normalizedRecurringPlanID,
		TaxIDs:          normalizedTaxIDs,
		DiscountIDs:     normalizedDiscountIDs,
		Variants:        normalizedVariants,
	}, nil
}

func isProductNameUniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23505" && pgError.ConstraintName == "product_data_product_name_key"
	}

	return false
}

func isForeignKeyViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23503"
	}

	return false
}

func isRecurringPlanForeignKeyViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23503" && pgError.ConstraintName == "fk_product_data_recurring_plan"
	}

	return false
}

func nullableRecurringPlanID(recurringPlanID string) interface{} {
	normalizedRecurringPlanID := strings.TrimSpace(recurringPlanID)
	if normalizedRecurringPlanID == "" {
		return nil
	}

	return normalizedRecurringPlanID
}

func scanProductBaseRow(row pgx.Row) (models.Product, error) {
	var product models.Product
	var recurringPlanID sql.NullString
	var recurringName sql.NullString
	var billingPeriod sql.NullString
	if err := row.Scan(
		&product.ProductID,
		&product.ProductName,
		&product.ProductType,
		&product.SalesPrice,
		&product.CostPrice,
		&recurringPlanID,
		&recurringName,
		&billingPeriod,
		&product.CreatedAt,
		&product.UpdatedAt,
	); err != nil {
		return models.Product{}, err
	}

	if recurringPlanID.Valid {
		product.RecurringPlanID = recurringPlanID.String
	}
	if recurringName.Valid {
		product.RecurringName = recurringName.String
	}
	if billingPeriod.Valid {
		product.BillingPeriod = billingPeriod.String
	}

	return product, nil
}

func fetchProductTaxes(ctx context.Context, querier productQuerier, productID string) ([]models.ProductTax, error) {
	const query = `
		SELECT
			t.tax_id,
			t.tax_name,
			t.tax_computation_unit,
			t.tax_computation_value::float8
		FROM taxes.product_tax pt
		JOIN taxes.tax_data t ON t.tax_id = pt.tax_id
		WHERE pt.product_id = $1
		ORDER BY t.tax_name ASC`

	rows, err := querier.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch product taxes: %w", err)
	}
	defer rows.Close()

	taxes := make([]models.ProductTax, 0)
	for rows.Next() {
		var tax models.ProductTax
		if err := rows.Scan(
			&tax.TaxID,
			&tax.TaxName,
			&tax.TaxComputationUnit,
			&tax.TaxComputationValue,
		); err != nil {
			return nil, fmt.Errorf("failed to scan product tax row: %w", err)
		}
		taxes = append(taxes, tax)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating product tax rows: %w", err)
	}

	return taxes, nil
}

func fetchProductVariants(ctx context.Context, querier productQuerier, productID string) ([]models.ProductVariant, error) {
	const query = `
		SELECT
			pv.attribute_id,
			a.attribute_name,
			COALESCE(MIN(av.default_extra_price), 0)::float8 AS default_extra_price
		FROM attributes.product_variants pv
		JOIN attributes.attribute a ON a.attribute_id = pv.attribute_id
		LEFT JOIN attributes.attribute_values av ON av.attribute_id = pv.attribute_id
		WHERE pv.product_id = $1
		GROUP BY pv.attribute_id, a.attribute_name
		ORDER BY a.attribute_name ASC`

	rows, err := querier.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch product variants: %w", err)
	}
	defer rows.Close()

	variants := make([]models.ProductVariant, 0)
	for rows.Next() {
		var variant models.ProductVariant
		if err := rows.Scan(
			&variant.AttributeID,
			&variant.AttributeName,
			&variant.DefaultExtraPrice,
		); err != nil {
			return nil, fmt.Errorf("failed to scan product variant row: %w", err)
		}
		variants = append(variants, variant)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating product variant rows: %w", err)
	}

	return variants, nil
}

func fetchProductDiscounts(ctx context.Context, querier productQuerier, productID string) ([]models.ProductDiscount, error) {
	const query = `
		SELECT
			d.discount_id,
			d.discount_name,
			d.discount_unit,
			d.discount_value::float8
		FROM discount.product_discount pd
		JOIN discount.discount_data d ON d.discount_id = pd.discount_id
		WHERE pd.product_id = $1
		ORDER BY d.discount_name ASC`

	rows, err := querier.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch product discounts: %w", err)
	}
	defer rows.Close()

	discounts := make([]models.ProductDiscount, 0)
	for rows.Next() {
		var discount models.ProductDiscount
		if err := rows.Scan(
			&discount.DiscountID,
			&discount.DiscountName,
			&discount.DiscountUnit,
			&discount.DiscountValue,
		); err != nil {
			return nil, fmt.Errorf("failed to scan product discount row: %w", err)
		}
		discounts = append(discounts, discount)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating product discount rows: %w", err)
	}

	return discounts, nil
}

func insertProductTaxes(ctx context.Context, tx pgx.Tx, productID string, taxIDs []string) error {
	const query = `
		INSERT INTO taxes.product_tax (product_id, tax_id)
		VALUES ($1, $2)`

	for _, taxID := range taxIDs {
		if _, err := tx.Exec(ctx, query, productID, taxID); err != nil {
			if isForeignKeyViolation(err) {
				return ValidationError{Message: "One or more selected taxes are invalid."}
			}
			return fmt.Errorf("failed to assign taxes to product: %w", err)
		}
	}

	return nil
}

func insertProductDiscounts(ctx context.Context, tx pgx.Tx, productID string, discountIDs []string) error {
	const query = `
		INSERT INTO discount.product_discount (product_id, discount_id)
		VALUES ($1, $2)`

	for _, discountID := range discountIDs {
		if _, err := tx.Exec(ctx, query, productID, discountID); err != nil {
			if isForeignKeyViolation(err) {
				return ValidationError{Message: "One or more selected discounts are invalid."}
			}
			return fmt.Errorf("failed to assign discounts to product: %w", err)
		}
	}

	return nil
}

func insertProductVariants(ctx context.Context, tx pgx.Tx, productID string, variants []CreateProductVariantInput) error {
	const query = `
		INSERT INTO attributes.product_variants (product_id, attribute_id)
		VALUES ($1, $2)`

	for _, variant := range variants {
		if _, err := tx.Exec(ctx, query, productID, variant.AttributeID); err != nil {
			if isForeignKeyViolation(err) {
				return ValidationError{Message: "One or more selected variants are invalid."}
			}

			var pgError *pgconn.PgError
			if errors.As(err, &pgError) && pgError.Code == "23505" {
				return ValidationError{Message: "Duplicate product variants are not allowed."}
			}
			return fmt.Errorf("failed to assign variants to product: %w", err)
		}
	}

	return nil
}

func (service *ProductService) CreateProduct(ctx context.Context, input CreateProductInput) (models.Product, error) {
	validatedInput, err := validateProductInput(input)
	if err != nil {
		return models.Product{}, err
	}

	tx, err := service.db.Begin(ctx)
	if err != nil {
		return models.Product{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const query = `
		INSERT INTO products.product_data (product_name, product_type, sales_price, cost_price, recurring_plan_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING
			product_id,
			product_name,
			product_type,
			sales_price::float8,
			cost_price::float8,
			recurring_plan_id::text,
			NULL::text AS recurring_name,
			NULL::text AS billing_period,
			created_at,
			updated_at`

	createdProduct, err := scanProductBaseRow(tx.QueryRow(
		ctx,
		query,
		validatedInput.ProductName,
		validatedInput.ProductType,
		validatedInput.SalesPrice,
		validatedInput.CostPrice,
		nullableRecurringPlanID(validatedInput.RecurringPlanID),
	))
	if err != nil {
		if isProductNameUniqueViolation(err) {
			return models.Product{}, ErrProductAlreadyExists
		}
		if isRecurringPlanForeignKeyViolation(err) {
			return models.Product{}, ValidationError{Message: "Selected recurring plan is invalid."}
		}
		return models.Product{}, fmt.Errorf("failed to create product: %w", err)
	}

	if err := insertProductTaxes(ctx, tx, createdProduct.ProductID, validatedInput.TaxIDs); err != nil {
		return models.Product{}, err
	}

	if err := insertProductDiscounts(ctx, tx, createdProduct.ProductID, validatedInput.DiscountIDs); err != nil {
		return models.Product{}, err
	}

	if err := insertProductVariants(ctx, tx, createdProduct.ProductID, validatedInput.Variants); err != nil {
		return models.Product{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Product{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return service.GetProductByID(ctx, createdProduct.ProductID)
}

func (service *ProductService) ListProducts(ctx context.Context, search string) ([]models.Product, error) {
	normalizedSearch := strings.TrimSpace(search)

	const query = `
		SELECT
			p.product_id,
			p.product_name,
			p.product_type,
			p.sales_price::float8,
			p.cost_price::float8,
			p.recurring_plan_id::text,
			rp.recurring_name,
			rp.billing_period,
			p.created_at,
			p.updated_at
		FROM products.product_data p
		LEFT JOIN recurring_plans.recurring_plan_data rp ON rp.recurring_plan_id = p.recurring_plan_id
		WHERE ($1 = '' OR p.product_name ILIKE '%' || $1 || '%')
		ORDER BY p.product_name ASC`

	rows, err := service.db.Query(ctx, query, normalizedSearch)
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}
	defer rows.Close()

	products := make([]models.Product, 0)
	for rows.Next() {
		product, scanErr := scanProductBaseRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan product row: %w", scanErr)
		}

		discounts, discountErr := fetchProductDiscounts(ctx, service.db, product.ProductID)
		if discountErr != nil {
			return nil, discountErr
		}

		variants, variantErr := fetchProductVariants(ctx, service.db, product.ProductID)
		if variantErr != nil {
			return nil, variantErr
		}

		product.Discounts = discounts
		product.Variants = variants
		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating product rows: %w", err)
	}

	return products, nil
}

func (service *ProductService) GetProductByID(ctx context.Context, productID string) (models.Product, error) {
	normalizedProductID := strings.TrimSpace(productID)
	if normalizedProductID == "" {
		return models.Product{}, ValidationError{Message: "Product ID is required."}
	}

	const query = `
		SELECT
			p.product_id,
			p.product_name,
			p.product_type,
			p.sales_price::float8,
			p.cost_price::float8,
			p.recurring_plan_id::text,
			rp.recurring_name,
			rp.billing_period,
			p.created_at,
			p.updated_at
		FROM products.product_data p
		LEFT JOIN recurring_plans.recurring_plan_data rp ON rp.recurring_plan_id = p.recurring_plan_id
		WHERE p.product_id = $1`

	product, err := scanProductBaseRow(service.db.QueryRow(ctx, query, normalizedProductID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Product{}, ErrProductNotFound
		}
		return models.Product{}, fmt.Errorf("failed to fetch product: %w", err)
	}

	taxes, err := fetchProductTaxes(ctx, service.db, normalizedProductID)
	if err != nil {
		return models.Product{}, err
	}

	discounts, err := fetchProductDiscounts(ctx, service.db, normalizedProductID)
	if err != nil {
		return models.Product{}, err
	}

	variants, err := fetchProductVariants(ctx, service.db, normalizedProductID)
	if err != nil {
		return models.Product{}, err
	}

	product.Taxes = taxes
	product.Discounts = discounts
	product.Variants = variants

	return product, nil
}

func (service *ProductService) UpdateProduct(ctx context.Context, productID string, input CreateProductInput) (models.Product, error) {
	normalizedProductID := strings.TrimSpace(productID)
	if normalizedProductID == "" {
		return models.Product{}, ValidationError{Message: "Product ID is required."}
	}

	validatedInput, err := validateProductInput(input)
	if err != nil {
		return models.Product{}, err
	}

	tx, err := service.db.Begin(ctx)
	if err != nil {
		return models.Product{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const query = `
		UPDATE products.product_data
		SET
			product_name = $1,
			product_type = $2,
			sales_price = $3,
			cost_price = $4,
			recurring_plan_id = $5,
			updated_at = NOW()
		WHERE product_id = $6
		RETURNING
			product_id,
			product_name,
			product_type,
			sales_price::float8,
			cost_price::float8,
			recurring_plan_id::text,
			(
				SELECT rp.recurring_name
				FROM recurring_plans.recurring_plan_data rp
				WHERE rp.recurring_plan_id = products.product_data.recurring_plan_id
			) AS recurring_name,
			(
				SELECT rp.billing_period
				FROM recurring_plans.recurring_plan_data rp
				WHERE rp.recurring_plan_id = products.product_data.recurring_plan_id
			) AS billing_period,
			created_at,
			updated_at`

	if _, err := scanProductBaseRow(tx.QueryRow(
		ctx,
		query,
		validatedInput.ProductName,
		validatedInput.ProductType,
		validatedInput.SalesPrice,
		validatedInput.CostPrice,
		nullableRecurringPlanID(validatedInput.RecurringPlanID),
		normalizedProductID,
	)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Product{}, ErrProductNotFound
		}
		if isProductNameUniqueViolation(err) {
			return models.Product{}, ErrProductAlreadyExists
		}
		if isRecurringPlanForeignKeyViolation(err) {
			return models.Product{}, ValidationError{Message: "Selected recurring plan is invalid."}
		}
		return models.Product{}, fmt.Errorf("failed to update product: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM taxes.product_tax WHERE product_id = $1`, normalizedProductID); err != nil {
		return models.Product{}, fmt.Errorf("failed to refresh product taxes: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM discount.product_discount WHERE product_id = $1`, normalizedProductID); err != nil {
		return models.Product{}, fmt.Errorf("failed to refresh product discounts: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM attributes.product_variants WHERE product_id = $1`, normalizedProductID); err != nil {
		return models.Product{}, fmt.Errorf("failed to refresh product variants: %w", err)
	}

	if err := insertProductTaxes(ctx, tx, normalizedProductID, validatedInput.TaxIDs); err != nil {
		return models.Product{}, err
	}

	if err := insertProductDiscounts(ctx, tx, normalizedProductID, validatedInput.DiscountIDs); err != nil {
		return models.Product{}, err
	}

	if err := insertProductVariants(ctx, tx, normalizedProductID, validatedInput.Variants); err != nil {
		return models.Product{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Product{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return service.GetProductByID(ctx, normalizedProductID)
}

func (service *ProductService) DeleteProduct(ctx context.Context, productID string) error {
	normalizedProductID := strings.TrimSpace(productID)
	if normalizedProductID == "" {
		return ValidationError{Message: "Product ID is required."}
	}

	result, err := service.db.Exec(ctx, `DELETE FROM products.product_data WHERE product_id = $1`, normalizedProductID)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrProductNotFound
	}

	return nil
}
