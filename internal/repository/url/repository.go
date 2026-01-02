package url

import (
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

func (s *Repository) SaveURLPair(urlOrigin, urlShort string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.urlMap[urlOrigin] = urlShort

	// for debug: log saved value: key and value
	fmt.Printf("saved url pair: %s -> %s\n", urlOrigin, urlShort)

	// пока без ошибок, но потребуются в будущем, при работе с реальным хранилищем
	return nil
}

func (s *Repository) GetURLOrigin(urlShort string) (string, error) {
	if result, ok := s.urlMap[urlShort]; ok {
		return result, nil
	}

	return "", nil
}
