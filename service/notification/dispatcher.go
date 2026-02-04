package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"prism/service/signal"

	webpush "github.com/SherClockHolmes/webpush-go"
)

type Dispatcher struct {
	store        *Store
	signalClient *signal.Client
	logger       *slog.Logger
}

func NewDispatcher(store *Store, signalClient *signal.Client, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{
		store:        store,
		signalClient: signalClient,
		logger:       logger,
	}
}

func (d *Dispatcher) Send(appName string, notif Notification) error {
	mapping, err := d.store.GetApp(appName)
	if err != nil {
		d.logger.Error("Failed to get app mapping", "app", appName, "error", err)
		return fmt.Errorf("failed to get mapping: %w", err)
	}

	if mapping == nil {
		d.logger.Info("Registering new app", "app", appName)

		if err := d.store.RegisterDefault(appName); err != nil {
			d.logger.Error("Failed to register app", "app", appName, "error", err)
			return fmt.Errorf("failed to register app: %w", err)
		}

		mapping, err = d.store.GetApp(appName)
		if err != nil {
			d.logger.Error("Failed to get mapping after registration", "app", appName, "error", err)
			return fmt.Errorf("failed to get mapping after registration: %w", err)
		}
	}

	switch mapping.Channel {
	case ChannelWebPush:
		return d.sendWebPush(mapping, notif)
	case ChannelSignal:
		return d.sendSignal(mapping, notif)
	default:
		d.logger.Error("Unknown channel type", "channel", mapping.Channel)
		return fmt.Errorf("unknown channel: %s", mapping.Channel)
	}
}

func (d *Dispatcher) sendSignal(mapping *Mapping, notif Notification) error {
	account, err := d.signalClient.GetLinkedAccount()
	if err != nil {
		d.logger.Error("Failed to get linked account", "error", err)
		return fmt.Errorf("failed to get linked account: %w", err)
	}
	if account == nil {
		d.logger.Error("No linked Signal account found")
		return fmt.Errorf("no linked Signal account")
	}

	var signalGroupID string
	needsNewGroup := mapping.Signal == nil || mapping.Signal.GroupID == ""

	if !needsNewGroup && mapping.Signal.Account != account.Number {
		d.logger.Info("Signal account changed, recreating group",
			"app", mapping.AppName,
			"oldAccount", mapping.Signal.Account,
			"newAccount", account.Number)
		needsNewGroup = true
	}

	if needsNewGroup {
		d.logger.Info("Creating new group", "app", mapping.AppName, "account", account.Number)

		newGroupID, accountNumber, err := d.createGroup(mapping.AppName)
		if err != nil {
			d.logger.Error("Failed to create group", "app", mapping.AppName, "error", err)
			return fmt.Errorf("failed to create group: %w", err)
		}
		signalGroupID = newGroupID

		d.logger.Info("Created new group", "app", mapping.AppName, "groupID", signalGroupID, "account", accountNumber)

		if err := d.store.UpdateSignal(mapping.AppName, &SignalSubscription{
			GroupID: signalGroupID,
			Account: accountNumber,
		}); err != nil {
			d.logger.Warn("Failed to update signal subscription", "error", err)
		}
	} else {
		signalGroupID = mapping.Signal.GroupID
	}

	if err := d.sendGroupMessage(signalGroupID, notif); err != nil {
		d.logger.Error("Failed to send group message", "groupID", signalGroupID, "error", err)
	}

	return nil
}

func (d *Dispatcher) createGroup(appName string) (string, string, error) {
	account, err := d.signalClient.GetLinkedAccount()
	if err != nil {
		return "", "", fmt.Errorf("failed to get linked account: %w", err)
	}
	if account == nil {
		return "", "", fmt.Errorf("no linked Signal account")
	}

	params := map[string]interface{}{
		"name":   appName,
		"member": []string{},
	}

	result, err := d.signalClient.CallWithAccount("updateGroup", params, account.Number)
	if err != nil {
		return "", "", err
	}

	var response struct {
		GroupID string `json:"groupId"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return "", "", fmt.Errorf("failed to parse updateGroup response: %w", err)
	}

	if response.GroupID == "" {
		return "", "", fmt.Errorf("empty groupId in response")
	}

	return response.GroupID, account.Number, nil
}

func (d *Dispatcher) sendGroupMessage(groupID string, notif Notification) error {
	account, err := d.signalClient.GetLinkedAccount()
	if err != nil {
		d.logger.Error("Failed to get linked account", "error", err)
		return fmt.Errorf("failed to get linked account: %w", err)
	}
	if account == nil {
		d.logger.Error("No linked Signal account found")
		return fmt.Errorf("no linked Signal account")
	}

	message := notif.Message
	if notif.Title != "" {
		message = fmt.Sprintf("%s\n%s", notif.Title, notif.Message)
	}

	params := map[string]interface{}{
		"groupId":     groupID,
		"message":     message,
		"notify-self": true,
	}

	_, err = d.signalClient.CallWithAccount("send", params, account.Number)
	if err != nil {
		d.logger.Error("Signal send API call failed", "error", err)
	}
	return err
}

func (d *Dispatcher) sendWebPush(mapping *Mapping, notif Notification) error {
	if mapping.WebPush == nil {
		return fmt.Errorf("no push endpoint configured for %s", mapping.AppName)
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

		d.logger.Debug("Sent encrypted webpush notification", "app", mapping.AppName, "url", mapping.WebPush.Endpoint)
	} else {
		resp, err := http.Post(mapping.WebPush.Endpoint, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			return fmt.Errorf("failed to send webhook: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("webhook returned status %d", resp.StatusCode)
		}

		d.logger.Debug("Sent plain webhook notification", "app", mapping.AppName, "url", mapping.WebPush.Endpoint)
	}

	return nil
}
