package notification

import (
	"fmt"
	"log/slog"
	"time"

	"prism/service/util"
)

type NotificationSender interface {
	Send(mapping *Mapping, notif Notification) error
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

func (d *Dispatcher) RegisterApp(appName string) error {
	availableChannels := d.GetAvailableChannels()
	return d.store.RegisterDefault(appName, availableChannels)
}

func (d *Dispatcher) Send(appName string, notif Notification) error {
	mapping, err := d.store.GetApp(appName)
	if err != nil {
		return util.LogError(d.logger, "Failed to get app mapping", err, "app", appName)
	}

	if mapping == nil {
		d.logger.Info("Registering new app", "app", appName)

		availableChannels := d.GetAvailableChannels()
		if err := d.store.RegisterDefault(appName, availableChannels); err != nil {
			return util.LogError(d.logger, "Failed to register app", err, "app", appName)
		}

		mapping, err = d.store.GetApp(appName)
		if err != nil {
			return util.LogError(d.logger, "Failed to get mapping after registration", err, "app", appName)
		}

		if mapping == nil {
			return fmt.Errorf("mapping still nil after registration for app: %s", appName)
		}
	}

	sender, ok := d.senders[mapping.Channel]
	if !ok {
		d.logger.Error("No sender registered for channel", "channel", mapping.Channel)
		return fmt.Errorf("no sender for channel: %s", mapping.Channel)
	}

	return d.sendWithRetry(sender, mapping, notif, appName)
}

func (d *Dispatcher) sendWithRetry(sender NotificationSender, mapping *Mapping, notif Notification, appName string) error {
	maxRetries := 10
	baseDelay := 500 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := sender.Send(mapping, notif)
		if err == nil {
			if attempt > 0 {
				d.logger.Info("Notification sent after retry", "app", appName, "attempt", attempt+1)
			}
			return nil
		}

		lastErr = err
		if attempt < maxRetries-1 {
			delay := baseDelay * time.Duration(1<<uint(attempt))
			d.logger.Warn("Failed to send notification, retrying", "app", appName, "attempt", attempt+1, "error", err, "retryIn", delay)
			time.Sleep(delay)
		}
	}

	d.logger.Error("Failed to send notification after retries", "app", appName, "attempts", maxRetries, "error", lastErr)
	return lastErr
}
