package telegram

import (
	"fmt"
	"log/slog"
	"strconv"

	"prism/service/delivery"
	"prism/service/subscription"
)

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

type Sender struct {
	client        *Client
	store         *subscription.Store
	logger        *slog.Logger
	DefaultChatID int64
}

func NewSender(client *Client, store *subscription.Store, logger *slog.Logger, defaultChatID int64) *Sender {
	return &Sender{
		client:        client,
		store:         store,
		logger:        logger,
		DefaultChatID: defaultChatID,
	}
}

func (s *Sender) GetChatID() int64 {
	return s.DefaultChatID
}

func (s *Sender) IsLinked() (bool, error) {
	if s.client == nil {
		return false, nil
	}

	if !s.client.IsAvailable() {
		return false, nil
	}

	if s.DefaultChatID == 0 {
		return false, nil
	}

	return true, nil
}

func (s *Sender) Send(sub *subscription.Subscription, notif delivery.Notification) error {
	if s.client == nil {
		return delivery.NewPermanentError(fmt.Errorf("telegram integration not enabled"))
	}

	if sub.Telegram == nil || sub.Telegram.ChatID == "" {
		return delivery.NewPermanentError(fmt.Errorf("no telegram chat configured for subscription"))
	}

	message := notif.Message
	if notif.Title != "" {
		message = fmt.Sprintf("<b>%s</b>\n%s", notif.Title, notif.Message)
	}

	fullMessage := fmt.Sprintf("<b>%s</b>\n\n%s", sub.AppName, message)

	chatID, err := parseInt64(sub.Telegram.ChatID)
	if err != nil {
		return delivery.NewPermanentError(fmt.Errorf("invalid chat ID: %w", err))
	}

	if err := s.client.SendMessage(chatID, fullMessage); err != nil {
		s.logger.Error("Failed to send telegram message", "chatID", chatID, "error", err)
		return err
	}

	return nil
}
