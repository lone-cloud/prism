package integration

import (
	"context"
	"log/slog"
	"net/http"

	"prism/service/config"
	"prism/service/integration/proton"
	"prism/service/integration/signal"
	"prism/service/integration/telegram"
	"prism/service/integration/webpush"
	"prism/service/notification"

	"github.com/go-chi/chi/v5"
)

type Integration interface {
	RegisterRoutes(router *chi.Mux, auth func(http.Handler) http.Handler)
	Start(ctx context.Context, logger *slog.Logger)
	IsEnabled() bool
}

type Integrations struct {
	Dispatcher   *notification.Dispatcher
	Signal       *signal.Integration
	integrations []Integration
}

func Initialize(cfg *config.Config, store *notification.Store, logger *slog.Logger) *Integrations {
	signalIntegration := signal.NewIntegration(cfg, store, logger)
	telegramIntegration := telegram.NewIntegration(cfg, store, logger)
	dispatcher := notification.NewDispatcher(store, logger)

	if signalSender := signalIntegration.GetSender(); signalSender != nil {
		dispatcher.RegisterSender(notification.ChannelSignal, signalSender)
	}
	if telegramSender := telegramIntegration.GetSender(); telegramSender != nil {
		dispatcher.RegisterSender(notification.ChannelTelegram, telegramSender)
	}
	dispatcher.RegisterSender(notification.ChannelWebPush, webpush.NewSender(logger))

	integrations := []Integration{
		signalIntegration,
		telegramIntegration,
		proton.NewIntegration(cfg, dispatcher, logger),
		webpush.NewIntegration(store, logger),
	}

	return &Integrations{
		Dispatcher:   dispatcher,
		Signal:       signalIntegration,
		integrations: integrations,
	}
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
	for _, integration := range integrations.integrations {
		integration.RegisterRoutes(router, auth)
	}
}
