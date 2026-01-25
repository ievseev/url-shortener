package config

import (
	"flag"
	"log/slog"

	"github.com/caarlos0/env/v6"
)

const (
	defaultServerAddress = "localhost:8888"
	defaultBaseURL       = "http://localhost:8888"
)

type AppConfig struct {
	ServerAddress string `env:"SERVER_ADDRESS"`
	BaseURL       string `env:"BASE_URL"`
}

func Init(logger *slog.Logger) *AppConfig {
	appConfig := &AppConfig{}
	err := env.Parse(appConfig)
	if err != nil {
		logger.Warn("failed to parse env variables", "error", err)
	}

	if appConfig.ServerAddress == "" || appConfig.BaseURL == "" {
		// Регистрируем флаги при инициализации пакета
		appConfig.ServerAddress = *flag.String("a", defaultServerAddress, "server address")
		appConfig.BaseURL = *flag.String("b", defaultBaseURL, "base url")
		flag.Parse()
	}

	return appConfig
}
