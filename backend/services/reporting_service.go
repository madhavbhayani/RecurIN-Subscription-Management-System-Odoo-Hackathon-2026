package services

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jung-kurt/gofpdf"
)

// ReportingSummary stores top-level reporting metrics.
type ReportingSummary struct {
	TotalUsers         int     `json:"total_users"`
	TotalProducts      int     `json:"total_products"`
	TotalSubscriptions int     `json:"total_subscriptions"`
	TotalPayments      int     `json:"total_payments"`
	TotalRevenueINR    float64 `json:"total_revenue_inr"`
}

// ReportingRevenuePoint stores monthly revenue values.
type ReportingRevenuePoint struct {
	PeriodKey   string  `json:"period_key"`
	PeriodLabel string  `json:"period_label"`
	RevenueINR  float64 `json:"revenue_inr"`
}

// ReportingPaymentStatus stores grouped payment status counts.
type ReportingPaymentStatus struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// ReportingModuleFrequencyPoint stores per-period registration counts for a module.
type ReportingModuleFrequencyPoint struct {
	PeriodKey     string `json:"period_key"`
	PeriodLabel   string `json:"period_label"`
	Registrations int    `json:"registrations"`
}

// ReportingModuleStatistic stores module-level totals and frequency.
type ReportingModuleStatistic struct {
	ModuleKey          string                          `json:"module_key"`
	ModuleLabel        string                          `json:"module_label"`
	TotalRegistrations int                             `json:"total_registrations"`
	Frequency          []ReportingModuleFrequencyPoint `json:"frequency"`
}

// AdminReportingDashboard stores response payload for reporting dashboard APIs.
type AdminReportingDashboard struct {
	GeneratedAt      string                     `json:"generated_at"`
	Months           int                        `json:"months"`
	Summary          ReportingSummary           `json:"summary"`
	RevenueTrend     []ReportingRevenuePoint    `json:"revenue_trend"`
	PaymentStatuses  []ReportingPaymentStatus   `json:"payment_statuses"`
	ModuleStatistics []ReportingModuleStatistic `json:"module_statistics"`
}

type reportingModuleSource struct {
	Key                   string
	Label                 string
	TotalCountQuery       string
	MonthlyFrequencyQuery string
}

