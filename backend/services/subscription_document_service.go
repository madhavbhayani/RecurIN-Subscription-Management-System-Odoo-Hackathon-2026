package services

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
)

// SubscriptionDocumentService generates invoice and quotation PDFs.
type SubscriptionDocumentService struct {
	logoPath string
}

func NewSubscriptionDocumentService(configuredLogoPath string) *SubscriptionDocumentService {
	return &SubscriptionDocumentService{
		logoPath: resolveSubscriptionLogoPath(configuredLogoPath),
	}
}

func resolveSubscriptionLogoPath(configuredLogoPath string) string {
	candidates := []string{
		strings.TrimSpace(configuredLogoPath),
		"src/assets/image.png",
		"./src/assets/image.png",
		"../src/assets/image.png",
	}

	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		normalizedCandidate := strings.TrimSpace(candidate)
		if normalizedCandidate == "" {
			continue
		}

		resolvedPath := normalizedCandidate
		if !filepath.IsAbs(normalizedCandidate) {
			absolutePath, err := filepath.Abs(normalizedCandidate)
			if err != nil {
				continue
			}
			resolvedPath = absolutePath
		}

		if _, exists := seen[resolvedPath]; exists {
			continue
		}
		seen[resolvedPath] = struct{}{}

		fileInfo, err := os.Stat(resolvedPath)
		if err != nil || fileInfo.IsDir() {
			continue
		}

		return resolvedPath
	}

	return ""
}

func formatDocumentAmountUSD(value float64) string {
	return fmt.Sprintf("$%.2f", value)
}

func formatDocumentDate(value time.Time) string {
	if value.IsZero() {
		return "-"
	}

	return value.Format("2006-01-02")
}

func (service *SubscriptionDocumentService) drawFallbackBrandHeader(pdf *gofpdf.Fpdf) {
	pdf.SetFillColor(255, 107, 0)
	pdf.RoundedRect(12, 12, 15, 15, 3, "F", "1234")

	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 13)
	pdf.Text(17.3, 21.8, "R")

	pdf.SetTextColor(255, 107, 0)
	pdf.SetFont("Arial", "B", 20)
	pdf.Text(31, 18.6, "Recur")
	pdf.SetTextColor(19, 136, 8)
	pdf.Text(57.8, 18.6, "IN")

	pdf.SetTextColor(30, 44, 120)
	pdf.SetFont("Arial", "", 9)
	pdf.Text(31, 23.4, "Subscription & Management")
}

func (service *SubscriptionDocumentService) drawBrandHeader(pdf *gofpdf.Fpdf) {
	if service.logoPath != "" {
		imageOptions := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
		pdf.ImageOptions(service.logoPath, 12, 10, 90, 24, false, imageOptions, 0, "")
	} else {
		service.drawFallbackBrandHeader(pdf)
	}

	pdf.SetDrawColor(220, 223, 233)
	pdf.Line(12, 36, 198, 36)
}

