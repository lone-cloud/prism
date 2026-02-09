package telegram

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"

	"prism/service/config"
	"prism/service/credentials"
	"prism/service/notification"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

type Integration struct {
	cfg      *config.Config
	handlers *Handlers
	sender   *Sender
	logger   *slog.Logger
}

func NewIntegration(cfg *config.Config, store *notification.Store, logger *slog.Logger, tmpl *util.TemplateRenderer) *Integration {
	var client *Client
	var chatID int64
	var sender *Sender

	credStore, err := credentials.NewStoreWithLogger(store.GetDB(), cfg.APIKey, logger)
	if err != nil {
		logger.Warn("Failed to initialize credentials store for Telegram", "error", err)
	} else {
		creds, err := credStore.GetTelegram()
		if err == nil && creds != nil {
			logger.Info("Loading Telegram credentials from database")
			client, err = NewClient(creds.BotToken)
			if err != nil {
				logger.Error("Failed to create Telegram client", "error", err)
				client = nil
			}
			if creds.ChatID != "" {
				chatID, _ = strconv.ParseInt(creds.ChatID, 10, 64)
			}
		}
	}

	if client != nil {
		sender = NewSender(client, store, logger, chatID)
	}

	handlers := NewHandlers(client, chatID, tmpl, logger)

	return &Integration{
		cfg:      cfg,
		handlers: handlers,
		sender:   sender,
		logger:   logger,
	}
}

func (t *Integration) GetSender() *Sender {
	return t.sender
}

func (t *Integration) GetHandlers() *Handlers {
	return t.handlers
}

func (t *Integration) RegisterRoutes(router *chi.Mux, auth func(http.Handler) http.Handler, db *sql.DB, apiKey string, logger *slog.Logger) {
	RegisterRoutes(router, t.handlers, auth, db, apiKey, logger)
}

func (t *Integration) Start(ctx context.Context, logger *slog.Logger) {
	client := t.handlers.GetClient()
	if client == nil {
		logger.Info("Telegram not configured", "action", "visit admin UI to configure")
		return
	}

	bot, err := client.GetMe()
	if err != nil {
		logger.Error("Telegram bot error", "error", err)
		return
	}

	chatID := t.handlers.chatID
	if chatID == 0 {
		logger.Info("Telegram bot connected", "bot", bot.Username, "status", "needs_chat_id")
	} else {
		logger.Info("Telegram enabled", "bot", bot.Username, "chat_id", chatID)
	}
}

func (t *Integration) IsEnabled() bool {
	return t.handlers != nil && t.handlers.GetClient() != nil
}
