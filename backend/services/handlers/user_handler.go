package handlers

import (
	"log"
	"net/http"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services"
)

// UserHandler handles user administration endpoints.
type UserHandler struct {
	userService *services.UserService
}

func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (handler *UserHandler) HandleListCustomerUsers(writer http.ResponseWriter, request *http.Request) {
	search := request.URL.Query().Get("search")

	users, err := handler.userService.ListUsersByRole(request.Context(), models.RoleUser, search, 5)
	if err != nil {
		log.Printf("customer user list handler error: %v", err)
		http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
		return
	}

	items := make([]map[string]interface{}, 0, len(users))
	for _, user := range users {
		items = append(items, map[string]interface{}{
			"id":           user.ID,
			"name":         user.Name,
			"email":        user.Email,
			"phone_number": user.PhoneNumber,
			"role":         user.Role,
		})
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"users": items,
	})
}
