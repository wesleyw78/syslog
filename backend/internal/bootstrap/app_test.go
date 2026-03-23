package bootstrap

import (
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	schema "syslog/internal/db"
)

func TestNewLoadsConfigOpensDBRunsMigrationAndAssemblesApp(t *testing.T) {
	originalOpenDB := openDB
	defer func() { openDB = originalOpenDB }()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	capturedDriver := ""
	capturedDSN := ""
	openDB = func(driverName, dsn string) (*sql.DB, error) {
		capturedDriver = driverName
		capturedDSN = dsn
		return db, nil
	}

	mock.ExpectExec(regexp.QuoteMeta(schema.SQL())).WillReturnResult(sqlmock.NewResult(0, 1))

	app, err := New(func(string) string { return "" })
	if err != nil {
		t.Fatalf("expected bootstrap to succeed, got %v", err)
	}

	if capturedDriver != "mysql" {
		t.Fatalf("expected mysql driver, got %s", capturedDriver)
	}
	if capturedDSN == "" {
		t.Fatal("expected mysql dsn to be built")
	}
	if app.DB == nil {
		t.Fatal("expected db to be attached to app")
	}
	if app.Config.Timezone != "Asia/Shanghai" {
		t.Fatalf("expected fixed timezone, got %s", app.Config.Timezone)
	}
	if app.Location == nil || app.Location.String() != "Asia/Shanghai" {
		t.Fatalf("expected asia/shanghai location, got %+v", app.Location)
	}
	if app.Repositories.Employees == nil || app.Repositories.SyslogMessages == nil || app.Repositories.ClientEvents == nil || app.Repositories.Attendance == nil || app.Repositories.Reports == nil || app.Repositories.Settings == nil || app.Repositories.SyslogRules == nil || app.Repositories.DayEndRuns == nil {
		t.Fatalf("expected repositories to be assembled, got %+v", app.Repositories)
	}
	if app.Services.SyslogPipeline == nil {
		t.Fatalf("expected syslog pipeline service to be assembled")
	}
	if app.Services.EmployeeAdmin == nil || app.Services.SettingsAdmin == nil || app.Services.SyslogRuleAdmin == nil || app.Services.AttendanceAdmin == nil || app.Services.DayEndDispatcher == nil {
		t.Fatalf("expected admin services to be assembled, got %+v", app.Services)
	}

	mock.ExpectClose()
	if err := app.Close(); err != nil {
		t.Fatalf("expected app close to succeed, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected migration and close to run, got %v", err)
	}
}
