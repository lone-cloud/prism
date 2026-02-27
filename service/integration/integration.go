package integration

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"

	"prism/service/config"
	"prism/service/delivery"
	"prism/service/integration/proton"
	"prism/service/integration/signal"
	"prism/service/integration/telegram"
	"prism/service/integration/webpush"
	"prism/service/subscription"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

//go:embed templates/*.html
var templates embed.FS

type Integration interface {
	RegisterRoutes(router *chi.Mux, auth func(http.Handler) http.Handler, logger *slog.Logger)
	Start(ctx context.Context, logger *slog.Logger)
	IsEnabled() bool
	Health() (linked bool, account string)
}

type AutoSubscriber interface {
	AutoSubscribe(appName string, store *subscription.Store, publisher *delivery.Publisher) error
}

type Integrations struct {
	Publisher    *delivery.Publisher
	Signal       *signal.Integration
	Telegram     *telegram.Integration
	Proton       *proton.Integration
	store        *subscription.Store
	logger       *slog.Logger
	integrations []Integration
}

func Initialize(cfg *config.Config, store *subscription.Store, logger *slog.Logger, baseTmpl *template.Template) (*Integrations, *template.Template, error) {
	var err error
	baseTmpl, err = baseTmpl.ParseFS(templates, "templates/*.html")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse integration templates: %w", err)
	}

	var signalIntegration *signal.Integration
	var telegramIntegration *telegram.Integration
	var protonIntegration *proton.Integration

	integrations := []Integration{webpush.NewIntegration(store, logger)}

	if cfg.EnableSignal {
		tmplRenderer, err := newIntegrationRenderer(baseTmpl, "signal", signal.Templates)
		if err != nil {
			return nil, nil, err
		}
		signalIntegration = signal.NewIntegration(cfg, store, logger, tmplRenderer)
		integrations = append(integrations, signalIntegration)
	}

	if cfg.EnableTelegram {
		tmplRenderer, err := newIntegrationRenderer(baseTmpl, "telegram", telegram.Templates)
		if err != nil {
			return nil, nil, err
		}
		telegramIntegration = telegram.NewIntegration(store, logger, tmplRenderer, cfg.APIKey)
		integrations = append(integrations, telegramIntegration)
	}

	if cfg.EnableProton {
		tmplRenderer, err := newIntegrationRenderer(baseTmpl, "proton", proton.Templates)
		if err != nil {
			return nil, nil, err
		}
		protonIntegration = proton.NewIntegration(cfg, logger, tmplRenderer, store.DB, cfg.APIKey)
		integrations = append(integrations, protonIntegration)
	}

	i := &Integrations{
		Signal:       signalIntegration,
		Telegram:     telegramIntegration,
		Proton:       protonIntegration,
		store:        store,
		logger:       logger,
		integrations: integrations,
	}

	publisher := delivery.NewPublisher(store, logger, i.autoSubscribeApp)
	publisher.RegisterSender(subscription.ChannelWebPush, webpush.NewSender(logger))
	if signalIntegration != nil && signalIntegration.Sender != nil {
		if linked, err := signalIntegration.Sender.IsLinked(); err == nil && linked {
			publisher.RegisterSender(subscription.ChannelSignal, signalIntegration.Sender)
		}
	}
	if telegramIntegration != nil && telegramIntegration.Sender != nil {
		publisher.RegisterSender(subscription.ChannelTelegram, telegramIntegration.Sender)
	}
	i.Publisher = publisher

	if telegramIntegration != nil {
		telegramIntegration.OnUnlink = func() {
			i.Publisher.DeregisterSender(subscription.ChannelTelegram)
			if err := i.store.DeleteSubscriptionsByChannel(subscription.ChannelTelegram); err != nil {
				i.logger.Error("Failed to delete Telegram subscriptions on unlink", "error", err)
			}
		}
	}

	if protonIntegration != nil {
		protonIntegration.Publisher = publisher
	}

	return i, baseTmpl, nil
}

func newIntegrationRenderer(base *template.Template, name string, templates fs.FS) (*util.TemplateRenderer, error) {
	integrationTmpl, err := base.Clone()
	if err != nil {
		return nil, fmt.Errorf("failed to clone base templates for %s: %w", name, err)
	}
	integrationTmpl, err = integrationTmpl.ParseFS(templates, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s templates: %w", name, err)
	}

	return util.NewTemplateRenderer(integrationTmpl), nil
}

func (i *Integrations) Start(ctx context.Context, logger *slog.Logger) {
	for _, integration := range i.integrations {
		if integration.IsEnabled() {
			integration.Start(ctx, logger)
		}
	}
}

func RegisterAll(integrations *Integrations, router *chi.Mux, cfg *config.Config, logger *slog.Logger, authMiddleware func(string) func(http.Handler) http.Handler) {
	auth := authMiddleware(cfg.APIKey)
	for _, integration := range integrations.integrations {
		integration.RegisterRoutes(router, auth, logger)
	}
}

func (i *Integrations) autoSubscribeApp(appName string) error {
	app, err := i.store.GetApp(appName)
	if err != nil {
		return err
	}
	if app != nil {
		return nil
	}

	if err := i.store.RegisterApp(appName); err != nil {
		return fmt.Errorf("failed to register app %q: %w", appName, err)
	}
	i.logger.Info("Registering new app and auto-configuring subscription", "app", appName)

	for _, integration := range i.integrations {
		sub, ok := integration.(AutoSubscriber)
		if !ok {
			continue
		}
		if err := sub.AutoSubscribe(appName, i.store, i.Publisher); err == nil {
			return nil
		} else {
			i.logger.Warn("Auto-config failed", "integration", fmt.Sprintf("%T", integration), "app", appName, "error", err)
		}
	}

	return fmt.Errorf("no integration available to auto-configure app %q", appName)
}

func (i *Integrations) IsSignalLinked() bool {
	if i.Signal == nil {
		return false
	}
	linked, _ := i.Signal.Health()
	return linked
}

func (i *Integrations) IsTelegramLinked() bool {
	if i.Telegram == nil {
		return false
	}
	linked, _ := i.Telegram.Health()
	return linked
}
