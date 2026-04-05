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
	ErrTaxAlreadyExists = errors.New("tax already exists")
	ErrTaxNotFound      = errors.New("tax not found")
)

type CreateTaxInput struct {
	TaxName             string
	TaxComputationUnit  string
	TaxComputationValue float64
}

// TaxService encapsulates tax operations.
type TaxService struct {
	db *pgxpool.Pool
}

func NewTaxService(db *pgxpool.Pool) *TaxService {
	return &TaxService{db: db}
}

func normalizeTaxComputationUnit(unit string) (string, bool) {
	normalizedUnit := strings.ToLower(strings.TrimSpace(unit))
	switch {
	case strings.Contains(normalizedUnit, "fixed") && strings.Contains(normalizedUnit, "price"):
		return models.TaxComputationUnitFixedPrice, true
	case strings.Contains(normalizedUnit, "percentage"):
		return models.TaxComputationUnitPercentage, true
	default:
		return "", false
	}
}

func validateTaxInput(input CreateTaxInput) (CreateTaxInput, error) {
	taxName := strings.TrimSpace(input.TaxName)
	if taxName == "" {
		return CreateTaxInput{}, ValidationError{Message: "Tax name is required."}
	}

	normalizedUnit, validUnit := normalizeTaxComputationUnit(input.TaxComputationUnit)
	if !validUnit {
		return CreateTaxInput{}, ValidationError{Message: "Tax computation unit must be either Fixed Price or Percentage."}
	}

	if input.TaxComputationValue < 0 {
		return CreateTaxInput{}, ValidationError{Message: "Tax computation value cannot be negative."}
	}
	if normalizedUnit == models.TaxComputationUnitPercentage && input.TaxComputationValue > 100 {
		return CreateTaxInput{}, ValidationError{Message: "Percentage tax value cannot be greater than 100."}
	}

	return CreateTaxInput{
		TaxName:             taxName,
		TaxComputationUnit:  normalizedUnit,
		TaxComputationValue: input.TaxComputationValue,
	}, nil
}

func scanTax(row pgx.Row) (models.Tax, error) {
	var tax models.Tax
	if err := row.Scan(
		&tax.TaxID,
		&tax.TaxName,
		&tax.TaxComputationUnit,
		&tax.TaxComputationValue,
		&tax.CreatedAt,
		&tax.UpdatedAt,
	); err != nil {
		return models.Tax{}, err
	}

	return tax, nil
}

func isTaxNameUniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23505" && pgError.ConstraintName == "tax_data_tax_name_key"
	}

	return false
}

func (service *TaxService) CreateTax(ctx context.Context, input CreateTaxInput) (models.Tax, error) {
	validatedInput, err := validateTaxInput(input)
	if err != nil {
		return models.Tax{}, err
	}

	const query = `
		INSERT INTO taxes.tax_data (tax_name, tax_computation_unit, tax_computation_value)
		VALUES ($1, $2, $3)
		RETURNING tax_id, tax_name, tax_computation_unit, tax_computation_value::float8, created_at, updated_at`

	tax, err := scanTax(service.db.QueryRow(
		ctx,
		query,
		validatedInput.TaxName,
		validatedInput.TaxComputationUnit,
		validatedInput.TaxComputationValue,
	))
	if err != nil {
		if isTaxNameUniqueViolation(err) {
			return models.Tax{}, ErrTaxAlreadyExists
		}
		return models.Tax{}, fmt.Errorf("failed to create tax: %w", err)
	}

	return tax, nil
}

