package repository

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"syslog/internal/domain"
)

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	return db, mock
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("expected valid time %q, got %v", value, err)
	}
	return parsed
}

func TestMySQLEmployeeRepositoryFindByMACAddress(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewMySQLEmployeeRepository(db)
	now := mustTime(t, "2026-03-21T08:00:00Z")
	rows := sqlmock.NewRows([]string{"id", "employee_no", "system_no", "name", "status", "created_at", "updated_at"}).
		AddRow(uint64(7), "EMP-007", "SYS-007", "Alice", "active", now, now)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT e.id, e.employee_no, e.system_no, e.name, e.status, e.created_at, e.updated_at
		FROM employees e
		JOIN employee_devices d ON d.employee_id = e.id
		WHERE d.mac_address = ?
		LIMIT 1
	`))).WithArgs("aa:bb:cc:dd:ee:ff").WillReturnRows(rows)

	got, err := repo.FindByMACAddress(context.Background(), "aa:bb:cc:dd:ee:ff")
	if err != nil {
		t.Fatalf("expected lookup to succeed, got %v", err)
	}
	if got == nil {
		t.Fatalf("expected employee, got nil")
	}
	if got.ID != 7 || got.EmployeeNo != "EMP-007" || got.SystemNo != "SYS-007" || got.Name != "Alice" || got.Status != "active" {
		t.Fatalf("unexpected employee: %+v", got)
	}
	if !got.CreatedAt.Equal(now) || !got.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected timestamps: %+v", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all query expectations to be met, got %v", err)
	}
}

func TestMySQLEmployeeRepositoryList(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewMySQLEmployeeRepository(db)
	now := mustTime(t, "2026-03-21T08:00:00Z")
	rows := sqlmock.NewRows([]string{"id", "employee_no", "system_no", "name", "status", "created_at", "updated_at"}).
		AddRow(uint64(1), "EMP-001", "SYS-001", "Alice", "active", now, now).
		AddRow(uint64(2), "EMP-002", "SYS-002", "Bob", "disabled", now, now)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, employee_no, system_no, name, status, created_at, updated_at
		FROM employees
		ORDER BY id ASC
	`))).WillReturnRows(rows)

	got, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("expected list to succeed, got %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 employees, got %d", len(got))
	}
	if got[0].EmployeeNo != "EMP-001" || got[1].EmployeeNo != "EMP-002" {
		t.Fatalf("unexpected list result: %+v", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all query expectations to be met, got %v", err)
	}
}

func TestMySQLSyslogMessageRepositorySaveAndListRecent(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewMySQLSyslogMessageRepository(db)
	receivedAt := mustTime(t, "2026-03-21T08:00:00Z")
	logTime := mustTime(t, "2026-03-21T08:01:00Z")
	expiresAt := mustTime(t, "2026-03-31T08:00:00Z")
	message := &domain.SyslogMessage{
		ReceivedAt:        receivedAt,
		LogTime:           &logTime,
		RawMessage:        "<134>AP connect",
		SourceIP:          "10.0.0.1",
		Protocol:          "udp",
		ParseStatus:       "parsed",
		RetentionExpireAt: expiresAt,
	}
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		INSERT INTO syslog_messages (
			received_at,
			log_time,
			raw_message,
			source_ip,
			protocol,
			parse_status,
			retention_expire_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`))).WithArgs(receivedAt, logTime, "<134>AP connect", "10.0.0.1", "udp", "parsed", expiresAt).
		WillReturnResult(sqlmock.NewResult(19, 1))

	if err := repo.Save(context.Background(), message); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}
	if message.ID != 19 {
		t.Fatalf("expected inserted id 19, got %d", message.ID)
	}

	rows := sqlmock.NewRows([]string{"id", "received_at", "log_time", "raw_message", "source_ip", "protocol", "parse_status", "retention_expire_at"}).
		AddRow(uint64(19), receivedAt, logTime, "<134>AP connect", "10.0.0.1", "udp", "parsed", expiresAt)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, received_at, log_time, raw_message, source_ip, protocol, parse_status, retention_expire_at
		FROM syslog_messages
		ORDER BY received_at DESC, id DESC
		LIMIT ?
	`))).WithArgs(5).WillReturnRows(rows)

	got, err := repo.ListRecent(context.Background(), 5)
	if err != nil {
		t.Fatalf("expected list recent to succeed, got %v", err)
	}
	if len(got) != 1 || got[0].ID != 19 {
		t.Fatalf("unexpected recent logs: %+v", got)
	}
	if got[0].LogTime == nil || !got[0].LogTime.Equal(logTime) {
		t.Fatalf("expected log time to round-trip, got %+v", got[0].LogTime)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all sql expectations to be met, got %v", err)
	}
}

