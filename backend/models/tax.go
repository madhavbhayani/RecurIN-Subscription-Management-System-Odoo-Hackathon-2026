package models

import "time"

const (
	TaxComputationUnitFixedPrice = "Fixed Price"
	TaxComputationUnitPercentage = "Percentage"
)

// Tax represents a tax configuration record.
type Tax struct {
	TaxID               string    `json:"tax_id"`
	TaxName             string    `json:"tax_name"`
	TaxComputationUnit  string    `json:"tax_computation_unit"`
	TaxComputationValue float64   `json:"tax_computation_value"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
