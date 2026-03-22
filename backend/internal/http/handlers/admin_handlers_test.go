package handlers_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"syslog/internal/domain"
	httpapi "syslog/internal/http"
	"syslog/internal/service"
)

type fakeEmployeeAdminWriter struct {
	createInput service.EmployeeWriteInput
	updateID    uint64
	updateInput service.EmployeeWriteInput
	disableID   uint64
	returned    *domain.Employee
	createErr   error
	updateErr   error
	disableErr  error
}

func (f *fakeEmployeeAdminWriter) CreateEmployee(_ context.Context, input service.EmployeeWriteInput) (*domain.Employee, error) {
	f.createInput = input
	return f.returned, f.createErr
}

func (f *fakeEmployeeAdminWriter) UpdateEmployee(_ context.Context, id uint64, input service.EmployeeWriteInput) (*domain.Employee, error) {
	f.updateID = id
	f.updateInput = input
	return f.returned, f.updateErr
}

func (f *fakeEmployeeAdminWriter) DisableEmployee(_ context.Context, id uint64) (*domain.Employee, error) {
	f.disableID = id
	return f.returned, f.disableErr
}

type fakeSettingsAdminWriter struct {
	input   []service.SettingWriteInput
	returns []domain.SystemSetting
	err     error
}

func (f *fakeSettingsAdminWriter) UpdateSettings(_ context.Context, input []service.SettingWriteInput) ([]domain.SystemSetting, error) {
	f.input = input
	return append([]domain.SystemSetting(nil), f.returns...), f.err
}

type fakeAttendanceAdminWriter struct {
	id     uint64
	input  service.AttendanceCorrectionInput
	result *service.AttendanceCorrectionResult
	err    error
}

func (f *fakeAttendanceAdminWriter) CorrectAttendance(_ context.Context, id uint64, input service.AttendanceCorrectionInput) (*service.AttendanceCorrectionResult, error) {
	f.id = id
	f.input = input
	return f.result, f.err
}

type fakeSyslogRuleReader struct {
	items []domain.SyslogReceiveRule
	err   error
}

func (f *fakeSyslogRuleReader) List(context.Context) ([]domain.SyslogReceiveRule, error) {
	return append([]domain.SyslogReceiveRule(nil), f.items...), f.err
}

func (f *fakeSyslogRuleReader) ListEnabled(context.Context) ([]domain.SyslogReceiveRule, error) {
	return append([]domain.SyslogReceiveRule(nil), f.items...), f.err
}

