package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/models"
)

type SubscriptionQuoteNotifierConfig struct {
	SMTPHost        string
	SMTPPort        int
	SMTPUsername    string
	SMTPPassword    string
	SMTPFromEmail   string
	SMTPFromName    string
	FrontendBaseURL string
	PDFLogoPath     string
}

type SubscriptionQuoteNotifier struct {
	smtpHost        string
	smtpPort        int
	smtpUsername    string
	smtpPassword    string
	smtpFromEmail   string
	smtpFromName    string
	frontendBaseURL string
	documentService *SubscriptionDocumentService
	enabled         bool
}

func NewSubscriptionQuoteNotifier(config SubscriptionQuoteNotifierConfig) *SubscriptionQuoteNotifier {
	notifier := &SubscriptionQuoteNotifier{
		smtpHost:        strings.TrimSpace(config.SMTPHost),
		smtpPort:        config.SMTPPort,
		smtpUsername:    strings.TrimSpace(config.SMTPUsername),
		smtpPassword:    config.SMTPPassword,
		smtpFromEmail:   strings.TrimSpace(config.SMTPFromEmail),
		smtpFromName:    strings.TrimSpace(config.SMTPFromName),
		frontendBaseURL: strings.TrimSpace(config.FrontendBaseURL),
		documentService: NewSubscriptionDocumentService(config.PDFLogoPath),
	}

	if notifier.smtpFromName == "" {
		notifier.smtpFromName = "RecurIN Subscriptions"
	}

	if notifier.smtpUsername == "" {
		notifier.smtpUsername = notifier.smtpFromEmail
	}

	notifier.enabled = notifier.smtpHost != "" && notifier.smtpPort > 0 && notifier.smtpFromEmail != ""
	return notifier
}

func (notifier *SubscriptionQuoteNotifier) IsEnabled() bool {
	return notifier != nil && notifier.enabled
}

func formatAmountINR(value float64) string {
	return fmt.Sprintf("INR %.2f", value)
}

func wrapBase64Content(content string, lineSize int) string {
	if lineSize <= 0 {
		lineSize = 76
	}

	var builder strings.Builder
	for start := 0; start < len(content); start += lineSize {
		end := start + lineSize
		if end > len(content) {
			end = len(content)
		}
		builder.WriteString(content[start:end])
		builder.WriteString("\r\n")
	}

	return builder.String()
}

func drawSubscriptionBrandHeader(pdf *gofpdf.Fpdf) {
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

	pdf.SetDrawColor(220, 223, 233)
	pdf.Line(12, 29, 198, 29)
}

