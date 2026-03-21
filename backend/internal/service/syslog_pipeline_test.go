package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"syslog/internal/domain"
	"syslog/internal/repository"
)

type fakePipelineSyslogMessageRepo struct {
	saved []*domain.SyslogMessage
}

func (f *fakePipelineSyslogMessageRepo) Save(_ context.Context, message *domain.SyslogMessage) error {
	copied := *message
	f.saved = append(f.saved, &copied)
	message.ID = uint64(len(f.saved))
	return nil
}

func (f *fakePipelineSyslogMessageRepo) ListRecent(context.Context, int) ([]domain.SyslogMessage, error) {
	return nil, nil
}

type fakePipelineClientEventRepo struct {
	saved []*domain.ClientEvent
}

func (f *fakePipelineClientEventRepo) Save(_ context.Context, event *domain.ClientEvent) error {
	copied := *event
	f.saved = append(f.saved, &copied)
	event.ID = uint64(len(f.saved))
	return nil
}

func (f *fakePipelineClientEventRepo) ListRecent(context.Context, int) ([]domain.ClientEvent, error) {
	return nil, nil
}

type fakePipelineEmployeeRepo struct {
	employee *domain.Employee
	lookup   []string
}

func (f *fakePipelineEmployeeRepo) FindByMACAddress(_ context.Context, mac string) (*domain.Employee, error) {
	f.lookup = append(f.lookup, mac)
	if f.employee == nil {
		return nil, sql.ErrNoRows
	}

	copied := *f.employee
	return &copied, nil
}

func (f *fakePipelineEmployeeRepo) FindByID(context.Context, uint64) (*domain.Employee, error) {
	return nil, sql.ErrNoRows
}

func (f *fakePipelineEmployeeRepo) List(context.Context) ([]domain.Employee, error) {
	return nil, nil
}

func (f *fakePipelineEmployeeRepo) Create(context.Context, *domain.Employee) error {
	return nil
}

func (f *fakePipelineEmployeeRepo) Update(context.Context, *domain.Employee) error {
	return nil
}

func (f *fakePipelineEmployeeRepo) Disable(context.Context, uint64) error {
	return nil
}

func (f *fakePipelineEmployeeRepo) ReplaceDevices(context.Context, uint64, []domain.EmployeeDevice) error {
	return nil
}

func (f *fakePipelineEmployeeRepo) DisableDevicesByEmployeeID(context.Context, uint64) error {
	return nil
}

func (f *fakePipelineEmployeeRepo) WithTx(*sql.Tx) repository.EmployeeRepository {
	return f
}

type fakePipelineAttendanceRepo struct {
	found     *domain.AttendanceRecord
	findCalls []string
	saved     []*domain.AttendanceRecord
	findErr   error
	saveErr   error
}

func (f *fakePipelineAttendanceRepo) FindByID(context.Context, uint64) (*domain.AttendanceRecord, error) {
	return nil, sql.ErrNoRows
}

func (f *fakePipelineAttendanceRepo) FindByEmployeeAndDate(_ context.Context, employeeID uint64, attendanceDate time.Time) (*domain.AttendanceRecord, error) {
	f.findCalls = append(f.findCalls, attendanceDate.Format("2006-01-02"))
	if f.findErr != nil {
		return nil, f.findErr
	}
	if f.found == nil || f.found.EmployeeID != employeeID || !sameDay(f.found.AttendanceDate, attendanceDate) {
		return nil, sql.ErrNoRows
	}

	copied := *f.found
	return &copied, nil
}

func (f *fakePipelineAttendanceRepo) Save(_ context.Context, record *domain.AttendanceRecord) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	copied := *record
	f.saved = append(f.saved, &copied)
	record.ID = uint64(len(f.saved))
	f.found = &copied
	return nil
}

func (f *fakePipelineAttendanceRepo) ListByDateRange(context.Context, time.Time, time.Time) ([]domain.AttendanceRecord, error) {
	return nil, nil
}

func (f *fakePipelineAttendanceRepo) WithTx(*sql.Tx) repository.AttendanceRepository {
	return f
}

