package service

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
	"testing"

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
	if len(got.Devices) != 1 || got.Devices[0].MacAddress != "aa:bb:cc:dd:ee:ff" {
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

	got, err := service.DisableEmployee(context.Background(), 11)
	if err != nil {
		t.Fatalf("expected disable to succeed, got %v", err)
	}
	if got.Status != "disabled" {
		t.Fatalf("expected disabled employee status, got %s", got.Status)
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
