package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/smtp"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultPasswordResetOTPExpiryMinutes   = 10
	defaultPasswordResetTokenExpiryMinutes = 15
	defaultPasswordResetOTPMaxAttempts     = 5
)

var otpCodeRegex = regexp.MustCompile(`^[0-9]{6}$`)

type PasswordResetServiceConfig struct {
	SMTPHost                 string
	SMTPPort                 int
	SMTPUsername             string
	SMTPPassword             string
	SMTPFromEmail            string
	SMTPFromName             string
	PasswordResetOTPExpiry   int
	PasswordResetTokenExpiry int
	PasswordResetMaxAttempts int
}

type PasswordResetService struct {
	db                            *pgxpool.Pool
	smtpHost                      string
	smtpPort                      int
	smtpUsername                  string
	smtpPassword                  string
	smtpFromEmail                 string
	smtpFromName                  string
	otpExpiryMinutes              int
	resetTokenExpiryMinutes       int
	maxOTPVerificationAttempts    int
	isEmailNotificationConfigured bool
}

func NewPasswordResetService(db *pgxpool.Pool, config PasswordResetServiceConfig) *PasswordResetService {
	service := &PasswordResetService{
		db:                            db,
		smtpHost:                      strings.TrimSpace(config.SMTPHost),
		smtpPort:                      config.SMTPPort,
		smtpUsername:                  strings.TrimSpace(config.SMTPUsername),
		smtpPassword:                  config.SMTPPassword,
		smtpFromEmail:                 strings.TrimSpace(config.SMTPFromEmail),
		smtpFromName:                  strings.TrimSpace(config.SMTPFromName),
		otpExpiryMinutes:              config.PasswordResetOTPExpiry,
		resetTokenExpiryMinutes:       config.PasswordResetTokenExpiry,
		maxOTPVerificationAttempts:    config.PasswordResetMaxAttempts,
		isEmailNotificationConfigured: false,
	}

	if service.smtpFromName == "" {
		service.smtpFromName = "RecurIN Subscriptions"
	}
	if service.smtpUsername == "" {
		service.smtpUsername = service.smtpFromEmail
	}
	if service.otpExpiryMinutes <= 0 {
		service.otpExpiryMinutes = defaultPasswordResetOTPExpiryMinutes
	}
	if service.resetTokenExpiryMinutes <= 0 {
		service.resetTokenExpiryMinutes = defaultPasswordResetTokenExpiryMinutes
	}
	if service.maxOTPVerificationAttempts <= 0 {
		service.maxOTPVerificationAttempts = defaultPasswordResetOTPMaxAttempts
	}

	service.isEmailNotificationConfigured = service.smtpHost != "" && service.smtpPort > 0 && service.smtpFromEmail != ""

	return service
}

func (service *PasswordResetService) RequestPasswordResetOTP(ctx context.Context, email string) error {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	if normalizedEmail == "" || !isValidEmail(normalizedEmail) {
		return ValidationError{Message: "A valid email address is required."}
	}

	if !service.isEmailNotificationConfigured {
		return fmt.Errorf("password reset email service is not configured")
	}

	const findUserQuery = `
		SELECT id, name
		FROM users."user"
		WHERE LOWER(email) = LOWER($1)
		LIMIT 1`

	var userID string
	var userName string
	if err := service.db.QueryRow(ctx, findUserQuery, normalizedEmail).Scan(&userID, &userName); err != nil {
		if err == pgx.ErrNoRows {
			return nil
		}
		return fmt.Errorf("failed to check user email for password reset: %w", err)
	}

	otpCode, err := generateNumericOTP(6)
	if err != nil {
		return fmt.Errorf("failed to generate otp: %w", err)
	}

	otpHash, err := bcrypt.GenerateFromPassword([]byte(otpCode), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash otp: %w", err)
	}

	expiresAt := time.Now().UTC().Add(time.Duration(service.otpExpiryMinutes) * time.Minute)

	tx, err := service.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start password reset transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const expirePreviousQuery = `
		UPDATE users.password_reset_otp
		SET
			used_at = NOW(),
			updated_at = NOW()
		WHERE user_id = $1
		  AND used_at IS NULL`

	if _, err := tx.Exec(ctx, expirePreviousQuery, userID); err != nil {
		return fmt.Errorf("failed to expire previous otp records: %w", err)
	}

	const insertOTPQuery = `
		INSERT INTO users.password_reset_otp (
			user_id,
			email,
			otp_hash,
			otp_expires_at
		)
		VALUES ($1, $2, $3, $4)`

	if _, err := tx.Exec(ctx, insertOTPQuery, userID, normalizedEmail, string(otpHash), expiresAt); err != nil {
		return fmt.Errorf("failed to create otp record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit otp transaction: %w", err)
	}

	if err := service.sendPasswordResetOTPEmail(normalizedEmail, userName, otpCode, expiresAt); err != nil {
		return err
	}

	return nil
}

