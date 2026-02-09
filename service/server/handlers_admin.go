package server

import (
	"encoding/json"
	"net/http"
	"time"

	"prism/service/notification"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleGetMappings(w http.ResponseWriter, r *http.Request) {
	mappings, err := s.store.GetAllMappings()
	if err != nil {
		util.LogAndError(w, s.logger, "Internal server error", http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(mappings); err != nil {
		s.logger.Error("Failed to encode response", "error", err)
	}
}

type createMappingRequest struct {
	App          string               `json:"app"`
	Channel      notification.Channel `json:"channel"`
	PushEndpoint *string              `json:"pushEndpoint,omitempty"`
}

func (s *Server) handleCreateMapping(w http.ResponseWriter, r *http.Request) {
	var req createMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.App == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if req.Channel == "" {
		req.Channel = notification.ChannelWebPush
	}

	if req.Channel == notification.ChannelWebPush && req.PushEndpoint == nil {
		http.Error(w, "pushEndpoint required for webpush channel", http.StatusBadRequest)
		return
	}

	var webPush *notification.WebPushSubscription
	if req.Channel == notification.ChannelWebPush && req.PushEndpoint != nil {
		webPush = &notification.WebPushSubscription{
			Endpoint: *req.PushEndpoint,
		}
	}

	if err := s.store.Register(req.App, &req.Channel, nil, webPush); err != nil {
		util.LogAndError(w, s.logger, "Failed to create mapping", http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "created"}); err != nil {
		s.logger.Error("Failed to encode response", "error", err)
	}
}

func (s *Server) handleDeleteMapping(w http.ResponseWriter, r *http.Request) {
	appName := chi.URLParam(r, "appName")
	if appName == "" {
		http.Error(w, "Missing appName parameter", http.StatusBadRequest)
		return
	}

	if err := s.store.RemoveApp(appName); err != nil {
		util.LogAndError(w, s.logger, "Failed to delete mapping", http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type updateChannelRequest struct {
	Channel    notification.Channel `json:"channel"`
	UpEndpoint *string              `json:"upEndpoint,omitempty"`
}

func (s *Server) handleUpdateChannel(w http.ResponseWriter, r *http.Request) {
	appName := chi.URLParam(r, "appName")
	if appName == "" {
		http.Error(w, "Missing appName parameter", http.StatusBadRequest)
		return
	}

	var req updateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Channel == notification.ChannelWebPush && req.UpEndpoint == nil {
		http.Error(w, "upEndpoint required for webpush channel", http.StatusBadRequest)
		return
	}

	if err := s.store.UpdateChannel(appName, req.Channel); err != nil {
		util.LogAndError(w, s.logger, "Failed to update channel", http.StatusInternalServerError, err)
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
		util.LogAndError(w, s.logger, "Failed to get mappings", http.StatusInternalServerError, err)
		return
	}

	uptime := time.Since(s.startTime)

	stats := map[string]any{
		"uptime":        util.FormatUptime(uptime),
		"uptimeSeconds": int(uptime.Seconds()),
		"mappingsCount": len(mappings),
		"signalCount":   countByChannel(mappings, notification.ChannelSignal),
		"webpushCount":  countByChannel(mappings, notification.ChannelWebPush),
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
