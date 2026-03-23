package apiuserurlsdelete

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/ievseev/url-shortener/internal/auth"
)

func TestHandlerHandleAcceptsDeleteRequest(t *testing.T) {
	deleter := &stubURLDeleter{
		calls: make(chan deleteCall, 1),
	}
	handler := New(deleter, slog.Default())

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBufferString(`["abc123","def456"]`))
	req.Header.Set(contentTypeHeaderName, contentTypeApplicationJSON)
	req = req.WithContext(auth.ContextWithUserID(req.Context(), "user-1"))

	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, recorder.Code)
	}

	select {
	case call := <-deleter.calls:
		if call.userID != "user-1" {
			t.Fatalf("expected user ID %q, got %q", "user-1", call.userID)
		}

		expected := []string{"abc123", "def456"}
		if !reflect.DeepEqual(call.shortURLs, expected) {
			t.Fatalf("expected short URLs %v, got %v", expected, call.shortURLs)
		}
	case <-time.After(time.Second):
		t.Fatal("expected delete call")
	}
}

func TestHandlerHandleRejectsInvalidContentType(t *testing.T) {
	deleter := &stubURLDeleter{
		calls: make(chan deleteCall, 1),
	}
	handler := New(deleter, slog.Default())

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBufferString(`["abc123"]`))
	req.Header.Set(contentTypeHeaderName, "text/plain")
	req = req.WithContext(auth.ContextWithUserID(req.Context(), "user-1"))

	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	assertNoDeleteCall(t, deleter.calls)
}

func TestHandlerHandleRejectsInvalidJSON(t *testing.T) {
	deleter := &stubURLDeleter{
		calls: make(chan deleteCall, 1),
	}
	handler := New(deleter, slog.Default())

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBufferString(`["abc123"`))
	req.Header.Set(contentTypeHeaderName, contentTypeApplicationJSON)
	req = req.WithContext(auth.ContextWithUserID(req.Context(), "user-1"))

	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	assertNoDeleteCall(t, deleter.calls)
}

func TestHandlerHandleRejectsUnauthorizedRequest(t *testing.T) {
	deleter := &stubURLDeleter{
		calls: make(chan deleteCall, 1),
	}
	handler := New(deleter, slog.Default())

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBufferString(`["abc123"]`))
	req.Header.Set(contentTypeHeaderName, contentTypeApplicationJSON)

	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	assertNoDeleteCall(t, deleter.calls)
}

type deleteCall struct {
	userID    string
	shortURLs []string
}

type stubURLDeleter struct {
	calls chan deleteCall
}

func (s *stubURLDeleter) DeleteUserURLs(ctx context.Context, userID string, shortURLs []string) error {
	s.calls <- deleteCall{
		userID:    userID,
		shortURLs: append([]string(nil), shortURLs...),
	}

	return nil
}

func assertNoDeleteCall(t *testing.T, calls <-chan deleteCall) {
	t.Helper()

	select {
	case call := <-calls:
		t.Fatalf("unexpected delete call: %+v", call)
	case <-time.After(100 * time.Millisecond):
	}
}
