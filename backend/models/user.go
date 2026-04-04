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
