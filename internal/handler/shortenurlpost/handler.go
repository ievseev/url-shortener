//go:generate mockgen -source=handler.go -destination=mocks/mock_url_shortener.go -package=mocks

package shortenurlpost

import (
	"context"
	"io"
	"log/slog"
	"mime"
	"net/http"
)

const (
	contentTypeTextPlain  = "text/plain"
	contentTypeHeaderName = "Content-Type"
)

type URLShortener interface {
	Shorten(ctx context.Context, url string) (string, error)
}

type Handler struct {
	baseURL      string
	URLShortener URLShortener
	logger       *slog.Logger
}

func New(baseURL string, urlShortener URLShortener, logger *slog.Logger) *Handler {
	return &Handler{
		baseURL:      baseURL,
		URLShortener: urlShortener,
		logger:       logger,
	}
}

func (c *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	isRequestContentTypeValid, err := validateRequestContentTypeValid(r)

	if err != nil || !isRequestContentTypeValid {
		c.logger.Error("invalid content type", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		c.logger.Error("read request body error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	url := string(body)

	shortUrl, err := c.URLShortener.Shorten(r.Context(), url)
	if err != nil {
		c.logger.Error("shorten service error", "error", err)
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