type fakePipelineReportRepo struct {
	found     *domain.AttendanceReport
	findCalls []string
	saved     []*domain.AttendanceReport
	findErr   error
	saveErr   error
}

func (f *fakePipelineReportRepo) FindByIdempotencyKey(_ context.Context, key string) (*domain.AttendanceReport, error) {
	f.findCalls = append(f.findCalls, key)
	if f.findErr != nil {
		return nil, f.findErr
	}
	if f.found == nil || f.found.IdempotencyKey != key {
		return nil, sql.ErrNoRows
	}

	copied := *f.found
	return &copied, nil
}

func (f *fakePipelineReportRepo) Save(_ context.Context, report *domain.AttendanceReport) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	copied := *report
	f.saved = append(f.saved, &copied)
	report.ID = uint64(len(f.saved))
	f.found = &copied
	return nil
}

func (f *fakePipelineReportRepo) ListByAttendanceRecordID(context.Context, uint64) ([]domain.AttendanceReport, error) {
	return nil, nil
}

type fakePipelineSettingRepo struct {
	settings map[string]string
	keys     []string
}

func (f *fakePipelineSettingRepo) GetByKey(_ context.Context, key string) (*domain.SystemSetting, error) {
	f.keys = append(f.keys, key)
	if value, ok := f.settings[key]; ok {
		return &domain.SystemSetting{SettingKey: key, SettingValue: value}, nil
	}
	return nil, sql.ErrNoRows
}

func (f *fakePipelineSettingRepo) List(context.Context) ([]domain.SystemSetting, error) {
	return nil, nil
}

func (f *fakePipelineSettingRepo) Save(context.Context, *domain.SystemSetting) error {
	return nil
}

func (f *fakePipelineSettingRepo) WithTx(*sql.Tx) repository.SystemSettingRepository {
	return f
}

func TestSyslogPipelineHandleSuccessCreatesDownstreamRecords(t *testing.T) {
	location := time.FixedZone("CST", 8*3600)
	receivedAt := time.Date(2026, 3, 21, 8, 1, 0, 0, location)
	raw := "Mar 21 08:01:00 stamgr: client_footprints connect Station[94:89:78:55:9a:f3] AP[28:b3:71:25:ae:a0] ssid[FactoryOps] osvendor[Unknown] hostname[scanner-01]"

	messageRepo := &fakePipelineSyslogMessageRepo{}
	eventRepo := &fakePipelineClientEventRepo{}
	employeeRepo := &fakePipelineEmployeeRepo{
		employee: &domain.Employee{ID: 42, EmployeeNo: "EMP-042", Name: "Alice", Status: "active"},
	}
	attendanceRepo := &fakePipelineAttendanceRepo{findErr: sql.ErrNoRows}
	reportRepo := &fakePipelineReportRepo{findErr: sql.ErrNoRows}
	settingsRepo := &fakePipelineSettingRepo{
		settings: map[string]string{
			"report_target_url": "http://example.test/report",
		},
	}

	pipeline := NewSyslogPipeline(SyslogPipelineDeps{
		Messages:       messageRepo,
		Events:         eventRepo,
		Employees:      employeeRepo,
		Attendance:     attendanceRepo,
		Reports:        reportRepo,
		Settings:       settingsRepo,
		RetentionDays:  30,
		AttendanceProc: NewAttendanceProcessor(),
		ReportSvc:      NewReportService(),
	})

	if err := pipeline.Handle(context.Background(), []byte(raw), &net.UDPAddr{IP: net.ParseIP("10.0.0.7"), Port: 1514}, receivedAt); err != nil {
		t.Fatalf("expected pipeline to succeed, got %v", err)
	}

	if len(messageRepo.saved) != 1 {
		t.Fatalf("expected 1 saved syslog message, got %d", len(messageRepo.saved))
	}
	if messageRepo.saved[0].ParseStatus != "parsed" {
		t.Fatalf("expected parsed message status, got %q", messageRepo.saved[0].ParseStatus)
	}
	if messageRepo.saved[0].SourceIP != "10.0.0.7" {
		t.Fatalf("expected source ip 10.0.0.7, got %q", messageRepo.saved[0].SourceIP)
	}

	if len(eventRepo.saved) != 1 {
		t.Fatalf("expected 1 saved client event, got %d", len(eventRepo.saved))
	}
	if eventRepo.saved[0].MatchedEmployeeID == nil || *eventRepo.saved[0].MatchedEmployeeID != 42 {
		t.Fatalf("expected matched employee id 42, got %+v", eventRepo.saved[0].MatchedEmployeeID)
	}
	if eventRepo.saved[0].MatchStatus != "matched" {
		t.Fatalf("expected matched status, got %q", eventRepo.saved[0].MatchStatus)
	}

	if len(attendanceRepo.saved) != 1 {
		t.Fatalf("expected 1 saved attendance record, got %d", len(attendanceRepo.saved))
	}
	if attendanceRepo.saved[0].EmployeeID != 42 {
		t.Fatalf("expected attendance record for employee 42, got %d", attendanceRepo.saved[0].EmployeeID)
	}
	if attendanceRepo.saved[0].FirstConnectAt == nil {
		t.Fatal("expected first connect timestamp to be set")
	}
	if len(reportRepo.saved) != 1 {
		t.Fatalf("expected 1 saved report, got %d", len(reportRepo.saved))
	}
	if reportRepo.saved[0].ReportStatus != "pending" {
		t.Fatalf("expected pending report, got %q", reportRepo.saved[0].ReportStatus)
	}
	if reportRepo.saved[0].TargetURL != "http://example.test/report" {
		t.Fatalf("expected report target url from settings, got %q", reportRepo.saved[0].TargetURL)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(reportRepo.saved[0].PayloadJSON), &payload); err != nil {
		t.Fatalf("expected report payload json, got %v", err)
	}
	if payload["reportType"] != "clock_in" {
		t.Fatalf("expected clock_in payload, got %#v", payload["reportType"])
	}
	if len(settingsRepo.keys) != 1 || settingsRepo.keys[0] != "report_target_url" {
		t.Fatalf("expected report target url lookup, got %+v", settingsRepo.keys)
	}
	if len(reportRepo.findCalls) != 1 {
		t.Fatalf("expected idempotency lookup before save, got %d calls", len(reportRepo.findCalls))
	}
}

