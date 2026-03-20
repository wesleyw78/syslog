package bootstrap

import (
	"os"

	"syslog/internal/config"
)

type App struct {
	Config config.Config
}

func New(getenv func(string) string) App {
	if getenv == nil {
		getenv = os.Getenv
	}

	return App{
		Config: config.LoadConfigFromEnv(getenv),
	}
}
