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

type subscriptionRequest struct {
	CustomerID      string                           `json:"customer_id"`
	NextInvoiceDate string                           `json:"next_invoice_date"`
	RecurringPlanID string                           `json:"recurring_plan_id"`
	PaymentTermID   string                           `json:"payment_term_id"`
	QuotationID     string                           `json:"quotation_id"`
	Status          string                           `json:"status"`
	Products        []subscriptionProductLineRequest `json:"products"`
	OtherInfo       subscriptionOtherInfoRequest     `json:"other_info"`
}

type subscriptionProductLineRequest struct {
	ProductID               string   `json:"product_id"`
	Quantity                int      `json:"quantity"`
	SelectedVariantValueIDs []string `json:"selected_variant_value_ids"`
}

type subscriptionOtherInfoRequest struct {
	SalesPerson   string `json:"sales_person"`
	StartDate     string `json:"start_date"`
	PaymentMethod string `json:"payment_method"`
	IsPaymentMode *bool  `json:"is_payment_mode"`
}

// SubscriptionHandler handles subscription administration endpoints.
type SubscriptionHandler struct {
	subscriptionService *services.SubscriptionService
}

func NewSubscriptionHandler(subscriptionService *services.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{subscriptionService: subscriptionService}
}

func parseSubscriptionDate(value string, fieldName string) (time.Time, error) {
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

func parseOptionalSubscriptionDate(value string, fieldName string) (*time.Time, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return nil, nil
	}

	parsedDate, err := time.Parse("2006-01-02", trimmedValue)
	if err != nil {
		return nil, services.ValidationError{Message: fieldName + " must be in YYYY-MM-DD format."}
	}

	return &parsedDate, nil
}

func mapSubscriptionInput(payload subscriptionRequest) (services.CreateSubscriptionInput, error) {
	nextInvoiceDate, err := parseSubscriptionDate(payload.NextInvoiceDate, "Next invoice date")
	if err != nil {
		return services.CreateSubscriptionInput{}, err
	}

	startDate, err := parseOptionalSubscriptionDate(payload.OtherInfo.StartDate, "Start date")
	if err != nil {
		return services.CreateSubscriptionInput{}, err
	}

	productLines := make([]services.CreateSubscriptionProductInput, 0, len(payload.Products))
	for _, product := range payload.Products {
		productLines = append(productLines, services.CreateSubscriptionProductInput{
			ProductID:               product.ProductID,
			Quantity:                product.Quantity,
			SelectedVariantValueIDs: product.SelectedVariantValueIDs,
		})
	}

	return services.CreateSubscriptionInput{
		CustomerID:      payload.CustomerID,
		NextInvoiceDate: nextInvoiceDate,
		RecurringPlanID: payload.RecurringPlanID,
		PaymentTermID:   payload.PaymentTermID,
		QuotationID:     payload.QuotationID,
		Status:          payload.Status,
		Products:        productLines,
		OtherInfo: services.CreateSubscriptionOtherInfoInput{
			SalesPerson:   payload.OtherInfo.SalesPerson,
			StartDate:     startDate,
			PaymentMethod: payload.OtherInfo.PaymentMethod,
			IsPaymentMode: payload.OtherInfo.IsPaymentMode,
		},
	}, nil
}

func (handler *SubscriptionHandler) HandleCreateSubscription(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	var payload subscriptionRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	input, err := mapSubscriptionInput(payload)
	if err != nil {
		handler.writeSubscriptionError(writer, err)
		return
	}

	createdSubscription, err := handler.subscriptionService.CreateSubscription(request.Context(), input)
	if err != nil {
		handler.writeSubscriptionError(writer, err)
		return
	}

	writeJSON(writer, http.StatusCreated, map[string]interface{}{
		"message":      "Subscription created successfully.",
		"subscription": buildSubscriptionResponse(createdSubscription),
	})
}

func (handler *SubscriptionHandler) HandleListSubscriptions(writer http.ResponseWriter, request *http.Request) {
	search := request.URL.Query().Get("search")

	subscriptions, err := handler.subscriptionService.ListSubscriptions(request.Context(), search)
	if err != nil {
		log.Printf("subscription list handler error: %v", err)
		http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
		return
	}

	items := make([]map[string]interface{}, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		items = append(items, buildSubscriptionResponse(subscription))
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"subscriptions": items,
	})
}

func (handler *SubscriptionHandler) HandleGetSubscriptionByID(writer http.ResponseWriter, request *http.Request) {
	subscriptionID := request.PathValue("subscriptionID")

	subscription, err := handler.subscriptionService.GetSubscriptionByID(request.Context(), subscriptionID)
	if err != nil {
		handler.writeSubscriptionError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"subscription": buildSubscriptionResponse(subscription),
	})
}

