package proton

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type Handlers struct {
	monitor  *Monitor
	username string
	logger   *slog.Logger
}

func NewHandlers(monitor *Monitor, username string, logger *slog.Logger) *Handlers {
	return &Handlers{
		monitor:  monitor,
		username: username,
		logger:   logger,
	}
}

func (h *Handlers) HandleFragment(w http.ResponseWriter, r *http.Request) {
	if h.monitor == nil {
		return // Proton not enabled
	}

	w.Header().Set("Content-Type", "text/html")

	statusBadge := `<span class="integration-status disconnected">Unlinked</span>`
	var content string
	var openAttr string

	if h.monitor.IsConnected() {
		statusBadge = fmt.Sprintf(`<span class="integration-status connected">Linked<span class="tooltip">%s</span></span>`, h.username)
		content = `
			<p><strong>Unlink Instructions:</strong></p>
			<ol class="link-instructions">
				<li>Run: <code>docker compose run protonmail-bridge init</code></li>
				<li>In the bridge CLI, use: <code>logout</code></li>
				<li>Then: <code>exit</code> to close the bridge</li>
			</ol>
		`
		openAttr = ""
	} else {
		content = `
			<p><strong>Setup Instructions:</strong></p>
			<ol class="link-instructions">
				<li>Run: <code>docker compose run --rm protonmail-bridge init</code></li>
				<li>At the prompt, use: <code>login</code> and enter your Proton Mail credentials</li>
				<li>Run: <code>info</code> to get your IMAP username and password</li>
				<li>Add credentials to <code>.env</code> and restart</li>
			</ol>
			<p class="text-muted">See <a href="https://github.com/lone-cloud/prism#proton-mail" target="_blank">full setup guide</a></p>
		`
		openAttr = " open"
	}

	html := fmt.Sprintf(`<details class="integration-card"%s>
		<summary class="integration-header">
			<span class="integration-name">Proton Mail</span>
			%s
		</summary>
		<div class="integration-content">%s</div>
	</details>`, openAttr, statusBadge, content)

	_, _ = fmt.Fprint(w, html)
}

func (h *Handlers) IsEnabled() bool {
	return h.monitor != nil
}

func (h *Handlers) GetMonitor() *Monitor {
	return h.monitor
}

type markReadRequest struct {
	UID uint32 `json:"uid"`
}

func (h *Handlers) HandleMarkRead(w http.ResponseWriter, r *http.Request) {
	var req markReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if req.UID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "uid (number) is required"})
		return
	}

	if h.monitor == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Proton integration not enabled"})
		return
	}

	if err := h.monitor.MarkAsRead(req.UID); err != nil {
		h.logger.Error("failed to mark email as read", "uid", req.UID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to mark as read"})
		return
	}

	h.logger.Info("marked email as read", "uid", req.UID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"success": true}); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}
