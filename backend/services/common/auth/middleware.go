package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const claimsContextKey contextKey = "authClaims"

// AuthMiddleware verifies bearer tokens and injects claims into request context.
func AuthMiddleware(tokenManager *TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			authHeader := request.Header.Get("Authorization")
			if strings.TrimSpace(authHeader) == "" {
				http.Error(writer, "missing Authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(writer, "Authorization header must be in Bearer format", http.StatusUnauthorized)
				return
			}

			claims, err := tokenManager.ValidateToken(parts[1])
			if err != nil {
				http.Error(writer, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctxWithClaims := context.WithValue(request.Context(), claimsContextKey, claims)
			next.ServeHTTP(writer, request.WithContext(ctxWithClaims))
		})
	}
}

// ClaimsFromContext reads JWT claims from request context.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	if !ok || claims == nil {
		return nil, false
	}

	return claims, true
}
