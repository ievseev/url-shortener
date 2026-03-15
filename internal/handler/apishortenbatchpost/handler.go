package apishortenbatchpost

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
)

const (
	contentTypeApplicationJSON = "application/json"
	contentTypeHeaderName      = "Content-Type"
)

type URLShortener interface {
	ShortenBatch(ctx context.Context, urls []string) ([]string, error)
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

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	if err := validateRequestContentTypeValid(r); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("read request body error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var request []Request
	if err := json.Unmarshal(body, &request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(request) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	urls := make([]string, len(request))
	for i, item := range request {
		urls[i] = item.OriginalURL
	}

	shortURLs, err := h.URLShortener.ShortenBatch(r.Context(), urls)
	if err != nil {
		h.logger.Error("shorten batch service error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := make([]Response, len(request))
	for i, item := range request {
		shortURL, err := url.JoinPath(h.baseURL, shortURLs[i])
		if err != nil {
			h.logger.Error("join path error", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response[i] = Response{
			CorrelationID: item.CorrelationID,
			ShortURL:      shortURL,
		}
	}

	payload, err := json.Marshal(response)
	if err != nil {
		h.logger.Error("json marshal error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set(contentTypeHeaderName, contentTypeApplicationJSON)
	w.WriteHeader(http.StatusCreated)
	w.Write(payload)
}

func validateRequestContentTypeValid(r *http.Request) error {
	contentTypeHeader := r.Header.Get(contentTypeHeaderName)
	mimeType, _, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		return err
	}

	if mimeType != contentTypeApplicationJSON {
		return fmt.Errorf("invalid content type: got %q", mimeType)
	}

	return nil
}
