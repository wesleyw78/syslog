package config

import "testing"

func TestLoadConfigDefaults(t *testing.T) {
	cfg := LoadConfigFromEnv(func(string) string { return "" })
	if cfg.Timezone != "Asia/Shanghai" {
		t.Fatalf("expected default timezone Asia/Shanghai, got %s", cfg.Timezone)
	}
	if cfg.SyslogRetentionDays != 30 {
		t.Fatalf("expected retention 30, got %d", cfg.SyslogRetentionDays)
	}
	if cfg.MySQLHost != "mysql" {
		t.Fatalf("expected default mysql host mysql, got %s", cfg.MySQLHost)
	}
	if cfg.MySQLPort != 3306 {
		t.Fatalf("expected default mysql port 3306, got %d", cfg.MySQLPort)
	}
	if cfg.MySQLUser != "syslog" {
		t.Fatalf("expected default mysql user syslog, got %s", cfg.MySQLUser)
	}
	if cfg.MySQLPassword != "syslog" {
		t.Fatalf("expected default mysql password syslog, got %s", cfg.MySQLPassword)
	}
	if cfg.MySQLDatabase != "syslog" {
		t.Fatalf("expected default mysql database syslog, got %s", cfg.MySQLDatabase)
	}
	if cfg.MySQLParams != "charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true" {
		t.Fatalf("expected default mysql params for local compose, got %s", cfg.MySQLParams)
	}
}

func TestLoadConfigFromEnvOverrides(t *testing.T) {
	values := map[string]string{
		"TIMEZONE":              "UTC",
		"SYSLOG_RETENTION_DAYS": "7",
		"MYSQL_HOST":            "127.0.0.1",
		"MYSQL_PORT":            "3307",
		"MYSQL_USER":            "reader",
		"MYSQL_PASSWORD":        "secret",
		"MYSQL_DATABASE":        "syslog_test",
		"MYSQL_PARAMS":          "charset=utf8mb4&parseTime=true",
	}

	cfg := LoadConfigFromEnv(func(key string) string {
		return values[key]
	})

	if cfg.Timezone != "UTC" {
		t.Fatalf("expected overridden timezone UTC, got %s", cfg.Timezone)
	}
	if cfg.SyslogRetentionDays != 7 {
		t.Fatalf("expected overridden retention 7, got %d", cfg.SyslogRetentionDays)
	}
	if cfg.MySQLHost != "127.0.0.1" {
		t.Fatalf("expected overridden mysql host 127.0.0.1, got %s", cfg.MySQLHost)
	}
	if cfg.MySQLPort != 3307 {
		t.Fatalf("expected overridden mysql port 3307, got %d", cfg.MySQLPort)
	}
	if cfg.MySQLUser != "reader" {
		t.Fatalf("expected overridden mysql user reader, got %s", cfg.MySQLUser)
	}
	if cfg.MySQLPassword != "secret" {
		t.Fatalf("expected overridden mysql password secret, got %s", cfg.MySQLPassword)
	}
	if cfg.MySQLDatabase != "syslog_test" {
		t.Fatalf("expected overridden mysql database syslog_test, got %s", cfg.MySQLDatabase)
	}
	if cfg.MySQLParams != "charset=utf8mb4&parseTime=true" {
		t.Fatalf("expected overridden mysql params, got %s", cfg.MySQLParams)
	}
}
