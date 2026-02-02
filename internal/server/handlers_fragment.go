package server

import (
	"fmt"
	"net/http"
	"net/url"

	"prism/internal/notification"
	"prism/internal/signal"
	"prism/internal/util"
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
		// TODO: check actual IMAP connection status with protonMonitor
		html += fmt.Sprintf(`
		<div class="status-item %s">Proton Mail: %s%s</div>`, protonClass, protonStatus, protonTooltip)
	}

	html += `</div>
	<div id="signal-info" hx-swap-oob="true">`
	html += s.getSignalInfoHTML()
	html += `</div>`

	_, _ = fmt.Fprint(w, html) //nolint:errcheck // Error writing to ResponseWriter is handled by HTTP server
}

func (s *Server) handleFragmentSignalInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprint(w, s.getSignalInfoHTML()) //nolint:errcheck // Error writing to ResponseWriter is handled by HTTP server
}

func (s *Server) getSignalInfoHTML() string {
	if s.getLinkedAccount() != nil {
		return fmt.Sprintf(`<details class="unlink-details">
			<summary class="unlink-summary">Unlink and remove device</summary>
			<div class="unlink-instructions">
				<ol>
					<li>Open Signal app → <strong>Settings → Linked Devices</strong></li>
					<li>Find <strong>"%s"</strong> and tap it</li>
					<li>Tap <strong>"Unlink Device"</strong></li>
				</ol>
			</div>
		</details>`, s.cfg.DeviceName)
	}
	return s.getQRCodeHTML()
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

	linkDevice := signal.NewLinkDevice(signal.NewClient(s.cfg.SignalCLISocketPath))
	qrCode, err := linkDevice.GenerateQR()
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

func (s *Server) handleFragmentEndpoints(w http.ResponseWriter, r *http.Request) {
	mappings, err := s.store.GetAllMappings()
	if err != nil {
		s.logger.Error("Failed to get mappings", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	if len(mappings) == 0 {
		_, _ = fmt.Fprint(w, `<p>No endpoints registered</p>`) //nolint:errcheck // Error writing to ResponseWriter is handled by HTTP server
		return
	}

	_, _ = fmt.Fprint(w, `<ul class="endpoint-list">`) //nolint:errcheck // Error writing to ResponseWriter is handled by HTTP server
	for _, m := range mappings {
		isSignal := m.Channel == notification.ChannelSignal
		isWebhook := m.Channel == notification.ChannelWebhook
		channelBadge := "Signal"
		if isWebhook {
			channelBadge = "Webhook"
		}

		html := fmt.Sprintf(`<li class="endpoint-item">
			<div class="endpoint-info">
				<div class="endpoint-name"><strong>%s</strong></div>
				<div class="endpoint-channel">
					<span class="channel-badge channel-%s">%s</span>`, m.AppName, m.Channel, channelBadge)

		if m.UpEndpoint != nil && isWebhook {
			// Extract hostname from webhook URL
			if u, err := url.Parse(*m.UpEndpoint); err == nil {
				html += fmt.Sprintf(`<span class="endpoint-detail">%s</span>`, u.Hostname())
			}
		}

		if m.GroupID != nil && isSignal {
			html += fmt.Sprintf(`<span class="endpoint-detail">%s</span>`, *m.GroupID)
		}

		html += `</div></div><div class="endpoint-actions">`

		if m.UpEndpoint != nil {
			html += fmt.Sprintf(`<form style="display: inline;">
				<input type="hidden" name="endpoint" value="%s" />
				<select class="channel-select" name="channel" hx-post="/action/toggle-channel" hx-target="#endpoints-list" hx-swap="innerHTML" hx-include="closest form">
					<option value="signal"%s>Signal</option>
					<option value="webhook"%s>Webhook</option>
				</select>
			</form>`, m.Endpoint, map[bool]string{true: " selected", false: ""}[isSignal], map[bool]string{true: " selected", false: ""}[isWebhook])
		}

		html += fmt.Sprintf(`<form style="display: inline;">
			<input type="hidden" name="endpoint" value="%s" />
			<button class="btn-delete" hx-delete="/action/delete-endpoint" hx-target="#endpoints-list" hx-swap="innerHTML" hx-include="closest form">Delete</button>
		</form></div></li>`, m.Endpoint)

		_, _ = fmt.Fprint(w, html) //nolint:errcheck // Error writing to ResponseWriter is handled by HTTP server
	}
	_, _ = fmt.Fprint(w, `</ul>`) //nolint:errcheck // Error writing to ResponseWriter is handled by HTTP server
}
