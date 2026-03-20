package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"syslog/internal/config"
	schema "syslog/internal/db"
)

var openDB = sql.Open

func OpenMySQL(cfg config.Config) (*sql.DB, error) {
	if cfg.MySQLDSN != "" {
		return openDB("mysql", cfg.MySQLDSN)
	}

	return openDB("mysql", buildMySQLDSN(cfg))
}

func RunMigrations(ctx context.Context, db *sql.DB) error {
	return schema.ApplyMigrations(ctx, db)
}

func buildMySQLDSN(cfg config.Config) string {
	mysqlConfig := mysql.NewConfig()
	mysqlConfig.User = cfg.MySQLUser
	mysqlConfig.Passwd = cfg.MySQLPassword
	mysqlConfig.Net = "tcp"
	mysqlConfig.Addr = fmt.Sprintf("%s:%d", cfg.MySQLHost, cfg.MySQLPort)
	mysqlConfig.DBName = cfg.MySQLDatabase
	mysqlConfig.Params = map[string]string{}
	mysqlConfig.ParseTime = true
	mysqlConfig.MultiStatements = true
	mysqlConfig.Loc = mustLoadLocation("Asia/Shanghai")

	params := parseMySQLParams(cfg.MySQLParams)
	if value, ok := params["charset"]; ok && value != "" {
		mysqlConfig.Params["charset"] = value
	} else {
		mysqlConfig.Params["charset"] = "utf8mb4"
	}

	if value, ok := params["parseTime"]; ok {
		mysqlConfig.ParseTime = parseBool(value, true)
	}
	if value, ok := params["multiStatements"]; ok {
		mysqlConfig.MultiStatements = parseBool(value, true)
	}
	if value, ok := params["loc"]; ok && value != "" {
		if loc, err := time.LoadLocation(value); err == nil {
			mysqlConfig.Loc = loc
		}
	}

	for key, value := range params {
		if key == "charset" || key == "parseTime" || key == "multiStatements" || key == "loc" || value == "" {
			continue
		}

		mysqlConfig.Params[key] = value
	}

	return mysqlConfig.FormatDSN()
}

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}

	return loc
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

func parseBool(value string, fallback bool) bool {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}