func TestSyslogPipelineHandleParseFailureOnlyPersistsRawMessage(t *testing.T) {
	location := time.FixedZone("CST", 8*3600)
	receivedAt := time.Date(2026, 3, 21, 8, 1, 0, 0, location)

	messageRepo := &fakePipelineSyslogMessageRepo{}
	eventRepo := &fakePipelineClientEventRepo{}
	employeeRepo := &fakePipelineEmployeeRepo{}
	attendanceRepo := &fakePipelineAttendanceRepo{}
	reportRepo := &fakePipelineReportRepo{}
	settingsRepo := &fakePipelineSettingRepo{}

	pipeline := NewSyslogPipeline(SyslogPipelineDeps{
		Messages:       messageRepo,
		Events:         eventRepo,
		Employees:      employeeRepo,
		Attendance:     attendanceRepo,
		Reports:        reportRepo,
		Settings:       settingsRepo,
		RetentionDays:  30,
		AttendanceProc: NewAttendanceProcessor(),
		ReportSvc:      NewReportService(),
	})

	if err := pipeline.Handle(context.Background(), []byte("invalid syslog"), &net.UDPAddr{IP: net.ParseIP("10.0.0.8"), Port: 1514}, receivedAt); err != nil {
		t.Fatalf("expected parse failure to be swallowed, got %v", err)
	}

	if len(messageRepo.saved) != 1 {
		t.Fatalf("expected only raw message to be persisted, got %d", len(messageRepo.saved))
	}
	if messageRepo.saved[0].ParseStatus != "failed" {
		t.Fatalf("expected failed parse status, got %q", messageRepo.saved[0].ParseStatus)
	}
	if len(eventRepo.saved) != 0 || len(attendanceRepo.saved) != 0 || len(reportRepo.saved) != 0 {
		t.Fatalf("expected no downstream records on parse failure, got events=%d attendance=%d reports=%d", len(eventRepo.saved), len(attendanceRepo.saved), len(reportRepo.saved))
	}
}

