//go:generate mockgen -source=handler.go -destination=mocks/mock_url_shortener.go -package=mocks

package expandurlget

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	urlshortenerservice "github.com/ievseev/url-shortener/internal/service/urlshortener"
)

type URLShortener interface {
	Expand(ctx context.Context, url string) (string, error)
}

type Handler struct {
	URLShortener URLShortener
	logger       *slog.Logger
}

func New(urlShortener URLShortener, logger *slog.Logger) *Handler {
	return &Handler{
		URLShortener: urlShortener,
		logger:       logger,
	}
}

func (c *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	shortURL := chi.URLParam(r, "id")

	expandedURL, err := c.URLShortener.Expand(r.Context(), shortURL)
	if err != nil {
		if errors.Is(err, urlshortenerservice.ErrorURLDeleted) {
			w.WriteHeader(http.StatusGone)
			return
		}

		c.logger.Error("expand service error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", expandedURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
