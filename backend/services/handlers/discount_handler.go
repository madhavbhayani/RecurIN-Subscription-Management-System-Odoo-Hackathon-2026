package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services"
)

type discountRequest struct {
	DiscountName    string   `json:"discount_name"`
	DiscountUnit    string   `json:"discount_unit"`
	DiscountValue   float64  `json:"discount_value"`
	MinimumPurchase float64  `json:"minimum_purchase"`
	MaximumPurchase float64  `json:"maximum_purchase"`
	StartDate       string   `json:"start_date"`
	EndDate         string   `json:"end_date"`
	IsLimit         bool     `json:"is_limit"`
	LimitUsers      *int     `json:"limit_users"`
	IsActive        *bool    `json:"is_active"`
	ProductIDs      []string `json:"product_ids"`
}

// DiscountHandler handles discount administration endpoints.
type DiscountHandler struct {
	discountService *services.DiscountService
}

func NewDiscountHandler(discountService *services.DiscountService) *DiscountHandler {
	return &DiscountHandler{discountService: discountService}
}

func parseDate(value string, fieldName string) (time.Time, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return time.Time{}, services.ValidationError{Message: fieldName + " is required."}
	}

	parsedDate, err := time.Parse("2006-01-02", trimmedValue)
	if err != nil {
		return time.Time{}, services.ValidationError{Message: fieldName + " must be in YYYY-MM-DD format."}
	}

	return parsedDate, nil
}

func mapDiscountInput(payload discountRequest) (services.CreateDiscountInput, error) {
	startDate, err := parseDate(payload.StartDate, "Start date")
	if err != nil {
		return services.CreateDiscountInput{}, err
	}

	endDate, err := parseDate(payload.EndDate, "End date")
	if err != nil {
		return services.CreateDiscountInput{}, err
	}

	isActive := true
	if payload.IsActive != nil {
		isActive = *payload.IsActive
	}

	return services.CreateDiscountInput{
		DiscountName:    payload.DiscountName,
		DiscountUnit:    payload.DiscountUnit,
		DiscountValue:   payload.DiscountValue,
		MinimumPurchase: payload.MinimumPurchase,
		MaximumPurchase: payload.MaximumPurchase,
		StartDate:       startDate,
		EndDate:         endDate,
		IsLimit:         payload.IsLimit,
		LimitUsers:      payload.LimitUsers,
		IsActive:        isActive,
		ProductIDs:      payload.ProductIDs,
	}, nil
}

func (handler *DiscountHandler) HandleCreateDiscount(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	var payload discountRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	input, err := mapDiscountInput(payload)
	if err != nil {
		handler.writeDiscountError(writer, err)
		return
	}

	createdDiscount, err := handler.discountService.CreateDiscount(request.Context(), input)
	if err != nil {
		handler.writeDiscountError(writer, err)
		return
	}

	writeJSON(writer, http.StatusCreated, map[string]interface{}{
		"message":  "Discount created successfully.",
		"discount": buildDiscountResponse(createdDiscount),
	})
}

func (handler *DiscountHandler) HandleListDiscounts(writer http.ResponseWriter, request *http.Request) {
	search := request.URL.Query().Get("search")

	page, hasPage, err := parsePageQuery(request)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	pageForQuery := 0
	pageForResponse := 1
	if hasPage {
		pageForQuery = page
		pageForResponse = page
	}

	discounts, totalRecords, err := handler.discountService.ListDiscounts(request.Context(), search, pageForQuery, adminListPageSize)
	if err != nil {
		log.Printf("discount list handler error: %v", err)
		http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
		return
	}

	items := make([]map[string]interface{}, 0, len(discounts))
	for _, discount := range discounts {
		items = append(items, buildDiscountResponse(discount))
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"discounts":   items,
		"pagination": buildPaginationResponse(pageForResponse, adminListPageSize, totalRecords),
	})
}

func (handler *DiscountHandler) HandleGetDiscountByID(writer http.ResponseWriter, request *http.Request) {
	discountID := request.PathValue("discountID")

	discount, err := handler.discountService.GetDiscountByID(request.Context(), discountID)
	if err != nil {
		handler.writeDiscountError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"discount": buildDiscountResponse(discount),
	})
}

func (handler *DiscountHandler) HandleUpdateDiscount(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	discountID := request.PathValue("discountID")

	var payload discountRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	input, err := mapDiscountInput(payload)
	if err != nil {
		handler.writeDiscountError(writer, err)
		return
	}

	updatedDiscount, err := handler.discountService.UpdateDiscount(request.Context(), discountID, input)
	if err != nil {
		handler.writeDiscountError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"message":  "Discount updated successfully.",
		"discount": buildDiscountResponse(updatedDiscount),
	})
}

func (handler *DiscountHandler) HandleDeleteDiscount(writer http.ResponseWriter, request *http.Request) {
	discountID := request.PathValue("discountID")

	if err := handler.discountService.DeleteDiscount(request.Context(), discountID); err != nil {
		handler.writeDiscountError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]string{
		"message": "Discount deleted successfully.",
	})
}

func (handler *DiscountHandler) writeDiscountError(writer http.ResponseWriter, err error) {
	var validationError services.ValidationError
	if errors.As(err, &validationError) {
		http.Error(writer, validationError.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, services.ErrDiscountNotFound) {
		http.Error(writer, "Discount not found.", http.StatusNotFound)
		return
	}
	if errors.Is(err, services.ErrDiscountAlreadyExists) {
		http.Error(writer, "Discount name already exists.", http.StatusConflict)
		return
	}

	log.Printf("discount handler error: %v", err)
	http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
}

func buildDiscountResponse(discount models.Discount) map[string]interface{} {
	products := make([]map[string]interface{}, 0, len(discount.Products))
	for _, product := range discount.Products {
		products = append(products, map[string]interface{}{
			"product_id":   product.ProductID,
			"product_name": product.ProductName,
			"product_type": product.ProductType,
			"sales_price":  product.SalesPrice,
		})
	}

	status := "inactive"
	if discount.IsActive {
		status = "active"
	}

	productCount := discount.ProductCount
	if productCount == 0 && len(products) > 0 {
		productCount = len(products)
	}

	return map[string]interface{}{
		"discount_id":        discount.DiscountID,
		"discount_name":      discount.DiscountName,
		"discount_unit":      discount.DiscountUnit,
		"discount_value":     discount.DiscountValue,
		"minimum_purchase":   discount.MinimumPurchase,
		"maximum_purchase":   discount.MaximumPurchase,
		"start_date":         discount.StartDate.Format("2006-01-02"),
		"end_date":           discount.EndDate.Format("2006-01-02"),
		"is_limit":           discount.IsLimit,
		"limit_users":        discount.LimitUsers,
		"applied_user_count": discount.AppliedUserCount,
		"is_active":          discount.IsActive,
		"status":             status,
		"product_count":      productCount,
		"products":           products,
		"created_at":         discount.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":         discount.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
