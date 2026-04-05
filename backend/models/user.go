package models

import "time"

type Role string

const (
	RoleAdmin    Role = "Admin"
	RoleInternal Role = "Internal"
	RoleUser     Role = "User"
)

// User represents platform users across admin, internal and portal access levels.
type User struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	PhoneNumber string    `json:"phone_number"`
	Address     *string   `json:"address,omitempty"`
	Role        Role      `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserSubscriptionSummary represents a user-linked subscription used in user edit screens.
type UserSubscriptionSummary struct {
	SubscriptionID     string    `json:"subscription_id"`
	SubscriptionNumber string    `json:"subscription_number"`
	NextInvoiceDate    time.Time `json:"next_invoice_date"`
	Recurring          *string   `json:"recurring,omitempty"`
	Plan               *string   `json:"plan,omitempty"`
	Status             string    `json:"status"`
}
