package shorten_url_post

import (
	"io"
	"log/slog"
	"mime"
	"net/http"
)

const (
	contentTypeTextPlain  = "text/plain"
	contentTypeHeaderName = "Content-Type"
)

type UrlShortener interface {
	Shorten(url string) (string, error)
}

type ShortenUrlPostHandler struct {
	UrlShortener UrlShortener
}

func New(urlShortener UrlShortener) *ShortenUrlPostHandler {
	return &ShortenUrlPostHandler{UrlShortener: urlShortener}
}

func (c *ShortenUrlPostHandler) Handle(w http.ResponseWriter, r *http.Request) {
	slog.Info("got shorten url request")
	slog.Info("got expand url request",
		"full_url", getFullURL(r),
	)

	isRequestContentTypeValid, err := validateRequestContentTypeValid(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !isRequestContentTypeValid {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	url := string(body)

	shortUrl, err := c.UrlShortener.Shorten(url)
	if err != nil {
		// TODO сделать возврат ошибки в зависимости от типа
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set(contentTypeHeaderName, contentTypeTextPlain)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(getFullURL(r) + shortUrl))
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

func getFullURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host + r.URL.String()
}
