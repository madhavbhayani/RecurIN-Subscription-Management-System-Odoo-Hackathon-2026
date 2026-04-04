package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services/common/auth"
)

type addCartItemRequest struct {
	ProductID                  string `json:"product_id"`
	Quantity                   int    `json:"quantity"`
	SelectedVariantAttributeID string `json:"selected_variant_attribute_id"`
}

type updateCartItemRequest struct {
	Quantity int `json:"quantity"`
}

// CartHandler handles authenticated cart endpoints.
type CartHandler struct {
	cartService *services.CartService
}

func NewCartHandler(cartService *services.CartService) *CartHandler {
	return &CartHandler{cartService: cartService}
}

func getAuthenticatedUserID(request *http.Request) (string, bool) {
	claims, ok := auth.ClaimsFromContext(request.Context())
	if !ok {
		return "", false
	}

	return claims.UserID, true
}

func (handler *CartHandler) HandleListCartItems(writer http.ResponseWriter, request *http.Request) {
	userID, ok := getAuthenticatedUserID(request)
	if !ok {
		http.Error(writer, "Unauthorized", http.StatusUnauthorized)
		return
	}

	items, err := handler.cartService.ListCartItems(request.Context(), userID)
	if err != nil {
		handler.writeCartError(writer, err)
		return
	}

	responseItems := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, buildCartItemResponse(item))
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"cart_items": responseItems,
	})
}

func (handler *CartHandler) HandleAddCartItem(writer http.ResponseWriter, request *http.Request) {
	userID, ok := getAuthenticatedUserID(request)
	if !ok {
		http.Error(writer, "Unauthorized", http.StatusUnauthorized)
		return
	}

	defer request.Body.Close()

	var payload addCartItemRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	item, err := handler.cartService.AddCartItem(request.Context(), services.AddCartItemInput{
		UserID:                     userID,
		ProductID:                  payload.ProductID,
		Quantity:                   payload.Quantity,
		SelectedVariantAttributeID: payload.SelectedVariantAttributeID,
	})
	if err != nil {
		handler.writeCartError(writer, err)
		return
	}

	writeJSON(writer, http.StatusCreated, map[string]interface{}{
		"message":   "Item added to cart successfully.",
		"cart_item": buildCartItemResponse(item),
	})
}

func (handler *CartHandler) HandleUpdateCartItem(writer http.ResponseWriter, request *http.Request) {
	userID, ok := getAuthenticatedUserID(request)
	if !ok {
		http.Error(writer, "Unauthorized", http.StatusUnauthorized)
		return
	}

	cartItemID := request.PathValue("cartItemID")
	defer request.Body.Close()

	var payload updateCartItemRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	item, err := handler.cartService.UpdateCartItemQuantity(request.Context(), userID, cartItemID, payload.Quantity)
	if err != nil {
		handler.writeCartError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"message":   "Cart item updated successfully.",
		"cart_item": buildCartItemResponse(item),
	})
}

func (handler *CartHandler) HandleDeleteCartItem(writer http.ResponseWriter, request *http.Request) {
	userID, ok := getAuthenticatedUserID(request)
	if !ok {
		http.Error(writer, "Unauthorized", http.StatusUnauthorized)
		return
	}

	cartItemID := request.PathValue("cartItemID")
	if err := handler.cartService.DeleteCartItem(request.Context(), userID, cartItemID); err != nil {
		handler.writeCartError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"message": "Cart item removed successfully.",
	})
}

func (handler *CartHandler) writeCartError(writer http.ResponseWriter, err error) {
	var validationError services.ValidationError
	if errors.As(err, &validationError) {
		http.Error(writer, validationError.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, services.ErrCartItemNotFound) {
		http.Error(writer, "Cart item not found.", http.StatusNotFound)
		return
	}

	log.Printf("cart handler error: %v", err)
	http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
}

func buildCartItemResponse(item models.CartItem) map[string]interface{} {
	response := map[string]interface{}{
		"cart_item_id":           item.CartItemID,
		"user_id":                item.UserID,
		"product_id":             item.ProductID,
		"product_name":           item.ProductName,
		"product_type":           item.ProductType,
		"recurring_name":         item.RecurringName,
		"billing_period":         item.BillingPeriod,
		"quantity":               item.Quantity,
		"unit_price":             item.UnitPrice,
		"selected_variant_price": item.SelectedVariantPrice,
		"discount_amount":        item.DiscountAmount,
		"effective_unit_price":   item.EffectiveUnitPrice,
		"line_total":             item.LineTotal,
		"created_at":             item.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":             item.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}

	if item.SelectedVariantAttributeID == nil {
		response["selected_variant_attribute_id"] = nil
	} else {
		response["selected_variant_attribute_id"] = *item.SelectedVariantAttributeID
	}

	if item.SelectedVariantAttributeName == nil {
		response["selected_variant_attribute_name"] = nil
	} else {
		response["selected_variant_attribute_name"] = *item.SelectedVariantAttributeName
	}

	return response
}
