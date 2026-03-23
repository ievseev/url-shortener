package urlshortener

import (
	"context"
	"errors"
	"testing"

	"github.com/ievseev/url-shortener/internal/model"
	urlrepo "github.com/ievseev/url-shortener/internal/repository/url"
)

func TestURLService_ShortenUsesRepositorySaveURL(t *testing.T) {
	repository := &stubRepository{
		saveURLResult: "c984d06a",
	}

	service, err := New(repository)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	shortURL, err := service.Shorten(context.Background(), "user-1", "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if shortURL != "c984d06a" {
		t.Fatalf("expected existing short URL, got %q", shortURL)
	}

	if repository.saveURLBase != "c984d06a" {
		t.Fatalf("expected generated short URL base %q, got %q", "c984d06a", repository.saveURLBase)
	}

	if repository.saveURLUserID != "user-1" {
		t.Fatalf("expected user ID %q, got %q", "user-1", repository.saveURLUserID)
	}
}

func TestURLService_ShortenReturnsRepositoryError(t *testing.T) {
	service, err := New(&stubRepository{
		saveURLErr: errors.New("repository is unavailable"),
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	_, err = service.Shorten(context.Background(), "user-1", "https://example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestURLService_ShortenReturnsConflictWithoutDroppingShortURL(t *testing.T) {
	service, err := New(&stubRepository{
		saveURLResult: "c984d06a",
		saveURLErr:    urlrepo.ErrOriginalURLConflict,
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	shortURL, err := service.Shorten(context.Background(), "user-1", "https://example.com")
	if !errors.Is(err, ErrorURLConflict) {
		t.Fatalf("expected conflict error, got %v", err)
	}

	if shortURL != "c984d06a" {
		t.Fatalf("expected existing short URL, got %q", shortURL)
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
		"user-1",
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

	_, err = service.ShortenBatch(context.Background(), "user-1", []string{"bad-url"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestURLService_GetUserURLs(t *testing.T) {
	repository := &stubRepository{
		userURLsResult: []model.UserURL{
			{ShortURL: "abc123", OriginalURL: "https://example.com"},
		},
	}

	service, err := New(repository)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	userURLs, err := service.GetUserURLs(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(userURLs) != 1 {
		t.Fatalf("expected 1 user URL, got %d", len(userURLs))
	}
}

func TestURLService_DeleteUserURLsUsesRepository(t *testing.T) {
	repository := &stubRepository{}

	service, err := New(repository)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	err = service.DeleteUserURLs(context.Background(), "user-1", []string{"abc123", "def456"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repository.deleteUserURLsUserID != "user-1" {
		t.Fatalf("expected user ID %q, got %q", "user-1", repository.deleteUserURLsUserID)
	}

	expected := []string{"abc123", "def456"}
	if len(repository.deleteUserURLsShortURLs) != len(expected) {
		t.Fatalf("expected %d short URLs, got %d", len(expected), len(repository.deleteUserURLsShortURLs))
	}
	for i := range expected {
		if repository.deleteUserURLsShortURLs[i] != expected[i] {
			t.Fatalf("expected short URL %q at index %d, got %q", expected[i], i, repository.deleteUserURLsShortURLs[i])
		}
	}
}

func TestURLService_DeleteUserURLsReturnsUnauthorizedWithoutUserID(t *testing.T) {
	service, err := New(&stubRepository{})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	err = service.DeleteUserURLs(context.Background(), "", []string{"abc123"})
	if !errors.Is(err, ErrorUnauthorized) {
		t.Fatalf("expected unauthorized error, got %v", err)
	}
}

func TestURLService_ExpandReturnsDeletedError(t *testing.T) {
	service, err := New(&stubRepository{
		originURLErr: urlrepo.ErrOriginURLDeleted,
	})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	_, err = service.Expand(context.Background(), "abc123")
	if !errors.Is(err, ErrorURLDeleted) {
		t.Fatalf("expected deleted error, got %v", err)
	}
}

func TestURLService_ReturnsUnauthorizedWithoutUserID(t *testing.T) {
	service, err := New(&stubRepository{})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	_, err = service.Shorten(context.Background(), "", "https://example.com")
	if !errors.Is(err, ErrorUnauthorized) {
		t.Fatalf("expected unauthorized error, got %v", err)
	}
}

type stubRepository struct {
	saveURLResult           string
	saveURLErr              error
	saveURLBase             string
	saveURLUserID           string
	saveURLBatchResult      []string
	saveURLBatchErr         error
	saveURLBatchBases       []string
	saveURLBatchUserID      string
	userURLsResult          []model.UserURL
	userURLsErr             error
	originURLResult         string
	originURLErr            error
	deleteUserURLsErr       error
	deleteUserURLsUserID    string
	deleteUserURLsShortURLs []string
}

func (s *stubRepository) SaveURL(ctx context.Context, userID, urlOrigin, shortURLBase string) (string, error) {
	s.saveURLUserID = userID
	s.saveURLBase = shortURLBase

	if s.saveURLErr != nil {
		return s.saveURLResult, s.saveURLErr
	}

	return s.saveURLResult, nil
}

func (s *stubRepository) SaveURLBatch(
	ctx context.Context,
	userID string,
	urlOrigins, shortURLBases []string,
) ([]string, error) {
	s.saveURLBatchUserID = userID
	s.saveURLBatchBases = append([]string(nil), shortURLBases...)

	if s.saveURLBatchErr != nil {
		return nil, s.saveURLBatchErr
	}

	return append([]string(nil), s.saveURLBatchResult...), nil
}

func (s *stubRepository) GetOriginURL(ctx context.Context, urlShort string) (string, error) {
	if s.originURLErr != nil {
		return "", s.originURLErr
	}

	return s.originURLResult, nil
}

func (s *stubRepository) GetUserURLs(ctx context.Context, userID string) ([]model.UserURL, error) {
	if s.userURLsErr != nil {
		return nil, s.userURLsErr
	}

	return append([]model.UserURL(nil), s.userURLsResult...), nil
}

func (s *stubRepository) Ping(ctx context.Context) error {
	return nil
}

func (s *stubRepository) DeleteUserURLs(ctx context.Context, userID string, shortURLs []string) error {
	s.deleteUserURLsUserID = userID
	s.deleteUserURLsShortURLs = append([]string(nil), shortURLs...)

	return s.deleteUserURLsErr
}
