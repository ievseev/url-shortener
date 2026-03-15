package config

import (
	"flag"
	"log/slog"

	"github.com/caarlos0/env/v6"
)

const (
	defaultServerAddress = "localhost:8080"
	defaultBaseURL       = "http://localhost:8080"
)

type AppConfig struct {
	ServerAddress   string `env:"SERVER_ADDRESS"`
	BaseURL         string `env:"BASE_URL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
}

func Init(logger *slog.Logger) *AppConfig {
	appConfig := &AppConfig{}

	// Определяем флаги командной строки
	serverAddr := flag.String("a", defaultServerAddress, "server address")
	baseURL := flag.String("b", defaultBaseURL, "base url")
	fileStoragePath := flag.String("f", "", "storage file path")
	databaseDSN := flag.String("d", "", "database dsn")

	flag.Parse()

	// 1. Пытаемся загрузить из переменных окружения
	err := env.Parse(appConfig)
	if err != nil {
		logger.Warn("failed to parse env variables", "error", err)
	}

	// 2. Если переменные окружения не установлены, используем флаги командной строки
	if appConfig.ServerAddress == "" {
		appConfig.ServerAddress = *serverAddr
	}
	if appConfig.BaseURL == "" {
		appConfig.BaseURL = *baseURL
	}
	if appConfig.FileStoragePath == "" {
		appConfig.FileStoragePath = *fileStoragePath
	}
	if appConfig.DatabaseDSN == "" {
		appConfig.DatabaseDSN = *databaseDSN
	}

	return appConfig
}
