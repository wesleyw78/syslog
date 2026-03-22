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

func TestParseInsertedIDRejectsNegative(t *testing.T) {
	id, err := parseInsertedID(sqlmock.NewResult(-1, 1))
	if err == nil {
		t.Fatalf("expected negative last insert id to fail")
	}
	if id != 0 {
		t.Fatalf("expected zero id on failure, got %d", id)
	}
}

func TestMySQLEmployeeRepositoryFindByMACAddress(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewMySQLEmployeeRepository(db)
	now := mustTime(t, "2026-03-21T08:00:00Z")
	rows := sqlmock.NewRows([]string{"id", "employee_no", "system_no", "feishu_employee_id", "name", "status", "created_at", "updated_at"}).
		AddRow(uint64(7), "EMP-007", "SYS-007", "fs_emp_007", "Alice", "active", now, now)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT e.id, e.employee_no, e.system_no, e.feishu_employee_id, e.name, e.status, e.created_at, e.updated_at
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
	if got.ID != 7 || got.EmployeeNo != "EMP-007" || got.SystemNo != "SYS-007" || got.FeishuEmployeeID != "fs_emp_007" || got.Name != "Alice" || got.Status != "active" {
		t.Fatalf("unexpected employee: %+v", got)
	}
	if !got.CreatedAt.Equal(now) || !got.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected timestamps: %+v", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all query expectations to be met, got %v", err)
	}
}

func TestMySQLEmployeeRepositoryFindByMACAddressHandlesNullFeishuEmployeeID(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewMySQLEmployeeRepository(db)
	now := mustTime(t, "2026-03-21T08:00:00Z")
	rows := sqlmock.NewRows([]string{"id", "employee_no", "system_no", "feishu_employee_id", "name", "status", "created_at", "updated_at"}).
		AddRow(uint64(8), "EMP-008", "SYS-008", nil, "Bob", "active", now, now)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT e.id, e.employee_no, e.system_no, e.feishu_employee_id, e.name, e.status, e.created_at, e.updated_at
		FROM employees e
		JOIN employee_devices d ON d.employee_id = e.id
		WHERE d.mac_address = ?
		LIMIT 1
	`))).WithArgs("11:22:33:44:55:66").WillReturnRows(rows)

	got, err := repo.FindByMACAddress(context.Background(), "11:22:33:44:55:66")
	if err != nil {
		t.Fatalf("expected lookup to succeed with null feishu employee id, got %v", err)
	}
	if got == nil {
		t.Fatal("expected employee, got nil")
	}
	if got.FeishuEmployeeID != "" {
		t.Fatalf("expected null feishu employee id to normalize to empty string, got %q", got.FeishuEmployeeID)
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
	rows := sqlmock.NewRows([]string{"id", "employee_no", "system_no", "feishu_employee_id", "name", "status", "created_at", "updated_at", "device_id", "mac_address", "device_label", "device_status", "device_created_at", "device_updated_at"}).
		AddRow(uint64(1), "EMP-001", "SYS-001", "fs_emp_001", "Alice", "active", now, now, nil, nil, nil, nil, nil, nil).
		AddRow(uint64(2), "EMP-002", "SYS-002", nil, "Bob", "disabled", now, now, nil, nil, nil, nil, nil, nil)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT e.id, e.employee_no, e.system_no, e.feishu_employee_id, e.name, e.status, e.created_at, e.updated_at,
		       d.id, d.mac_address, d.device_label, d.status, d.created_at, d.updated_at
		FROM employees e
		LEFT JOIN employee_devices d ON d.employee_id = e.id
		ORDER BY e.id ASC, d.id ASC
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
	if got[0].FeishuEmployeeID != "fs_emp_001" || got[1].FeishuEmployeeID != "" {
		t.Fatalf("unexpected feishu employee ids: %+v", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all query expectations to be met, got %v", err)
	}
}

func TestMySQLEmployeeRepositoryFindByID(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewMySQLEmployeeRepository(db)
	now := mustTime(t, "2026-03-21T08:00:00Z")
	rows := sqlmock.NewRows([]string{"id", "employee_no", "system_no", "feishu_employee_id", "name", "status", "created_at", "updated_at", "device_id", "mac_address", "device_label", "device_status", "device_created_at", "device_updated_at"}).
		AddRow(uint64(1), "EMP-001", "SYS-001", "fs_emp_001", "Alice", "active", now, now, uint64(11), "aa:bb:cc:dd:ee:ff", "Phone", "active", now, now)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT e.id, e.employee_no, e.system_no, e.feishu_employee_id, e.name, e.status, e.created_at, e.updated_at,
		       d.id, d.mac_address, d.device_label, d.status, d.created_at, d.updated_at
		FROM employees e
		LEFT JOIN employee_devices d ON d.employee_id = e.id
		WHERE e.id = ?
		ORDER BY d.id ASC
	`))).WithArgs(uint64(1)).WillReturnRows(rows)

	got, err := repo.FindByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected find by id to succeed, got %v", err)
	}
	if got == nil || got.ID != 1 {
		t.Fatalf("unexpected employee result: %+v", got)
	}
	if got.FeishuEmployeeID != "fs_emp_001" {
		t.Fatalf("expected feishu employee id to round-trip, got %+v", got)
	}
	if len(got.Devices) != 1 || got.Devices[0].MacAddress != "aa:bb:cc:dd:ee:ff" {
		t.Fatalf("expected device to round-trip, got %+v", got.Devices)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all query expectations to be met, got %v", err)
	}
}

