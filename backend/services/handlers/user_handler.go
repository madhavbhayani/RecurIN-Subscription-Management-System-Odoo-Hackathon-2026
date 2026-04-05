package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services"
)

// UserHandler handles user administration endpoints.
type UserHandler struct {
	userService     *services.UserService
	documentService *services.SubscriptionDocumentService
}

type userUpdateRequest struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	Address     string `json:"address"`
}

type userAddressUpdateRequest struct {
	Address string `json:"address"`
}

type quotationRespondRequest struct {
	Action string `json:"action"`
}

func NewUserHandler(userService *services.UserService, documentService *services.SubscriptionDocumentService) *UserHandler {
	return &UserHandler{
		userService:     userService,
		documentService: documentService,
	}
}

func buildAdminUserResponse(user models.User) map[string]interface{} {
	return map[string]interface{}{
		"id":           user.ID,
		"name":         user.Name,
		"email":        user.Email,
		"phone_number": user.PhoneNumber,
		"address":      user.Address,
		"role":         user.Role,
		"created_at":   user.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":   user.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func buildUserActiveSubscriptionResponse(item models.UserSubscriptionSummary) map[string]interface{} {
	return map[string]interface{}{
		"subscription_id":     item.SubscriptionID,
		"subscription_number": item.SubscriptionNumber,
		"next_invoice_date":   item.NextInvoiceDate.Format("2006-01-02"),
		"recurring":           item.Recurring,
		"plan":                item.Plan,
		"status":              item.Status,
	}
}

func (handler *UserHandler) writeUserError(writer http.ResponseWriter, err error) {
	var validationError services.ValidationError
	if errors.As(err, &validationError) {
		http.Error(writer, validationError.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, services.ErrUserNotFound) {
		http.Error(writer, "User not found.", http.StatusNotFound)
		return
	}
	if errors.Is(err, services.ErrEmailAlreadyExists) {
		http.Error(writer, "Email already exists.", http.StatusConflict)
		return
	}
	if errors.Is(err, services.ErrSubscriptionNotFound) {
		http.Error(writer, "Subscription not found.", http.StatusNotFound)
		return
	}

	log.Printf("user handler error: %v", err)
	http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
}

func (handler *UserHandler) HandleListUsers(writer http.ResponseWriter, request *http.Request) {
	search := request.URL.Query().Get("search")
	limit := 100

	limitText := request.URL.Query().Get("limit")
	if limitText != "" {
		parsedLimit, err := strconv.Atoi(limitText)
		if err != nil {
			http.Error(writer, "limit must be a valid integer", http.StatusBadRequest)
			return
		}
		limit = parsedLimit
	}

	users, err := handler.userService.ListUsers(request.Context(), search, limit)
	if err != nil {
		handler.writeUserError(writer, err)
		return
	}

	items := make([]map[string]interface{}, 0, len(users))
	for _, user := range users {
		items = append(items, buildAdminUserResponse(user))
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"users": items,
	})
}

func (handler *UserHandler) HandleGetUserByID(writer http.ResponseWriter, request *http.Request) {
	userID := request.PathValue("userID")

	user, err := handler.userService.GetUserByID(request.Context(), userID)
	if err != nil {
		handler.writeUserError(writer, err)
		return
	}

	activeSubscriptions, err := handler.userService.ListActiveSubscriptionsByUserID(request.Context(), userID)
	if err != nil {
		handler.writeUserError(writer, err)
		return
	}

	activeSubscriptionItems := make([]map[string]interface{}, 0, len(activeSubscriptions))
	for _, activeSubscription := range activeSubscriptions {
		activeSubscriptionItems = append(activeSubscriptionItems, buildUserActiveSubscriptionResponse(activeSubscription))
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"user":                 buildAdminUserResponse(user),
		"active_subscriptions": activeSubscriptionItems,
	})
}

func (handler *UserHandler) HandleUpdateUser(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	userID := request.PathValue("userID")

	var payload userUpdateRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	updatedUser, err := handler.userService.UpdateUser(request.Context(), userID, services.UpdateUserInput{
		Name:        payload.Name,
		Email:       payload.Email,
		PhoneNumber: payload.PhoneNumber,
		Address:     payload.Address,
	})
	if err != nil {
		handler.writeUserError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"message": "User updated successfully.",
		"user":    buildAdminUserResponse(updatedUser),
	})
}

func (handler *UserHandler) HandleDeleteUser(writer http.ResponseWriter, request *http.Request) {
	userID := request.PathValue("userID")

	if err := handler.userService.DeleteUser(request.Context(), userID); err != nil {
		handler.writeUserError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"message": "User deleted successfully.",
	})
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
		items = append(items, buildAdminUserResponse(user))
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"users": items,
	})
}

