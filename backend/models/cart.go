package models

import "time"

// CartItem represents one product line in a user's cart.
type CartItem struct {
	CartItemID                   string    `json:"cart_item_id"`
	UserID                       string    `json:"user_id"`
	ProductID                    string    `json:"product_id"`
	ProductName                  string    `json:"product_name"`
	ProductType                  string    `json:"product_type"`
	RecurringName                string    `json:"recurring_name,omitempty"`
	BillingPeriod                string    `json:"billing_period,omitempty"`
	Quantity                     int       `json:"quantity"`
	UnitPrice                    float64   `json:"unit_price"`
	SelectedVariantAttributeID   *string   `json:"selected_variant_attribute_id,omitempty"`
	SelectedVariantAttributeName *string   `json:"selected_variant_attribute_name,omitempty"`
	SelectedVariantPrice         float64   `json:"selected_variant_price"`
	DiscountAmount               float64   `json:"discount_amount"`
	EffectiveUnitPrice           float64   `json:"effective_unit_price"`
	LineTotal                    float64   `json:"line_total"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`
}
