//go:generate mockgen -source=handler.go -destination=mocks/mock_url_shortener.go -package=mocks

package apishortenurlpost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"

	"github.com/ievseev/url-shortener/internal/auth"
	urlshortenerservice "github.com/ievseev/url-shortener/internal/service/urlshortener"
)

const (
	contentTypeApplicationJSON = "application/json"
	contentTypeHeaderName      = "Content-Type"
)

type URLShortener interface {
	Shorten(ctx context.Context, userID, url string) (string, error)
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
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		c.logger.Error("read request body error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var request Request
	err = json.Unmarshal(body, &request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	shortURL, err := c.URLShortener.Shorten(r.Context(), userID, request.URL)
	if err != nil {
		if errors.Is(err, urlshortenerservice.ErrorURLConflict) {
			c.writeResponse(w, shortURL, http.StatusConflict)
			return
		}

		c.logger.Error("shorten service error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.writeResponse(w, shortURL, http.StatusCreated)
}

func validateRequestContentTypeValid(r *http.Request) error {
	contentTypeHeader := r.Header.Get(contentTypeHeaderName)
	// из всего заголовка проверяем только mime, остальная часть может меняться
	mimeType, _, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		return err
	}

	if mimeType != contentTypeApplicationJSON {
		return fmt.Errorf("invalid content type: got %q", mimeType)
	}

	return nil
}

func (c *Handler) writeResponse(w http.ResponseWriter, shortURL string, statusCode int) {
	result, err := url.JoinPath(c.baseURL, shortURL)
	if err != nil {
		c.logger.Error("join path error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response, err := json.Marshal(Response{Result: result})
	if err != nil {
		c.logger.Error("json marshal error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set(contentTypeHeaderName, contentTypeApplicationJSON)
	w.WriteHeader(statusCode)
	w.Write(response)
}
