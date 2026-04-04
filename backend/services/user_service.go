package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
)

// ValidationError indicates client-side invalid input.
type ValidationError struct {
	Message string
}

func (error ValidationError) Error() string {
	return error.Message
}

type CreateUserInput struct {
	Name        string
	Email       string
	Password    string
	CountryCode string
	PhoneNumber string
}

type UpdateUserInput struct {
	Name        string
	Email       string
	PhoneNumber string
	Address     string
}

// UserService encapsulates user account operations.
type UserService struct {
	db *pgxpool.Pool
}

func NewUserService(db *pgxpool.Pool) *UserService {
	return &UserService{db: db}
}

func (service *UserService) CreateUser(ctx context.Context, input CreateUserInput) (models.User, error) {
	name := strings.TrimSpace(input.Name)
	email := strings.ToLower(strings.TrimSpace(input.Email))
	password := strings.TrimSpace(input.Password)
	countryCode := normalizeCountryCode(input.CountryCode)
	phoneNumber := normalizeDigits(input.PhoneNumber)

	if name == "" || email == "" || password == "" || countryCode == "" || phoneNumber == "" {
		return models.User{}, ValidationError{Message: "name, email, password, country code and phone number are required"}
	}
	if len(phoneNumber) != 10 {
		return models.User{}, ValidationError{Message: "phone number must be exactly 10 digits"}
	}
	if !isValidEmail(email) {
		return models.User{}, ValidationError{Message: "invalid email format"}
	}

	fullPhoneNumber := countryCode + phoneNumber
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, fmt.Errorf("failed to hash password: %w", err)
	}

	const query = `
		INSERT INTO users."user" (name, phone_number, address, email, password_hash, role)
		VALUES ($1, $2, NULL, $3, $4, 'User')
		RETURNING id, name, email, phone_number, address, role, created_at, updated_at`

	user, err := scanUser(
		service.db.QueryRow(ctx, query, name, fullPhoneNumber, email, string(passwordHash)),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return models.User{}, ErrEmailAlreadyExists
		}
		if isPhoneConstraintViolation(err) {
			return models.User{}, ValidationError{Message: "phone number format is invalid"}
		}
		return models.User{}, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (service *UserService) AuthenticateUser(ctx context.Context, email, password string) (models.User, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	normalizedPassword := strings.TrimSpace(password)

	if normalizedEmail == "" || normalizedPassword == "" {
		return models.User{}, ErrInvalidCredentials
	}

	const query = `
		SELECT id, name, email, phone_number, address, role, created_at, updated_at, password_hash
		FROM users."user"
		WHERE email = $1
		LIMIT 1`

	var user models.User
	var address *string
	var role string
	var passwordHash string
	if err := service.db.QueryRow(ctx, query, normalizedEmail).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PhoneNumber,
		&address,
		&role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&passwordHash,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrInvalidCredentials
		}
		return models.User{}, fmt.Errorf("failed to fetch user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(normalizedPassword)); err != nil {
		return models.User{}, ErrInvalidCredentials
	}

	user.Address = address
	user.Role = models.Role(role)

	return user, nil
}

func (service *UserService) TokenRoleFromDBRole(role models.Role) string {
	switch strings.ToLower(strings.TrimSpace(string(role))) {
	case "admin":
		return "admin"
	case "internal":
		return "internal-user"
	default:
		return "user"
	}
}

func (service *UserService) ListUsersByRole(ctx context.Context, role models.Role, search string, limit int) ([]models.User, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	normalizedSearch := strings.TrimSpace(search)
	normalizedRole := strings.TrimSpace(string(role))
	if normalizedRole == "" {
		return nil, ValidationError{Message: "Role is required."}
	}

	const query = `
		SELECT id, name, email, phone_number, address, role, created_at, updated_at
		FROM users."user"
		WHERE role = $1
		  AND (
			$2 = ''
			OR name ILIKE '%' || $2 || '%'
			OR email ILIKE '%' || $2 || '%'
			OR phone_number ILIKE '%' || $2 || '%'
		  )
		ORDER BY name ASC
		LIMIT $3`

	rows, err := service.db.Query(ctx, query, normalizedRole, normalizedSearch, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		user, scanErr := scanUser(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", scanErr)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating user rows: %w", err)
	}

	return users, nil
}

