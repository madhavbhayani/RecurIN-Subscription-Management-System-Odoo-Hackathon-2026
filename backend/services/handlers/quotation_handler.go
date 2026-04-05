package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services"
)

type quotationRequest struct {
	LastForever           bool     `json:"last_forever"`
	QuotationValidityDays *int     `json:"quotation_validity_days"`
	RecurringPlanID       string   `json:"recurring_plan_id"`
	ProductIDs            []string `json:"product_ids"`
}

// QuotationHandler handles quotation administration endpoints.
type QuotationHandler struct {
	quotationService *services.QuotationService
}

func NewQuotationHandler(quotationService *services.QuotationService) *QuotationHandler {
	return &QuotationHandler{quotationService: quotationService}
}

func mapQuotationInput(payload quotationRequest) services.CreateQuotationInput {
	return services.CreateQuotationInput{
		LastForever:           payload.LastForever,
		QuotationValidityDays: payload.QuotationValidityDays,
		RecurringPlanID:       payload.RecurringPlanID,
		ProductIDs:            payload.ProductIDs,
	}
}

func (handler *QuotationHandler) HandleCreateQuotation(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	var payload quotationRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	createdQuotation, err := handler.quotationService.CreateQuotation(request.Context(), mapQuotationInput(payload))
	if err != nil {
		handler.writeQuotationError(writer, err)
		return
	}

	writeJSON(writer, http.StatusCreated, map[string]interface{}{
		"message":   "Quotation created successfully.",
		"quotation": buildQuotationResponse(createdQuotation),
	})
}

func (handler *QuotationHandler) HandleListQuotations(writer http.ResponseWriter, request *http.Request) {
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

	quotations, totalRecords, err := handler.quotationService.ListQuotations(request.Context(), search, pageForQuery, adminListPageSize)
	if err != nil {
		log.Printf("quotation list handler error: %v", err)
		http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
		return
	}

	items := make([]map[string]interface{}, 0, len(quotations))
	for _, quotation := range quotations {
		items = append(items, buildQuotationResponse(quotation))
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"quotations": items,
		"pagination": buildPaginationResponse(pageForResponse, adminListPageSize, totalRecords),
	})
}

func (handler *QuotationHandler) HandleGetQuotationByID(writer http.ResponseWriter, request *http.Request) {
	quotationID := request.PathValue("quotationID")

	quotation, err := handler.quotationService.GetQuotationByID(request.Context(), quotationID)
	if err != nil {
		handler.writeQuotationError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"quotation": buildQuotationResponse(quotation),
	})
}

func (handler *QuotationHandler) HandleUpdateQuotation(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	quotationID := request.PathValue("quotationID")

	var payload quotationRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	updatedQuotation, err := handler.quotationService.UpdateQuotation(request.Context(), quotationID, mapQuotationInput(payload))
	if err != nil {
		handler.writeQuotationError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"message":   "Quotation updated successfully.",
		"quotation": buildQuotationResponse(updatedQuotation),
	})
}

func (handler *QuotationHandler) HandleDeleteQuotation(writer http.ResponseWriter, request *http.Request) {
	quotationID := request.PathValue("quotationID")

	if err := handler.quotationService.DeleteQuotation(request.Context(), quotationID); err != nil {
		handler.writeQuotationError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]string{
		"message": "Quotation deleted successfully.",
	})
}

func (handler *QuotationHandler) writeQuotationError(writer http.ResponseWriter, err error) {
	var validationError services.ValidationError
	if errors.As(err, &validationError) {
		http.Error(writer, validationError.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, services.ErrQuotationNotFound) {
		http.Error(writer, "Quotation not found.", http.StatusNotFound)
		return
	}

	log.Printf("quotation handler error: %v", err)
	http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
}

func buildQuotationResponse(quotation models.Quotation) map[string]interface{} {
	products := make([]map[string]interface{}, 0, len(quotation.Products))
	for _, product := range quotation.Products {
		products = append(products, map[string]interface{}{
			"product_id":   product.ProductID,
			"product_name": product.ProductName,
			"product_type": product.ProductType,
			"sales_price":  product.SalesPrice,
		})
	}

	productCount := quotation.ProductCount
	if productCount == 0 && len(products) > 0 {
		productCount = len(products)
	}

	return map[string]interface{}{
		"quotation_id":            quotation.QuotationID,
		"last_forever":            quotation.LastForever,
		"quotation_validity_days": quotation.QuotationValidityDays,
		"recurring_plan_id":       quotation.RecurringPlanID,
		"recurring_plan_name":     quotation.RecurringPlanName,
		"product_count":           productCount,
		"products":                products,
		"created_at":              quotation.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":              quotation.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
