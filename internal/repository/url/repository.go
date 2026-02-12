package url

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"sync"
)

var ErrOriginURLNotFound = errors.New("origin URL not found")

type Repository struct {
	mu       sync.Mutex
	filepath string
	urlMap   map[string]string
	logger   *slog.Logger
}

func NewStorage(logger *slog.Logger, filepath string) (*Repository, error) {
	repo := &Repository{
		logger:   logger,
		filepath: filepath,
		urlMap:   make(map[string]string),
	}

	if err := repo.loadFromFile(); err != nil {
		logger.Error("load from file error", "error", err)
		return nil, err
	}

	return repo, nil
}

func (r *Repository) SaveURLPair(ctx context.Context, urlOrigin, urlShort string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.urlMap[urlShort] = urlOrigin

	if err := r.saveToFile(); err != nil {
		r.logger.Error("save to file error", "error", err)
		return err
	}

	r.logger.Debug("saved url pair", "urlShort", urlShort, "urlOrigin", urlOrigin)

	return nil
}

func (r *Repository) GetOriginURL(ctx context.Context, urlShort string) (string, error) {
	if result, ok := r.urlMap[urlShort]; ok {
		return result, nil
	}

	return "", ErrOriginURLNotFound
}

func (r *Repository) loadFromFile() error {
	data, err := os.ReadFile(r.filepath)

	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if len(data) == 0 {
		return nil
	}

	return json.Unmarshal(data, &r.urlMap)
}

func (r *Repository) saveToFile() error {
	data, err := json.Marshal(r.urlMap)
	if err != nil {
		return err
	}

	return os.WriteFile(r.filepath, data, 0644)
}
