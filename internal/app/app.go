package app

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/ievseev/url-shortener/internal/config"
	expandUrlGetHandler "github.com/ievseev/url-shortener/internal/handler/expandurlget"
	shortenUrlPostHandler "github.com/ievseev/url-shortener/internal/handler/shortenurlpost"
	URLRepo "github.com/ievseev/url-shortener/internal/repository/url"
	URLService "github.com/ievseev/url-shortener/internal/service/urlshortener"
)

func Run() error {
	logger := setupLogger()
	appConfig := config.Init()

	// init repos
	repository := URLRepo.NewStorage()

	// init services
	urlServ, err := URLService.New(repository)
	if err != nil {
		logger.Error("url service init error", "error", err)
		return err
	}

	// init handlers
	shortenUrlHandler := shortenUrlPostHandler.New(appConfig.BaseURL, urlServ, logger)
	expandUrlHandler := expandUrlGetHandler.New(urlServ, logger)

	// init router
	r := chi.NewRouter()
	r.Post("/", shortenUrlHandler.Handle)
	r.Get("/{id}", expandUrlHandler.Handle)

	logger.Info("Starting HTTP server", "address", appConfig.AppAddress)
	err = http.ListenAndServe(appConfig.AppAddress, r)
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