func normalizeUserRole(role string) (models.Role, bool) {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "admin":
		return models.RoleAdmin, true
	case "internal", "internal-user", "internal_user":
		return models.RoleInternal, true
	case "user", "portal-user", "portal_user":
		return models.RoleUser, true
	default:
		return "", false
	}
}

func normalizePhoneNumber(phoneNumber string) string {
	trimmedPhoneNumber := strings.TrimSpace(phoneNumber)
	if trimmedPhoneNumber == "" {
		return ""
	}

	digitsOnlyPhone := normalizeDigits(trimmedPhoneNumber)
	if digitsOnlyPhone == "" {
		return ""
	}

	if strings.HasPrefix(trimmedPhoneNumber, "+") {
		return "+" + digitsOnlyPhone
	}

	if len(digitsOnlyPhone) == 10 {
		return "+91" + digitsOnlyPhone
	}

	return "+" + digitsOnlyPhone
}

func isValidE164Phone(phoneNumber string) bool {
	e164PhoneRegex := regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
	return e164PhoneRegex.MatchString(phoneNumber)
}

func validateUserUpdateInput(input UpdateUserInput) (UpdateUserInput, error) {
	name := strings.TrimSpace(input.Name)
	email := strings.ToLower(strings.TrimSpace(input.Email))
	phoneNumber := normalizePhoneNumber(input.PhoneNumber)
	address := strings.TrimSpace(input.Address)

	if name == "" || email == "" || phoneNumber == "" {
		return UpdateUserInput{}, ValidationError{Message: "name, email and phone number are required"}
	}
	if !isValidEmail(email) {
		return UpdateUserInput{}, ValidationError{Message: "invalid email format"}
	}
	if !isValidE164Phone(phoneNumber) {
		return UpdateUserInput{}, ValidationError{Message: "phone number must be in valid format, e.g. +919876543210"}
	}

	return UpdateUserInput{
		Name:        name,
		Email:       email,
		PhoneNumber: phoneNumber,
		Address:     address,
	}, nil
}

