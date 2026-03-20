package repository

import (
	"context"
	"database/sql"

	"syslog/internal/domain"
)

type ReportRepository interface {
	FindByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.AttendanceReport, error)
	Save(ctx context.Context, report *domain.AttendanceReport) error
	ListByAttendanceRecordID(ctx context.Context, attendanceRecordID uint64) ([]domain.AttendanceReport, error)
}

type MySQLReportRepository struct {
	db *sql.DB
}

func NewMySQLReportRepository(db *sql.DB) *MySQLReportRepository {
	return &MySQLReportRepository{db: db}
}

func (r *MySQLReportRepository) FindByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.AttendanceReport, error) {
	const query = `
SELECT id, attendance_record_id, report_type, idempotency_key, payload_json, target_url, report_status, response_code, response_body, reported_at, retry_count
FROM attendance_reports
WHERE idempotency_key = ?
LIMIT 1`

	row := r.db.QueryRowContext(ctx, trimSQL(query), idempotencyKey)

	var report domain.AttendanceReport
	var responseCode sql.NullInt64
	var reportedAt sql.NullTime
	if err := row.Scan(
		&report.ID,
		&report.AttendanceRecordID,
		&report.ReportType,
		&report.IdempotencyKey,
		&report.PayloadJSON,
		&report.TargetURL,
		&report.ReportStatus,
		&responseCode,
		&report.ResponseBody,
		&reportedAt,
		&report.RetryCount,
	); err != nil {
		return nil, err
	}

	report.ResponseCode = intFromNullInt64(responseCode)
	report.ReportedAt = timeFromNullTime(reportedAt)
	return &report, nil
}

func (r *MySQLReportRepository) Save(ctx context.Context, report *domain.AttendanceReport) error {
	const query = `
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
	retry_count = VALUES(retry_count)`

	result, err := r.db.ExecContext(
		ctx,
		trimSQL(query),
		report.AttendanceRecordID,
		report.ReportType,
		report.IdempotencyKey,
		report.PayloadJSON,
		report.TargetURL,
		report.ReportStatus,
		nullableIntArg(report.ResponseCode),
		report.ResponseBody,
		nullableTime(report.ReportedAt),
		report.RetryCount,
	)
	if err != nil {
		return err
	}

	id, err := parseInsertedID(result)
	if err != nil {
		return err
	}

	report.ID = id
	return nil
}

func (r *MySQLReportRepository) ListByAttendanceRecordID(ctx context.Context, attendanceRecordID uint64) ([]domain.AttendanceReport, error) {
	const query = `
SELECT id, attendance_record_id, report_type, idempotency_key, payload_json, target_url, report_status, response_code, response_body, reported_at, retry_count
FROM attendance_reports
WHERE attendance_record_id = ?
ORDER BY id DESC`

	rows, err := r.db.QueryContext(ctx, trimSQL(query), attendanceRecordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reports := make([]domain.AttendanceReport, 0)
	for rows.Next() {
		var report domain.AttendanceReport
		var responseCode sql.NullInt64
		var reportedAt sql.NullTime
		if err := rows.Scan(
			&report.ID,
			&report.AttendanceRecordID,
			&report.ReportType,
			&report.IdempotencyKey,
			&report.PayloadJSON,
			&report.TargetURL,
			&report.ReportStatus,
			&responseCode,
			&report.ResponseBody,
			&reportedAt,
			&report.RetryCount,
		); err != nil {
			return nil, err
		}

		report.ResponseCode = intFromNullInt64(responseCode)
		report.ReportedAt = timeFromNullTime(reportedAt)
		reports = append(reports, report)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reports, nil
}
