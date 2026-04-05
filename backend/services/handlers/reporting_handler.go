package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/madhavbhayani/RecurIN-Subscription-Management-System-Odoo-Hackathon-2026.git/services"
)

// ReportingHandler handles admin reporting APIs and report downloads.
type ReportingHandler struct {
	reportingService *services.ReportingService
}

func NewReportingHandler(reportingService *services.ReportingService) *ReportingHandler {
	return &ReportingHandler{reportingService: reportingService}
}

func parseReportingMonths(request *http.Request) (int, error) {
	monthsText := strings.TrimSpace(request.URL.Query().Get("months"))
	if monthsText == "" {
		return 6, nil
	}

	months, err := strconv.Atoi(monthsText)
	if err != nil {
		return 0, services.ValidationError{Message: "months must be a valid integer"}
	}

	return months, nil
}

func writeReportPDFDownloadResponse(writer http.ResponseWriter, fileName string, payload []byte) {
	writer.Header().Set("Content-Type", "application/pdf")
	writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(payload)
}

func buildReportingFileName(prefix string) string {
	dateStamp := time.Now().UTC().Format("20060102")
	return fmt.Sprintf("%s-%s.pdf", prefix, dateStamp)
}

func (handler *ReportingHandler) writeReportingError(writer http.ResponseWriter, err error) {
	var validationError services.ValidationError
	if errors.As(err, &validationError) {
		http.Error(writer, validationError.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("reporting handler error: %v", err)
	http.Error(writer, "Request processing failed.", http.StatusInternalServerError)
}

func (handler *ReportingHandler) HandleGetDashboard(writer http.ResponseWriter, request *http.Request) {
	if handler.reportingService == nil {
		http.Error(writer, "Reporting service is not configured.", http.StatusInternalServerError)
		return
	}

	months, err := parseReportingMonths(request)
	if err != nil {
		handler.writeReportingError(writer, err)
		return
	}

	dashboard, err := handler.reportingService.GetDashboard(request.Context(), months)
	if err != nil {
		handler.writeReportingError(writer, err)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]interface{}{
		"dashboard": dashboard,
	})
}

func (handler *ReportingHandler) HandleDownloadSalesReportPDF(writer http.ResponseWriter, request *http.Request) {
	if handler.reportingService == nil {
		http.Error(writer, "Reporting service is not configured.", http.StatusInternalServerError)
		return
	}

	months, err := parseReportingMonths(request)
	if err != nil {
		handler.writeReportingError(writer, err)
		return
	}

	dashboard, err := handler.reportingService.GetDashboard(request.Context(), months)
	if err != nil {
		handler.writeReportingError(writer, err)
		return
	}

	pdfBytes, err := handler.reportingService.GenerateSalesReportPDF(dashboard)
	if err != nil {
		handler.writeReportingError(writer, err)
		return
	}

	writeReportPDFDownloadResponse(writer, buildReportingFileName("Sales-Report"), pdfBytes)
}

func (handler *ReportingHandler) HandleDownloadModuleFrequencyReportPDF(writer http.ResponseWriter, request *http.Request) {
	if handler.reportingService == nil {
		http.Error(writer, "Reporting service is not configured.", http.StatusInternalServerError)
		return
	}

	months, err := parseReportingMonths(request)
	if err != nil {
		handler.writeReportingError(writer, err)
		return
	}

	dashboard, err := handler.reportingService.GetDashboard(request.Context(), months)
	if err != nil {
		handler.writeReportingError(writer, err)
		return
	}

	pdfBytes, err := handler.reportingService.GenerateModuleFrequencyReportPDF(dashboard)
	if err != nil {
		handler.writeReportingError(writer, err)
		return
	}

	writeReportPDFDownloadResponse(writer, buildReportingFileName("Module-Frequency-Report"), pdfBytes)
}
