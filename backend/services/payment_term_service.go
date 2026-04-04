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
	ErrPaymentTermAlreadyExists = errors.New("payment term already exists")
	ErrPaymentTermNotFound      = errors.New("payment term not found")
)

type CreatePaymentTermInput struct {
	PaymentTermName string
	DueUnit         string
	DueValue        float64
	IntervalDays    int
}

// PaymentTermService encapsulates payment-term operations.
type PaymentTermService struct {
	db *pgxpool.Pool
}

func NewPaymentTermService(db *pgxpool.Pool) *PaymentTermService {
	return &PaymentTermService{db: db}
}

func normalizePaymentDueUnit(unit string) (string, bool) {
	normalizedUnit := strings.ToLower(strings.TrimSpace(unit))
	switch {
	case strings.Contains(normalizedUnit, "fixed") && strings.Contains(normalizedUnit, "price"):
		return models.PaymentTermDueUnitFixedPrice, true
	case strings.Contains(normalizedUnit, "percentage"):
		return models.PaymentTermDueUnitPercentage, true
	default:
		return "", false
	}
}

func validatePaymentTermInput(input CreatePaymentTermInput) (CreatePaymentTermInput, error) {
	paymentTermName := strings.TrimSpace(input.PaymentTermName)
	if paymentTermName == "" {
		return CreatePaymentTermInput{}, ValidationError{Message: "Payment term name is required."}
	}

	normalizedDueUnit, validDueUnit := normalizePaymentDueUnit(input.DueUnit)
	if !validDueUnit {
		return CreatePaymentTermInput{}, ValidationError{Message: "Due unit must be either Percentage or Fixed Price."}
	}

	if input.DueValue <= 0 {
		return CreatePaymentTermInput{}, ValidationError{Message: "Due value must be greater than zero."}
	}
	if normalizedDueUnit == models.PaymentTermDueUnitPercentage && input.DueValue > 100 {
		return CreatePaymentTermInput{}, ValidationError{Message: "Percentage due value cannot be greater than 100."}
	}

	if input.IntervalDays <= 0 {
		return CreatePaymentTermInput{}, ValidationError{Message: "Interval (in days) must be greater than zero."}
	}

	return CreatePaymentTermInput{
		PaymentTermName: paymentTermName,
		DueUnit:         normalizedDueUnit,
		DueValue:        input.DueValue,
		IntervalDays:    input.IntervalDays,
	}, nil
}

func scanPaymentTerm(row pgx.Row) (models.PaymentTerm, error) {
	var paymentTerm models.PaymentTerm
	if err := row.Scan(
		&paymentTerm.PaymentTermID,
		&paymentTerm.PaymentTermName,
		&paymentTerm.DueUnit,
		&paymentTerm.DueValue,
		&paymentTerm.IntervalDays,
		&paymentTerm.CreatedAt,
		&paymentTerm.UpdatedAt,
	); err != nil {
		return models.PaymentTerm{}, err
	}

	return paymentTerm, nil
}

func isPaymentTermNameUniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23505" && pgError.ConstraintName == "payment_term_data_payment_term_name_key"
	}

	return false
}

func (service *PaymentTermService) CreatePaymentTerm(ctx context.Context, input CreatePaymentTermInput) (models.PaymentTerm, error) {
	validatedInput, err := validatePaymentTermInput(input)
	if err != nil {
		return models.PaymentTerm{}, err
	}

	const query = `
		INSERT INTO payment_term.payment_term_data (payment_term_name, due_unit, due_value, interval_days)
		VALUES ($1, $2, $3, $4)
		RETURNING payment_term_id, payment_term_name, due_unit, due_value::float8, interval_days, created_at, updated_at`

	paymentTerm, err := scanPaymentTerm(service.db.QueryRow(
		ctx,
		query,
		validatedInput.PaymentTermName,
		validatedInput.DueUnit,
		validatedInput.DueValue,
		validatedInput.IntervalDays,
	))
	if err != nil {
		if isPaymentTermNameUniqueViolation(err) {
			return models.PaymentTerm{}, ErrPaymentTermAlreadyExists
		}
		return models.PaymentTerm{}, fmt.Errorf("failed to create payment term: %w", err)
	}

	return paymentTerm, nil
}

