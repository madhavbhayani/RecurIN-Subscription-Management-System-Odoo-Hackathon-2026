package models

import "time"

const (
	DiscountUnitFixedPrice = "Fixed Price"
	DiscountUnitPercentage = "Percentage"
)

// Discount represents a discount rule configured by admin.
type Discount struct {
	DiscountID       string            `json:"discount_id"`
	DiscountName     string            `json:"discount_name"`
	DiscountUnit     string            `json:"discount_unit"`
	DiscountValue    float64           `json:"discount_value"`
	MinimumPurchase  float64           `json:"minimum_purchase"`
	MaximumPurchase  float64           `json:"maximum_purchase"`
	StartDate        time.Time         `json:"start_date"`
	EndDate          time.Time         `json:"end_date"`
	IsLimit          bool              `json:"is_limit"`
	LimitUsers       *int              `json:"limit_users,omitempty"`
	AppliedUserCount int               `json:"applied_user_count"`
	IsActive         bool              `json:"is_active"`
	ProductCount     int               `json:"product_count"`
	Products         []DiscountProduct `json:"products,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// DiscountProduct represents a product mapped to a discount.
type DiscountProduct struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	ProductType string  `json:"product_type"`
	SalesPrice  float64 `json:"sales_price"`
}
