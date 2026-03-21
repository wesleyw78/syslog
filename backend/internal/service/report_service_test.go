package service

import (
	"encoding/json"
	"testing"
	"time"

	"syslog/internal/domain"
)

func TestBuildIdempotencyKey(t *testing.T) {
	service := NewReportService()
	reportTime := time.Date(2026, 3, 21, 8, 1, 0, 0, time.FixedZone("CST", 8*3600))
	record := domain.AttendanceRecord{
		ID:             88,
		EmployeeID:     42,
		AttendanceDate: time.Date(2026, 3, 21, 0, 0, 0, 0, time.FixedZone("CST", 8*3600)),
		Version:        5,
	}

	key := service.BuildIdempotencyKey(record, "clock_in", reportTime)

	expected := "attendance-report/employee-42-2026-03-21/clock_in/2026-03-21T00:01:00Z/v5"
	if key != expected {
		t.Fatalf("expected idempotency key %q, got %q", expected, key)
	}
}

func TestBuildIdempotencyKeyUsesStableBusinessIdentity(t *testing.T) {
	service := NewReportService()
	reportTime := time.Date(2026, 3, 21, 8, 1, 0, 0, time.FixedZone("CST", 8*3600))
	baseRecord := domain.AttendanceRecord{
		EmployeeID:     42,
		AttendanceDate: time.Date(2026, 3, 21, 0, 0, 0, 0, time.FixedZone("CST", 8*3600)),
		Version:        5,
	}

	keyWithoutID := service.BuildIdempotencyKey(baseRecord, "clock_in", reportTime)

	withID := baseRecord
	withID.ID = 88
	keyWithID := service.BuildIdempotencyKey(withID, "clock_in", reportTime)

	if keyWithID != keyWithoutID {
		t.Fatalf("expected same idempotency key for same business record, got %q and %q", keyWithoutID, keyWithID)
	}
}

func TestCreatePendingReport(t *testing.T) {
	service := NewReportService()
	reportTime := time.Date(2026, 3, 21, 18, 5, 0, 0, time.FixedZone("CST", 8*3600))
	record := domain.AttendanceRecord{
		ID:             99,
		EmployeeID:     7,
		AttendanceDate: time.Date(2026, 3, 21, 0, 0, 0, 0, time.FixedZone("CST", 8*3600)),
		Version:        2,
	}

	report := service.CreatePendingReport(record, "clock_out", reportTime, "https://example.test/report")

	if report.AttendanceRecordID != record.ID {
		t.Fatalf("expected record id %d, got %d", record.ID, report.AttendanceRecordID)
	}
	if report.ReportStatus != "pending" {
		t.Fatalf("expected pending report status, got %q", report.ReportStatus)
	}
	if report.IdempotencyKey == "" {
		t.Fatal("expected idempotency key to be populated")
	}
	if report.TargetURL != "https://example.test/report" {
		t.Fatalf("expected target url to be preserved, got %q", report.TargetURL)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(report.PayloadJSON), &payload); err != nil {
		t.Fatalf("expected valid payload json, got error: %v", err)
	}
	if payload["reportType"] != "clock_out" {
		t.Fatalf("expected payload reportType clock_out, got %#v", payload["reportType"])
	}
	if payload["version"] != float64(2) {
		t.Fatalf("expected payload version 2, got %#v", payload["version"])
	}
	if payload["attendanceDate"] != "2026-03-21" {
		t.Fatalf("expected payload attendance date 2026-03-21, got %#v", payload["attendanceDate"])
	}
}

func TestCreateClearReport(t *testing.T) {
	service := NewReportService()
	record := domain.AttendanceRecord{
		ID:             99,
		EmployeeID:     7,
		AttendanceDate: time.Date(2026, 3, 21, 0, 0, 0, 0, time.FixedZone("CST", 8*3600)),
		Version:        2,
	}

	report := service.CreateClearReport(record, "clock_in", "https://example.test/report")

	if report.TargetURL != "https://example.test/report" {
		t.Fatalf("expected target url to be preserved, got %q", report.TargetURL)
	}
	if report.IdempotencyKey != "attendance-report/employee-7-2026-03-21/clock_in/clear/v2" {
		t.Fatalf("expected clear idempotency key, got %q", report.IdempotencyKey)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(report.PayloadJSON), &payload); err != nil {
		t.Fatalf("expected valid payload json, got error: %v", err)
	}
	if payload["action"] != "clear" {
		t.Fatalf("expected payload action clear, got %#v", payload["action"])
	}
	if payload["timestamp"] != nil {
		t.Fatalf("expected clear payload timestamp to be null, got %#v", payload["timestamp"])
	}
}
