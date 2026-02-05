package server

import (
	"fmt"
	"net/http"
	"net/url"

	"prism/service/notification"
)

func (s *Server) handleFragmentApps(w http.ResponseWriter, r *http.Request) {
	mappings, err := s.store.GetAllMappings()
	if err != nil {
		s.logger.Error("Failed to get mappings", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprint(w, s.getAppsListHTML(mappings)) //nolint:errcheck
}

func (s *Server) getAppsListHTML(mappings []notification.Mapping) string {
	if len(mappings) == 0 {
		return `<p>No apps registered yet. See <a href="https://github.com/lone-cloud/prism?tab=readme-ov-file#real-world-examples" target="_blank" rel="noopener noreferrer">real-world examples</a> to get started.</p>`
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

	var html string
	html += `<ul class="app-list">`
	for _, m := range mappings {
		isSignal := m.Channel == notification.ChannelSignal
		isWebPush := m.Channel == notification.ChannelWebPush
		isTelegram := m.Channel == notification.ChannelTelegram
		channelBadge := ""
		channelTooltip := ""

		if isSignal && m.Signal != nil && m.Signal.GroupID != "" {
			channelBadge = "Signal"
			channelTooltip = fmt.Sprintf(`<span class="tooltip">Group ID: %s</span>`, m.Signal.GroupID)
		} else if isWebPush && m.WebPush != nil && m.WebPush.Endpoint != "" {
			channelBadge = "WebPush"
			channelTooltip = fmt.Sprintf(`<span class="tooltip">%s</span>`, m.WebPush.Endpoint)
		} else if isTelegram && telegramConfigured {
			channelBadge = "Telegram"
		} else {
			channelBadge = "Not Configured"
			if isSignal {
				channelTooltip = `<span class="tooltip">No Signal group created yet. Will auto-create on first notification.</span>`
			} else if isTelegram {
				channelTooltip = `<span class="tooltip">Set TELEGRAM_CHAT_ID in .env</span>`
			} else {
				channelTooltip = `<span class="tooltip">No WebPush endpoint registered.</span>`
			}
		}

		itemHTML := fmt.Sprintf(`<li class="app-item">
			<div class="app-info">
				<div class="app-name"><strong>%s</strong></div>
				<div class="app-channel">`, m.AppName)

		if channelBadge != "Not Configured" {
			itemHTML += fmt.Sprintf(`<span class="channel-badge channel-%s">%s%s</span>`, m.Channel, channelBadge, channelTooltip)
		} else {
			itemHTML += fmt.Sprintf(`<span class="channel-badge" style="background: var(--error); color: var(--text-on-color);">%s%s</span>`, channelBadge, channelTooltip)
		}

		if m.WebPush != nil && isWebPush {
			if u, err := url.Parse(m.WebPush.Endpoint); err == nil {
				itemHTML += fmt.Sprintf(`<span class="app-detail">%s</span>`, u.Hostname())
			}
		}

		itemHTML += `</div></div><div class="app-actions">`

		var channelOptions string
		optionCount := 0

		if signalLinked {
			channelOptions += fmt.Sprintf(`<option value="signal"%s>Signal</option>`, map[bool]string{true: " selected", false: ""}[isSignal])
			optionCount++
		}
		if telegramConfigured {
			channelOptions += fmt.Sprintf(`<option value="telegram"%s>Telegram</option>`, map[bool]string{true: " selected", false: ""}[isTelegram])
			optionCount++
		}
		if m.WebPush != nil && m.WebPush.Endpoint != "" {
			channelOptions += fmt.Sprintf(`<option value="webpush"%s>WebPush</option>`, map[bool]string{true: " selected", false: ""}[isWebPush])
			optionCount++
		}

		if optionCount > 1 {
			itemHTML += fmt.Sprintf(`<form style="display: inline;">
				<input type="hidden" name="app" value="%s" />
				<select class="channel-select" name="channel" hx-post="/action/toggle-channel" hx-target="#apps-list" hx-swap="innerHTML" hx-include="closest form">
					%s
				</select>
			</form>`, m.AppName, channelOptions)
		}

		itemHTML += fmt.Sprintf(`<button class="btn-delete" hx-delete="/action/app/%s" hx-target="#apps-list" hx-swap="innerHTML">Delete</button></div></li>`, m.AppName)

		html += itemHTML
	}
	html += `</ul>`
	return html
}
