package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
)

var (
	ErrAttributeAlreadyExists = errors.New("attribute already exists")
	ErrAttributeNotFound      = errors.New("attribute not found")
)

type CreateAttributeValueInput struct {
	AttributeValue    string
	DefaultExtraPrice float64
}

type CreateAttributeInput struct {
	AttributeName string
	Values        []CreateAttributeValueInput
}

// AttributeService encapsulates attribute operations.
type AttributeService struct {
	db *pgxpool.Pool
}

func NewAttributeService(db *pgxpool.Pool) *AttributeService {
	return &AttributeService{db: db}
}

func (service *AttributeService) CreateAttribute(ctx context.Context, input CreateAttributeInput) (models.Attribute, error) {
	attributeName := strings.TrimSpace(input.AttributeName)
	if attributeName == "" {
		return models.Attribute{}, ValidationError{Message: "Attribute name is required."}
	}

	if len(input.Values) == 0 {
		return models.Attribute{}, ValidationError{Message: "At least one attribute value is required."}
	}

	normalizedValues := make([]CreateAttributeValueInput, 0, len(input.Values))
	seenValues := make(map[string]struct{}, len(input.Values))
	for _, inputValue := range input.Values {
		attributeValue := strings.TrimSpace(inputValue.AttributeValue)
		if attributeValue == "" {
			return models.Attribute{}, ValidationError{Message: "Attribute value is required for all rows."}
		}
		if inputValue.DefaultExtraPrice < 0 {
			return models.Attribute{}, ValidationError{Message: "Default extra price cannot be negative."}
		}

		normalizedKey := strings.ToLower(attributeValue)
		if _, exists := seenValues[normalizedKey]; exists {
			return models.Attribute{}, ValidationError{Message: "Duplicate attribute values are not allowed."}
		}
		seenValues[normalizedKey] = struct{}{}

		normalizedValues = append(normalizedValues, CreateAttributeValueInput{
			AttributeValue:    attributeValue,
			DefaultExtraPrice: inputValue.DefaultExtraPrice,
		})
	}

	tx, err := service.db.Begin(ctx)
	if err != nil {
		return models.Attribute{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const insertAttributeQuery = `
		INSERT INTO attributes.attribute (attribute_name)
		VALUES ($1)
		RETURNING attribute_id, attribute_name, created_at, updated_at`

	var attribute models.Attribute
	if err := tx.QueryRow(ctx, insertAttributeQuery, attributeName).Scan(
		&attribute.AttributeID,
		&attribute.AttributeName,
		&attribute.CreatedAt,
		&attribute.UpdatedAt,
	); err != nil {
		if isAttributeNameUniqueViolation(err) {
			return models.Attribute{}, ErrAttributeAlreadyExists
		}
		return models.Attribute{}, fmt.Errorf("failed to create attribute: %w", err)
	}

	const insertAttributeValueQuery = `
		INSERT INTO attributes.attribute_values (attribute_id, attribute_value, default_extra_price)
		VALUES ($1, $2, $3)
		RETURNING attribute_value_id, attribute_id, attribute_value, default_extra_price::float8, created_at, updated_at`

	createdValues := make([]models.AttributeValue, 0, len(normalizedValues))
	for _, normalizedValue := range normalizedValues {
		var attributeValue models.AttributeValue
		if err := tx.QueryRow(ctx, insertAttributeValueQuery,
			attribute.AttributeID,
			normalizedValue.AttributeValue,
			normalizedValue.DefaultExtraPrice,
		).Scan(
			&attributeValue.AttributeValueID,
			&attributeValue.AttributeID,
			&attributeValue.AttributeValue,
			&attributeValue.DefaultExtraPrice,
			&attributeValue.CreatedAt,
			&attributeValue.UpdatedAt,
		); err != nil {
			if isAttributeValueUniqueViolation(err) {
				return models.Attribute{}, ValidationError{Message: "Duplicate attribute values are not allowed."}
			}
			return models.Attribute{}, fmt.Errorf("failed to create attribute values: %w", err)
		}

		createdValues = append(createdValues, attributeValue)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Attribute{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	attribute.Values = createdValues
	return attribute, nil
}

func (service *AttributeService) ListAttributes(ctx context.Context, search string, page int, limit int) ([]models.Attribute, int, error) {
	if limit <= 0 || limit > 30 {
		limit = 30
	}

	normalizedSearch := strings.TrimSpace(search)

	if page <= 0 {
		const query = `
			SELECT
				a.attribute_id,
				a.attribute_name,
				a.created_at,
				a.updated_at,
				av.attribute_value_id,
				av.attribute_id,
				av.attribute_value,
				av.default_extra_price::float8,
				av.created_at,
				av.updated_at
			FROM attributes.attribute a
			LEFT JOIN attributes.attribute_values av ON av.attribute_id = a.attribute_id
			WHERE (
				$1 = ''
				OR a.attribute_name ILIKE '%' || $1 || '%'
				OR EXISTS (
					SELECT 1
					FROM attributes.attribute_values avs
					WHERE avs.attribute_id = a.attribute_id
					  AND avs.attribute_value ILIKE '%' || $1 || '%'
				)
			)
			ORDER BY a.attribute_name ASC, av.attribute_value ASC`

		rows, err := service.db.Query(ctx, query, normalizedSearch)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to list attributes: %w", err)
		}
		defer rows.Close()

		attributes, err := scanAttributeListRows(rows)
		if err != nil {
			return nil, 0, err
		}

		return attributes, len(attributes), nil
	}

	offset := (page - 1) * limit

	const countQuery = `
		SELECT COUNT(*)
		FROM attributes.attribute a
		WHERE (
			$1 = ''
			OR a.attribute_name ILIKE '%' || $1 || '%'
			OR EXISTS (
				SELECT 1
				FROM attributes.attribute_values avs
				WHERE avs.attribute_id = a.attribute_id
				  AND avs.attribute_value ILIKE '%' || $1 || '%'
			)
		)`

	var totalRecords int
	if err := service.db.QueryRow(ctx, countQuery, normalizedSearch).Scan(&totalRecords); err != nil {
		return nil, 0, fmt.Errorf("failed to count attributes: %w", err)
	}

	const query = `
		WITH paged_attributes AS (
			SELECT
				a.attribute_id,
				a.attribute_name,
				a.created_at,
				a.updated_at
			FROM attributes.attribute a
			WHERE (
				$1 = ''
				OR a.attribute_name ILIKE '%' || $1 || '%'
				OR EXISTS (
					SELECT 1
					FROM attributes.attribute_values avs
					WHERE avs.attribute_id = a.attribute_id
					  AND avs.attribute_value ILIKE '%' || $1 || '%'
				)
			)
			ORDER BY a.attribute_name ASC
			LIMIT $2 OFFSET $3
		)
		SELECT
			pa.attribute_id,
			pa.attribute_name,
			pa.created_at,
			pa.updated_at,
			av.attribute_value_id,
			av.attribute_id,
			av.attribute_value,
			av.default_extra_price::float8,
			av.created_at,
			av.updated_at
		FROM paged_attributes pa
		LEFT JOIN attributes.attribute_values av ON av.attribute_id = pa.attribute_id
		ORDER BY pa.attribute_name ASC, av.attribute_value ASC`

	rows, err := service.db.Query(ctx, query, normalizedSearch, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list attributes: %w", err)
	}
	defer rows.Close()

	attributes, err := scanAttributeListRows(rows)
	if err != nil {
		return nil, 0, err
	}

	return attributes, totalRecords, nil
}

func scanAttributeListRows(rows pgx.Rows) ([]models.Attribute, error) {
	attributes := make([]models.Attribute, 0)
	attributeIndexByID := make(map[string]int)

	for rows.Next() {
		var (
			attributeID       string
			attributeName     string
			attributeCreated  time.Time
			attributeUpdated  time.Time
			attributeValueID  *string
			valueAttributeID  *string
			attributeValue    *string
			defaultExtraPrice *float64
			valueCreated      *time.Time
			valueUpdated      *time.Time
		)

		if err := rows.Scan(
			&attributeID,
			&attributeName,
			&attributeCreated,
			&attributeUpdated,
			&attributeValueID,
			&valueAttributeID,
			&attributeValue,
			&defaultExtraPrice,
			&valueCreated,
			&valueUpdated,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attribute row: %w", err)
		}

		attributePosition, exists := attributeIndexByID[attributeID]
		if !exists {
			attributes = append(attributes, models.Attribute{
				AttributeID:   attributeID,
				AttributeName: attributeName,
				Values:        make([]models.AttributeValue, 0),
				CreatedAt:     attributeCreated,
				UpdatedAt:     attributeUpdated,
			})
			attributePosition = len(attributes) - 1
			attributeIndexByID[attributeID] = attributePosition
		}

		if attributeValueID != nil && valueAttributeID != nil && attributeValue != nil && defaultExtraPrice != nil && valueCreated != nil && valueUpdated != nil {
			attributes[attributePosition].Values = append(attributes[attributePosition].Values, models.AttributeValue{
				AttributeValueID:  *attributeValueID,
				AttributeID:       *valueAttributeID,
				AttributeValue:    *attributeValue,
				DefaultExtraPrice: *defaultExtraPrice,
				CreatedAt:         *valueCreated,
				UpdatedAt:         *valueUpdated,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating attribute rows: %w", err)
	}

	return attributes, nil
}

func (service *AttributeService) GetAttributeByID(ctx context.Context, attributeID string) (models.Attribute, error) {
	normalizedAttributeID := strings.TrimSpace(attributeID)
	if normalizedAttributeID == "" {
		return models.Attribute{}, ValidationError{Message: "Attribute ID is required."}
	}

	const query = `
		SELECT
			a.attribute_id,
			a.attribute_name,
			a.created_at,
			a.updated_at,
			av.attribute_value_id,
			av.attribute_id,
			av.attribute_value,
			av.default_extra_price::float8,
			av.created_at,
			av.updated_at
		FROM attributes.attribute a
		LEFT JOIN attributes.attribute_values av ON av.attribute_id = a.attribute_id
		WHERE a.attribute_id = $1
		ORDER BY av.attribute_value ASC`

	rows, err := service.db.Query(ctx, query, normalizedAttributeID)
	if err != nil {
		return models.Attribute{}, fmt.Errorf("failed to get attribute: %w", err)
	}
	defer rows.Close()

	var (
		attribute models.Attribute
		found     bool
	)

	for rows.Next() {
		var (
			rowAttributeID    string
			rowAttributeName  string
			attributeCreated  time.Time
			attributeUpdated  time.Time
			attributeValueID  *string
			valueAttributeID  *string
			attributeValue    *string
			defaultExtraPrice *float64
			valueCreated      *time.Time
			valueUpdated      *time.Time
		)

		if err := rows.Scan(
			&rowAttributeID,
			&rowAttributeName,
			&attributeCreated,
			&attributeUpdated,
			&attributeValueID,
			&valueAttributeID,
			&attributeValue,
			&defaultExtraPrice,
			&valueCreated,
			&valueUpdated,
		); err != nil {
			return models.Attribute{}, fmt.Errorf("failed to scan attribute row: %w", err)
		}

		if !found {
			attribute = models.Attribute{
				AttributeID:   rowAttributeID,
				AttributeName: rowAttributeName,
				Values:        make([]models.AttributeValue, 0),
				CreatedAt:     attributeCreated,
				UpdatedAt:     attributeUpdated,
			}
			found = true
		}

		if attributeValueID != nil && valueAttributeID != nil && attributeValue != nil && defaultExtraPrice != nil && valueCreated != nil && valueUpdated != nil {
			attribute.Values = append(attribute.Values, models.AttributeValue{
				AttributeValueID:  *attributeValueID,
				AttributeID:       *valueAttributeID,
				AttributeValue:    *attributeValue,
				DefaultExtraPrice: *defaultExtraPrice,
				CreatedAt:         *valueCreated,
				UpdatedAt:         *valueUpdated,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return models.Attribute{}, fmt.Errorf("failed while iterating attribute rows: %w", err)
	}

	if !found {
		return models.Attribute{}, ErrAttributeNotFound
	}

	return attribute, nil
}

func (service *AttributeService) UpdateAttribute(ctx context.Context, attributeID string, input CreateAttributeInput) (models.Attribute, error) {
	normalizedAttributeID := strings.TrimSpace(attributeID)
	if normalizedAttributeID == "" {
		return models.Attribute{}, ValidationError{Message: "Attribute ID is required."}
	}

	attributeName := strings.TrimSpace(input.AttributeName)
	if attributeName == "" {
		return models.Attribute{}, ValidationError{Message: "Attribute name is required."}
	}

	if len(input.Values) == 0 {
		return models.Attribute{}, ValidationError{Message: "At least one attribute value is required."}
	}

	normalizedValues := make([]CreateAttributeValueInput, 0, len(input.Values))
	seenValues := make(map[string]struct{}, len(input.Values))
	for _, inputValue := range input.Values {
		attributeValue := strings.TrimSpace(inputValue.AttributeValue)
		if attributeValue == "" {
			return models.Attribute{}, ValidationError{Message: "Attribute value is required for all rows."}
		}
		if inputValue.DefaultExtraPrice < 0 {
			return models.Attribute{}, ValidationError{Message: "Default extra price cannot be negative."}
		}

		normalizedKey := strings.ToLower(attributeValue)
		if _, exists := seenValues[normalizedKey]; exists {
			return models.Attribute{}, ValidationError{Message: "Duplicate attribute values are not allowed."}
		}
		seenValues[normalizedKey] = struct{}{}

		normalizedValues = append(normalizedValues, CreateAttributeValueInput{
			AttributeValue:    attributeValue,
			DefaultExtraPrice: inputValue.DefaultExtraPrice,
		})
	}

	tx, err := service.db.Begin(ctx)
	if err != nil {
		return models.Attribute{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var existingAttributeID string
	if err := tx.QueryRow(ctx, `SELECT attribute_id FROM attributes.attribute WHERE attribute_id = $1`, normalizedAttributeID).Scan(&existingAttributeID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Attribute{}, ErrAttributeNotFound
		}
		return models.Attribute{}, fmt.Errorf("failed to validate attribute: %w", err)
	}

	const updateAttributeQuery = `
		UPDATE attributes.attribute
		SET attribute_name = $1, updated_at = NOW()
		WHERE attribute_id = $2
		RETURNING attribute_id, attribute_name, created_at, updated_at`

	var attribute models.Attribute
	if err := tx.QueryRow(ctx, updateAttributeQuery, attributeName, normalizedAttributeID).Scan(
		&attribute.AttributeID,
		&attribute.AttributeName,
		&attribute.CreatedAt,
		&attribute.UpdatedAt,
	); err != nil {
		if isAttributeNameUniqueViolation(err) {
			return models.Attribute{}, ErrAttributeAlreadyExists
		}
		return models.Attribute{}, fmt.Errorf("failed to update attribute: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM attributes.attribute_values WHERE attribute_id = $1`, normalizedAttributeID); err != nil {
		return models.Attribute{}, fmt.Errorf("failed to refresh attribute values: %w", err)
	}

	const insertAttributeValueQuery = `
		INSERT INTO attributes.attribute_values (attribute_id, attribute_value, default_extra_price)
		VALUES ($1, $2, $3)
		RETURNING attribute_value_id, attribute_id, attribute_value, default_extra_price::float8, created_at, updated_at`

	createdValues := make([]models.AttributeValue, 0, len(normalizedValues))
	for _, normalizedValue := range normalizedValues {
		var attributeValue models.AttributeValue
		if err := tx.QueryRow(ctx, insertAttributeValueQuery,
			attribute.AttributeID,
			normalizedValue.AttributeValue,
			normalizedValue.DefaultExtraPrice,
		).Scan(
			&attributeValue.AttributeValueID,
			&attributeValue.AttributeID,
			&attributeValue.AttributeValue,
			&attributeValue.DefaultExtraPrice,
			&attributeValue.CreatedAt,
			&attributeValue.UpdatedAt,
		); err != nil {
			if isAttributeValueUniqueViolation(err) {
				return models.Attribute{}, ValidationError{Message: "Duplicate attribute values are not allowed."}
			}
			return models.Attribute{}, fmt.Errorf("failed to update attribute values: %w", err)
		}

		createdValues = append(createdValues, attributeValue)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Attribute{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	attribute.Values = createdValues
	return attribute, nil
}

func (service *AttributeService) DeleteAttribute(ctx context.Context, attributeID string) error {
	normalizedAttributeID := strings.TrimSpace(attributeID)
	if normalizedAttributeID == "" {
		return ValidationError{Message: "Attribute ID is required."}
	}

	result, err := service.db.Exec(ctx, `DELETE FROM attributes.attribute WHERE attribute_id = $1`, normalizedAttributeID)
	if err != nil {
		return fmt.Errorf("failed to delete attribute: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrAttributeNotFound
	}

	return nil
}

func isAttributeNameUniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23505" && pgError.ConstraintName == "attribute_attribute_name_key"
	}

	return false
}

func isAttributeValueUniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23505" && pgError.ConstraintName == "uq_attribute_values_attribute_id_value"
	}

	return false
}
