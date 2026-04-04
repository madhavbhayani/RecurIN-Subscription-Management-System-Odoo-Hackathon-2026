package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services"
)

type paymentTermRequest struct {
	PaymentTermName string  `json:"payment_term_name"`
	DueUnit         string  `json:"due_unit"`
	DueValue        float64 `json:"due_value"`
	IntervalDays    int     `json:"interval_days"`
}

// PaymentTermHandler handles payment-term administration endpoints.
type PaymentTermHandler struct {
	paymentTermService *services.PaymentTermService
}

func NewPaymentTermHandler(paymentTermService *services.PaymentTermService) *PaymentTermHandler {
	return &PaymentTermHandler{paymentTermService: paymentTermService}
}

func mapPaymentTermInput(payload paymentTermRequest) services.CreatePaymentTermInput {
	return services.CreatePaymentTermInput{
		PaymentTermName: payload.PaymentTermName,
		DueUnit:         payload.DueUnit,
		DueValue:        payload.DueValue,
		IntervalDays:    payload.IntervalDays,
	}
}

func (handler *PaymentTermHandler) HandleCreatePaymentTerm(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	var payload paymentTermRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	createdPaymentTerm, err := handler.paymentTermService.CreatePaymentTerm(request.Context(), mapPaymentTermInput(payload))
	if err != nil {
		handler.writePaymentTermError(writer, err)
		return
	}

	writeJSON(writer, http.StatusCreated, map[string]interface{}{
		"message":      "Payment term created successfully.",
		"payment_term": buildPaymentTermResponse(createdPaymentTerm),
	})
}

func (handler *PaymentTermHandler) HandleListPaymentTerms(writer http.ResponseWriter, request *http.Request) {
	search := request.URL.Query().Get("search")

	paymentTerms, err := handler.paymentTermService.ListPaymentTerms(request.Context(), search)
	if err != nil {
		log.Printf("payment term list handler error: %v", err)
		http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
		return
	}

	items := make([]map[string]interface{}, 0, len(paymentTerms))
	for _, paymentTerm := range paymentTerms {
		items = append(items, buildPaymentTermResponse(paymentTerm))
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"payment_terms": items,
	})
}

func (handler *PaymentTermHandler) HandleGetPaymentTermByID(writer http.ResponseWriter, request *http.Request) {
	paymentTermID := request.PathValue("paymentTermID")

	paymentTerm, err := handler.paymentTermService.GetPaymentTermByID(request.Context(), paymentTermID)
	if err != nil {
		handler.writePaymentTermError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"payment_term": buildPaymentTermResponse(paymentTerm),
	})
}

func (handler *PaymentTermHandler) HandleUpdatePaymentTerm(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	paymentTermID := request.PathValue("paymentTermID")

	var payload paymentTermRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	updatedPaymentTerm, err := handler.paymentTermService.UpdatePaymentTerm(request.Context(), paymentTermID, mapPaymentTermInput(payload))
	if err != nil {
		handler.writePaymentTermError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"message":      "Payment term updated successfully.",
		"payment_term": buildPaymentTermResponse(updatedPaymentTerm),
	})
}

func (handler *PaymentTermHandler) HandleDeletePaymentTerm(writer http.ResponseWriter, request *http.Request) {
	paymentTermID := request.PathValue("paymentTermID")

	if err := handler.paymentTermService.DeletePaymentTerm(request.Context(), paymentTermID); err != nil {
		handler.writePaymentTermError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]string{
		"message": "Payment term deleted successfully.",
	})
}

func (handler *PaymentTermHandler) writePaymentTermError(writer http.ResponseWriter, err error) {
	var validationError services.ValidationError
	if errors.As(err, &validationError) {
		http.Error(writer, validationError.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, services.ErrPaymentTermNotFound) {
		http.Error(writer, "Payment term not found.", http.StatusNotFound)
		return
	}
	if errors.Is(err, services.ErrPaymentTermAlreadyExists) {
		http.Error(writer, "Payment term name already exists.", http.StatusConflict)
		return
	}

	log.Printf("payment term handler error: %v", err)
	http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
}

func buildPaymentTermResponse(paymentTerm models.PaymentTerm) map[string]interface{} {
	return map[string]interface{}{
		"payment_term_id":   paymentTerm.PaymentTermID,
		"payment_term_name": paymentTerm.PaymentTermName,
		"due_unit":          paymentTerm.DueUnit,
		"due_value":         paymentTerm.DueValue,
		"interval_days":     paymentTerm.IntervalDays,
		"created_at":        paymentTerm.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":        paymentTerm.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
