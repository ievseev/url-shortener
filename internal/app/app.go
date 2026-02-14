package app

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/ievseev/url-shortener/internal/config"
	apiShortenURLPostHandler "github.com/ievseev/url-shortener/internal/handler/apishortenurlpost"
	expandURLGetHandler "github.com/ievseev/url-shortener/internal/handler/expandurlget"
	shortenURLPostHandler "github.com/ievseev/url-shortener/internal/handler/shortenurlpost"
	"github.com/ievseev/url-shortener/internal/middleware"
	URLRepo "github.com/ievseev/url-shortener/internal/repository/url"
	URLService "github.com/ievseev/url-shortener/internal/service/urlshortener"
	fileStorage "github.com/ievseev/url-shortener/internal/storage"
)

func Run() error {
	logger := setupLogger()
	appConfig := config.Init(logger)

	// init storage
	storage := fileStorage.New(logger, appConfig.FileStoragePath)

	// init repos
	repository, err := URLRepo.New(logger, storage)
	if err != nil {
		logger.Error("storage init error", "error", err)
		return err
	}

	// init services
	urlServ, err := URLService.New(repository)
	if err != nil {
		logger.Error("url service init error", "error", err)
		return err
	}

	// init handlers
	shortenURLHandler := shortenURLPostHandler.New(appConfig.BaseURL, urlServ, logger)
	expandURLHandler := expandURLGetHandler.New(urlServ, logger)
	apiShortenURLHandler := apiShortenURLPostHandler.New(appConfig.BaseURL, urlServ, logger)

	// init router
	r := chi.NewRouter()

	r.Use(middleware.RequestLogger(logger))
	r.Use(middleware.GzipMiddleware)

	r.Post("/", shortenURLHandler.Handle)
	r.Post("/api/shorten", apiShortenURLHandler.Handle)
	r.Get("/{id}", expandURLHandler.Handle)

	logger.Info("Starting HTTP server", "address", appConfig.ServerAddress)
	err = http.ListenAndServe(appConfig.ServerAddress, r)
	logger.Error("HTTP server stopped", "error", err)

	return err
}

func setupLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return logger
}
