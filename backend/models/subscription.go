package models

import "time"

type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusPaused    SubscriptionStatus = "paused"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
)

// Subscription stores recurring billing lifecycle details.
type Subscription struct {
	ID              string             `json:"id"`
	CustomerID      string             `json:"customer_id"`
	ProductID       string             `json:"product_id"`
	PlanName        string             `json:"plan_name"`
	Amount          float64            `json:"amount"`
	Currency        string             `json:"currency"`
	BillingCycle    string             `json:"billing_cycle"`
	NextBillingDate time.Time          `json:"next_billing_date"`
	Status          SubscriptionStatus `json:"status"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}
