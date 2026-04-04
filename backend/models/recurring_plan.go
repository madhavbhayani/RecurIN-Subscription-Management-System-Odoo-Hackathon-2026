package models

import "time"

// RecurringPlan stores recurring billing configuration.
type RecurringPlan struct {
	RecurringPlanID      string                 `json:"recurring_plan_id"`
	RecurringName        string                 `json:"recurring_name"`
	BillingPeriod        string                 `json:"billing_period"`
	IsClosable           bool                   `json:"is_closable"`
	AutomaticCloseCycles *int                   `json:"automatic_close_cycles,omitempty"`
	IsPausable           bool                   `json:"is_pausable"`
	IsRenewable          bool                   `json:"is_renewable"`
	IsActive             bool                   `json:"is_active"`
	ProductCount         int                    `json:"product_count"`
	Products             []RecurringPlanProduct `json:"products,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

// RecurringPlanProduct represents a product mapped to a recurring plan.
type RecurringPlanProduct struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	ProductType string  `json:"product_type"`
	SalesPrice  float64 `json:"sales_price"`
	MinQty      int     `json:"min_qty"`
}
