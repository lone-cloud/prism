package delivery

import (
	"log/slog"
	"time"

	"prism/service/subscription"
	"prism/service/util"
)

type NotificationSender interface {
	Send(sub *subscription.Subscription, notif Notification) error
}

type Publisher struct {
	Store           *subscription.Store
	senders         map[subscription.Channel]NotificationSender
	autoSubscribeFn func(string) error
	logger          *slog.Logger
}

func NewPublisher(store *subscription.Store, logger *slog.Logger, autoSubscribeFn func(string) error) *Publisher {
	return &Publisher{
		Store:           store,
		senders:         make(map[subscription.Channel]NotificationSender),
		autoSubscribeFn: autoSubscribeFn,
		logger:          logger,
	}
}

func (p *Publisher) RegisterSender(channel subscription.Channel, sender NotificationSender) {
	p.senders[channel] = sender
}

func (p *Publisher) DeregisterSender(channel subscription.Channel) {
	delete(p.senders, channel)
}

func (p *Publisher) HasChannel(channel subscription.Channel) bool {
	_, ok := p.senders[channel]
	return ok
}

func (p *Publisher) IsValidChannel(channel subscription.Channel) bool {
	return p.HasChannel(channel)
}

func (p *Publisher) Publish(appName string, notif Notification) error {
	app, err := p.Store.GetApp(appName)
	if err != nil {
		return util.LogError(p.logger, "Failed to get app", err, "app", appName)
	}

	if app == nil || len(app.Subscriptions) == 0 {
		if p.autoSubscribeFn != nil {
			if err := p.autoSubscribeFn(appName); err != nil {
				p.logger.Warn("Failed to auto-configure subscription", "app", appName, "error", err)
			}
			app, err = p.Store.GetApp(appName)
			if err != nil {
				return util.LogError(p.logger, "Failed to get app", err, "app", appName)
			}
		}
	}

	if app == nil || len(app.Subscriptions) == 0 {
		p.logger.Warn("No subscriptions found for app, dropping notification", "app", appName)
		return nil
	}

	var lastErr error
	successCount := 0

	for _, sub := range app.Subscriptions {
		sender, ok := p.senders[sub.Channel]
		if !ok {
			p.logger.Debug("Skipping subscription for disabled channel", "channel", sub.Channel, "subscriptionID", sub.ID)
			continue
		}

		if err := p.sendWithRetry(sender, &sub, notif, appName, sub.ID); err != nil {
			lastErr = err
		} else {
			successCount++
		}
	}

	if successCount == 0 && lastErr != nil {
		return lastErr
	}

	return nil
}

func (p *Publisher) sendWithRetry(sender NotificationSender, sub *subscription.Subscription, notif Notification, appName, subscriptionID string) error {
	maxRetries := 10
	baseDelay := 500 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := sender.Send(sub, notif)
		if err == nil {
			if attempt > 0 {
				p.logger.Info("Notification sent after retry", "app", appName, "subscriptionID", subscriptionID, "attempt", attempt+1)
			}
			return nil
		}

		lastErr = err

		if IsPermanent(err) {
			p.logger.Error("Permanent error, not retrying", "app", appName, "subscriptionID", subscriptionID, "error", err)
			return err
		}

		if attempt < maxRetries-1 {
			delay := baseDelay * time.Duration(1<<uint(attempt))
			p.logger.Warn("Failed to send notification, retrying", "app", appName, "subscriptionID", subscriptionID, "attempt", attempt+1, "error", err, "retryIn", delay)
			time.Sleep(delay)
		}
	}

	p.logger.Error("Failed to send notification after retries", "app", appName, "subscriptionID", subscriptionID, "attempts", maxRetries, "error", lastErr)
	return lastErr
}
