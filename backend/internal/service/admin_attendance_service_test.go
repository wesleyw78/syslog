package service

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"syslog/internal/domain"
	"syslog/internal/repository"
)

type fakeAttendanceSettingsRepo struct {
	targetURL string
	keys      []string
}

func (f *fakeAttendanceSettingsRepo) GetByKey(_ context.Context, key string) (*domain.SystemSetting, error) {
	f.keys = append(f.keys, key)
	return &domain.SystemSetting{SettingKey: key, SettingValue: f.targetURL}, nil
}

func (f *fakeAttendanceSettingsRepo) List(context.Context) ([]domain.SystemSetting, error) {
	return nil, nil
}

func (f *fakeAttendanceSettingsRepo) Save(context.Context, *domain.SystemSetting) error {
	return nil
}

func (f *fakeAttendanceSettingsRepo) WithTx(*sql.Tx) repository.SystemSettingRepository {
	return f
}

func TestAttendanceAdminServiceCorrectAttendancePersistsRecordAndPendingReports(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	attendanceRepo := repository.NewMySQLAttendanceRepository(db)
	reportRepo := repository.NewMySQLReportRepository(db)
	settingsRepo := &fakeAttendanceSettingsRepo{targetURL: "http://example.test/report"}
	service := NewAttendanceAdminService(db, attendanceRepo, reportRepo, settingsRepo, NewReportService())

	loc := time.FixedZone("CST", 8*3600)
	attendanceDate := time.Date(2026, 3, 21, 0, 0, 0, 0, loc)
	existingFirst := time.Date(2026, 3, 21, 8, 0, 0, 0, loc)
	existingLast := time.Date(2026, 3, 21, 17, 0, 0, 0, loc)
	newFirst := time.Date(2026, 3, 21, 8, 15, 0, 0, loc)
	newLast := time.Date(2026, 3, 21, 17, 30, 0, 0, loc)
	recordID := uint64(55)

	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, employee_id, attendance_date, first_connect_at, last_disconnect_at, clock_in_status, clock_out_status, exception_status, source_mode, version, last_calculated_at
		FROM attendance_records
		WHERE id = ?
		LIMIT 1
	`))).
		WithArgs(int64(recordID)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "employee_id", "attendance_date", "first_connect_at", "last_disconnect_at", "clock_in_status", "clock_out_status", "exception_status", "source_mode", "version", "last_calculated_at"}).
			AddRow(recordID, uint64(42), attendanceDate, existingFirst, existingLast, "done", "done", "none", "syslog", uint32(2), nil))
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
		WithArgs(int64(42), attendanceDate, newFirst, newLast, "done", "done", "none", "manual", int64(3), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(int64(recordID), 1))
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
		WithArgs(int64(recordID), "clock_in", sqlmock.AnyArg(), sqlmock.AnyArg(), "http://example.test/report", "pending", nil, "", nil, int64(0)).
		WillReturnResult(sqlmock.NewResult(81, 1))
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
		WithArgs(int64(recordID), "clock_out", sqlmock.AnyArg(), sqlmock.AnyArg(), "http://example.test/report", "pending", nil, "", nil, int64(0)).
		WillReturnResult(sqlmock.NewResult(82, 1))
	mock.ExpectCommit()

	result, err := service.CorrectAttendance(context.Background(), recordID, AttendanceCorrectionInput{
		FirstConnectAt:   OptionalTimeField{Provided: true, Valid: true, Value: &newFirst},
		LastDisconnectAt: OptionalTimeField{Provided: true, Valid: true, Value: &newLast},
	})
	if err != nil {
		t.Fatalf("expected correction to succeed, got %v", err)
	}
	if result.Record.Version != 3 {
		t.Fatalf("expected version to increment, got %d", result.Record.Version)
	}
	if result.Record.SourceMode != "manual" {
		t.Fatalf("expected manual source mode, got %s", result.Record.SourceMode)
	}
	if result.Record.FirstConnectAt == nil || !result.Record.FirstConnectAt.Equal(newFirst) {
		t.Fatalf("expected first connect to be updated, got %+v", result.Record.FirstConnectAt)
	}
	if result.Record.LastDisconnectAt == nil || !result.Record.LastDisconnectAt.Equal(newLast) {
		t.Fatalf("expected last disconnect to be updated, got %+v", result.Record.LastDisconnectAt)
	}
	if len(result.Reports) != 2 {
		t.Fatalf("expected two pending reports, got %d", len(result.Reports))
	}
	if result.Reports[0].ReportStatus != "pending" || result.Reports[1].ReportStatus != "pending" {
		t.Fatalf("expected pending reports, got %+v", result.Reports)
	}
	if len(settingsRepo.keys) != 1 || settingsRepo.keys[0] != reportTargetURLSettingKey {
		t.Fatalf("expected report target lookup, got %+v", settingsRepo.keys)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all sql expectations to be met, got %v", err)
	}
}

func TestAttendanceAdminServiceCorrectAttendanceKeepsExistingLastDisconnectWhenOnlyFirstConnectProvided(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	attendanceRepo := repository.NewMySQLAttendanceRepository(db)
	reportRepo := repository.NewMySQLReportRepository(db)
	settingsRepo := &fakeAttendanceSettingsRepo{targetURL: "http://example.test/report"}
	service := NewAttendanceAdminService(db, attendanceRepo, reportRepo, settingsRepo, NewReportService())

	loc := time.FixedZone("CST", 8*3600)
	attendanceDate := time.Date(2026, 3, 21, 0, 0, 0, 0, loc)
	existingLast := time.Date(2026, 3, 21, 17, 0, 0, 0, loc)
	newFirst := time.Date(2026, 3, 21, 8, 15, 0, 0, loc)
	recordID := uint64(55)

	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, employee_id, attendance_date, first_connect_at, last_disconnect_at, clock_in_status, clock_out_status, exception_status, source_mode, version, last_calculated_at
		FROM attendance_records
		WHERE id = ?
		LIMIT 1
	`))).
		WithArgs(int64(recordID)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "employee_id", "attendance_date", "first_connect_at", "last_disconnect_at", "clock_in_status", "clock_out_status", "exception_status", "source_mode", "version", "last_calculated_at"}).
			AddRow(recordID, uint64(42), attendanceDate, nil, existingLast, "pending", "done", "none", "syslog", uint32(2), nil))
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
		WithArgs(int64(42), attendanceDate, newFirst, existingLast, "done", "done", "none", "manual", int64(3), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(int64(recordID), 1))
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
		WithArgs(int64(recordID), "clock_in", sqlmock.AnyArg(), sqlmock.AnyArg(), "http://example.test/report", "pending", nil, "", nil, int64(0)).
		WillReturnResult(sqlmock.NewResult(81, 1))
	mock.ExpectCommit()

	result, err := service.CorrectAttendance(context.Background(), recordID, AttendanceCorrectionInput{
		FirstConnectAt: OptionalTimeField{Provided: true, Valid: true, Value: &newFirst},
	})
	if err != nil {
		t.Fatalf("expected correction to succeed, got %v", err)
	}
	if result.Record.LastDisconnectAt == nil || !result.Record.LastDisconnectAt.Equal(existingLast) {
		t.Fatalf("expected last disconnect to remain unchanged, got %+v", result.Record.LastDisconnectAt)
	}
	if len(result.Reports) != 1 || result.Reports[0].ReportType != "clock_in" {
		t.Fatalf("expected only clock_in report, got %+v", result.Reports)
	}
	if result.Reports[0].TargetURL != "http://example.test/report" {
		t.Fatalf("expected report target url from settings, got %q", result.Reports[0].TargetURL)
	}
	if len(settingsRepo.keys) != 1 || settingsRepo.keys[0] != reportTargetURLSettingKey {
		t.Fatalf("expected report target lookup, got %+v", settingsRepo.keys)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all sql expectations to be met, got %v", err)
	}
}

