package signal

import (
	"fmt"
	"net/http"
)

type Handlers struct {
	client     *Client
	linkDevice *LinkDevice
}

func NewHandlers(client *Client, linkDevice *LinkDevice) *Handlers {
	return &Handlers{
		client:     client,
		linkDevice: linkDevice,
	}
}

func (h *Handlers) HandleFragment(w http.ResponseWriter, r *http.Request) {
	if h.client == nil {
		return // Signal not enabled
	}

	w.Header().Set("Content-Type", "text/html")

	account, _ := h.client.GetLinkedAccount()
	var content string
	var statusBadge string
	var openAttr string
	var pollAttrs string

	if account != nil {
		statusBadge = fmt.Sprintf(`<span class="integration-status connected">Linked<span class="tooltip">%s</span></span>`, FormatPhoneNumber(account.Number))
		content = fmt.Sprintf(`
			<p><strong>Unlink Instructions:</strong></p>
			<ol class="link-instructions">
				<li>Open Signal on your phone</li>
				<li>Go to Settings → Linked Devices</li>
				<li>Find and remove <strong>%s</strong></li>
			</ol>
		`, h.linkDevice.deviceName)
		openAttr = ""
		pollAttrs = ""
	} else {
		statusBadge = `<span class="integration-status unlinked">Unlinked</span>`
		qrCode, err := h.linkDevice.GenerateQR()
		if err != nil {
			content = fmt.Sprintf(`<p>Error generating QR code: %s</p>`, err)
		} else {
			content = fmt.Sprintf(`
				<p><strong>Link your Signal (or <a href="https://molly.im" target="_blank" rel="noopener">Molly</a>) account:</strong></p>
				<ol class="link-instructions">
					<li>Open Signal on your phone</li>
					<li>Go to Settings → Linked Devices</li>
					<li>Scan the QR code below</li>
				</ol>
				<div class="qr-code-container">
					<img src="%s" alt="Signal QR Code" class="qr-code"/>
				</div>
			`, qrCode)
		}
		openAttr = " open"
		pollAttrs = ` hx-get="/fragment/signal" hx-trigger="every 3s" hx-swap="outerHTML"`
	}

	html := fmt.Sprintf(`<details class="integration-card"%s%s>
		<summary class="integration-header">
			<span class="integration-name">Signal</span>
			%s
		</summary>
		<div class="integration-content">%s</div>
	</details>`, openAttr, pollAttrs, statusBadge, content)

	_, _ = fmt.Fprint(w, html)
}

func (h *Handlers) IsEnabled() bool {
	return h.client != nil
}

func (h *Handlers) GetClient() *Client {
	return h.client
}
