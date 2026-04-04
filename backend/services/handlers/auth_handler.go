package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services/common/auth"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	tokenManager *auth.TokenManager
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewAuthHandler(tokenManager *auth.TokenManager) *AuthHandler {
	return &AuthHandler{tokenManager: tokenManager}
}

func (handler *AuthHandler) HandleLogin(writer http.ResponseWriter, request *http.Request) {
	var payload loginRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		http.Error(writer, "invalid request payload", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(payload.Email) == "" || strings.TrimSpace(payload.Password) == "" {
		http.Error(writer, "email and password are required", http.StatusBadRequest)
		return
	}

	role := deriveRoleFromEmail(payload.Email)
	token, expiresAt, err := handler.tokenManager.GenerateToken(payload.Email, role)
	if err != nil {
		http.Error(writer, "failed to issue token", http.StatusInternalServerError)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_at":   expiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"role":         role,
	})
}

func (handler *AuthHandler) HandleWhoAmI(writer http.ResponseWriter, request *http.Request) {
	claims, ok := auth.ClaimsFromContext(request.Context())
	if !ok {
		http.Error(writer, "missing claims", http.StatusUnauthorized)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"user_id": claims.UserID,
		"role":    claims.Role,
	})
}

func (handler *AuthHandler) HandleAdminPing(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{
		"message": "admin access granted",
	})
}

func deriveRoleFromEmail(email string) string {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))

	if strings.HasSuffix(normalizedEmail, "@admin.com") {
		return "admin"
	}
	if strings.Contains(normalizedEmail, "internal") {
		return "internal-user"
	}

	return "portal-user"
}
