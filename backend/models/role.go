package models

import "time"

// RolePermission stores CRUD permissions for a resource key.
type RolePermission struct {
	ResourceKey string `json:"resource_key"`
	CanCreate   bool   `json:"can_create"`
	CanRead     bool   `json:"can_read"`
	CanUpdate   bool   `json:"can_update"`
	CanDelete   bool   `json:"can_delete"`
}

// RoleProfile represents an assignable role profile mapped to a user.
type RoleProfile struct {
	RoleID      string           `json:"role_id"`
	RoleName    string           `json:"role_name"`
	UserID      *string          `json:"user_id,omitempty"`
	UserName    *string          `json:"user_name,omitempty"`
	UserEmail   *string          `json:"user_email,omitempty"`
	IsSystem    bool             `json:"is_system"`
	Permissions []RolePermission `json:"permissions"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}