func (service *UserService) ListUsers(ctx context.Context, search string, limit int) ([]models.User, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	normalizedSearch := strings.TrimSpace(search)

	const query = `
		SELECT id, name, email, phone_number, address, role, created_at, updated_at
		FROM users."user"
		WHERE (
			$1 = ''
			OR name ILIKE '%' || $1 || '%'
			OR email ILIKE '%' || $1 || '%'
			OR phone_number ILIKE '%' || $1 || '%'
			OR role::text ILIKE '%' || $1 || '%'
		)
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := service.db.Query(ctx, query, normalizedSearch, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		user, scanErr := scanUser(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", scanErr)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating user rows: %w", err)
	}

	return users, nil
}

func (service *UserService) GetUserByID(ctx context.Context, userID string) (models.User, error) {
	normalizedUserID := strings.TrimSpace(userID)
	if normalizedUserID == "" {
		return models.User{}, ValidationError{Message: "User ID is required."}
	}

	const query = `
		SELECT id, name, email, phone_number, address, role, created_at, updated_at
		FROM users."user"
		WHERE id = $1`

	user, err := scanUser(service.db.QueryRow(ctx, query, normalizedUserID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrUserNotFound
		}
		return models.User{}, fmt.Errorf("failed to fetch user: %w", err)
	}

	return user, nil
}

func (service *UserService) ListActiveSubscriptionsByUserID(ctx context.Context, userID string) ([]models.UserSubscriptionSummary, error) {
	normalizedUserID := strings.TrimSpace(userID)
	if normalizedUserID == "" {
		return nil, ValidationError{Message: "User ID is required."}
	}

	const query = `
		SELECT
			s.subscription_id,
			s.subscription_number,
			s.next_invoice_date,
			rp.billing_period,
			rp.recurring_name,
			s.status
		FROM subscription.subscriptions s
		JOIN recurring_plans.recurring_plan_data rp ON rp.recurring_plan_id = s.recurring_plan_id
		WHERE s.customer_id = $1
		  AND s.status IN ('Active', 'Confirmed')
		ORDER BY s.next_invoice_date ASC, s.created_at DESC`

	rows, err := service.db.Query(ctx, query, normalizedUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active subscriptions: %w", err)
	}
	defer rows.Close()

	items := make([]models.UserSubscriptionSummary, 0)
	for rows.Next() {
		var item models.UserSubscriptionSummary
		if err := rows.Scan(
			&item.SubscriptionID,
			&item.SubscriptionNumber,
			&item.NextInvoiceDate,
			&item.Recurring,
			&item.Plan,
			&item.Status,
		); err != nil {
			return nil, fmt.Errorf("failed to scan active subscription row: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating active subscriptions: %w", err)
	}

	return items, nil
}

func (service *UserService) ListPortalSubscriptionsByUserID(ctx context.Context, userID string) ([]models.UserSubscriptionSummary, error) {
	normalizedUserID := strings.TrimSpace(userID)
	if normalizedUserID == "" {
		return nil, ValidationError{Message: "User ID is required."}
	}

	const query = `
		SELECT
			s.subscription_id,
			s.subscription_number,
			s.next_invoice_date,
			rp.billing_period,
			rp.recurring_name,
			s.status
		FROM subscription.subscriptions s
		JOIN recurring_plans.recurring_plan_data rp ON rp.recurring_plan_id = s.recurring_plan_id
		WHERE s.customer_id = $1
		  AND s.status IN ('Draft', 'Quotation Sent', 'Active', 'Confirmed')
		ORDER BY s.updated_at DESC, s.created_at DESC`

	rows, err := service.db.Query(ctx, query, normalizedUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to list portal subscriptions: %w", err)
	}
	defer rows.Close()

	items := make([]models.UserSubscriptionSummary, 0)
	for rows.Next() {
		var item models.UserSubscriptionSummary
		if err := rows.Scan(
			&item.SubscriptionID,
			&item.SubscriptionNumber,
			&item.NextInvoiceDate,
			&item.Recurring,
			&item.Plan,
			&item.Status,
		); err != nil {
			return nil, fmt.Errorf("failed to scan portal subscription row: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating portal subscriptions: %w", err)
	}

	return items, nil
}

func (service *UserService) ListPortalSubscriptionsDetailedByUserID(ctx context.Context, userID string) ([]models.Subscription, error) {
	normalizedUserID := strings.TrimSpace(userID)
	if normalizedUserID == "" {
		return nil, ValidationError{Message: "User ID is required."}
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
		JOIN recurring_plans.recurring_plan_data rp ON rp.recurring_plan_id = s.recurring_plan_id
		LEFT JOIN payment_term.payment_term_data pt ON pt.payment_term_id = s.payment_term_id
		WHERE s.customer_id = $1
		  AND s.status IN ('Draft', 'Quotation Sent', 'Active', 'Confirmed')
		ORDER BY s.updated_at DESC, s.created_at DESC`

	rows, err := service.db.Query(ctx, query, normalizedUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to list portal subscriptions: %w", err)
	}
	defer rows.Close()

	subscriptions := make([]models.Subscription, 0)
	for rows.Next() {
		subscription, scanErr := scanSubscriptionRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan portal subscription row: %w", scanErr)
		}

		products, productErr := fetchSubscriptionProducts(ctx, service.db, subscription.SubscriptionID)
		if productErr != nil {
			return nil, productErr
		}

		otherInfo, otherInfoErr := fetchSubscriptionOtherInfo(ctx, service.db, subscription.SubscriptionID)
		if otherInfoErr != nil {
			return nil, otherInfoErr
		}

		payment, paymentErr := fetchLatestSubscriptionPayment(ctx, service.db, subscription.SubscriptionID)
		if paymentErr != nil {
			return nil, paymentErr
		}

		subscription.Products = products
		subscription.OtherInfo = otherInfo
		subscription.Payment = payment
		subscriptions = append(subscriptions, subscription)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating portal subscriptions: %w", err)
	}

	return subscriptions, nil
}

