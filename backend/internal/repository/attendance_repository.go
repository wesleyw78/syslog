package repository

import (
	"context"
	"database/sql"
	"time"

	"syslog/internal/domain"
)

type AttendanceRepository interface {
	FindByEmployeeAndDate(ctx context.Context, employeeID uint64, attendanceDate time.Time) (*domain.AttendanceRecord, error)
	Save(ctx context.Context, record *domain.AttendanceRecord) error
	ListByDateRange(ctx context.Context, from, to time.Time) ([]domain.AttendanceRecord, error)
}

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type MySQLAttendanceRepository struct {
	db sqlExecutor
}

func NewMySQLAttendanceRepository(db *sql.DB) *MySQLAttendanceRepository {
	return &MySQLAttendanceRepository{db: db}
}

func (r *MySQLAttendanceRepository) WithTx(tx *sql.Tx) AttendanceRepository {
	return &MySQLAttendanceRepository{db: tx}
}

func (r *MySQLAttendanceRepository) FindByEmployeeAndDate(ctx context.Context, employeeID uint64, attendanceDate time.Time) (*domain.AttendanceRecord, error) {
	const query = `
SELECT id, employee_id, attendance_date, first_connect_at, last_disconnect_at, clock_in_status, clock_out_status, exception_status, source_mode, version, last_calculated_at
FROM attendance_records
WHERE employee_id = ? AND attendance_date = ?
LIMIT 1`

	row := r.db.QueryRowContext(ctx, trimSQL(query), employeeID, attendanceDate)

	var record domain.AttendanceRecord
	var firstConnectAt sql.NullTime
	var lastDisconnectAt sql.NullTime
	var lastCalculatedAt sql.NullTime
	if err := row.Scan(
		&record.ID,
		&record.EmployeeID,
		&record.AttendanceDate,
		&firstConnectAt,
		&lastDisconnectAt,
		&record.ClockInStatus,
		&record.ClockOutStatus,
		&record.ExceptionStatus,
		&record.SourceMode,
		&record.Version,
		&lastCalculatedAt,
	); err != nil {
		return nil, err
	}

	record.FirstConnectAt = timeFromNullTime(firstConnectAt)
	record.LastDisconnectAt = timeFromNullTime(lastDisconnectAt)
	record.LastCalculatedAt = timeFromNullTime(lastCalculatedAt)
	return &record, nil
}

func (r *MySQLAttendanceRepository) Save(ctx context.Context, record *domain.AttendanceRecord) error {
	const query = `
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
	last_calculated_at = VALUES(last_calculated_at)`

	result, err := r.db.ExecContext(
		ctx,
		trimSQL(query),
		record.EmployeeID,
		record.AttendanceDate,
		nullableTime(record.FirstConnectAt),
		nullableTime(record.LastDisconnectAt),
		record.ClockInStatus,
		record.ClockOutStatus,
		record.ExceptionStatus,
		record.SourceMode,
		record.Version,
		nullableTime(record.LastCalculatedAt),
	)
	if err != nil {
		return err
	}

	id, err := parseInsertedID(result)
	if err != nil {
		return err
	}

	record.ID = id
	return nil
}

func (r *MySQLAttendanceRepository) ListByDateRange(ctx context.Context, from, to time.Time) ([]domain.AttendanceRecord, error) {
	const query = `
SELECT id, employee_id, attendance_date, first_connect_at, last_disconnect_at, clock_in_status, clock_out_status, exception_status, source_mode, version, last_calculated_at
FROM attendance_records
WHERE attendance_date BETWEEN ? AND ?
ORDER BY attendance_date DESC, employee_id ASC, id DESC`

	rows, err := r.db.QueryContext(ctx, trimSQL(query), from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]domain.AttendanceRecord, 0)
	for rows.Next() {
		var record domain.AttendanceRecord
		var firstConnectAt sql.NullTime
		var lastDisconnectAt sql.NullTime
		var lastCalculatedAt sql.NullTime
		if err := rows.Scan(
			&record.ID,
			&record.EmployeeID,
			&record.AttendanceDate,
			&firstConnectAt,
			&lastDisconnectAt,
			&record.ClockInStatus,
			&record.ClockOutStatus,
			&record.ExceptionStatus,
			&record.SourceMode,
			&record.Version,
			&lastCalculatedAt,
		); err != nil {
			return nil, err
		}

		record.FirstConnectAt = timeFromNullTime(firstConnectAt)
		record.LastDisconnectAt = timeFromNullTime(lastDisconnectAt)
		record.LastCalculatedAt = timeFromNullTime(lastCalculatedAt)
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}
