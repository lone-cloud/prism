package proton

import (
	"context"
	"log/slog"
	"net/http"

	"prism/service/config"
	"prism/service/notification"

	"github.com/go-chi/chi/v5"
)

type Integration struct {
	cfg        *config.Config
	dispatcher *notification.Dispatcher
	logger     *slog.Logger
	handlers   *Handlers
}

func NewIntegration(cfg *config.Config, dispatcher *notification.Dispatcher, logger *slog.Logger) *Integration {
	return &Integration{
		cfg:        cfg,
		dispatcher: dispatcher,
		logger:     logger,
	}
}

func (p *Integration) RegisterRoutes(router *chi.Mux, auth func(http.Handler) http.Handler) {
	p.handlers = RegisterRoutes(router, p.cfg, p.dispatcher, p.logger, auth)
}

func (p *Integration) Start(ctx context.Context, logger *slog.Logger) {
	if p.handlers != nil && p.handlers.IsEnabled() {
		logger.Info("Proton Mail configured", "status", "connecting in background", "bridge", p.cfg.ProtonBridgeAddr)
		go func() {
			monitor := p.handlers.GetMonitor()
			if err := monitor.Start(ctx); err != nil && ctx.Err() == nil {
				logger.Error("Proton monitor error", "error", err)
			}
		}()
	}
}

func (p *Integration) IsEnabled() bool {
	return p.cfg.IsProtonEnabled()
}

func (p *Integration) GetHandlers() *Handlers {
	return p.handlers
}
