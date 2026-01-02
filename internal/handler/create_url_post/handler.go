package create_url_post

import (
	"io"
	"net/http"
)

const (
	contentTypeHeader    = "Content-Type"
	contentTypeTextPlain = "text/plain"
)

type UrlShortener interface {
	Create(url string) (string, error)
}

type CreateUrlPostHandler struct {
	UrlShortener UrlShortener
}

func New(urlShortener UrlShortener) *CreateUrlPostHandler {
	return &CreateUrlPostHandler{UrlShortener: urlShortener}
}

func (c *CreateUrlPostHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// check POST type of handler
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

	// validate content type
	if r.Header.Get(contentTypeHeader) != contentTypeTextPlain {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	url := string(body)

	shortUrl, err := c.UrlShortener.Create(url)
	if err != nil {
		// TODO сделать возврат ошибки в зависимости от типа
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set(contentTypeHeader, contentTypeTextPlain)
	w.Write([]byte(shortUrl))
}
