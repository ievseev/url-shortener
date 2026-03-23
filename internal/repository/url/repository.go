package url

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/ievseev/url-shortener/internal/model"
)

var (
	ErrOriginURLNotFound   = errors.New("origin URL not found")
	ErrOriginURLDeleted    = errors.New("origin URL deleted")
	ErrOriginalURLConflict = errors.New("original URL conflict")
)

type Storage interface {
	Save(ctx context.Context, snapshot Snapshot) error
	Load(ctx context.Context) (Snapshot, error)
}

type Repository struct {
	mu          sync.RWMutex
	urlMap      map[string]string
	userURLs    map[string][]string
	deletedURLs map[string]bool
	creators    map[string]string
	storage     Storage
	logger      *slog.Logger
}

func New(logger *slog.Logger, storage Storage) (*Repository, error) {
	repo := &Repository{
		logger:      logger,
		storage:     storage,
		urlMap:      make(map[string]string),
		userURLs:    make(map[string][]string),
		deletedURLs: make(map[string]bool),
		creators:    make(map[string]string),
	}

	snapshot, err := storage.Load(context.Background())
	if err != nil {
		logger.Error("load from storage error", "error", err)
		return nil, err
	}
	if snapshot.URLs != nil {
		repo.urlMap = snapshot.URLs
	}
	if snapshot.UserURLs != nil {
		repo.userURLs = snapshot.UserURLs
	}
	if snapshot.DeletedURLs != nil {
		repo.deletedURLs = snapshot.DeletedURLs
	}
	if snapshot.Creators != nil {
		repo.creators = snapshot.Creators
	}

	return repo, nil
}

func (r *Repository) SaveURL(ctx context.Context, userID, urlOrigin, shortURLBase string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	updatedURLs := cloneMap(r.urlMap)
	updatedUserURLs := cloneUserURLs(r.userURLs)
	updatedDeletedURLs := cloneDeletedURLs(r.deletedURLs)
	updatedCreators := cloneCreators(r.creators)

	if shortURL, found := findShortURLByOriginal(updatedURLs, urlOrigin); found {
		addUserShortURL(updatedUserURLs, userID, shortURL)
		if updatedCreators[shortURL] == "" {
			updatedCreators[shortURL] = userID
		}

		if err := r.storage.Save(ctx, Snapshot{
			URLs:        updatedURLs,
			UserURLs:    updatedUserURLs,
			DeletedURLs: updatedDeletedURLs,
			Creators:    updatedCreators,
		}); err != nil {
			r.logger.Error("save to storage error", "error", err)
			return "", err
		}

		r.urlMap = updatedURLs
		r.userURLs = updatedUserURLs
		r.deletedURLs = updatedDeletedURLs
		r.creators = updatedCreators

		return shortURL, ErrOriginalURLConflict
	}

	shortURL := reserveShortURL(updatedURLs, urlOrigin, shortURLBase)
	addUserShortURL(updatedUserURLs, userID, shortURL)
	delete(updatedDeletedURLs, shortURL)
	updatedCreators[shortURL] = userID

	if err := r.storage.Save(ctx, Snapshot{
		URLs:        updatedURLs,
		UserURLs:    updatedUserURLs,
		DeletedURLs: updatedDeletedURLs,
		Creators:    updatedCreators,
	}); err != nil {
		r.logger.Error("save to storage error", "error", err)
		return "", err
	}

	r.urlMap = updatedURLs
	r.userURLs = updatedUserURLs
	r.deletedURLs = updatedDeletedURLs
	r.creators = updatedCreators
	r.logger.Debug("saved url pair", "urlShort", shortURL, "urlOrigin", urlOrigin)

	return shortURL, nil
}

