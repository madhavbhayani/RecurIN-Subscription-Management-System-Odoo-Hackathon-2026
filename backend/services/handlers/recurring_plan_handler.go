package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services"
)

type recurringPlanRequest struct {
	RecurringName        string `json:"recurring_name"`
	BillingPeriod        string `json:"billing_period"`
	IsClosable           bool   `json:"is_closable"`
	AutomaticCloseCycles *int   `json:"automatic_close_cycles"`
	IsPausable           bool   `json:"is_pausable"`
	IsRenewable          bool   `json:"is_renewable"`
	IsActive             bool   `json:"is_active"`
}

// RecurringPlanHandler handles recurring plan administration endpoints.
type RecurringPlanHandler struct {
	recurringPlanService *services.RecurringPlanService
}

func NewRecurringPlanHandler(recurringPlanService *services.RecurringPlanService) *RecurringPlanHandler {
	return &RecurringPlanHandler{recurringPlanService: recurringPlanService}
}

func mapRecurringPlanInput(payload recurringPlanRequest) services.CreateRecurringPlanInput {
	return services.CreateRecurringPlanInput{
		RecurringName:        payload.RecurringName,
		BillingPeriod:        payload.BillingPeriod,
		IsClosable:           payload.IsClosable,
		AutomaticCloseCycles: payload.AutomaticCloseCycles,
		IsPausable:           payload.IsPausable,
		IsRenewable:          payload.IsRenewable,
		IsActive:             payload.IsActive,
	}
}

func (handler *RecurringPlanHandler) HandleCreateRecurringPlan(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	var payload recurringPlanRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	createdRecurringPlan, err := handler.recurringPlanService.CreateRecurringPlan(request.Context(), mapRecurringPlanInput(payload))
	if err != nil {
		handler.writeRecurringPlanError(writer, err)
		return
	}

	writeJSON(writer, http.StatusCreated, map[string]interface{}{
		"message":        "Recurring plan created successfully.",
		"recurring_plan": buildRecurringPlanResponse(createdRecurringPlan),
	})
}

func (handler *RecurringPlanHandler) HandleListRecurringPlans(writer http.ResponseWriter, request *http.Request) {
	search := request.URL.Query().Get("search")
	activeOnly := false

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

	activeOnlyValue := strings.TrimSpace(request.URL.Query().Get("active_only"))
	if activeOnlyValue != "" {
		parsedActiveOnly, err := strconv.ParseBool(activeOnlyValue)
		if err != nil {
			http.Error(writer, "active_only must be either true or false.", http.StatusBadRequest)
			return
		}
		activeOnly = parsedActiveOnly
	}

	recurringPlans, totalRecords, err := handler.recurringPlanService.ListRecurringPlans(request.Context(), search, activeOnly, pageForQuery, adminListPageSize)
	if err != nil {
		log.Printf("recurring plan list handler error: %v", err)
		http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
		return
	}

	items := make([]map[string]interface{}, 0, len(recurringPlans))
	for _, recurringPlan := range recurringPlans {
		items = append(items, buildRecurringPlanResponse(recurringPlan))
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"recurring_plans": items,
		"pagination":      buildPaginationResponse(pageForResponse, adminListPageSize, totalRecords),
	})
}

func (handler *RecurringPlanHandler) HandleGetRecurringPlanByID(writer http.ResponseWriter, request *http.Request) {
	recurringPlanID := request.PathValue("recurringPlanID")

	recurringPlan, err := handler.recurringPlanService.GetRecurringPlanByID(request.Context(), recurringPlanID)
	if err != nil {
		handler.writeRecurringPlanError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"recurring_plan": buildRecurringPlanResponse(recurringPlan),
	})
}

func (handler *RecurringPlanHandler) HandleUpdateRecurringPlan(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	recurringPlanID := request.PathValue("recurringPlanID")

	var payload recurringPlanRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	updatedRecurringPlan, err := handler.recurringPlanService.UpdateRecurringPlan(request.Context(), recurringPlanID, mapRecurringPlanInput(payload))
	if err != nil {
		handler.writeRecurringPlanError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"message":        "Recurring plan updated successfully.",
		"recurring_plan": buildRecurringPlanResponse(updatedRecurringPlan),
	})
}

func (handler *RecurringPlanHandler) HandleDeleteRecurringPlan(writer http.ResponseWriter, request *http.Request) {
	recurringPlanID := request.PathValue("recurringPlanID")

	if err := handler.recurringPlanService.DeleteRecurringPlan(request.Context(), recurringPlanID); err != nil {
		handler.writeRecurringPlanError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]string{
		"message": "Recurring plan deleted successfully.",
	})
}

func (handler *RecurringPlanHandler) writeRecurringPlanError(writer http.ResponseWriter, err error) {
	var validationError services.ValidationError
	if errors.As(err, &validationError) {
		http.Error(writer, validationError.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, services.ErrRecurringPlanNotFound) {
		http.Error(writer, "Recurring plan not found.", http.StatusNotFound)
		return
	}
	if errors.Is(err, services.ErrRecurringPlanAlreadyExists) {
		http.Error(writer, "Recurring plan name already exists.", http.StatusConflict)
		return
	}

	log.Printf("recurring plan handler error: %v", err)
	http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
}

func buildRecurringPlanResponse(recurringPlan models.RecurringPlan) map[string]interface{} {
	products := make([]map[string]interface{}, 0, len(recurringPlan.Products))
	for _, product := range recurringPlan.Products {
		products = append(products, map[string]interface{}{
			"product_id":   product.ProductID,
			"product_name": product.ProductName,
			"product_type": product.ProductType,
			"sales_price":  product.SalesPrice,
			"min_qty":      product.MinQty,
		})
	}

	productCount := recurringPlan.ProductCount
	if productCount == 0 && len(products) > 0 {
		productCount = len(products)
	}

	return map[string]interface{}{
		"recurring_plan_id":      recurringPlan.RecurringPlanID,
		"recurring_name":         recurringPlan.RecurringName,
		"billing_period":         recurringPlan.BillingPeriod,
		"is_closable":            recurringPlan.IsClosable,
		"automatic_close_cycles": recurringPlan.AutomaticCloseCycles,
		"is_pausable":            recurringPlan.IsPausable,
		"is_renewable":           recurringPlan.IsRenewable,
		"is_active":              recurringPlan.IsActive,
		"product_count":          productCount,
		"products":               products,
		"created_at":             recurringPlan.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":             recurringPlan.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
