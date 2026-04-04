package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
)

const (
	payPalDefaultAPIBaseURL = "https://api.sandbox.paypal.com"
	payPalDefaultCurrency   = "USD"
	payPalSandboxHost       = "www.sandbox.paypal.com"
	payPalSandboxBusiness   = "receiver@example.com"
)

// PayPalCreateOrderResult represents create-order response metadata needed by frontend.
type PayPalCreateOrderResult struct {
	OrderID     string
	ApprovalURL string
	Amount      float64
	Currency    string
}

// PayPalCaptureOrderResult represents capture response metadata shown on success page.
type PayPalCaptureOrderResult struct {
	OrderID    string
	CaptureID  string
	Status     string
	Amount     float64
	Currency   string
	PayerEmail string
}

// PayPalService handles create/capture operations against PayPal Orders API.
type PayPalService struct {
	client          *http.Client
	clientID        string
	secret          string
	apiBaseURL      string
	currencyCode    string
	frontendBaseURL string
	sandboxBusiness string
	cartService     *CartService
}

func NewPayPalService(clientID, secret, frontendBaseURL string, cartService *CartService) *PayPalService {

	normalizedFrontendBaseURL := strings.TrimRight(strings.TrimSpace(frontendBaseURL), "/")
	if normalizedFrontendBaseURL == "" {
		normalizedFrontendBaseURL = "http://localhost:5173"
	}

	sandboxBusiness := strings.TrimSpace(os.Getenv("PAYPAL_SANDBOX_BUSINESS"))
	if sandboxBusiness == "" {
		sandboxBusiness = payPalSandboxBusiness
	}

	return &PayPalService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		clientID:        strings.TrimSpace(clientID),
		secret:          strings.TrimSpace(secret),
		apiBaseURL:      payPalDefaultAPIBaseURL,
		currencyCode:    payPalDefaultCurrency,
		frontendBaseURL: normalizedFrontendBaseURL,
		sandboxBusiness: sandboxBusiness,
		cartService:     cartService,
	}
}

func (service *PayPalService) ensureConfigured() error {
	if service.cartService == nil {
		return fmt.Errorf("paypal service is not initialized correctly")
	}
	if strings.TrimSpace(service.clientID) == "" || strings.TrimSpace(service.secret) == "" {
		return ValidationError{Message: "PayPal credentials are missing in backend configuration."}
	}

	return nil
}

func (service *PayPalService) CreateOrder(ctx context.Context, userID string) (PayPalCreateOrderResult, error) {
	normalizedUserID := strings.TrimSpace(userID)
	if normalizedUserID == "" {
		return PayPalCreateOrderResult{}, ValidationError{Message: "User ID is required."}
	}
	if err := service.ensureConfigured(); err != nil {
		return PayPalCreateOrderResult{}, err
	}

	cartItems, err := service.cartService.ListCartItems(ctx, normalizedUserID)
	if err != nil {
		return PayPalCreateOrderResult{}, err
	}
	if len(cartItems) == 0 {
		return PayPalCreateOrderResult{}, ValidationError{Message: "Your cart is empty."}
	}

	totalAmount := 0.0
	for _, item := range cartItems {
		if item.LineTotal > 0 {
			totalAmount += item.LineTotal
		}
	}
	totalAmount = roundCurrency(totalAmount)
	if totalAmount <= 0 {
		return PayPalCreateOrderResult{}, ValidationError{Message: "Cart total must be greater than zero."}
	}

	accessToken, err := service.requestAccessToken(ctx)
	if err != nil {
		return PayPalCreateOrderResult{}, err
	}

	itemName := buildPayPalItemSummary(cartItems)
	paymentID, err := service.createLegacyPayment(ctx, accessToken, totalAmount, itemName)
	if err != nil {
		return PayPalCreateOrderResult{}, err
	}

	approvalURL := service.buildSandboxXClickURL(paymentID, totalAmount, itemName)

	return PayPalCreateOrderResult{
		OrderID:     paymentID,
		ApprovalURL: approvalURL,
		Amount:      totalAmount,
		Currency:    service.currencyCode,
	}, nil
}

