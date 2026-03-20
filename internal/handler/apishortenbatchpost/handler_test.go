package apishortenbatchpost

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/ievseev/url-shortener/internal/auth"
)

func TestHandlerHandleSuccess(t *testing.T) {
	shortener := &stubURLShortener{
		shortenBatchResult: []string{"abc123", "def456"},
	}
	handler := New("http://localhost:8080", shortener, slog.Default())

	requestBody, err := json.Marshal([]Request{
		{
			CorrelationID: "first",
			OriginalURL:   "https://example.com/1",
		},
		{
			CorrelationID: "second",
			OriginalURL:   "https://example.com/2",
		},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBuffer(requestBody))
	req.Header.Set(contentTypeHeaderName, contentTypeApplicationJSON)
	req = req.WithContext(auth.ContextWithUserID(req.Context(), "user-1"))

	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	if got := recorder.Header().Get(contentTypeHeaderName); got != contentTypeApplicationJSON {
		t.Fatalf("expected content type %q, got %q", contentTypeApplicationJSON, got)
	}

	expectedURLs := []string{"https://example.com/1", "https://example.com/2"}
	if !reflect.DeepEqual(shortener.receivedURLs, expectedURLs) {
		t.Fatalf("expected URLs %+v, got %+v", expectedURLs, shortener.receivedURLs)
	}
	if shortener.receivedUserID != "user-1" {
		t.Fatalf("expected user ID %q, got %q", "user-1", shortener.receivedUserID)
	}

	var response []Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	expectedResponse := []Response{
		{
			CorrelationID: "first",
			ShortURL:      "http://localhost:8080/abc123",
		},
		{
			CorrelationID: "second",
			ShortURL:      "http://localhost:8080/def456",
		},
	}
	if !reflect.DeepEqual(response, expectedResponse) {
		t.Fatalf("expected response %+v, got %+v", expectedResponse, response)
	}
}

func TestHandlerHandleRejectsInvalidContentType(t *testing.T) {
	shortener := &stubURLShortener{}
	handler := New("http://localhost:8080", shortener, slog.Default())

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBufferString(`[]`))
	req.Header.Set(contentTypeHeaderName, "text/plain")
	req = req.WithContext(auth.ContextWithUserID(req.Context(), "user-1"))

	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	if shortener.called {
		t.Fatal("shortener must not be called")
	}
}

func TestHandlerHandleRejectsInvalidJSON(t *testing.T) {
	shortener := &stubURLShortener{}
	handler := New("http://localhost:8080", shortener, slog.Default())

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBufferString(`[{"correlation_id":"first"`))
	req.Header.Set(contentTypeHeaderName, contentTypeApplicationJSON)
	req = req.WithContext(auth.ContextWithUserID(req.Context(), "user-1"))

	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	if shortener.called {
		t.Fatal("shortener must not be called")
	}
}

func TestHandlerHandleRejectsEmptyBatch(t *testing.T) {
	shortener := &stubURLShortener{}
	handler := New("http://localhost:8080", shortener, slog.Default())

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBufferString(`[]`))
	req.Header.Set(contentTypeHeaderName, contentTypeApplicationJSON)
	req = req.WithContext(auth.ContextWithUserID(req.Context(), "user-1"))

	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	if shortener.called {
		t.Fatal("shortener must not be called")
	}
}

func TestHandlerHandleReturnsInternalServerErrorOnServiceError(t *testing.T) {
	shortener := &stubURLShortener{
		shortenBatchErr: errors.New("storage error"),
	}
	handler := New("http://localhost:8080", shortener, slog.Default())

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten/batch",
		bytes.NewBufferString(`[{"correlation_id":"first","original_url":"https://example.com"}]`),
	)
	req.Header.Set(contentTypeHeaderName, contentTypeApplicationJSON)
	req = req.WithContext(auth.ContextWithUserID(req.Context(), "user-1"))

	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
}

type stubURLShortener struct {
	shortenBatchResult []string
	shortenBatchErr    error
	receivedURLs       []string
	receivedUserID     string
	called             bool
}

func (s *stubURLShortener) ShortenBatch(ctx context.Context, userID string, urls []string) ([]string, error) {
	s.called = true
	s.receivedUserID = userID
	s.receivedURLs = append([]string(nil), urls...)

	if s.shortenBatchErr != nil {
		return nil, s.shortenBatchErr
	}

	return append([]string(nil), s.shortenBatchResult...), nil
}

func TestHandlerHandleRejectsUnauthorizedRequest(t *testing.T) {
	shortener := &stubURLShortener{}
	handler := New("http://localhost:8080", shortener, slog.Default())

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten/batch",
		bytes.NewBufferString(`[{"correlation_id":"first","original_url":"https://example.com"}]`),
	)
	req.Header.Set(contentTypeHeaderName, contentTypeApplicationJSON)

	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	if shortener.called {
		t.Fatal("shortener must not be called")
	}
}
