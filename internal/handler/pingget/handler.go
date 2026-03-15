package pingget

import (
	"context"
	"log/slog"
	"net/http"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	pinger Pinger
	logger *slog.Logger
}

func New(pinger Pinger, logger *slog.Logger) *Handler {
	return &Handler{
		pinger: pinger,
		logger: logger,
	}
}

func (c *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	err := c.pinger.Ping(r.Context())
	if err != nil {
		c.logger.Error("ping service error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
