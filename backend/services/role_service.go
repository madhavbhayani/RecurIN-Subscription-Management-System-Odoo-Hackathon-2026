package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services/common/rbac"
)

var (
	ErrRoleNotFound                  = errors.New("role not found")
	ErrRoleAlreadyExists             = errors.New("role already exists")
	ErrRoleUserAlreadyAssigned       = errors.New("user already assigned to a role")
	ErrSystemRoleModificationBlocked = errors.New("system role cannot be modified")
)

type RolePermissionInput struct {
	ResourceKey string
	CanCreate   bool
	CanRead     bool
	CanUpdate   bool
	CanDelete   bool
}

type CreateRoleInput struct {
	RoleName    string
	UserID      string
	Permissions []RolePermissionInput
}

func normalizeManagedRoleName(roleName string) (string, bool) {
	switch strings.TrimSpace(strings.ToLower(roleName)) {
	case "user":
		return "User", true
	case "admin":
		return "Admin", true
	case "internal", "internal-user", "internal_user":
		return "Internal", true
	default:
		return "", false
	}
}

// RoleService encapsulates role profile and permission operations.
type RoleService struct {
	db *pgxpool.Pool
}

func NewRoleService(db *pgxpool.Pool) *RoleService {
	return &RoleService{db: db}
}

func normalizeRolePermissions(permissionInputs []RolePermissionInput) ([]models.RolePermission, error) {
	resourceKeys := rbac.ListPermissionResources()
	permissionsByResource := make(map[string]models.RolePermission, len(resourceKeys))

	for _, resourceKey := range resourceKeys {
		permissionsByResource[resourceKey] = models.RolePermission{
			ResourceKey: resourceKey,
		}
	}

	for _, permissionInput := range permissionInputs {
		normalizedResourceKey := strings.TrimSpace(strings.ToLower(permissionInput.ResourceKey))
		if !rbac.IsValidPermissionResource(normalizedResourceKey) {
			return nil, ValidationError{Message: "Invalid resource key in permissions."}
		}

		permissionsByResource[normalizedResourceKey] = models.RolePermission{
			ResourceKey: normalizedResourceKey,
			CanCreate:   permissionInput.CanCreate,
			CanRead:     permissionInput.CanRead,
			CanUpdate:   permissionInput.CanUpdate,
			CanDelete:   permissionInput.CanDelete,
		}
	}

	normalizedPermissions := make([]models.RolePermission, 0, len(resourceKeys))
	for _, resourceKey := range resourceKeys {
		normalizedPermissions = append(normalizedPermissions, permissionsByResource[resourceKey])
	}

	return normalizedPermissions, nil
}

func validateRoleInput(input CreateRoleInput) (CreateRoleInput, []models.RolePermission, error) {
	rawRoleName := strings.TrimSpace(input.RoleName)
	if rawRoleName == "" {
		return CreateRoleInput{}, nil, ValidationError{Message: "Role name is required."}
	}

	roleName, validRoleName := normalizeManagedRoleName(rawRoleName)
	if !validRoleName {
		return CreateRoleInput{}, nil, ValidationError{Message: "Role name must be User, Admin or Internal."}
	}

	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return CreateRoleInput{}, nil, ValidationError{Message: "User is required."}
	}

	normalizedPermissions, err := normalizeRolePermissions(input.Permissions)
	if err != nil {
		return CreateRoleInput{}, nil, err
	}

	return CreateRoleInput{
		RoleName: roleName,
		UserID:   userID,
	}, normalizedPermissions, nil
}

type roleQuerier interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

func scanRoleBaseRow(row pgx.Row) (models.RoleProfile, error) {
	var role models.RoleProfile
	var userID *string
	var userName *string
	var userEmail *string

	if err := row.Scan(
		&role.RoleID,
		&role.RoleName,
		&userID,
		&userName,
		&userEmail,
		&role.IsSystem,
		&role.CreatedAt,
		&role.UpdatedAt,
	); err != nil {
		return models.RoleProfile{}, err
	}

	role.UserID = userID
	role.UserName = userName
	role.UserEmail = userEmail
	role.Permissions = []models.RolePermission{}

	return role, nil
}

