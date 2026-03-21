package service

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"syslog/internal/repository"
)

func TestEmployeeAdminServiceCreatePersistsEmployeeAndDevices(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	repo := repository.NewMySQLEmployeeRepository(db)
	service := NewEmployeeAdminService(db, repo)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		INSERT INTO employees (
			employee_no,
			system_no,
			name,
			status
		) VALUES (?, ?, ?, ?)
	`))).
		WithArgs("EMP-001", "SYS-001", "Alice", "active").
		WillReturnResult(sqlmock.NewResult(11, 1))
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		DELETE FROM employee_devices
		WHERE employee_id = ?
	`))).
		WithArgs(int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		INSERT INTO employee_devices (
			employee_id,
			mac_address,
			device_label,
			status
		) VALUES (?, ?, ?, ?)
	`))).
		WithArgs(int64(11), "aa:bb:cc:dd:ee:ff", "iPhone", "active").
		WillReturnResult(sqlmock.NewResult(21, 1))
	mock.ExpectCommit()
	now := time.Date(2026, 3, 21, 8, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT e.id, e.employee_no, e.system_no, e.name, e.status, e.created_at, e.updated_at,
		       d.id, d.mac_address, d.device_label, d.status, d.created_at, d.updated_at
		FROM employees e
		LEFT JOIN employee_devices d ON d.employee_id = e.id
		WHERE e.id = ?
		ORDER BY d.id ASC
	`))).
		WithArgs(uint64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "employee_no", "system_no", "name", "status", "created_at", "updated_at", "device_id", "mac_address", "device_label", "device_status", "device_created_at", "device_updated_at"}).
			AddRow(uint64(11), "EMP-001", "SYS-001", "Alice", "active", now, now, uint64(21), "aa:bb:cc:dd:ee:ff", "iPhone", "active", now, now))

	got, err := service.CreateEmployee(context.Background(), EmployeeWriteInput{
		EmployeeNo: "EMP-001",
		SystemNo:   "SYS-001",
		Name:       "Alice",
		Status:     "active",
		Devices: []EmployeeDeviceInput{
			{
				MacAddress:  "AA:BB:CC:DD:EE:FF",
				DeviceLabel: "iPhone",
				Status:      "active",
			},
		},
	})
	if err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}
	if got.ID != 11 {
		t.Fatalf("expected inserted employee id 11, got %d", got.ID)
	}
	if len(got.Devices) != 1 || got.Devices[0].ID != 21 || got.Devices[0].CreatedAt.IsZero() || got.Devices[0].UpdatedAt.IsZero() {
		t.Fatalf("expected normalized device list, got %+v", got.Devices)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all sql expectations to be met, got %v", err)
	}
}