var reportingModuleSources = []reportingModuleSource{
	{
		Key:             "users",
		Label:           "Users",
		TotalCountQuery: `SELECT COUNT(*)::bigint FROM users."user"`,
		MonthlyFrequencyQuery: `
			SELECT TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') AS month_key, COUNT(*)::bigint
			FROM users."user"
			WHERE created_at >= $1
			GROUP BY DATE_TRUNC('month', created_at)
			ORDER BY DATE_TRUNC('month', created_at)`,
	},
	{
		Key:             "roles",
		Label:           "Roles",
		TotalCountQuery: `SELECT COUNT(*)::bigint FROM privileges.role_data`,
		MonthlyFrequencyQuery: `
			SELECT TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') AS month_key, COUNT(*)::bigint
			FROM privileges.role_data
			WHERE created_at >= $1
			GROUP BY DATE_TRUNC('month', created_at)
			ORDER BY DATE_TRUNC('month', created_at)`,
	},
	{
		Key:             "products",
		Label:           "Products",
		TotalCountQuery: `SELECT COUNT(*)::bigint FROM products.product_data`,
		MonthlyFrequencyQuery: `
			SELECT TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') AS month_key, COUNT(*)::bigint
			FROM products.product_data
			WHERE created_at >= $1
			GROUP BY DATE_TRUNC('month', created_at)
			ORDER BY DATE_TRUNC('month', created_at)`,
	},
	{
		Key:             "subscriptions",
		Label:           "Subscriptions",
		TotalCountQuery: `SELECT COUNT(*)::bigint FROM subscription.subscriptions`,
		MonthlyFrequencyQuery: `
			SELECT TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') AS month_key, COUNT(*)::bigint
			FROM subscription.subscriptions
			WHERE created_at >= $1
			GROUP BY DATE_TRUNC('month', created_at)
			ORDER BY DATE_TRUNC('month', created_at)`,
	},
	{
		Key:             "quotations",
		Label:           "Quotation Templates",
		TotalCountQuery: `SELECT COUNT(*)::bigint FROM quotations.quotation`,
		MonthlyFrequencyQuery: `
			SELECT TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') AS month_key, COUNT(*)::bigint
			FROM quotations.quotation
			WHERE created_at >= $1
			GROUP BY DATE_TRUNC('month', created_at)
			ORDER BY DATE_TRUNC('month', created_at)`,
	},
	{
		Key:             "recurring-plans",
		Label:           "Recurring Plans",
		TotalCountQuery: `SELECT COUNT(*)::bigint FROM recurring_plans.recurring_plan_data`,
		MonthlyFrequencyQuery: `
			SELECT TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') AS month_key, COUNT(*)::bigint
			FROM recurring_plans.recurring_plan_data
			WHERE created_at >= $1
			GROUP BY DATE_TRUNC('month', created_at)
			ORDER BY DATE_TRUNC('month', created_at)`,
	},
	{
		Key:             "payment-terms",
		Label:           "Payment Terms",
		TotalCountQuery: `SELECT COUNT(*)::bigint FROM payment_term.payment_term_data`,
		MonthlyFrequencyQuery: `
			SELECT TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') AS month_key, COUNT(*)::bigint
			FROM payment_term.payment_term_data
			WHERE created_at >= $1
			GROUP BY DATE_TRUNC('month', created_at)
			ORDER BY DATE_TRUNC('month', created_at)`,
	},
	{
		Key:             "discounts",
		Label:           "Discounts",
		TotalCountQuery: `SELECT COUNT(*)::bigint FROM discount.discount_data`,
		MonthlyFrequencyQuery: `
			SELECT TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') AS month_key, COUNT(*)::bigint
			FROM discount.discount_data
			WHERE created_at >= $1
			GROUP BY DATE_TRUNC('month', created_at)
			ORDER BY DATE_TRUNC('month', created_at)`,
	},
	{
		Key:             "taxes",
		Label:           "Taxes",
		TotalCountQuery: `SELECT COUNT(*)::bigint FROM taxes.tax_data`,
		MonthlyFrequencyQuery: `
			SELECT TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') AS month_key, COUNT(*)::bigint
			FROM taxes.tax_data
			WHERE created_at >= $1
			GROUP BY DATE_TRUNC('month', created_at)
			ORDER BY DATE_TRUNC('month', created_at)`,
	},
	{
		Key:             "attributes",
		Label:           "Attributes",
		TotalCountQuery: `SELECT COUNT(*)::bigint FROM attributes.attribute`,
		MonthlyFrequencyQuery: `
			SELECT TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') AS month_key, COUNT(*)::bigint
			FROM attributes.attribute
			WHERE created_at >= $1
			GROUP BY DATE_TRUNC('month', created_at)
			ORDER BY DATE_TRUNC('month', created_at)`,
	},
	{
		Key:             "payments",
		Label:           "Payments",
		TotalCountQuery: `SELECT COUNT(*)::bigint FROM users.payments`,
		MonthlyFrequencyQuery: `
			SELECT TO_CHAR(DATE_TRUNC('month', payment_date), 'YYYY-MM') AS month_key, COUNT(*)::bigint
			FROM users.payments
			WHERE payment_date >= $1
			GROUP BY DATE_TRUNC('month', payment_date)
			ORDER BY DATE_TRUNC('month', payment_date)`,
	},
}

// ReportingService encapsulates admin reporting and report-PDF generation.
type ReportingService struct {
	db       *pgxpool.Pool
	logoPath string
}

func NewReportingService(db *pgxpool.Pool, configuredLogoPath string) *ReportingService {
	return &ReportingService{
		db:       db,
		logoPath: resolveSubscriptionLogoPath(configuredLogoPath),
	}
}

func normalizeReportingMonths(months int) int {
	switch {
	case months <= 0:
		return 6
	case months > 24:
		return 24
	default:
		return months
	}
}

func buildReportingMonthBuckets(months int, referenceTime time.Time) []time.Time {
	normalizedReference := referenceTime.UTC()
	currentMonth := time.Date(normalizedReference.Year(), normalizedReference.Month(), 1, 0, 0, 0, 0, time.UTC)
	buckets := make([]time.Time, 0, months)
	for index := months - 1; index >= 0; index-- {
		buckets = append(buckets, currentMonth.AddDate(0, -index, 0))
	}
	return buckets
}

