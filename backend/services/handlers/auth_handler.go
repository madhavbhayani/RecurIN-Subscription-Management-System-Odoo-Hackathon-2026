package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/queue"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services/common/auth"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	tokenManager *auth.TokenManager
	queue        *queue.WorkerPool
	userService  *services.UserService
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type signupRequest struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	CountryCode string `json:"country_code"`
	PhoneNumber string `json:"phone_number"`
}

type authResult struct {
	User      models.User
	Token     string
	ExpiresAt time.Time
	Err       error
}

func NewAuthHandler(tokenManager *auth.TokenManager, workerPool *queue.WorkerPool, userService *services.UserService) *AuthHandler {
	return &AuthHandler{tokenManager: tokenManager, queue: workerPool, userService: userService}
}

func (handler *AuthHandler) HandleSignup(writer http.ResponseWriter, request *http.Request) {
	var payload signupRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		http.Error(writer, "invalid request payload", http.StatusBadRequest)
		return
	}

	resultChannel := make(chan authResult, 1)
	err := handler.queue.Submit(func(jobCtx context.Context) error {
		user, err := handler.userService.CreateUser(jobCtx, services.CreateUserInput{
			Name:        payload.Name,
			Email:       payload.Email,
			Password:    payload.Password,
			CountryCode: payload.CountryCode,
			PhoneNumber: payload.PhoneNumber,
		})
		if err != nil {
			resultChannel <- authResult{Err: err}
			return nil
		}

		tokenRole := handler.userService.TokenRoleFromDBRole(user.Role)
		token, expiresAt, err := handler.tokenManager.GenerateToken(user.ID, tokenRole)
		if err != nil {
			resultChannel <- authResult{Err: err}
			return nil
		}

		resultChannel <- authResult{User: user, Token: token, ExpiresAt: expiresAt}
		return nil
	})
	if err != nil {
		if errors.Is(err, queue.ErrQueueFull) {
			http.Error(writer, "server is busy, try again", http.StatusServiceUnavailable)
			return
		}
		http.Error(writer, "failed to process signup", http.StatusInternalServerError)
		return
	}

	select {
	case <-request.Context().Done():
		http.Error(writer, "request canceled", http.StatusRequestTimeout)
	case result := <-resultChannel:
		if result.Err != nil {
			handler.writeAuthError(writer, result.Err)
			return
		}

		writeJSON(writer, http.StatusCreated, map[string]interface{}{
			"access_token": result.Token,
			"token_type":   "Bearer",
			"expires_at":   result.ExpiresAt.UTC().Format(time.RFC3339),
			"user":         buildUserResponse(result.User),
		})
	}
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

	resultChannel := make(chan authResult, 1)
	err := handler.queue.Submit(func(jobCtx context.Context) error {
		user, err := handler.userService.AuthenticateUser(jobCtx, payload.Email, payload.Password)
		if err != nil {
			resultChannel <- authResult{Err: err}
			return nil
		}

		tokenRole := handler.userService.TokenRoleFromDBRole(user.Role)
		token, expiresAt, err := handler.tokenManager.GenerateToken(user.ID, tokenRole)
		if err != nil {
			resultChannel <- authResult{Err: err}
			return nil
		}

		resultChannel <- authResult{User: user, Token: token, ExpiresAt: expiresAt}
		return nil
	})
	if err != nil {
		if errors.Is(err, queue.ErrQueueFull) {
			http.Error(writer, "server is busy, try again", http.StatusServiceUnavailable)
			return
		}
		http.Error(writer, "failed to process login", http.StatusInternalServerError)
		return
	}

	select {
	case <-request.Context().Done():
		http.Error(writer, "request canceled", http.StatusRequestTimeout)
	case result := <-resultChannel:
		if result.Err != nil {
			handler.writeAuthError(writer, result.Err)
			return
		}

		writeJSON(writer, http.StatusOK, map[string]interface{}{
			"access_token": result.Token,
			"token_type":   "Bearer",
			"expires_at":   result.ExpiresAt.UTC().Format(time.RFC3339),
			"user":         buildUserResponse(result.User),
		})
	}
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

func (handler *AuthHandler) writeAuthError(writer http.ResponseWriter, err error) {
	var validationError services.ValidationError
	if errors.As(err, &validationError) {
		http.Error(writer, validationError.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, services.ErrEmailAlreadyExists) {
		http.Error(writer, "email is already registered", http.StatusConflict)
		return
	}
	if errors.Is(err, services.ErrInvalidCredentials) {
		http.Error(writer, "invalid email or password", http.StatusUnauthorized)
		return
	}

	log.Printf("auth handler error: %v", err)
	http.Error(writer, "request processing failed", http.StatusInternalServerError)
}

func buildUserResponse(user models.User) map[string]interface{} {
	response := map[string]interface{}{
		"id":           user.ID,
		"name":         user.Name,
		"email":        user.Email,
		"phone_number": user.PhoneNumber,
		"role":         user.Role,
		"created_at":   user.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":   user.UpdatedAt.UTC().Format(time.RFC3339),
	}

	if user.Address == nil {
		response["address"] = nil
	} else {
		response["address"] = *user.Address
	}

	return response
}