func (f *fakeSyslogRuleReader) FindByID(_ context.Context, id uint64) (*domain.SyslogReceiveRule, error) {
	for _, item := range f.items {
		if item.ID == id {
			copied := item
			return &copied, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (f *fakeSyslogRuleReader) Create(context.Context, *domain.SyslogReceiveRule) error {
	return nil
}

func (f *fakeSyslogRuleReader) Update(context.Context, *domain.SyslogReceiveRule) error {
	return nil
}

func (f *fakeSyslogRuleReader) Delete(context.Context, uint64) error {
	return nil
}

func (f *fakeSyslogRuleReader) Move(context.Context, uint64, string) error {
	return nil
}

type fakeSyslogRuleAdminWriter struct {
	createInput service.SyslogReceiveRuleWriteInput
	updateID    uint64
	updateInput service.SyslogReceiveRuleWriteInput
	deleteID    uint64
	moveID      uint64
	moveDir     string
	preview     service.SyslogRulePreviewInput
	returned    *domain.SyslogReceiveRule
	previewResp *service.SyslogRulePreviewResult
	createErr   error
	updateErr   error
	deleteErr   error
}

func (f *fakeSyslogRuleAdminWriter) CreateRule(_ context.Context, input service.SyslogReceiveRuleWriteInput) (*domain.SyslogReceiveRule, error) {
	f.createInput = input
	return f.returned, f.createErr
}

func (f *fakeSyslogRuleAdminWriter) UpdateRule(_ context.Context, id uint64, input service.SyslogReceiveRuleWriteInput) (*domain.SyslogReceiveRule, error) {
	f.updateID = id
	f.updateInput = input
	return f.returned, f.updateErr
}

func (f *fakeSyslogRuleAdminWriter) DeleteRule(_ context.Context, id uint64) error {
	f.deleteID = id
	return f.deleteErr
}

func (f *fakeSyslogRuleAdminWriter) MoveRule(_ context.Context, id uint64, direction string) (*domain.SyslogReceiveRule, error) {
	f.moveID = id
	f.moveDir = direction
	return f.returned, nil
}

func (f *fakeSyslogRuleAdminWriter) PreviewRule(_ context.Context, input service.SyslogRulePreviewInput) (*service.SyslogRulePreviewResult, error) {
	f.preview = input
	if f.previewResp != nil {
		return f.previewResp, nil
	}
	return &service.SyslogRulePreviewResult{}, nil
}

func TestAdminWriteRoutes(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	createdAt := time.Date(2026, 3, 21, 8, 0, 0, 0, loc)
	fakeEmployee := &fakeEmployeeAdminWriter{
		returned: &domain.Employee{
			ID:         1,
			EmployeeNo: "EMP-001",
			SystemNo:   "SYS-001",
			Name:       "Alice",
			Status:     "active",
			Devices: []domain.EmployeeDevice{
				{ID: 7, EmployeeID: 1, MacAddress: "aa:bb:cc:dd:ee:ff", DeviceLabel: "Phone", Status: "active", CreatedAt: createdAt, UpdatedAt: createdAt},
			},
		},
	}
	fakeSettings := &fakeSettingsAdminWriter{
		returns: []domain.SystemSetting{
			{ID: 1, SettingKey: "day_end_time", SettingValue: "22:00", UpdatedAt: createdAt},
		},
	}
	fakeAttendance := &fakeAttendanceAdminWriter{
		result: &service.AttendanceCorrectionResult{
			Record: domain.AttendanceRecord{
				ID:             55,
				EmployeeID:     1,
				AttendanceDate: time.Date(2026, 3, 21, 0, 0, 0, 0, loc),
				SourceMode:     "manual",
				Version:        3,
			},
			Reports: []domain.AttendanceReport{
				{ID: 81, AttendanceRecordID: 55, ReportType: "clock_in", ReportStatus: "pending"},
			},
		},
	}
	fakeSyslogRules := &fakeSyslogRuleAdminWriter{
		returned: &domain.SyslogReceiveRule{
			ID:              11,
			Name:            "Connect Rule",
			Enabled:         true,
			EventType:       "connect",
			MessagePattern:  `connect Station\[(?P<station_mac>[^\]]+)\]`,
			StationMacGroup: "station_mac",
		},
		previewResp: &service.SyslogRulePreviewResult{
			Matched: true,
			Event: &domain.ClientEvent{
				EventType:  "connect",
				StationMac: "aa:bb:cc:dd:ee:ff",
			},
		},
	}
	fakeSyslogRuleReader := &fakeSyslogRuleReader{
		items: []domain.SyslogReceiveRule{
			{
				ID:              11,
				Name:            "Connect Rule",
				Enabled:         true,
				EventType:       "connect",
				MessagePattern:  `connect Station\[(?P<station_mac>[^\]]+)\]`,
				StationMacGroup: "station_mac",
			},
		},
	}

	router := httpapi.NewRouter(httpapi.Dependencies{
		Employees:       &fakeEmployeeRepo{},
		EmployeeAdmin:   fakeEmployee,
		Attendance:      &fakeAttendanceRepo{},
		AttendanceAdmin: fakeAttendance,
		Settings:        &fakeSystemSettingRepo{},
		SettingsAdmin:   fakeSettings,
		SyslogRules:     fakeSyslogRuleReader,
		SyslogRuleAdmin: fakeSyslogRules,
	})

	t.Run("create employee", func(t *testing.T) {
		body := `{"employeeNo":"EMP-001","systemNo":"SYS-001","name":"Alice","status":"active","devices":[{"macAddress":"AA:BB:CC:DD:EE:FF","deviceLabel":"Phone","status":"active"}]}`
		req := httptest.NewRequest(http.MethodPost, "/api/employees", strings.NewReader(body))
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected created status, got %d", resp.Code)
		}
		if fakeEmployee.createInput.EmployeeNo != "EMP-001" || len(fakeEmployee.createInput.Devices) != 1 {
			t.Fatalf("unexpected employee input: %+v", fakeEmployee.createInput)
		}
		var payload struct {
			Employee domain.Employee `json:"employee"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if payload.Employee.ID != 1 || len(payload.Employee.Devices) != 1 {
			t.Fatalf("unexpected response: %+v", payload.Employee)
		}
	})

	t.Run("update employee", func(t *testing.T) {
		body := `{"employeeNo":"EMP-001","systemNo":"SYS-001","name":"Alice Updated","status":"active","devices":[{"macAddress":"11:22:33:44:55:66","deviceLabel":"Laptop","status":"active"}]}`
		req := httptest.NewRequest(http.MethodPut, "/api/employees/7", strings.NewReader(body))
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected ok status, got %d", resp.Code)
		}
		if fakeEmployee.updateID != 7 || fakeEmployee.updateInput.Name != "Alice Updated" {
			t.Fatalf("unexpected update input: id=%d input=%+v", fakeEmployee.updateID, fakeEmployee.updateInput)
		}
	})

	t.Run("disable employee", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/employees/7/disable", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected ok status, got %d", resp.Code)
		}
		if fakeEmployee.disableID != 7 {
			t.Fatalf("expected disable id 7, got %d", fakeEmployee.disableID)
		}
	})

	t.Run("update settings", func(t *testing.T) {
		body := `{"items":[{"settingKey":"day_end_time","settingValue":"22:00"}]}`
		req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(body))
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected ok status, got %d", resp.Code)
		}
		if len(fakeSettings.input) != 1 || fakeSettings.input[0].SettingKey != "day_end_time" {
			t.Fatalf("unexpected settings input: %+v", fakeSettings.input)
		}
		var payload struct {
			Items []domain.SystemSetting `json:"items"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(payload.Items) != 1 || payload.Items[0].SettingValue != "22:00" {
			t.Fatalf("unexpected settings response: %+v", payload.Items)
		}
	})

	t.Run("correct attendance", func(t *testing.T) {
		body := `{"firstConnectAt":"2026-03-21T08:10:00+08:00","lastDisconnectAt":"2026-03-21T17:30:00+08:00"}`
		req := httptest.NewRequest(http.MethodPost, "/api/attendance/55/correction", strings.NewReader(body))
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected ok status, got %d", resp.Code)
		}
		if fakeAttendance.id != 55 || !fakeAttendance.input.FirstConnectAt.Provided || !fakeAttendance.input.FirstConnectAt.Valid {
			t.Fatalf("unexpected attendance correction input: id=%d input=%+v", fakeAttendance.id, fakeAttendance.input)
		}
		var payload struct {
			Attendance domain.AttendanceRecord   `json:"attendance"`
			Reports    []domain.AttendanceReport `json:"reports"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if payload.Attendance.ID != 55 || len(payload.Reports) != 1 {
			t.Fatalf("unexpected attendance response: %+v", payload)
		}
	})

	t.Run("list syslog rules", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/syslog-rules", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected ok status, got %d", resp.Code)
		}
		var payload struct {
			Items []domain.SyslogReceiveRule `json:"items"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(payload.Items) != 1 || payload.Items[0].ID != 11 {
			t.Fatalf("unexpected syslog rules response: %+v", payload.Items)
		}
	})

	t.Run("create syslog rule", func(t *testing.T) {
		body := `{"name":"Disconnect Rule","enabled":true,"eventType":"disconnect","messagePattern":"disconnect Station\\[(?P<station_mac>[^\\]]+)\\]","stationMacGroup":"station_mac"}`
		req := httptest.NewRequest(http.MethodPost, "/api/syslog-rules", strings.NewReader(body))
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected created status, got %d", resp.Code)
		}
		if fakeSyslogRules.createInput.EventType != "disconnect" || fakeSyslogRules.createInput.StationMacGroup != "station_mac" {
			t.Fatalf("unexpected create rule input: %+v", fakeSyslogRules.createInput)
		}
	})

	t.Run("update syslog rule", func(t *testing.T) {
		body := `{"name":"Connect Rule 2","enabled":false,"eventType":"connect","messagePattern":"connect Station\\[(?P<station_mac>[^\\]]+)\\]","stationMacGroup":"station_mac"}`
		req := httptest.NewRequest(http.MethodPut, "/api/syslog-rules/11", strings.NewReader(body))
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected ok status, got %d", resp.Code)
		}
		if fakeSyslogRules.updateID != 11 || fakeSyslogRules.updateInput.Enabled {
			t.Fatalf("unexpected update rule input: id=%d input=%+v", fakeSyslogRules.updateID, fakeSyslogRules.updateInput)
		}
	})

	t.Run("delete syslog rule", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/syslog-rules/11", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusNoContent {
			t.Fatalf("expected no content status, got %d", resp.Code)
		}
		if fakeSyslogRules.deleteID != 11 {
			t.Fatalf("expected delete id 11, got %d", fakeSyslogRules.deleteID)
		}
	})

	t.Run("move syslog rule", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/syslog-rules/11/move", strings.NewReader(`{"direction":"down"}`))
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected ok status, got %d", resp.Code)
		}
		if fakeSyslogRules.moveID != 11 || fakeSyslogRules.moveDir != "down" {
			t.Fatalf("unexpected move request: id=%d dir=%q", fakeSyslogRules.moveID, fakeSyslogRules.moveDir)
		}
	})

	t.Run("preview syslog rule", func(t *testing.T) {
		body := `{"receivedAt":"2026-03-22T09:15:00+08:00","rawMessage":"Mar 22 09:15:00 stamgr: client_footprints connect Station[aa:bb:cc:dd:ee:ff]","rule":{"name":"Connect Rule","enabled":true,"eventType":"connect","messagePattern":"connect Station\\[(?P<station_mac>[^\\]]+)\\]","stationMacGroup":"station_mac"}}`
		req := httptest.NewRequest(http.MethodPost, "/api/syslog-rules/preview", strings.NewReader(body))
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected ok status, got %d", resp.Code)
		}
		if fakeSyslogRules.preview.RawMessage == "" || fakeSyslogRules.preview.Rule.StationMacGroup != "station_mac" {
			t.Fatalf("unexpected preview request: %+v", fakeSyslogRules.preview)
		}
	})
}

