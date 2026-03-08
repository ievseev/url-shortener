//go:generate mockgen -source=handler.go -destination=mocks/mock_url_shortener.go -package=mocks

package shortenurlpost

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"

	urlshortenerservice "github.com/ievseev/url-shortener/internal/service/urlshortener"
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
	if err := validateRequestContentTypeValid(r); err != nil {
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

	originalURL := string(body)

	shortURL, err := c.URLShortener.Shorten(r.Context(), originalURL)
	if err != nil {
		if errors.Is(err, urlshortenerservice.ErrorURLConflict) {
			result, joinErr := url.JoinPath(c.baseURL, shortURL)
			if joinErr != nil {
				c.logger.Error("join path error", "error", joinErr)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Header().Set(contentTypeHeaderName, contentTypeTextPlain)
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(result))
			return
		}

		c.logger.Error("shorten service error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	result, err := url.JoinPath(c.baseURL, shortURL)
	if err != nil {
		c.logger.Error("join path error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set(contentTypeHeaderName, contentTypeTextPlain)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(result))
}

func validateRequestContentTypeValid(r *http.Request) error {
	contentTypeHeader := r.Header.Get(contentTypeHeaderName)
	// из всего заголовка проверяем только mime, остальная часть может меняться
	mimeType, _, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		return err
	}

	if mimeType != contentTypeTextPlain {
		return errors.New("invalid content type")
	}

	return nil
}
