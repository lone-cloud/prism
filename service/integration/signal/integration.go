package signal

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"

	"prism/service/config"
	"prism/service/notification"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

type Integration struct {
	cfg      *config.Config
	client   *Client
	handlers *Handlers
	sender   *Sender
	tmpl     *util.TemplateRenderer
}

func NewIntegration(cfg *config.Config, store *notification.Store, logger *slog.Logger, tmpl *util.TemplateRenderer) *Integration {
	client := NewClient()
	var sender *Sender
	if client.IsEnabled() {
		sender = NewSender(client, store, logger)
	}
	return &Integration{
		cfg:    cfg,
		client: client,
		sender: sender,
		tmpl:   tmpl,
	}
}

func (s *Integration) GetSender() *Sender {
	return s.sender
}

func (s *Integration) RegisterRoutes(router *chi.Mux, auth func(http.Handler) http.Handler, db *sql.DB, apiKey string, logger *slog.Logger) {
	s.handlers = RegisterRoutes(router, s.cfg, auth, s.tmpl, logger, s.client)
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
	} else {
		logger.Info("Signal disabled", "reason", "signal-cli not found in PATH")
	}
}

func (s *Integration) IsEnabled() bool {
	if s.handlers == nil {
		return false
	}
	return s.handlers.IsEnabled()
}

func (s *Integration) GetHandlers() *Handlers {
	return s.handlers
}
