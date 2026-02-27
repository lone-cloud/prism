package signal

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"prism/service/config"
	"prism/service/delivery"
	"prism/service/subscription"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

type Integration struct {
	cfg      *config.Config
	client   *Client
	Handlers *Handlers
	Sender   *Sender
	Groups   *GroupCache
	store    *subscription.Store
	logger   *slog.Logger
	tmpl     *util.TemplateRenderer
}

func NewIntegration(cfg *config.Config, store *subscription.Store, logger *slog.Logger, tmpl *util.TemplateRenderer) *Integration {
	client := NewClient()
	groups, err := NewGroupCache(store.DB)
	if err != nil {
		logger.Error("Failed to initialize Signal group cache", "error", err)
	}
	var sender *Sender
	if client != nil {
		sender = NewSender(client, groups, logger)
	}
	return &Integration{
		cfg:    cfg,
		client: client,
		Sender: sender,
		Groups: groups,
		store:  store,
		logger: logger,
		tmpl:   tmpl,
	}
}

func (s *Integration) RegisterRoutes(router *chi.Mux, auth func(http.Handler) http.Handler, logger *slog.Logger) {
	s.Handlers = RegisterRoutes(router, s.cfg, auth, s.tmpl, logger, s.client)
}

func (s *Integration) Start(ctx context.Context, logger *slog.Logger) {
	if s.Handlers != nil && s.Handlers.Client != nil {
		client := s.Handlers.Client
		account, _ := client.GetLinkedAccount()
		if account != nil {
			logger.Debug("Signal enabled", "status", "linked", "number", FormatPhoneNumber(account.Number))
		} else {
			logger.Debug("Signal enabled", "status", "unlinked", "action", "visit admin UI to link")
		}
	} else {
		logger.Debug("Signal disabled", "reason", "signal-cli not found in PATH")
	}
}

func (s *Integration) IsEnabled() bool {
	return s.Handlers != nil && s.Handlers.Client != nil
}

func (s *Integration) Health() (bool, string) {
	if s.Handlers == nil || s.Handlers.Client == nil {
		return false, ""
	}
	account, _ := s.Handlers.Client.GetLinkedAccount()
	if account == nil {
		return false, ""
	}
	return true, account.Number
}

func (s *Integration) AutoSubscribe(appName string, store *subscription.Store, publisher *delivery.Publisher) error {
	if s.Sender == nil {
		return fmt.Errorf("signal-cli not available")
	}
	linked, err := s.Sender.IsLinked()
	if err != nil {
		return fmt.Errorf("failed to check Signal link status: %w", err)
	}
	if !linked {
		return fmt.Errorf("no linked Signal account")
	}
	if !publisher.HasChannel(subscription.ChannelSignal) {
		publisher.RegisterSender(subscription.ChannelSignal, s.Sender)
	}

	var signalSub *subscription.SignalSubscription
	if cachedGroup, err := s.Groups.Get(appName); err != nil {
		s.logger.Warn("Failed to check for cached Signal group", "error", err)
	} else if cachedGroup != nil {
		s.logger.Debug("Reusing cached Signal group", "app", appName)
		signalSub = cachedGroup
	}

	if signalSub == nil {
		if signalSub, err = s.Sender.CreateDefaultSignalSubscription(appName); err != nil {
			return err
		}
	}

	subID, err := store.AddSubscription(subscription.Subscription{
		AppName: appName,
		Channel: subscription.ChannelSignal,
		Signal:  signalSub,
	})
	if err != nil {
		return err
	}

	s.logger.Info("Auto-configured Signal subscription", "app", appName, "subscriptionID", subID)
	return nil
}