func fetchRolePermissions(ctx context.Context, querier roleQuerier, roleID string) ([]models.RolePermission, error) {
	const query = `
		SELECT resource_key, can_create, can_read, can_update, can_delete
		FROM privileges.role_permissions
		WHERE role_id = $1`

	rows, err := querier.Query(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch role permissions: %w", err)
	}
	defer rows.Close()

	permissionsByResource := make(map[string]models.RolePermission)
	for rows.Next() {
		var permission models.RolePermission
		if err := rows.Scan(
			&permission.ResourceKey,
			&permission.CanCreate,
			&permission.CanRead,
			&permission.CanUpdate,
			&permission.CanDelete,
		); err != nil {
			return nil, fmt.Errorf("failed to scan role permission row: %w", err)
		}
		permissionsByResource[strings.ToLower(strings.TrimSpace(permission.ResourceKey))] = permission
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating role permission rows: %w", err)
	}

	resourceKeys := rbac.ListPermissionResources()
	permissions := make([]models.RolePermission, 0, len(resourceKeys))
	for _, resourceKey := range resourceKeys {
		permission, exists := permissionsByResource[resourceKey]
		if !exists {
			permission = models.RolePermission{ResourceKey: resourceKey}
		}
		permissions = append(permissions, permission)
	}

	sort.SliceStable(permissions, func(left, right int) bool {
		return permissions[left].ResourceKey < permissions[right].ResourceKey
	})

	return permissions, nil
}

func isRoleNameUniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23505" && strings.Contains(pgError.ConstraintName, "role_name")
	}

	return false
}

func isRoleUserUniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23505" && strings.Contains(pgError.ConstraintName, "user_id")
	}

	return false
}

func (service *RoleService) ensureAssignableUser(ctx context.Context, querier roleQuerier, userID string) error {
	const query = `
		SELECT role
		FROM users."user"
		WHERE id = $1`

	var userRole string
	if err := querier.QueryRow(ctx, query, userID).Scan(&userRole); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ValidationError{Message: "Selected user does not exist."}
		}
		return fmt.Errorf("failed to validate user for role assignment: %w", err)
	}

	normalizedUserRole := strings.TrimSpace(strings.ToLower(userRole))
	if normalizedUserRole == "admin" {
		return ValidationError{Message: "Admin user cannot be assigned to managed internal roles."}
	}

	return nil
}

func insertRolePermissions(ctx context.Context, tx pgx.Tx, roleID string, permissions []models.RolePermission) error {
	const query = `
		INSERT INTO privileges.role_permissions (
			role_id,
			resource_key,
			can_create,
			can_read,
			can_update,
			can_delete
		)
		VALUES ($1, $2, $3, $4, $5, $6)`

	for _, permission := range permissions {
		if _, err := tx.Exec(
			ctx,
			query,
			roleID,
			permission.ResourceKey,
			permission.CanCreate,
			permission.CanRead,
			permission.CanUpdate,
			permission.CanDelete,
		); err != nil {
			return fmt.Errorf("failed to insert role permissions: %w", err)
		}
	}

	return nil
}

