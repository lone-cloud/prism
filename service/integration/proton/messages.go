package proton

import (
	"time"

	"prism/service/notification"

	"github.com/emersion/hydroxide/protonmail"
)

func (m *Monitor) processMessageEvents(events []*protonmail.EventMessage) {
	for _, evt := range events {
		switch evt.Action {
		case protonmail.EventCreate:
			if evt.Created != nil {
				msg := evt.Created
				if msg.Unread == 1 && hasLabel(msg, protonmail.LabelInbox) && msg.Time.Time().After(m.startTime) {
					if _, seen := m.unseenMessageIDs[msg.ID]; !seen {
						m.unseenMessageIDs[msg.ID] = time.Now()
						m.sendNotification(msg)
					}
				}
			}
		case protonmail.EventUpdate, protonmail.EventUpdateFlags:
			if evt.Updated != nil && evt.Updated.Unread != nil && *evt.Updated.Unread == 0 {
				if _, wasSent := m.unseenMessageIDs[evt.ID]; wasSent {
					m.clearNotification(evt.ID)
				}
				delete(m.unseenMessageIDs, evt.ID)
			}
		case protonmail.EventDelete:
			if _, wasSent := m.unseenMessageIDs[evt.ID]; wasSent {
				m.clearNotification(evt.ID)
			}
			delete(m.unseenMessageIDs, evt.ID)
		}
	}
}

func (m *Monitor) cleanupOldMessages() {
	cutoff := time.Now().Add(-24 * time.Hour)
	for msgID, notifiedAt := range m.unseenMessageIDs {
		if notifiedAt.Before(cutoff) {
			delete(m.unseenMessageIDs, msgID)
		}
	}
	if len(m.unseenMessageIDs) > 0 {
		m.logger.Debug("Cleaned up old message IDs", "remaining", len(m.unseenMessageIDs))
	}
}

func hasLabel(msg *protonmail.Message, labelID string) bool {
	for _, id := range msg.LabelIDs {
		if id == labelID {
			return true
		}
	}
	return false
}

func (m *Monitor) sendNotification(msg *protonmail.Message) {
	from := "Unknown"
	if msg.Sender != nil {
		if msg.Sender.Name != "" {
			from = msg.Sender.Name
		} else {
			from = msg.Sender.Address
		}
	}

	subject := msg.Subject
	if subject == "" {
		subject = "(No subject)"
	}

	notif := notification.Notification{
		Title:   from,
		Message: subject,
		Tag:     "proton-" + msg.ID,
		Actions: []notification.Action{
			{
				ID:       "archive",
				Label:    "Archive",
				Endpoint: "/api/proton/archive",
				Method:   "POST",
				Data: map[string]any{
					"uid": msg.ID,
				},
			},
			{
				ID:       "mark-read",
				Label:    "Mark as Read",
				Endpoint: "/api/proton/mark-read",
				Method:   "POST",
				Data: map[string]any{
					"uid": msg.ID,
				},
			},
		},
	}

	if err := m.dispatcher.Send(prismTopic, notif); err != nil {
		m.logger.Error("Failed to send notification", "error", err)
	} else {
		m.logger.Info("Sent notification", "from", from, "subject", subject, "msgID", msg.ID)
	}
}

func (m *Monitor) clearNotification(msgID string) {
	mapping, err := m.dispatcher.GetStore().GetApp(prismTopic)
	if err != nil || mapping == nil {
		return
	}

	if mapping.Channel != notification.ChannelWebPush {
		return
	}

	notif := notification.Notification{
		Tag:     "proton-" + msgID,
		Title:   "",
		Message: "",
	}

	if err := m.dispatcher.Send(prismTopic, notif); err != nil {
		m.logger.Error("Failed to clear notification", "error", err, "msgID", msgID)
	} else {
		m.logger.Debug("Cleared notification", "msgID", msgID)
	}
}
