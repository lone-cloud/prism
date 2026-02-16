package webpush

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"prism/service/notification"

	webpush "github.com/SherClockHolmes/webpush-go"
)

type Sender struct {
	logger *slog.Logger
}

func NewSender(logger *slog.Logger) *Sender {
	return &Sender{
		logger: logger,
	}
}

func (s *Sender) Send(sub *notification.Subscription, notif notification.Notification) error {
	if sub.WebPush == nil {
		return notification.NewPermanentError(fmt.Errorf("no push endpoint configured for subscription %s", sub.ID))
	}

	payload, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	if sub.WebPush.HasEncryption() {
		subscription := &webpush.Subscription{
			Endpoint: sub.WebPush.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.WebPush.P256dh,
				Auth:   sub.WebPush.Auth,
			},
		}

		resp, err := webpush.SendNotification(payload, subscription, &webpush.Options{
			VAPIDPrivateKey: sub.WebPush.VapidPrivateKey,
			TTL:             86400,
		})
		if err != nil {
			return fmt.Errorf("failed to send webpush: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("webpush returned status %d", resp.StatusCode)
		}

		s.logger.Debug("Sent encrypted webpush notification", "app", sub.AppName, "url", sub.WebPush.Endpoint)
	} else {
		resp, err := http.Post(sub.WebPush.Endpoint, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			return fmt.Errorf("failed to send webhook: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("webhook returned status %d", resp.StatusCode)
		}

		s.logger.Debug("Sent plain webhook notification", "app", sub.AppName, "url", sub.WebPush.Endpoint)
	}

	return nil
}
