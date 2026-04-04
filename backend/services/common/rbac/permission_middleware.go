package rbac

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services/common/auth"
)

type rolePermissionRow struct {
	CanCreate bool
	CanRead   bool
	CanUpdate bool
	CanDelete bool
}

func fetchUserRolePermission(ctx context.Context, databasePool *pgxpool.Pool, userID string, resourceKey string) (*rolePermissionRow, error) {
	const query = `
		SELECT
			rp.can_create,
			rp.can_read,
			rp.can_update,
			rp.can_delete
		FROM privileges.role_data rd
		JOIN privileges.role_permissions rp ON rp.role_id = rd.role_id
		WHERE rd.user_id = $1
		  AND rp.resource_key = $2
		LIMIT 1`

	var permission rolePermissionRow
	if err := databasePool.QueryRow(ctx, query, userID, resourceKey).Scan(
		&permission.CanCreate,
		&permission.CanRead,
		&permission.CanUpdate,
		&permission.CanDelete,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch user permission: %w", err)
	}

	return &permission, nil
}

func isActionAllowed(permission *rolePermissionRow, action PermissionAction) bool {
	if permission == nil {
		return false
	}

	switch normalizePermissionAction(action) {
	case PermissionActionCreate:
		return permission.CanCreate
	case PermissionActionRead:
		return permission.CanRead
	case PermissionActionUpdate:
		return permission.CanUpdate
	case PermissionActionDelete:
		return permission.CanDelete
	default:
		return false
	}
}

// RequirePermission authorizes internal users against per-resource CRUD permissions.
func RequirePermission(databasePool *pgxpool.Pool, resourceKey string, action PermissionAction) func(http.Handler) http.Handler {
	normalizedResourceKey := normalizePermissionResourceKey(resourceKey)
	normalizedAction := normalizePermissionAction(action)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if databasePool == nil {
				http.Error(writer, "permission database unavailable", http.StatusInternalServerError)
				return
			}

			if !IsValidPermissionResource(normalizedResourceKey) || !IsValidPermissionAction(normalizedAction) {
				http.Error(writer, "invalid permission rule", http.StatusInternalServerError)
				return
			}

			claims, ok := auth.ClaimsFromContext(request.Context())
			if !ok {
				http.Error(writer, "missing authentication claims", http.StatusUnauthorized)
				return
			}

			normalizedRole := strings.TrimSpace(strings.ToLower(claims.Role))
			if normalizedRole == RoleAdmin {
				next.ServeHTTP(writer, request)
				return
			}

			permission, err := fetchUserRolePermission(request.Context(), databasePool, claims.UserID, normalizedResourceKey)
			if err != nil {
				http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
				return
			}

			if !isActionAllowed(permission, normalizedAction) {
				http.Error(writer, "forbidden: insufficient operation permission", http.StatusForbidden)
				return
			}

			next.ServeHTTP(writer, request)
		})
	}
}
