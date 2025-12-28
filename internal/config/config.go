package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	Host, Port string
}

func Must() *AppConfig {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	host := os.Getenv("APP_HOST")
	port := os.Getenv("APP_PORT")

	return &AppConfig{
		Host: host,
		Port: port,
	}
}