func (service *PayPalService) CaptureOrder(ctx context.Context, userID, orderID, paymentID, payerID string) (PayPalCaptureOrderResult, error) {
	normalizedUserID := strings.TrimSpace(userID)
	normalizedOrderID := strings.TrimSpace(orderID)
	normalizedPaymentID := strings.TrimSpace(paymentID)
	normalizedPayerID := strings.TrimSpace(payerID)
	if normalizedUserID == "" {
		return PayPalCaptureOrderResult{}, ValidationError{Message: "User ID is required."}
	}

	if normalizedPaymentID != "" {
		if normalizedPayerID == "" {
			return PayPalCaptureOrderResult{
				OrderID:  normalizedPaymentID,
				Status:   "PENDING",
				Currency: service.currencyCode,
			}, nil
		}

		return service.executeLegacyPayment(ctx, normalizedUserID, normalizedPaymentID, normalizedPayerID)
	}

	if normalizedOrderID == "" {
		return PayPalCaptureOrderResult{}, ValidationError{Message: "Order ID is required."}
	}
	if err := service.ensureConfigured(); err != nil {
		return PayPalCaptureOrderResult{}, err
	}

	accessToken, err := service.requestAccessToken(ctx)
	if err != nil {
		return PayPalCaptureOrderResult{}, err
	}

	statusCode, responseBody, err := service.doPayPalRequest(
		ctx,
		http.MethodPost,
		"/v2/checkout/orders/"+normalizedOrderID+"/capture",
		accessToken,
		map[string]interface{}{},
		nil,
	)
	if err != nil {
		return PayPalCaptureOrderResult{}, err
	}
	if statusCode < 200 || statusCode >= 300 {
		return PayPalCaptureOrderResult{}, parsePayPalAPIError(statusCode, responseBody)
	}

	var responsePayload struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Payer  struct {
			EmailAddress string `json:"email_address"`
		} `json:"payer"`
		PurchaseUnits []struct {
			CustomID string `json:"custom_id"`
			Payments struct {
				Captures []struct {
					ID     string `json:"id"`
					Status string `json:"status"`
					Amount struct {
						CurrencyCode string `json:"currency_code"`
						Value        string `json:"value"`
					} `json:"amount"`
				} `json:"captures"`
			} `json:"payments"`
		} `json:"purchase_units"`
	}
	if err := json.Unmarshal(responseBody, &responsePayload); err != nil {
		return PayPalCaptureOrderResult{}, fmt.Errorf("failed to decode paypal capture response: %w", err)
	}

	if strings.TrimSpace(responsePayload.ID) == "" {
		return PayPalCaptureOrderResult{}, fmt.Errorf("paypal capture response is missing order id")
	}

	if len(responsePayload.PurchaseUnits) == 0 {
		return PayPalCaptureOrderResult{}, fmt.Errorf("paypal capture response is missing purchase units")
	}

	customID := strings.TrimSpace(responsePayload.PurchaseUnits[0].CustomID)
	if customID != "" && customID != normalizedUserID {
		return PayPalCaptureOrderResult{}, ValidationError{Message: "Payment order does not belong to the current user."}
	}

	captures := responsePayload.PurchaseUnits[0].Payments.Captures
	if len(captures) == 0 {
		return PayPalCaptureOrderResult{}, fmt.Errorf("paypal capture response is missing capture details")
	}

	capture := captures[0]
	amountValue := 0.0
	if parsedAmount, parseErr := strconv.ParseFloat(strings.TrimSpace(capture.Amount.Value), 64); parseErr == nil {
		amountValue = roundCurrency(parsedAmount)
	}

	captureStatus := strings.TrimSpace(capture.Status)
	if captureStatus == "" {
		captureStatus = strings.TrimSpace(responsePayload.Status)
	}

	if strings.EqualFold(captureStatus, "COMPLETED") {
		// Persist the captured payment and create subscriptions
		rawPayload, marshalErr := json.Marshal(responsePayload)
		if marshalErr != nil {
			return PayPalCaptureOrderResult{}, fmt.Errorf("failed to encode paypal payment result: %w", marshalErr)
		}

		if persistErr := service.persistCapturedPayment(
			ctx,
			normalizedUserID,
			strings.TrimSpace(responsePayload.ID),
			"", // payerID is not available in modern orders API
			strings.TrimSpace(capture.ID),
			captureStatus,
			amountValue,
			strings.TrimSpace(capture.Amount.CurrencyCode),
			rawPayload,
		); persistErr != nil {
			return PayPalCaptureOrderResult{}, persistErr
		}

		if err := service.cartService.ClearCart(ctx, normalizedUserID); err != nil {
			return PayPalCaptureOrderResult{}, err
		}
	}

	return PayPalCaptureOrderResult{
		OrderID:    responsePayload.ID,
		CaptureID:  strings.TrimSpace(capture.ID),
		Status:     captureStatus,
		Amount:     amountValue,
		Currency:   strings.TrimSpace(capture.Amount.CurrencyCode),
		PayerEmail: strings.TrimSpace(responsePayload.Payer.EmailAddress),
	}, nil
}

