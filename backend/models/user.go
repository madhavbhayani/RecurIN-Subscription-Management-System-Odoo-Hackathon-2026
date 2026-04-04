package models

import "time"

type Role string

const (
	RoleAdmin        Role = "admin"
	RoleInternalUser Role = "internal-user"
	RolePortalUser   Role = "portal-user"
)

// User represents platform users across admin, internal and portal access levels.
type User struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	CountryCode string    `json:"country_code"`
	Role        Role      `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