func TestAttendanceAdminServiceCorrectAttendanceRollsBackWhenReportInsertFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	attendanceRepo := repository.NewMySQLAttendanceRepository(db)
	reportRepo := repository.NewMySQLReportRepository(db)
	settingsRepo := &fakeAttendanceSettingsRepo{targetURL: "http://example.test/report"}
	service := NewAttendanceAdminService(db, attendanceRepo, reportRepo, settingsRepo, NewReportService())

	loc := time.FixedZone("CST", 8*3600)
	attendanceDate := time.Date(2026, 3, 21, 0, 0, 0, 0, loc)
	first := time.Date(2026, 3, 21, 8, 10, 0, 0, loc)
	last := time.Date(2026, 3, 21, 18, 0, 0, 0, loc)
	recordID := uint64(55)

	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, employee_id, attendance_date, first_connect_at, last_disconnect_at, clock_in_status, clock_out_status, exception_status, source_mode, version, last_calculated_at
		FROM attendance_records
		WHERE id = ?
		LIMIT 1
	`))).
		WithArgs(int64(recordID)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "employee_id", "attendance_date", "first_connect_at", "last_disconnect_at", "clock_in_status", "clock_out_status", "exception_status", "source_mode", "version", "last_calculated_at"}).
			AddRow(recordID, uint64(42), attendanceDate, nil, nil, "pending", "pending", "none", "syslog", uint32(2), nil))
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
		WithArgs(int64(42), attendanceDate, first, last, "done", "done", "none", "manual", int64(3), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(int64(recordID), 1))
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
		WithArgs(int64(recordID), "clock_in", sqlmock.AnyArg(), sqlmock.AnyArg(), "http://example.test/report", "pending", nil, "", nil, int64(0)).
		WillReturnResult(sqlmock.NewResult(81, 1))
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
		WithArgs(int64(recordID), "clock_out", sqlmock.AnyArg(), sqlmock.AnyArg(), "http://example.test/report", "pending", nil, "", nil, int64(0)).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	_, err = service.CorrectAttendance(context.Background(), recordID, AttendanceCorrectionInput{
		FirstConnectAt:   OptionalTimeField{Provided: true, Valid: true, Value: &first},
		LastDisconnectAt: OptionalTimeField{Provided: true, Valid: true, Value: &last},
	})
	if err == nil {
		t.Fatal("expected report insert failure to surface")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected rollback expectations to be met, got %v", err)
	}
}

func TestAttendanceAdminServiceCorrectAttendanceNoopPatchDoesNotPersistOrCreateReports(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	attendanceRepo := repository.NewMySQLAttendanceRepository(db)
	reportRepo := repository.NewMySQLReportRepository(db)
	settingsRepo := &fakeAttendanceSettingsRepo{targetURL: "http://example.test/report"}
	service := NewAttendanceAdminService(db, attendanceRepo, reportRepo, settingsRepo, NewReportService())

	loc := time.FixedZone("CST", 8*3600)
	attendanceDate := time.Date(2026, 3, 21, 0, 0, 0, 0, loc)
	first := time.Date(2026, 3, 21, 8, 10, 0, 0, loc)
	last := time.Date(2026, 3, 21, 18, 0, 0, 0, loc)
	recordID := uint64(55)

	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, employee_id, attendance_date, first_connect_at, last_disconnect_at, clock_in_status, clock_out_status, exception_status, source_mode, version, last_calculated_at
		FROM attendance_records
		WHERE id = ?
		LIMIT 1
	`))).
		WithArgs(int64(recordID)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "employee_id", "attendance_date", "first_connect_at", "last_disconnect_at", "clock_in_status", "clock_out_status", "exception_status", "source_mode", "version", "last_calculated_at"}).
			AddRow(recordID, uint64(42), attendanceDate, first, last, "done", "done", "none", "manual", uint32(3), nil))

	result, err := service.CorrectAttendance(context.Background(), recordID, AttendanceCorrectionInput{
		FirstConnectAt:   OptionalTimeField{Provided: true, Valid: true, Value: &first},
		LastDisconnectAt: OptionalTimeField{Provided: true, Valid: true, Value: &last},
	})
	if err != nil {
		t.Fatalf("expected noop correction to succeed, got %v", err)
	}
	if result.Record.Version != 3 {
		t.Fatalf("expected version to remain unchanged, got %d", result.Record.Version)
	}
	if len(result.Reports) != 0 {
		t.Fatalf("expected no reports for noop patch, got %+v", result.Reports)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected no persistence expectations beyond lookup, got %v", err)
	}
}

