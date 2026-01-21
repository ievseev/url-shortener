//go:generate mockgen -source=handler.go -destination=mocks/mock_url_shortener.go -package=mocks

package expandUrlGet

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type UrlShortener interface {
	Expand(ctx context.Context, url string) (string, error)
}

type ExpandUrlGetHandler struct {
	UrlShortener UrlShortener
	logger       *slog.Logger
}

func New(urlShortener UrlShortener, logger *slog.Logger) *ExpandUrlGetHandler {
	return &ExpandUrlGetHandler{
		UrlShortener: urlShortener,
		logger:       logger,
	}
}

func (c *ExpandUrlGetHandler) Handle(w http.ResponseWriter, r *http.Request) {
	shortURL := chi.URLParam(r, "id")

	expandedURL, err := c.UrlShortener.Expand(r.Context(), shortURL)
	if err != nil {
		// TODO сделать возврат ошибки в зависимости от типа
		c.logger.Error("expand service error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", expandedURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
