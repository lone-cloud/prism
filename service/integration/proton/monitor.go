package proton

import (
	"context"
	"log/slog"
	"time"

	"prism/service/config"
	"prism/service/notification"

	"github.com/emersion/go-imap/v2/imapclient"
)

const (
	imapInbox                = "INBOX"
	imapReconnectBaseDelay   = 10 * time.Second
	imapMaxReconnectDelay    = 5 * time.Minute
	imapMaxReconnectAttempts = 50
	prismTopic               = "Proton Mail"
)

type Monitor struct {
	cfg              *config.Config
	dispatcher       *notification.Dispatcher
	logger           *slog.Logger
	client           *imapclient.Client
	monitorStartTime time.Time
	newMessagesChan  chan struct{}
}

func NewMonitor(cfg *config.Config, dispatcher *notification.Dispatcher, logger *slog.Logger) *Monitor {
	return &Monitor{
		cfg:             cfg,
		dispatcher:      dispatcher,
		logger:          logger,
		newMessagesChan: make(chan struct{}, 10),
	}
}

func (m *Monitor) Start(ctx context.Context) error {
	if !m.cfg.IsProtonEnabled() {
		return nil
	}

	m.logger.Info("Starting Proton Mail monitor")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := m.connect(); err != nil {
				m.logger.Error("Failed to connect to IMAP", "error", err)
				time.Sleep(imapReconnectBaseDelay)
				continue
			}

			if err := m.monitor(ctx); err != nil {
				m.logger.Error("Monitor error", "error", err)
			}

			if m.client != nil {
				if err := m.client.Logout(); err != nil {
					m.logger.Error("Logout failed", "error", err)
				}
				m.client = nil
			}

			time.Sleep(imapReconnectBaseDelay)
		}
	}
}

func (m *Monitor) IsConnected() bool {
	return m.client != nil
}
