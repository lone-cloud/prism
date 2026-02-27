package proton

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"

	"prism/service/credentials"
	"prism/service/util"
)

type Handlers struct {
	monitor  *Monitor
	username string
	logger   *slog.Logger
	tmpl     *util.TemplateRenderer
	DB       *sql.DB
	APIKey   string
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
}

func NewHandlers(monitor *Monitor, username string, logger *slog.Logger, tmpl *util.TemplateRenderer) *Handlers {
	return &Handlers{
		monitor:  monitor,
		username: username,
		logger:   logger,
		tmpl:     tmpl,
		DB:       nil,
		APIKey:   "",
	}
}

func (h *Handlers) LoadFreshCredentials() (string, bool) {
	if h.DB == nil || h.APIKey == "" {
		return "", false
	}

	credStore, err := credentials.NewStore(h.DB, h.APIKey)
	if err != nil {
		return "", false
	}

	creds, err := credStore.GetProton()
	if err != nil || creds == nil {
		return "", false
	}

	return creds.Email, true
}

func (h *Handlers) HandleFragment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	email, hasCredentials := h.LoadFreshCredentials()

	var contentData ProtonContentData
	var integData IntegrationData
	integData.Name = "Proton Mail"

	if !hasCredentials {
		integData.StatusClass = "disconnected"
		integData.StatusText = "Unlinked"
		integData.StatusTooltip = "Enter credentials to link"
		integData.Open = true
	} else if h.monitor == nil || !h.monitor.IsConnected() {
		integData.StatusClass = "disconnected"
		integData.StatusText = "Connecting…"
		integData.StatusTooltip = email
		integData.Open = false
		contentData.Connected = true
	} else {
		integData.StatusClass = "connected"
		integData.StatusText = "Linked"
		integData.StatusTooltip = email
		integData.Open = false
		contentData.Connected = true
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

type markReadRequest struct {
	UID string `json:"uid"`
}

func (h *Handlers) HandleMarkRead(w http.ResponseWriter, r *http.Request) {
	var req markReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.UID == "" {
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

type archiveRequest struct {
	UID string `json:"uid"`
}

func (h *Handlers) HandleArchive(w http.ResponseWriter, r *http.Request) {
	var req archiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.UID == "" {
		util.JSONError(w, "uid is required", http.StatusBadRequest)
		return
	}

	if h.monitor == nil {
		util.JSONError(w, "Proton integration not enabled", http.StatusBadRequest)
		return
	}

	if err := h.monitor.Archive(req.UID); err != nil {
		h.logger.Error("failed to archive email", "uid", req.UID, "error", err)
		util.JSONError(w, "failed to archive", http.StatusInternalServerError)
		return
	}

	h.logger.Info("archived email", "uid", req.UID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"success": true}); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

type deleteRequest struct {
	UID string `json:"uid"`
}

func (h *Handlers) HandleDelete(w http.ResponseWriter, r *http.Request) {
	var req deleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.UID == "" {
		util.JSONError(w, "uid is required", http.StatusBadRequest)
		return
	}

	if h.monitor == nil {
		util.JSONError(w, "Proton integration not enabled", http.StatusBadRequest)
		return
	}

	if err := h.monitor.Delete(req.UID); err != nil {
		h.logger.Error("failed to delete email", "uid", req.UID, "error", err)
		util.JSONError(w, "failed to delete", http.StatusInternalServerError)
		return
	}

	h.logger.Info("deleted email", "uid", req.UID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"success": true}); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}