func TestMySQLEmployeeRepositoryCreateUpdateDisableAndReplaceDevices(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewMySQLEmployeeRepository(db)
	employee := &domain.Employee{EmployeeNo: "EMP-001", SystemNo: "SYS-001", FeishuEmployeeID: "fs_emp_001", Name: "Alice", Status: "active"}
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		INSERT INTO employees (
			employee_no,
			system_no,
			feishu_employee_id,
			name,
			status
		) VALUES (?, ?, ?, ?, ?)
	`))).WithArgs("EMP-001", "SYS-001", "fs_emp_001", "Alice", "active").
		WillReturnResult(sqlmock.NewResult(11, 1))

	if err := repo.Create(context.Background(), employee); err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}
	if employee.ID != 11 {
		t.Fatalf("expected inserted id 11, got %d", employee.ID)
	}

	employee.Name = "Alice Updated"
	employee.Status = "active"
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		UPDATE employees
		SET employee_no = ?, system_no = ?, feishu_employee_id = ?, name = ?, status = ?
		WHERE id = ?
	`))).WithArgs("EMP-001", "SYS-001", "fs_emp_001", "Alice Updated", "active", uint64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.Update(context.Background(), employee); err != nil {
		t.Fatalf("expected update to succeed, got %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		UPDATE employees
		SET employee_no = ?, system_no = ?, feishu_employee_id = ?, name = ?, status = ?
		WHERE id = ?
	`))).WithArgs("EMP-001", "SYS-001", nil, "Alice Updated", "active", uint64(12)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	if err := repo.Update(context.Background(), &domain.Employee{ID: 12, EmployeeNo: "EMP-001", SystemNo: "SYS-001", Name: "Alice Updated", Status: "active"}); err != nil {
		t.Fatalf("expected update no-op to succeed, got %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		UPDATE employees
		SET status = ?
		WHERE id = ?
	`))).WithArgs("disabled", uint64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.Disable(context.Background(), 11); err != nil {
		t.Fatalf("expected disable to succeed, got %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		UPDATE employees
		SET status = ?
		WHERE id = ?
	`))).WithArgs("disabled", uint64(12)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	if err := repo.Disable(context.Background(), 12); err != nil {
		t.Fatalf("expected disable no-op to succeed, got %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		DELETE FROM employee_devices
		WHERE employee_id = ?
	`))).WithArgs(uint64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		INSERT INTO employee_devices (
			employee_id,
			mac_address,
			device_label,
			status
		) VALUES (?, ?, ?, ?)
	`))).WithArgs(uint64(11), "aa:bb:cc:dd:ee:ff", "Phone", "active").
		WillReturnResult(sqlmock.NewResult(21, 1))
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		INSERT INTO employee_devices (
			employee_id,
			mac_address,
			device_label,
			status
		) VALUES (?, ?, ?, ?)
	`))).WithArgs(uint64(11), "11:22:33:44:55:66", "Tablet", "disabled").
		WillReturnResult(sqlmock.NewResult(22, 1))
	if err := repo.ReplaceDevices(context.Background(), 11, []domain.EmployeeDevice{
		{MacAddress: "aa:bb:cc:dd:ee:ff", DeviceLabel: "Phone", Status: "active"},
		{MacAddress: "11:22:33:44:55:66", DeviceLabel: "Tablet", Status: "disabled"},
	}); err != nil {
		t.Fatalf("expected replace devices to succeed, got %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		UPDATE employee_devices
		SET status = ?
		WHERE employee_id = ?
	`))).WithArgs("disabled", uint64(11)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	if err := repo.DisableDevicesByEmployeeID(context.Background(), 11); err != nil {
		t.Fatalf("expected disable devices to succeed, got %v", err)
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
			matched_rule_id,
			retention_expire_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`))).WithArgs(receivedAt, logTime, "<134>AP connect", "10.0.0.1", "udp", "parsed", nil, expiresAt).
		WillReturnResult(sqlmock.NewResult(19, 1))

	if err := repo.Save(context.Background(), message); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}
	if message.ID != 19 {
		t.Fatalf("expected inserted id 19, got %d", message.ID)
	}

	rows := sqlmock.NewRows([]string{"id", "received_at", "log_time", "raw_message", "source_ip", "protocol", "parse_status", "matched_rule_id", "retention_expire_at"}).
		AddRow(uint64(19), receivedAt, logTime, "<134>AP connect", "10.0.0.1", "udp", "parsed", nil, expiresAt)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, received_at, log_time, raw_message, source_ip, protocol, parse_status, matched_rule_id, retention_expire_at
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

