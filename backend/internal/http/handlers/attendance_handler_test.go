package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httpapi "syslog/internal/http"
)

func TestListAttendanceReturnsOK(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{})

	req := httptest.NewRequest(http.MethodGet, "/api/attendance", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var payload struct {
		Items []any `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(payload.Items) != 0 {
		t.Fatalf("expected empty items, got %d", len(payload.Items))
	}
}

func TestAdminRoutesReturnOK(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{})

	paths := []string{
		"/api/attendance",
		"/api/employees",
		"/api/logs",
		"/api/settings",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
			}
		})
	}
}

func TestNewServerUsesAdminRouter(t *testing.T) {
	server := httpapi.NewServer(":0", httpapi.Dependencies{})

	if server.Addr != ":0" {
		t.Fatalf("expected addr :0, got %s", server.Addr)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/attendance", nil)
	resp := httptest.NewRecorder()

	server.Handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
}
