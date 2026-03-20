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

func TestOpenMySQLUsesConfigValues(t *testing.T) {
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
		MySQLHost:     "127.0.0.1",
		MySQLPort:     3306,
		MySQLUser:     "syslog",
		MySQLPassword: "secret",
		MySQLDatabase: "syslog",
		MySQLParams:   "charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true",
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
	if !strings.HasPrefix(capturedDSN, "syslog:secret@tcp(127.0.0.1:3306)/syslog?") {
		t.Fatalf("unexpected dsn prefix %s", capturedDSN)
	}
	for _, fragment := range []string{"charset=utf8mb4", "parseTime=true", "loc=Local", "multiStatements=true"} {
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
