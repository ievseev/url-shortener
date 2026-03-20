package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ievseev/url-shortener/internal/auth"
)

func TestAuthMiddleware_IssuesCookieWhenMissing(t *testing.T) {
	var userID string

	handler := AuthMiddleware(nilLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		userID, ok = auth.UserIDFromContext(r.Context())
		if !ok {
			t.Fatal("expected user ID in context")
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if userID == "" {
		t.Fatal("expected non-empty user ID")
	}

	if len(rr.Result().Cookies()) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(rr.Result().Cookies()))
	}
}

func TestAuthMiddleware_ReusesValidCookie(t *testing.T) {
	signedValue := auth.EncodeCookieValue("user-1")
	var userID string

	handler := AuthMiddleware(nilLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		userID, ok = auth.UserIDFromContext(r.Context())
		if !ok {
			t.Fatal("expected user ID in context")
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: signedValue})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if userID != "user-1" {
		t.Fatalf("expected user ID %q, got %q", "user-1", userID)
	}

	if len(rr.Result().Cookies()) != 0 {
		t.Fatalf("expected no new cookies, got %d", len(rr.Result().Cookies()))
	}
}

func TestAuthMiddleware_ReissuesCookieWhenInvalid(t *testing.T) {
	var userID string

	handler := AuthMiddleware(nilLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		userID, ok = auth.UserIDFromContext(r.Context())
		if !ok {
			t.Fatal("expected user ID in context")
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "broken"})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if userID == "" {
		t.Fatal("expected regenerated user ID")
	}

	if len(rr.Result().Cookies()) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(rr.Result().Cookies()))
	}
}

func TestAuthMiddleware_LeavesEmptyCookieUnauthorized(t *testing.T) {
	handler := AuthMiddleware(nilLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.UserIDFromContext(r.Context()); ok {
			t.Fatal("did not expect user ID in context")
		}

		w.WriteHeader(http.StatusUnauthorized)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: ""})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	if len(rr.Result().Cookies()) != 0 {
		t.Fatalf("expected no new cookies, got %d", len(rr.Result().Cookies()))
	}
}

func nilLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
