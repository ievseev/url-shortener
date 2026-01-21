package config

import "flag"

type AppConfig struct {
	AppAddress, BaseURL string
}

var (
	address *string
	baseURL *string
)

func Init() *AppConfig {
	// Регистрируем флаги при инициализации пакета
	address = flag.String("a", "localhost:8080", "app address")
	baseURL = flag.String("b", "http://localhost:8080", "base url for response")
	flag.Parse()

	return &AppConfig{
		AppAddress: *address,
		BaseURL:    *baseURL,
	}
}
