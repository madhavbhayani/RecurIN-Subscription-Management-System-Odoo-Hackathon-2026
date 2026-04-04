package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const freeCurrencyAPIBaseURL = "https://api.freecurrencyapi.com/v1/latest"

// CurrencyService handles currency conversion through FreeCurrencyAPI.
type CurrencyService struct {
	apiKey string
	client *http.Client
}

func NewCurrencyService(apiKey string) *CurrencyService {
	return &CurrencyService{
		apiKey: strings.TrimSpace(apiKey),
		client: &http.Client{Timeout: 12 * time.Second},
	}
}

func (service *CurrencyService) Convert(ctx context.Context, amount float64, fromCurrency, toCurrency string) (float64, error) {
	if amount < 0 {
		return 0, ValidationError{Message: "Amount must be zero or greater."}
	}

	from := strings.ToUpper(strings.TrimSpace(fromCurrency))
	to := strings.ToUpper(strings.TrimSpace(toCurrency))
	if from == "" || to == "" {
		return 0, ValidationError{Message: "Both source and target currencies are required."}
	}
	if from == to {
		return roundCurrency(amount), nil
	}
	if strings.TrimSpace(service.apiKey) == "" {
		return 0, ValidationError{Message: "Currency conversion API key is missing in backend configuration."}
	}

	query := url.Values{}
	query.Set("apikey", service.apiKey)
	query.Set("base_currency", from)
	query.Set("currencies", to)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, freeCurrencyAPIBaseURL+"?"+query.Encode(), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to build currency conversion request: %w", err)
	}

	response, err := service.client.Do(request)
	if err != nil {
		return 0, fmt.Errorf("failed to call currency conversion service: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read currency conversion response: %w", err)
	}

	var payload struct {
		Data    map[string]float64 `json:"data"`
		Message string             `json:"message"`
	}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return 0, fmt.Errorf("failed to decode currency conversion response: %w", err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message := strings.TrimSpace(payload.Message)
		if message == "" {
			message = fmt.Sprintf("currency conversion request failed with status %d", response.StatusCode)
		}
		return 0, ValidationError{Message: message}
	}

	rate, ok := payload.Data[to]
	if !ok || rate <= 0 {
		return 0, fmt.Errorf("currency conversion rate unavailable for %s to %s", from, to)
	}

	return roundCurrency(amount * rate), nil
}
