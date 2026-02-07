package proton

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"

	"prism/service/util"
)

type Handlers struct {
	monitor  *Monitor
	username string
	logger   *slog.Logger
	tmpl     *util.TemplateRenderer
}

type ProtonContentData struct {
	Connected bool
}

type IntegrationData struct {
	Name          string
	StatusClass   string
	StatusText    string
	StatusTooltip string
	Content       template.HTML
	Open          bool
	PollAttrs     string
}

func NewHandlers(monitor *Monitor, username string, logger *slog.Logger, tmpl *util.TemplateRenderer) *Handlers {
	return &Handlers{
		monitor:  monitor,
		username: username,
		logger:   logger,
		tmpl:     tmpl,
	}
}

func (h *Handlers) HandleFragment(w http.ResponseWriter, r *http.Request) {
	if h.monitor == nil {
		return
	}

	w.Header().Set("Content-Type", "text/html")

	var contentData ProtonContentData
	var integData IntegrationData
	integData.Name = "Proton Mail"

	if h.monitor.IsConnected() {
		integData.StatusClass = "connected"
		integData.StatusText = "Linked"
		integData.StatusTooltip = h.username
		integData.Open = false
		contentData.Connected = true
	} else {
		integData.StatusClass = "disconnected"
		integData.StatusText = "Unlinked"
		integData.Open = true
		contentData.Connected = false
	}

	content, err := h.tmpl.RenderHTML("proton-content.html", contentData)
	if err != nil {
		util.LogAndError(w, h.logger, "Internal server error", http.StatusInternalServerError, err)
		return
	}
	integData.Content = content

	html, err := h.tmpl.Render("integration.html", integData)
	if err != nil {
		util.LogAndError(w, h.logger, "Internal server error", http.StatusInternalServerError, err)
		return
	}

	w.Write([]byte(html))
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
		util.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.UID == 0 {
		util.JSONError(w, "uid (number) is required", http.StatusBadRequest)
		return
	}

	if h.monitor == nil {
		util.JSONError(w, "Proton integration not enabled", http.StatusBadRequest)
		return
	}

	if err := h.monitor.MarkAsRead(req.UID); err != nil {
		h.logger.Error("failed to mark email as read", "uid", req.UID, "error", err)
		util.JSONError(w, "failed to mark as read", http.StatusInternalServerError)
		return
	}

	h.logger.Info("marked email as read", "uid", req.UID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"success": true}); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}
