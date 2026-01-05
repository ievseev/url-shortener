package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/ievseev/url-shortener/internal/config"
	expandUrlGetHandler "github.com/ievseev/url-shortener/internal/handler/expand_url_get"
	shortenUrlPostHandler "github.com/ievseev/url-shortener/internal/handler/shorten_url_post"
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
	shortenUrlHandler := shortenUrlPostHandler.New(urlServ)
	expandUrlHandler := expandUrlGetHandler.New(urlServ)

	// create router with chi
	r := chi.NewRouter()

	// register routes
	r.Post("/", shortenUrlHandler.Handle)
	r.Get("/{id}", expandUrlHandler.Handle)

	logger.Info("Starting HTTP server", "address", appConfig.AppAddress)

	// start server - убираем обработку ошибки после запуска
	err := http.ListenAndServe(appConfig.AppAddress, r)
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

	// Устанавливаем как глобальный логгер
	slog.SetDefault(logger)

	return logger
}
