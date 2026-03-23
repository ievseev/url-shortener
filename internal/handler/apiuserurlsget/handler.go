package apiuserurlsget

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/ievseev/url-shortener/internal/auth"
	"github.com/ievseev/url-shortener/internal/model"
)

const contentTypeApplicationJSON = "application/json"

type URLReader interface {
	GetUserURLs(ctx context.Context, userID string) ([]model.UserURL, error)
}

type Response struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type Handler struct {
	baseURL   string
	urlReader URLReader
	logger    *slog.Logger
}

func New(baseURL string, urlReader URLReader, logger *slog.Logger) *Handler {
	return &Handler{
		baseURL:   baseURL,
		urlReader: urlReader,
		logger:    logger,
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userURLs, err := h.urlReader.GetUserURLs(r.Context(), userID)
	if err != nil {
		h.logger.Error("get user urls error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(userURLs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	response := make([]Response, len(userURLs))
	for i, userURL := range userURLs {
		shortURL, err := url.JoinPath(h.baseURL, userURL.ShortURL)
		if err != nil {
			h.logger.Error("join path error", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response[i] = Response{
			ShortURL:    shortURL,
			OriginalURL: userURL.OriginalURL,
		}
	}

	payload, err := json.Marshal(response)
	if err != nil {
		h.logger.Error("json marshal error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentTypeApplicationJSON)
	w.WriteHeader(http.StatusOK)
	w.Write(payload)
}
