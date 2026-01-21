package url

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

var ErrOriginURLNotFound = errors.New("origin URL not found")

type Repository struct {
	mu     sync.Mutex
	urlMap map[string]string
	logger *slog.Logger
}

func NewStorage(logger *slog.Logger) *Repository {
	return &Repository{
		logger: logger,
		urlMap: make(map[string]string),
	}
}

func (r *Repository) SaveURLPair(ctx context.Context, urlOrigin, urlShort string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.urlMap[urlShort] = urlOrigin

	r.logger.Debug("saved url pair", "urlShort", urlShort, "urlOrigin", urlOrigin)

	// пока без ошибок, но потребуются в будущем, при работе с реальным хранилищем
	return nil
}

func (r *Repository) GetOriginURL(ctx context.Context, urlShort string) (string, error) {
	if result, ok := r.urlMap[urlShort]; ok {
		return result, nil
	}

	return "", ErrOriginURLNotFound
}
