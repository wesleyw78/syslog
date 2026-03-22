package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"syslog/internal/repository"
)

type paginatedLogsResponse struct {
	Items      []logEntry         `json:"items"`
	Pagination paginationResponse `json:"pagination"`
}

type logEntry struct {
	Message any `json:"message"`
	Event   any `json:"event,omitempty"`
}

type paginationResponse struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	TotalItems int `json:"totalItems"`
	TotalPages int `json:"totalPages"`
}

func NewLogsHandler(logsRepo repository.LogQueryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if logsRepo == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		params := repository.LogListParams{
			Page:     parsePositiveInt(r.URL.Query().Get("page"), 1),
			PageSize: 10,
			Query:    strings.TrimSpace(r.URL.Query().Get("query")),
			Scope:    normalizeLogsScope(r.URL.Query().Get("scope")),
			FromDate: strings.TrimSpace(r.URL.Query().Get("fromDate")),
			ToDate:   strings.TrimSpace(r.URL.Query().Get("toDate")),
		}
		result, err := logsRepo.ListPage(r.Context(), params)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		items := make([]logEntry, 0, len(result.Items))
		for _, item := range result.Items {
			entry := logEntry{Message: item.Message}
			if item.Event != nil {
				entry.Event = item.Event
			}
			items = append(items, entry)
		}

		writeJSON(w, http.StatusOK, paginatedLogsResponse{
			Items: items,
			Pagination: paginationResponse{
				Page:       result.Page,
				PageSize:   result.PageSize,
				TotalItems: result.TotalItems,
				TotalPages: result.TotalPages,
			},
		})
	}
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

func normalizeLogsScope(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "all") {
		return "all"
	}
	return "matched"
}
