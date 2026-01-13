package url

import (
	"context"
	"fmt"
	"sync"
)

type Repository struct {
	mu     sync.RWMutex
	urlMap map[string]string
}

func NewStorage() *Repository {
	return &Repository{urlMap: make(map[string]string)}
}

func (s *Repository) SaveURLPair(ctx context.Context, urlOrigin, urlShort string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.urlMap[urlShort] = urlOrigin

	// for debug: log saved value: key and value
	fmt.Printf("saved url pair: %s -> %s\n", urlShort, urlOrigin)

	// пока без ошибок, но потребуются в будущем, при работе с реальным хранилищем
	return nil
}

func (s *Repository) GetOriginURL(ctx context.Context, urlShort string) (string, error) {
	if result, ok := s.urlMap[urlShort]; ok {
		return result, nil
	}

	return "", nil
}
