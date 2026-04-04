package rbac

import (
	"net/http"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services/common/auth"
)

// RequireRoles allows route access only for specific roles.
func RequireRoles(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			claims, ok := auth.ClaimsFromContext(request.Context())
			if !ok {
				http.Error(writer, "missing authentication claims", http.StatusUnauthorized)
				return
			}

			if !IsRoleAllowed(claims.Role, allowedRoles) {
				http.Error(writer, "forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(writer, request)
		})
	}
}
