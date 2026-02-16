package notification

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"prism/service/util"
)

type NotificationSender interface {
	Send(sub *Subscription, notif Notification) error
}

type linkedChecker interface {
	IsLinked() (bool, error)
}

type signalAutoConfigurer interface {
	CreateDefaultSignalSubscription(appName string) (*SignalSubscription, error)
}

type telegramSender interface {
	GetChatID() int64
}

type Dispatcher struct {
	store   *Store
	senders map[Channel]NotificationSender
	logger  *slog.Logger
}

func NewDispatcher(store *Store, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{
		store:   store,
		senders: make(map[Channel]NotificationSender),
		logger:  logger,
	}
}

func (d *Dispatcher) RegisterSender(channel Channel, sender NotificationSender) {
	d.senders[channel] = sender
}

func (d *Dispatcher) GetStore() *Store {
	return d.store
}

func (d *Dispatcher) HasSignal() bool {
	_, ok := d.senders[ChannelSignal]
	return ok
}

func (d *Dispatcher) HasTelegram() bool {
	_, ok := d.senders[ChannelTelegram]
	return ok
}

func (d *Dispatcher) GetAvailableChannels() []Channel {
	var channels []Channel
	if d.HasSignal() {
		channels = append(channels, ChannelSignal)
	}
	if d.HasTelegram() {
		channels = append(channels, ChannelTelegram)
	}
	channels = append(channels, ChannelWebPush)
	return channels
}

func (d *Dispatcher) IsValidChannel(channel Channel) bool {
	return channel.IsAvailable(d.HasSignal(), d.HasTelegram())
}

