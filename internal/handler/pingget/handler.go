package pingget

import (
	"log/slog"
	"net/http"

	"github.com/ievseev/url-shortener/internal/storage/db"
)

type Handler struct {
	postgres *db.Postgres
	logger   *slog.Logger
}

func New(postgres *db.Postgres, logger *slog.Logger) *Handler {
	return &Handler{
		postgres: postgres,
		logger:   logger,
	}
}

func (c *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	err := c.postgres.Ping(r.Context())
	if err != nil {
		c.logger.Error("ping service error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
