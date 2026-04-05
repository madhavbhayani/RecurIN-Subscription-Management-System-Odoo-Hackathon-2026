package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
)

var (
	ErrQuotationNotFound = errors.New("quotation not found")
)

type CreateQuotationInput struct {
	LastForever           bool
	QuotationValidityDays *int
	RecurringPlanID       string
	ProductIDs            []string
}

// QuotationService encapsulates quotation operations.
type QuotationService struct {
	db *pgxpool.Pool
}

func NewQuotationService(db *pgxpool.Pool) *QuotationService {
	return &QuotationService{db: db}
}

func validateQuotationInput(input CreateQuotationInput) (CreateQuotationInput, error) {
	recurringPlanID := strings.TrimSpace(input.RecurringPlanID)
	if recurringPlanID == "" {
		return CreateQuotationInput{}, ValidationError{Message: "Recurring plan is required."}
	}

	var normalizedValidityDays *int
	if input.LastForever {
		normalizedValidityDays = nil
	} else {
		if input.QuotationValidityDays == nil {
			return CreateQuotationInput{}, ValidationError{Message: "Quotation validity (in days) is required when Last Forever is disabled."}
		}

		validityDays := *input.QuotationValidityDays
		if validityDays <= 0 {
			return CreateQuotationInput{}, ValidationError{Message: "Quotation validity (in days) must be greater than zero."}
		}

		normalizedValidityDays = &validityDays
	}

	normalizedProductIDs := make([]string, 0, len(input.ProductIDs))
	seenProductIDs := make(map[string]struct{}, len(input.ProductIDs))
	for _, productID := range input.ProductIDs {
		normalizedProductID := strings.TrimSpace(productID)
		if normalizedProductID == "" {
			return CreateQuotationInput{}, ValidationError{Message: "Product ID cannot be empty."}
		}
		if _, exists := seenProductIDs[normalizedProductID]; exists {
			continue
		}

		seenProductIDs[normalizedProductID] = struct{}{}
		normalizedProductIDs = append(normalizedProductIDs, normalizedProductID)
	}

	if len(normalizedProductIDs) == 0 {
		return CreateQuotationInput{}, ValidationError{Message: "At least one product is required."}
	}

	return CreateQuotationInput{
		LastForever:           input.LastForever,
		QuotationValidityDays: normalizedValidityDays,
		RecurringPlanID:       recurringPlanID,
		ProductIDs:            normalizedProductIDs,
	}, nil
}

func isQuotationForeignKeyViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23503"
	}

	return false
}

func scanQuotationBaseRow(row pgx.Row) (models.Quotation, error) {
	var quotation models.Quotation
	q := &quotation

	if err := row.Scan(
		&q.QuotationID,
		&q.LastForever,
		&q.QuotationValidityDays,
		&q.RecurringPlanID,
		&q.RecurringPlanName,
		&q.CreatedAt,
		&q.UpdatedAt,
	); err != nil {
		return models.Quotation{}, err
	}

	return quotation, nil
}

type quotationQuerier interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
}

func fetchQuotationProducts(ctx context.Context, querier quotationQuerier, quotationID string) ([]models.QuotationProduct, error) {
	const query = `
		SELECT
			p.product_id,
			p.product_name,
			p.product_type,
			p.sales_price::float8
		FROM quotations.quotations_products qp
		JOIN products.product_data p ON p.product_id = qp.product_id
		WHERE qp.quotation_id = $1
		ORDER BY p.product_name ASC`

	rows, err := querier.Query(ctx, query, quotationID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quotation products: %w", err)
	}
	defer rows.Close()

	products := make([]models.QuotationProduct, 0)
	for rows.Next() {
		var product models.QuotationProduct
		if err := rows.Scan(
			&product.ProductID,
			&product.ProductName,
			&product.ProductType,
			&product.SalesPrice,
		); err != nil {
			return nil, fmt.Errorf("failed to scan quotation product row: %w", err)
		}
		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating quotation product rows: %w", err)
	}

	return products, nil
}

