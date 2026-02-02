package proton

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"prism/internal/config"
	"prism/internal/notification"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type Monitor struct {
	cfg              *config.Config
	dispatcher       *notification.Dispatcher
	logger           *slog.Logger
	client           *client.Client
	monitorStartTime time.Time
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

func (m *Monitor) IsConnected() bool {
	return m.client != nil
}
