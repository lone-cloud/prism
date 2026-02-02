package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

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
		return fmt.Errorf("failed to get mapping: %w", err)
	}

	if mapping == nil {
		appName := strings.TrimPrefix(endpoint, "ntfy-")
		appName = strings.TrimPrefix(appName, "proton-")

		if err := d.store.Register(endpoint, appName, ChannelSignal, nil, nil); err != nil {
			return fmt.Errorf("failed to register endpoint: %w", err)
		}

		mapping, err = d.store.GetEndpointMapping(endpoint)
		if err != nil {
			return fmt.Errorf("failed to get mapping after registration: %w", err)
		}
	}

	switch mapping.Channel {
	case ChannelWebhook:
		return d.sendWebhook(mapping, notif)
	case ChannelSignal:
		return d.sendSignal(mapping, notif)
	default:
		return fmt.Errorf("unknown channel: %s", mapping.Channel)
	}
}

func (d *Dispatcher) sendSignal(mapping *Mapping, notif Notification) error {
	groupID := mapping.GroupID
	if groupID == nil || *groupID == "" {
		newGroupID, err := d.createGroup(mapping.AppName)
		if err != nil {
			return fmt.Errorf("failed to create group: %w", err)
		}
		groupID = &newGroupID

		if err := d.store.UpdateGroupID(mapping.Endpoint, *groupID); err != nil {
			d.logger.Warn("Failed to update group ID", "error", err)
		}
	}

	if err := d.sendGroupMessage(*groupID, notif); err != nil {
		return fmt.Errorf("failed to send group message: %w", err)
	}

	d.logger.Debug("Sent Signal notification", "app", mapping.AppName, "message", notif.Message)
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
		return fmt.Errorf("failed to get linked account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("no linked Signal account")
	}

	message := notif.Message
	if notif.Title != "" {
		message = fmt.Sprintf("%s\n\n%s", notif.Title, notif.Message)
	}

	params := map[string]interface{}{
		"groupId":     groupID,
		"message":     message,
		"notify-self": true,
	}

	_, err = d.signalClient.CallWithAccount("send", params, account.Number)
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