func TestSyslogPipelineHandleEarlierConnectCreatesNewClockInReport(t *testing.T) {
	location := time.FixedZone("CST", 8*3600)
	receivedAt := time.Date(2026, 3, 21, 7, 50, 0, 0, location)
	raw := "Mar 21 07:50:00 stamgr: client_footprints connect Station[94:89:78:55:9a:f3] AP[28:b3:71:25:ae:a0] ssid[FactoryOps] osvendor[Unknown] hostname[scanner-01]"
	existingFirstConnect := time.Date(2026, 3, 21, 8, 10, 0, 0, location)

	messageRepo := &fakePipelineSyslogMessageRepo{}
	eventRepo := &fakePipelineClientEventRepo{}
	employeeRepo := &fakePipelineEmployeeRepo{
		employee: &domain.Employee{ID: 42, EmployeeNo: "EMP-042", Name: "Alice", Status: "active"},
	}
	attendanceRepo := &fakePipelineAttendanceRepo{
		found: &domain.AttendanceRecord{
			ID:              1001,
			EmployeeID:      42,
			AttendanceDate:  time.Date(2026, 3, 21, 0, 0, 0, 0, location),
			FirstConnectAt:  &existingFirstConnect,
			ClockInStatus:   "pending",
			ClockOutStatus:  "pending",
			ExceptionStatus: "none",
			SourceMode:      "syslog",
			Version:         1,
		},
	}
	reportRepo := &fakePipelineReportRepo{findErr: sql.ErrNoRows}
	settingsRepo := &fakePipelineSettingRepo{
		settings: map[string]string{
			"report_target_url": "http://example.test/report",
		},
	}

	pipeline := NewSyslogPipeline(SyslogPipelineDeps{
		Messages:       messageRepo,
		Events:         eventRepo,
		Employees:      employeeRepo,
		Attendance:     attendanceRepo,
		Reports:        reportRepo,
		Settings:       settingsRepo,
		RetentionDays:  30,
		AttendanceProc: NewAttendanceProcessor(),
		ReportSvc:      NewReportService(),
	})

	if err := pipeline.Handle(context.Background(), []byte(raw), &net.UDPAddr{IP: net.ParseIP("10.0.0.7"), Port: 1514}, receivedAt); err != nil {
		t.Fatalf("expected pipeline to succeed, got %v", err)
	}

	if len(attendanceRepo.saved) != 1 {
		t.Fatalf("expected attendance to be saved once, got %d", len(attendanceRepo.saved))
	}
	if attendanceRepo.saved[0].FirstConnectAt == nil || !attendanceRepo.saved[0].FirstConnectAt.Equal(receivedAt) {
		t.Fatalf("expected first connect to move earlier to %s, got %+v", receivedAt, attendanceRepo.saved[0].FirstConnectAt)
	}
	if len(reportRepo.saved) != 1 {
		t.Fatalf("expected new clock_in report to be saved, got %d", len(reportRepo.saved))
	}
	if len(reportRepo.findCalls) != 1 {
		t.Fatalf("expected one idempotency lookup, got %d", len(reportRepo.findCalls))
	}
	if reportRepo.saved[0].ReportType != "clock_in" {
		t.Fatalf("expected clock_in report, got %q", reportRepo.saved[0].ReportType)
	}
}

