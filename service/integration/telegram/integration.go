package telegram

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"prism/service/credentials"
	"prism/service/delivery"
	"prism/service/subscription"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

type Integration struct {
	Handlers *Handlers
	Sender   *Sender
	OnUnlink func()
	db       *sql.DB
	apiKey   string
	logger   *slog.Logger
}

func NewIntegration(store *subscription.Store, logger *slog.Logger, tmpl *util.TemplateRenderer, apiKey string) *Integration {
	var client *Client
	var chatID int64
	var sender *Sender

	credStore, err := credentials.NewStoreWithLogger(store.DB, apiKey, logger)
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

	if client != nil && chatID != 0 {
		sender = NewSender(client, store, logger, chatID)
	}

	handlers := NewHandlers(client, chatID, tmpl, logger)

	return &Integration{
		Handlers: handlers,
		Sender:   sender,
		db:       store.DB,
		apiKey:   apiKey,
		logger:   logger,
	}
}

func (t *Integration) RegisterRoutes(router *chi.Mux, auth func(http.Handler) http.Handler, logger *slog.Logger) {
	RegisterRoutes(router, t.Handlers, auth, t.db, t.apiKey, logger, t.OnUnlink)
}

func (t *Integration) Start(ctx context.Context, logger *slog.Logger) {
	client := t.Handlers.GetClient()
	if client == nil {
		logger.Info("Telegram not configured", "action", "visit admin UI to configure")
		return
	}

	bot, err := client.GetMe()
	if err != nil {
		logger.Error("Telegram bot error", "error", err)
		return
	}

	chatID := t.Handlers.chatID
	if chatID == 0 {
		logger.Info("Telegram bot connected", "bot", bot.Username, "status", "needs_chat_id")
	} else {
		logger.Info("Telegram enabled", "bot", bot.Username, "chat_id", chatID)
	}
}

func (t *Integration) IsEnabled() bool {
	return t.Handlers != nil && t.Handlers.GetClient() != nil
}

func (t *Integration) Health() (bool, string) {
	if t.Handlers == nil {
		return false, ""
	}
	client := t.Handlers.GetClient()
	if client == nil {
		return false, ""
	}
	bot, err := client.GetMe()
	if err != nil {
		return false, ""
	}
	return true, "@" + bot.Username
}

func (t *Integration) AutoSubscribe(appName string, store *subscription.Store, publisher *delivery.Publisher) error {
	credStore, err := credentials.NewStore(t.db, t.apiKey)
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}
	creds, err := credStore.GetTelegram()
	if err != nil || creds == nil {
		return fmt.Errorf("telegram not configured")
	}
	if creds.ChatID == "" {
		return fmt.Errorf("no chat ID configured")
	}
	chatID, err := strconv.ParseInt(creds.ChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	if !publisher.HasChannel(subscription.ChannelTelegram) {
		client, err := NewClient(creds.BotToken)
		if err != nil {
			return fmt.Errorf("failed to create telegram client: %w", err)
		}
		sender := NewSender(client, store, t.logger, chatID)
		t.Sender = sender
		publisher.RegisterSender(subscription.ChannelTelegram, sender)
	}

	subID, err := store.AddSubscription(subscription.Subscription{
		AppName:  appName,
		Channel:  subscription.ChannelTelegram,
		Telegram: &subscription.TelegramSubscription{ChatID: fmt.Sprintf("%d", chatID)},
	})
	if err != nil {
		return err
	}

	t.logger.Info("Auto-configured Telegram subscription", "app", appName, "subscriptionID", subID, "chatID", chatID)
	return nil
}