func TestEmployeeAdminServiceUpdateReplacesDevices(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	repo := repository.NewMySQLEmployeeRepository(db)
	service := NewEmployeeAdminService(db, repo)

	now := time.Date(2026, 3, 21, 8, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT e.id, e.employee_no, e.system_no, e.name, e.status, e.created_at, e.updated_at,
		       d.id, d.mac_address, d.device_label, d.status, d.created_at, d.updated_at
		FROM employees e
		LEFT JOIN employee_devices d ON d.employee_id = e.id
		WHERE e.id = ?
		ORDER BY d.id ASC
	`))).
		WithArgs(uint64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "employee_no", "system_no", "name", "status", "created_at", "updated_at", "device_id", "mac_address", "device_label", "device_status", "device_created_at", "device_updated_at"}).
			AddRow(uint64(11), "EMP-002", "SYS-002", "Bob", "active", now, now, uint64(22), "11:22:33:44:55:66", "Laptop", "active", now, now).
			AddRow(uint64(11), "EMP-002", "SYS-002", "Bob", "active", now, now, uint64(23), "aa:bb:cc:dd:ee:11", "Tablet", "disabled", now, now))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		UPDATE employees
		SET employee_no = ?, system_no = ?, name = ?, status = ?
		WHERE id = ?
	`))).
		WithArgs("EMP-002", "SYS-002", "Bob", "active", int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		DELETE FROM employee_devices
		WHERE employee_id = ?
	`))).
		WithArgs(int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		INSERT INTO employee_devices (
			employee_id,
			mac_address,
			device_label,
			status
		) VALUES (?, ?, ?, ?)
	`))).
		WithArgs(int64(11), "11:22:33:44:55:66", "Laptop", "active").
		WillReturnResult(sqlmock.NewResult(22, 1))
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		INSERT INTO employee_devices (
			employee_id,
			mac_address,
			device_label,
			status
		) VALUES (?, ?, ?, ?)
	`))).
		WithArgs(int64(11), "aa:bb:cc:dd:ee:11", "Tablet", "disabled").
		WillReturnResult(sqlmock.NewResult(23, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT e.id, e.employee_no, e.system_no, e.name, e.status, e.created_at, e.updated_at,
		       d.id, d.mac_address, d.device_label, d.status, d.created_at, d.updated_at
		FROM employees e
		LEFT JOIN employee_devices d ON d.employee_id = e.id
		WHERE e.id = ?
		ORDER BY d.id ASC
	`))).
		WithArgs(uint64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "employee_no", "system_no", "name", "status", "created_at", "updated_at", "device_id", "mac_address", "device_label", "device_status", "device_created_at", "device_updated_at"}).
			AddRow(uint64(11), "EMP-002", "SYS-002", "Bob", "active", now, now, uint64(22), "11:22:33:44:55:66", "Laptop", "active", now, now).
			AddRow(uint64(11), "EMP-002", "SYS-002", "Bob", "active", now, now, uint64(23), "aa:bb:cc:dd:ee:11", "Tablet", "disabled", now, now))

	got, err := service.UpdateEmployee(context.Background(), 11, EmployeeWriteInput{
		EmployeeNo: "EMP-002",
		SystemNo:   "SYS-002",
		Name:       "Bob",
		Status:     "active",
		Devices: []EmployeeDeviceInput{
			{
				MacAddress:  "11:22:33:44:55:66",
				DeviceLabel: "Laptop",
				Status:      "active",
			},
			{
				MacAddress:  "AA:BB:CC:DD:EE:11",
				DeviceLabel: "Tablet",
				Status:      "disabled",
			},
		},
	})
	if err != nil {
		t.Fatalf("expected update to succeed, got %v", err)
	}
	if got.ID != 11 {
		t.Fatalf("expected updated employee id 11, got %d", got.ID)
	}
	if len(got.Devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(got.Devices))
	}
	if got.Devices[0].ID != 22 || got.Devices[1].ID != 23 {
		t.Fatalf("expected real device metadata, got %+v", got.Devices)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all sql expectations to be met, got %v", err)
	}
}