func TestSyslogPipelineHandleReportSaveFailureDoesNotLeaveHalfWrittenAttendance(t *testing.T) {
	location := time.FixedZone("CST", 8*3600)
	receivedAt := time.Date(2026, 3, 21, 8, 1, 0, 0, location)
	attendanceDate := time.Date(2026, 3, 21, 0, 0, 0, 0, location)
	raw := "Mar 21 08:01:00 stamgr: client_footprints connect Station[94:89:78:55:9a:f3] AP[28:b3:71:25:ae:a0] ssid[FactoryOps] osvendor[Unknown] hostname[scanner-01]"

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	messageRepo := &fakePipelineSyslogMessageRepo{}
	eventRepo := &fakePipelineClientEventRepo{}
	employeeRepo := &fakePipelineEmployeeRepo{
		employee: &domain.Employee{ID: 42, EmployeeNo: "EMP-042", Name: "Alice", Status: "active"},
	}
	settingsRepo := &fakePipelineSettingRepo{
		settings: map[string]string{
			"report_target_url": "http://example.test/report",
		},
	}

	attendanceRepo := repository.NewMySQLAttendanceRepository(db)
	reportRepo := repository.NewMySQLReportRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, employee_id, attendance_date, first_connect_at, last_disconnect_at, clock_in_status, clock_out_status, exception_status, source_mode, version, last_calculated_at
		FROM attendance_records
		WHERE employee_id = ? AND attendance_date = ?
		LIMIT 1
	`))).
		WithArgs(int64(42), attendanceDate).
		WillReturnRows(sqlmock.NewRows([]string{"id", "employee_id", "attendance_date", "first_connect_at", "last_disconnect_at", "clock_in_status", "clock_out_status", "exception_status", "source_mode", "version", "last_calculated_at"}))
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, attendance_record_id, report_type, idempotency_key, payload_json, target_url, report_status, response_code, response_body, reported_at, retry_count
		FROM attendance_reports
		WHERE idempotency_key = ?
		LIMIT 1
	`))).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "attendance_record_id", "report_type", "idempotency_key", "payload_json", "target_url", "report_status", "response_code", "response_body", "reported_at", "retry_count"}))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		INSERT INTO attendance_records (
			employee_id,
			attendance_date,
			first_connect_at,
			last_disconnect_at,
			clock_in_status,
			clock_out_status,
			exception_status,
			source_mode,
			version,
			last_calculated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			id = LAST_INSERT_ID(id),
			first_connect_at = VALUES(first_connect_at),
			last_disconnect_at = VALUES(last_disconnect_at),
			clock_in_status = VALUES(clock_in_status),
			clock_out_status = VALUES(clock_out_status),
			exception_status = VALUES(exception_status),
			source_mode = VALUES(source_mode),
			version = VALUES(version),
			last_calculated_at = VALUES(last_calculated_at)
	`))).
		WillReturnResult(sqlmock.NewResult(1001, 1))
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		INSERT INTO attendance_reports (
			attendance_record_id,
			report_type,
			idempotency_key,
			payload_json,
			target_url,
			report_status,
			response_code,
			response_body,
			reported_at,
			retry_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			id = LAST_INSERT_ID(id),
			attendance_record_id = VALUES(attendance_record_id),
			report_type = VALUES(report_type),
			payload_json = VALUES(payload_json),
			target_url = VALUES(target_url),
			report_status = VALUES(report_status),
			response_code = VALUES(response_code),
			response_body = VALUES(response_body),
			reported_at = VALUES(reported_at),
			retry_count = VALUES(retry_count)
	`))).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	pipeline := NewSyslogPipeline(SyslogPipelineDeps{
		DB:             db,
		Messages:       messageRepo,
		Events:         eventRepo,
		Employees:      employeeRepo,
		Attendance:     attendanceRepo,
		Reports:        reportRepo,
		Settings:       settingsRepo,
		RetentionDays:  30,
		AttendanceProc: NewAttendanceProcessor(),
		ReportSvc:      NewReportService(),
	})

	if err := pipeline.Handle(context.Background(), []byte(raw), &net.UDPAddr{IP: net.ParseIP("10.0.0.7"), Port: 1514}, receivedAt); err == nil {
		t.Fatal("expected report save failure to surface")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected transaction to roll back, got %v", err)
	}
}

func sameDay(a, b time.Time) bool {
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

var _ repository.SyslogMessageRepository = (*fakePipelineSyslogMessageRepo)(nil)
var _ repository.ClientEventRepository = (*fakePipelineClientEventRepo)(nil)
var _ repository.EmployeeRepository = (*fakePipelineEmployeeRepo)(nil)
var _ repository.AttendanceRepository = (*fakePipelineAttendanceRepo)(nil)
var _ repository.ReportRepository = (*fakePipelineReportRepo)(nil)
var _ repository.SystemSettingRepository = (*fakePipelineSettingRepo)(nil)
