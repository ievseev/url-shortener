package memory

import (
	"context"
	"sync"

	urlrepo "github.com/ievseev/url-shortener/internal/repository/url"
)

type Storage struct {
	mu   sync.RWMutex
	data urlrepo.Snapshot
}

func New() *Storage {
	return &Storage{
		data: urlrepo.Snapshot{
			URLs:     make(map[string]string),
			UserURLs: make(map[string][]string),
		},
	}
}

func (s *Storage) Save(ctx context.Context, data urlrepo.Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = cloneSnapshot(data)

	return nil
}

func (s *Storage) Load(ctx context.Context) (urlrepo.Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return cloneSnapshot(s.data), nil
}

func cloneSnapshot(source urlrepo.Snapshot) urlrepo.Snapshot {
	result := urlrepo.Snapshot{
		URLs:     make(map[string]string, len(source.URLs)),
		UserURLs: make(map[string][]string, len(source.UserURLs)),
	}

	for key, value := range source.URLs {
		result.URLs[key] = value
	}

	for userID, shortURLs := range source.UserURLs {
		result.UserURLs[userID] = append([]string(nil), shortURLs...)
	}

	return result
}