func (service *PayPalService) createLegacyPayment(ctx context.Context, accessToken string, amount float64, itemName string) (string, error) {
	returnURL := service.frontendBaseURL + "/success"
	cancelURL := service.frontendBaseURL + "/check-out"

	requestPayload := map[string]interface{}{
		"intent": "sale",
		"payer": map[string]interface{}{
			"payment_method": "paypal",
		},
		"transactions": []map[string]interface{}{
			{
				"amount": map[string]interface{}{
					"total":    fmt.Sprintf("%.2f", roundCurrency(amount)),
					"currency": service.currencyCode,
					"details": map[string]interface{}{
						"subtotal": fmt.Sprintf("%.2f", roundCurrency(amount)),
					},
				},
				"description": fmt.Sprintf("Checkout for %s", itemName),
			},
		},
		"redirect_urls": map[string]interface{}{
			"return_url": returnURL,
			"cancel_url": cancelURL,
		},
	}

	statusCode, responseBody, err := service.doPayPalRequest(
		ctx,
		http.MethodPost,
		"/v1/payments/payment",
		accessToken,
		requestPayload,
		nil,
	)
	if err != nil {
		return "", err
	}
	if statusCode < 200 || statusCode >= 300 {
		return "", parsePayPalAPIError(statusCode, responseBody)
	}

	var responsePayload struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(responseBody, &responsePayload); err != nil {
		return "", fmt.Errorf("failed to decode paypal legacy create response: %w", err)
	}

	paymentID := strings.TrimSpace(responsePayload.ID)
	if paymentID == "" {
		return "", fmt.Errorf("paypal legacy create response is missing payment id")
	}

	return paymentID, nil
}

func (service *PayPalService) executeLegacyPayment(ctx context.Context, userID, paymentID, payerID string) (PayPalCaptureOrderResult, error) {
	if err := service.ensureConfigured(); err != nil {
		return PayPalCaptureOrderResult{}, err
	}

	accessToken, err := service.requestAccessToken(ctx)
	if err != nil {
		return PayPalCaptureOrderResult{}, err
	}

	requestPayload := map[string]interface{}{
		"payer_id": payerID,
	}

	statusCode, responseBody, err := service.doPayPalRequest(
		ctx,
		http.MethodPost,
		"/v1/payments/payment/"+paymentID+"/execute",
		accessToken,
		requestPayload,
		nil,
	)
	if err != nil {
		return PayPalCaptureOrderResult{}, err
	}
	if statusCode < 200 || statusCode >= 300 {
		return PayPalCaptureOrderResult{}, parsePayPalAPIError(statusCode, responseBody)
	}

	var responsePayload struct {
		ID    string `json:"id"`
		State string `json:"state"`
		Payer struct {
			PayerInfo struct {
				Email string `json:"email"`
			} `json:"payer_info"`
		} `json:"payer"`
		Transactions []struct {
			Amount struct {
				Total    string `json:"total"`
				Currency string `json:"currency"`
			} `json:"amount"`
			RelatedResources []struct {
				Sale struct {
					ID    string `json:"id"`
					State string `json:"state"`
				} `json:"sale"`
			} `json:"related_resources"`
		} `json:"transactions"`
	}
	if err := json.Unmarshal(responseBody, &responsePayload); err != nil {
		return PayPalCaptureOrderResult{}, fmt.Errorf("failed to decode paypal legacy execute response: %w", err)
	}

	amountValue := 0.0
	currency := service.currencyCode
	captureID := ""
	status := strings.TrimSpace(responsePayload.State)

	if len(responsePayload.Transactions) > 0 {
		transaction := responsePayload.Transactions[0]
		if parsedAmount, parseErr := strconv.ParseFloat(strings.TrimSpace(transaction.Amount.Total), 64); parseErr == nil {
			amountValue = roundCurrency(parsedAmount)
		}
		if strings.TrimSpace(transaction.Amount.Currency) != "" {
			currency = strings.TrimSpace(transaction.Amount.Currency)
		}

		if len(transaction.RelatedResources) > 0 {
			sale := transaction.RelatedResources[0].Sale
			if strings.TrimSpace(sale.ID) != "" {
				captureID = strings.TrimSpace(sale.ID)
			}
			if strings.TrimSpace(sale.State) != "" {
				status = strings.TrimSpace(sale.State)
			}
		}
	}

	if strings.EqualFold(status, "approved") || strings.EqualFold(status, "completed") {
		rawPayload, marshalErr := json.Marshal(responsePayload)
		if marshalErr != nil {
			return PayPalCaptureOrderResult{}, fmt.Errorf("failed to encode paypal payment result: %w", marshalErr)
		}

		if persistErr := service.persistCapturedPayment(ctx, userID, paymentID, payerID, captureID, status, amountValue, currency, rawPayload); persistErr != nil {
			return PayPalCaptureOrderResult{}, persistErr
		}
	}

	return PayPalCaptureOrderResult{
		OrderID:    strings.TrimSpace(responsePayload.ID),
		CaptureID:  captureID,
		Status:     status,
		Amount:     amountValue,
		Currency:   currency,
		PayerEmail: strings.TrimSpace(responsePayload.Payer.PayerInfo.Email),
	}, nil
}

