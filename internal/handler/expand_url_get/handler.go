package expand_url_get

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

const (
	contentTypeTextPlain  = "text/plain"
	contentTypeHeaderName = "Content-Type"
)

type UrlShortener interface {
	Expand(url string) (string, error)
}

type ExpandUrlGetHandler struct {
	UrlShortener UrlShortener
}

func New(urlShortener UrlShortener) *ExpandUrlGetHandler {
	return &ExpandUrlGetHandler{UrlShortener: urlShortener}
}

func (c *ExpandUrlGetHandler) Handle(w http.ResponseWriter, r *http.Request) {
	slog.Info("got expand url request")
	slog.Info("request url: ", r.URL.String())

	shortURL := chi.URLParam(r, "id")

	expandedURL, err := c.UrlShortener.Expand(shortURL)
	if err != nil {
		// TODO сделать возврат ошибки в зависимости от типа
		slog.Error("get expand service error: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", expandedURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