func TestAdminWriteRoutesReturnExpectedErrorStatuses(t *testing.T) {
	duplicateErr := &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}
	router := httpapi.NewRouter(httpapi.Dependencies{
		Employees: &fakeEmployeeRepo{},
		EmployeeAdmin: &fakeEmployeeAdminWriter{
			createErr: duplicateErr,
			updateErr: sql.ErrNoRows,
		},
		Attendance:      &fakeAttendanceRepo{},
		AttendanceAdmin: &fakeAttendanceAdminWriter{},
		Settings:        &fakeSystemSettingRepo{},
		SettingsAdmin:   &fakeSettingsAdminWriter{err: service.ErrInvalidSettingsInput},
		SyslogRules:     &fakeSyslogRuleReader{},
		SyslogRuleAdmin: &fakeSyslogRuleAdminWriter{createErr: service.ErrInvalidSyslogRuleInput},
	})

	t.Run("trailing json body returns bad request", func(t *testing.T) {
		body := `{"employeeNo":"EMP-001","systemNo":"SYS-001","name":"Alice","devices":[]}{ }`
		req := httptest.NewRequest(http.MethodPost, "/api/employees", strings.NewReader(body))
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected bad request, got %d", resp.Code)
		}
	})

	t.Run("duplicate employee returns conflict", func(t *testing.T) {
		body := `{"employeeNo":"EMP-001","systemNo":"SYS-001","name":"Alice","devices":[]}`
		req := httptest.NewRequest(http.MethodPost, "/api/employees", strings.NewReader(body))
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusConflict {
			t.Fatalf("expected conflict, got %d", resp.Code)
		}
	})

	t.Run("missing employee returns not found", func(t *testing.T) {
		body := `{"employeeNo":"EMP-001","systemNo":"SYS-001","name":"Alice","devices":[]}`
		req := httptest.NewRequest(http.MethodPut, "/api/employees/7", strings.NewReader(body))
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected not found, got %d", resp.Code)
		}
	})

	t.Run("duplicate settings batch returns bad request", func(t *testing.T) {
		body := `{"items":[{"settingKey":"day_end_time","settingValue":"22:00"}]}`
		req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(body))
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected bad request, got %d", resp.Code)
		}
	})

	t.Run("missing attendance correction returns not found", func(t *testing.T) {
		router := httpapi.NewRouter(httpapi.Dependencies{
			Employees:       &fakeEmployeeRepo{},
			EmployeeAdmin:   &fakeEmployeeAdminWriter{},
			Attendance:      &fakeAttendanceRepo{},
			AttendanceAdmin: &fakeAttendanceAdminWriter{err: sql.ErrNoRows},
			Settings:        &fakeSystemSettingRepo{},
			SettingsAdmin:   &fakeSettingsAdminWriter{},
			SyslogRules:     &fakeSyslogRuleReader{},
			SyslogRuleAdmin: &fakeSyslogRuleAdminWriter{},
		})

		body := `{"firstConnectAt":"2026-03-21T08:10:00+08:00"}`
		req := httptest.NewRequest(http.MethodPost, "/api/attendance/55/correction", strings.NewReader(body))
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected not found, got %d", resp.Code)
		}
	})

	t.Run("invalid syslog rule returns bad request", func(t *testing.T) {
		body := `{"name":"","enabled":true,"eventType":"connect","messagePattern":"","stationMacGroup":""}`
		req := httptest.NewRequest(http.MethodPost, "/api/syslog-rules", strings.NewReader(body))
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected bad request, got %d", resp.Code)
		}
	})
}