func TestMySQLLogQueryRepositoryListPage(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewMySQLLogQueryRepository(db)
	receivedAt := mustTime(t, "2026-03-21T08:00:00Z")
	logTime := mustTime(t, "2026-03-21T07:59:00Z")
	eventTime := mustTime(t, "2026-03-21T08:00:30Z")

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(25)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT COUNT(*)
		FROM syslog_messages AS sm
		LEFT JOIN client_events AS ce ON ce.syslog_message_id = sm.id
		WHERE ce.id IS NOT NULL
		AND (
			sm.raw_message LIKE ?
			OR sm.parse_status LIKE ?
			OR sm.source_ip LIKE ?
			OR sm.protocol LIKE ?
			OR ce.event_type LIKE ?
			OR ce.station_mac LIKE ?
			OR ce.hostname LIKE ?
			OR ce.ap_mac LIKE ?
			OR ce.ssid LIKE ?
			OR ce.ipv4 LIKE ?
			OR ce.ipv6 LIKE ?
			OR ce.os_vendor LIKE ?
			OR ce.match_status LIKE ?
		)
		AND sm.received_at >= ?
		AND sm.received_at < ?
	`))).WithArgs(
		"%device%", "%device%", "%device%", "%device%",
		"%device%", "%device%", "%device%", "%device%",
		"%device%", "%device%", "%device%", "%device%", "%device%",
		"2026-03-20 00:00:00",
		"2026-03-22 00:00:00",
	).WillReturnRows(countRows)

	resultRows := sqlmock.NewRows([]string{
		"message_id", "received_at", "log_time", "raw_message", "source_ip", "protocol", "parse_status", "matched_rule_id", "retention_expire_at", "matched_rule_name",
		"event_id", "syslog_message_id", "event_date", "event_time", "event_type", "station_mac", "ap_mac", "ssid", "ipv4", "ipv6", "hostname", "os_vendor", "matched_employee_id", "match_status",
	}).AddRow(
		uint64(19), receivedAt, logTime, "<134> device connected", "10.0.0.1", "udp", "parsed", nil, receivedAt.Add(24*time.Hour), "默认 connect 规则",
		uint64(33), uint64(19), receivedAt, eventTime, "connect", "aa:bb:cc:dd:ee:ff", "11:22:33:44:55:66", "corp", "10.0.0.2", "", "device-1", "apple", nil, "matched",
	)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT
			sm.id,
			sm.received_at,
			sm.log_time,
			sm.raw_message,
			sm.source_ip,
			sm.protocol,
			sm.parse_status,
			sm.matched_rule_id,
			sm.retention_expire_at,
			sr.name,
			ce.id,
			ce.syslog_message_id,
			ce.event_date,
			ce.event_time,
			ce.event_type,
			ce.station_mac,
			ce.ap_mac,
			ce.ssid,
			ce.ipv4,
			ce.ipv6,
			ce.hostname,
			ce.os_vendor,
			ce.matched_employee_id,
			ce.match_status
		FROM syslog_messages AS sm
		LEFT JOIN syslog_receive_rules AS sr ON sr.id = sm.matched_rule_id
		LEFT JOIN client_events AS ce ON ce.syslog_message_id = sm.id
		WHERE ce.id IS NOT NULL
		AND (
			sm.raw_message LIKE ?
			OR sm.parse_status LIKE ?
			OR sm.source_ip LIKE ?
			OR sm.protocol LIKE ?
			OR ce.event_type LIKE ?
			OR ce.station_mac LIKE ?
			OR ce.hostname LIKE ?
			OR ce.ap_mac LIKE ?
			OR ce.ssid LIKE ?
			OR ce.ipv4 LIKE ?
			OR ce.ipv6 LIKE ?
			OR ce.os_vendor LIKE ?
			OR ce.match_status LIKE ?
		)
		AND sm.received_at >= ?
		AND sm.received_at < ?
		ORDER BY sm.received_at DESC, sm.id DESC
		LIMIT ? OFFSET ?
	`))).WithArgs(
		"%device%", "%device%", "%device%", "%device%",
		"%device%", "%device%", "%device%", "%device%",
		"%device%", "%device%", "%device%", "%device%", "%device%",
		"2026-03-20 00:00:00",
		"2026-03-22 00:00:00",
		10, 10,
	).WillReturnRows(resultRows)

	result, err := repo.ListPage(context.Background(), LogListParams{
		FromDate: "2026-03-20",
		Page:     2,
		PageSize: 10,
		Query:    "device",
		ToDate:   "2026-03-21",
	})
	if err != nil {
		t.Fatalf("expected list page to succeed, got %v", err)
	}

	if result.Page != 2 || result.PageSize != 10 {
		t.Fatalf("unexpected pagination: %+v", result)
	}
	if result.TotalItems != 25 || result.TotalPages != 3 {
		t.Fatalf("unexpected totals: %+v", result)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected one log item, got %d", len(result.Items))
	}
	if result.Items[0].Message.ID != 19 {
		t.Fatalf("unexpected message id: %+v", result.Items[0].Message)
	}
	if result.Items[0].Event == nil || result.Items[0].Event.ID != 33 {
		t.Fatalf("unexpected event: %+v", result.Items[0].Event)
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

func TestMySQLAttendanceRepositoryFindByID(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewMySQLAttendanceRepository(db)
	attendanceDate := mustTime(t, "2026-03-21T00:00:00Z")
	firstConnect := mustTime(t, "2026-03-21T08:00:00Z")
	lastDisconnect := mustTime(t, "2026-03-21T17:00:00Z")
	lastCalculated := mustTime(t, "2026-03-21T17:05:00Z")
	rows := sqlmock.NewRows([]string{"id", "employee_id", "attendance_date", "first_connect_at", "last_disconnect_at", "clock_in_status", "clock_out_status", "exception_status", "source_mode", "version", "last_calculated_at"}).
		AddRow(uint64(55), uint64(42), attendanceDate, firstConnect, lastDisconnect, "done", "done", "none", "manual", uint32(3), lastCalculated)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, employee_id, attendance_date, first_connect_at, last_disconnect_at, clock_in_status, clock_out_status, exception_status, source_mode, version, last_calculated_at
		FROM attendance_records
		WHERE id = ?
		LIMIT 1
	`))).WithArgs(uint64(55)).WillReturnRows(rows)

	got, err := repo.FindByID(context.Background(), 55)
	if err != nil {
		t.Fatalf("expected find by id to succeed, got %v", err)
	}
	if got == nil || got.ID != 55 || got.Version != 3 {
		t.Fatalf("unexpected attendance record: %+v", got)
	}
	if got.FirstConnectAt == nil || !got.FirstConnectAt.Equal(firstConnect) {
		t.Fatalf("expected first connect to round-trip, got %+v", got.FirstConnectAt)
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
		AttendanceRecordID:       55,
		ReportType:               "clock_in",
		IdempotencyKey:           "attendance-report/employee-42-2026-03-21/clock_in/2026-03-21T08:00:00Z/v2",
		PayloadJSON:              `{"attendanceRecordId":55}`,
		TargetURL:                "http://example.test/report",
		ExternalRecordID:         "flow_001",
		DeleteRecordID:           "flow_old_001",
		ReportStatus:             "pending",
		ResponseCode:             nil,
		ResponseBody:             "",
		NotificationStatus:       "pending",
		NotificationMessageID:    "",
		NotificationResponseCode: nil,
		NotificationResponseBody: "",
		NotificationSentAt:       nil,
		NotificationRetryCount:   0,
		ReportedAt:               &reportedAt,
		RetryCount:               1,
	}
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		INSERT INTO attendance_reports (
			attendance_record_id,
			report_type,
			idempotency_key,
			payload_json,
			target_url,
			external_record_id,
			delete_record_id,
			report_status,
			response_code,
			response_body,
			notification_status,
			notification_message_id,
			notification_response_code,
			notification_response_body,
			notification_sent_at,
			notification_retry_count,
			reported_at,
			retry_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			id = LAST_INSERT_ID(id),
			attendance_record_id = VALUES(attendance_record_id),
			report_type = VALUES(report_type),
			payload_json = VALUES(payload_json),
			target_url = VALUES(target_url),
			external_record_id = VALUES(external_record_id),
			delete_record_id = VALUES(delete_record_id),
			report_status = VALUES(report_status),
			response_code = VALUES(response_code),
			response_body = VALUES(response_body),
			notification_status = VALUES(notification_status),
			notification_message_id = VALUES(notification_message_id),
			notification_response_code = VALUES(notification_response_code),
			notification_response_body = VALUES(notification_response_body),
			notification_sent_at = VALUES(notification_sent_at),
			notification_retry_count = VALUES(notification_retry_count),
			reported_at = VALUES(reported_at),
			retry_count = VALUES(retry_count)
	`))).WithArgs(int64(55), "clock_in", report.IdempotencyKey, report.PayloadJSON, report.TargetURL, report.ExternalRecordID, report.DeleteRecordID, "pending", nil, "", "pending", "", nil, "", nil, int64(0), reportedAt, int64(1)).
		WillReturnResult(sqlmock.NewResult(88, 1))

	if err := repo.Save(context.Background(), report); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}
	if report.ID != 88 {
		t.Fatalf("expected inserted id 88, got %d", report.ID)
	}

	rows := sqlmock.NewRows([]string{"id", "attendance_record_id", "report_type", "idempotency_key", "payload_json", "target_url", "external_record_id", "delete_record_id", "report_status", "response_code", "response_body", "notification_status", "notification_message_id", "notification_response_code", "notification_response_body", "notification_sent_at", "notification_retry_count", "reported_at", "retry_count"}).
		AddRow(uint64(88), uint64(55), "clock_in", report.IdempotencyKey, report.PayloadJSON, report.TargetURL, report.ExternalRecordID, report.DeleteRecordID, "pending", nil, "", "pending", "", nil, "", nil, uint32(0), reportedAt, uint32(1))
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, attendance_record_id, report_type, idempotency_key, payload_json, target_url, external_record_id, delete_record_id, report_status, response_code, response_body, notification_status, notification_message_id, notification_response_code, notification_response_body, notification_sent_at, notification_retry_count, reported_at, retry_count
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
	if found.ExternalRecordID != "flow_001" || found.DeleteRecordID != "flow_old_001" {
		t.Fatalf("expected report ids to round-trip, got %+v", found)
	}
	if found.NotificationStatus != "pending" {
		t.Fatalf("expected notification status to round-trip, got %+v", found)
	}

	listRows := sqlmock.NewRows([]string{"id", "attendance_record_id", "report_type", "idempotency_key", "payload_json", "target_url", "external_record_id", "delete_record_id", "report_status", "response_code", "response_body", "notification_status", "notification_message_id", "notification_response_code", "notification_response_body", "notification_sent_at", "notification_retry_count", "reported_at", "retry_count"}).
		AddRow(uint64(88), uint64(55), "clock_in", report.IdempotencyKey, report.PayloadJSON, report.TargetURL, report.ExternalRecordID, report.DeleteRecordID, "pending", nil, "", "pending", "", nil, "", nil, uint32(0), reportedAt, uint32(1))
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, attendance_record_id, report_type, idempotency_key, payload_json, target_url, external_record_id, delete_record_id, report_status, response_code, response_body, notification_status, notification_message_id, notification_response_code, notification_response_body, notification_sent_at, notification_retry_count, reported_at, retry_count
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