func (service *ReportingService) queryCount(ctx context.Context, query string) (int, error) {
	var count int64
	if err := service.db.QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to query count: %w", err)
	}
	return int(count), nil
}

func (service *ReportingService) queryMonthlyCounts(ctx context.Context, query string, startDate time.Time) (map[string]int64, error) {
	rows, err := service.db.Query(ctx, query, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query monthly counts: %w", err)
	}
	defer rows.Close()

	valueByMonth := make(map[string]int64)
	for rows.Next() {
		var monthKey string
		var count int64
		if scanErr := rows.Scan(&monthKey, &count); scanErr != nil {
			return nil, fmt.Errorf("failed to scan monthly count row: %w", scanErr)
		}
		valueByMonth[strings.TrimSpace(monthKey)] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating monthly count rows: %w", err)
	}

	return valueByMonth, nil
}

func (service *ReportingService) fetchSummary(ctx context.Context) (ReportingSummary, error) {
	const summaryQuery = `
		SELECT
			(SELECT COUNT(*)::bigint FROM users."user") AS total_users,
			(SELECT COUNT(*)::bigint FROM products.product_data) AS total_products,
			(SELECT COUNT(*)::bigint FROM subscription.subscriptions) AS total_subscriptions,
			(SELECT COUNT(*)::bigint FROM users.payments) AS total_payments,
			(
				SELECT COALESCE(SUM(sp.total_amount), 0)::float8
				FROM subscription.subscriptions s
				JOIN subscription.subscription_products sp ON sp.subscription_id = s.subscription_id
				WHERE s.status IN ('Active', 'Confirmed')
			) AS total_revenue_inr`

	var summary ReportingSummary
	var totalUsers int64
	var totalProducts int64
	var totalSubscriptions int64
	var totalPayments int64
	if err := service.db.QueryRow(ctx, summaryQuery).Scan(
		&totalUsers,
		&totalProducts,
		&totalSubscriptions,
		&totalPayments,
		&summary.TotalRevenueINR,
	); err != nil {
		return ReportingSummary{}, fmt.Errorf("failed to fetch reporting summary: %w", err)
	}

	summary.TotalUsers = int(totalUsers)
	summary.TotalProducts = int(totalProducts)
	summary.TotalSubscriptions = int(totalSubscriptions)
	summary.TotalPayments = int(totalPayments)
	return summary, nil
}

func (service *ReportingService) fetchRevenueTrend(ctx context.Context, startDate time.Time, monthBuckets []time.Time) ([]ReportingRevenuePoint, error) {
	const revenueQuery = `
		SELECT
			TO_CHAR(DATE_TRUNC('month', s.created_at), 'YYYY-MM') AS month_key,
			COALESCE(SUM(sp.total_amount), 0)::float8 AS total_revenue
		FROM subscription.subscriptions s
		LEFT JOIN subscription.subscription_products sp ON sp.subscription_id = s.subscription_id
		WHERE s.created_at >= $1
		  AND s.status IN ('Active', 'Confirmed')
		GROUP BY DATE_TRUNC('month', s.created_at)
		ORDER BY DATE_TRUNC('month', s.created_at)`

	rows, err := service.db.Query(ctx, revenueQuery, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch revenue trend: %w", err)
	}
	defer rows.Close()

	revenueByMonth := make(map[string]float64)
	for rows.Next() {
		var monthKey string
		var revenueValue float64
		if scanErr := rows.Scan(&monthKey, &revenueValue); scanErr != nil {
			return nil, fmt.Errorf("failed to scan revenue trend row: %w", scanErr)
		}
		revenueByMonth[strings.TrimSpace(monthKey)] = revenueValue
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating revenue trend rows: %w", err)
	}

	trend := make([]ReportingRevenuePoint, 0, len(monthBuckets))
	for _, monthBucket := range monthBuckets {
		monthKey := monthBucket.Format("2006-01")
		trend = append(trend, ReportingRevenuePoint{
			PeriodKey:   monthKey,
			PeriodLabel: monthBucket.Format("Jan 2006"),
			RevenueINR:  revenueByMonth[monthKey],
		})
	}

	return trend, nil
}

