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

func (s *Sender) IsLinked() (bool, error) {
	if s.client == nil {
		return false, nil
	}

	account, err := s.client.GetLinkedAccount()
	if err != nil {
		return false, err
	}

	return account != nil, nil
}

func (s *Sender) CreateDefaultSignalSubscription(appName string) (*notification.SignalSubscription, error) {
	if s.client == nil {
		return nil, fmt.Errorf("signal integration not enabled")
	}

	groupID, account, err := s.client.CreateGroup(appName)
	if err != nil {
		return nil, err
	}

	return &notification.SignalSubscription{
		GroupID: groupID,
		Account: account,
	}, nil
}

func (s *Sender) Send(sub *notification.Subscription, notif notification.Notification) error {
	if s.client == nil {
		return notification.NewPermanentError(fmt.Errorf("signal integration not enabled"))
	}

	if sub.Signal == nil {
		return notification.NewPermanentError(fmt.Errorf("no signal subscription data"))
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
	needsNewGroup := sub.Signal.GroupID == ""

	if !needsNewGroup && sub.Signal.Account != account.Number {
		s.logger.Info("Signal account changed, recreating group",
			"app", sub.AppName,
			"oldAccount", sub.Signal.Account,
			"newAccount", account.Number)
		needsNewGroup = true
	}

	if needsNewGroup {
		return notification.NewPermanentError(fmt.Errorf("signal group not configured for subscription %s", sub.ID))
	}

	signalGroupID = sub.Signal.GroupID

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
