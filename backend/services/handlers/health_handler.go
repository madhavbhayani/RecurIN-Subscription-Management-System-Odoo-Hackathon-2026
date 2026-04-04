package handlers

import (
	"net/http"
	"time"
)

// HealthHandler provides system health endpoints.
type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (handler *HealthHandler) HandleHealth(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"service":   "recurin-subscription-management",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
