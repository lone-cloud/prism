package signal

import (
	"fmt"
	"log/slog"

	"prism/service/delivery"
	"prism/service/subscription"
	"prism/service/util"
)

type Sender struct {
	client *Client
	groups *GroupCache
	logger *slog.Logger
}

func NewSender(client *Client, groups *GroupCache, logger *slog.Logger) *Sender {
	return &Sender{
		client: client,
		groups: groups,
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

func (s *Sender) CreateDefaultSignalSubscription(appName string) (*subscription.SignalSubscription, error) {
	if s.client == nil {
		return nil, fmt.Errorf("signal integration not enabled")
	}

	groupID, account, err := s.client.CreateGroup(appName)
	if err != nil {
		return nil, err
	}

	signalSub := &subscription.SignalSubscription{
		GroupID: groupID,
		Account: account,
	}

	if err := s.groups.Save(appName, signalSub); err != nil {
		s.logger.Warn("Failed to cache Signal group", "error", err)
	}

	return signalSub, nil
}

func (s *Sender) Send(sub *subscription.Subscription, notif delivery.Notification) error {
	if s.client == nil {
		return delivery.NewPermanentError(fmt.Errorf("signal integration not enabled"))
	}

	if sub.Signal == nil {
		return delivery.NewPermanentError(fmt.Errorf("no signal subscription data"))
	}

	account, err := s.client.GetLinkedAccount()
	if err != nil {
		return util.LogError(s.logger, "Failed to get linked account", err)
	}
	if account == nil {
		s.logger.Error("No linked Signal account found")
		return delivery.NewPermanentError(fmt.Errorf("no linked Signal account"))
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
		return delivery.NewPermanentError(fmt.Errorf("signal group not configured for subscription %s", sub.ID))
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
