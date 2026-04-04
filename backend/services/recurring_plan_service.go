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
	ErrRecurringPlanAlreadyExists = errors.New("recurring plan already exists")
	ErrRecurringPlanNotFound      = errors.New("recurring plan not found")
)

var billingPeriodLimits = map[string]struct {
	Min int
	Max int
}{
	"Daily":   {Min: 1, Max: 365},
	"Weekly":  {Min: 1, Max: 52},
	"Monthly": {Min: 1, Max: 12},
	"Yearly":  {Min: 1, Max: 10},
}

type CreateRecurringPlanInput struct {
	RecurringName        string
	BillingPeriod        string
	IsClosable           bool
	AutomaticCloseCycles *int
	IsPausable           bool
	IsRenewable          bool
	IsActive             bool
}

// RecurringPlanService encapsulates recurring plan operations.
type RecurringPlanService struct {
	db *pgxpool.Pool
}

func NewRecurringPlanService(db *pgxpool.Pool) *RecurringPlanService {
	return &RecurringPlanService{db: db}
}

func normalizeBillingPeriod(billingPeriod string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(billingPeriod))
	switch normalized {
	case "daily":
		return "Daily", true
	case "weekly":
		return "Weekly", true
	case "monthly":
		return "Monthly", true
	case "yearly", "annual":
		return "Yearly", true
	default:
		return "", false
	}
}

func validateRecurringPlanInput(input CreateRecurringPlanInput) (CreateRecurringPlanInput, error) {
	recurringName := strings.TrimSpace(input.RecurringName)
	if recurringName == "" {
		return CreateRecurringPlanInput{}, ValidationError{Message: "Recurring name is required."}
	}

	normalizedBillingPeriod, validBillingPeriod := normalizeBillingPeriod(input.BillingPeriod)
	if !validBillingPeriod {
		return CreateRecurringPlanInput{}, ValidationError{Message: "Billing period must be Daily, Weekly, Monthly, or Yearly."}
	}

	periodLimit := billingPeriodLimits[normalizedBillingPeriod]

	var normalizedAutomaticCloseCycles *int
	if input.IsClosable {
		if input.AutomaticCloseCycles == nil {
			return CreateRecurringPlanInput{}, ValidationError{Message: "Automatic close cycles are required when recurring plan is closable."}
		}

		cycles := *input.AutomaticCloseCycles
		if cycles < periodLimit.Min || cycles > periodLimit.Max {
			return CreateRecurringPlanInput{}, ValidationError{Message: fmt.Sprintf("Automatic close cycles must be between %d and %d for %s.", periodLimit.Min, periodLimit.Max, normalizedBillingPeriod)}
		}

		normalizedAutomaticCloseCycles = &cycles
	}

	return CreateRecurringPlanInput{
		RecurringName:        recurringName,
		BillingPeriod:        normalizedBillingPeriod,
		IsClosable:           input.IsClosable,
		AutomaticCloseCycles: normalizedAutomaticCloseCycles,
		IsPausable:           input.IsPausable,
		IsRenewable:          input.IsRenewable,
		IsActive:             input.IsActive,
	}, nil
}

func isRecurringPlanNameUniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23505" && strings.Contains(pgError.ConstraintName, "recurring_name")
	}

	return false
}

func scanRecurringPlanBaseRow(row pgx.Row) (models.RecurringPlan, error) {
	var recurringPlan models.RecurringPlan
	if err := row.Scan(
		&recurringPlan.RecurringPlanID,
		&recurringPlan.RecurringName,
		&recurringPlan.BillingPeriod,
		&recurringPlan.IsClosable,
		&recurringPlan.AutomaticCloseCycles,
		&recurringPlan.IsPausable,
		&recurringPlan.IsRenewable,
		&recurringPlan.IsActive,
		&recurringPlan.CreatedAt,
		&recurringPlan.UpdatedAt,
	); err != nil {
		return models.RecurringPlan{}, err
	}

	return recurringPlan, nil
}

func (service *RecurringPlanService) CreateRecurringPlan(ctx context.Context, input CreateRecurringPlanInput) (models.RecurringPlan, error) {
	validatedInput, err := validateRecurringPlanInput(input)
	if err != nil {
		return models.RecurringPlan{}, err
	}

	const query = `
		INSERT INTO recurring_plans.recurring_plan_data (
			recurring_name,
			billing_period,
			is_closable,
			automatic_close_cycles,
			is_pausable,
			is_renewable,
			is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING recurring_plan_id, recurring_name, billing_period, is_closable, automatic_close_cycles, is_pausable, is_renewable, is_active, created_at, updated_at`

	createdRecurringPlan, err := scanRecurringPlanBaseRow(service.db.QueryRow(
		ctx,
		query,
		validatedInput.RecurringName,
		validatedInput.BillingPeriod,
		validatedInput.IsClosable,
		validatedInput.AutomaticCloseCycles,
		validatedInput.IsPausable,
		validatedInput.IsRenewable,
		validatedInput.IsActive,
	))
	if err != nil {
		if isRecurringPlanNameUniqueViolation(err) {
			return models.RecurringPlan{}, ErrRecurringPlanAlreadyExists
		}
		return models.RecurringPlan{}, fmt.Errorf("failed to create recurring plan: %w", err)
	}

	createdRecurringPlan.ProductCount = 0
	createdRecurringPlan.Products = []models.RecurringPlanProduct{}
	return service.GetRecurringPlanByID(ctx, createdRecurringPlan.RecurringPlanID)
}

