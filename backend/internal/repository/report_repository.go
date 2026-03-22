package repository

import (
	"context"
	"database/sql"

	"syslog/internal/domain"
)

type ReportRepository interface {
	FindByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.AttendanceReport, error)
	FindLatestSuccessfulByAttendanceRecordAndType(ctx context.Context, attendanceRecordID uint64, reportType string) (*domain.AttendanceReport, error)
	Save(ctx context.Context, report *domain.AttendanceReport) error
	ListDispatchable(ctx context.Context, limit int, retryLimit uint32) ([]domain.AttendanceReport, error)
	ListNotificationDispatchable(ctx context.Context, limit int, retryLimit uint32) ([]domain.AttendanceReport, error)
	ListByAttendanceRecordID(ctx context.Context, attendanceRecordID uint64) ([]domain.AttendanceReport, error)
}

type MySQLReportRepository struct {
	db sqlExecutor
}

func NewMySQLReportRepository(db *sql.DB) *MySQLReportRepository {
	return &MySQLReportRepository{db: db}
}

func (r *MySQLReportRepository) WithTx(tx *sql.Tx) ReportRepository {
	return &MySQLReportRepository{db: tx}
}

func (r *MySQLReportRepository) FindByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.AttendanceReport, error) {
	const query = `
SELECT id, attendance_record_id, report_type, idempotency_key, payload_json, target_url, external_record_id, delete_record_id, report_status, response_code, response_body, notification_status, notification_message_id, notification_response_code, notification_response_body, notification_sent_at, notification_retry_count, reported_at, retry_count
FROM attendance_reports
WHERE idempotency_key = ?
LIMIT 1`

	row := r.db.QueryRowContext(ctx, trimSQL(query), idempotencyKey)

	var report domain.AttendanceReport
	var responseCode sql.NullInt64
	var reportedAt sql.NullTime
	var payloadJSON sql.NullString
	var externalRecordID sql.NullString
	var deleteRecordID sql.NullString
	var responseBody sql.NullString
	var notificationStatus sql.NullString
	var notificationMessageID sql.NullString
	var notificationResponseCode sql.NullInt64
	var notificationResponseBody sql.NullString
	var notificationSentAt sql.NullTime
	if err := row.Scan(
		&report.ID,
		&report.AttendanceRecordID,
		&report.ReportType,
		&report.IdempotencyKey,
		&payloadJSON,
		&report.TargetURL,
		&externalRecordID,
		&deleteRecordID,
		&report.ReportStatus,
		&responseCode,
		&responseBody,
		&notificationStatus,
		&notificationMessageID,
		&notificationResponseCode,
		&notificationResponseBody,
		&notificationSentAt,
		&report.NotificationRetryCount,
		&reportedAt,
		&report.RetryCount,
	); err != nil {
		return nil, err
	}

	report.PayloadJSON = stringFromNullString(payloadJSON)
	report.ExternalRecordID = stringFromNullString(externalRecordID)
	report.DeleteRecordID = stringFromNullString(deleteRecordID)
	report.ResponseBody = stringFromNullString(responseBody)
	report.ResponseCode = intFromNullInt64(responseCode)
	report.NotificationStatus = stringFromNullString(notificationStatus)
	report.NotificationMessageID = stringFromNullString(notificationMessageID)
	report.NotificationResponseCode = intFromNullInt64(notificationResponseCode)
	report.NotificationResponseBody = stringFromNullString(notificationResponseBody)
	report.NotificationSentAt = timeFromNullTime(notificationSentAt)
	report.ReportedAt = timeFromNullTime(reportedAt)
	return &report, nil
}