func TestEmployeeAdminServiceUpdateReplacesDevicesWhenMainFieldsUnchanged(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	repo := repository.NewMySQLEmployeeRepository(db)
	service := NewEmployeeAdminService(db, repo)

	now := time.Date(2026, 3, 21, 8, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT e.id, e.employee_no, e.system_no, e.name, e.status, e.created_at, e.updated_at,
		       d.id, d.mac_address, d.device_label, d.status, d.created_at, d.updated_at
		FROM employees e
		LEFT JOIN employee_devices d ON d.employee_id = e.id
		WHERE e.id = ?
		ORDER BY d.id ASC
	`))).
		WithArgs(uint64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "employee_no", "system_no", "name", "status", "created_at", "updated_at", "device_id", "mac_address", "device_label", "device_status", "device_created_at", "device_updated_at"}).
			AddRow(uint64(11), "EMP-001", "SYS-001", "Alice", "active", now, now, uint64(21), "aa:bb:cc:dd:ee:ff", "Phone", "active", now, now))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		UPDATE employees
		SET employee_no = ?, system_no = ?, name = ?, status = ?
		WHERE id = ?
	`))).
		WithArgs("EMP-001", "SYS-001", "Alice", "active", int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		DELETE FROM employee_devices
		WHERE employee_id = ?
	`))).
		WithArgs(int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		INSERT INTO employee_devices (
			employee_id,
			mac_address,
			device_label,
			status
		) VALUES (?, ?, ?, ?)
	`))).
		WithArgs(int64(11), "11:22:33:44:55:66", "Laptop", "active").
		WillReturnResult(sqlmock.NewResult(22, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT e.id, e.employee_no, e.system_no, e.name, e.status, e.created_at, e.updated_at,
		       d.id, d.mac_address, d.device_label, d.status, d.created_at, d.updated_at
		FROM employees e
		LEFT JOIN employee_devices d ON d.employee_id = e.id
		WHERE e.id = ?
		ORDER BY d.id ASC
	`))).
		WithArgs(uint64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "employee_no", "system_no", "name", "status", "created_at", "updated_at", "device_id", "mac_address", "device_label", "device_status", "device_created_at", "device_updated_at"}).
			AddRow(uint64(11), "EMP-001", "SYS-001", "Alice", "active", now, now, uint64(22), "11:22:33:44:55:66", "Laptop", "active", now, now))

	got, err := service.UpdateEmployee(context.Background(), 11, EmployeeWriteInput{
		EmployeeNo: "EMP-001",
		SystemNo:   "SYS-001",
		Name:       "Alice",
		Status:     "active",
		Devices: []EmployeeDeviceInput{
			{
				MacAddress:  "11:22:33:44:55:66",
				DeviceLabel: "Laptop",
				Status:      "active",
			},
		},
	})
	if err != nil {
		t.Fatalf("expected device-only update to succeed, got %v", err)
	}
	if len(got.Devices) != 1 || got.Devices[0].ID != 22 {
		t.Fatalf("expected replaced devices to persist, got %+v", got.Devices)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all sql expectations to be met, got %v", err)
	}
}

func TestEmployeeAdminServiceDisableDisablesEmployeeAndDevices(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	repo := repository.NewMySQLEmployeeRepository(db)
	service := NewEmployeeAdminService(db, repo)

	now := time.Date(2026, 3, 21, 8, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT e.id, e.employee_no, e.system_no, e.name, e.status, e.created_at, e.updated_at,
		       d.id, d.mac_address, d.device_label, d.status, d.created_at, d.updated_at
		FROM employees e
		LEFT JOIN employee_devices d ON d.employee_id = e.id
		WHERE e.id = ?
		ORDER BY d.id ASC
	`))).
		WithArgs(uint64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "employee_no", "system_no", "name", "status", "created_at", "updated_at", "device_id", "mac_address", "device_label", "device_status", "device_created_at", "device_updated_at"}).
			AddRow(uint64(11), "EMP-001", "SYS-001", "Alice", "active", now, now, uint64(21), "aa:bb:cc:dd:ee:ff", "iPhone", "active", now, now))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		UPDATE employees
		SET status = ?
		WHERE id = ?
	`))).
		WithArgs("disabled", int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		UPDATE employee_devices
		SET status = ?
		WHERE employee_id = ?
	`))).
		WithArgs("disabled", int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT e.id, e.employee_no, e.system_no, e.name, e.status, e.created_at, e.updated_at,
		       d.id, d.mac_address, d.device_label, d.status, d.created_at, d.updated_at
		FROM employees e
		LEFT JOIN employee_devices d ON d.employee_id = e.id
		WHERE e.id = ?
		ORDER BY d.id ASC
	`))).
		WithArgs(uint64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "employee_no", "system_no", "name", "status", "created_at", "updated_at", "device_id", "mac_address", "device_label", "device_status", "device_created_at", "device_updated_at"}).
			AddRow(uint64(11), "EMP-001", "SYS-001", "Alice", "disabled", now, now, uint64(21), "aa:bb:cc:dd:ee:ff", "iPhone", "disabled", now, now))

	got, err := service.DisableEmployee(context.Background(), 11)
	if err != nil {
		t.Fatalf("expected disable to succeed, got %v", err)
	}
	if got.Status != "disabled" {
		t.Fatalf("expected disabled employee status, got %s", got.Status)
	}
	if got.Devices[0].Status != "disabled" || got.Devices[0].ID != 21 {
		t.Fatalf("expected real employee reload after disable, got %+v", got.Devices)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all sql expectations to be met, got %v", err)
	}
}

func TestEmployeeAdminServiceDisableAlreadyDisabledEmployeeSucceeds(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	repo := repository.NewMySQLEmployeeRepository(db)
	service := NewEmployeeAdminService(db, repo)

	now := time.Date(2026, 3, 21, 8, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT e.id, e.employee_no, e.system_no, e.name, e.status, e.created_at, e.updated_at,
		       d.id, d.mac_address, d.device_label, d.status, d.created_at, d.updated_at
		FROM employees e
		LEFT JOIN employee_devices d ON d.employee_id = e.id
		WHERE e.id = ?
		ORDER BY d.id ASC
	`))).
		WithArgs(uint64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "employee_no", "system_no", "name", "status", "created_at", "updated_at", "device_id", "mac_address", "device_label", "device_status", "device_created_at", "device_updated_at"}).
			AddRow(uint64(11), "EMP-001", "SYS-001", "Alice", "disabled", now, now, uint64(21), "aa:bb:cc:dd:ee:ff", "iPhone", "disabled", now, now))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		UPDATE employees
		SET status = ?
		WHERE id = ?
	`))).
		WithArgs("disabled", int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		UPDATE employee_devices
		SET status = ?
		WHERE employee_id = ?
	`))).
		WithArgs("disabled", int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT e.id, e.employee_no, e.system_no, e.name, e.status, e.created_at, e.updated_at,
		       d.id, d.mac_address, d.device_label, d.status, d.created_at, d.updated_at
		FROM employees e
		LEFT JOIN employee_devices d ON d.employee_id = e.id
		WHERE e.id = ?
		ORDER BY d.id ASC
	`))).
		WithArgs(uint64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "employee_no", "system_no", "name", "status", "created_at", "updated_at", "device_id", "mac_address", "device_label", "device_status", "device_created_at", "device_updated_at"}).
			AddRow(uint64(11), "EMP-001", "SYS-001", "Alice", "disabled", now, now, uint64(21), "aa:bb:cc:dd:ee:ff", "iPhone", "disabled", now, now))

	got, err := service.DisableEmployee(context.Background(), 11)
	if err != nil {
		t.Fatalf("expected disable to succeed for already disabled employee, got %v", err)
	}
	if got.Status != "disabled" || got.Devices[0].Status != "disabled" {
		t.Fatalf("expected disabled employee to remain disabled, got %+v", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all sql expectations to be met, got %v", err)
	}
}

func TestEmployeeAdminServiceCreateRollsBackWhenDeviceInsertFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	repo := repository.NewMySQLEmployeeRepository(db)
	service := NewEmployeeAdminService(db, repo)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		INSERT INTO employees (
			employee_no,
			system_no,
			name,
			status
		) VALUES (?, ?, ?, ?)
	`))).
		WithArgs("EMP-003", "SYS-003", "Carol", "active").
		WillReturnResult(sqlmock.NewResult(12, 1))
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		DELETE FROM employee_devices
		WHERE employee_id = ?
	`))).
		WithArgs(int64(12)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		INSERT INTO employee_devices (
			employee_id,
			mac_address,
			device_label,
			status
		) VALUES (?, ?, ?, ?)
	`))).
		WithArgs(int64(12), "aa:bb:cc:dd:ee:12", "Phone", "active").
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	_, err = service.CreateEmployee(context.Background(), EmployeeWriteInput{
		EmployeeNo: "EMP-003",
		SystemNo:   "SYS-003",
		Name:       "Carol",
		Status:     "active",
		Devices: []EmployeeDeviceInput{
			{
				MacAddress:  "AA:BB:CC:DD:EE:12",
				DeviceLabel: "Phone",
				Status:      "active",
			},
		},
	})
	if err == nil {
		t.Fatal("expected device insert failure to surface")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected transaction rollback expectations to be met, got %v", err)
	}
}