func (service *PayPalService) buildSandboxXClickURL(paymentID string, amount float64, itemName string) string {
	returnURL := fmt.Sprintf("%s/success?paymentId=%s&orderId=%s", service.frontendBaseURL, paymentID, paymentID)
	cancelReturnURL := service.frontendBaseURL + "/check-out"

	query := url.Values{}
	query.Set("cmd", "_xclick")
	query.Set("business", service.sandboxBusiness)
	query.Set("item_name", itemName)
	query.Set("amount", fmt.Sprintf("%.2f", roundCurrency(amount)))
	query.Set("invoice", paymentID)
	query.Set("currency_code", service.currencyCode)
	query.Set("return", returnURL)
	query.Set("cancel_return", cancelReturnURL)

	return "https://" + payPalSandboxHost + "/cgi-bin/webscr?" + query.Encode()
}

func buildPayPalItemSummary(cartItems []models.CartItem) string {
	if len(cartItems) == 0 {
		return "Cart Checkout"
	}

	firstItemName := strings.TrimSpace(cartItems[0].ProductName)
	if firstItemName == "" {
		firstItemName = "Subscription Product"
	}

	if len(cartItems) == 1 {
		return firstItemName
	}

	return fmt.Sprintf("%s and %d more item(s)", firstItemName, len(cartItems)-1)
}

func (service *PayPalService) requestAccessToken(ctx context.Context) (string, error) {
	requestBody := strings.NewReader("grant_type=client_credentials")
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, service.apiBaseURL+"/v1/oauth2/token", requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to build paypal auth request: %w", err)
	}

	request.SetBasicAuth(service.clientID, service.secret)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	response, err := service.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("failed to request paypal access token: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read paypal auth response: %w", err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", parsePayPalAPIError(response.StatusCode, responseBody)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return "", fmt.Errorf("failed to decode paypal auth response: %w", err)
	}

	if strings.TrimSpace(payload.AccessToken) == "" {
		return "", fmt.Errorf("paypal auth response is missing access token")
	}

	return payload.AccessToken, nil
}

func (service *PayPalService) doPayPalRequest(ctx context.Context, method, path, accessToken string, payload interface{}, headers map[string]string) (int, []byte, error) {
	var requestBody io.Reader
	if payload != nil {
		encodedPayload, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to encode paypal request payload: %w", err)
		}
		requestBody = bytes.NewReader(encodedPayload)
	}

	request, err := http.NewRequestWithContext(ctx, method, service.apiBaseURL+path, requestBody)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to build paypal request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		request.Header.Set(trimmedKey, trimmedValue)
	}

	response, err := service.client.Do(request)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to call paypal API: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read paypal response: %w", err)
	}

	return response.StatusCode, responseBody, nil
}

func parsePayPalAPIError(statusCode int, responseBody []byte) error {
	message := strings.TrimSpace(string(responseBody))
	if message == "" {
		message = fmt.Sprintf("paypal request failed with status %d", statusCode)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(responseBody, &payload); err == nil {
		if parsedMessage := strings.TrimSpace(fmt.Sprintf("%v", payload["message"])); parsedMessage != "" && parsedMessage != "<nil>" {
			message = parsedMessage
		}

		if details, ok := payload["details"].([]interface{}); ok && len(details) > 0 {
			if detailMap, ok := details[0].(map[string]interface{}); ok {
				if detailMessage := strings.TrimSpace(fmt.Sprintf("%v", detailMap["description"])); detailMessage != "" && detailMessage != "<nil>" {
					message = detailMessage
				}
			}
		}
	}

	if statusCode >= 400 && statusCode < 500 {
		return ValidationError{Message: message}
	}

	return fmt.Errorf("paypal request failed (status %d): %s", statusCode, message)
}
