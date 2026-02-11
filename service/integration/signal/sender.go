package signal

import (
	"fmt"
	"log/slog"

	"prism/service/notification"
	"prism/service/util"
)

type Sender struct {
	client *Client
	store  *notification.Store
	logger *slog.Logger
}

func NewSender(client *Client, store *notification.Store, logger *slog.Logger) *Sender {
	return &Sender{
		client: client,
		store:  store,
		logger: logger,
	}
}

func (s *Sender) Send(mapping *notification.Mapping, notif notification.Notification) error {
	if s.client == nil {
		return notification.NewPermanentError(fmt.Errorf("signal integration not enabled"))
	}

	account, err := s.client.GetLinkedAccount()
	if err != nil {
		return util.LogError(s.logger, "Failed to get linked account", err)
	}
	if account == nil {
		s.logger.Error("No linked Signal account found")
		return notification.NewPermanentError(fmt.Errorf("no linked Signal account"))
	}

	var signalGroupID string
	needsNewGroup := mapping.Signal == nil || mapping.Signal.GroupID == ""

	if !needsNewGroup && mapping.Signal.Account != account.Number {
		s.logger.Info("Signal account changed, recreating group",
			"app", mapping.AppName,
			"oldAccount", mapping.Signal.Account,
			"newAccount", account.Number)
		needsNewGroup = true
	}

	if needsNewGroup {
		newGroupID, accountNumber, err := s.client.CreateGroup(mapping.AppName)
		if err != nil {
			return util.LogError(s.logger, "Failed to create group", err, "app", mapping.AppName)
		}
		signalGroupID = newGroupID

		if err := s.store.UpdateSignal(mapping.AppName, &notification.SignalSubscription{
			GroupID: signalGroupID,
			Account: accountNumber,
		}); err != nil {
			return util.LogError(s.logger, "Failed to persist signal group", err)
		}
	} else {
		signalGroupID = mapping.Signal.GroupID
	}

	message := notif.Message
	if notif.Title != "" {
		message = fmt.Sprintf("%s\n\n%s", notif.Title, notif.Message)
	}

	if err := s.client.SendGroupMessage(signalGroupID, message); err != nil {
		s.logger.Error("Failed to send group message", "groupID", signalGroupID, "error", err)
		return err
	}

	return nil
}
