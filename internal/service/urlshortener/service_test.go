package urlshortener

import (
	"context"
	"errors"
	"testing"
)

func TestURLService_ShortenUsesRepositorySaveURL(t *testing.T) {
	repository := &stubRepository{
		saveURLResult: "c984d06a",
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

	if repository.saveURLBase != "c984d06a" {
		t.Fatalf("expected generated short URL base %q, got %q", "c984d06a", repository.saveURLBase)
	}
}

func TestURLService_ShortenReturnsRepositoryError(t *testing.T) {
	service, err := New(&stubRepository{
		saveURLErr: errors.New("repository is unavailable"),
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	_, err = service.Shorten(context.Background(), "https://example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestURLService_ShortenBatch(t *testing.T) {
	repository := &stubRepository{
		saveURLBatchResult: []string{"c984d06a", "85d7c0fa"},
	}

	service, err := New(repository)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	shortURLs, err := service.ShortenBatch(
		context.Background(),
		[]string{"https://example.com", "https://example.org"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"c984d06a", "85d7c0fa"}
	for i := range expected {
		if shortURLs[i] != expected[i] {
			t.Fatalf("expected short URL %q at index %d, got %q", expected[i], i, shortURLs[i])
		}
	}

	if len(repository.saveURLBatchBases) != 2 {
		t.Fatalf("expected 2 generated short URL bases, got %d", len(repository.saveURLBatchBases))
	}
}

func TestURLService_ShortenBatchReturnsValidationError(t *testing.T) {
	service, err := New(&stubRepository{})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	_, err = service.ShortenBatch(context.Background(), []string{"bad-url"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

type stubRepository struct {
	saveURLResult      string
	saveURLErr         error
	saveURLBase        string
	saveURLBatchResult []string
	saveURLBatchErr    error
	saveURLBatchBases  []string
}

func (s *stubRepository) SaveURL(ctx context.Context, urlOrigin, shortURLBase string) (string, error) {
	s.saveURLBase = shortURLBase

	if s.saveURLErr != nil {
		return "", s.saveURLErr
	}

	return s.saveURLResult, nil
}

func (s *stubRepository) SaveURLBatch(
	ctx context.Context,
	urlOrigins, shortURLBases []string,
) ([]string, error) {
	s.saveURLBatchBases = append([]string(nil), shortURLBases...)

	if s.saveURLBatchErr != nil {
		return nil, s.saveURLBatchErr
	}

	return append([]string(nil), s.saveURLBatchResult...), nil
}

func (s *stubRepository) GetOriginURL(ctx context.Context, urlShort string) (string, error) {
	return "", nil
}
