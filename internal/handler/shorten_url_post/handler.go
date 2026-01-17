//go:generate mockgen -source=handler.go -destination=mocks/mock_url_shortener.go -package=mocks

package shorten_url_post

import (
	"context"
	"io"
	"mime"
	"net/http"
)

const (
	contentTypeTextPlain  = "text/plain"
	contentTypeHeaderName = "Content-Type"
)

type UrlShortener interface {
	Shorten(ctx context.Context, url string) (string, error)
}

type ShortenUrlPostHandler struct {
	baseURL      string
	UrlShortener UrlShortener
}

func New(baseURL string, urlShortener UrlShortener) *ShortenUrlPostHandler {
	return &ShortenUrlPostHandler{
		baseURL:      baseURL,
		UrlShortener: urlShortener,
	}
}

func (c *ShortenUrlPostHandler) Handle(w http.ResponseWriter, r *http.Request) {
	isRequestContentTypeValid, err := validateRequestContentTypeValid(r)

	if err != nil || !isRequestContentTypeValid {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	url := string(body)

	shortUrl, err := c.UrlShortener.Shorten(r.Context(), url)
	if err != nil {
		// TODO сделать возврат ошибки в зависимости от типа
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set(contentTypeHeaderName, contentTypeTextPlain)
	w.WriteHeader(http.StatusCreated)

	w.Write([]byte(c.baseURL + "/" + shortUrl))
}

func validateRequestContentTypeValid(r *http.Request) (bool, error) {
	contentTypeHeader := r.Header.Get(contentTypeHeaderName)
	// из всего заголовка проверяем только mime, остальная часть может меняться
	mimeType, _, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		return false, err
	}

	if mimeType != contentTypeTextPlain {
		return false, nil
	}

	return true, nil
}
