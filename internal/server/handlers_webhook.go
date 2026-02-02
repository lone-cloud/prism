package server

import (
	"encoding/json"
	"net/http"
	"net/url"

	"prism/internal/notification"
)

type registerWebhookRequest struct {
	AppName    string `json:"appName"`
	UpEndpoint string `json:"upEndpoint"`
}

func (s *Server) handleWebhookRegister(w http.ResponseWriter, r *http.Request) {
	var req registerWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AppName == "" || req.UpEndpoint == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "appName and upEndpoint are required"}) //nolint:errcheck // Error encoding response is not critical
		return
	}

	if _, err := url.Parse(req.UpEndpoint); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid upEndpoint URL"}) //nolint:errcheck // Error encoding response is not critical
		return
	}

	endpoint := s.cfg.EndpointPrefixUP + req.AppName

	if err := s.store.Register(endpoint, req.AppName, notification.ChannelWebhook, nil, &req.UpEndpoint); err != nil {
		s.logger.Error("Failed to register webhook endpoint", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Internal server error"}) //nolint:errcheck // Error encoding response is not critical
		return
	}

	s.logger.Debug("Registered webhook endpoint", "app", req.AppName, "upEndpoint", req.UpEndpoint)

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"endpoint": endpoint,
		"appName":  req.AppName,
		"channel":  "webhook",
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Failed to encode response", "error", err)
	}
}
