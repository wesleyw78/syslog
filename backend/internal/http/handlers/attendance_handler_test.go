package handlers_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"syslog/internal/domain"
	httpapi "syslog/internal/http"
	"syslog/internal/repository"
)

type fakeEmployeeRepo struct {
	employees []domain.Employee
}

func (f *fakeEmployeeRepo) FindByMACAddress(context.Context, string) (*domain.Employee, error) {
	return nil, nil
}

func (f *fakeEmployeeRepo) FindByID(context.Context, uint64) (*domain.Employee, error) {
	return nil, nil
}

func (f *fakeEmployeeRepo) List(context.Context) ([]domain.Employee, error) {
	return append([]domain.Employee(nil), f.employees...), nil
}

func (f *fakeEmployeeRepo) Create(context.Context, *domain.Employee) error {
	return nil
}

func (f *fakeEmployeeRepo) Update(context.Context, *domain.Employee) error {
	return nil
}

func (f *fakeEmployeeRepo) Disable(context.Context, uint64) error {
	return nil
}

func (f *fakeEmployeeRepo) ReplaceDevices(context.Context, uint64, []domain.EmployeeDevice) error {
	return nil
}

func (f *fakeEmployeeRepo) DisableDevicesByEmployeeID(context.Context, uint64) error {
	return nil
}

func (f *fakeEmployeeRepo) WithTx(*sql.Tx) repository.EmployeeRepository {
	return f
}

type fakeSyslogMessageRepo struct {
	messages []domain.SyslogMessage
}

func (f *fakeSyslogMessageRepo) Save(context.Context, *domain.SyslogMessage) error {
	return nil
}

func (f *fakeSyslogMessageRepo) ListRecent(context.Context, int) ([]domain.SyslogMessage, error) {
	return append([]domain.SyslogMessage(nil), f.messages...), nil
}

type fakeClientEventRepo struct {
	events []domain.ClientEvent
}

func (f *fakeClientEventRepo) Save(context.Context, *domain.ClientEvent) error {
	return nil
}

func (f *fakeClientEventRepo) ListRecent(context.Context, int) ([]domain.ClientEvent, error) {
	return append([]domain.ClientEvent(nil), f.events...), nil
}

type fakeLogQueryRepo struct {
	items []repository.LogListItem
}

func (f *fakeLogQueryRepo) ListPage(context.Context, repository.LogListParams) (repository.LogListResult, error) {
	return repository.LogListResult{
		Items:      append([]repository.LogListItem(nil), f.items...),
		Page:       1,
		PageSize:   10,
		TotalItems: len(f.items),
		TotalPages: 1,
	}, nil
}

type fakeAttendanceRepo struct {
	records []domain.AttendanceRecord
}

func (f *fakeAttendanceRepo) FindByID(context.Context, uint64) (*domain.AttendanceRecord, error) {
	return nil, nil
}

func (f *fakeAttendanceRepo) FindByEmployeeAndDate(context.Context, uint64, time.Time) (*domain.AttendanceRecord, error) {
	return nil, nil
}

func (f *fakeAttendanceRepo) Save(context.Context, *domain.AttendanceRecord) error {
	return nil
}

func (f *fakeAttendanceRepo) ListByDateRange(context.Context, time.Time, time.Time) ([]domain.AttendanceRecord, error) {
	return append([]domain.AttendanceRecord(nil), f.records...), nil
}

func (f *fakeAttendanceRepo) WithTx(*sql.Tx) repository.AttendanceRepository {
	return f
}

type fakeSystemSettingRepo struct {
	settings []domain.SystemSetting
}

func (f *fakeSystemSettingRepo) GetByKey(context.Context, string) (*domain.SystemSetting, error) {
	return nil, nil
}

func (f *fakeSystemSettingRepo) List(context.Context) ([]domain.SystemSetting, error) {
	return append([]domain.SystemSetting(nil), f.settings...), nil
}

func (f *fakeSystemSettingRepo) Save(context.Context, *domain.SystemSetting) error {
	return nil
}

func (f *fakeSystemSettingRepo) WithTx(*sql.Tx) repository.SystemSettingRepository {
	return f
}