func (service *RoleService) ListRoles(ctx context.Context, search string) ([]models.RoleProfile, error) {
	normalizedSearch := strings.TrimSpace(search)

	const query = `
		SELECT
			rd.role_id,
			rd.role_name,
			rd.user_id,
			u.name,
			u.email,
			rd.is_system,
			rd.created_at,
			rd.updated_at
		FROM privileges.role_data rd
		LEFT JOIN users."user" u ON u.id = rd.user_id
		WHERE (
			$1 = ''
			OR rd.role_name ILIKE '%' || $1 || '%'
			OR COALESCE(u.name, '') ILIKE '%' || $1 || '%'
			OR COALESCE(u.email, '') ILIKE '%' || $1 || '%'
		)
		ORDER BY rd.is_system DESC, rd.created_at DESC`

	rows, err := service.db.Query(ctx, query, normalizedSearch)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	roles := make([]models.RoleProfile, 0)
	for rows.Next() {
		role, scanErr := scanRoleBaseRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan role row: %w", scanErr)
		}

		permissions, permissionsErr := fetchRolePermissions(ctx, service.db, role.RoleID)
		if permissionsErr != nil {
			return nil, permissionsErr
		}
		role.Permissions = permissions
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating role rows: %w", err)
	}

	return roles, nil
}

func (service *RoleService) GetRoleByID(ctx context.Context, roleID string) (models.RoleProfile, error) {
	normalizedRoleID := strings.TrimSpace(roleID)
	if normalizedRoleID == "" {
		return models.RoleProfile{}, ValidationError{Message: "Role ID is required."}
	}

	const query = `
		SELECT
			rd.role_id,
			rd.role_name,
			rd.user_id,
			u.name,
			u.email,
			rd.is_system,
			rd.created_at,
			rd.updated_at
		FROM privileges.role_data rd
		LEFT JOIN users."user" u ON u.id = rd.user_id
		WHERE rd.role_id = $1`

	role, err := scanRoleBaseRow(service.db.QueryRow(ctx, query, normalizedRoleID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.RoleProfile{}, ErrRoleNotFound
		}
		return models.RoleProfile{}, fmt.Errorf("failed to fetch role: %w", err)
	}

	permissions, err := fetchRolePermissions(ctx, service.db, normalizedRoleID)
	if err != nil {
		return models.RoleProfile{}, err
	}

	role.Permissions = permissions
	return role, nil
}