func (service *SubscriptionDocumentService) drawProductTableHeader(pdf *gofpdf.Fpdf) {
	headers := []string{"Product", "Qty", "Unit", "Discount", "Tax", "Total"}
	widths := []float64{66, 14, 25, 25, 22, 28}

	pdf.SetFillColor(243, 245, 251)
	pdf.SetDrawColor(220, 223, 233)
	pdf.SetTextColor(30, 44, 120)
	pdf.SetFont("Arial", "B", 9)

	for index, header := range headers {
		pdf.CellFormat(widths[index], 8, header, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
}

func normalizeProductLineTotal(product models.SubscriptionProduct) float64 {
	totalAmount := product.TotalAmount
	if totalAmount > 0 {
		return totalAmount
	}

	normalizedQuantity := product.Quantity
	if normalizedQuantity < 1 {
		normalizedQuantity = 1
	}

	unitPrice := product.UnitPrice + product.VariantExtraAmount
	if unitPrice < 0 {
		unitPrice = 0
	}

	totalAmount = (unitPrice * float64(normalizedQuantity)) - product.DiscountAmount + product.TaxAmount
	if totalAmount < 0 {
		totalAmount = 0
	}

	return totalAmount
}

func (service *SubscriptionDocumentService) drawProductsTable(pdf *gofpdf.Fpdf, subscription models.Subscription) (float64, float64) {
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(42, 51, 84)
	pdf.SetDrawColor(228, 231, 240)

	grandTotal := 0.0
	for _, product := range subscription.Products {
		if pdf.GetY() > 262 {
			pdf.AddPage()
			service.drawBrandHeader(pdf)
			pdf.SetXY(12, 42)
			service.drawProductTableHeader(pdf)
		}

		lineHeight := 7.5
		productName := product.ProductName
		if strings.TrimSpace(productName) == "" {
			productName = "Product"
		}
		if len(productName) > 44 {
			productName = productName[:41] + "..."
		}

		normalizedQuantity := product.Quantity
		if normalizedQuantity < 1 {
			normalizedQuantity = 1
		}

		unitPrice := product.UnitPrice + product.VariantExtraAmount
		if unitPrice < 0 {
			unitPrice = 0
		}

		lineTotal := normalizeProductLineTotal(product)

		pdf.SetX(12)
		pdf.CellFormat(66, lineHeight, productName, "1", 0, "L", false, 0, "")
		pdf.CellFormat(14, lineHeight, fmt.Sprintf("%d", normalizedQuantity), "1", 0, "C", false, 0, "")
		pdf.CellFormat(25, lineHeight, formatDocumentAmountUSD(unitPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(25, lineHeight, formatDocumentAmountUSD(product.DiscountAmount), "1", 0, "R", false, 0, "")
		pdf.CellFormat(22, lineHeight, formatDocumentAmountUSD(product.TaxAmount), "1", 0, "R", false, 0, "")
		pdf.CellFormat(28, lineHeight, formatDocumentAmountUSD(lineTotal), "1", 0, "R", false, 0, "")
		pdf.Ln(-1)

		grandTotal += lineTotal
	}

	if len(subscription.Products) == 0 {
		pdf.SetX(12)
		pdf.CellFormat(180, 8, "No products found for this subscription.", "1", 0, "L", false, 0, "")
		pdf.Ln(-1)
	}

	return pdf.GetY(), grandTotal
}

func (service *SubscriptionDocumentService) drawSubscriptionSummary(pdf *gofpdf.Fpdf, title string, subscription models.Subscription) {
	recurringText := "-"
	if subscription.Recurring != nil && strings.TrimSpace(*subscription.Recurring) != "" {
		recurringText = strings.TrimSpace(*subscription.Recurring)
	}

	planText := "-"
	if subscription.Plan != nil && strings.TrimSpace(*subscription.Plan) != "" {
		planText = strings.TrimSpace(*subscription.Plan)
	}

	paymentTermText := "-"
	if subscription.PaymentTermName != nil && strings.TrimSpace(*subscription.PaymentTermName) != "" {
		paymentTermText = strings.TrimSpace(*subscription.PaymentTermName)
	}

	pdf.SetTextColor(30, 44, 120)
	pdf.SetFont("Arial", "B", 16)
	pdf.SetXY(12, 42)
	pdf.CellFormat(120, 8, title, "", 0, "L", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(60, 70, 100)

	infoY := 51.0
	pdf.SetXY(12, infoY)
	pdf.CellFormat(120, 6, "Subscription Number: "+strings.TrimSpace(subscription.SubscriptionNumber), "", 0, "L", false, 0, "")
	infoY += 6
	pdf.SetXY(12, infoY)
	pdf.CellFormat(120, 6, "Customer: "+strings.TrimSpace(subscription.CustomerName), "", 0, "L", false, 0, "")
	infoY += 6
	pdf.SetXY(12, infoY)
	pdf.CellFormat(120, 6, "Status: "+string(subscription.Status), "", 0, "L", false, 0, "")
	infoY += 6
	pdf.SetXY(12, infoY)
	pdf.CellFormat(120, 6, "Next Invoice Date: "+formatDocumentDate(subscription.NextInvoiceDate), "", 0, "L", false, 0, "")
	infoY += 6
	pdf.SetXY(12, infoY)
	pdf.CellFormat(120, 6, "Recurring Plan: "+planText, "", 0, "L", false, 0, "")
	infoY += 6
	pdf.SetXY(12, infoY)
	pdf.CellFormat(120, 6, "Billing Period: "+recurringText, "", 0, "L", false, 0, "")
	infoY += 6
	pdf.SetXY(12, infoY)
	pdf.CellFormat(120, 6, "Payment Term: "+paymentTermText, "", 0, "L", false, 0, "")
	infoY += 8

	pdf.SetXY(12, infoY)
	service.drawProductTableHeader(pdf)
}

func (service *SubscriptionDocumentService) GenerateQuotationPDF(subscription models.Subscription) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 16)
	pdf.AddPage()

	service.drawBrandHeader(pdf)
	service.drawSubscriptionSummary(pdf, "Quotation", subscription)

	finalY, grandTotal := service.drawProductsTable(pdf, subscription)

	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(30, 44, 120)
	pdf.SetX(12)
	pdf.CellFormat(152, 9, "Grand Total", "1", 0, "R", false, 0, "")
	pdf.CellFormat(28, 9, formatDocumentAmountUSD(grandTotal), "1", 0, "R", false, 0, "")
	pdf.Ln(12)

	if finalY > 270 {
		pdf.AddPage()
		service.drawBrandHeader(pdf)
		pdf.SetXY(12, 42)
	}

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(85, 94, 125)
	pdf.MultiCell(186, 5.3, "This quotation is generated by RecurIN for your review.", "", "L", false)

	var document bytes.Buffer
	if err := pdf.Output(&document); err != nil {
		return nil, fmt.Errorf("failed to generate quotation pdf: %w", err)
	}

	return document.Bytes(), nil
}

func (service *SubscriptionDocumentService) GenerateInvoicePDF(subscription models.Subscription) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 16)
	pdf.AddPage()

	service.drawBrandHeader(pdf)
	service.drawSubscriptionSummary(pdf, "Invoice", subscription)

	finalY, grandTotal := service.drawProductsTable(pdf, subscription)

	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(30, 44, 120)
	pdf.SetX(12)
	pdf.CellFormat(152, 9, "Grand Total", "1", 0, "R", false, 0, "")
	pdf.CellFormat(28, 9, formatDocumentAmountUSD(grandTotal), "1", 0, "R", false, 0, "")
	pdf.Ln(10)

	paymentMethod := "PayPal"
	paymentStatus := "-"
	paymentDate := "-"
	amountPaid := grandTotal

	if subscription.Payment != nil {
		if strings.TrimSpace(subscription.Payment.PaymentMethod) != "" {
			paymentMethod = strings.TrimSpace(subscription.Payment.PaymentMethod)
		}
		if strings.TrimSpace(subscription.Payment.PayPalStatus) != "" {
			paymentStatus = strings.TrimSpace(subscription.Payment.PayPalStatus)
		}
		if subscription.Payment.AmountINR > 0 {
			amountPaid = subscription.Payment.AmountINR
		}
		paymentDate = formatDocumentDate(subscription.Payment.PaymentDate)
	}

	if finalY > 250 {
		pdf.AddPage()
		service.drawBrandHeader(pdf)
		pdf.SetXY(12, 42)
	}

	pdf.SetFont("Arial", "B", 11)
	pdf.SetTextColor(30, 44, 120)
	pdf.SetX(12)
	pdf.CellFormat(120, 7, "Payment Details", "", 0, "L", false, 0, "")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(60, 70, 100)
	pdf.SetX(12)
	pdf.CellFormat(180, 5.5, "Amount Paid: "+formatDocumentAmountUSD(amountPaid), "", 0, "L", false, 0, "")
	pdf.Ln(5.5)
	pdf.SetX(12)
	pdf.CellFormat(180, 5.5, "Payment Status: "+paymentStatus, "", 0, "L", false, 0, "")
	pdf.Ln(5.5)
	pdf.SetX(12)
	pdf.CellFormat(180, 5.5, "Payment Method: "+paymentMethod, "", 0, "L", false, 0, "")
	pdf.Ln(5.5)
	pdf.SetX(12)
	pdf.CellFormat(180, 5.5, "Payment Date: "+paymentDate, "", 0, "L", false, 0, "")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(85, 94, 125)
	pdf.MultiCell(186, 5.3, "Thank you for choosing RecurIN. This invoice is auto-generated by the backend service.", "", "L", false)

	var document bytes.Buffer
	if err := pdf.Output(&document); err != nil {
		return nil, fmt.Errorf("failed to generate invoice pdf: %w", err)
	}

	return document.Bytes(), nil
}
