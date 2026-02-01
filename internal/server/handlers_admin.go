package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lone-cloud/prism/internal/notification"
	"github.com/lone-cloud/prism/internal/util"
)

func (s *Server) handleGetMappings(w http.ResponseWriter, r *http.Request) {
	mappings, err := s.store.GetAllMappings()
	if err != nil {
		s.logger.Error("Failed to get mappings", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(mappings); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

type createMappingRequest struct {
	Endpoint   string               `json:"endpoint"`
	AppName    string               `json:"appName"`
	Channel    notification.Channel `json:"channel"`
	UpEndpoint *string              `json:"upEndpoint,omitempty"`
}

func (s *Server) handleCreateMapping(w http.ResponseWriter, r *http.Request) {
	var req createMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Endpoint == "" || req.AppName == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if req.Channel == "" {
		req.Channel = notification.ChannelSignal
	}

	if req.Channel == notification.ChannelWebhook && req.UpEndpoint == nil {
		http.Error(w, "upEndpoint required for webhook channel", http.StatusBadRequest)
		return
	}

	if err := s.store.Register(req.Endpoint, req.AppName, req.Channel, nil, req.UpEndpoint); err != nil {
		s.logger.Error("Failed to register mapping", "error", err)
		http.Error(w, "Failed to create mapping", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "created"}); err != nil {
		s.logger.Error("Failed to encode response", "error", err)
	}
}

func (s *Server) handleDeleteMapping(w http.ResponseWriter, r *http.Request) {
	endpoint := chi.URLParam(r, "endpoint")
	if endpoint == "" {
		http.Error(w, "Missing endpoint parameter", http.StatusBadRequest)
		return
	}

	if err := s.store.RemoveEndpoint(endpoint); err != nil {
		s.logger.Error("Failed to delete mapping", "error", err)
		http.Error(w, "Failed to delete mapping", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type updateChannelRequest struct {
	Channel    notification.Channel `json:"channel"`
	UpEndpoint *string              `json:"upEndpoint,omitempty"`
}

func (s *Server) handleUpdateChannel(w http.ResponseWriter, r *http.Request) {
	endpoint := chi.URLParam(r, "endpoint")
	if endpoint == "" {
		http.Error(w, "Missing endpoint parameter", http.StatusBadRequest)
		return
	}

	var req updateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Channel == notification.ChannelWebhook && req.UpEndpoint == nil {
		http.Error(w, "upEndpoint required for webhook channel", http.StatusBadRequest)
		return
	}

	if err := s.store.UpdateChannel(endpoint, req.Channel); err != nil {
		s.logger.Error("Failed to update channel", "error", err)
		http.Error(w, "Failed to update channel", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "updated"}); err != nil {
		s.logger.Error("Failed to encode response", "error", err)
	}
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	mappings, err := s.store.GetAllMappings()
	if err != nil {
		s.logger.Error("Failed to get mappings", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	uptime := time.Since(s.startTime)

	stats := map[string]interface{}{
		"uptime":        util.FormatUptime(uptime),
		"uptimeSeconds": int(uptime.Seconds()),
		"mappingsCount": len(mappings),
		"signalCount":   countByChannel(mappings, notification.ChannelSignal),
		"webhookCount":  countByChannel(mappings, notification.ChannelWebhook),
		"protonEnabled": s.cfg.IsProtonEnabled(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		s.logger.Error("Failed to encode response", "error", err)
	}
}

func countByChannel(mappings []notification.Mapping, channel notification.Channel) int {
	count := 0
	for _, m := range mappings {
		if m.Channel == channel {
			count++
		}
	}
	return count
}
