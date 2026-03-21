package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"syslog/internal/config"
	schema "syslog/internal/db"
)

var openDB = sql.Open

func OpenMySQL(cfg config.Config) (*sql.DB, error) {
	dsn, err := buildMySQLDSN(cfg)
	if err != nil {
		return nil, err
	}

	return openDB("mysql", dsn)
}

func RunMigrations(ctx context.Context, db *sql.DB) error {
	return schema.ApplyMigrations(ctx, db)
}

func buildMySQLDSN(cfg config.Config) (string, error) {
	if cfg.MySQLDSN != "" {
		mysqlConfig, err := mysql.ParseDSN(cfg.MySQLDSN)
		if err != nil {
			return "", err
		}

		normalizeMySQLConfig(mysqlConfig, "Asia/Shanghai")
		return mysqlConfig.FormatDSN(), nil
	}

	mysqlConfig := mysql.NewConfig()
	mysqlConfig.User = cfg.MySQLUser
	mysqlConfig.Passwd = cfg.MySQLPassword
	mysqlConfig.Net = "tcp"
	mysqlConfig.Addr = fmt.Sprintf("%s:%d", cfg.MySQLHost, cfg.MySQLPort)
	mysqlConfig.DBName = cfg.MySQLDatabase
	mysqlConfig.Params = map[string]string{}

	params := parseMySQLParams(cfg.MySQLParams)
	if value, ok := params["charset"]; ok && value != "" {
		mysqlConfig.Params["charset"] = value
	} else {
		mysqlConfig.Params["charset"] = "utf8mb4"
	}
	for key, value := range params {
		if isControlledMySQLParam(key) || value == "" {
			continue
		}

		mysqlConfig.Params[key] = value
	}

	normalizeMySQLConfig(mysqlConfig, "Asia/Shanghai")
	return mysqlConfig.FormatDSN(), nil
}

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}

	return loc
}

func normalizeMySQLConfig(cfg *mysql.Config, locationName string) {
	cfg.ParseTime = true
	cfg.MultiStatements = true
	cfg.Loc = mustLoadLocation(locationName)

	if cfg.Params == nil {
		cfg.Params = map[string]string{}
	}

	for key := range cfg.Params {
		if isControlledMySQLParam(key) {
			delete(cfg.Params, key)
		}
	}
}

func isControlledMySQLParam(key string) bool {
	switch key {
	case "charset", "parseTime", "multiStatements", "loc":
		return true
	default:
		return false
	}
}

func parseMySQLParams(raw string) map[string]string {
	if strings.TrimSpace(raw) == "" {
		return map[string]string{}
	}

	values, err := url.ParseQuery(raw)
	if err != nil {
		return map[string]string{}
	}

	params := make(map[string]string, len(values))
	for key, entries := range values {
		if len(entries) == 0 {
			continue
		}
		params[key] = entries[len(entries)-1]
	}

	return params
}
