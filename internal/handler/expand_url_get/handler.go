//go:generate mockgen -source=handler.go -destination=mocks/mock_url_shortener.go -package=mocks

package expand_url_get

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type UrlShortener interface {
	Expand(ctx context.Context, url string) (string, error)
}

type ExpandUrlGetHandler struct {
	UrlShortener UrlShortener
}

func New(urlShortener UrlShortener) *ExpandUrlGetHandler {
	return &ExpandUrlGetHandler{
		UrlShortener: urlShortener,
	}
}

func (c *ExpandUrlGetHandler) Handle(w http.ResponseWriter, r *http.Request) {
	shortURL := chi.URLParam(r, "id")

	expandedURL, err := c.UrlShortener.Expand(r.Context(), shortURL)
	if err != nil {
		// TODO сделать возврат ошибки в зависимости от типа
		//slog.Error("get expand service error: ", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", expandedURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