func TestMySQLClientEventRepositorySaveAndListRecent(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewMySQLClientEventRepository(db)
	eventTime := mustTime(t, "2026-03-21T08:05:00Z")
	eventDate := mustTime(t, "2026-03-21T00:00:00Z")
	matchedID := uint64(42)
	event := &domain.ClientEvent{
		SyslogMessageID:   19,
		EventDate:         eventDate,
		EventTime:         eventTime,
		EventType:         "connect",
		StationMac:        "aa:bb:cc:dd:ee:ff",
		APMac:             "11:22:33:44:55:66",
		SSID:              "corp",
		IPv4:              "10.0.0.2",
		IPv6:              "",
		Hostname:          "device-1",
		OSVendor:          "apple",
		MatchedEmployeeID: &matchedID,
		MatchStatus:       "matched",
	}
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		INSERT INTO client_events (
			syslog_message_id,
			event_date,
			event_time,
			event_type,
			station_mac,
			ap_mac,
			ssid,
			ipv4,
			ipv6,
			hostname,
			os_vendor,
			matched_employee_id,
			match_status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`))).WithArgs(int64(19), eventDate, eventTime, "connect", "aa:bb:cc:dd:ee:ff", "11:22:33:44:55:66", "corp", "10.0.0.2", "", "device-1", "apple", int64(matchedID), "matched").
		WillReturnResult(sqlmock.NewResult(33, 1))

	if err := repo.Save(context.Background(), event); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}
	if event.ID != 33 {
		t.Fatalf("expected inserted id 33, got %d", event.ID)
	}

	rows := sqlmock.NewRows([]string{"id", "syslog_message_id", "event_date", "event_time", "event_type", "station_mac", "ap_mac", "ssid", "ipv4", "ipv6", "hostname", "os_vendor", "matched_employee_id", "match_status"}).
		AddRow(uint64(33), uint64(19), eventDate, eventTime, "connect", "aa:bb:cc:dd:ee:ff", "11:22:33:44:55:66", "corp", "10.0.0.2", "", "device-1", "apple", int64(matchedID), "matched")
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, syslog_message_id, event_date, event_time, event_type, station_mac, ap_mac, ssid, ipv4, ipv6, hostname, os_vendor, matched_employee_id, match_status
		FROM client_events
		ORDER BY event_time DESC, id DESC
		LIMIT ?
	`))).WithArgs(10).WillReturnRows(rows)

	got, err := repo.ListRecent(context.Background(), 10)
	if err != nil {
		t.Fatalf("expected list recent to succeed, got %v", err)
	}
	if len(got) != 1 || got[0].ID != 33 {
		t.Fatalf("unexpected recent events: %+v", got)
	}
	if got[0].MatchedEmployeeID == nil || *got[0].MatchedEmployeeID != matchedID {
		t.Fatalf("expected matched employee id to round trip, got %+v", got[0].MatchedEmployeeID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all sql expectations to be met, got %v", err)
	}
}