func (d *Dispatcher) Send(appName string, notif Notification) error {
	app, err := d.store.GetApp(appName)
	if err != nil {
		return util.LogError(d.logger, "Failed to get app", err, "app", appName)
	}

	if app == nil {
		d.logger.Info("Registering new app and auto-configuring subscription", "app", appName)
		app, err = d.CreateAppWithDefaultSubscription(appName)
		if err != nil {
			d.logger.Warn("Failed to auto-configure subscription", "app", appName, "error", err)
			return nil
		}
	}

	if len(app.Subscriptions) == 0 {
		d.logger.Info("App has no subscriptions, attempting to auto-configure", "app", appName)
		app, err = d.CreateAppWithDefaultSubscription(appName)
		if err != nil {
			d.logger.Warn("Failed to auto-configure subscription", "app", appName, "error", err)
			return nil
		}
	}

	var lastErr error
	successCount := 0

	for _, sub := range app.Subscriptions {
		sender, ok := d.senders[sub.Channel]
		if !ok {
			d.logger.Debug("Skipping subscription for disabled channel", "channel", sub.Channel, "subscriptionID", sub.ID)
			continue
		}

		if err := d.sendWithRetry(sender, &sub, notif, appName, sub.ID); err != nil {
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

func (d *Dispatcher) sendWithRetry(sender NotificationSender, sub *Subscription, notif Notification, appName, subscriptionID string) error {
	maxRetries := 10
	baseDelay := 500 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := sender.Send(sub, notif)
		if err == nil {
			if attempt > 0 {
				d.logger.Info("Notification sent after retry", "app", appName, "subscriptionID", subscriptionID, "attempt", attempt+1)
			}
			return nil
		}

		lastErr = err

		if IsPermanent(err) {
			d.logger.Error("Permanent error, not retrying", "app", appName, "subscriptionID", subscriptionID, "error", err)
			return err
		}

		if attempt < maxRetries-1 {
			delay := baseDelay * time.Duration(1<<uint(attempt))
			d.logger.Warn("Failed to send notification, retrying", "app", appName, "subscriptionID", subscriptionID, "attempt", attempt+1, "error", err, "retryIn", delay)
			time.Sleep(delay)
		}
	}

	d.logger.Error("Failed to send notification after retries", "app", appName, "subscriptionID", subscriptionID, "attempts", maxRetries, "error", lastErr)
	return lastErr
}

func (d *Dispatcher) CreateAppWithDefaultSubscription(appName string) (*App, error) {
	app, signalErr := d.trySignalAutoConfig(appName)
	if signalErr == nil {
		return app, nil
	}

	d.logger.Warn("Signal auto-config failed, trying Telegram", "app", appName, "error", signalErr)

	app, telegramErr := d.tryTelegramAutoConfig(appName)
	if telegramErr == nil {
		return app, nil
	}

	return nil, fmt.Errorf("signal auto-config failed: %v; telegram unavailable: %w", signalErr, telegramErr)
}

func (d *Dispatcher) trySignalAutoConfig(appName string) (*App, error) {
	sender, ok := d.senders[ChannelSignal]
	if !ok {
		return nil, fmt.Errorf("signal sender not found")
	}

	linkChecker, ok := sender.(linkedChecker)
	if !ok {
		return nil, fmt.Errorf("sender does not implement signal linked check")
	}

	linked, err := linkChecker.IsLinked()
	if err != nil {
		return nil, fmt.Errorf("failed to check Signal link status: %w", err)
	}
	if !linked {
		return nil, fmt.Errorf("no linked Signal account")
	}

	autoConfigurer, ok := sender.(signalAutoConfigurer)
	if !ok {
		return nil, fmt.Errorf("sender does not implement signal auto-configuration")
	}

	var signalSub *SignalSubscription

	cachedGroup, err := d.store.GetSignalGroup(appName)
	if err != nil {
		d.logger.Warn("Failed to check for cached Signal group", "error", err)
	}

	if cachedGroup != nil {
		d.logger.Debug("Reusing cached Signal group", "app", appName)
		signalSub = cachedGroup
	} else {
		signalSub, err = autoConfigurer.CreateDefaultSignalSubscription(appName)
		if err != nil {
			return nil, err
		}
	}

	subID, err := GenerateSubscriptionID()
	if err != nil {
		return nil, err
	}

	sub := Subscription{ID: subID, AppName: appName, Channel: ChannelSignal, Signal: signalSub}
	if err := d.store.AddSubscription(sub); err != nil {
		return nil, err
	}

	d.logger.Info("Auto-configured Signal subscription", "app", appName, "subscriptionID", subID)
	return d.store.GetApp(appName)
}

func (d *Dispatcher) tryTelegramAutoConfig(appName string) (*App, error) {
	chatID, err := d.getTelegramChatID()
	if err != nil || chatID == "" {
		return nil, fmt.Errorf("telegram not linked or no chat ID configured: %w", err)
	}

	subID, err := GenerateSubscriptionID()
	if err != nil {
		return nil, err
	}

	sub := Subscription{
		ID:       subID,
		AppName:  appName,
		Channel:  ChannelTelegram,
		Telegram: &TelegramSubscription{ChatID: chatID},
	}
	if err := d.store.AddSubscription(sub); err != nil {
		return nil, err
	}

	d.logger.Info("Auto-configured Telegram subscription", "app", appName, "subscriptionID", subID, "chatID", chatID)
	return d.store.GetApp(appName)
}

func (d *Dispatcher) getTelegramChatID() (string, error) {
	sender, ok := d.senders[ChannelTelegram]
	if !ok {
		return "", fmt.Errorf("telegram sender not found")
	}

	linkChecker, ok := sender.(linkedChecker)
	if !ok {
		return "", fmt.Errorf("sender does not implement telegram linked check")
	}

	linked, err := linkChecker.IsLinked()
	if err != nil {
		return "", err
	}
	if !linked {
		return "", fmt.Errorf("telegram not linked")
	}

	tgSender, ok := sender.(telegramSender)
	if !ok {
		return "", fmt.Errorf("sender does not implement telegram interface")
	}

	chatID := tgSender.GetChatID()
	if chatID == 0 {
		return "", fmt.Errorf("no chat ID configured")
	}

	return fmt.Sprintf("%d", chatID), nil
}

func GenerateSubscriptionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