func (service *TaxService) ListTaxes(ctx context.Context, search string, page int, limit int) ([]models.Tax, int, error) {
	if limit <= 0 || limit > 30 {
		limit = 30
	}

	normalizedSearch := strings.TrimSpace(search)

	const baseQuery = `
		SELECT tax_id, tax_name, tax_computation_unit, tax_computation_value::float8, created_at, updated_at
		FROM taxes.tax_data
		WHERE (
			$1 = ''
			OR tax_name ILIKE '%' || $1 || '%'
			OR tax_computation_unit ILIKE '%' || $1 || '%'
		)
		ORDER BY tax_name ASC`

	totalRecords := 0
	query := baseQuery
	queryArgs := []interface{}{normalizedSearch}

	if page > 0 {
		offset := (page - 1) * limit

		const countQuery = `
			SELECT COUNT(*)
			FROM taxes.tax_data
			WHERE (
				$1 = ''
				OR tax_name ILIKE '%' || $1 || '%'
				OR tax_computation_unit ILIKE '%' || $1 || '%'
			)`

		if err := service.db.QueryRow(ctx, countQuery, normalizedSearch).Scan(&totalRecords); err != nil {
			return nil, 0, fmt.Errorf("failed to count taxes: %w", err)
		}

		query = query + "\n\t\tLIMIT $2 OFFSET $3"
		queryArgs = append(queryArgs, limit, offset)
	}

	rows, err := service.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list taxes: %w", err)
	}
	defer rows.Close()

	taxes := make([]models.Tax, 0)
	for rows.Next() {
		tax, scanErr := scanTax(rows)
		if scanErr != nil {
			return nil, 0, fmt.Errorf("failed to scan tax row: %w", scanErr)
		}
		taxes = append(taxes, tax)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed while iterating tax rows: %w", err)
	}

	if page <= 0 {
		totalRecords = len(taxes)
	}

	return taxes, totalRecords, nil
}

func (service *TaxService) GetTaxByID(ctx context.Context, taxID string) (models.Tax, error) {
	normalizedTaxID := strings.TrimSpace(taxID)
	if normalizedTaxID == "" {
		return models.Tax{}, ValidationError{Message: "Tax ID is required."}
	}

	const query = `
		SELECT tax_id, tax_name, tax_computation_unit, tax_computation_value::float8, created_at, updated_at
		FROM taxes.tax_data
		WHERE tax_id = $1`

	tax, err := scanTax(service.db.QueryRow(ctx, query, normalizedTaxID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Tax{}, ErrTaxNotFound
		}
		return models.Tax{}, fmt.Errorf("failed to fetch tax: %w", err)
	}

	return tax, nil
}

func (service *TaxService) UpdateTax(ctx context.Context, taxID string, input CreateTaxInput) (models.Tax, error) {
	normalizedTaxID := strings.TrimSpace(taxID)
	if normalizedTaxID == "" {
		return models.Tax{}, ValidationError{Message: "Tax ID is required."}
	}

	validatedInput, err := validateTaxInput(input)
	if err != nil {
		return models.Tax{}, err
	}

	const query = `
		UPDATE taxes.tax_data
		SET
			tax_name = $1,
			tax_computation_unit = $2,
			tax_computation_value = $3,
			updated_at = NOW()
		WHERE tax_id = $4
		RETURNING tax_id, tax_name, tax_computation_unit, tax_computation_value::float8, created_at, updated_at`

	tax, err := scanTax(service.db.QueryRow(
		ctx,
		query,
		validatedInput.TaxName,
		validatedInput.TaxComputationUnit,
		validatedInput.TaxComputationValue,
		normalizedTaxID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Tax{}, ErrTaxNotFound
		}
		if isTaxNameUniqueViolation(err) {
			return models.Tax{}, ErrTaxAlreadyExists
		}
		return models.Tax{}, fmt.Errorf("failed to update tax: %w", err)
	}

	return tax, nil
}

func (service *TaxService) DeleteTax(ctx context.Context, taxID string) error {
	normalizedTaxID := strings.TrimSpace(taxID)
	if normalizedTaxID == "" {
		return ValidationError{Message: "Tax ID is required."}
	}

	result, err := service.db.Exec(ctx, `DELETE FROM taxes.tax_data WHERE tax_id = $1`, normalizedTaxID)
	if err != nil {
		return fmt.Errorf("failed to delete tax: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTaxNotFound
	}

	return nil
}
