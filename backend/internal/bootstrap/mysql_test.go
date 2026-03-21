package bootstrap

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"syslog/internal/config"
	schema "syslog/internal/db"
)

func TestOpenMySQLUsesConfigDSNWhenProvided(t *testing.T) {
	originalOpenDB := openDB
	defer func() { openDB = originalOpenDB }()

	capturedDriver := ""
	capturedDSN := ""
	openDB = func(driverName, dsn string) (*sql.DB, error) {
		capturedDriver = driverName
		capturedDSN = dsn

		db, _, err := sqlmock.New()
		return db, err
	}

	cfg := config.Config{
		MySQLDSN:  "reader:secret@tcp(db.example.com:3306)/syslog?parseTime=false&multiStatements=false&loc=UTC",
		MySQLHost: "127.0.0.1",
		MySQLPort: 3306,
		MySQLUser: "syslog",
	}

	db, err := OpenMySQL(cfg)
	if err != nil {
		t.Fatalf("expected open mysql to succeed, got %v", err)
	}
	if db == nil {
		t.Fatalf("expected db to be returned")
	}
	if capturedDriver != "mysql" {
		t.Fatalf("expected mysql driver, got %s", capturedDriver)
	}
	for _, fragment := range []string{
		"reader:secret@tcp(db.example.com:3306)/syslog?",
		"parseTime=true",
		"multiStatements=true",
		"loc=Asia%2FShanghai",
	} {
		if !strings.Contains(capturedDSN, fragment) {
			t.Fatalf("expected normalized dsn to contain %s, got %s", fragment, capturedDSN)
		}
	}
}

func TestOpenMySQLBuildsDSNFromSplitFields(t *testing.T) {
	originalOpenDB := openDB
	defer func() { openDB = originalOpenDB }()

	capturedDSN := ""
	openDB = func(driverName, dsn string) (*sql.DB, error) {
		capturedDSN = dsn

		db, _, err := sqlmock.New()
		return db, err
	}

	cfg := config.Config{
		MySQLHost:     "127.0.0.1",
		MySQLPort:     3306,
		MySQLUser:     "syslog",
		MySQLPassword: "secret",
		MySQLDatabase: "syslog",
		MySQLParams:   "charset=utf8mb4&parseTime=true&loc=Asia/Shanghai&multiStatements=true",
	}

	if _, err := OpenMySQL(cfg); err != nil {
		t.Fatalf("expected open mysql to succeed, got %v", err)
	}
	for _, fragment := range []string{"syslog:secret@tcp(127.0.0.1:3306)/syslog?", "charset=utf8mb4", "parseTime=true", "loc=Asia%2FShanghai", "multiStatements=true"} {
		if !strings.Contains(capturedDSN, fragment) {
			t.Fatalf("expected dsn to contain %s, got %s", fragment, capturedDSN)
		}
	}
}

func TestRunMigrationsExecutesEmbeddedSQL(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sqlmock db, got %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(schema.SQL())).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := RunMigrations(context.Background(), db); err != nil {
		t.Fatalf("expected migrations to run, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected all migration expectations to be met, got %v", err)
	}
}

func TestMigrationSQLIsIdempotent(t *testing.T) {
	sql := schema.SQL()
	for _, fragment := range []string{"CREATE TABLE IF NOT EXISTS employees", "INSERT IGNORE INTO system_settings"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected migration sql to contain %q, got %s", fragment, sql)
		}
	}
}