func TestAttendanceAdminServiceCorrectAttendanceClearPatchGeneratesClearReport(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	attendanceRepo := repository.NewMySQLAttendanceRepository(db)
	reportRepo := repository.NewMySQLReportRepository(db)
	settingsRepo := &fakeAttendanceSettingsRepo{targetURL: "http://example.test/report"}
	service := NewAttendanceAdminService(db, attendanceRepo, reportRepo, settingsRepo, NewReportService())

	loc := time.FixedZone("CST", 8*3600)
	attendanceDate := time.Date(2026, 3, 21, 0, 0, 0, 0, loc)
	existingFirst := time.Date(2026, 3, 21, 8, 10, 0, 0, loc)
	recordID := uint64(55)

	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, employee_id, attendance_date, first_connect_at, last_disconnect_at, clock_in_status, clock_out_status, exception_status, source_mode, version, last_calculated_at
		FROM attendance_records
		WHERE id = ?
		LIMIT 1
	`))).
		WithArgs(int64(recordID)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "employee_id", "attendance_date", "first_connect_at", "last_disconnect_at", "clock_in_status", "clock_out_status", "exception_status", "source_mode", "version", "last_calculated_at"}).
			AddRow(recordID, uint64(42), attendanceDate, existingFirst, nil, "done", "pending", "none", "syslog", uint32(2), nil))
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
		WithArgs(int64(42), attendanceDate, nil, nil, "pending", "missing", "missing_disconnect", "manual", int64(3), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(int64(recordID), 1))
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
		WithArgs(int64(recordID), "clock_in", sqlmock.AnyArg(), sqlmock.AnyArg(), "http://example.test/report", "pending", nil, "", nil, int64(0)).
		WillReturnResult(sqlmock.NewResult(81, 1))
	mock.ExpectCommit()

	result, err := service.CorrectAttendance(context.Background(), recordID, AttendanceCorrectionInput{
		FirstConnectAt:   OptionalTimeField{Provided: true, Valid: false},
		LastDisconnectAt: OptionalTimeField{},
	})
	if err != nil {
		t.Fatalf("expected clear patch to succeed, got %v", err)
	}
	if result.Record.Version != 3 {
		t.Fatalf("expected version to increment, got %d", result.Record.Version)
	}
	if result.Record.FirstConnectAt != nil {
		t.Fatalf("expected first connect to be cleared, got %+v", result.Record.FirstConnectAt)
	}
	if len(result.Reports) != 1 || result.Reports[0].ReportType != "clock_in" {
		t.Fatalf("expected clear patch to generate clock_in clear report, got %+v", result.Reports)
	}
	if result.Reports[0].TargetURL != "http://example.test/report" {
		t.Fatalf("expected clear report target url from settings, got %q", result.Reports[0].TargetURL)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected clear patch expectations to be met, got %v", err)
	}
}

func TestAttendanceAdminServiceCorrectAttendanceNullToNilDoesNotIncrementVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	attendanceRepo := repository.NewMySQLAttendanceRepository(db)
	reportRepo := repository.NewMySQLReportRepository(db)
	settingsRepo := &fakeAttendanceSettingsRepo{targetURL: "http://example.test/report"}
	service := NewAttendanceAdminService(db, attendanceRepo, reportRepo, settingsRepo, NewReportService())

	loc := time.FixedZone("CST", 8*3600)
	attendanceDate := time.Date(2026, 3, 21, 0, 0, 0, 0, loc)
	recordID := uint64(55)

	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, employee_id, attendance_date, first_connect_at, last_disconnect_at, clock_in_status, clock_out_status, exception_status, source_mode, version, last_calculated_at
		FROM attendance_records
		WHERE id = ?
		LIMIT 1
	`))).
		WithArgs(int64(recordID)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "employee_id", "attendance_date", "first_connect_at", "last_disconnect_at", "clock_in_status", "clock_out_status", "exception_status", "source_mode", "version", "last_calculated_at"}).
			AddRow(recordID, uint64(42), attendanceDate, nil, nil, "pending", "pending", "none", "syslog", uint32(2), nil))

	result, err := service.CorrectAttendance(context.Background(), recordID, AttendanceCorrectionInput{
		FirstConnectAt: OptionalTimeField{Provided: true, Valid: false},
	})
	if err != nil {
		t.Fatalf("expected nil-to-nil correction to succeed, got %v", err)
	}
	if result.Record.Version != 2 {
		t.Fatalf("expected version to remain unchanged, got %d", result.Record.Version)
	}
	if len(result.Reports) != 0 {
		t.Fatalf("expected no reports for nil-to-nil correction, got %+v", result.Reports)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected noop nil-to-nil expectations to be met, got %v", err)
	}
}

var _ repository.AttendanceRepository = (*repository.MySQLAttendanceRepository)(nil)
var _ repository.ReportRepository = (*repository.MySQLReportRepository)(nil)
var _ repository.SystemSettingRepository = (*fakeAttendanceSettingsRepo)(nil)
var _ = domain.AttendanceRecord{}
