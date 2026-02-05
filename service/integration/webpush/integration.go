package webpush

import (
	"context"
	"log/slog"
	"net/http"

	"prism/service/notification"

	"github.com/go-chi/chi/v5"
)

type Integration struct {
	store  *notification.Store
	logger *slog.Logger
}

func NewIntegration(store *notification.Store, logger *slog.Logger) *Integration {
	return &Integration{
		store:  store,
		logger: logger,
	}
}

func (w *Integration) RegisterRoutes(router *chi.Mux, auth func(http.Handler) http.Handler) {
	RegisterRoutes(router, w.store, w.logger, auth)
}

func (w *Integration) Start(ctx context.Context, logger *slog.Logger) {}

func (w *Integration) IsEnabled() bool {
	return true
}
