package middleware

import (
	"log/slog"
	"net/http"

	"github.com/ievseev/url-shortener/internal/auth"
)

func AuthMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(auth.CookieName)
			switch {
			case err == http.ErrNoCookie:
				var ok bool
				r, ok = issueSignedCookie(w, r, logger)
				if !ok {
					return
				}
			case err != nil:
				logger.Error("read auth cookie error", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			case cookie.Value == "":
				next.ServeHTTP(w, r)
				return
			default:
				userID, decodeErr := auth.DecodeCookieValue(cookie.Value)
				if decodeErr != nil {
					var ok bool
					r, ok = issueSignedCookie(w, r, logger)
					if !ok {
						return
					}
				} else if userID != "" {
					r = r.WithContext(auth.ContextWithUserID(r.Context(), userID))
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func issueSignedCookie(w http.ResponseWriter, r *http.Request, logger *slog.Logger) (*http.Request, bool) {
	userID, err := auth.GenerateUserID()
	if err != nil {
		logger.Error("generate user id error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return r, false
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    auth.EncodeCookieValue(userID),
		Path:     "/",
		HttpOnly: true,
	})

	return r.WithContext(auth.ContextWithUserID(r.Context(), userID)), true
}
