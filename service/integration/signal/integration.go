package signal

import (
	"context"
	"log/slog"
	"net/http"

	"prism/service/config"
	"prism/service/notification"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

type Integration struct {
	cfg      *config.Config
	handlers *Handlers
	sender   *Sender
	tmpl     *util.TemplateRenderer
	logger   *slog.Logger
}

func NewIntegration(cfg *config.Config, store *notification.Store, logger *slog.Logger, tmpl *util.TemplateRenderer) *Integration {
	var sender *Sender
	if cfg.IsSignalEnabled() {
		client := NewClient(cfg.SignalSocket)
		sender = NewSender(client, store, logger)
	}
	return &Integration{
		cfg:    cfg,
		sender: sender,
		tmpl:   tmpl,
		logger: logger,
	}
}

func (s *Integration) GetSender() *Sender {
	return s.sender
}

func (s *Integration) RegisterRoutes(router *chi.Mux, auth func(http.Handler) http.Handler) {
	s.handlers = RegisterRoutes(router, s.cfg, auth, s.tmpl, s.logger)
}

func (s *Integration) Start(ctx context.Context, logger *slog.Logger) {
	if s.handlers != nil && s.handlers.IsEnabled() {
		client := s.handlers.GetClient()
		account, _ := client.GetLinkedAccount()
		if account != nil {
			logger.Info("Signal enabled", "status", "linked", "number", FormatPhoneNumber(account.Number))
		} else {
			logger.Info("Signal enabled", "status", "unlinked", "action", "visit admin UI to link")
		}
	}
}

func (s *Integration) IsEnabled() bool {
	return s.cfg.IsSignalEnabled()
}

func (s *Integration) GetHandlers() *Handlers {
	return s.handlers
}
