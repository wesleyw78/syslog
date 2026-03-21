package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
}

func (f *fakeEmployeeAdminWriter) CreateEmployee(_ context.Context, input service.EmployeeWriteInput) (*domain.Employee, error) {
	f.createInput = input
	return f.returned, nil
}

func (f *fakeEmployeeAdminWriter) UpdateEmployee(_ context.Context, id uint64, input service.EmployeeWriteInput) (*domain.Employee, error) {
	f.updateID = id
	f.updateInput = input
	return f.returned, nil
}

func (f *fakeEmployeeAdminWriter) DisableEmployee(_ context.Context, id uint64) (*domain.Employee, error) {
	f.disableID = id
	return f.returned, nil
}

type fakeSettingsAdminWriter struct {
	input   []service.SettingWriteInput
	returns []domain.SystemSetting
}

func (f *fakeSettingsAdminWriter) UpdateSettings(_ context.Context, input []service.SettingWriteInput) ([]domain.SystemSetting, error) {
	f.input = input
	return append([]domain.SystemSetting(nil), f.returns...), nil
}

type fakeAttendanceAdminWriter struct {
	id     uint64
	input  service.AttendanceCorrectionInput
	result *service.AttendanceCorrectionResult
}

func (f *fakeAttendanceAdminWriter) CorrectAttendance(_ context.Context, id uint64, input service.AttendanceCorrectionInput) (*service.AttendanceCorrectionResult, error) {
	f.id = id
	f.input = input
	return f.result, nil
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

	router := httpapi.NewRouter(httpapi.Dependencies{
		Employees:       &fakeEmployeeRepo{},
		EmployeeAdmin:   fakeEmployee,
		Attendance:      &fakeAttendanceRepo{},
		AttendanceAdmin: fakeAttendance,
		Settings:        &fakeSystemSettingRepo{},
		SettingsAdmin:   fakeSettings,
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
}
