package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"syslog/internal/domain"
	"syslog/internal/repository"
)

type fakeLogQueryRepository struct {
	lastParams repository.LogListParams
	result     repository.LogListResult
	err        error
}

func (r *fakeLogQueryRepository) ListPage(ctx context.Context, params repository.LogListParams) (repository.LogListResult, error) {
	r.lastParams = params
	return r.result, r.err
}

func TestLogsHandlerReturnsPaginatedResults(t *testing.T) {
	receivedAt := time.Date(2026, 3, 21, 3, 8, 0, 0, time.UTC)
	repo := &fakeLogQueryRepository{
		result: repository.LogListResult{
			Page:       2,
			PageSize:   10,
			TotalItems: 25,
			TotalPages: 3,
			Items: []repository.LogListItem{
				{
					Message: domain.SyslogMessage{
						ID:         88,
						ReceivedAt: receivedAt,
						RawMessage: "Station connected",
					},
					Event: &domain.ClientEvent{
						ID:         44,
						EventType:  "connect",
						StationMac: "aa:bb:cc:dd:ee:ff",
						Hostname:   "device-1",
					},
				},
			},
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/api/logs?page=2&query=device&fromDate=2026-03-20&toDate=2026-03-21", nil)
	recorder := httptest.NewRecorder()

	NewLogsHandler(repo).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	if repo.lastParams.Page != 2 {
		t.Fatalf("expected page 2, got %d", repo.lastParams.Page)
	}
	if repo.lastParams.PageSize != 10 {
		t.Fatalf("expected page size 10, got %d", repo.lastParams.PageSize)
	}
	if repo.lastParams.Query != "device" {
		t.Fatalf("expected query to be device, got %q", repo.lastParams.Query)
	}
	if repo.lastParams.FromDate != "2026-03-20" {
		t.Fatalf("expected fromDate to be set, got %q", repo.lastParams.FromDate)
	}
	if repo.lastParams.ToDate != "2026-03-21" {
		t.Fatalf("expected toDate to be set, got %q", repo.lastParams.ToDate)
	}
	if repo.lastParams.Scope != "matched" {
		t.Fatalf("expected default scope to be matched, got %q", repo.lastParams.Scope)
	}

	var response struct {
		Items []struct {
			Message domain.SyslogMessage `json:"message"`
			Event   *domain.ClientEvent  `json:"event"`
		} `json:"items"`
		Pagination struct {
			Page       int `json:"page"`
			PageSize   int `json:"pageSize"`
			TotalItems int `json:"totalItems"`
			TotalPages int `json:"totalPages"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(response.Items) != 1 || response.Items[0].Message.ID != 88 {
		t.Fatalf("unexpected items: %+v", response.Items)
	}
	if response.Pagination.Page != 2 || response.Pagination.PageSize != 10 {
		t.Fatalf("unexpected pagination: %+v", response.Pagination)
	}
	if response.Pagination.TotalItems != 25 || response.Pagination.TotalPages != 3 {
		t.Fatalf("unexpected totals: %+v", response.Pagination)
	}
}

func TestLogsHandlerDefaultsInvalidPageValues(t *testing.T) {
	repo := &fakeLogQueryRepository{
		result: repository.LogListResult{
			Page:       1,
			PageSize:   10,
			TotalItems: 0,
			TotalPages: 0,
			Items:      []repository.LogListItem{},
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/api/logs?page=0&query=%20%20scan%20%20", nil)
	recorder := httptest.NewRecorder()

	NewLogsHandler(repo).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if repo.lastParams.Page != 1 {
		t.Fatalf("expected invalid page to default to 1, got %d", repo.lastParams.Page)
	}
	if repo.lastParams.Query != "scan" {
		t.Fatalf("expected trimmed query, got %q", repo.lastParams.Query)
	}
}

func TestLogsHandlerAllowsAllScope(t *testing.T) {
	repo := &fakeLogQueryRepository{
		result: repository.LogListResult{
			Page:       1,
			PageSize:   10,
			TotalItems: 0,
			TotalPages: 0,
			Items:      []repository.LogListItem{},
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/api/logs?page=1&scope=all", nil)
	recorder := httptest.NewRecorder()

	NewLogsHandler(repo).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if repo.lastParams.Scope != "all" {
		t.Fatalf("expected scope all, got %q", repo.lastParams.Scope)
	}
}
