package url

import (
	"context"
	"errors"
	"log/slog"
	"testing"
)

func TestRepositorySaveURLAddsOwnershipForMultipleUsers(t *testing.T) {
	repository, err := New(slog.Default(), &stubStorage{})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}

	shortURL, err := repository.SaveURL(context.Background(), "user-1", "https://example.com", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if shortURL != "abc123" {
		t.Fatalf("expected short URL %q, got %q", "abc123", shortURL)
	}

	shortURL, err = repository.SaveURL(context.Background(), "user-2", "https://example.com", "abc123")
	if !errors.Is(err, ErrOriginalURLConflict) {
		t.Fatalf("expected original URL conflict, got %v", err)
	}

	if shortURL != "abc123" {
		t.Fatalf("expected short URL %q, got %q", "abc123", shortURL)
	}

	user1URLs, err := repository.GetUserURLs(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("get user-1 urls: %v", err)
	}
	user2URLs, err := repository.GetUserURLs(context.Background(), "user-2")
	if err != nil {
		t.Fatalf("get user-2 urls: %v", err)
	}

	if len(user1URLs) != 1 || len(user2URLs) != 1 {
		t.Fatalf("expected both users to have 1 URL, got %d and %d", len(user1URLs), len(user2URLs))
	}
}

func TestRepositorySaveURLDoesNotDuplicateUserHistory(t *testing.T) {
	repository, err := New(slog.Default(), &stubStorage{})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}

	_, _ = repository.SaveURL(context.Background(), "user-1", "https://example.com", "abc123")
	_, _ = repository.SaveURL(context.Background(), "user-1", "https://example.com", "abc123")

	userURLs, err := repository.GetUserURLs(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("get user urls: %v", err)
	}

	if len(userURLs) != 1 {
		t.Fatalf("expected 1 URL in history, got %d", len(userURLs))
	}
}

func TestRepositoryDeleteUserURLsMarksCreatorURLAsDeleted(t *testing.T) {
	repository, err := New(slog.Default(), &stubStorage{})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}

	_, err = repository.SaveURL(context.Background(), "user-1", "https://example.com", "abc123")
	if err != nil {
		t.Fatalf("save url: %v", err)
	}

	if err := repository.DeleteUserURLs(context.Background(), "user-1", []string{"abc123"}); err != nil {
		t.Fatalf("delete user urls: %v", err)
	}

	_, err = repository.GetOriginURL(context.Background(), "abc123")
	if !errors.Is(err, ErrOriginURLDeleted) {
		t.Fatalf("expected deleted error, got %v", err)
	}
}

func TestRepositoryDeleteUserURLsDoesNotDeleteForeignURL(t *testing.T) {
	repository, err := New(slog.Default(), &stubStorage{})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}

	_, err = repository.SaveURL(context.Background(), "user-1", "https://example.com", "abc123")
	if err != nil {
		t.Fatalf("save url: %v", err)
	}

	_, err = repository.SaveURL(context.Background(), "user-2", "https://example.com", "abc123")
	if !errors.Is(err, ErrOriginalURLConflict) {
		t.Fatalf("expected original URL conflict, got %v", err)
	}

	if err := repository.DeleteUserURLs(context.Background(), "user-2", []string{"abc123"}); err != nil {
		t.Fatalf("delete user urls: %v", err)
	}

	originalURL, err := repository.GetOriginURL(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("expected URL to stay accessible, got %v", err)
	}

	if originalURL != "https://example.com" {
		t.Fatalf("expected original URL %q, got %q", "https://example.com", originalURL)
	}
}

type stubStorage struct {
	snapshot Snapshot
}

func (s *stubStorage) Save(ctx context.Context, snapshot Snapshot) error {
	s.snapshot = Snapshot{
		URLs:        cloneMap(snapshot.URLs),
		UserURLs:    cloneUserURLs(snapshot.UserURLs),
		DeletedURLs: cloneDeletedURLs(snapshot.DeletedURLs),
		Creators:    cloneCreators(snapshot.Creators),
	}

	return nil
}

func (s *stubStorage) Load(ctx context.Context) (Snapshot, error) {
	return Snapshot{
		URLs:        cloneMap(s.snapshot.URLs),
		UserURLs:    cloneUserURLs(s.snapshot.UserURLs),
		DeletedURLs: cloneDeletedURLs(s.snapshot.DeletedURLs),
		Creators:    cloneCreators(s.snapshot.Creators),
	}, nil
}
