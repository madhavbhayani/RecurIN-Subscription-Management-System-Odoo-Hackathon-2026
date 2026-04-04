package models

import "time"

// Quotation stores quotation template configuration.
type Quotation struct {
	QuotationID           string             `json:"quotation_id"`
	LastForever           bool               `json:"last_forever"`
	QuotationValidityDays *int               `json:"quotation_validity_days,omitempty"`
	RecurringPlanID       string             `json:"recurring_plan_id"`
	RecurringPlanName     string             `json:"recurring_plan_name"`
	ProductCount          int                `json:"product_count"`
	Products              []QuotationProduct `json:"products,omitempty"`
	CreatedAt             time.Time          `json:"created_at"`
	UpdatedAt             time.Time          `json:"updated_at"`
}

// QuotationProduct represents a product mapped to a quotation template.
type QuotationProduct struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	ProductType string  `json:"product_type"`
	SalesPrice  float64 `json:"sales_price"`
}