func TestDebugRoutesAreRegistered(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Employees: &fakeEmployeeRepo{},
		Attendance: &fakeAttendanceRepo{
			records: []domain.AttendanceRecord{
				{
					ID:             55,
					EmployeeID:     1,
					AttendanceDate: time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC),
					ClockInStatus:  "done",
					ClockOutStatus: "done",
					Version:        3,
				},
			},
		},
		Settings: &fakeSystemSettingRepo{},
	})

	t.Run("syslog debug route accepts injection payload", func(t *testing.T) {
		body := `{"rawMessage":"Mar 21 08:01:00 stamgr: Mef85d2S4D0 client_footprints connect Station[94:89:78:55:9a:f3] AP[28:b3:71:25:ae:a0] ssid[FactoryOps] osvendor[Unknown] hostname[scanner-01]","receivedAt":"2026-03-21T08:01:00+08:00"}`
		req := httptest.NewRequest(http.MethodPost, "/api/debug/syslog", strings.NewReader(body))
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code == http.StatusNotFound {
			t.Fatalf("expected debug syslog route to be registered, got %d", resp.Code)
		}
	})

	t.Run("attendance debug route accepts manual dispatch payload", func(t *testing.T) {
		body := `{"reportType":"clock_in"}`
		req := httptest.NewRequest(http.MethodPost, "/api/debug/attendance/55/dispatch", strings.NewReader(body))
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		if resp.Code == http.StatusNotFound {
			t.Fatalf("expected attendance debug dispatch route to be registered, got %d", resp.Code)
		}
	})
}
