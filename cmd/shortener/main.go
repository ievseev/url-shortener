package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/ievseev/url-shortener/internal/config"
	createUrlPostHandler "github.com/ievseev/url-shortener/internal/handler/create_url_post"
	urlRepo "github.com/ievseev/url-shortener/internal/repository/url"
	urlService "github.com/ievseev/url-shortener/internal/service/url_shortener"
)

const fail = 1

func main() {
	if err := run(); err != nil {
		os.Exit(fail)
	}
}

func run() error {
	logger := setupLogger()

	// init configs
	appConfig := config.Must()

	// create repos
	repository := urlRepo.NewStorage()

	// create services
	urlServ := urlService.New(repository)

	// create handlers
	createUrlHandler := createUrlPostHandler.New(urlServ)

	// register handlers
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, createUrlHandler.Handle)

	serverAddr := appConfig.Host + ":" + appConfig.Port
	logger.Info("Starting HTTP server", "address", serverAddr)

	// start server - убираем обработку ошибки после запуска
	err := http.ListenAndServe(serverAddr, mux)
	// Если сервер упал, логируем и возвращаем ошибку
	logger.Error("HTTP server stopped", "error", err)

	return err

}

func setupLogger() *slog.Logger {
	// Создаем JSON handler для структурированных логов
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo, // минимальный уровень логирования
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	// Устанавливаем как глобальный логгер (опционально)
	slog.SetDefault(logger)

	return logger
}
