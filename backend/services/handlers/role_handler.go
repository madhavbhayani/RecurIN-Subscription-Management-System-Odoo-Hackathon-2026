package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services"
)

type rolePermissionRequest struct {
	ResourceKey string `json:"resource_key"`
	CanCreate   bool   `json:"can_create"`
	CanRead     bool   `json:"can_read"`
	CanUpdate   bool   `json:"can_update"`
	CanDelete   bool   `json:"can_delete"`
}

type roleRequest struct {
	RoleName    string                  `json:"role_name"`
	UserID      string                  `json:"user_id"`
	Permissions []rolePermissionRequest `json:"permissions"`
}

// RoleHandler handles role profile and permission endpoints.
type RoleHandler struct {
	roleService *services.RoleService
}

func NewRoleHandler(roleService *services.RoleService) *RoleHandler {
	return &RoleHandler{roleService: roleService}
}

func buildRoleResponse(role models.RoleProfile) map[string]interface{} {
	permissions := make([]map[string]interface{}, 0, len(role.Permissions))
	for _, permission := range role.Permissions {
		permissions = append(permissions, map[string]interface{}{
			"resource_key": permission.ResourceKey,
			"can_create":   permission.CanCreate,
			"can_read":     permission.CanRead,
			"can_update":   permission.CanUpdate,
			"can_delete":   permission.CanDelete,
		})
	}

	return map[string]interface{}{
		"role_id":     role.RoleID,
		"role_name":   role.RoleName,
		"user_id":     role.UserID,
		"user_name":   role.UserName,
		"user_email":  role.UserEmail,
		"is_system":   role.IsSystem,
		"permissions": permissions,
		"created_at":  role.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":  role.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func mapRoleRequest(payload roleRequest) services.CreateRoleInput {
	permissions := make([]services.RolePermissionInput, 0, len(payload.Permissions))
	for _, permission := range payload.Permissions {
		permissions = append(permissions, services.RolePermissionInput{
			ResourceKey: permission.ResourceKey,
			CanCreate:   permission.CanCreate,
			CanRead:     permission.CanRead,
			CanUpdate:   permission.CanUpdate,
			CanDelete:   permission.CanDelete,
		})
	}

	return services.CreateRoleInput{
		RoleName:    payload.RoleName,
		UserID:      payload.UserID,
		Permissions: permissions,
	}
}

func (handler *RoleHandler) writeRoleError(writer http.ResponseWriter, err error) {
	var validationError services.ValidationError
	if errors.As(err, &validationError) {
		http.Error(writer, validationError.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, services.ErrRoleNotFound) {
		http.Error(writer, "Role not found.", http.StatusNotFound)
		return
	}
	if errors.Is(err, services.ErrRoleAlreadyExists) {
		http.Error(writer, "Role name already exists.", http.StatusConflict)
		return
	}
	if errors.Is(err, services.ErrRoleUserAlreadyAssigned) {
		http.Error(writer, "Selected user is already assigned to another role.", http.StatusConflict)
		return
	}
	if errors.Is(err, services.ErrSystemRoleModificationBlocked) {
		http.Error(writer, "System role cannot be modified or deleted.", http.StatusForbidden)
		return
	}

	log.Printf("role handler error: %v", err)
	http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
}

func (handler *RoleHandler) HandleCreateRole(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	var payload roleRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	createdRole, err := handler.roleService.CreateRole(request.Context(), mapRoleRequest(payload))
	if err != nil {
		handler.writeRoleError(writer, err)
		return
	}

	writeJSON(writer, http.StatusCreated, map[string]interface{}{
		"message": "Role created successfully.",
		"role":    buildRoleResponse(createdRole),
	})
}

func (handler *RoleHandler) HandleListRoles(writer http.ResponseWriter, request *http.Request) {
	search := request.URL.Query().Get("search")

	page, hasPage, err := parsePageQuery(request)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	pageForQuery := 0
	pageForResponse := 1
	if hasPage {
		pageForQuery = page
		pageForResponse = page
	}

	roles, totalRecords, err := handler.roleService.ListRoles(request.Context(), search, pageForQuery, adminListPageSize)
	if err != nil {
		handler.writeRoleError(writer, err)
		return
	}

	items := make([]map[string]interface{}, 0, len(roles))
	for _, role := range roles {
		items = append(items, buildRoleResponse(role))
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"roles":      items,
		"pagination": buildPaginationResponse(pageForResponse, adminListPageSize, totalRecords),
	})
}

func (handler *RoleHandler) HandleGetRoleByID(writer http.ResponseWriter, request *http.Request) {
	roleID := request.PathValue("roleID")

	role, err := handler.roleService.GetRoleByID(request.Context(), roleID)
	if err != nil {
		handler.writeRoleError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"role": buildRoleResponse(role),
	})
}

func (handler *RoleHandler) HandleUpdateRole(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	roleID := request.PathValue("roleID")

	var payload roleRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	updatedRole, err := handler.roleService.UpdateRole(request.Context(), roleID, mapRoleRequest(payload))
	if err != nil {
		handler.writeRoleError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"message": "Role updated successfully.",
		"role":    buildRoleResponse(updatedRole),
	})
}

func (handler *RoleHandler) HandleDeleteRole(writer http.ResponseWriter, request *http.Request) {
	roleID := request.PathValue("roleID")

	if err := handler.roleService.DeleteRole(request.Context(), roleID); err != nil {
		handler.writeRoleError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"message": "Role deleted successfully.",
	})
}