func (service *RoleService) CreateRole(ctx context.Context, input CreateRoleInput) (models.RoleProfile, error) {
	validatedInput, normalizedPermissions, err := validateRoleInput(input)
	if err != nil {
		return models.RoleProfile{}, err
	}

	tx, err := service.db.Begin(ctx)
	if err != nil {
		return models.RoleProfile{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := service.ensureAssignableUser(ctx, tx, validatedInput.UserID); err != nil {
		return models.RoleProfile{}, err
	}

	const query = `
		INSERT INTO privileges.role_data (
			role_name,
			user_id,
			is_system
		)
		VALUES ($1, $2, FALSE)
		RETURNING role_id`

	var roleID string
	if err := tx.QueryRow(ctx, query, validatedInput.RoleName, validatedInput.UserID).Scan(&roleID); err != nil {
		if isRoleNameUniqueViolation(err) {
			return models.RoleProfile{}, ErrRoleAlreadyExists
		}
		if isRoleUserUniqueViolation(err) {
			return models.RoleProfile{}, ErrRoleUserAlreadyAssigned
		}
		return models.RoleProfile{}, fmt.Errorf("failed to create role: %w", err)
	}

	if err := insertRolePermissions(ctx, tx, roleID, normalizedPermissions); err != nil {
		return models.RoleProfile{}, err
	}

	if _, err := tx.Exec(ctx, `UPDATE users."user" SET role = 'Internal', updated_at = NOW() WHERE id = $1`, validatedInput.UserID); err != nil {
		return models.RoleProfile{}, fmt.Errorf("failed to update user role to Internal: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.RoleProfile{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return service.GetRoleByID(ctx, roleID)
}

func (service *RoleService) UpdateRole(ctx context.Context, roleID string, input CreateRoleInput) (models.RoleProfile, error) {
	normalizedRoleID := strings.TrimSpace(roleID)
	if normalizedRoleID == "" {
		return models.RoleProfile{}, ValidationError{Message: "Role ID is required."}
	}

	validatedInput, normalizedPermissions, err := validateRoleInput(input)
	if err != nil {
		return models.RoleProfile{}, err
	}

	tx, err := service.db.Begin(ctx)
	if err != nil {
		return models.RoleProfile{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const loadExistingRoleQuery = `
		SELECT user_id, is_system
		FROM privileges.role_data
		WHERE role_id = $1
		FOR UPDATE`

	var previousUserID *string
	var isSystemRole bool
	if err := tx.QueryRow(ctx, loadExistingRoleQuery, normalizedRoleID).Scan(&previousUserID, &isSystemRole); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.RoleProfile{}, ErrRoleNotFound
		}
		return models.RoleProfile{}, fmt.Errorf("failed to fetch role before update: %w", err)
	}

	if isSystemRole {
		return models.RoleProfile{}, ErrSystemRoleModificationBlocked
	}

	if err := service.ensureAssignableUser(ctx, tx, validatedInput.UserID); err != nil {
		return models.RoleProfile{}, err
	}

	const query = `
		UPDATE privileges.role_data
		SET
			role_name = $1,
			user_id = $2,
			updated_at = NOW()
		WHERE role_id = $3
		RETURNING role_id`

	var updatedRoleID string
	if err := tx.QueryRow(ctx, query, validatedInput.RoleName, validatedInput.UserID, normalizedRoleID).Scan(&updatedRoleID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.RoleProfile{}, ErrRoleNotFound
		}
		if isRoleNameUniqueViolation(err) {
			return models.RoleProfile{}, ErrRoleAlreadyExists
		}
		if isRoleUserUniqueViolation(err) {
			return models.RoleProfile{}, ErrRoleUserAlreadyAssigned
		}
		return models.RoleProfile{}, fmt.Errorf("failed to update role: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM privileges.role_permissions WHERE role_id = $1`, normalizedRoleID); err != nil {
		return models.RoleProfile{}, fmt.Errorf("failed to refresh role permissions: %w", err)
	}

	if err := insertRolePermissions(ctx, tx, normalizedRoleID, normalizedPermissions); err != nil {
		return models.RoleProfile{}, err
	}

	if previousUserID != nil && strings.TrimSpace(*previousUserID) != "" && strings.TrimSpace(*previousUserID) != validatedInput.UserID {
		if _, err := tx.Exec(ctx, `UPDATE users."user" SET role = 'User', updated_at = NOW() WHERE id = $1`, *previousUserID); err != nil {
			return models.RoleProfile{}, fmt.Errorf("failed to reset previous user role: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE users."user" SET role = 'Internal', updated_at = NOW() WHERE id = $1`, validatedInput.UserID); err != nil {
		return models.RoleProfile{}, fmt.Errorf("failed to update user role to Internal: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.RoleProfile{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return service.GetRoleByID(ctx, updatedRoleID)
}

func (service *RoleService) DeleteRole(ctx context.Context, roleID string) error {
	normalizedRoleID := strings.TrimSpace(roleID)
	if normalizedRoleID == "" {
		return ValidationError{Message: "Role ID is required."}
	}

	tx, err := service.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const loadRoleQuery = `
		SELECT user_id, is_system
		FROM privileges.role_data
		WHERE role_id = $1
		FOR UPDATE`

	var userID *string
	var isSystemRole bool
	if err := tx.QueryRow(ctx, loadRoleQuery, normalizedRoleID).Scan(&userID, &isSystemRole); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRoleNotFound
		}
		return fmt.Errorf("failed to fetch role before delete: %w", err)
	}

	if isSystemRole {
		return ErrSystemRoleModificationBlocked
	}

	if _, err := tx.Exec(ctx, `DELETE FROM privileges.role_data WHERE role_id = $1`, normalizedRoleID); err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	if userID != nil && strings.TrimSpace(*userID) != "" {
		if _, err := tx.Exec(ctx, `UPDATE users."user" SET role = 'User', updated_at = NOW() WHERE id = $1`, *userID); err != nil {
			return fmt.Errorf("failed to reset user role after role delete: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
