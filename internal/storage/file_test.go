package file

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestStorageLoadSupportsLegacyFormat(t *testing.T) {
	tempDir := t.TempDir()
	storagePath := filepath.Join(tempDir, "storage.json")
	if err := os.WriteFile(storagePath, []byte(`{"abc123":"https://example.com"}`), 0644); err != nil {
		t.Fatalf("write legacy data: %v", err)
	}

	storage := New(slog.Default(), storagePath)
	snapshot, err := storage.Load(context.Background())
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}

	if snapshot.URLs["abc123"] != "https://example.com" {
		t.Fatalf("expected legacy URL to be loaded, got %+v", snapshot.URLs)
	}

	if len(snapshot.UserURLs) != 0 {
		t.Fatalf("expected empty user URLs, got %+v", snapshot.UserURLs)
	}

	if len(snapshot.DeletedURLs) != 0 {
		t.Fatalf("expected empty deleted URLs, got %+v", snapshot.DeletedURLs)
	}

	if len(snapshot.Creators) != 0 {
		t.Fatalf("expected empty creators, got %+v", snapshot.Creators)
	}
}
