package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func RequestLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			next.ServeHTTP(w, r)

			logger.Info(
				"request",
				"uri", r.RequestURI,
				"method", r.Method,
				"duration", time.Since(start))
		})
	}
}