func (r *Repository) SaveURLBatch(
	ctx context.Context,
	userID string,
	urlOrigins, shortURLBases []string,
) ([]string, error) {
	if len(urlOrigins) != len(shortURLBases) {
		return nil, errors.New("url origins and short URL bases length mismatch")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	updatedURLs := cloneMap(r.urlMap)
	updatedUserURLs := cloneUserURLs(r.userURLs)
	updatedDeletedURLs := cloneDeletedURLs(r.deletedURLs)
	updatedCreators := cloneCreators(r.creators)
	shortURLs := make([]string, len(urlOrigins))

	for i := range urlOrigins {
		if shortURL, found := findShortURLByOriginal(updatedURLs, urlOrigins[i]); found {
			shortURLs[i] = shortURL
			addUserShortURL(updatedUserURLs, userID, shortURL)
			if updatedCreators[shortURL] == "" {
				updatedCreators[shortURL] = userID
			}
			continue
		}

		shortURLs[i] = reserveShortURL(updatedURLs, urlOrigins[i], shortURLBases[i])
		addUserShortURL(updatedUserURLs, userID, shortURLs[i])
		delete(updatedDeletedURLs, shortURLs[i])
		updatedCreators[shortURLs[i]] = userID
	}

	if err := r.storage.Save(ctx, Snapshot{
		URLs:        updatedURLs,
		UserURLs:    updatedUserURLs,
		DeletedURLs: updatedDeletedURLs,
		Creators:    updatedCreators,
	}); err != nil {
		r.logger.Error("save batch to storage error", "error", err)
		return nil, err
	}

	r.urlMap = updatedURLs
	r.userURLs = updatedUserURLs
	r.deletedURLs = updatedDeletedURLs
	r.creators = updatedCreators

	return shortURLs, nil
}

func (r *Repository) GetOriginURL(ctx context.Context, urlShort string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if result, ok := r.urlMap[urlShort]; ok {
		if r.deletedURLs[urlShort] {
			return "", ErrOriginURLDeleted
		}

		return result, nil
	}

	return "", ErrOriginURLNotFound
}

func (r *Repository) Ping(ctx context.Context) error {
	return nil
}

func (r *Repository) GetUserURLs(ctx context.Context, userID string) ([]model.UserURL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	shortURLs := r.userURLs[userID]
	result := make([]model.UserURL, 0, len(shortURLs))

	for _, shortURL := range shortURLs {
		originalURL, ok := r.urlMap[shortURL]
		if !ok {
			continue
		}

		result = append(result, model.UserURL{
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		})
	}

	return result, nil
}

func (r *Repository) DeleteUserURLs(ctx context.Context, userID string, shortURLs []string) error {
	if len(shortURLs) == 0 {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	updatedDeletedURLs := cloneDeletedURLs(r.deletedURLs)
	changed := false

	for _, shortURL := range shortURLs {
		if r.creators[shortURL] != userID {
			continue
		}
		if _, ok := r.urlMap[shortURL]; !ok {
			continue
		}
		if updatedDeletedURLs[shortURL] {
			continue
		}

		updatedDeletedURLs[shortURL] = true
		changed = true
	}

	if !changed {
		return nil
	}

	if err := r.storage.Save(ctx, Snapshot{
		URLs:        cloneMap(r.urlMap),
		UserURLs:    cloneUserURLs(r.userURLs),
		DeletedURLs: updatedDeletedURLs,
		Creators:    cloneCreators(r.creators),
	}); err != nil {
		r.logger.Error("delete user urls error", "error", err)
		return err
	}

	r.deletedURLs = updatedDeletedURLs

	return nil
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

func cloneUserURLs(source map[string][]string) map[string][]string {
	result := make(map[string][]string, len(source))

	for userID, shortURLs := range source {
		result[userID] = append([]string(nil), shortURLs...)
	}

	return result
}

func cloneDeletedURLs(source map[string]bool) map[string]bool {
	result := make(map[string]bool, len(source))
	for shortURL, deleted := range source {
		result[shortURL] = deleted
	}

	return result
}

func cloneCreators(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for shortURL, userID := range source {
		result[shortURL] = userID
	}

	return result
}

func findShortURLByOriginal(urlMap map[string]string, urlOrigin string) (string, bool) {
	for shortURL, existingOrigin := range urlMap {
		if existingOrigin == urlOrigin {
			return shortURL, true
		}
	}

	return "", false
}

func addUserShortURL(userURLs map[string][]string, userID, shortURL string) {
	existing := userURLs[userID]
	for _, current := range existing {
		if current == shortURL {
			return
		}
	}

	userURLs[userID] = append(existing, shortURL)
}
