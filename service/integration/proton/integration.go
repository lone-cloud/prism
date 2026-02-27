package proton

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"

	"prism/service/config"
	"prism/service/credentials"
	"prism/service/delivery"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

type Integration struct {
	cfg       *config.Config
	Publisher *delivery.Publisher
	Handlers  *Handlers
	monitor   *Monitor
	db        *sql.DB
	apiKey    string
}

func NewIntegration(cfg *config.Config, logger *slog.Logger, tmpl *util.TemplateRenderer, db *sql.DB, apiKey string) *Integration {
	monitor := NewMonitor(cfg, logger)
	handlers := NewHandlers(monitor, "", logger, tmpl)
	return &Integration{
		cfg:      cfg,
		Handlers: handlers,
		monitor:  monitor,
		db:       db,
		apiKey:   apiKey,
	}
}

func (p *Integration) RegisterRoutes(router *chi.Mux, auth func(http.Handler) http.Handler, logger *slog.Logger) {
	credStore, err := credentials.NewStore(p.db, p.apiKey)
	if err == nil {
		if creds, err := credStore.GetProton(); err == nil {
			p.Handlers.username = creds.Email
		}
	}

	RegisterRoutes(router, p.Handlers, auth, p.db, p.apiKey, logger, p)
}

func (p *Integration) Start(ctx context.Context, logger *slog.Logger) {
	credStore, err := credentials.NewStore(p.db, p.apiKey)
	if err != nil {
		logger.Error("Failed to initialize credentials store for Proton", "error", err)
		return
	}

	creds, err := credStore.GetProton()
	if err != nil {
		logger.Info("Proton credentials not configured", "error", err)
		return
	}

	if err := p.monitor.Start(ctx, credStore, p.Publisher); err != nil {
		logger.Error("Failed to start Proton monitor", "error", err)
		return
	}

	if p.Handlers != nil {
		p.Handlers.username = creds.Email
	}

	logger.Info("Proton Mail enabled", "email", creds.Email)
}

func (p *Integration) IsEnabled() bool {
	return p.cfg.EnableProton
}

func (p *Integration) Health() (bool, string) {
	if p.Handlers == nil {
		return false, ""
	}
	email, ok := p.Handlers.LoadFreshCredentials()
	return ok, email
}
