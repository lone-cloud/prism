package proton

import (
	"fmt"

	"prism/service/notification"

	"github.com/emersion/go-imap"
)

func (m *Monitor) handleNewMessages() error {
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{m.cfg.IMAPSeenFlag}

	uids, err := m.client.UidSearch(criteria)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(uids) == 0 {
		return nil
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids...)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	go func() {
		done <- m.client.UidFetch(seqSet, []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid}, messages)
	}()

	for msg := range messages {
		if err := m.processMessage(msg); err != nil {
			m.logger.Error("Failed to process message", "error", err)
		}
	}

	if err := <-done; err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}

	return nil
}

func (m *Monitor) processMessage(msg *imap.Message) error {
	if msg.Envelope == nil {
		return nil
	}

	if msg.Envelope.Date.Before(m.monitorStartTime) {
		m.logger.Debug("Skipping old email", "date", msg.Envelope.Date, "subject", msg.Envelope.Subject)
		return nil
	}

	var from string
	if len(msg.Envelope.From) > 0 {
		addr := msg.Envelope.From[0]
		if addr.PersonalName != "" {
			from = addr.PersonalName
		} else {
			from = addr.Address()
		}
	} else {
		from = "Unknown sender"
	}

	subject := msg.Envelope.Subject
	if subject == "" {
		subject = "No subject"
	}

	endpoint := m.cfg.EndpointPrefixProton + m.cfg.ProtonPrismTopic

	notif := notification.Notification{
		Title:   from,
		Message: subject,
		Actions: []notification.Action{
			{
				ID:       "mark-read",
				Endpoint: "/api/proton-mail/mark-read",
				Method:   "POST",
				Data: map[string]interface{}{
					"uid": msg.Uid,
				},
			},
		},
	}

	if err := m.dispatcher.Send(endpoint, notif); err != nil {
		m.logger.Error("Failed to send notification", "endpoint", endpoint, "error", err)
		return err
	}

	m.logger.Info("Processed Proton Mail notification", "from", from, "subject", subject)
	return nil
}

func (m *Monitor) MarkAsRead(uid uint32) error {
	if m.client == nil {
		return fmt.Errorf("not connected")
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{m.cfg.IMAPSeenFlag}

	return m.client.UidStore(seqSet, item, flags, nil)
}
