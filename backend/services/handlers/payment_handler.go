package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services"
)

type capturePayPalOrderRequest struct {
	OrderID   string `json:"order_id"`
	PaymentID string `json:"payment_id"`
	PayerID   string `json:"payer_id"`
}

// PaymentHandler handles authenticated payment flows.
type PaymentHandler struct {
	payPalService *services.PayPalService
}

func NewPaymentHandler(payPalService *services.PayPalService) *PaymentHandler {
	return &PaymentHandler{payPalService: payPalService}
}

func (handler *PaymentHandler) HandleCreatePayPalOrder(writer http.ResponseWriter, request *http.Request) {
	userID, ok := getAuthenticatedUserID(request)
	if !ok {
		http.Error(writer, "Unauthorized", http.StatusUnauthorized)
		return
	}

	order, err := handler.payPalService.CreateOrder(request.Context(), userID)
	if err != nil {
		handler.writePaymentError(writer, err)
		return
	}

	writeJSON(writer, http.StatusCreated, map[string]interface{}{
		"message":      "PayPal order created successfully.",
		"order_id":     order.OrderID,
		"payment_id":   order.OrderID,
		"approval_url": order.ApprovalURL,
		"amount":       order.Amount,
		"currency":     order.Currency,
	})
}

func (handler *PaymentHandler) HandleCapturePayPalOrder(writer http.ResponseWriter, request *http.Request) {
	userID, ok := getAuthenticatedUserID(request)
	if !ok {
		http.Error(writer, "Unauthorized", http.StatusUnauthorized)
		return
	}

	defer request.Body.Close()

	var payload capturePayPalOrderRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		log.Printf("[HANDLER] Invalid capture payload from user %s: %v", userID, err)
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	log.Printf("[HANDLER] HandleCapturePayPalOrder: userID=%s orderID=%q paymentID=%q payerID=%q", userID, payload.OrderID, payload.PaymentID, payload.PayerID)

	captureResult, err := handler.payPalService.CaptureOrder(request.Context(), userID, payload.OrderID, payload.PaymentID, payload.PayerID)
	if err != nil {
		log.Printf("[HANDLER] CaptureOrder error for user %s: %v", userID, err)
		handler.writePaymentError(writer, err)
		return
	}

	log.Printf("[HANDLER] CaptureOrder success: status=%s orderID=%s captureID=%s amount=%.2f", captureResult.Status, captureResult.OrderID, captureResult.CaptureID, captureResult.Amount)

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"message": "Payment captured successfully.",
		"payment": map[string]interface{}{
			"order_id":    captureResult.OrderID,
			"capture_id":  captureResult.CaptureID,
			"status":      captureResult.Status,
			"amount":      captureResult.Amount,
			"currency":    captureResult.Currency,
			"payer_email": captureResult.PayerEmail,
		},
	})
}

func (handler *PaymentHandler) writePaymentError(writer http.ResponseWriter, err error) {
	var validationError services.ValidationError
	if errors.As(err, &validationError) {
		http.Error(writer, validationError.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("payment handler error: %v", err)
	http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
}
