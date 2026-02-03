package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"prism/service/signal"
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

func (d *Dispatcher) Send(endpoint string, notif Notification) error {
	mapping, err := d.store.GetEndpointMapping(endpoint)
	if err != nil {
		d.logger.Error("Failed to get endpoint mapping", "endpoint", endpoint, "error", err)
		return fmt.Errorf("failed to get mapping: %w", err)
	}

	if mapping == nil {
		d.logger.Info("No mapping found for endpoint, creating new registration", "endpoint", endpoint)

		d.logger.Info("Registering new endpoint", "endpoint", endpoint, "appName", endpoint, "channel", ChannelSignal)

		if err := d.store.Register(endpoint, endpoint, ChannelSignal, nil, nil); err != nil {
			d.logger.Error("Failed to register endpoint", "endpoint", endpoint, "error", err)
			return fmt.Errorf("failed to register endpoint: %w", err)
		}

		mapping, err = d.store.GetEndpointMapping(endpoint)
		if err != nil {
			d.logger.Error("Failed to get mapping after registration", "endpoint", endpoint, "error", err)
			return fmt.Errorf("failed to get mapping after registration: %w", err)
		}
	}

	switch mapping.Channel {
	case ChannelWebhook:
		return d.sendWebhook(mapping, notif)
	case ChannelSignal:
		return d.sendSignal(mapping, notif)
	default:
		d.logger.Error("Unknown channel type", "channel", mapping.Channel)
		return fmt.Errorf("unknown channel: %s", mapping.Channel)
	}
}

func (d *Dispatcher) sendSignal(mapping *Mapping, notif Notification) error {
	groupID := mapping.GroupID
	if groupID == nil || *groupID == "" {
		d.logger.Info("No group ID found, creating new group", "appName", mapping.AppName)

		newGroupID, err := d.createGroup(mapping.AppName)
		if err != nil {
			d.logger.Error("Failed to create group", "appName", mapping.AppName, "error", err)
			return fmt.Errorf("failed to create group: %w", err)
		}
		groupID = &newGroupID

		d.logger.Info("Created new group", "appName", mapping.AppName, "groupID", *groupID)

		if err := d.store.UpdateGroupID(mapping.Endpoint, *groupID); err != nil {
			d.logger.Warn("Failed to update group ID", "error", err)
		}
	}

	if err := d.sendGroupMessage(*groupID, notif); err != nil {
		d.logger.Error("Failed to send group message", "groupID", *groupID, "error", err)
		return fmt.Errorf("failed to send group message: %w", err)
	}

	return nil
}

func (d *Dispatcher) createGroup(appName string) (string, error) {
	account, err := d.signalClient.GetLinkedAccount()
	if err != nil {
		return "", fmt.Errorf("failed to get linked account: %w", err)
	}
	if account == nil {
		return "", fmt.Errorf("no linked Signal account")
	}

	params := map[string]interface{}{
		"name":   appName,
		"member": []string{},
	}

	result, err := d.signalClient.CallWithAccount("updateGroup", params, account.Number)
	if err != nil {
		return "", err
	}

	var response struct {
		GroupID string `json:"groupId"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return "", fmt.Errorf("failed to parse updateGroup response: %w", err)
	}

	if response.GroupID == "" {
		return "", fmt.Errorf("empty groupId in response")
	}

	return response.GroupID, nil
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

func (d *Dispatcher) sendWebhook(mapping *Mapping, notif Notification) error {
	if mapping.UpEndpoint == nil || *mapping.UpEndpoint == "" {
		return fmt.Errorf("webhook endpoint not configured for %s", mapping.AppName)
	}

	payload, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	resp, err := http.Post(*mapping.UpEndpoint, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	d.logger.Debug("Sent webhook notification", "app", mapping.AppName, "endpoint", *mapping.UpEndpoint)
	return nil
}