func (service *PaymentTermService) ListPaymentTerms(ctx context.Context, search string) ([]models.PaymentTerm, error) {
	normalizedSearch := strings.TrimSpace(search)

	const query = `
		SELECT payment_term_id, payment_term_name, due_unit, due_value::float8, interval_days, created_at, updated_at
		FROM payment_term.payment_term_data
		WHERE (
			$1 = ''
			OR payment_term_name ILIKE '%' || $1 || '%'
			OR due_unit ILIKE '%' || $1 || '%'
		)
		ORDER BY payment_term_name ASC`

	rows, err := service.db.Query(ctx, query, normalizedSearch)
	if err != nil {
		return nil, fmt.Errorf("failed to list payment terms: %w", err)
	}
	defer rows.Close()

	paymentTerms := make([]models.PaymentTerm, 0)
	for rows.Next() {
		paymentTerm, scanErr := scanPaymentTerm(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan payment term row: %w", scanErr)
		}
		paymentTerms = append(paymentTerms, paymentTerm)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating payment term rows: %w", err)
	}

	return paymentTerms, nil
}

func (service *PaymentTermService) GetPaymentTermByID(ctx context.Context, paymentTermID string) (models.PaymentTerm, error) {
	normalizedPaymentTermID := strings.TrimSpace(paymentTermID)
	if normalizedPaymentTermID == "" {
		return models.PaymentTerm{}, ValidationError{Message: "Payment term ID is required."}
	}

	const query = `
		SELECT payment_term_id, payment_term_name, due_unit, due_value::float8, interval_days, created_at, updated_at
		FROM payment_term.payment_term_data
		WHERE payment_term_id = $1`

	paymentTerm, err := scanPaymentTerm(service.db.QueryRow(ctx, query, normalizedPaymentTermID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.PaymentTerm{}, ErrPaymentTermNotFound
		}
		return models.PaymentTerm{}, fmt.Errorf("failed to fetch payment term: %w", err)
	}

	return paymentTerm, nil
}

func (service *PaymentTermService) UpdatePaymentTerm(ctx context.Context, paymentTermID string, input CreatePaymentTermInput) (models.PaymentTerm, error) {
	normalizedPaymentTermID := strings.TrimSpace(paymentTermID)
	if normalizedPaymentTermID == "" {
		return models.PaymentTerm{}, ValidationError{Message: "Payment term ID is required."}
	}

	validatedInput, err := validatePaymentTermInput(input)
	if err != nil {
		return models.PaymentTerm{}, err
	}

	const query = `
		UPDATE payment_term.payment_term_data
		SET
			payment_term_name = $1,
			due_unit = $2,
			due_value = $3,
			interval_days = $4,
			updated_at = NOW()
		WHERE payment_term_id = $5
		RETURNING payment_term_id, payment_term_name, due_unit, due_value::float8, interval_days, created_at, updated_at`

	paymentTerm, err := scanPaymentTerm(service.db.QueryRow(
		ctx,
		query,
		validatedInput.PaymentTermName,
		validatedInput.DueUnit,
		validatedInput.DueValue,
		validatedInput.IntervalDays,
		normalizedPaymentTermID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.PaymentTerm{}, ErrPaymentTermNotFound
		}
		if isPaymentTermNameUniqueViolation(err) {
			return models.PaymentTerm{}, ErrPaymentTermAlreadyExists
		}
		return models.PaymentTerm{}, fmt.Errorf("failed to update payment term: %w", err)
	}

	return paymentTerm, nil
}

func (service *PaymentTermService) DeletePaymentTerm(ctx context.Context, paymentTermID string) error {
	normalizedPaymentTermID := strings.TrimSpace(paymentTermID)
	if normalizedPaymentTermID == "" {
		return ValidationError{Message: "Payment term ID is required."}
	}

	result, err := service.db.Exec(ctx, `DELETE FROM payment_term.payment_term_data WHERE payment_term_id = $1`, normalizedPaymentTermID)
	if err != nil {
		return fmt.Errorf("failed to delete payment term: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrPaymentTermNotFound
	}

	return nil
}
