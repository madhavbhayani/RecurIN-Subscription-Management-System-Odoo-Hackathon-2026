package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims stores identity and role information for authorization decisions.
type Claims struct {
	UserID string `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// TokenManager handles JWT generation and validation.
type TokenManager struct {
	secret   []byte
	issuer   string
	audience string
	expiry   time.Duration
}

// NewTokenManager initializes token manager with validation defaults.
func NewTokenManager(secret, issuer, audience string, expiryMinutes int) (*TokenManager, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, errors.New("JWT secret is required")
	}
	if strings.TrimSpace(issuer) == "" {
		return nil, errors.New("JWT issuer is required")
	}
	if strings.TrimSpace(audience) == "" {
		return nil, errors.New("JWT audience is required")
	}
	if expiryMinutes <= 0 {
		expiryMinutes = 60
	}

	return &TokenManager{
		secret:   []byte(secret),
		issuer:   issuer,
		audience: audience,
		expiry:   time.Duration(expiryMinutes) * time.Minute,
	}, nil
}

// GenerateToken creates a signed JWT for a user and role.
func (tm *TokenManager) GenerateToken(userID, role string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(tm.expiry)

	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tm.issuer,
			Audience:  jwt.ClaimStrings{tm.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(tm.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign JWT: %w", err)
	}

	return signedToken, expiresAt, nil
}

// ValidateToken parses and validates a JWT string.
func (tm *TokenManager) ValidateToken(tokenString string) (*Claims, error) {
	parsedClaims := &Claims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		parsedClaims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return tm.secret, nil
		},
		jwt.WithAudience(tm.audience),
		jwt.WithIssuer(tm.issuer),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("token is not valid")
	}

	return parsedClaims, nil
}
