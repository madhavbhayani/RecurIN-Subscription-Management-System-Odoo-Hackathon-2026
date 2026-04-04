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
