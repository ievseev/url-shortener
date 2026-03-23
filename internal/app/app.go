package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/ievseev/url-shortener/internal/config"
	apiShortenBatchPostHandler "github.com/ievseev/url-shortener/internal/handler/apishortenbatchpost"
	apiShortenURLPostHandler "github.com/ievseev/url-shortener/internal/handler/apishortenurlpost"
	apiUserURLsDeleteHandler "github.com/ievseev/url-shortener/internal/handler/apiuserurlsdelete"
	apiUserURLsGetHandler "github.com/ievseev/url-shortener/internal/handler/apiuserurlsget"
	expandURLGetHandler "github.com/ievseev/url-shortener/internal/handler/expandurlget"
	pingGetHandler "github.com/ievseev/url-shortener/internal/handler/pingget"
	shortenURLPostHandler "github.com/ievseev/url-shortener/internal/handler/shortenurlpost"
	"github.com/ievseev/url-shortener/internal/middleware"
	urlrepo "github.com/ievseev/url-shortener/internal/repository/url"
	urlservice "github.com/ievseev/url-shortener/internal/service/urlshortener"
	fileStorage "github.com/ievseev/url-shortener/internal/storage"
	"github.com/ievseev/url-shortener/internal/storage/db"
	memoryStorage "github.com/ievseev/url-shortener/internal/storage/memory"
)

func Run() error {
	logger := setupLogger()
	appConfig := config.Init(logger)
	ctx := context.Background()

	repository, closeStorage, err := initDependencies(ctx, logger, appConfig)
	if err != nil {
		logger.Error("dependency init error", "error", err)
		return err
	}
	if closeStorage != nil {
		defer closeStorage()
	}

	// init services
	urlServ, err := urlservice.New(repository)
	if err != nil {
		logger.Error("url service init error", "error", err)
		return err
	}

	// init handlers
	shortenURLHandler := shortenURLPostHandler.New(appConfig.BaseURL, urlServ, logger)
	expandURLHandler := expandURLGetHandler.New(urlServ, logger)
	apiShortenURLHandler := apiShortenURLPostHandler.New(appConfig.BaseURL, urlServ, logger)
	apiShortenBatchHandler := apiShortenBatchPostHandler.New(appConfig.BaseURL, urlServ, logger)
	apiUserURLsHandler := apiUserURLsGetHandler.New(appConfig.BaseURL, urlServ, logger)
	apiUserURLsDeleteHandler := apiUserURLsDeleteHandler.New(urlServ, logger)

	// init router
	r := chi.NewRouter()

	r.Use(middleware.RequestLogger(logger))
	r.Use(middleware.GzipMiddleware)

	pingHandler := pingGetHandler.New(repository, logger)
	r.Get("/ping", pingHandler.Handle)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(logger))
		r.Post("/", shortenURLHandler.Handle)
		r.Post("/api/shorten/batch", apiShortenBatchHandler.Handle)
		r.Post("/api/shorten", apiShortenURLHandler.Handle)
		r.Get("/api/user/urls", apiUserURLsHandler.Handle)
		r.Delete("/api/user/urls", apiUserURLsDeleteHandler.Handle)
	})

	r.Get("/{id}", expandURLHandler.Handle)

	logger.Info("Starting HTTP server", "address", appConfig.ServerAddress)
	err = http.ListenAndServe(appConfig.ServerAddress, r)
	logger.Error("HTTP server stopped", "error", err)

	return err
}

func initDependencies(
	ctx context.Context,
	logger *slog.Logger,
	appConfig *config.AppConfig,
) (urlservice.Repository, func(), error) {
	if appConfig.DatabaseDSN != "" {
		dbStorage, err := db.NewPostgres(ctx, appConfig.DatabaseDSN)
		if err != nil {
			return nil, nil, err
		}

		logger.Info("using postgres storage")

		return dbStorage, dbStorage.Close, nil
	}

	if appConfig.FileStoragePath != "" {
		storage := fileStorage.New(logger, appConfig.FileStoragePath)
		repository, err := urlrepo.New(logger, storage)
		if err != nil {
			return nil, nil, err
		}

		logger.Info("using file storage", "path", appConfig.FileStoragePath)

		return repository, nil, nil
	}

	storage := memoryStorage.New()
	repository, err := urlrepo.New(logger, storage)
	if err != nil {
		return nil, nil, err
	}

	logger.Info("using in-memory storage")

	return repository, nil, nil
}

func setupLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return logger
}