func TestMySQLAttendanceRepositoryFindSaveAndListByDateRange(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewMySQLAttendanceRepository(db)
	attendanceDate := mustTime(t, "2026-03-21T00:00:00Z")
	firstConnect := mustTime(t, "2026-03-21T08:00:00Z")
	lastDisconnect := mustTime(t, "2026-03-21T17:00:00Z")
	lastCalculated := mustTime(t, "2026-03-21T17:05:00Z")
	record := &domain.AttendanceRecord{
		EmployeeID:       42,
		AttendanceDate:   attendanceDate,
		FirstConnectAt:   &firstConnect,
		LastDisconnectAt: &lastDisconnect,
		ClockInStatus:    "done",
		ClockOutStatus:   "done",
		ExceptionStatus:  "none",
		SourceMode:       "auto",
		Version:          2,
		LastCalculatedAt: &lastCalculated,
	}
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
	`))).WithArgs(int64(42), attendanceDate, firstConnect, lastDisconnect, "done", "done", "none", "auto", int64(2), lastCalculated).
		WillReturnResult(sqlmock.NewResult(55, 1))

	if err := repo.Save(context.Background(), record); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}
	if record.ID != 55 {
		t.Fatalf("expected inserted id 55, got %d", record.ID)
	}

	rows := sqlmock.NewRows([]string{"id", "employee_id", "attendance_date", "first_connect_at", "last_disconnect_at", "clock_in_status", "clock_out_status", "exception_status", "source_mode", "version", "last_calculated_at"}).
		AddRow(uint64(55), uint64(42), attendanceDate, firstConnect, lastDisconnect, "done", "done", "none", "auto", uint32(2), lastCalculated)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, employee_id, attendance_date, first_connect_at, last_disconnect_at, clock_in_status, clock_out_status, exception_status, source_mode, version, last_calculated_at
		FROM attendance_records
		WHERE employee_id = ? AND attendance_date = ?
		LIMIT 1
	`))).WithArgs(int64(42), attendanceDate).WillReturnRows(rows)

	found, err := repo.FindByEmployeeAndDate(context.Background(), 42, attendanceDate)
	if err != nil {
		t.Fatalf("expected find to succeed, got %v", err)
	}
	if found == nil || found.ID != 55 {
		t.Fatalf("unexpected find result: %+v", found)
	}

	listRows := sqlmock.NewRows([]string{"id", "employee_id", "attendance_date", "first_connect_at", "last_disconnect_at", "clock_in_status", "clock_out_status", "exception_status", "source_mode", "version", "last_calculated_at"}).
		AddRow(uint64(55), uint64(42), attendanceDate, firstConnect, lastDisconnect, "done", "done", "none", "auto", uint32(2), lastCalculated)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, employee_id, attendance_date, first_connect_at, last_disconnect_at, clock_in_status, clock_out_status, exception_status, source_mode, version, last_calculated_at
		FROM attendance_records
		WHERE attendance_date BETWEEN ? AND ?
		ORDER BY attendance_date DESC, employee_id ASC, id DESC
	`))).WithArgs(attendanceDate, attendanceDate).WillReturnRows(listRows)

	got, err := repo.ListByDateRange(context.Background(), attendanceDate, attendanceDate)
	if err != nil {
		t.Fatalf("expected list by date range to succeed, got %v", err)
	}
	if len(got) != 1 || got[0].ID != 55 {
		t.Fatalf("unexpected list result: %+v", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all sql expectations to be met, got %v", err)
	}
}

func TestMySQLReportRepositoryFindSaveAndListByAttendanceRecordID(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewMySQLReportRepository(db)
	reportedAt := mustTime(t, "2026-03-21T09:00:00Z")
	report := &domain.AttendanceReport{
		AttendanceRecordID: 55,
		ReportType:         "clock_in",
		IdempotencyKey:     "attendance-report/employee-42-2026-03-21/clock_in/2026-03-21T08:00:00Z/v2",
		PayloadJSON:        `{"attendanceRecordId":55}`,
		TargetURL:          "http://example.test/report",
		ReportStatus:       "pending",
		ResponseCode:       nil,
		ResponseBody:       "",
		ReportedAt:         &reportedAt,
		RetryCount:         1,
	}
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
	`))).WithArgs(int64(55), "clock_in", report.IdempotencyKey, report.PayloadJSON, report.TargetURL, "pending", nil, "", reportedAt, int64(1)).
		WillReturnResult(sqlmock.NewResult(88, 1))

	if err := repo.Save(context.Background(), report); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}
	if report.ID != 88 {
		t.Fatalf("expected inserted id 88, got %d", report.ID)
	}

	rows := sqlmock.NewRows([]string{"id", "attendance_record_id", "report_type", "idempotency_key", "payload_json", "target_url", "report_status", "response_code", "response_body", "reported_at", "retry_count"}).
		AddRow(uint64(88), uint64(55), "clock_in", report.IdempotencyKey, report.PayloadJSON, report.TargetURL, "pending", nil, "", reportedAt, uint32(1))
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, attendance_record_id, report_type, idempotency_key, payload_json, target_url, report_status, response_code, response_body, reported_at, retry_count
		FROM attendance_reports
		WHERE idempotency_key = ?
		LIMIT 1
	`))).WithArgs(report.IdempotencyKey).WillReturnRows(rows)

	found, err := repo.FindByIdempotencyKey(context.Background(), report.IdempotencyKey)
	if err != nil {
		t.Fatalf("expected lookup to succeed, got %v", err)
	}
	if found == nil || found.ID != 88 {
		t.Fatalf("unexpected report result: %+v", found)
	}

	listRows := sqlmock.NewRows([]string{"id", "attendance_record_id", "report_type", "idempotency_key", "payload_json", "target_url", "report_status", "response_code", "response_body", "reported_at", "retry_count"}).
		AddRow(uint64(88), uint64(55), "clock_in", report.IdempotencyKey, report.PayloadJSON, report.TargetURL, "pending", nil, "", reportedAt, uint32(1))
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, attendance_record_id, report_type, idempotency_key, payload_json, target_url, report_status, response_code, response_body, reported_at, retry_count
		FROM attendance_reports
		WHERE attendance_record_id = ?
		ORDER BY id DESC
	`))).WithArgs(int64(55)).WillReturnRows(listRows)

	got, err := repo.ListByAttendanceRecordID(context.Background(), 55)
	if err != nil {
		t.Fatalf("expected list to succeed, got %v", err)
	}
	if len(got) != 1 || got[0].ID != 88 {
		t.Fatalf("unexpected report list: %+v", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all sql expectations to be met, got %v", err)
	}
}

