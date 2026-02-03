package proton

import (
	"fmt"

	"prism/service/notification"

	"github.com/emersion/go-imap/v2"
)

func (m *Monitor) sendNotification() error {
	selectData, err := m.client.Select(m.cfg.IMAPInbox, nil).Wait()
	if err != nil {
		m.logger.Error("Failed to select inbox", "error", err)
		return err
	}

	if selectData.NumMessages == 0 {
		m.logger.Warn("No messages in mailbox")
		return nil
	}

	latestSeq := selectData.NumMessages

	seqSet := imap.SeqSetNum(latestSeq)

	fetchOptions := &imap.FetchOptions{
		Envelope: true,
		UID:      true,
	}

	fetchCmd := m.client.Fetch(seqSet, fetchOptions)
	defer fetchCmd.Close()

	msg := fetchCmd.Next()
	if msg == nil {
		m.logger.Warn("No message found to send notification")
		return nil
	}

	msgData, err := msg.Collect()
	if err != nil {
		m.logger.Error("Failed to collect message", "error", err)
		return err
	}

	if msgData.Envelope == nil {
		m.logger.Warn("Message has no envelope")
		return nil
	}

	var from string
	if len(msgData.Envelope.From) > 0 {
		addr := msgData.Envelope.From[0]
		if addr.Name != "" {
			from = addr.Name
		} else {
			from = addr.Addr()
		}
	} else {
		from = "Unknown sender"
	}

	subject := msgData.Envelope.Subject
	if subject == "" {
		subject = "No subject"
	}

	notif := notification.Notification{
		Title:   from,
		Message: subject,
		Actions: []notification.Action{
			{
				ID:       "mark-read",
				Endpoint: "/api/proton-mail/mark-read",
				Method:   "POST",
				Data: map[string]interface{}{
					"uid": msgData.UID,
				},
			},
		},
	}

	if err := m.dispatcher.Send(m.cfg.ProtonPrismTopic, notif); err != nil {
		m.logger.Error("Failed to send notification", "error", err)
		return err
	}

	m.logger.Info("Sent ProtonMail notification", "from", from, "subject", subject, "uid", msgData.UID)
	return nil
}

func (m *Monitor) MarkAsRead(uid uint32) error {
	if m.client == nil {
		return fmt.Errorf("not connected")
	}

	uidSet := imap.UIDSet{}
	uidSet.AddNum(imap.UID(uid))

	storeFlags := &imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Flags:  []imap.Flag{imap.FlagSeen},
		Silent: false,
	}

	storeCmd := m.client.Store(uidSet, storeFlags, nil)

	for {
		msg := storeCmd.Next()
		if msg == nil {
			break
		}
	}

	if err := storeCmd.Close(); err != nil {
		return fmt.Errorf("failed to mark message as read: %w", err)
	}

	m.logger.Info("Marked message as read", "uid", uid)
	return nil
}
