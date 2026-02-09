package proton

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"prism/service/config"
	"prism/service/credentials"
	"prism/service/notification"

	"github.com/emersion/hydroxide/protonmail"
)

const (
	pollInterval = 30 * time.Second
	prismTopic   = "Proton Mail"
)

type Monitor struct {
	cfg              *config.Config
	dispatcher       *notification.Dispatcher
	logger           *slog.Logger
	credStore        *credentials.Store
	client           *protonmail.Client
	eventID          string
	unseenMessageIDs map[string]time.Time
	startTime        time.Time
}

func NewMonitor(cfg *config.Config, dispatcher *notification.Dispatcher, logger *slog.Logger) *Monitor {
	return &Monitor{
		cfg:        cfg,
		dispatcher: dispatcher,
		logger:     logger,
	}
}

func (m *Monitor) Start(ctx context.Context, credStore *credentials.Store) error {
	creds, err := credStore.GetProton()
	if err != nil {
		m.logger.Debug("Proton credentials not configured", "error", err)
		return nil
	}

	m.credStore = credStore
	m.logger.Info("Starting Proton Mail monitor", "email", creds.Email)

	c := &protonmail.Client{
		RootURL:    "https://mail.proton.me/api",
		AppVersion: "Other",
	}

	var auth *protonmail.Auth

	if creds.UID != "" && creds.AccessToken != "" && creds.RefreshToken != "" {
		auth = &protonmail.Auth{
			UID:          creds.UID,
			AccessToken:  creds.AccessToken,
			RefreshToken: creds.RefreshToken,
			Scope:        creds.Scope,
		}

		_, err = c.Unlock(auth, creds.KeySalts, creds.Password)
		if err != nil {
			m.logger.Error("Failed to unlock keys - password may have changed", "error", err)
			if deleteErr := credStore.DeleteIntegration(credentials.IntegrationProton); deleteErr != nil {
				m.logger.Error("Failed to clear invalid credentials", "error", deleteErr)
			}
			return fmt.Errorf("failed to unlock keys (password changed?): %v", err)
		}

		m.logger.Info("Restored Proton session from stored tokens")
	} else if creds.Password != "" {
		authInfo, err := c.AuthInfo(creds.Email)
		if err != nil {
			return err
		}

		authResult, err := c.Auth(creds.Email, creds.Password, authInfo)
		if err != nil {
			return err
		}
		auth = authResult

		keySalts, err := c.ListKeySalts()
		if err != nil {
			return fmt.Errorf("failed to get key salts: %v", err)
		}

		_, err = c.Unlock(auth, keySalts, creds.Password)
		if err != nil {
			m.logger.Error("Failed to unlock keys", "error", err)
			if deleteErr := credStore.DeleteIntegration(credentials.IntegrationProton); deleteErr != nil {
				m.logger.Error("Failed to clear invalid credentials", "error", deleteErr)
			}
			return fmt.Errorf("failed to unlock keys: %v", err)
		}

		creds.KeySalts = keySalts
		if err := credStore.SaveProton(creds); err != nil {
			m.logger.Warn("Failed to cache key salts", "error", err)
		}

		m.logger.Info("Authenticated and unlocked Proton session")
	} else {
		return fmt.Errorf("no valid credentials found - need password or tokens")
	}

	c.ReAuth = func() error {
		newAuth, err := c.AuthRefresh(auth)
		if err != nil {
			m.logger.Error("Token refresh failed", "error", err)
			return err
		}

		_, err = c.Unlock(newAuth, creds.KeySalts, creds.Password)
		if err != nil {
			m.logger.Error("Token refresh failed - cannot unlock keys", "error", err)
			return err
		}

		auth = newAuth

		updatedCreds, err := m.credStore.GetProton()
		if err != nil {
			m.logger.Warn("Failed to get credentials for token update", "error", err)
			return nil
		}

		updatedCreds.UID = newAuth.UID
		updatedCreds.AccessToken = newAuth.AccessToken
		updatedCreds.RefreshToken = newAuth.RefreshToken
		updatedCreds.Scope = newAuth.Scope

		if err := m.credStore.SaveProton(updatedCreds); err != nil {
			m.logger.Warn("Failed to save refreshed tokens", "error", err)
		} else {
			m.logger.Info("Proton tokens refreshed and saved")
		}

		return nil
	}

	m.client = c
	m.startTime = time.Now()

	if creds.State != nil {
		m.eventID = creds.State.LastEventID
		m.logger.Info("Restored Proton state", "eventID", m.eventID)
	} else {
		m.eventID = auth.EventID
		if err := m.saveState(creds); err != nil {
			m.logger.Warn("Failed to save initial state", "error", err)
		}
		m.logger.Info("Initialized Proton state", "eventID", m.eventID)
	}

	m.unseenMessageIDs = make(map[string]time.Time)

	go m.pollEvents(ctx)

	return nil
}

func (m *Monitor) pollEvents(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	cleanupTicker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.checkEvents(); err != nil {
				m.logger.Error("Failed to check events", "error", err)
			}
		case <-cleanupTicker.C:
			m.cleanupOldMessages()
		}
	}
}

func (m *Monitor) checkEvents() error {
	if m.client == nil {
		return nil
	}

	event, err := m.client.GetEvent(m.eventID)
	if err != nil {
		return err
	}

	if event.ID != m.eventID {
		m.processMessageEvents(event.Messages)
		m.eventID = event.ID

		creds, err := m.credStore.GetProton()
		if err != nil {
			return err
		}
		if err := m.saveState(creds); err != nil {
			m.logger.Warn("Failed to save state", "error", err)
		}
	}

	return nil
}

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

func (m *Monitor) saveState(creds *credentials.ProtonCredentials) error {
	creds.State = &credentials.ProtonState{
		LastEventID: m.eventID,
	}
	return m.credStore.SaveProton(creds)
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

func (m *Monitor) IsConnected() bool {
	return m.client != nil
}

func (m *Monitor) MarkAsRead(msgID string) error {
	if m.client == nil {
		return nil
	}
	return m.client.MarkMessagesRead([]string{msgID})
}

func (m *Monitor) Archive(msgID string) error {
	if m.client == nil {
		return nil
	}
	return m.client.UnlabelMessages(protonmail.LabelInbox, []string{msgID})
}
