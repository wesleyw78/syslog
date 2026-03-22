package service

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"syslog/internal/domain"
	"syslog/internal/repository"
)

type debugDispatchReportRepo struct {
	testReportRepo
	listDispatchableCalls int
}

func (r *debugDispatchReportRepo) ListDispatchable(context.Context, int, uint32) ([]domain.AttendanceReport, error) {
	r.listDispatchableCalls++
	return nil, nil
}

func (r *debugDispatchReportRepo) ListNotificationDispatchable(context.Context, int, uint32) ([]domain.AttendanceReport, error) {
	return nil, nil
}

type debugSyslogRuleRepo struct {
	rules []domain.SyslogReceiveRule
}

func (r *debugSyslogRuleRepo) List(context.Context) ([]domain.SyslogReceiveRule, error) {
	return append([]domain.SyslogReceiveRule(nil), r.rules...), nil
}

func (r *debugSyslogRuleRepo) ListEnabled(context.Context) ([]domain.SyslogReceiveRule, error) {
	enabled := make([]domain.SyslogReceiveRule, 0, len(r.rules))
	for _, rule := range r.rules {
		if rule.Enabled {
			enabled = append(enabled, rule)
		}
	}
	return enabled, nil
}

func (r *debugSyslogRuleRepo) FindByID(_ context.Context, id uint64) (*domain.SyslogReceiveRule, error) {
	for _, rule := range r.rules {
		if rule.ID == id {
			copied := rule
			return &copied, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (r *debugSyslogRuleRepo) Create(context.Context, *domain.SyslogReceiveRule) error { return nil }
func (r *debugSyslogRuleRepo) Update(context.Context, *domain.SyslogReceiveRule) error { return nil }
func (r *debugSyslogRuleRepo) Delete(context.Context, uint64) error                     { return nil }
func (r *debugSyslogRuleRepo) Move(context.Context, uint64, string) error               { return nil }

func TestDebugAdminServiceInjectSyslogUsesProvidedReceivedAt(t *testing.T) {
	location := time.FixedZone("CST", 8*3600)
	receivedAt := time.Date(2026, 3, 21, 8, 1, 0, 0, location)
	raw := "Mar 21 08:01:00 stamgr: client_footprints connect Station[94:89:78:55:9a:f3] AP[28:b3:71:25:ae:a0] ssid[FactoryOps] osvendor[Unknown] hostname[scanner-01]"

	messageRepo := &fakePipelineSyslogMessageRepo{}
	eventRepo := &fakePipelineClientEventRepo{}
	employeeRepo := &fakePipelineEmployeeRepo{}
	attendanceRepo := &fakePipelineAttendanceRepo{}
	reportRepo := &fakePipelineReportRepo{}
	ruleRepo := &debugSyslogRuleRepo{
		rules: []domain.SyslogReceiveRule{
			{
				ID:              1,
				Name:            "connect",
				Enabled:         true,
				EventType:       "connect",
				MessagePattern:  `connect .*?Station\[(?P<station_mac>[^\]]+)\]`,
				StationMacGroup: "station_mac",
			},
		},
	}

	pipeline := NewSyslogPipeline(SyslogPipelineDeps{
		Messages:       messageRepo,
		Events:         eventRepo,
		Employees:      employeeRepo,
		Attendance:     attendanceRepo,
		Reports:        reportRepo,
		Rules:          ruleRepo,
		RetentionDays:  30,
		AttendanceProc: NewAttendanceProcessor(),
		ReportSvc:      NewReportService(),
	})
	debugService := NewDebugAdminService(location, pipeline, nil, nil, nil, NewReportService())

	result, err := debugService.InjectSyslog(context.Background(), DebugSyslogInjectInput{
		RawMessage: raw,
		ReceivedAt: "2026-03-21T08:01",
	})
	if err != nil {
		t.Fatalf("expected syslog injection to succeed, got %v", err)
	}
	if !result.Accepted {
		t.Fatal("expected debug syslog injection to be accepted")
	}
	if result.ParseStatus != "parsed" {
		t.Fatalf("expected parsed status, got %q", result.ParseStatus)
	}
	if !result.ReceivedAt.Equal(receivedAt) {
		t.Fatalf("expected receivedAt %s, got %s", receivedAt, result.ReceivedAt)
	}
	if len(messageRepo.saved) != 1 {
		t.Fatalf("expected one saved message, got %d", len(messageRepo.saved))
	}
	if !messageRepo.saved[0].ReceivedAt.Equal(receivedAt) {
		t.Fatalf("expected persisted receivedAt %s, got %s", receivedAt, messageRepo.saved[0].ReceivedAt)
	}
	if messageRepo.saved[0].SourceIP != "127.0.0.1" {
		t.Fatalf("expected debug source ip 127.0.0.1, got %q", messageRepo.saved[0].SourceIP)
	}
}

func TestDebugAdminServiceInjectSyslogReturnsFailedParseStatusButStillPersists(t *testing.T) {
	location := time.FixedZone("CST", 8*3600)
	messageRepo := &fakePipelineSyslogMessageRepo{}
	ruleRepo := &debugSyslogRuleRepo{
		rules: []domain.SyslogReceiveRule{
			{
				ID:              1,
				Name:            "connect",
				Enabled:         true,
				EventType:       "connect",
				MessagePattern:  `connect .*?Station\[(?P<station_mac>[^\]]+)\]`,
				StationMacGroup: "station_mac",
			},
		},
	}
	pipeline := NewSyslogPipeline(SyslogPipelineDeps{
		Messages:       messageRepo,
		Events:         &fakePipelineClientEventRepo{},
		Employees:      &fakePipelineEmployeeRepo{},
		Attendance:     &fakePipelineAttendanceRepo{},
		Reports:        &fakePipelineReportRepo{},
		Rules:          ruleRepo,
		RetentionDays:  30,
		AttendanceProc: NewAttendanceProcessor(),
		ReportSvc:      NewReportService(),
	})
	debugService := NewDebugAdminService(location, pipeline, nil, nil, nil, NewReportService())

	result, err := debugService.InjectSyslog(context.Background(), DebugSyslogInjectInput{
		RawMessage: "invalid syslog",
		ReceivedAt: "2026-03-21T08:01:00+08:00",
	})
	if err != nil {
		t.Fatalf("expected parse failure to be reported without returning error, got %v", err)
	}
	if result.ParseStatus != "failed" {
		t.Fatalf("expected failed parse status, got %q", result.ParseStatus)
	}
	if strings.TrimSpace(result.ParseError) == "" {
		t.Fatal("expected parse error to be returned")
	}
	if len(messageRepo.saved) != 1 || messageRepo.saved[0].ParseStatus != "ignored" {
		t.Fatalf("expected unmatched syslog to be kept as ignored raw inbox row, got %+v", messageRepo.saved)
	}
}

func TestDebugAdminServiceDispatchAttendanceReportCreatesManualReplacementReport(t *testing.T) {
	location := time.FixedZone("CST", 8*3600)
	firstConnectAt := time.Date(2026, 3, 21, 8, 1, 0, 0, location)
	record := &domain.AttendanceRecord{
		ID:             41,
		EmployeeID:     7,
		AttendanceDate: time.Date(2026, 3, 21, 0, 0, 0, 0, location),
		FirstConnectAt: &firstConnectAt,
		ClockInStatus:  "done",
		ClockOutStatus: "pending",
		Version:        3,
	}
	attendanceRepo := &testAttendanceRepo{record: record}
	reportRepo := &debugDispatchReportRepo{
		testReportRepo: testReportRepo{
			latestClockIn: &domain.AttendanceReport{
				ID:                 91,
				AttendanceRecordID: 41,
				ReportType:         "clock_in",
				ReportStatus:       "success",
				ExternalRecordID:   "flow_prev_001",
			},
		},
	}
	employeeRepo := &dispatcherEmployeeRepo{
		employee: &domain.Employee{
			ID:               7,
			Name:             "Alice",
			FeishuEmployeeID: "fs_emp_007",
		},
	}
	settingsRepo := &dispatcherSettingsRepo{
		values: map[string]string{
			"feishu_app_id":          "cli_123",
			"feishu_app_secret":      "secret_456",
			"feishu_location_name":   "总部办公区",
			"report_timeout_seconds": "15",
			"report_retry_limit":     "3",
		},
	}
	client := &fakeFeishuAttendanceClient{recordID: "flow_new_001"}
	dispatcher := NewAttendanceReportDispatcher(AttendanceReportDispatcherDeps{
		Reports:   reportRepo,
		Employees: employeeRepo,
		Settings:  settingsRepo,
		Client:    client,
	})

	frozenNow := time.Date(2026, 3, 21, 20, 0, 0, 123000000, time.UTC)
	previousNow := debugNow
	debugNow = func() time.Time { return frozenNow }
	defer func() { debugNow = previousNow }()

	debugService := NewDebugAdminService(location, nil, attendanceRepo, reportRepo, dispatcher, NewReportService())

	result, err := debugService.DispatchAttendanceReport(context.Background(), 41, DebugAttendanceDispatchInput{
		ReportType: "clock_in",
	})
	if err != nil {
		t.Fatalf("expected manual attendance dispatch to succeed, got %v", err)
	}
	if !strings.Contains(result.Report.IdempotencyKey, "/manual/") {
		t.Fatalf("expected manual idempotency suffix, got %q", result.Report.IdempotencyKey)
	}
	if len(client.deleteRequests) != 1 || len(client.deleteRequests[0]) != 1 || client.deleteRequests[0][0] != "flow_prev_001" {
		t.Fatalf("expected old feishu flow to be deleted, got %+v", client.deleteRequests)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("expected one create request, got %d", len(client.createRequests))
	}
	if client.createRequests[0].UserID != "fs_emp_007" {
		t.Fatalf("expected Feishu user id fs_emp_007, got %q", client.createRequests[0].UserID)
	}
	if reportRepo.listDispatchableCalls != 0 {
		t.Fatalf("expected single dispatch path to skip queue scan, got %d list calls", reportRepo.listDispatchableCalls)
	}
	if len(reportRepo.saved) != 3 {
		t.Fatalf("expected initial save + delete progress save + dispatched save, got %d saves", len(reportRepo.saved))
	}
	if reportRepo.saved[0].DeleteRecordID != "flow_prev_001" {
		t.Fatalf("expected initial report to capture previous flow id, got %q", reportRepo.saved[0].DeleteRecordID)
	}
	if reportRepo.saved[1].DeleteRecordID != "" {
		t.Fatalf("expected intermediate delete progress save to clear deleteRecordID, got %q", reportRepo.saved[1].DeleteRecordID)
	}
	if reportRepo.saved[2].ReportStatus != "success" {
		t.Fatalf("expected dispatched report to succeed, got %q", reportRepo.saved[2].ReportStatus)
	}
	if reportRepo.saved[2].ExternalRecordID != "flow_new_001" {
		t.Fatalf("expected new external record id flow_new_001, got %q", reportRepo.saved[2].ExternalRecordID)
	}
}

func TestDebugAdminServiceDispatchAttendanceReportRejectsMissingTimestamp(t *testing.T) {
	location := time.FixedZone("CST", 8*3600)
	attendanceRepo := &testAttendanceRepo{
		record: &domain.AttendanceRecord{
			ID:             41,
			EmployeeID:     7,
			AttendanceDate: time.Date(2026, 3, 21, 0, 0, 0, 0, location),
			ClockInStatus:  "pending",
			ClockOutStatus: "pending",
			Version:        3,
		},
	}

	debugService := NewDebugAdminService(location, nil, attendanceRepo, &testReportRepo{}, nil, NewReportService())

	_, err := debugService.DispatchAttendanceReport(context.Background(), 41, DebugAttendanceDispatchInput{
		ReportType: "clock_in",
	})
	if !errors.Is(err, ErrInvalidDebugInput) {
		t.Fatalf("expected invalid debug input error, got %v", err)
	}
}

var _ repository.ReportRepository = (*debugDispatchReportRepo)(nil)
var _ repository.AttendanceRepository = (*testAttendanceRepo)(nil)
var _ repository.EmployeeRepository = (*dispatcherEmployeeRepo)(nil)
var _ repository.SystemSettingRepository = (*dispatcherSettingsRepo)(nil)
var _ = net.IP{}
