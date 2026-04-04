package rbac

import "strings"

const (
	RoleAdmin        = "admin"
	RoleInternalUser = "internal-user"
	RolePortalUser   = "portal-user"
)

// IsRoleAllowed verifies whether the role can access a route.
func IsRoleAllowed(role string, allowedRoles []string) bool {
	normalizedRole := strings.TrimSpace(strings.ToLower(role))
	if normalizedRole == RoleAdmin {
		return true
	}

	for _, allowedRole := range allowedRoles {
		if normalizedRole == strings.TrimSpace(strings.ToLower(allowedRole)) {
			return true
		}
	}

	return false
}
