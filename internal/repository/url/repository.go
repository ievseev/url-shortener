package url

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

var ErrOriginURLNotFound = errors.New("origin URL not found")

type Storage interface {
	Save(ctx context.Context, data map[string]string) error
	Load(ctx context.Context) (map[string]string, error)
}

type Repository struct {
	mu      sync.Mutex
	urlMap  map[string]string
	storage Storage
	logger  *slog.Logger
}

func New(logger *slog.Logger, storage Storage) (*Repository, error) {
	repo := &Repository{
		logger:  logger,
		storage: storage,
		urlMap:  make(map[string]string),
	}

	// Загружаем данные из хранилища при инициализации
	data, err := storage.Load(context.Background())
	if err != nil {
		logger.Error("load from storage error", "error", err)
		return nil, err
	}
	repo.urlMap = data

	return repo, nil
}

func (r *Repository) SaveURLPair(ctx context.Context, urlOrigin, urlShort string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.urlMap[urlShort] = urlOrigin

	if err := r.storage.Save(ctx, r.urlMap); err != nil {
		r.logger.Error("save to storage error", "error", err)
		return err
	}

	r.logger.Debug("saved url pair", "urlShort", urlShort, "urlOrigin", urlOrigin)
	return nil
}

func (r *Repository) GetOriginURL(ctx context.Context, urlShort string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if result, ok := r.urlMap[urlShort]; ok {
		return result, nil
	}

	return "", ErrOriginURLNotFound
}
