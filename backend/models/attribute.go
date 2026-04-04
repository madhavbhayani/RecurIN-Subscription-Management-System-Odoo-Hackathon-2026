package models

import "time"

// Attribute represents an attribute master record.
type Attribute struct {
	AttributeID   string           `json:"attribute_id"`
	AttributeName string           `json:"attribute_name"`
	Values        []AttributeValue `json:"values,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

// AttributeValue represents a single selectable value for an attribute.
type AttributeValue struct {
	AttributeValueID  string    `json:"attribute_value_id"`
	AttributeID       string    `json:"attribute_id"`
	AttributeValue    string    `json:"attribute_value"`
	DefaultExtraPrice float64   `json:"default_extra_price"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