func (service *UserService) UpdateUser(ctx context.Context, userID string, input UpdateUserInput) (models.User, error) {
	normalizedUserID := strings.TrimSpace(userID)
	if normalizedUserID == "" {
		return models.User{}, ValidationError{Message: "User ID is required."}
	}

	validatedInput, err := validateUserUpdateInput(input)
	if err != nil {
		return models.User{}, err
	}

	const query = `
		UPDATE users."user"
		SET
			name = $1,
			phone_number = $2,
			address = $3,
			email = $4,
			updated_at = NOW()
		WHERE id = $5
		RETURNING id, name, email, phone_number, address, role, created_at, updated_at`

	user, err := scanUser(service.db.QueryRow(
		ctx,
		query,
		validatedInput.Name,
		validatedInput.PhoneNumber,
		nullableString(validatedInput.Address),
		validatedInput.Email,
		normalizedUserID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrUserNotFound
		}
		if isUniqueViolation(err) {
			return models.User{}, ErrEmailAlreadyExists
		}
		if isPhoneConstraintViolation(err) {
			return models.User{}, ValidationError{Message: "phone number format is invalid"}
		}
		return models.User{}, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

func (service *UserService) UpdateUserAddress(ctx context.Context, userID, address string) (models.User, error) {
	normalizedUserID := strings.TrimSpace(userID)
	if normalizedUserID == "" {
		return models.User{}, ValidationError{Message: "User ID is required."}
	}

	normalizedAddress := strings.TrimSpace(address)
	if normalizedAddress == "" {
		return models.User{}, ValidationError{Message: "Address is required."}
	}

	const query = `
		UPDATE users."user"
		SET
			address = $1,
			updated_at = NOW()
		WHERE id = $2
		RETURNING id, name, email, phone_number, address, role, created_at, updated_at`

	user, err := scanUser(service.db.QueryRow(
		ctx,
		query,
		normalizedAddress,
		normalizedUserID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrUserNotFound
		}
		return models.User{}, fmt.Errorf("failed to update user address: %w", err)
	}

	return user, nil
}

func (service *UserService) DeleteUser(ctx context.Context, userID string) error {
	normalizedUserID := strings.TrimSpace(userID)
	if normalizedUserID == "" {
		return ValidationError{Message: "User ID is required."}
	}

	result, err := service.db.Exec(ctx, `DELETE FROM users."user" WHERE id = $1`, normalizedUserID)
	if err != nil {
		if isForeignKeyViolation(err) {
			return ValidationError{Message: "Cannot delete user because related subscriptions exist."}
		}
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

func scanUser(row pgx.Row) (models.User, error) {
	var user models.User
	var address *string
	var role string
	if err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PhoneNumber,
		&address,
		&role,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return models.User{}, err
	}

	user.Address = address
	user.Role = models.Role(role)

	return user, nil
}

func normalizeCountryCode(code string) string {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "+") {
		return "+" + normalizeDigits(trimmed)
	}
	return "+" + normalizeDigits(trimmed)
}

func normalizeDigits(value string) string {
	digitRegex := regexp.MustCompile(`\d`)
	return strings.Join(digitRegex.FindAllString(value, -1), "")
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func isUniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23505"
	}
	return false
}

func isPhoneConstraintViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23514" && pgError.ConstraintName == "chk_user_phone_e164_format"
	}
	return false
}
