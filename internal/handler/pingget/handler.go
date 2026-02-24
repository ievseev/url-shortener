//go:generate mockgen -source=handler.go -destination=mocks/mock_url_shortener.go -package=mocks

package expandurlget

import (
	"context"
	"log/slog"
	"net/http"
)

type Storage interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	Storage Storage
	logger  *slog.Logger
}

func New(storage Storage, logger *slog.Logger) *Handler {
	return &Handler{
		Storage: storage,
		logger:  logger,
	}
}

func (c *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	err := c.Storage.Ping(r.Context())
	if err != nil {
		c.logger.Error("expand service error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
