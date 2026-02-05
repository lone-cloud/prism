package telegram

import (
	"context"
	"log/slog"
	"net/http"

	"prism/service/config"
	"prism/service/notification"

	"github.com/go-chi/chi/v5"
)

type Integration struct {
	cfg      *config.Config
	handlers *Handlers
	sender   *Sender
}

func NewIntegration(cfg *config.Config, store *notification.Store, logger *slog.Logger) *Integration {
	client, err := NewClient(cfg.TelegramBotToken)
	if err != nil {
		logger.Error("Failed to create telegram client", "error", err)
	}

	var sender *Sender
	if client != nil {
		sender = NewSender(client, store, logger, cfg.TelegramChatID)
	}

	handlers := NewHandlers(client, cfg.TelegramChatID)

	return &Integration{
		cfg:      cfg,
		handlers: handlers,
		sender:   sender,
	}
}

func (t *Integration) GetSender() *Sender {
	return t.sender
}

func (t *Integration) RegisterRoutes(router *chi.Mux, auth func(http.Handler) http.Handler) {
	RegisterRoutes(router, t.handlers, auth)
}

func (t *Integration) Start(ctx context.Context, logger *slog.Logger) {
	if t.handlers != nil && t.handlers.IsEnabled() {
		client := t.handlers.GetClient()
		bot, err := client.GetMe()
		if err != nil {
			logger.Error("Telegram bot error", "error", err)
		} else {
			logger.Info("Telegram enabled", "bot", bot.Username, "id", bot.ID)
		}
	}
}

func (t *Integration) IsEnabled() bool {
	return t.cfg.IsTelegramEnabled()
}