func TestMySQLReportRepositoryHandlesNullTextColumns(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewMySQLReportRepository(db)
	reportedAt := mustTime(t, "2026-03-21T09:00:00Z")
	rows := sqlmock.NewRows([]string{"id", "attendance_record_id", "report_type", "idempotency_key", "payload_json", "target_url", "external_record_id", "delete_record_id", "report_status", "response_code", "response_body", "notification_status", "notification_message_id", "notification_response_code", "notification_response_body", "notification_sent_at", "notification_retry_count", "reported_at", "retry_count"}).
		AddRow(uint64(88), uint64(55), "clock_in", "attendance-report/employee-42-2026-03-21/clock_in/2026-03-21T08:00:00Z/v2", nil, "http://example.test/report", nil, nil, "pending", nil, nil, nil, nil, nil, nil, nil, uint32(0), reportedAt, uint32(1))
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, attendance_record_id, report_type, idempotency_key, payload_json, target_url, external_record_id, delete_record_id, report_status, response_code, response_body, notification_status, notification_message_id, notification_response_code, notification_response_body, notification_sent_at, notification_retry_count, reported_at, retry_count
		FROM attendance_reports
		WHERE idempotency_key = ?
		LIMIT 1
	`))).WithArgs("attendance-report/employee-42-2026-03-21/clock_in/2026-03-21T08:00:00Z/v2").WillReturnRows(rows)

	found, err := repo.FindByIdempotencyKey(context.Background(), "attendance-report/employee-42-2026-03-21/clock_in/2026-03-21T08:00:00Z/v2")
	if err != nil {
		t.Fatalf("expected lookup to succeed, got %v", err)
	}
	if found.PayloadJSON != "" {
		t.Fatalf("expected null payload_json to become empty string, got %q", found.PayloadJSON)
	}
	if found.ResponseBody != "" {
		t.Fatalf("expected null response_body to become empty string, got %q", found.ResponseBody)
	}

	listRows := sqlmock.NewRows([]string{"id", "attendance_record_id", "report_type", "idempotency_key", "payload_json", "target_url", "external_record_id", "delete_record_id", "report_status", "response_code", "response_body", "notification_status", "notification_message_id", "notification_response_code", "notification_response_body", "notification_sent_at", "notification_retry_count", "reported_at", "retry_count"}).
		AddRow(uint64(88), uint64(55), "clock_in", "attendance-report/employee-42-2026-03-21/clock_in/2026-03-21T08:00:00Z/v2", nil, "http://example.test/report", nil, nil, "pending", nil, nil, nil, nil, nil, nil, nil, uint32(0), reportedAt, uint32(1))
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, attendance_record_id, report_type, idempotency_key, payload_json, target_url, external_record_id, delete_record_id, report_status, response_code, response_body, notification_status, notification_message_id, notification_response_code, notification_response_body, notification_sent_at, notification_retry_count, reported_at, retry_count
		FROM attendance_reports
		WHERE attendance_record_id = ?
		ORDER BY id DESC
	`))).WithArgs(int64(55)).WillReturnRows(listRows)

	got, err := repo.ListByAttendanceRecordID(context.Background(), 55)
	if err != nil {
		t.Fatalf("expected list to succeed, got %v", err)
	}
	if got[0].PayloadJSON != "" {
		t.Fatalf("expected null payload_json to become empty string, got %q", got[0].PayloadJSON)
	}
	if got[0].ResponseBody != "" {
		t.Fatalf("expected null response_body to become empty string, got %q", got[0].ResponseBody)
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

func TestMySQLSystemSettingRepositorySave(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewMySQLSystemSettingRepository(db)
	setting := &domain.SystemSetting{
		SettingKey:   "day_end_time",
		SettingValue: "22:00",
	}
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		INSERT INTO system_settings (
			setting_key,
			setting_value
		) VALUES (?, ?)
		ON DUPLICATE KEY UPDATE
			id = LAST_INSERT_ID(id),
			setting_value = VALUES(setting_value)
	`))).WithArgs("day_end_time", "22:00").
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.Save(context.Background(), setting); err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}
	if setting.ID != 1 {
		t.Fatalf("expected inserted id 1, got %d", setting.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all query expectations to be met, got %v", err)
	}
}
