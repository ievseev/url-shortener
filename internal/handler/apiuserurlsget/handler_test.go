package apiuserurlsget

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ievseev/url-shortener/internal/auth"
	"github.com/ievseev/url-shortener/internal/model"
)

func TestHandlerHandleReturnsUnauthorizedWithoutUserID(t *testing.T) {
	handler := New("http://localhost:8080", &stubURLReader{}, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestHandlerHandleReturnsNoContent(t *testing.T) {
	handler := New("http://localhost:8080", &stubURLReader{}, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req = req.WithContext(auth.ContextWithUserID(req.Context(), "user-1"))

	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}

func TestHandlerHandleReturnsUserURLs(t *testing.T) {
	handler := New("http://localhost:8080", &stubURLReader{
		result: []model.UserURL{
			{ShortURL: "abc123", OriginalURL: "https://example.com"},
		},
	}, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req = req.WithContext(auth.ContextWithUserID(req.Context(), "user-1"))

	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response []Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if len(response) != 1 {
		t.Fatalf("expected 1 response item, got %d", len(response))
	}

	if response[0].ShortURL != "http://localhost:8080/abc123" {
		t.Fatalf("expected short URL %q, got %q", "http://localhost:8080/abc123", response[0].ShortURL)
	}
}

func TestHandlerHandleReturnsInternalServerErrorOnServiceError(t *testing.T) {
	handler := New("http://localhost:8080", &stubURLReader{
		err: errors.New("boom"),
	}, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req = req.WithContext(auth.ContextWithUserID(req.Context(), "user-1"))

	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

type stubURLReader struct {
	result []model.UserURL
	err    error
}

func (s *stubURLReader) GetUserURLs(ctx context.Context, userID string) ([]model.UserURL, error) {
	if s.err != nil {
		return nil, s.err
	}

	return append([]model.UserURL(nil), s.result...), nil
}
