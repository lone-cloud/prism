package proton

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/lone-cloud/prism/internal/config"
	"github.com/lone-cloud/prism/internal/notification"
)

type Monitor struct {
	cfg        *config.Config
	dispatcher *notification.Dispatcher
	logger     *slog.Logger
	client     *client.Client
}

func NewMonitor(cfg *config.Config, dispatcher *notification.Dispatcher, logger *slog.Logger) *Monitor {
	return &Monitor{
		cfg:        cfg,
		dispatcher: dispatcher,
		logger:     logger,
	}
}

func (m *Monitor) Start(ctx context.Context) error {
	if !m.cfg.IsProtonEnabled() {
		m.logger.Info("Proton Mail monitoring disabled")
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
				time.Sleep(time.Duration(m.cfg.IMAPReconnectBaseDelay) * time.Millisecond)
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

			time.Sleep(time.Duration(m.cfg.IMAPReconnectBaseDelay) * time.Millisecond)
		}
	}
}

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

func (m *Monitor) handleNewMessages() error {
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{m.cfg.IMAPSeenFlag}

	seqNums, err := m.client.Search(criteria)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(seqNums) == 0 {
		return nil
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(seqNums...)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	section := &imap.BodySectionName{}

	go func() {
		done <- m.client.Fetch(seqSet, []imap.FetchItem{imap.FetchEnvelope, section.FetchItem()}, messages)
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

	subject := msg.Envelope.Subject
	if !strings.HasPrefix(subject, m.cfg.PrismEndpointPrefix) {
		return nil
	}

	parts := strings.SplitN(subject, "]", 2)
	if len(parts) != 2 {
		return nil
	}

	endpoint := strings.TrimSpace(parts[0])
	endpoint = strings.TrimPrefix(endpoint, m.cfg.PrismEndpointPrefix)
	endpoint = m.cfg.EndpointPrefixProton + endpoint

	message := strings.TrimSpace(parts[1])

	notif := notification.Notification{
		Title:   m.cfg.ProtonPrismTopic,
		Message: message,
	}

	if err := m.dispatcher.Send(endpoint, notif); err != nil {
		m.logger.Error("Failed to send notification", "endpoint", endpoint, "error", err)
		return err
	}

	m.logger.Info("Processed Proton Mail notification", "endpoint", endpoint)
	return nil
}

func (m *Monitor) MarkAsRead(messageID string) error {
	if m.client == nil {
		return fmt.Errorf("not connected")
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(1)

	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{m.cfg.IMAPSeenFlag}

	return m.client.Store(seqSet, item, flags, nil)
}
