package config

import (
	"flag"
)

type AppConfig struct {
	AppAddress, BaseURL string
}

var (
	address *string
	baseURL *string
)

func init() {
	// Регистрируем флаги при инициализации пакета
	address = flag.String("a", "localhost:8080", "app address")
	baseURL = flag.String("b", "http://localhost:8000", "base url for response")
}

func Init() *AppConfig {
	return &AppConfig{
		AppAddress: *address,
		BaseURL:    *baseURL,
	}
}
