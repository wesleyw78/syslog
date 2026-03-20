package config

type Config struct {
	Timezone            string
	SyslogRetentionDays int
}

func LoadConfigFromEnv(getenv func(string) string) Config {
	return Config{
		Timezone:            "Asia/Shanghai",
		SyslogRetentionDays: 30,
	}
}
