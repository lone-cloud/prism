package telegram

import (
	"fmt"
	"log/slog"

	"prism/service/notification"
)

type Sender struct {
	client        *Client
	store         *notification.Store
	logger        *slog.Logger
	DefaultChatID int64
}

func NewSender(client *Client, store *notification.Store, logger *slog.Logger, defaultChatID int64) *Sender {
	return &Sender{
		client:        client,
		store:         store,
		logger:        logger,
		DefaultChatID: defaultChatID,
	}
}

func (s *Sender) Send(mapping *notification.Mapping, notif notification.Notification) error {
	if s.client == nil {
		return notification.NewPermanentError(fmt.Errorf("telegram integration not enabled"))
	}

	if s.DefaultChatID == 0 {
		return notification.NewPermanentError(fmt.Errorf("no telegram chat configured (set TELEGRAM_CHAT_ID in .env)"))
	}

	message := notif.Message
	if notif.Title != "" {
		message = fmt.Sprintf("<b>%s</b>\n%s", notif.Title, notif.Message)
	}

	fullMessage := fmt.Sprintf("<b>%s</b>\n\n%s", mapping.AppName, message)

	if err := s.client.SendMessage(s.DefaultChatID, fullMessage); err != nil {
		s.logger.Error("Failed to send telegram message", "chatID", s.DefaultChatID, "error", err)
		return err
	}

	return nil
}
