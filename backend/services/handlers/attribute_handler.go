package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services"
)

type createAttributeRequest struct {
	AttributeName string                        `json:"attribute_name"`
	Values        []createAttributeValueRequest `json:"values"`
}

type createAttributeValueRequest struct {
	AttributeValue    string  `json:"attribute_value"`
	DefaultExtraPrice float64 `json:"default_extra_price"`
}

// AttributeHandler handles attribute administration endpoints.
type AttributeHandler struct {
	attributeService *services.AttributeService
}

func NewAttributeHandler(attributeService *services.AttributeService) *AttributeHandler {
	return &AttributeHandler{attributeService: attributeService}
}

func mapCreateAttributeValues(values []createAttributeValueRequest) []services.CreateAttributeValueInput {
	inputValues := make([]services.CreateAttributeValueInput, 0, len(values))
	for _, value := range values {
		inputValues = append(inputValues, services.CreateAttributeValueInput{
			AttributeValue:    value.AttributeValue,
			DefaultExtraPrice: value.DefaultExtraPrice,
		})
	}

	return inputValues
}

func (handler *AttributeHandler) HandleCreateAttribute(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	var payload createAttributeRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	createdAttribute, err := handler.attributeService.CreateAttribute(request.Context(), services.CreateAttributeInput{
		AttributeName: payload.AttributeName,
		Values:        mapCreateAttributeValues(payload.Values),
	})
	if err != nil {
		handler.writeAttributeError(writer, err)
		return
	}

	writeJSON(writer, http.StatusCreated, map[string]interface{}{
		"message":   "Attribute created successfully",
		"attribute": buildAttributeResponse(createdAttribute),
	})
}

func (handler *AttributeHandler) HandleListAttributes(writer http.ResponseWriter, request *http.Request) {
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

	attributes, totalRecords, err := handler.attributeService.ListAttributes(request.Context(), search, pageForQuery, adminListPageSize)
	if err != nil {
		log.Printf("attribute list handler error: %v", err)
		http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
		return
	}

	items := make([]map[string]interface{}, 0, len(attributes))
	for _, attribute := range attributes {
		items = append(items, buildAttributeResponse(attribute))
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"attributes": items,
		"pagination": buildPaginationResponse(pageForResponse, adminListPageSize, totalRecords),
	})
}

func (handler *AttributeHandler) HandleGetAttributeByID(writer http.ResponseWriter, request *http.Request) {
	attributeID := request.PathValue("attributeID")

	attribute, err := handler.attributeService.GetAttributeByID(request.Context(), attributeID)
	if err != nil {
		handler.writeAttributeError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"attribute": buildAttributeResponse(attribute),
	})
}

func (handler *AttributeHandler) HandleUpdateAttribute(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	attributeID := request.PathValue("attributeID")

	var payload createAttributeRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	updatedAttribute, err := handler.attributeService.UpdateAttribute(request.Context(), attributeID, services.CreateAttributeInput{
		AttributeName: payload.AttributeName,
		Values:        mapCreateAttributeValues(payload.Values),
	})
	if err != nil {
		handler.writeAttributeError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"message":   "Attribute updated successfully.",
		"attribute": buildAttributeResponse(updatedAttribute),
	})
}

func (handler *AttributeHandler) HandleDeleteAttribute(writer http.ResponseWriter, request *http.Request) {
	attributeID := request.PathValue("attributeID")

	if err := handler.attributeService.DeleteAttribute(request.Context(), attributeID); err != nil {
		handler.writeAttributeError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]string{
		"message": "Attribute deleted successfully.",
	})
}

func (handler *AttributeHandler) writeAttributeError(writer http.ResponseWriter, err error) {
	var validationError services.ValidationError
	if errors.As(err, &validationError) {
		http.Error(writer, validationError.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, services.ErrAttributeNotFound) {
		http.Error(writer, "Attribute not found.", http.StatusNotFound)
		return
	}
	if errors.Is(err, services.ErrAttributeAlreadyExists) {
		http.Error(writer, "Attribute name already exists.", http.StatusConflict)
		return
	}

	log.Printf("attribute handler error: %v", err)
	http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
}

func buildAttributeResponse(attribute models.Attribute) map[string]interface{} {
	attributeValues := make([]map[string]interface{}, 0, len(attribute.Values))
	for _, attributeValue := range attribute.Values {
		attributeValues = append(attributeValues, map[string]interface{}{
			"attribute_value_id":  attributeValue.AttributeValueID,
			"attribute_id":        attributeValue.AttributeID,
			"attribute_value":     attributeValue.AttributeValue,
			"default_extra_price": attributeValue.DefaultExtraPrice,
			"created_at":          attributeValue.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			"updated_at":          attributeValue.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return map[string]interface{}{
		"attribute_id":   attribute.AttributeID,
		"attribute_name": attribute.AttributeName,
		"created_at":     attribute.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":     attribute.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"values":         attributeValues,
	}
}