func (handler *UserHandler) HandleGetMyProfile(writer http.ResponseWriter, request *http.Request) {
	userID, ok := getAuthenticatedUserID(request)
	if !ok {
		http.Error(writer, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := handler.userService.GetUserByID(request.Context(), userID)
	if err != nil {
		handler.writeUserError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"user": buildAdminUserResponse(user),
	})
}

func (handler *UserHandler) HandleListMySubscriptions(writer http.ResponseWriter, request *http.Request) {
	userID, ok := getAuthenticatedUserID(request)
	if !ok {
		http.Error(writer, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userSubscriptions, err := handler.userService.ListPortalSubscriptionsDetailedByUserID(request.Context(), userID)
	if err != nil {
		handler.writeUserError(writer, err)
		return
	}

	activeSubscriptionItems := make([]map[string]interface{}, 0)
	quotationSubscriptionItems := make([]map[string]interface{}, 0)

	for _, userSubscription := range userSubscriptions {
		item := buildSubscriptionResponse(userSubscription)
		normalizedStatus := strings.ToLower(strings.TrimSpace(string(userSubscription.Status)))

		if normalizedStatus == "active" || normalizedStatus == "confirmed" {
			activeSubscriptionItems = append(activeSubscriptionItems, item)
			continue
		}

		if normalizedStatus == "draft" || strings.Contains(normalizedStatus, "quotation") {
			quotationSubscriptionItems = append(quotationSubscriptionItems, item)
			continue
		}

		if normalizedStatus == "cancelled" {
			quotationSubscriptionItems = append(quotationSubscriptionItems, item)
		}
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"active_subscriptions":    activeSubscriptionItems,
		"quotation_subscriptions": quotationSubscriptionItems,
	})
}

func (handler *UserHandler) HandleUpdateMyAddress(writer http.ResponseWriter, request *http.Request) {
	userID, ok := getAuthenticatedUserID(request)
	if !ok {
		http.Error(writer, "Unauthorized", http.StatusUnauthorized)
		return
	}

	defer request.Body.Close()

	var payload userAddressUpdateRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	updatedUser, err := handler.userService.UpdateUserAddress(request.Context(), userID, payload.Address)
	if err != nil {
		handler.writeUserError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"message": "Address updated successfully.",
		"user":    buildAdminUserResponse(updatedUser),
	})
}

func buildPDFDownloadFileName(prefix, subscriptionNumber string) string {
	normalizedNumber := strings.TrimSpace(subscriptionNumber)
	if normalizedNumber == "" {
		normalizedNumber = "RecurIN"
	}

	normalizedNumber = strings.ReplaceAll(normalizedNumber, " ", "-")
	return fmt.Sprintf("%s-%s.pdf", prefix, normalizedNumber)
}

func writePDFDownloadResponse(writer http.ResponseWriter, fileName string, payload []byte) {
	writer.Header().Set("Content-Type", "application/pdf")
	writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(payload)
}

func (handler *UserHandler) HandleDownloadMyInvoicePDF(writer http.ResponseWriter, request *http.Request) {
	userID, ok := getAuthenticatedUserID(request)
	if !ok {
		http.Error(writer, "Unauthorized", http.StatusUnauthorized)
		return
	}

	subscriptionID := strings.TrimSpace(request.PathValue("subscriptionID"))
	if subscriptionID == "" {
		http.Error(writer, "Subscription ID is required.", http.StatusBadRequest)
		return
	}

	subscription, err := handler.userService.GetPortalSubscriptionByID(request.Context(), userID, subscriptionID)
	if err != nil {
		handler.writeUserError(writer, err)
		return
	}

	if handler.documentService == nil {
		http.Error(writer, "PDF document service is not configured.", http.StatusInternalServerError)
		return
	}

	pdfBytes, err := handler.documentService.GenerateInvoicePDF(subscription)
	if err != nil {
		log.Printf("invoice pdf generation error: %v", err)
		http.Error(writer, "Failed to generate invoice PDF.", http.StatusInternalServerError)
		return
	}

	writePDFDownloadResponse(writer, buildPDFDownloadFileName("Invoice", subscription.SubscriptionNumber), pdfBytes)
}

func (handler *UserHandler) HandleDownloadMyQuotationPDF(writer http.ResponseWriter, request *http.Request) {
	userID, ok := getAuthenticatedUserID(request)
	if !ok {
		http.Error(writer, "Unauthorized", http.StatusUnauthorized)
		return
	}

	subscriptionID := strings.TrimSpace(request.PathValue("subscriptionID"))
	if subscriptionID == "" {
		http.Error(writer, "Subscription ID is required.", http.StatusBadRequest)
		return
	}

	subscription, err := handler.userService.GetPortalSubscriptionByID(request.Context(), userID, subscriptionID)
	if err != nil {
		handler.writeUserError(writer, err)
		return
	}

	if handler.documentService == nil {
		http.Error(writer, "PDF document service is not configured.", http.StatusInternalServerError)
		return
	}

	pdfBytes, err := handler.documentService.GenerateQuotationPDF(subscription)
	if err != nil {
		log.Printf("quotation pdf generation error: %v", err)
		http.Error(writer, "Failed to generate quotation PDF.", http.StatusInternalServerError)
		return
	}

	writePDFDownloadResponse(writer, buildPDFDownloadFileName("Quotation", subscription.SubscriptionNumber), pdfBytes)
}

func (handler *UserHandler) HandleRespondToQuotation(writer http.ResponseWriter, request *http.Request) {
	userID, ok := getAuthenticatedUserID(request)
	if !ok {
		http.Error(writer, "Unauthorized", http.StatusUnauthorized)
		return
	}

	subscriptionID := strings.TrimSpace(request.PathValue("subscriptionID"))
	if subscriptionID == "" {
		http.Error(writer, "Subscription ID is required.", http.StatusBadRequest)
		return
	}

	defer request.Body.Close()

	var payload quotationRespondRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	updatedStatus, err := handler.userService.RespondToQuotation(request.Context(), userID, subscriptionID, payload.Action)
	if err != nil {
		handler.writeUserError(writer, err)
		return
	}

	responseMessage := "Quotation accepted successfully."
	if updatedStatus == models.SubscriptionStatusCancelled {
		responseMessage = "Quotation rejected successfully."
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"message":         responseMessage,
		"subscription_id": subscriptionID,
		"status":          string(updatedStatus),
	})
}
