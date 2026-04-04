package rbac

import "strings"

type PermissionAction string

const (
	PermissionActionCreate PermissionAction = "create"
	PermissionActionRead   PermissionAction = "read"
	PermissionActionUpdate PermissionAction = "update"
	PermissionActionDelete PermissionAction = "delete"
)

const (
	ResourceSubscriptions             = "subscriptions"
	ResourceProducts                  = "products"
	ResourceReporting                 = "reporting"
	ResourceUsers                     = "users"
	ResourceRoles                     = "roles"
	ResourceConfigurationsAttribute   = "configurations.attribute"
	ResourceConfigurationsRecurring   = "configurations.recurring-plan"
	ResourceConfigurationsQuotation   = "configurations.quotation-template"
	ResourceConfigurationsPaymentTerm = "configurations.payment-term"
	ResourceConfigurationsDiscount    = "configurations.discount"
	ResourceConfigurationsTaxes       = "configurations.taxes"
)

var permissionResourceKeys = []string{
	ResourceSubscriptions,
	ResourceProducts,
	ResourceReporting,
	ResourceUsers,
	ResourceRoles,
	ResourceConfigurationsAttribute,
	ResourceConfigurationsRecurring,
	ResourceConfigurationsQuotation,
	ResourceConfigurationsPaymentTerm,
	ResourceConfigurationsDiscount,
	ResourceConfigurationsTaxes,
}

func normalizePermissionResourceKey(resourceKey string) string {
	return strings.TrimSpace(strings.ToLower(resourceKey))
}

func normalizePermissionAction(action PermissionAction) PermissionAction {
	return PermissionAction(strings.TrimSpace(strings.ToLower(string(action))))
}

func IsValidPermissionResource(resourceKey string) bool {
	normalizedResourceKey := normalizePermissionResourceKey(resourceKey)
	for _, allowedResourceKey := range permissionResourceKeys {
		if normalizedResourceKey == allowedResourceKey {
			return true
		}
	}

	return false
}

func IsValidPermissionAction(action PermissionAction) bool {
	normalizedAction := normalizePermissionAction(action)
	switch normalizedAction {
	case PermissionActionCreate, PermissionActionRead, PermissionActionUpdate, PermissionActionDelete:
		return true
	default:
		return false
	}
}

func ListPermissionResources() []string {
	resources := make([]string, 0, len(permissionResourceKeys))
	resources = append(resources, permissionResourceKeys...)
	return resources
}