func TestEmployeeAdminServiceUpdateReturnsNotFoundWhenEmployeeMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	repo := repository.NewMySQLEmployeeRepository(db)
	service := NewEmployeeAdminService(db, repo)

	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT e.id, e.employee_no, e.system_no, e.name, e.status, e.created_at, e.updated_at,
		       d.id, d.mac_address, d.device_label, d.status, d.created_at, d.updated_at
		FROM employees e
		LEFT JOIN employee_devices d ON d.employee_id = e.id
		WHERE e.id = ?
		ORDER BY d.id ASC
	`))).
		WithArgs(uint64(404)).
		WillReturnError(sql.ErrNoRows)

	_, err = service.UpdateEmployee(context.Background(), 404, EmployeeWriteInput{
		EmployeeNo: "EMP-404",
		SystemNo:   "SYS-404",
		Name:       "Nobody",
		Status:     "active",
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected not found error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected not found lookup expectations to be met, got %v", err)
	}
}

func TestEmployeeAdminServiceDisableReturnsNotFoundWhenEmployeeMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	repo := repository.NewMySQLEmployeeRepository(db)
	service := NewEmployeeAdminService(db, repo)

	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT e.id, e.employee_no, e.system_no, e.name, e.status, e.created_at, e.updated_at,
		       d.id, d.mac_address, d.device_label, d.status, d.created_at, d.updated_at
		FROM employees e
		LEFT JOIN employee_devices d ON d.employee_id = e.id
		WHERE e.id = ?
		ORDER BY d.id ASC
	`))).
		WithArgs(uint64(404)).
		WillReturnError(sql.ErrNoRows)

	_, err = service.DisableEmployee(context.Background(), 404)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected not found error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected not found lookup expectations to be met, got %v", err)
	}
}
