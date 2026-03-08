package urlshortener

import (
	"context"
	"errors"
	"testing"

	URLRepo "github.com/ievseev/url-shortener/internal/repository/url"
)

func TestURLService_ShortenReturnsExistingShortURLForSameOrigin(t *testing.T) {
	repository := &stubRepository{
		urls: map[string]string{
			"c984d06a": "https://example.com",
		},
	}

	service, err := New(repository)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	shortURL, err := service.Shorten(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if shortURL != "c984d06a" {
		t.Fatalf("expected existing short URL, got %q", shortURL)
	}
}

func TestURLService_ShortenReturnsRepositoryErrorFromCollisionCheck(t *testing.T) {
	service, err := New(&stubRepository{
		getErr: errors.New("repository is unavailable"),
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	_, err = service.Shorten(context.Background(), "https://example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

type stubRepository struct {
	urls   map[string]string
	getErr error
}

func (s *stubRepository) SaveURLPair(ctx context.Context, urlOrigin, urlShort string) error {
	if s.urls == nil {
		s.urls = make(map[string]string)
	}

	s.urls[urlShort] = urlOrigin

	return nil
}

func (s *stubRepository) GetOriginURL(ctx context.Context, urlShort string) (string, error) {
	if s.getErr != nil {
		return "", s.getErr
	}

	originalURL, ok := s.urls[urlShort]
	if !ok {
		return "", URLRepo.ErrOriginURLNotFound
	}

	return originalURL, nil
}
