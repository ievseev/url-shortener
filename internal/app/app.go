package app

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/ievseev/url-shortener/internal/config"
	apiShortenURLPostHandler "github.com/ievseev/url-shortener/internal/handler/apishortenurlpost"
	expandURLGetHandler "github.com/ievseev/url-shortener/internal/handler/expandurlget"
	shortenURLPostHandler "github.com/ievseev/url-shortener/internal/handler/shortenurlpost"
	gzipmiddleware "github.com/ievseev/url-shortener/internal/middleware"
	URLRepo "github.com/ievseev/url-shortener/internal/repository/url"
	URLService "github.com/ievseev/url-shortener/internal/service/urlshortener"
)

func Run() error {
	logger := setupLogger()
	appConfig := config.Init(logger)

	// init repos
	repository := URLRepo.NewStorage(logger)

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

	r.Use(requestLogger(logger))
	r.Use(gzipmiddleware.GzipDecompressMiddleware()) // обработка входящих сжатых запросов
	r.Use(gzipmiddleware.GzipCompressMiddleware())   // сжатие исходящих ответов

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

func requestLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			logger.Info(
				"request",
				"uri", r.RequestURI,
				"method", r.Method,
				"duration", time.Since(start))
		})
	}
}
