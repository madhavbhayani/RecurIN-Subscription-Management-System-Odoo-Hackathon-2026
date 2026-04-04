package models

import "time"

const (
	PaymentTermDueUnitFixedPrice = "Fixed Price"
	PaymentTermDueUnitPercentage = "Percentage"
)

// PaymentTerm represents a due rule configured by admin.
type PaymentTerm struct {
	PaymentTermID   string    `json:"payment_term_id"`
	PaymentTermName string    `json:"payment_term_name"`
	DueUnit         string    `json:"due_unit"`
	DueValue        float64   `json:"due_value"`
	IntervalDays    int       `json:"interval_days"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