func TestAdminRoutesReturnRealJSON(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Employees: &fakeEmployeeRepo{
			employees: []domain.Employee{
				{ID: 1, EmployeeNo: "EMP-001", Name: "Alice", Status: "active"},
			},
		},
		SyslogMessages: &fakeSyslogMessageRepo{
			messages: []domain.SyslogMessage{
				{ID: 11, RawMessage: "connect payload", ParseStatus: "parsed"},
			},
		},
		ClientEvents: &fakeClientEventRepo{
			events: []domain.ClientEvent{
				{ID: 21, SyslogMessageID: 11, EventType: "connect", StationMac: "aa:bb:cc:dd:ee:ff", MatchStatus: "matched"},
			},
		},
		Logs: &fakeLogQueryRepo{
			items: []repository.LogListItem{
				{
					Message: domain.SyslogMessage{ID: 11, RawMessage: "connect payload", ParseStatus: "parsed"},
					Event:   &domain.ClientEvent{ID: 21, SyslogMessageID: 11, EventType: "connect", StationMac: "aa:bb:cc:dd:ee:ff", MatchStatus: "matched"},
				},
			},
		},
		Attendance: &fakeAttendanceRepo{
			records: []domain.AttendanceRecord{
				{ID: 31, EmployeeID: 1, ClockInStatus: "done", ClockOutStatus: "pending", SourceMode: "syslog"},
			},
		},
		Settings: &fakeSystemSettingRepo{
			settings: []domain.SystemSetting{
				{ID: 41, SettingKey: "report_target_url", SettingValue: "http://example.test/report"},
			},
		},
	})

	tests := []struct {
		name string
		path string
		want func(t *testing.T, body []byte)
	}{
		{
			name: "employees",
			path: "/api/employees",
			want: func(t *testing.T, body []byte) {
				t.Helper()
				var payload struct {
					Items []domain.Employee `json:"items"`
				}
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatalf("decode employees response: %v", err)
				}
				if len(payload.Items) != 1 || payload.Items[0].Name != "Alice" {
					t.Fatalf("unexpected employees payload: %+v", payload.Items)
				}
			},
		},
		{
			name: "logs",
			path: "/api/logs",
			want: func(t *testing.T, body []byte) {
				t.Helper()
				var payload struct {
					Items []map[string]any `json:"items"`
				}
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatalf("decode logs response: %v", err)
				}
				if len(payload.Items) != 1 {
					t.Fatalf("expected 1 log item, got %d", len(payload.Items))
				}
				if payload.Items[0]["message"] == nil {
					t.Fatalf("expected log item to include raw message, got %+v", payload.Items[0])
				}
				if payload.Items[0]["event"] == nil {
					t.Fatalf("expected log item to include parsed event, got %+v", payload.Items[0])
				}
			},
		},
		{
			name: "attendance",
			path: "/api/attendance",
			want: func(t *testing.T, body []byte) {
				t.Helper()
				var payload struct {
					Items []domain.AttendanceRecord `json:"items"`
				}
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatalf("decode attendance response: %v", err)
				}
				if len(payload.Items) != 1 || payload.Items[0].ClockInStatus != "done" {
					t.Fatalf("unexpected attendance payload: %+v", payload.Items)
				}
			},
		},
		{
			name: "settings",
			path: "/api/settings",
			want: func(t *testing.T, body []byte) {
				t.Helper()
				var payload struct {
					Items []domain.SystemSetting `json:"items"`
				}
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatalf("decode settings response: %v", err)
				}
				if len(payload.Items) != 1 || payload.Items[0].SettingKey != "report_target_url" {
					t.Fatalf("unexpected settings payload: %+v", payload.Items)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
			}
			tc.want(t, resp.Body.Bytes())
		})
	}
}

func TestNewServerUsesAdminRouter(t *testing.T) {
	server := httpapi.NewServer(":0", httpapi.Dependencies{
		Employees: &fakeEmployeeRepo{
			employees: []domain.Employee{{ID: 1, EmployeeNo: "EMP-001", Name: "Alice", Status: "active"}},
		},
	})

	if server.Addr != ":0" {
		t.Fatalf("expected addr :0, got %s", server.Addr)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/employees", nil)
	resp := httptest.NewRecorder()

	server.Handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
}
