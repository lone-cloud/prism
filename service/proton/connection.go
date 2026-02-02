package proton

import (
	"context"
	"fmt"
	"time"

	"github.com/emersion/go-imap/client"
)

func (m *Monitor) connect() error {
	addr := fmt.Sprintf("%s:%d", m.cfg.ProtonBridgeHost, m.cfg.ProtonBridgePort)
	c, err := client.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to dial: %w", err)
	}

	if err := c.Login(m.cfg.ProtonIMAPUsername, m.cfg.ProtonIMAPPassword); err != nil {
		if logoutErr := c.Logout(); logoutErr != nil {
			m.logger.Error("Logout failed", "error", logoutErr)
		}
		return fmt.Errorf("failed to login: %w", err)
	}

	m.client = c
	m.logger.Info("Connected to Proton Bridge")
	return nil
}

func (m *Monitor) monitor(ctx context.Context) error {
	_, err := m.client.Select(m.cfg.IMAPInbox, false)
	if err != nil {
		return fmt.Errorf("failed to select inbox: %w", err)
	}

	if m.monitorStartTime.IsZero() {
		m.monitorStartTime = time.Now()
		m.logger.Info("Monitor start time set", "time", m.monitorStartTime)
	}

	updates := make(chan client.Update, 10)
	m.client.Updates = updates

	stop := make(chan struct{})
	defer close(stop)

	idleErr := make(chan error, 1)
	go func() {
		idleErr <- m.client.Idle(stop, nil)
	}()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case update := <-updates:
			switch u := update.(type) {
			case *client.MailboxUpdate:
				if u.Mailbox.UnseenSeqNum > 0 {
					if err := m.handleNewMessages(); err != nil {
						m.logger.Error("Failed to handle new messages", "error", err)
					}
				}
			}

		case <-ticker.C:
			close(stop)
			<-idleErr

			if err := m.client.Noop(); err != nil {
				return fmt.Errorf("noop failed: %w", err)
			}

			stop = make(chan struct{})
			go func() {
				idleErr <- m.client.Idle(stop, nil)
			}()

		case err := <-idleErr:
			return fmt.Errorf("idle ended: %w", err)
		}
	}
}
