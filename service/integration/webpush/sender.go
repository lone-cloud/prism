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

func (s *Sender) Send(mapping *notification.Mapping, notif notification.Notification) error {
	if mapping.WebPush == nil {
		return notification.NewPermanentError(fmt.Errorf("no push endpoint configured for %s", mapping.AppName))
	}

	payload, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	if mapping.WebPush.HasEncryption() {
		subscription := &webpush.Subscription{
			Endpoint: mapping.WebPush.Endpoint,
			Keys: webpush.Keys{
				P256dh: mapping.WebPush.P256dh,
				Auth:   mapping.WebPush.Auth,
			},
		}

		resp, err := webpush.SendNotification(payload, subscription, &webpush.Options{
			VAPIDPrivateKey: mapping.WebPush.VapidPrivateKey,
			TTL:             86400,
		})
		if err != nil {
			return fmt.Errorf("failed to send webpush: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("webpush returned status %d", resp.StatusCode)
		}

		s.logger.Debug("Sent encrypted webpush notification", "app", mapping.AppName, "url", mapping.WebPush.Endpoint)
	} else {
		resp, err := http.Post(mapping.WebPush.Endpoint, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			return fmt.Errorf("failed to send webhook: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("webhook returned status %d", resp.StatusCode)
		}

		s.logger.Debug("Sent plain webhook notification", "app", mapping.AppName, "url", mapping.WebPush.Endpoint)
	}

	return nil
}
