package models

import "time"

// Product represents a product configuration.
type Product struct {
	ProductID       string            `json:"product_id"`
	ProductName     string            `json:"product_name"`
	ProductType     string            `json:"product_type"`
	SalesPrice      float64           `json:"sales_price"`
	CostPrice       float64           `json:"cost_price"`
	RecurringPlanID string            `json:"recurring_plan_id,omitempty"`
	RecurringName   string            `json:"recurring_name,omitempty"`
	BillingPeriod   string            `json:"billing_period,omitempty"`
	Taxes           []ProductTax      `json:"taxes,omitempty"`
	Discounts       []ProductDiscount `json:"discounts,omitempty"`
	Variants        []ProductVariant  `json:"variants,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// ProductTax references a tax assigned to a product.
type ProductTax struct {
	TaxID               string  `json:"tax_id"`
	TaxName             string  `json:"tax_name"`
	TaxComputationUnit  string  `json:"tax_computation_unit"`
	TaxComputationValue float64 `json:"tax_computation_value"`
}

// ProductVariant references an attribute selected for a product.
type ProductVariant struct {
	AttributeID   string `json:"attribute_id"`
	AttributeName string `json:"attribute_name"`
}

// ProductDiscount references a discount assigned to a product.
type ProductDiscount struct {
	DiscountID    string  `json:"discount_id"`
	DiscountName  string  `json:"discount_name"`
	DiscountUnit  string  `json:"discount_unit"`
	DiscountValue float64 `json:"discount_value"`
}