func insertQuotationProducts(ctx context.Context, tx pgx.Tx, quotationID string, productIDs []string) error {
	const query = `
		INSERT INTO quotations.quotations_products (quotation_id, product_id)
		VALUES ($1, $2)`

	for _, productID := range productIDs {
		if _, err := tx.Exec(ctx, query, quotationID, productID); err != nil {
			if isQuotationForeignKeyViolation(err) {
				return ValidationError{Message: "One or more selected products are invalid."}
			}
			return fmt.Errorf("failed to assign products to quotation: %w", err)
		}
	}

	return nil
}

func (service *QuotationService) ensureRecurringPlanExists(ctx context.Context, recurringPlanID string) error {
	const query = `
		SELECT 1
		FROM recurring_plans.recurring_plan_data
		WHERE recurring_plan_id = $1`

	var exists int
	if err := service.db.QueryRow(ctx, query, recurringPlanID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ValidationError{Message: "Selected recurring plan does not exist."}
		}
		return fmt.Errorf("failed to validate recurring plan: %w", err)
	}

	return nil
}

func (service *QuotationService) CreateQuotation(ctx context.Context, input CreateQuotationInput) (models.Quotation, error) {
	validatedInput, err := validateQuotationInput(input)
	if err != nil {
		return models.Quotation{}, err
	}

	if err := service.ensureRecurringPlanExists(ctx, validatedInput.RecurringPlanID); err != nil {
		return models.Quotation{}, err
	}

	tx, err := service.db.Begin(ctx)
	if err != nil {
		return models.Quotation{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const query = `
		INSERT INTO quotations.quotation (
			last_forever,
			quotation_validity_days,
			recurring_plan_id
		)
		VALUES ($1, $2, $3)
		RETURNING quotation_id`

	var quotationID string
	if err := tx.QueryRow(
		ctx,
		query,
		validatedInput.LastForever,
		validatedInput.QuotationValidityDays,
		validatedInput.RecurringPlanID,
	).Scan(&quotationID); err != nil {
		return models.Quotation{}, fmt.Errorf("failed to create quotation: %w", err)
	}

	if err := insertQuotationProducts(ctx, tx, quotationID, validatedInput.ProductIDs); err != nil {
		return models.Quotation{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Quotation{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return service.GetQuotationByID(ctx, quotationID)
}

func (service *QuotationService) ListQuotations(ctx context.Context, search string, page int, limit int) ([]models.Quotation, int, error) {
	if limit <= 0 || limit > 30 {
		limit = 30
	}

	normalizedSearch := strings.TrimSpace(search)

	const baseQuery = `
		SELECT
			q.quotation_id,
			q.last_forever,
			q.quotation_validity_days,
			q.recurring_plan_id,
			rp.recurring_name,
			q.created_at,
			q.updated_at,
			COALESCE(qp.product_count, 0)::int AS product_count
		FROM quotations.quotation q
		JOIN recurring_plans.recurring_plan_data rp
			ON rp.recurring_plan_id = q.recurring_plan_id
		LEFT JOIN (
			SELECT quotation_id, COUNT(*)::int AS product_count
			FROM quotations.quotations_products
			GROUP BY quotation_id
		) qp ON qp.quotation_id = q.quotation_id
		WHERE (
			$1 = ''
			OR rp.recurring_name ILIKE '%' || $1 || '%'
			OR q.quotation_id::text ILIKE '%' || $1 || '%'
		)
		ORDER BY q.created_at DESC`

	totalRecords := 0
	query := baseQuery
	queryArgs := []interface{}{normalizedSearch}

	if page > 0 {
		offset := (page - 1) * limit

		const countQuery = `
			SELECT COUNT(*)
			FROM quotations.quotation q
			JOIN recurring_plans.recurring_plan_data rp
				ON rp.recurring_plan_id = q.recurring_plan_id
			WHERE (
				$1 = ''
				OR rp.recurring_name ILIKE '%' || $1 || '%'
				OR q.quotation_id::text ILIKE '%' || $1 || '%'
			)`

		if err := service.db.QueryRow(ctx, countQuery, normalizedSearch).Scan(&totalRecords); err != nil {
			return nil, 0, fmt.Errorf("failed to count quotations: %w", err)
		}

		query = query + "\n\t\tLIMIT $2 OFFSET $3"
		queryArgs = append(queryArgs, limit, offset)
	}

	rows, err := service.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list quotations: %w", err)
	}
	defer rows.Close()

	quotations := make([]models.Quotation, 0)
	for rows.Next() {
		var quotation models.Quotation
		if err := rows.Scan(
			&quotation.QuotationID,
			&quotation.LastForever,
			&quotation.QuotationValidityDays,
			&quotation.RecurringPlanID,
			&quotation.RecurringPlanName,
			&quotation.CreatedAt,
			&quotation.UpdatedAt,
			&quotation.ProductCount,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan quotation row: %w", err)
		}

		quotation.Products = []models.QuotationProduct{}
		quotations = append(quotations, quotation)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed while iterating quotation rows: %w", err)
	}

	if page <= 0 {
		totalRecords = len(quotations)
	}

	return quotations, totalRecords, nil
}

func (service *QuotationService) GetQuotationByID(ctx context.Context, quotationID string) (models.Quotation, error) {
	normalizedQuotationID := strings.TrimSpace(quotationID)
	if normalizedQuotationID == "" {
		return models.Quotation{}, ValidationError{Message: "Quotation ID is required."}
	}

	const query = `
		SELECT
			q.quotation_id,
			q.last_forever,
			q.quotation_validity_days,
			q.recurring_plan_id,
			rp.recurring_name,
			q.created_at,
			q.updated_at
		FROM quotations.quotation q
		JOIN recurring_plans.recurring_plan_data rp
			ON rp.recurring_plan_id = q.recurring_plan_id
		WHERE q.quotation_id = $1`

	quotation, err := scanQuotationBaseRow(service.db.QueryRow(ctx, query, normalizedQuotationID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Quotation{}, ErrQuotationNotFound
		}
		return models.Quotation{}, fmt.Errorf("failed to fetch quotation: %w", err)
	}

	products, err := fetchQuotationProducts(ctx, service.db, normalizedQuotationID)
	if err != nil {
		return models.Quotation{}, err
	}

	quotation.Products = products
	quotation.ProductCount = len(products)

	return quotation, nil
}

func (service *QuotationService) UpdateQuotation(ctx context.Context, quotationID string, input CreateQuotationInput) (models.Quotation, error) {
	normalizedQuotationID := strings.TrimSpace(quotationID)
	if normalizedQuotationID == "" {
		return models.Quotation{}, ValidationError{Message: "Quotation ID is required."}
	}

	validatedInput, err := validateQuotationInput(input)
	if err != nil {
		return models.Quotation{}, err
	}

	if err := service.ensureRecurringPlanExists(ctx, validatedInput.RecurringPlanID); err != nil {
		return models.Quotation{}, err
	}

	tx, err := service.db.Begin(ctx)
	if err != nil {
		return models.Quotation{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const query = `
		UPDATE quotations.quotation
		SET
			last_forever = $1,
			quotation_validity_days = $2,
			recurring_plan_id = $3,
			updated_at = NOW()
		WHERE quotation_id = $4
		RETURNING quotation_id`

	var updatedQuotationID string
	if err := tx.QueryRow(
		ctx,
		query,
		validatedInput.LastForever,
		validatedInput.QuotationValidityDays,
		validatedInput.RecurringPlanID,
		normalizedQuotationID,
	).Scan(&updatedQuotationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Quotation{}, ErrQuotationNotFound
		}
		return models.Quotation{}, fmt.Errorf("failed to update quotation: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM quotations.quotations_products WHERE quotation_id = $1`, normalizedQuotationID); err != nil {
		return models.Quotation{}, fmt.Errorf("failed to refresh quotation products: %w", err)
	}

	if err := insertQuotationProducts(ctx, tx, normalizedQuotationID, validatedInput.ProductIDs); err != nil {
		return models.Quotation{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Quotation{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return service.GetQuotationByID(ctx, updatedQuotationID)
}

func (service *QuotationService) DeleteQuotation(ctx context.Context, quotationID string) error {
	normalizedQuotationID := strings.TrimSpace(quotationID)
	if normalizedQuotationID == "" {
		return ValidationError{Message: "Quotation ID is required."}
	}

	result, err := service.db.Exec(ctx, `DELETE FROM quotations.quotation WHERE quotation_id = $1`, normalizedQuotationID)
	if err != nil {
		return fmt.Errorf("failed to delete quotation: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrQuotationNotFound
	}

	return nil
}
