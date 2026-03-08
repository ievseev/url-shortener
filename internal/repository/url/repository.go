package url

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

var ErrOriginURLNotFound = errors.New("origin URL not found")

type Storage interface {
	Save(ctx context.Context, data map[string]string) error
	Load(ctx context.Context) (map[string]string, error)
}

type Repository struct {
	mu      sync.RWMutex
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

func (r *Repository) SaveURL(ctx context.Context, urlOrigin, shortURLBase string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	updated := cloneMap(r.urlMap)
	shortURL := reserveShortURL(updated, urlOrigin, shortURLBase)

	if err := r.storage.Save(ctx, updated); err != nil {
		r.logger.Error("save to storage error", "error", err)
		return "", err
	}

	r.urlMap = updated
	r.logger.Debug("saved url pair", "urlShort", shortURL, "urlOrigin", urlOrigin)

	return shortURL, nil
}

func (r *Repository) SaveURLBatch(
	ctx context.Context,
	urlOrigins, shortURLBases []string,
) ([]string, error) {
	if len(urlOrigins) != len(shortURLBases) {
		return nil, errors.New("url origins and short URL bases length mismatch")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	updated := cloneMap(r.urlMap)
	shortURLs := make([]string, len(urlOrigins))

	for i := range urlOrigins {
		shortURLs[i] = reserveShortURL(updated, urlOrigins[i], shortURLBases[i])
	}

	if err := r.storage.Save(ctx, updated); err != nil {
		r.logger.Error("save batch to storage error", "error", err)
		return nil, err
	}

	r.urlMap = updated

	return shortURLs, nil
}

func (r *Repository) GetOriginURL(ctx context.Context, urlShort string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if result, ok := r.urlMap[urlShort]; ok {
		return result, nil
	}

	return "", ErrOriginURLNotFound
}

func reserveShortURL(urlMap map[string]string, urlOrigin, shortURLBase string) string {
	shortURL := shortURLBase

	for counter := 0; ; counter++ {
		if counter > 0 {
			shortURL = fmt.Sprintf("%s_%d", shortURLBase, counter)
		}

		existingURL, ok := urlMap[shortURL]
		if !ok || existingURL == urlOrigin {
			urlMap[shortURL] = urlOrigin
			return shortURL
		}
	}
}

func cloneMap(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}

	return result
}
