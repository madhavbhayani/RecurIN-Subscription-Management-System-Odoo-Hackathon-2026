package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services"
)

type taxRequest struct {
	TaxName             string  `json:"tax_name"`
	TaxComputationUnit  string  `json:"tax_computation_unit"`
	TaxComputationValue float64 `json:"tax_computation_value"`
}

// TaxHandler handles tax administration endpoints.
type TaxHandler struct {
	taxService *services.TaxService
}

func NewTaxHandler(taxService *services.TaxService) *TaxHandler {
	return &TaxHandler{taxService: taxService}
}

func (handler *TaxHandler) HandleCreateTax(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	var payload taxRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	createdTax, err := handler.taxService.CreateTax(request.Context(), services.CreateTaxInput{
		TaxName:             payload.TaxName,
		TaxComputationUnit:  payload.TaxComputationUnit,
		TaxComputationValue: payload.TaxComputationValue,
	})
	if err != nil {
		handler.writeTaxError(writer, err)
		return
	}

	writeJSON(writer, http.StatusCreated, map[string]interface{}{
		"message": "Tax created successfully.",
		"tax":     buildTaxResponse(createdTax),
	})
}

func (handler *TaxHandler) HandleListTaxes(writer http.ResponseWriter, request *http.Request) {
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

	taxes, totalRecords, err := handler.taxService.ListTaxes(request.Context(), search, pageForQuery, adminListPageSize)
	if err != nil {
		log.Printf("tax list handler error: %v", err)
		http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
		return
	}

	items := make([]map[string]interface{}, 0, len(taxes))
	for _, tax := range taxes {
		items = append(items, buildTaxResponse(tax))
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"taxes":      items,
		"pagination": buildPaginationResponse(pageForResponse, adminListPageSize, totalRecords),
	})
}

func (handler *TaxHandler) HandleGetTaxByID(writer http.ResponseWriter, request *http.Request) {
	taxID := request.PathValue("taxID")

	tax, err := handler.taxService.GetTaxByID(request.Context(), taxID)
	if err != nil {
		handler.writeTaxError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"tax": buildTaxResponse(tax),
	})
}

func (handler *TaxHandler) HandleUpdateTax(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	taxID := request.PathValue("taxID")

	var payload taxRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	updatedTax, err := handler.taxService.UpdateTax(request.Context(), taxID, services.CreateTaxInput{
		TaxName:             payload.TaxName,
		TaxComputationUnit:  payload.TaxComputationUnit,
		TaxComputationValue: payload.TaxComputationValue,
	})
	if err != nil {
		handler.writeTaxError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"message": "Tax updated successfully.",
		"tax":     buildTaxResponse(updatedTax),
	})
}

func (handler *TaxHandler) HandleDeleteTax(writer http.ResponseWriter, request *http.Request) {
	taxID := request.PathValue("taxID")

	if err := handler.taxService.DeleteTax(request.Context(), taxID); err != nil {
		handler.writeTaxError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]string{
		"message": "Tax deleted successfully.",
	})
}

func (handler *TaxHandler) writeTaxError(writer http.ResponseWriter, err error) {
	var validationError services.ValidationError
	if errors.As(err, &validationError) {
		http.Error(writer, validationError.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, services.ErrTaxNotFound) {
		http.Error(writer, "Tax not found.", http.StatusNotFound)
		return
	}
	if errors.Is(err, services.ErrTaxAlreadyExists) {
		http.Error(writer, "Tax name already exists.", http.StatusConflict)
		return
	}

	log.Printf("tax handler error: %v", err)
	http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
}

func buildTaxResponse(tax models.Tax) map[string]interface{} {
	return map[string]interface{}{
		"tax_id":                tax.TaxID,
		"tax_name":              tax.TaxName,
		"tax_computation_unit":  tax.TaxComputationUnit,
		"tax_computation_value": tax.TaxComputationValue,
		"created_at":            tax.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":            tax.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
