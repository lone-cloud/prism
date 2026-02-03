package proton

import (
	"context"
	"fmt"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"
)

func (m *Monitor) connect() error {
	addr := fmt.Sprintf("%s:%d", m.cfg.ProtonBridgeHost, m.cfg.ProtonBridgePort)

	options := &imapclient.Options{
		UnilateralDataHandler: &imapclient.UnilateralDataHandler{
			Mailbox: func(data *imapclient.UnilateralDataMailbox) {
				if data.NumMessages != nil {
					select {
					case m.newMessagesChan <- struct{}{}:
					default:
						m.logger.Warn("New messages channel full, skipping signal")
					}
				}
			},
		},
	}

	c, err := imapclient.DialInsecure(addr, options)
	if err != nil {
		return fmt.Errorf("failed to dial: %w", err)
	}

	if err := c.Login(m.cfg.ProtonIMAPUsername, m.cfg.ProtonIMAPPassword).Wait(); err != nil {
		c.Close()
		return fmt.Errorf("failed to login: %w", err)
	}

	m.client = c
	m.logger.Info("Connected to Proton Bridge")
	return nil
}

func (m *Monitor) monitor(ctx context.Context) error {
	selectCmd := m.client.Select(m.cfg.IMAPInbox, nil)
	_, err := selectCmd.Wait()
	if err != nil {
		return fmt.Errorf("failed to select inbox: %w", err)
	}

	if m.monitorStartTime.IsZero() {
		m.monitorStartTime = time.Now()
		m.logger.Info("Proton Mail monitor start time set", "time", m.monitorStartTime)
	}

	idleCmd, err := m.client.Idle()
	if err != nil {
		return fmt.Errorf("failed to start idle: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			idleCmd.Close()
			return ctx.Err()

		case <-m.newMessagesChan:
			idleCmd.Close()

			if err := m.sendNotification(); err != nil {
				m.logger.Error("Failed to send notification", "error", err)
			}

			idleCmd, err = m.client.Idle()
			if err != nil {
				return fmt.Errorf("failed to restart idle: %w", err)
			}
		}
	}
}
