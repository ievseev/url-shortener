package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	AppAddress string
}

func Must() *AppConfig {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	return &AppConfig{
		AppAddress: os.Getenv("APP_ADDRESS"),
	}
}