func (r *MySQLReportRepository) FindLatestSuccessfulByAttendanceRecordAndType(ctx context.Context, attendanceRecordID uint64, reportType string) (*domain.AttendanceReport, error) {
	const query = `
SELECT id, attendance_record_id, report_type, idempotency_key, payload_json, target_url, external_record_id, delete_record_id, report_status, response_code, response_body, notification_status, notification_message_id, notification_response_code, notification_response_body, notification_sent_at, notification_retry_count, reported_at, retry_count
FROM attendance_reports
WHERE attendance_record_id = ?
  AND report_type = ?
  AND report_status = 'success'
  AND external_record_id <> ''
ORDER BY id DESC
LIMIT 1`

	row := r.db.QueryRowContext(ctx, trimSQL(query), attendanceRecordID, reportType)

	var report domain.AttendanceReport
	var responseCode sql.NullInt64
	var reportedAt sql.NullTime
	var payloadJSON sql.NullString
	var externalRecordID sql.NullString
	var deleteRecordID sql.NullString
	var responseBody sql.NullString
	var notificationStatus sql.NullString
	var notificationMessageID sql.NullString
	var notificationResponseCode sql.NullInt64
	var notificationResponseBody sql.NullString
	var notificationSentAt sql.NullTime
	if err := row.Scan(
		&report.ID,
		&report.AttendanceRecordID,
		&report.ReportType,
		&report.IdempotencyKey,
		&payloadJSON,
		&report.TargetURL,
		&externalRecordID,
		&deleteRecordID,
		&report.ReportStatus,
		&responseCode,
		&responseBody,
		&notificationStatus,
		&notificationMessageID,
		&notificationResponseCode,
		&notificationResponseBody,
		&notificationSentAt,
		&report.NotificationRetryCount,
		&reportedAt,
		&report.RetryCount,
	); err != nil {
		return nil, err
	}

	report.PayloadJSON = stringFromNullString(payloadJSON)
	report.ExternalRecordID = stringFromNullString(externalRecordID)
	report.DeleteRecordID = stringFromNullString(deleteRecordID)
	report.ResponseBody = stringFromNullString(responseBody)
	report.ResponseCode = intFromNullInt64(responseCode)
	report.NotificationStatus = stringFromNullString(notificationStatus)
	report.NotificationMessageID = stringFromNullString(notificationMessageID)
	report.NotificationResponseCode = intFromNullInt64(notificationResponseCode)
	report.NotificationResponseBody = stringFromNullString(notificationResponseBody)
	report.NotificationSentAt = timeFromNullTime(notificationSentAt)
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
	retry_count = VALUES(retry_count)`

	result, err := r.db.ExecContext(
		ctx,
		trimSQL(query),
		report.AttendanceRecordID,
		report.ReportType,
		report.IdempotencyKey,
		report.PayloadJSON,
		report.TargetURL,
		report.ExternalRecordID,
		report.DeleteRecordID,
		report.ReportStatus,
		nullableIntArg(report.ResponseCode),
		report.ResponseBody,
		report.NotificationStatus,
		report.NotificationMessageID,
		nullableIntArg(report.NotificationResponseCode),
		report.NotificationResponseBody,
		nullableTime(report.NotificationSentAt),
		report.NotificationRetryCount,
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
SELECT id, attendance_record_id, report_type, idempotency_key, payload_json, target_url, external_record_id, delete_record_id, report_status, response_code, response_body, notification_status, notification_message_id, notification_response_code, notification_response_body, notification_sent_at, notification_retry_count, reported_at, retry_count
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
		var payloadJSON sql.NullString
		var externalRecordID sql.NullString
		var deleteRecordID sql.NullString
		var responseBody sql.NullString
		var notificationStatus sql.NullString
		var notificationMessageID sql.NullString
		var notificationResponseCode sql.NullInt64
		var notificationResponseBody sql.NullString
		var notificationSentAt sql.NullTime
		if err := rows.Scan(
			&report.ID,
			&report.AttendanceRecordID,
			&report.ReportType,
			&report.IdempotencyKey,
			&payloadJSON,
			&report.TargetURL,
			&externalRecordID,
			&deleteRecordID,
			&report.ReportStatus,
			&responseCode,
			&responseBody,
			&notificationStatus,
			&notificationMessageID,
			&notificationResponseCode,
			&notificationResponseBody,
			&notificationSentAt,
			&report.NotificationRetryCount,
			&reportedAt,
			&report.RetryCount,
		); err != nil {
			return nil, err
		}

		report.PayloadJSON = stringFromNullString(payloadJSON)
		report.ExternalRecordID = stringFromNullString(externalRecordID)
		report.DeleteRecordID = stringFromNullString(deleteRecordID)
		report.ResponseBody = stringFromNullString(responseBody)
		report.ResponseCode = intFromNullInt64(responseCode)
		report.NotificationStatus = stringFromNullString(notificationStatus)
		report.NotificationMessageID = stringFromNullString(notificationMessageID)
		report.NotificationResponseCode = intFromNullInt64(notificationResponseCode)
		report.NotificationResponseBody = stringFromNullString(notificationResponseBody)
		report.NotificationSentAt = timeFromNullTime(notificationSentAt)
		report.ReportedAt = timeFromNullTime(reportedAt)
		reports = append(reports, report)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reports, nil
}

func (r *MySQLReportRepository) ListDispatchable(ctx context.Context, limit int, retryLimit uint32) ([]domain.AttendanceReport, error) {
	const query = `
SELECT id, attendance_record_id, report_type, idempotency_key, payload_json, target_url, external_record_id, delete_record_id, report_status, response_code, response_body, notification_status, notification_message_id, notification_response_code, notification_response_body, notification_sent_at, notification_retry_count, reported_at, retry_count
FROM attendance_reports
WHERE report_status IN ('pending', 'failed')
  AND retry_count < ?
ORDER BY id ASC
LIMIT ?`

	rows, err := r.db.QueryContext(ctx, trimSQL(query), retryLimit, limitOrDefault(limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reports := make([]domain.AttendanceReport, 0)
	for rows.Next() {
		var report domain.AttendanceReport
		var responseCode sql.NullInt64
		var reportedAt sql.NullTime
		var payloadJSON sql.NullString
		var externalRecordID sql.NullString
		var deleteRecordID sql.NullString
		var responseBody sql.NullString
		var notificationStatus sql.NullString
		var notificationMessageID sql.NullString
		var notificationResponseCode sql.NullInt64
		var notificationResponseBody sql.NullString
		var notificationSentAt sql.NullTime
		if err := rows.Scan(
			&report.ID,
			&report.AttendanceRecordID,
			&report.ReportType,
			&report.IdempotencyKey,
			&payloadJSON,
			&report.TargetURL,
			&externalRecordID,
			&deleteRecordID,
			&report.ReportStatus,
			&responseCode,
			&responseBody,
			&notificationStatus,
			&notificationMessageID,
			&notificationResponseCode,
			&notificationResponseBody,
			&notificationSentAt,
			&report.NotificationRetryCount,
			&reportedAt,
			&report.RetryCount,
		); err != nil {
			return nil, err
		}

		report.PayloadJSON = stringFromNullString(payloadJSON)
		report.ExternalRecordID = stringFromNullString(externalRecordID)
		report.DeleteRecordID = stringFromNullString(deleteRecordID)
		report.ResponseBody = stringFromNullString(responseBody)
		report.ResponseCode = intFromNullInt64(responseCode)
		report.NotificationStatus = stringFromNullString(notificationStatus)
		report.NotificationMessageID = stringFromNullString(notificationMessageID)
		report.NotificationResponseCode = intFromNullInt64(notificationResponseCode)
		report.NotificationResponseBody = stringFromNullString(notificationResponseBody)
		report.NotificationSentAt = timeFromNullTime(notificationSentAt)
		report.ReportedAt = timeFromNullTime(reportedAt)
		reports = append(reports, report)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reports, nil
}

func (r *MySQLReportRepository) ListNotificationDispatchable(ctx context.Context, limit int, retryLimit uint32) ([]domain.AttendanceReport, error) {
	const query = `
SELECT id, attendance_record_id, report_type, idempotency_key, payload_json, target_url, external_record_id, delete_record_id, report_status, response_code, response_body, notification_status, notification_message_id, notification_response_code, notification_response_body, notification_sent_at, notification_retry_count, reported_at, retry_count
FROM attendance_reports
WHERE report_status = 'success'
  AND notification_status IN ('pending', 'failed')
  AND notification_retry_count < ?
ORDER BY id ASC
LIMIT ?`

	rows, err := r.db.QueryContext(ctx, trimSQL(query), retryLimit, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reports := make([]domain.AttendanceReport, 0)
	for rows.Next() {
		var report domain.AttendanceReport
		var responseCode sql.NullInt64
		var reportedAt sql.NullTime
		var payloadJSON sql.NullString
		var externalRecordID sql.NullString
		var deleteRecordID sql.NullString
		var responseBody sql.NullString
		var notificationStatus sql.NullString
		var notificationMessageID sql.NullString
		var notificationResponseCode sql.NullInt64
		var notificationResponseBody sql.NullString
		var notificationSentAt sql.NullTime
		if err := rows.Scan(
			&report.ID,
			&report.AttendanceRecordID,
			&report.ReportType,
			&report.IdempotencyKey,
			&payloadJSON,
			&report.TargetURL,
			&externalRecordID,
			&deleteRecordID,
			&report.ReportStatus,
			&responseCode,
			&responseBody,
			&notificationStatus,
			&notificationMessageID,
			&notificationResponseCode,
			&notificationResponseBody,
			&notificationSentAt,
			&report.NotificationRetryCount,
			&reportedAt,
			&report.RetryCount,
		); err != nil {
			return nil, err
		}

		report.PayloadJSON = stringFromNullString(payloadJSON)
		report.ExternalRecordID = stringFromNullString(externalRecordID)
		report.DeleteRecordID = stringFromNullString(deleteRecordID)
		report.ResponseBody = stringFromNullString(responseBody)
		report.ResponseCode = intFromNullInt64(responseCode)
		report.NotificationStatus = stringFromNullString(notificationStatus)
		report.NotificationMessageID = stringFromNullString(notificationMessageID)
		report.NotificationResponseCode = intFromNullInt64(notificationResponseCode)
		report.NotificationResponseBody = stringFromNullString(notificationResponseBody)
		report.NotificationSentAt = timeFromNullTime(notificationSentAt)
		report.ReportedAt = timeFromNullTime(reportedAt)
		reports = append(reports, report)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reports, nil
}
