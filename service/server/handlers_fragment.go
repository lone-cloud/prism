package server

import (
	"fmt"
	"net/http"
	"net/url"

	"prism/service/notification"
	"prism/service/signal"
	"prism/service/util"
)

func (s *Server) handleFragmentHealth(w http.ResponseWriter, r *http.Request) {
	linked := s.getLinkedAccount()
	signalOk := s.signalDaemon.IsRunning()
	hasProton := s.cfg.IsProtonEnabled()

	w.Header().Set("Content-Type", "text/html")

	statusClass := "status-error"
	statusText := "Disconnected and Unlinked"
	var tooltip string
	if signalOk && linked != nil {
		statusClass = "status-ok"
		statusText = "Connected and Linked"
		tooltip = fmt.Sprintf(`<span class="tooltip">%s</span>`, util.FormatPhoneNumber(linked.Number))
	} else if signalOk {
		statusText = "Connected and Unlinked"
	}

	html := fmt.Sprintf(`<div class="status">
		<div class="status-item %s">Signal: %s%s</div>`, statusClass, statusText, tooltip)

	if hasProton {
		protonStatus := "Disconnected"
		protonClass := "status-error"
		protonTooltip := ""
		if s.protonMonitor.IsConnected() {
			protonStatus = "Connected"
			protonClass = "status-ok"
			protonTooltip = fmt.Sprintf(`<span class="tooltip">%s</span>`, s.cfg.ProtonIMAPUsername)
		}
		html += fmt.Sprintf(`
		<div class="status-item %s">Proton Mail: %s%s</div>`, protonClass, protonStatus, protonTooltip)
	}

	html += `</div>`

	currentLinked := linked != nil
	if s.lastSignalLinked == nil || *s.lastSignalLinked != currentLinked {
		html += `
	<div id="signal-info" hx-swap-oob="true">`
		html += s.getSignalInfoHTML()
		html += `</div>`
		s.lastSignalLinked = &currentLinked
	}

	// Check if apps changed
	mappings, err := s.store.GetAllMappings()
	if err == nil {
		currentAppsCount := len(mappings)
		if s.lastAppsCount == nil || *s.lastAppsCount != currentAppsCount {
			html += `
	<div id="apps-list" hx-swap-oob="true">`
			html += s.getAppsListHTML(mappings)
			html += `</div>`
			s.lastAppsCount = &currentAppsCount
		}
	}

	_, _ = fmt.Fprint(w, html) //nolint:errcheck // Error writing to ResponseWriter is handled by HTTP server
}

func (s *Server) handleFragmentSignalInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprint(w, s.getSignalInfoHTML()) //nolint:errcheck
}

func (s *Server) getSignalInfoHTML() string {
	if s.getLinkedAccount() != nil {
		return fmt.Sprintf(`<div id="signal-info"><details class="unlink-details">
			<summary class="unlink-summary">Unlink and remove device</summary>
			<div class="unlink-instructions">
				<ol>
					<li>Open Signal app → <strong>Settings → Linked Devices</strong></li>
					<li>Find <strong>"%s"</strong> and tap it</li>
					<li>Tap <strong>"Unlink Device"</strong></li>
				</ol>
			</div>
		</details></div>`, s.cfg.DeviceName)
	}

	return fmt.Sprintf(`<div id="signal-info" hx-get="/fragment/signal-info" hx-trigger="every 3s">%s</div>`, s.getQRCodeHTML())
}

func (s *Server) getLinkedAccount() *signal.AccountInfo {
	client := signal.NewClient(s.cfg.SignalCLISocketPath)
	account, _ := client.GetLinkedAccount() //nolint:errcheck // Nil account is valid, indicates not linked
	return account
}

func (s *Server) getQRCodeHTML() string {
	if s.getLinkedAccount() != nil {
		return `<p>Account already linked</p>`
	}

	qrCode, err := s.linkDevice.GenerateQR()
	if err != nil {
		s.logger.Error("Failed to generate QR code", "error", err)
		return `<p>Signal daemon is starting up, please refresh in a few seconds...</p>`
	}

	return fmt.Sprintf(`<p>Scan this QR code with your Signal app:</p>
	<p class="qr-instructions"><strong>Settings → Linked Devices → Link New Device</strong></p>
	<div class="qr-container">
		<img src="%s" class="qr-image" alt="QR Code" />
	</div>`, qrCode)
}

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
		return `<p>No apps registered</p>`
	}

	var html string
	html += `<ul class="app-list">`
	for _, m := range mappings {
		isSignal := m.Channel == notification.ChannelSignal
		isWebPush := m.Channel == notification.ChannelWebPush
		channelBadge := "Signal"
		channelTooltip := ""

		if isWebPush && m.WebPush != nil {
			if m.WebPush.HasEncryption() {
				channelBadge = "WebPush"
			} else {
				channelBadge = "Webhook"
			}
			channelTooltip = fmt.Sprintf(`<span class="tooltip">%s</span>`, m.WebPush.Endpoint)
		}

		if isSignal && m.Signal != nil && m.Signal.GroupID != "" {
			channelTooltip = fmt.Sprintf(`<span class="tooltip">Group ID: %s</span>`, m.Signal.GroupID)
		}

		itemHTML := fmt.Sprintf(`<li class="app-item">
			<div class="app-info">
				<div class="app-name"><strong>%s</strong></div>
				<div class="app-channel">
					<span class="channel-badge channel-%s">%s%s</span>`, m.AppName, m.Channel, channelBadge, channelTooltip)

		if m.WebPush != nil && isWebPush {
			if u, err := url.Parse(m.WebPush.Endpoint); err == nil {
				itemHTML += fmt.Sprintf(`<span class="app-detail">%s</span>`, u.Hostname())
			}
		}

		itemHTML += `</div></div><div class="app-actions">`

		if m.WebPush != nil {
			webpushLabel := "WebPush"
			if !m.WebPush.HasEncryption() {
				webpushLabel = "Webhook"
			}
			itemHTML += fmt.Sprintf(`<form style="display: inline;">
				<input type="hidden" name="app" value="%s" />
				<select class="channel-select" name="channel" hx-post="/action/toggle-channel" hx-target="#apps-list" hx-swap="innerHTML" hx-include="closest form">
					<option value="signal"%s>Signal</option>
					<option value="webpush"%s>%s</option>
				</select>
			</form>`, m.AppName, map[bool]string{true: " selected", false: ""}[isSignal], map[bool]string{true: " selected", false: ""}[isWebPush], webpushLabel)
		}

		itemHTML += fmt.Sprintf(`<button class="btn-delete" hx-delete="/action/delete-app/%s" hx-target="#apps-list" hx-swap="innerHTML">Delete</button></div></li>`, m.AppName)

		html += itemHTML
	}
	html += `</ul>`
	return html
}
