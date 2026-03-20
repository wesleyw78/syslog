package integration

import (
	"encoding/json"
	"testing"
	"time"

	"syslog/internal/domain"
	"syslog/internal/parser"
	"syslog/internal/service"
)

func TestSyslogFlow(t *testing.T) {
	location := time.FixedZone("CST", 8*3600)
	attendanceDate := time.Date(2026, 3, 21, 0, 0, 0, 0, location)
	connectAt := time.Date(2026, 3, 21, 8, 1, 0, 0, location)
	disconnectAt := time.Date(2026, 3, 21, 18, 5, 0, 0, location)
	dayEndAt := time.Date(2026, 3, 21, 23, 59, 0, 0, location)

	connectRaw := "Mar 21 08:01:00 stamgr: Mef85d2S4D0 client_footprints connect Station[94:89:78:55:9a:f3] AP[28:b3:71:25:ae:a0] ssid[FactoryOps] osvendor[Unknown] hostname[scanner-01]"
	disconnectRaw := "Mar 21 18:05:00 stamgr: Mef85d2S4D0 client_footprints disconnect Station[94:89:78:55:9a:f3] AP[28:b3:71:25:ae:a0] ssid[FactoryOps] osvendor[Unknown] hostname[scanner-01]"

	employee := domain.Employee{
		ID:         42,
		EmployeeNo: "E-42",
		Name:       "Wesley Zhang",
		Status:     "active",
	}
	record := domain.AttendanceRecord{
		ID:              1001,
		AttendanceDate:  attendanceDate,
		ClockOutStatus:  "pending",
		ExceptionStatus: "none",
		SourceMode:      "syslog",
		Version:         1,
	}

	attendanceProcessor := service.NewAttendanceProcessor()
	reportService := service.NewReportService()
	dayEndService := service.NewDayEndService()

	connectEvent, err := parser.ParseAPSyslog(connectRaw, connectAt)
	if err != nil {
		t.Fatalf("parse connect syslog: %v", err)
	}
	if connectEvent.EventType != "connect" {
		t.Fatalf("expected connect event type, got %q", connectEvent.EventType)
	}
	if connectEvent.StationMac != "94:89:78:55:9a:f3" {
		t.Fatalf("expected parsed station mac, got %q", connectEvent.StationMac)
	}

	connectResult := attendanceProcessor.ApplyEvent(record, employee, connectEvent)
	if connectResult.Record.FirstConnectAt == nil {
		t.Fatal("expected first connect timestamp to be set")
	}
	if !connectResult.Record.FirstConnectAt.Equal(connectAt) {
		t.Fatalf("expected first connect %s, got %s", connectAt, connectResult.Record.FirstConnectAt)
	}
	if !connectResult.ClockInNeedsReport {
		t.Fatal("expected first connect to require clock-in report")
	}

	clockInReport := reportService.CreatePendingReport(
		connectResult.Record,
		"clock_in",
		*connectResult.Record.FirstConnectAt,
		"http://report.local/clock-in",
	)
	if clockInReport.ReportType != "clock_in" {
		t.Fatalf("expected clock_in report type, got %q", clockInReport.ReportType)
	}
	if clockInReport.ReportStatus != "pending" {
		t.Fatalf("expected pending clock_in report, got %q", clockInReport.ReportStatus)
	}

	disconnectEvent, err := parser.ParseAPSyslog(disconnectRaw, disconnectAt)
	if err != nil {
		t.Fatalf("parse disconnect syslog: %v", err)
	}
	if disconnectEvent.EventType != "disconnect" {
		t.Fatalf("expected disconnect event type, got %q", disconnectEvent.EventType)
	}

	disconnectResult := attendanceProcessor.ApplyEvent(connectResult.Record, employee, disconnectEvent)
	if disconnectResult.Record.LastDisconnectAt == nil {
		t.Fatal("expected last disconnect timestamp to be set")
	}
	if !disconnectResult.Record.LastDisconnectAt.Equal(disconnectAt) {
		t.Fatalf("expected last disconnect %s, got %s", disconnectAt, disconnectResult.Record.LastDisconnectAt)
	}
	if disconnectResult.ClockInNeedsReport {
		t.Fatal("did not expect disconnect to request a clock-in report")
	}

	finalizedRecord := dayEndService.FinalizeForDay(disconnectResult.Record, dayEndAt)
	if finalizedRecord.ClockOutStatus != "ready" {
		t.Fatalf("expected ready clock-out status after day-end, got %q", finalizedRecord.ClockOutStatus)
	}
	if finalizedRecord.ExceptionStatus != "none" {
		t.Fatalf("expected cleared exception status after day-end, got %q", finalizedRecord.ExceptionStatus)
	}

	clockOutReport := reportService.CreatePendingReport(
		finalizedRecord,
		"clock_out",
		*finalizedRecord.LastDisconnectAt,
		"http://report.local/clock-out",
	)
	if clockOutReport.ReportType != "clock_out" {
		t.Fatalf("expected clock_out report type, got %q", clockOutReport.ReportType)
	}
	if clockOutReport.ReportStatus != "pending" {
		t.Fatalf("expected pending clock_out report, got %q", clockOutReport.ReportStatus)
	}
	if clockOutReport.IdempotencyKey == clockInReport.IdempotencyKey {
		t.Fatal("expected distinct idempotency keys for clock_in and clock_out reports")
	}

	var clockOutPayload map[string]any
	if err := json.Unmarshal([]byte(clockOutReport.PayloadJSON), &clockOutPayload); err != nil {
		t.Fatalf("unmarshal clock_out payload: %v", err)
	}
	if clockOutPayload["reportType"] != "clock_out" {
		t.Fatalf("expected payload reportType clock_out, got %#v", clockOutPayload["reportType"])
	}
	if clockOutPayload["timestamp"] != disconnectAt.UTC().Format(time.RFC3339) {
		t.Fatalf(
			"expected payload timestamp %q, got %#v",
			disconnectAt.UTC().Format(time.RFC3339),
			clockOutPayload["timestamp"],
		)
	}
}
