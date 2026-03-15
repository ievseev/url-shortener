package memory

import (
	"context"
	"sync"
)

type Storage struct {
	mu   sync.RWMutex
	data map[string]string
}

func New() *Storage {
	return &Storage{
		data: make(map[string]string),
	}
}

func (s *Storage) Save(ctx context.Context, data map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = clone(data)

	return nil
}

func (s *Storage) Load(ctx context.Context) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return clone(s.data), nil
}

func clone(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}

	return result
}