func TestMySQLSystemSettingRepositoryGetAndList(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewMySQLSystemSettingRepository(db)
	updatedAt := mustTime(t, "2026-03-21T08:30:00Z")
	rows := sqlmock.NewRows([]string{"id", "setting_key", "setting_value", "updated_at"}).
		AddRow(uint64(1), "day_end_time", "23:59", updatedAt)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, setting_key, setting_value, updated_at
		FROM system_settings
		WHERE setting_key = ?
		LIMIT 1
	`))).WithArgs("day_end_time").WillReturnRows(rows)

	got, err := repo.GetByKey(context.Background(), "day_end_time")
	if err != nil {
		t.Fatalf("expected get by key to succeed, got %v", err)
	}
	if got == nil || got.SettingValue != "23:59" {
		t.Fatalf("unexpected setting: %+v", got)
	}

	listRows := sqlmock.NewRows([]string{"id", "setting_key", "setting_value", "updated_at"}).
		AddRow(uint64(1), "day_end_time", "23:59", updatedAt).
		AddRow(uint64(2), "syslog_retention_days", "30", updatedAt)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, setting_key, setting_value, updated_at
		FROM system_settings
		ORDER BY setting_key ASC
	`))).WillReturnRows(listRows)

	list, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("expected list to succeed, got %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 settings, got %d", len(list))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all sql expectations to be met, got %v", err)
	}
}
