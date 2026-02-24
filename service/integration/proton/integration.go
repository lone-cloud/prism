package proton

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"

	"prism/service/config"
	"prism/service/credentials"
	"prism/service/notification"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

type Integration struct {
	cfg        *config.Config
	dispatcher *notification.Dispatcher
	handlers   *Handlers
	monitor    *Monitor
	db         *sql.DB
	apiKey     string
}

func NewIntegration(cfg *config.Config, dispatcher *notification.Dispatcher, logger *slog.Logger, tmpl *util.TemplateRenderer, db *sql.DB, apiKey string) *Integration {
	monitor := NewMonitor(cfg, dispatcher, logger)
	handlers := NewHandlers(monitor, "", logger, tmpl)
	return &Integration{
		cfg:        cfg,
		dispatcher: dispatcher,
		handlers:   handlers,
		monitor:    monitor,
		db:         db,
		apiKey:     apiKey,
	}
}

func (p *Integration) RegisterRoutes(router *chi.Mux, auth func(http.Handler) http.Handler, db *sql.DB, apiKey string, logger *slog.Logger) {
	credStore, err := credentials.NewStore(db, apiKey)
	if err == nil {
		if creds, err := credStore.GetProton(); err == nil {
			p.handlers.username = creds.Email
		}
	}

	RegisterRoutes(router, p.handlers, auth, db, apiKey, logger, p)
}

func (p *Integration) Start(ctx context.Context, logger *slog.Logger) {
	credStore, err := credentials.NewStore(p.db, p.apiKey)
	if err != nil {
		logger.Error("Failed to initialize credentials store for Proton", "error", err)
		return
	}

	creds, err := credStore.GetProton()
	if err != nil {
		logger.Debug("Proton credentials not configured", "error", err)
		return
	}

	if err := p.monitor.Start(ctx, credStore); err != nil {
		logger.Error("Failed to start Proton monitor", "error", err)
		return
	}

	if p.handlers != nil {
		p.handlers.username = creds.Email
	}

	logger.Info("Proton Mail enabled", "email", creds.Email)
}

func (p *Integration) IsEnabled() bool {
	return p.cfg.EnableProton
}

func (p *Integration) GetHandlers() *Handlers {
	return p.handlers
}
