package config

import "strconv"

type Config struct {
	Timezone            string
	SyslogRetentionDays int
	SyslogUDPAddr       string
	MySQLDSN            string
	MySQLHost           string
	MySQLPort           int
	MySQLUser           string
	MySQLPassword       string
	MySQLDatabase       string
	MySQLParams         string
}

func LoadConfigFromEnv(getenv func(string) string) Config {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}

	return Config{
		Timezone:            "Asia/Shanghai",
		SyslogRetentionDays: intOrDefault(getenv("SYSLOG_RETENTION_DAYS"), 30),
		SyslogUDPAddr:       stringOrDefault(getenv("SYSLOG_UDP_ADDR"), ":514"),
		MySQLDSN:            stringOrDefault(getenv("MYSQL_DSN"), ""),
		MySQLHost:           stringOrDefault(getenv("MYSQL_HOST"), "127.0.0.1"),
		MySQLPort:           intOrDefault(getenv("MYSQL_PORT"), 3306),
		MySQLUser:           stringOrDefault(getenv("MYSQL_USER"), "syslog"),
		MySQLPassword:       stringOrDefault(getenv("MYSQL_PASSWORD"), "syslog"),
		MySQLDatabase:       stringOrDefault(getenv("MYSQL_DATABASE"), "syslog"),
		MySQLParams:         stringOrDefault(getenv("MYSQL_PARAMS"), "charset=utf8mb4&parseTime=true&loc=Asia/Shanghai&multiStatements=true"),
	}
}

func stringOrDefault(value string, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}

func intOrDefault(value string, fallback int) int {
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
