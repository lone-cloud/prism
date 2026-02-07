package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"

	"prism/service/notification"
	"prism/service/util"
)

func (s *Server) handleFragmentApps(w http.ResponseWriter, r *http.Request) {
	mappings, err := s.store.GetAllMappings()
	if err != nil {
		util.LogAndError(w, s.logger, "Internal server error", http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	var buf bytes.Buffer
	if err := s.fragmentTmpl.ExecuteTemplate(&buf, "app-list.html", s.buildAppListData(mappings)); err != nil {
		util.LogAndError(w, s.logger, "Failed to execute template", http.StatusInternalServerError, err)
		return
	}

	_, _ = w.Write(buf.Bytes())
}

func (s *Server) buildAppListData(mappings []notification.Mapping) []AppListItem {
	if len(mappings) == 0 {
		return nil
	}

	signalLinked := false
	if s.integrations.Signal != nil {
		handlers := s.integrations.Signal.GetHandlers()
		if handlers != nil {
			account, _ := handlers.GetClient().GetLinkedAccount()
			signalLinked = account != nil
		}
	}
	telegramConfigured := s.cfg.IsTelegramEnabled() && s.cfg.TelegramChatID != 0

	items := make([]AppListItem, 0, len(mappings))
	for _, m := range mappings {
		item := s.buildAppListItem(m, signalLinked, telegramConfigured)
		items = append(items, item)
	}
	return items
}

func (s *Server) buildAppListItem(m notification.Mapping, signalLinked, telegramConfigured bool) AppListItem {
	isSignal := m.Channel == notification.ChannelSignal
	isWebPush := m.Channel == notification.ChannelWebPush
	isTelegram := m.Channel == notification.ChannelTelegram

	item := AppListItem{
		AppName: m.AppName,
		Channel: string(m.Channel),
	}

	if isSignal && m.Signal != nil && m.Signal.GroupID != "" {
		item.ChannelBadge = m.Channel.Label()
		item.Tooltip = fmt.Sprintf("Group ID: %s", m.Signal.GroupID)
		item.ChannelConfigured = true
	} else if isWebPush && m.WebPush != nil && m.WebPush.Endpoint != "" {
		item.ChannelBadge = m.Channel.Label()
		item.Tooltip = m.WebPush.Endpoint
		item.ChannelConfigured = true
		if u, err := url.Parse(m.WebPush.Endpoint); err == nil {
			item.Hostname = u.Hostname()
		}
	} else if isTelegram && telegramConfigured {
		item.ChannelBadge = m.Channel.Label()
		item.ChannelConfigured = true
	} else {
		item.ChannelBadge = "Not Configured"
		item.ChannelConfigured = false
		if isSignal {
			item.Tooltip = "No Signal group created yet. Will auto-create on first notification."
		} else if isTelegram {
			item.Tooltip = "Set TELEGRAM_CHAT_ID in .env"
		} else {
			item.Tooltip = "No WebPush endpoint registered."
		}
	}

	item.ChannelOptions = s.buildChannelOptions(m, signalLinked, telegramConfigured)
	return item
}

func (s *Server) buildChannelOptions(m notification.Mapping, signalLinked, telegramConfigured bool) []SelectOption {
	isSignal := m.Channel == notification.ChannelSignal
	isWebPush := m.Channel == notification.ChannelWebPush
	isTelegram := m.Channel == notification.ChannelTelegram

	var options []SelectOption

	if signalLinked {
		options = append(options, SelectOption{
			Value:    notification.ChannelSignal.String(),
			Label:    notification.ChannelSignal.Label(),
			Selected: isSignal,
		})
	}
	if telegramConfigured {
		options = append(options, SelectOption{
			Value:    notification.ChannelTelegram.String(),
			Label:    notification.ChannelTelegram.Label(),
			Selected: isTelegram,
		})
	}
	if m.WebPush != nil && m.WebPush.Endpoint != "" {
		options = append(options, SelectOption{
			Value:    notification.ChannelWebPush.String(),
			Label:    notification.ChannelWebPush.Label(),
			Selected: isWebPush,
		})
	}

	return options
}