func (service *PasswordResetService) VerifyPasswordResetOTP(ctx context.Context, email, otpCode string) (string, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	if normalizedEmail == "" || !isValidEmail(normalizedEmail) {
		return "", ValidationError{Message: "A valid email address is required."}
	}

	normalizedOTP := strings.TrimSpace(otpCode)
	if !otpCodeRegex.MatchString(normalizedOTP) {
		return "", ValidationError{Message: "OTP must be a 6-digit code."}
	}

	const latestOTPQuery = `
		SELECT
			password_reset_otp_id,
			otp_hash,
			otp_expires_at,
			verify_attempts
		FROM users.password_reset_otp
		WHERE LOWER(email) = LOWER($1)
		  AND used_at IS NULL
		  AND verified_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1`

	var otpID string
	var otpHash string
	var otpExpiresAt time.Time
	var verifyAttempts int
	if err := service.db.QueryRow(ctx, latestOTPQuery, normalizedEmail).Scan(&otpID, &otpHash, &otpExpiresAt, &verifyAttempts); err != nil {
		if err == pgx.ErrNoRows {
			return "", ValidationError{Message: "No active OTP found. Please request a new OTP."}
		}
		return "", fmt.Errorf("failed to fetch otp record: %w", err)
	}

	now := time.Now().UTC()
	if now.After(otpExpiresAt) {
		if _, err := service.db.Exec(ctx, `UPDATE users.password_reset_otp SET used_at = NOW(), updated_at = NOW() WHERE password_reset_otp_id = $1`, otpID); err != nil {
			return "", fmt.Errorf("failed to expire otp record: %w", err)
		}
		return "", ValidationError{Message: "OTP has expired. Please request a new OTP."}
	}

	if verifyAttempts >= service.maxOTPVerificationAttempts {
		if _, err := service.db.Exec(ctx, `UPDATE users.password_reset_otp SET used_at = NOW(), updated_at = NOW() WHERE password_reset_otp_id = $1`, otpID); err != nil {
			return "", fmt.Errorf("failed to mark otp record as locked: %w", err)
		}
		return "", ValidationError{Message: "OTP verification attempts exceeded. Please request a new OTP."}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(otpHash), []byte(normalizedOTP)); err != nil {
		nextAttempts := verifyAttempts + 1
		if nextAttempts >= service.maxOTPVerificationAttempts {
			if _, updateErr := service.db.Exec(
				ctx,
				`UPDATE users.password_reset_otp SET verify_attempts = $2, used_at = NOW(), updated_at = NOW() WHERE password_reset_otp_id = $1`,
				otpID,
				nextAttempts,
			); updateErr != nil {
				return "", fmt.Errorf("failed to update otp attempts: %w", updateErr)
			}
			return "", ValidationError{Message: "OTP verification attempts exceeded. Please request a new OTP."}
		}

		if _, updateErr := service.db.Exec(
			ctx,
			`UPDATE users.password_reset_otp SET verify_attempts = $2, updated_at = NOW() WHERE password_reset_otp_id = $1`,
			otpID,
			nextAttempts,
		); updateErr != nil {
			return "", fmt.Errorf("failed to update otp attempts: %w", updateErr)
		}

		return "", ValidationError{Message: "Invalid OTP. Please try again."}
	}

	resetToken, resetTokenHash, err := generateResetToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate reset token: %w", err)
	}

	resetTokenExpiresAt := now.Add(time.Duration(service.resetTokenExpiryMinutes) * time.Minute)

	const verifyQuery = `
		UPDATE users.password_reset_otp
		SET
			verified_at = NOW(),
			reset_token_hash = $2,
			reset_token_expires_at = $3,
			updated_at = NOW()
		WHERE password_reset_otp_id = $1`

	if _, err := service.db.Exec(ctx, verifyQuery, otpID, resetTokenHash, resetTokenExpiresAt); err != nil {
		return "", fmt.Errorf("failed to verify otp: %w", err)
	}

	return resetToken, nil
}

