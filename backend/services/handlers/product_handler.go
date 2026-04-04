package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services"
)

type productVariantRequest struct {
	AttributeID string `json:"attribute_id"`
}

type productRequest struct {
	ProductName     string                  `json:"product_name"`
	ProductType     string                  `json:"product_type"`
	SalesPrice      float64                 `json:"sales_price"`
	CostPrice       float64                 `json:"cost_price"`
	RecurringPlanID string                  `json:"recurring_plan_id"`
	TaxIDs          []string                `json:"tax_ids"`
	DiscountIDs     []string                `json:"discount_ids"`
	Variants        []productVariantRequest `json:"variants"`
}

// ProductHandler handles product administration endpoints.
type ProductHandler struct {
	productService *services.ProductService
}

func NewProductHandler(productService *services.ProductService) *ProductHandler {
	return &ProductHandler{productService: productService}
}

func mapProductVariants(variants []productVariantRequest) []services.CreateProductVariantInput {
	mappedVariants := make([]services.CreateProductVariantInput, 0, len(variants))
	for _, variant := range variants {
		mappedVariants = append(mappedVariants, services.CreateProductVariantInput{
			AttributeID: variant.AttributeID,
		})
	}

	return mappedVariants
}

func (handler *ProductHandler) HandleCreateProduct(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	var payload productRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	createdProduct, err := handler.productService.CreateProduct(request.Context(), services.CreateProductInput{
		ProductName:     payload.ProductName,
		ProductType:     payload.ProductType,
		SalesPrice:      payload.SalesPrice,
		CostPrice:       payload.CostPrice,
		RecurringPlanID: payload.RecurringPlanID,
		TaxIDs:          payload.TaxIDs,
		DiscountIDs:     payload.DiscountIDs,
		Variants:        mapProductVariants(payload.Variants),
	})
	if err != nil {
		handler.writeProductError(writer, err)
		return
	}

	writeJSON(writer, http.StatusCreated, map[string]interface{}{
		"message": "Product created successfully.",
		"product": buildProductResponse(createdProduct),
	})
}

func (handler *ProductHandler) HandleListProducts(writer http.ResponseWriter, request *http.Request) {
	search := request.URL.Query().Get("search")

	products, err := handler.productService.ListProducts(request.Context(), search)
	if err != nil {
		log.Printf("product list handler error: %v", err)
		http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
		return
	}

	items := make([]map[string]interface{}, 0, len(products))
	for _, product := range products {
		items = append(items, buildProductResponse(product))
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"products": items,
	})
}

func (handler *ProductHandler) HandleGetProductByID(writer http.ResponseWriter, request *http.Request) {
	productID := request.PathValue("productID")

	product, err := handler.productService.GetProductByID(request.Context(), productID)
	if err != nil {
		handler.writeProductError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"product": buildProductResponse(product),
	})
}

func (handler *ProductHandler) HandleUpdateProduct(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	productID := request.PathValue("productID")

	var payload productRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(writer, "Invalid request payload.", http.StatusBadRequest)
		return
	}

	updatedProduct, err := handler.productService.UpdateProduct(request.Context(), productID, services.CreateProductInput{
		ProductName:     payload.ProductName,
		ProductType:     payload.ProductType,
		SalesPrice:      payload.SalesPrice,
		CostPrice:       payload.CostPrice,
		RecurringPlanID: payload.RecurringPlanID,
		TaxIDs:          payload.TaxIDs,
		DiscountIDs:     payload.DiscountIDs,
		Variants:        mapProductVariants(payload.Variants),
	})
	if err != nil {
		handler.writeProductError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"message": "Product updated successfully.",
		"product": buildProductResponse(updatedProduct),
	})
}

func (handler *ProductHandler) HandleDeleteProduct(writer http.ResponseWriter, request *http.Request) {
	productID := request.PathValue("productID")

	if err := handler.productService.DeleteProduct(request.Context(), productID); err != nil {
		handler.writeProductError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]string{
		"message": "Product deleted successfully.",
	})
}

func (handler *ProductHandler) writeProductError(writer http.ResponseWriter, err error) {
	var validationError services.ValidationError
	if errors.As(err, &validationError) {
		http.Error(writer, validationError.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, services.ErrProductNotFound) {
		http.Error(writer, "Product not found.", http.StatusNotFound)
		return
	}
	if errors.Is(err, services.ErrProductAlreadyExists) {
		http.Error(writer, "Product name already exists.", http.StatusConflict)
		return
	}

	log.Printf("product handler error: %v", err)
	http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
}

func buildProductResponse(product models.Product) map[string]interface{} {
	productTaxes := make([]map[string]interface{}, 0, len(product.Taxes))
	for _, tax := range product.Taxes {
		productTaxes = append(productTaxes, map[string]interface{}{
			"tax_id":                tax.TaxID,
			"tax_name":              tax.TaxName,
			"tax_computation_unit":  tax.TaxComputationUnit,
			"tax_computation_value": tax.TaxComputationValue,
		})
	}

	productDiscounts := make([]map[string]interface{}, 0, len(product.Discounts))
	for _, discount := range product.Discounts {
		productDiscounts = append(productDiscounts, map[string]interface{}{
			"discount_id":    discount.DiscountID,
			"discount_name":  discount.DiscountName,
			"discount_unit":  discount.DiscountUnit,
			"discount_value": discount.DiscountValue,
		})
	}

	productVariants := make([]map[string]interface{}, 0, len(product.Variants))
	for _, variant := range product.Variants {
		productVariants = append(productVariants, map[string]interface{}{
			"attribute_id":        variant.AttributeID,
			"attribute_name":      variant.AttributeName,
			"default_extra_price": variant.DefaultExtraPrice,
		})
	}

	return map[string]interface{}{
		"product_id":        product.ProductID,
		"product_name":      product.ProductName,
		"product_type":      product.ProductType,
		"sales_price":       product.SalesPrice,
		"cost_price":        product.CostPrice,
		"recurring_plan_id": product.RecurringPlanID,
		"recurring_name":    product.RecurringName,
		"billing_period":    product.BillingPeriod,
		"taxes":             productTaxes,
		"discounts":         productDiscounts,
		"variants":          productVariants,
		"created_at":        product.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":        product.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
