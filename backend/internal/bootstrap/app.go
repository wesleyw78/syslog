package bootstrap

import (
	"database/sql"
	"os"

	"syslog/internal/config"
)

type App struct {
	Config config.Config
	DB     *sql.DB
}

func New(getenv func(string) string) App {
	if getenv == nil {
		getenv = os.Getenv
	}

	return App{
		Config: config.LoadConfigFromEnv(getenv),
	}
}