func (service *PasswordResetService) ResetPassword(ctx context.Context, resetToken, newPassword string) error {
	normalizedResetToken := strings.TrimSpace(resetToken)
	if normalizedResetToken == "" {
		return ValidationError{Message: "Reset token is required."}
	}

	normalizedPassword := strings.TrimSpace(newPassword)
	if len(normalizedPassword) < 8 {
		return ValidationError{Message: "Password must be at least 8 characters long."}
	}

	resetTokenHash := hashResetToken(normalizedResetToken)
	if resetTokenHash == "" {
		return ValidationError{Message: "Reset token is invalid."}
	}

	tx, err := service.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start reset password transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const tokenQuery = `
		SELECT
			password_reset_otp_id,
			user_id,
			reset_token_expires_at
		FROM users.password_reset_otp
		WHERE reset_token_hash = $1
		  AND verified_at IS NOT NULL
		  AND used_at IS NULL
		ORDER BY verified_at DESC
		LIMIT 1
		FOR UPDATE`

	var otpID string
	var userID string
	var resetTokenExpiresAt *time.Time
	if err := tx.QueryRow(ctx, tokenQuery, resetTokenHash).Scan(&otpID, &userID, &resetTokenExpiresAt); err != nil {
		if err == pgx.ErrNoRows {
			return ValidationError{Message: "Reset token is invalid or expired."}
		}
		return fmt.Errorf("failed to validate reset token: %w", err)
	}

	now := time.Now().UTC()
	if resetTokenExpiresAt == nil || now.After(*resetTokenExpiresAt) {
		if _, err := tx.Exec(ctx, `UPDATE users.password_reset_otp SET used_at = NOW(), updated_at = NOW() WHERE password_reset_otp_id = $1`, otpID); err != nil {
			return fmt.Errorf("failed to expire reset token: %w", err)
		}
		return ValidationError{Message: "Reset token has expired. Please verify OTP again."}
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(normalizedPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	updateResult, err := tx.Exec(ctx, `UPDATE users."user" SET password_hash = $1, updated_at = NOW() WHERE id = $2`, string(passwordHash), userID)
	if err != nil {
		return fmt.Errorf("failed to update user password: %w", err)
	}
	if updateResult.RowsAffected() == 0 {
		return ValidationError{Message: "User account not found for this reset token."}
	}

	if _, err := tx.Exec(ctx, `UPDATE users.password_reset_otp SET used_at = NOW(), updated_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return fmt.Errorf("failed to finalize reset token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit password reset transaction: %w", err)
	}

	return nil
}

func (service *PasswordResetService) sendPasswordResetOTPEmail(recipientEmail, recipientName, otpCode string, expiresAt time.Time) error {
	if !service.isEmailNotificationConfigured {
		return fmt.Errorf("password reset email service is not configured")
	}

	trimmedName := strings.TrimSpace(recipientName)
	if trimmedName == "" {
		trimmedName = "User"
	}

	subject := "Your RecurIN Password Reset OTP"
	body := fmt.Sprintf(
		"Hello %s,\r\n\r\nYour OTP for password reset is: %s\r\n\r\nThis OTP is valid for %d minutes (until %s UTC).\r\nIf you did not request this, you can ignore this email.\r\n\r\nRegards,\r\nRecurIN Support",
		trimmedName,
		otpCode,
		service.otpExpiryMinutes,
		expiresAt.UTC().Format("2006-01-02 15:04:05"),
	)

	var messageBuilder strings.Builder
	messageBuilder.WriteString("From: " + service.smtpFromName + " <" + service.smtpFromEmail + ">\r\n")
	messageBuilder.WriteString("To: <" + recipientEmail + ">\r\n")
	messageBuilder.WriteString("Subject: " + subject + "\r\n")
	messageBuilder.WriteString("MIME-Version: 1.0\r\n")
	messageBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	messageBuilder.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
	messageBuilder.WriteString(body)

	smtpAddress := fmt.Sprintf("%s:%d", service.smtpHost, service.smtpPort)
	var smtpAuth smtp.Auth
	if service.smtpUsername != "" && service.smtpPassword != "" {
		smtpAuth = smtp.PlainAuth("", service.smtpUsername, service.smtpPassword, service.smtpHost)
	}

	if err := smtp.SendMail(smtpAddress, smtpAuth, service.smtpFromEmail, []string{recipientEmail}, []byte(messageBuilder.String())); err != nil {
		return fmt.Errorf("failed to send password reset otp email: %w", err)
	}

	return nil
}

func generateNumericOTP(length int) (string, error) {
	if length <= 0 {
		length = 6
	}

	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	digits := make([]byte, length)
	for index := range randomBytes {
		digits[index] = byte('0' + (randomBytes[index] % 10))
	}

	return string(digits), nil
}

func generateResetToken() (string, string, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", err
	}

	resetToken := base64.RawURLEncoding.EncodeToString(randomBytes)
	return resetToken, hashResetToken(resetToken), nil
}

func hashResetToken(token string) string {
	trimmedToken := strings.TrimSpace(token)
	if trimmedToken == "" {
		return ""
	}

	hash := sha256.Sum256([]byte(trimmedToken))
	return hex.EncodeToString(hash[:])
}
