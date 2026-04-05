package models

import "time"

type SubscriptionStatus string

const (
	SubscriptionStatusDraft         SubscriptionStatus = "Draft"
	SubscriptionStatusQuotationSent SubscriptionStatus = "Quotation Sent"
	SubscriptionStatusActive        SubscriptionStatus = "Active"
	SubscriptionStatusConfirmed     SubscriptionStatus = "Confirmed"
	SubscriptionStatusCancelled     SubscriptionStatus = "Cancelled"
)

// Subscription stores the subscription module record details.
type Subscription struct {
	SubscriptionID     string                 `json:"subscription_id"`
	SubscriptionNumber string                 `json:"subscription_number"`
	CustomerID         string                 `json:"customer_id"`
	CustomerName       string                 `json:"customer_name"`
	NextInvoiceDate    time.Time              `json:"next_invoice_date"`
	Recurring          *string                `json:"recurring,omitempty"`
	Plan               *string                `json:"plan,omitempty"`
	RecurringPlanID    *string                `json:"recurring_plan_id,omitempty"`
	PaymentTermID      *string                `json:"payment_term_id,omitempty"`
	PaymentTermName    *string                `json:"payment_term_name,omitempty"`
	QuotationID        *string                `json:"quotation_id,omitempty"`
	Products           []SubscriptionProduct  `json:"products,omitempty"`
	OtherInfo          *SubscriptionOtherInfo `json:"other_info,omitempty"`
	Payment            *SubscriptionPayment   `json:"payment,omitempty"`
	Status             SubscriptionStatus     `json:"status"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// SubscriptionPayment stores latest payment details tied to a subscription.
type SubscriptionPayment struct {
	PaymentID       string                 `json:"payment_id"`
	PayPalPaymentID string                 `json:"paypal_payment_id"`
	PayPalPayerID   *string                `json:"paypal_payer_id,omitempty"`
	PayPalCaptureID *string                `json:"paypal_capture_id,omitempty"`
	PayPalStatus    string                 `json:"paypal_status"`
	AmountINR       float64                `json:"amount_inr"`
	AmountUSD       float64                `json:"amount_usd"`
	CurrencyFrom    string                 `json:"currency_from"`
	CurrencyTo      string                 `json:"currency_to"`
	PaymentMethod   string                 `json:"payment_method"`
	PaymentDate     time.Time              `json:"payment_date"`
	RawPayload      map[string]interface{} `json:"raw_payload,omitempty"`
}

// SubscriptionProduct stores product rows attached to a subscription.
type SubscriptionProduct struct {
	SubscriptionProductID string                       `json:"subscription_product_id"`
	ProductID             string                       `json:"product_id"`
	ProductName           string                       `json:"product_name"`
	Quantity              int                          `json:"quantity"`
	UnitPrice             float64                      `json:"unit_price"`
	VariantExtraAmount    float64                      `json:"variant_extra_amount"`
	DiscountAmount        float64                      `json:"discount_amount"`
	TaxAmount             float64                      `json:"tax_amount"`
	TotalAmount           float64                      `json:"total_amount"`
	SelectedVariants      []SubscriptionProductVariant `json:"selected_variants,omitempty"`
}

// SubscriptionProductVariant stores selected variant values for a subscription line item.
type SubscriptionProductVariant struct {
	SubscriptionProductVariantID string  `json:"subscription_product_variant_id"`
	SubscriptionProductID        string  `json:"subscription_product_id"`
	ProductID                    string  `json:"product_id"`
	AttributeID                  string  `json:"attribute_id"`
	AttributeName                string  `json:"attribute_name"`
	AttributeValueID             string  `json:"attribute_value_id"`
	AttributeValue               string  `json:"attribute_value"`
	ExtraPrice                   float64 `json:"extra_price"`
}

// SubscriptionOtherInfo stores supplementary details for a subscription.
type SubscriptionOtherInfo struct {
	SubscriptionOtherInfoID string     `json:"subscription_other_info_id"`
	SubscriptionID          string     `json:"subscription_id"`
	SalesPerson             *string    `json:"sales_person,omitempty"`
	StartDate               *time.Time `json:"start_date,omitempty"`
	PaymentMethod           *string    `json:"payment_method,omitempty"`
	IsPaymentMode           *bool      `json:"is_payment_mode,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}