func (handler *SubscriptionHandler) HandleUpdateSubscription(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	subscriptionID := request.PathValue("subscriptionID")

	var payload subscriptionRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	input, err := mapSubscriptionInput(payload)
	if err != nil {
		handler.writeSubscriptionError(writer, err)
		return
	}

	updatedSubscription, err := handler.subscriptionService.UpdateSubscription(request.Context(), subscriptionID, input)
	if err != nil {
		handler.writeSubscriptionError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"message":      "Subscription updated successfully.",
		"subscription": buildSubscriptionResponse(updatedSubscription),
	})
}

func (handler *SubscriptionHandler) HandleDeleteSubscription(writer http.ResponseWriter, request *http.Request) {
	subscriptionID := request.PathValue("subscriptionID")

	if err := handler.subscriptionService.DeleteSubscription(request.Context(), subscriptionID); err != nil {
		handler.writeSubscriptionError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]string{
		"message": "Subscription deleted successfully.",
	})
}

func (handler *SubscriptionHandler) writeSubscriptionError(writer http.ResponseWriter, err error) {
	var validationError services.ValidationError
	if errors.As(err, &validationError) {
		http.Error(writer, validationError.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, services.ErrSubscriptionNotFound) {
		http.Error(writer, "Subscription not found.", http.StatusNotFound)
		return
	}
	if errors.Is(err, services.ErrSubscriptionAlreadyExists) {
		http.Error(writer, "Subscription number already exists.", http.StatusConflict)
		return
	}

	log.Printf("subscription handler error: %v", err)
	http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
}

func buildSubscriptionResponse(subscription models.Subscription) map[string]interface{} {
	products := make([]map[string]interface{}, 0, len(subscription.Products))
	for _, product := range subscription.Products {
		selectedVariants := make([]map[string]interface{}, 0, len(product.SelectedVariants))
		for _, selectedVariant := range product.SelectedVariants {
			selectedVariants = append(selectedVariants, map[string]interface{}{
				"subscription_product_variant_id": selectedVariant.SubscriptionProductVariantID,
				"subscription_product_id":         selectedVariant.SubscriptionProductID,
				"product_id":                      selectedVariant.ProductID,
				"attribute_id":                    selectedVariant.AttributeID,
				"attribute_name":                  selectedVariant.AttributeName,
				"attribute_value_id":              selectedVariant.AttributeValueID,
				"attribute_value":                 selectedVariant.AttributeValue,
				"extra_price":                     selectedVariant.ExtraPrice,
			})
		}

		products = append(products, map[string]interface{}{
			"subscription_product_id": product.SubscriptionProductID,
			"product_id":              product.ProductID,
			"product_name":            product.ProductName,
			"quantity":                product.Quantity,
			"unit_price":              product.UnitPrice,
			"variant_extra_amount":    product.VariantExtraAmount,
			"selected_variants":       selectedVariants,
			"discount_amount":         product.DiscountAmount,
			"tax_amount":              product.TaxAmount,
			"total_amount":            product.TotalAmount,
		})
	}

	var otherInfo map[string]interface{}
	if subscription.OtherInfo != nil {
		startDate := interface{}(nil)
		if subscription.OtherInfo.StartDate != nil {
			startDate = subscription.OtherInfo.StartDate.Format("2006-01-02")
		}

		otherInfo = map[string]interface{}{
			"subscription_other_info_id": subscription.OtherInfo.SubscriptionOtherInfoID,
			"subscription_id":            subscription.OtherInfo.SubscriptionID,
			"sales_person":               subscription.OtherInfo.SalesPerson,
			"start_date":                 startDate,
			"payment_method":             subscription.OtherInfo.PaymentMethod,
			"is_payment_mode":            subscription.OtherInfo.IsPaymentMode,
			"created_at":                 subscription.OtherInfo.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			"updated_at":                 subscription.OtherInfo.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return map[string]interface{}{
		"subscription_id":     subscription.SubscriptionID,
		"subscription_number": subscription.SubscriptionNumber,
		"customer_id":         subscription.CustomerID,
		"customer_name":       subscription.CustomerName,
		"next_invoice_date":   subscription.NextInvoiceDate.Format("2006-01-02"),
		"recurring":           subscription.Recurring,
		"plan":                subscription.Plan,
		"recurring_plan_id":   subscription.RecurringPlanID,
		"payment_term_id":     subscription.PaymentTermID,
		"payment_term_name":   subscription.PaymentTermName,
		"quotation_id":        subscription.QuotationID,
		"products":            products,
		"other_info":          otherInfo,
		"status":              string(subscription.Status),
		"created_at":          subscription.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":          subscription.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
