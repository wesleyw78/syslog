package db

import (
	"context"
	"database/sql"
	_ "embed"
)

//go:embed migrations/001_init.sql
var migrationSQL string

func SQL() string {
	return migrationSQL
}

func ApplyMigrations(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, migrationSQL)
	return err
}