func (service *ReportingService) fetchPaymentStatuses(ctx context.Context) ([]ReportingPaymentStatus, error) {
	const paymentStatusQuery = `
		SELECT
			COALESCE(NULLIF(BTRIM(paypal_status), ''), 'Unknown') AS payment_status,
			COUNT(*)::bigint AS status_count
		FROM users.payments
		GROUP BY 1
		ORDER BY 2 DESC, 1 ASC`

	rows, err := service.db.Query(ctx, paymentStatusQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payment statuses: %w", err)
	}
	defer rows.Close()

	statuses := make([]ReportingPaymentStatus, 0)
	for rows.Next() {
		var status string
		var count int64
		if scanErr := rows.Scan(&status, &count); scanErr != nil {
			return nil, fmt.Errorf("failed to scan payment status row: %w", scanErr)
		}
		statuses = append(statuses, ReportingPaymentStatus{
			Status: status,
			Count:  int(count),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating payment status rows: %w", err)
	}

	return statuses, nil
}

func (service *ReportingService) fetchModuleStatistics(ctx context.Context, startDate time.Time, monthBuckets []time.Time) ([]ReportingModuleStatistic, error) {
	statistics := make([]ReportingModuleStatistic, 0, len(reportingModuleSources))

	for _, moduleSource := range reportingModuleSources {
		totalRegistrations, err := service.queryCount(ctx, moduleSource.TotalCountQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch total registrations for %s: %w", moduleSource.Key, err)
		}

		monthlyCounts, err := service.queryMonthlyCounts(ctx, moduleSource.MonthlyFrequencyQuery, startDate)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch monthly registrations for %s: %w", moduleSource.Key, err)
		}

		frequencyPoints := make([]ReportingModuleFrequencyPoint, 0, len(monthBuckets))
		for _, monthBucket := range monthBuckets {
			monthKey := monthBucket.Format("2006-01")
			frequencyPoints = append(frequencyPoints, ReportingModuleFrequencyPoint{
				PeriodKey:     monthKey,
				PeriodLabel:   monthBucket.Format("Jan 2006"),
				Registrations: int(monthlyCounts[monthKey]),
			})
		}

		statistics = append(statistics, ReportingModuleStatistic{
			ModuleKey:          moduleSource.Key,
			ModuleLabel:        moduleSource.Label,
			TotalRegistrations: totalRegistrations,
			Frequency:          frequencyPoints,
		})
	}

	sort.SliceStable(statistics, func(left int, right int) bool {
		if statistics[left].TotalRegistrations == statistics[right].TotalRegistrations {
			return statistics[left].ModuleLabel < statistics[right].ModuleLabel
		}
		return statistics[left].TotalRegistrations > statistics[right].TotalRegistrations
	})

	return statistics, nil
}

func (service *ReportingService) GetDashboard(ctx context.Context, months int) (AdminReportingDashboard, error) {
	normalizedMonths := normalizeReportingMonths(months)
	monthBuckets := buildReportingMonthBuckets(normalizedMonths, time.Now().UTC())
	startDate := monthBuckets[0]

	summary, err := service.fetchSummary(ctx)
	if err != nil {
		return AdminReportingDashboard{}, err
	}

	revenueTrend, err := service.fetchRevenueTrend(ctx, startDate, monthBuckets)
	if err != nil {
		return AdminReportingDashboard{}, err
	}

	paymentStatuses, err := service.fetchPaymentStatuses(ctx)
	if err != nil {
		return AdminReportingDashboard{}, err
	}

	moduleStatistics, err := service.fetchModuleStatistics(ctx, startDate, monthBuckets)
	if err != nil {
		return AdminReportingDashboard{}, err
	}

	return AdminReportingDashboard{
		GeneratedAt:      time.Now().UTC().Format("2006-01-02 15:04 UTC"),
		Months:           normalizedMonths,
		Summary:          summary,
		RevenueTrend:     revenueTrend,
		PaymentStatuses:  paymentStatuses,
		ModuleStatistics: moduleStatistics,
	}, nil
}

func formatReportingAmountUSD(value float64) string {
	return fmt.Sprintf("$%.2f", value)
}

func (service *ReportingService) drawReportHeader(pdf *gofpdf.Fpdf, title, generatedAt string) {
	if service.logoPath != "" {
		imageOptions := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
		pdf.ImageOptions(service.logoPath, 12, 8, 78, 22, false, imageOptions, 0, "")
	} else {
		pdf.SetTextColor(255, 107, 0)
		pdf.SetFont("Arial", "B", 18)
		pdf.SetXY(12, 16)
		pdf.CellFormat(80, 8, "RecurIN", "", 0, "L", false, 0, "")
	}

	if strings.TrimSpace(generatedAt) == "" {
		generatedAt = time.Now().UTC().Format("2006-01-02 15:04 UTC")
	}

	pdf.SetTextColor(30, 44, 120)
	pdf.SetFont("Arial", "B", 15)
	pdf.SetXY(12, 34)
	pdf.CellFormat(0, 8, title, "", 0, "L", false, 0, "")

	pdf.SetTextColor(85, 94, 125)
	pdf.SetFont("Arial", "", 9)
	pdf.SetXY(12, 41)
	pdf.CellFormat(0, 5, "Generated at: "+generatedAt, "", 0, "L", false, 0, "")

	pageWidth, _ := pdf.GetPageSize()
	pdf.SetDrawColor(220, 223, 233)
	pdf.Line(12, 48, pageWidth-12, 48)
}

func shortReportMonthLabel(periodKey string) string {
	parsedMonth, err := time.Parse("2006-01", strings.TrimSpace(periodKey))
	if err != nil {
		return periodKey
	}
	return parsedMonth.Format("Jan-06")
}

func (service *ReportingService) GenerateSalesReportPDF(report AdminReportingDashboard) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(12, 10, 12)
	pdf.SetAutoPageBreak(true, 14)
	pdf.AddPage()

	service.drawReportHeader(pdf, "Sales & Operational Report", report.GeneratedAt)

	pdf.SetY(54)
	pdf.SetTextColor(30, 44, 120)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(186, 7, "Summary Metrics", "", 1, "L", false, 0, "")

	summaryRows := []struct {
		Label string
		Value string
	}{
		{Label: "Total Users", Value: fmt.Sprintf("%d", report.Summary.TotalUsers)},
		{Label: "Total Products", Value: fmt.Sprintf("%d", report.Summary.TotalProducts)},
		{Label: "Total Subscriptions", Value: fmt.Sprintf("%d", report.Summary.TotalSubscriptions)},
		{Label: "Total Payments", Value: fmt.Sprintf("%d", report.Summary.TotalPayments)},
		{Label: "Total Revenue", Value: formatReportingAmountUSD(report.Summary.TotalRevenueINR)},
	}

	pdf.SetFillColor(243, 245, 251)
	pdf.SetDrawColor(220, 223, 233)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(116, 8, "Metric", "1", 0, "L", true, 0, "")
	pdf.CellFormat(70, 8, "Value", "1", 1, "R", true, 0, "")

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(42, 51, 84)
	for _, summaryRow := range summaryRows {
		pdf.CellFormat(116, 7, summaryRow.Label, "1", 0, "L", false, 0, "")
		pdf.CellFormat(70, 7, summaryRow.Value, "1", 1, "R", false, 0, "")
	}

	pdf.Ln(3)
	pdf.SetTextColor(30, 44, 120)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(186, 7, "Revenue Trend", "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(243, 245, 251)
	pdf.CellFormat(96, 8, "Period", "1", 0, "L", true, 0, "")
	pdf.CellFormat(90, 8, "Revenue ($)", "1", 1, "R", true, 0, "")

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(42, 51, 84)
	for _, revenuePoint := range report.RevenueTrend {
		if pdf.GetY() > 272 {
			pdf.AddPage()
			service.drawReportHeader(pdf, "Sales & Operational Report", report.GeneratedAt)
			pdf.SetY(54)
			pdf.SetFont("Arial", "B", 9)
			pdf.SetFillColor(243, 245, 251)
			pdf.CellFormat(96, 8, "Period", "1", 0, "L", true, 0, "")
			pdf.CellFormat(90, 8, "Revenue ($)", "1", 1, "R", true, 0, "")
			pdf.SetFont("Arial", "", 9)
			pdf.SetTextColor(42, 51, 84)
		}

		pdf.CellFormat(96, 7, revenuePoint.PeriodLabel, "1", 0, "L", false, 0, "")
		pdf.CellFormat(90, 7, formatReportingAmountUSD(revenuePoint.RevenueINR), "1", 1, "R", false, 0, "")
	}

	pdf.Ln(3)
	pdf.SetTextColor(30, 44, 120)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(186, 7, "Payment Status Overview", "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(243, 245, 251)
	pdf.CellFormat(116, 8, "Status", "1", 0, "L", true, 0, "")
	pdf.CellFormat(70, 8, "Count", "1", 1, "R", true, 0, "")

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(42, 51, 84)
	if len(report.PaymentStatuses) == 0 {
		pdf.CellFormat(186, 7, "No payment data available.", "1", 1, "L", false, 0, "")
	} else {
		for _, paymentStatus := range report.PaymentStatuses {
			if pdf.GetY() > 272 {
				pdf.AddPage()
				service.drawReportHeader(pdf, "Sales & Operational Report", report.GeneratedAt)
				pdf.SetY(54)
				pdf.SetFont("Arial", "B", 9)
				pdf.SetFillColor(243, 245, 251)
				pdf.CellFormat(116, 8, "Status", "1", 0, "L", true, 0, "")
				pdf.CellFormat(70, 8, "Count", "1", 1, "R", true, 0, "")
				pdf.SetFont("Arial", "", 9)
				pdf.SetTextColor(42, 51, 84)
			}

			pdf.CellFormat(116, 7, paymentStatus.Status, "1", 0, "L", false, 0, "")
			pdf.CellFormat(70, 7, fmt.Sprintf("%d", paymentStatus.Count), "1", 1, "R", false, 0, "")
		}
	}

	pdf.Ln(3)
	pdf.SetTextColor(30, 44, 120)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(186, 7, "Top Modules by Registrations", "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(243, 245, 251)
	pdf.CellFormat(116, 8, "Module", "1", 0, "L", true, 0, "")
	pdf.CellFormat(70, 8, "Registrations", "1", 1, "R", true, 0, "")

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(42, 51, 84)
	if len(report.ModuleStatistics) == 0 {
		pdf.CellFormat(186, 7, "No module statistics available.", "1", 1, "L", false, 0, "")
	} else {
		visibleRows := report.ModuleStatistics
		if len(visibleRows) > 8 {
			visibleRows = visibleRows[:8]
		}

		for _, moduleStatistic := range visibleRows {
			if pdf.GetY() > 272 {
				pdf.AddPage()
				service.drawReportHeader(pdf, "Sales & Operational Report", report.GeneratedAt)
				pdf.SetY(54)
				pdf.SetFont("Arial", "B", 9)
				pdf.SetFillColor(243, 245, 251)
				pdf.CellFormat(116, 8, "Module", "1", 0, "L", true, 0, "")
				pdf.CellFormat(70, 8, "Registrations", "1", 1, "R", true, 0, "")
				pdf.SetFont("Arial", "", 9)
				pdf.SetTextColor(42, 51, 84)
			}

			pdf.CellFormat(116, 7, moduleStatistic.ModuleLabel, "1", 0, "L", false, 0, "")
			pdf.CellFormat(70, 7, fmt.Sprintf("%d", moduleStatistic.TotalRegistrations), "1", 1, "R", false, 0, "")
		}
	}

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, fmt.Errorf("failed to generate sales report PDF: %w", err)
	}

	return output.Bytes(), nil
}

func (service *ReportingService) GenerateModuleFrequencyReportPDF(report AdminReportingDashboard) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(12, 10, 12)
	pdf.SetAutoPageBreak(true, 14)
	pdf.AddPage()

	service.drawReportHeader(pdf, "Module Registration Frequency Report", report.GeneratedAt)

	moduleStatistics := report.ModuleStatistics
	if len(moduleStatistics) == 0 {
		pdf.SetY(54)
		pdf.SetTextColor(42, 51, 84)
		pdf.SetFont("Arial", "", 10)
		pdf.MultiCell(0, 6, "No module statistics are available for the selected reporting window.", "", "L", false)

		var emptyOutput bytes.Buffer
		if err := pdf.Output(&emptyOutput); err != nil {
			return nil, fmt.Errorf("failed to generate module frequency PDF: %w", err)
		}
		return emptyOutput.Bytes(), nil
	}

	referenceFrequency := moduleStatistics[0].Frequency
	if len(referenceFrequency) > 12 {
		referenceFrequency = referenceFrequency[len(referenceFrequency)-12:]
	}

	if len(referenceFrequency) == 0 {
		normalizedMonths := normalizeReportingMonths(report.Months)
		monthBuckets := buildReportingMonthBuckets(normalizedMonths, time.Now().UTC())
		for _, monthBucket := range monthBuckets {
			referenceFrequency = append(referenceFrequency, ReportingModuleFrequencyPoint{
				PeriodKey:   monthBucket.Format("2006-01"),
				PeriodLabel: monthBucket.Format("Jan 2006"),
			})
		}
		if len(referenceFrequency) > 12 {
			referenceFrequency = referenceFrequency[len(referenceFrequency)-12:]
		}
	}

	periodKeys := make([]string, 0, len(referenceFrequency))
	for _, frequencyPoint := range referenceFrequency {
		periodKeys = append(periodKeys, frequencyPoint.PeriodKey)
	}

	pageWidth, _ := pdf.GetPageSize()
	usableWidth := pageWidth - 24
	moduleWidth := 58.0
	totalWidth := 20.0
	monthWidth := (usableWidth - moduleWidth - totalWidth) / float64(len(periodKeys))
	if monthWidth < 12 {
		monthWidth = 12
	}

	pdf.SetY(54)
	pdf.SetTextColor(30, 44, 120)
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(0, 7, "Frequency of Registrations by Module", "", 1, "L", false, 0, "")

	pdf.SetFillColor(243, 245, 251)
	pdf.SetDrawColor(220, 223, 233)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(moduleWidth, 8, "Module", "1", 0, "L", true, 0, "")
	pdf.CellFormat(totalWidth, 8, "Total", "1", 0, "R", true, 0, "")
	for _, periodKey := range periodKeys {
		pdf.CellFormat(monthWidth, 8, shortReportMonthLabel(periodKey), "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(42, 51, 84)
	for _, moduleStatistic := range moduleStatistics {
		if pdf.GetY() > 190 {
			pdf.AddPage()
			service.drawReportHeader(pdf, "Module Registration Frequency Report", report.GeneratedAt)
			pdf.SetY(54)
			pdf.SetFillColor(243, 245, 251)
			pdf.SetDrawColor(220, 223, 233)
			pdf.SetFont("Arial", "B", 8)
			pdf.CellFormat(moduleWidth, 8, "Module", "1", 0, "L", true, 0, "")
			pdf.CellFormat(totalWidth, 8, "Total", "1", 0, "R", true, 0, "")
			for _, periodKey := range periodKeys {
				pdf.CellFormat(monthWidth, 8, shortReportMonthLabel(periodKey), "1", 0, "C", true, 0, "")
			}
			pdf.Ln(-1)
			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(42, 51, 84)
		}

		countsByPeriod := make(map[string]int, len(moduleStatistic.Frequency))
		for _, frequencyPoint := range moduleStatistic.Frequency {
			countsByPeriod[frequencyPoint.PeriodKey] = frequencyPoint.Registrations
		}

		pdf.CellFormat(moduleWidth, 7, moduleStatistic.ModuleLabel, "1", 0, "L", false, 0, "")
		pdf.CellFormat(totalWidth, 7, fmt.Sprintf("%d", moduleStatistic.TotalRegistrations), "1", 0, "R", false, 0, "")
		for _, periodKey := range periodKeys {
			pdf.CellFormat(monthWidth, 7, fmt.Sprintf("%d", countsByPeriod[periodKey]), "1", 0, "C", false, 0, "")
		}
		pdf.Ln(-1)
	}

	pdf.Ln(3)
	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(85, 94, 125)
	pdf.MultiCell(0, 4.8, "This report captures registration frequency trends for core modules used in RecurIN operations.", "", "L", false)

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, fmt.Errorf("failed to generate module frequency report PDF: %w", err)
	}

	return output.Bytes(), nil
}
