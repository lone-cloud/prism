package integration

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"prism/service/config"
	"prism/service/integration/proton"
	"prism/service/integration/signal"
	"prism/service/integration/telegram"
	"prism/service/integration/webpush"
	"prism/service/notification"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

type Integration interface {
	RegisterRoutes(router *chi.Mux, auth func(http.Handler) http.Handler, db *sql.DB, apiKey string, logger *slog.Logger)
	Start(ctx context.Context, logger *slog.Logger)
	IsEnabled() bool
}

type Integrations struct {
	Dispatcher   *notification.Dispatcher
	Signal       *signal.Integration
	Telegram     *telegram.Integration
	Proton       *proton.Integration
	integrations []Integration
}

func Initialize(cfg *config.Config, store *notification.Store, logger *slog.Logger, baseTmpl *template.Template) (*Integrations, *template.Template, error) {
	fragmentTmpl := baseTmpl
	var err error

	fragmentTmpl, err = fragmentTmpl.ParseFS(GetTemplates(), "templates/*.html")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse integration templates: %w", err)
	}
	fragmentTmpl, err = fragmentTmpl.ParseFS(signal.GetTemplates(), "templates/*.html")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse signal templates: %w", err)
	}
	fragmentTmpl, err = fragmentTmpl.ParseFS(telegram.GetTemplates(), "templates/*.html")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse telegram templates: %w", err)
	}
	fragmentTmpl, err = fragmentTmpl.ParseFS(proton.GetTemplates(), "templates/*.html")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse proton templates: %w", err)
	}

	tmplRenderer := util.NewTemplateRenderer(fragmentTmpl)

	var signalIntegration *signal.Integration
	var telegramIntegration *telegram.Integration
	dispatcher := notification.NewDispatcher(store, logger)

	var integrations []Integration

	var protonIntegration *proton.Integration

	if cfg.EnableSignal {
		signalIntegration = signal.NewIntegration(cfg, store, logger, tmplRenderer)
		if signalSender := signalIntegration.GetSender(); signalSender != nil {
			dispatcher.RegisterSender(notification.ChannelSignal, signalSender)
		}
		integrations = append(integrations, signalIntegration)
	}

	if cfg.EnableTelegram {
		telegramIntegration = telegram.NewIntegration(cfg, store, logger, tmplRenderer)
		if telegramSender := telegramIntegration.GetSender(); telegramSender != nil {
			dispatcher.RegisterSender(notification.ChannelTelegram, telegramSender)
		}
		integrations = append(integrations, telegramIntegration)
	}

	if cfg.EnableProton {
		protonIntegration = proton.NewIntegration(cfg, dispatcher, logger, tmplRenderer, store.GetDB(), cfg.APIKey)
		integrations = append(integrations, protonIntegration)
	}

	dispatcher.RegisterSender(notification.ChannelWebPush, webpush.NewSender(logger))
	integrations = append(integrations, webpush.NewIntegration(store, logger))

	return &Integrations{
		Dispatcher:   dispatcher,
		Signal:       signalIntegration,
		Telegram:     telegramIntegration,
		Proton:       protonIntegration,
		integrations: integrations,
	}, fragmentTmpl, nil
}

func (i *Integrations) Start(ctx context.Context, cfg *config.Config, logger *slog.Logger) {
	for _, integration := range i.integrations {
		if integration.IsEnabled() {
			integration.Start(ctx, logger)
		}
	}
}

func RegisterAll(integrations *Integrations, router *chi.Mux, cfg *config.Config, store *notification.Store, logger *slog.Logger, authMiddleware func(string) func(http.Handler) http.Handler) {
	auth := authMiddleware(cfg.APIKey)
	db := store.GetDB()
	for _, integration := range integrations.integrations {
		integration.RegisterRoutes(router, auth, db, cfg.APIKey, logger)
	}
}
