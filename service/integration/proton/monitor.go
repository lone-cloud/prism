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
	consecutiveErrs  int
	lastConnected    time.Time
}

func NewMonitor(cfg *config.Config, dispatcher *notification.Dispatcher, logger *slog.Logger) *Monitor {
	return &Monitor{
		cfg:        cfg,
		dispatcher: dispatcher,
		logger:     logger,
	}
}

func (m *Monitor) Start(ctx context.Context, credStore *credentials.Store) error {
	if err := m.authenticateAndSetup(credStore); err != nil {
		return err
	}

	if m.client == nil {
		return nil
	}

	m.startTime = time.Now()
	m.lastConnected = time.Now()
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
			if err := m.checkEventsWithRetry(ctx); err != nil {
				m.logger.Error("Failed to check events after retries", "error", err, "consecutive_errors", m.consecutiveErrs)
				m.consecutiveErrs++
			} else {
				if m.consecutiveErrs > 0 {
					m.logger.Info("Proton connection recovered", "downtime", time.Since(m.lastConnected).String())
					m.consecutiveErrs = 0
				}
				m.lastConnected = time.Now()
			}
		case <-cleanupTicker.C:
			m.cleanupOldMessages()
		}
	}
}

func (m *Monitor) checkEventsWithRetry(ctx context.Context) error {
	maxRetries := 3
	baseDelay := 1 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			m.logger.Debug("Retrying event check", "attempt", attempt+1, "delay", delay)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := m.checkEvents()
		if err == nil {
			return nil
		}

		if attempt < maxRetries-1 {
			m.logger.Warn("Event check failed, will retry", "error", err, "attempt", attempt+1)
		}
	}

	return fmt.Errorf("event check failed after %d attempts", maxRetries)
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

func (m *Monitor) saveState(creds *credentials.ProtonCredentials) error {
	creds.State = &credentials.ProtonState{
		LastEventID: m.eventID,
	}
	return m.credStore.SaveProton(creds)
}

func (m *Monitor) IsConnected() bool {
	if m.client == nil {
		return false
	}
	return m.consecutiveErrs < 5
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