func (service *RecurringPlanService) ListRecurringPlans(ctx context.Context, search string, activeOnly bool) ([]models.RecurringPlan, error) {
	normalizedSearch := strings.TrimSpace(search)

	const query = `
		SELECT
			rp.recurring_plan_id,
			rp.recurring_name,
			rp.billing_period,
			rp.is_closable,
			rp.automatic_close_cycles,
			rp.is_pausable,
			rp.is_renewable,
			rp.is_active,
			rp.created_at,
			rp.updated_at,
			0::int AS product_count
		FROM recurring_plans.recurring_plan_data rp
		WHERE ($1 = '' OR rp.recurring_name ILIKE '%' || $1 || '%')
		  AND ($2 = FALSE OR rp.is_active = TRUE)
		ORDER BY rp.created_at DESC`

	rows, err := service.db.Query(ctx, query, normalizedSearch, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to list recurring plans: %w", err)
	}
	defer rows.Close()

	recurringPlans := make([]models.RecurringPlan, 0)
	for rows.Next() {
		var recurringPlan models.RecurringPlan
		if err := rows.Scan(
			&recurringPlan.RecurringPlanID,
			&recurringPlan.RecurringName,
			&recurringPlan.BillingPeriod,
			&recurringPlan.IsClosable,
			&recurringPlan.AutomaticCloseCycles,
			&recurringPlan.IsPausable,
			&recurringPlan.IsRenewable,
			&recurringPlan.IsActive,
			&recurringPlan.CreatedAt,
			&recurringPlan.UpdatedAt,
			&recurringPlan.ProductCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan recurring plan row: %w", err)
		}
		recurringPlan.Products = []models.RecurringPlanProduct{}
		recurringPlans = append(recurringPlans, recurringPlan)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating recurring plan rows: %w", err)
	}

	return recurringPlans, nil
}

func (service *RecurringPlanService) GetRecurringPlanByID(ctx context.Context, recurringPlanID string) (models.RecurringPlan, error) {
	normalizedRecurringPlanID := strings.TrimSpace(recurringPlanID)
	if normalizedRecurringPlanID == "" {
		return models.RecurringPlan{}, ValidationError{Message: "Recurring plan ID is required."}
	}

	const query = `
		SELECT recurring_plan_id, recurring_name, billing_period, is_closable, automatic_close_cycles, is_pausable, is_renewable, is_active, created_at, updated_at
		FROM recurring_plans.recurring_plan_data
		WHERE recurring_plan_id = $1`

	recurringPlan, err := scanRecurringPlanBaseRow(service.db.QueryRow(ctx, query, normalizedRecurringPlanID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.RecurringPlan{}, ErrRecurringPlanNotFound
		}
		return models.RecurringPlan{}, fmt.Errorf("failed to fetch recurring plan: %w", err)
	}

	recurringPlan.Products = []models.RecurringPlanProduct{}
	recurringPlan.ProductCount = 0
	return recurringPlan, nil
}

func (service *RecurringPlanService) UpdateRecurringPlan(ctx context.Context, recurringPlanID string, input CreateRecurringPlanInput) (models.RecurringPlan, error) {
	normalizedRecurringPlanID := strings.TrimSpace(recurringPlanID)
	if normalizedRecurringPlanID == "" {
		return models.RecurringPlan{}, ValidationError{Message: "Recurring plan ID is required."}
	}

	validatedInput, err := validateRecurringPlanInput(input)
	if err != nil {
		return models.RecurringPlan{}, err
	}

	const query = `
		UPDATE recurring_plans.recurring_plan_data
		SET
			recurring_name = $1,
			billing_period = $2,
			is_closable = $3,
			automatic_close_cycles = $4,
			is_pausable = $5,
			is_renewable = $6,
			is_active = $7,
			updated_at = NOW()
		WHERE recurring_plan_id = $8
		RETURNING recurring_plan_id, recurring_name, billing_period, is_closable, automatic_close_cycles, is_pausable, is_renewable, is_active, created_at, updated_at`

	if _, err := scanRecurringPlanBaseRow(service.db.QueryRow(
		ctx,
		query,
		validatedInput.RecurringName,
		validatedInput.BillingPeriod,
		validatedInput.IsClosable,
		validatedInput.AutomaticCloseCycles,
		validatedInput.IsPausable,
		validatedInput.IsRenewable,
		validatedInput.IsActive,
		normalizedRecurringPlanID,
	)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.RecurringPlan{}, ErrRecurringPlanNotFound
		}
		if isRecurringPlanNameUniqueViolation(err) {
			return models.RecurringPlan{}, ErrRecurringPlanAlreadyExists
		}
		return models.RecurringPlan{}, fmt.Errorf("failed to update recurring plan: %w", err)
	}

	return service.GetRecurringPlanByID(ctx, normalizedRecurringPlanID)
}

func (service *RecurringPlanService) DeleteRecurringPlan(ctx context.Context, recurringPlanID string) error {
	normalizedRecurringPlanID := strings.TrimSpace(recurringPlanID)
	if normalizedRecurringPlanID == "" {
		return ValidationError{Message: "Recurring plan ID is required."}
	}

	result, err := service.db.Exec(ctx, `DELETE FROM recurring_plans.recurring_plan_data WHERE recurring_plan_id = $1`, normalizedRecurringPlanID)
	if err != nil {
		return fmt.Errorf("failed to delete recurring plan: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrRecurringPlanNotFound
	}

	return nil
}
