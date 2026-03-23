package apiuserurlsdelete

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"

	"github.com/ievseev/url-shortener/internal/auth"
)

const (
	contentTypeApplicationJSON = "application/json"
	contentTypeHeaderName      = "Content-Type"
)

type URLDeleter interface {
	DeleteUserURLs(ctx context.Context, userID string, shortURLs []string) error
}

type Handler struct {
	urlDeleter URLDeleter
	logger     *slog.Logger
}

func New(urlDeleter URLDeleter, logger *slog.Logger) *Handler {
	return &Handler{
		urlDeleter: urlDeleter,
		logger:     logger,
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

	var shortURLs []string
	if err := json.Unmarshal(body, &shortURLs); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	shortURLsCopy := append([]string(nil), shortURLs...)
	go func() {
		if err := h.urlDeleter.DeleteUserURLs(context.Background(), userID, shortURLsCopy); err != nil {
			h.logger.Error("delete user urls error", "error", err)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
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
