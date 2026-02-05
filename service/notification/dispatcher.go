package notification

import (
	"fmt"
	"log/slog"
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

func (d *Dispatcher) Send(appName string, notif Notification) error {
	mapping, err := d.store.GetApp(appName)
	if err != nil {
		d.logger.Error("Failed to get app mapping", "app", appName, "error", err)
		return fmt.Errorf("failed to get mapping: %w", err)
	}

	if mapping == nil {
		d.logger.Info("Registering new app", "app", appName)

		availableChannels := d.GetAvailableChannels()
		if err := d.store.RegisterDefault(appName, availableChannels); err != nil {
			d.logger.Error("Failed to register app", "app", appName, "error", err)
			return fmt.Errorf("failed to register app: %w", err)
		}

		mapping, err = d.store.GetApp(appName)
		if err != nil {
			d.logger.Error("Failed to get mapping after registration", "app", appName, "error", err)
			return fmt.Errorf("failed to get mapping after registration: %w", err)
		}
	}

	sender, ok := d.senders[mapping.Channel]
	if !ok {
		d.logger.Error("No sender registered for channel", "channel", mapping.Channel)
		return fmt.Errorf("no sender for channel: %s", mapping.Channel)
	}

	return sender.Send(mapping, notif)
}