func drawSubscriptionSummary(pdf *gofpdf.Fpdf, subscription models.Subscription, recipientName string) {
	pdf.SetTextColor(30, 44, 120)
	pdf.SetFont("Arial", "B", 15)
	pdf.SetXY(12, 34)
	pdf.CellFormat(120, 8, "Quotation", "", 0, "L", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.SetXY(12, 42)
	pdf.CellFormat(120, 6, "Subscription Number: "+subscription.SubscriptionNumber, "", 0, "L", false, 0, "")

	pdf.SetXY(12, 48)
	pdf.CellFormat(120, 6, "Customer: "+recipientName, "", 0, "L", false, 0, "")

	pdf.SetXY(12, 54)
	pdf.CellFormat(120, 6, "Status: "+string(subscription.Status), "", 0, "L", false, 0, "")

	pdf.SetXY(12, 60)
	pdf.CellFormat(120, 6, "Next Invoice Date: "+subscription.NextInvoiceDate.Format("2006-01-02"), "", 0, "L", false, 0, "")

	recurringText := "N/A"
	if subscription.Recurring != nil {
		recurringText = *subscription.Recurring
	}
	planText := "N/A"
	if subscription.Plan != nil {
		planText = *subscription.Plan
	}

	pdf.SetXY(12, 66)
	pdf.CellFormat(120, 6, "Recurring: "+recurringText+" - "+planText, "", 0, "L", false, 0, "")
}

func drawProductTableHeader(pdf *gofpdf.Fpdf, startY float64) {
	headers := []string{"Product", "Qty", "Unit", "Discount", "Tax", "Total"}
	widths := []float64{66, 14, 25, 25, 22, 28}

	pdf.SetXY(12, startY)
	pdf.SetFillColor(243, 245, 251)
	pdf.SetDrawColor(220, 223, 233)
	pdf.SetTextColor(30, 44, 120)
	pdf.SetFont("Arial", "B", 9)

	for index, header := range headers {
		pdf.CellFormat(widths[index], 8, header, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
}

func buildQuotationPDF(subscription models.Subscription, recipientName string) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 16)
	pdf.AddPage()

	drawSubscriptionBrandHeader(pdf)
	drawSubscriptionSummary(pdf, subscription, recipientName)
	drawProductTableHeader(pdf, 76)

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(42, 51, 84)
	pdf.SetDrawColor(228, 231, 240)

	totalAmount := 0.0
	for _, product := range subscription.Products {
		if pdf.GetY() > 262 {
			pdf.AddPage()
			drawProductTableHeader(pdf, 16)
		}

		lineHeight := 7.5
		productName := product.ProductName
		if len(productName) > 44 {
			productName = productName[:41] + "..."
		}

		pdf.SetX(12)
		pdf.CellFormat(66, lineHeight, productName, "1", 0, "L", false, 0, "")
		pdf.CellFormat(14, lineHeight, fmt.Sprintf("%d", product.Quantity), "1", 0, "C", false, 0, "")
		pdf.CellFormat(25, lineHeight, formatAmountINR(product.UnitPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(25, lineHeight, formatAmountINR(product.DiscountAmount), "1", 0, "R", false, 0, "")
		pdf.CellFormat(22, lineHeight, formatAmountINR(product.TaxAmount), "1", 0, "R", false, 0, "")
		pdf.CellFormat(28, lineHeight, formatAmountINR(product.TotalAmount), "1", 0, "R", false, 0, "")
		pdf.Ln(-1)

		totalAmount += product.TotalAmount
	}

	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(30, 44, 120)
	pdf.SetX(12)
	pdf.CellFormat(152, 9, "Grand Total", "1", 0, "R", false, 0, "")
	pdf.CellFormat(28, 9, formatAmountINR(totalAmount), "1", 0, "R", false, 0, "")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(85, 94, 125)
	pdf.MultiCell(186, 5.3, "Thank you for choosing RecurIN. This quotation includes subscription details prepared for your review.", "", "L", false)

	var document bytes.Buffer
	if err := pdf.Output(&document); err != nil {
		return nil, fmt.Errorf("failed to generate quotation pdf: %w", err)
	}

	return document.Bytes(), nil
}

func buildQuotationEmailHTMLBody(subscription models.Subscription, recipientName string, frontendBaseURL string) string {
	viewLink := ""
	if frontendBaseURL != "" {
		viewLink = strings.TrimRight(frontendBaseURL, "/") + "/admin/subscriptions"
	}

	var builder strings.Builder
	builder.WriteString("<html><body style=\"font-family:Arial,sans-serif;color:#1E2C78;line-height:1.5;\">")
	builder.WriteString("<h2 style=\"margin-bottom:6px;\">Quotation Sent</h2>")
	builder.WriteString("<p>Hello " + recipientName + ",</p>")
	builder.WriteString("<p>Your subscription quotation is ready. Please find the attached PDF containing complete subscription and quotation details.</p>")
	builder.WriteString("<p><strong>Subscription Number:</strong> " + subscription.SubscriptionNumber + "<br/>")
	builder.WriteString("<strong>Status:</strong> " + string(subscription.Status) + "<br/>")
	builder.WriteString("<strong>Next Invoice Date:</strong> " + subscription.NextInvoiceDate.Format("2006-01-02") + "</p>")
	if viewLink != "" {
		builder.WriteString("<p>You can also review subscriptions in your dashboard: <a href=\"" + viewLink + "\">Open Dashboard</a></p>")
	}
	builder.WriteString("<p>Regards,<br/>RecurIN Subscription Management</p>")
	builder.WriteString("</body></html>")
	return builder.String()
}

func (notifier *SubscriptionQuoteNotifier) SendQuotationEmail(ctx context.Context, recipientEmail string, recipientName string, subscription models.Subscription) error {
	if !notifier.IsEnabled() {
		return nil
	}

	normalizedRecipientEmail := strings.TrimSpace(recipientEmail)
	if normalizedRecipientEmail == "" {
		return ValidationError{Message: "recipient email is required for quotation notification"}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	var pdfContent []byte
	var err error
	if notifier.documentService != nil {
		pdfContent, err = notifier.documentService.GenerateQuotationPDF(subscription)
	} else {
		pdfContent, err = buildQuotationPDF(subscription, recipientName)
	}
	if err != nil {
		return err
	}

	subject := "Quotation Sent - " + subscription.SubscriptionNumber
	htmlBody := buildQuotationEmailHTMLBody(subscription, recipientName, notifier.frontendBaseURL)
	attachmentName := "Quotation_" + subscription.SubscriptionNumber + ".pdf"
	boundary := fmt.Sprintf("recurin-mixed-%d", time.Now().UnixNano())

	encodedPDF := base64.StdEncoding.EncodeToString(pdfContent)

	var messageBuilder strings.Builder
	messageBuilder.WriteString("From: " + notifier.smtpFromName + " <" + notifier.smtpFromEmail + ">\r\n")
	messageBuilder.WriteString("To: <" + normalizedRecipientEmail + ">\r\n")
	messageBuilder.WriteString("Subject: " + subject + "\r\n")
	messageBuilder.WriteString("MIME-Version: 1.0\r\n")
	messageBuilder.WriteString("Content-Type: multipart/mixed; boundary=\"" + boundary + "\"\r\n\r\n")

	messageBuilder.WriteString("--" + boundary + "\r\n")
	messageBuilder.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	messageBuilder.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
	messageBuilder.WriteString(htmlBody + "\r\n\r\n")

	messageBuilder.WriteString("--" + boundary + "\r\n")
	messageBuilder.WriteString("Content-Type: application/pdf; name=\"" + attachmentName + "\"\r\n")
	messageBuilder.WriteString("Content-Disposition: attachment; filename=\"" + attachmentName + "\"\r\n")
	messageBuilder.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	messageBuilder.WriteString(wrapBase64Content(encodedPDF, 76))
	messageBuilder.WriteString("\r\n--" + boundary + "--\r\n")

	smtpAddress := fmt.Sprintf("%s:%d", notifier.smtpHost, notifier.smtpPort)
	var smtpAuth smtp.Auth
	if notifier.smtpUsername != "" && notifier.smtpPassword != "" {
		smtpAuth = smtp.PlainAuth("", notifier.smtpUsername, notifier.smtpPassword, notifier.smtpHost)
	}

	if err := smtp.SendMail(smtpAddress, smtpAuth, notifier.smtpFromEmail, []string{normalizedRecipientEmail}, []byte(messageBuilder.String())); err != nil {
		return fmt.Errorf("failed to send quotation email: %w", err)
	}

	return nil
}
