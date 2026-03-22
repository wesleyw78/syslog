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
	"syslog/internal/domain"
	"syslog/internal/repository"
)

func TestSettingsAdminServiceUpdateBatchPersistsOnlyKnownKeys(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	repo := repository.NewMySQLSystemSettingRepository(db)
	service := NewSettingsAdminService(db, repo)

	now := time.Date(2026, 3, 21, 8, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"id", "setting_key", "setting_value", "updated_at"}).
		AddRow(uint64(1), "day_end_time", "23:59", now).
		AddRow(uint64(2), "syslog_retention_days", "30", now).
		AddRow(uint64(3), "feishu_app_id", "", now).
		AddRow(uint64(4), "feishu_app_secret", "", now).
		AddRow(uint64(5), "feishu_creator_employee_id", "", now).
		AddRow(uint64(6), "feishu_location_name", "", now).
		AddRow(uint64(7), "report_timeout_seconds", "10", now).
		AddRow(uint64(8), "report_retry_limit", "3", now)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, setting_key, setting_value, updated_at
		FROM system_settings
		ORDER BY setting_key ASC
	`))).WillReturnRows(rows)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		INSERT INTO system_settings (
			setting_key,
			setting_value
		) VALUES (?, ?)
		ON DUPLICATE KEY UPDATE
			id = LAST_INSERT_ID(id),
			setting_value = VALUES(setting_value)
	`))).
		WithArgs("day_end_time", "22:00").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(strings.TrimSpace(`
		INSERT INTO system_settings (
			setting_key,
			setting_value
		) VALUES (?, ?)
		ON DUPLICATE KEY UPDATE
			id = LAST_INSERT_ID(id),
			setting_value = VALUES(setting_value)
	`))).
		WithArgs("feishu_app_id", "cli_abc").
		WillReturnResult(sqlmock.NewResult(3, 1))
	mock.ExpectCommit()

	got, err := service.UpdateSettings(context.Background(), []SettingWriteInput{
		{SettingKey: "day_end_time", SettingValue: "22:00"},
		{SettingKey: "feishu_app_id", SettingValue: "cli_abc"},
	})
	if err != nil {
		t.Fatalf("expected settings update to succeed, got %v", err)
	}
	if len(got) != 8 {
		t.Fatalf("expected all current settings to be returned, got %d", len(got))
	}

	gotMap := make(map[string]string, len(got))
	for _, item := range got {
		gotMap[item.SettingKey] = item.SettingValue
	}
	if gotMap["day_end_time"] != "22:00" {
		t.Fatalf("expected day_end_time to be updated, got %q", gotMap["day_end_time"])
	}
	if gotMap["feishu_app_id"] != "cli_abc" {
		t.Fatalf("expected feishu_app_id to be updated, got %q", gotMap["feishu_app_id"])
	}
	if gotMap["syslog_retention_days"] != "30" {
		t.Fatalf("expected untouched settings to be preserved, got %q", gotMap["syslog_retention_days"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all sql expectations to be met, got %v", err)
	}
}

func TestSettingsAdminServiceRejectsDuplicateKeysInBatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	repo := repository.NewMySQLSystemSettingRepository(db)
	service := NewSettingsAdminService(db, repo)

	now := time.Date(2026, 3, 21, 8, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"id", "setting_key", "setting_value", "updated_at"}).
		AddRow(uint64(1), "day_end_time", "23:59", now)
	mock.ExpectQuery(regexp.QuoteMeta(strings.TrimSpace(`
		SELECT id, setting_key, setting_value, updated_at
		FROM system_settings
		ORDER BY setting_key ASC
	`))).WillReturnRows(rows)

	_, err = service.UpdateSettings(context.Background(), []SettingWriteInput{
		{SettingKey: "day_end_time", SettingValue: "22:00"},
		{SettingKey: "day_end_time", SettingValue: "21:00"},
	})
	if !errors.Is(err, ErrInvalidSettingsInput) {
		t.Fatalf("expected duplicate batch to be invalid, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected duplicate-key validation to stop before writes, got %v", err)
	}
}

var _ repository.SystemSettingRepository = (*repository.MySQLSystemSettingRepository)(nil)
var _ = domain.SystemSetting{}
var _ = sql.ErrNoRows
