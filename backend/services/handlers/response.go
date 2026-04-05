package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

const adminListPageSize = 30

func writeJSON(writer http.ResponseWriter, statusCode int, payload interface{}) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}

func parsePageQuery(request *http.Request) (int, bool, error) {
	pageText := strings.TrimSpace(request.URL.Query().Get("page"))
	if pageText == "" {
		return 0, false, nil
	}

	page, err := strconv.Atoi(pageText)
	if err != nil || page < 1 {
		return 0, false, errors.New("page must be a valid integer greater than zero")
	}

	return page, true, nil
}

func buildPaginationResponse(page, pageSize, totalRecords int) map[string]interface{} {
	if pageSize <= 0 {
		pageSize = adminListPageSize
	}

	if totalRecords < 0 {
		totalRecords = 0
	}

	totalPages := 0
	if totalRecords > 0 {
		totalPages = (totalRecords + pageSize - 1) / pageSize
	}

	return map[string]interface{}{
		"page":          page,
		"per_page":      pageSize,
		"total_records": totalRecords,
		"total_pages":   totalPages,
	}
}
