package server

import (
	"encoding/json"
	"net/http"
	"net/url"

	"prism/service/notification"

	"github.com/go-chi/chi/v5"
)

type registerWebPushRequest struct {
	AppName         string  `json:"appName"`
	PushEndpoint    string  `json:"pushEndpoint"`
	P256dh          *string `json:"p256dh,omitempty"`
	Auth            *string `json:"auth,omitempty"`
	VapidPrivateKey *string `json:"vapidPrivateKey,omitempty"`
}

func (s *Server) handleWebPushRegister(w http.ResponseWriter, r *http.Request) {
	var req registerWebPushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AppName == "" || req.PushEndpoint == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "appName and pushEndpoint are required"}) //nolint:errcheck
		return
	}

	if _, err := url.Parse(req.PushEndpoint); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid pushEndpoint URL"}) //nolint:errcheck
		return
	}

	existing, err := s.store.GetApp(req.AppName)
	if err != nil {
		s.logger.Error("Failed to check existing app", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var webPush *notification.WebPushSubscription
	if req.P256dh != nil && req.Auth != nil && req.VapidPrivateKey != nil {
		webPush = &notification.WebPushSubscription{
			Endpoint:        req.PushEndpoint,
			P256dh:          *req.P256dh,
			Auth:            *req.Auth,
			VapidPrivateKey: *req.VapidPrivateKey,
		}
	} else {
		webPush = &notification.WebPushSubscription{
			Endpoint: req.PushEndpoint,
		}
	}

	if existing != nil {
		if err := s.store.UpdateWebPush(req.AppName, webPush); err != nil {
			s.logger.Error("Failed to update webpush endpoint", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		s.logger.Info("Updated webpush for existing endpoint", "app", req.AppName, "pushEndpoint", req.PushEndpoint)
	} else {
		channel := notification.ChannelWebPush
		if err := s.store.Register(req.AppName, &channel, nil, webPush); err != nil {
			s.logger.Error("Failed to register webpush endpoint", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		s.logger.Info("Registered new webpush endpoint", "app", req.AppName, "pushEndpoint", req.PushEndpoint)
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"appName": req.AppName,
		"channel": "webpush",
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Failed to encode response", "error", err)
	}
}

func (s *Server) handleWebPushUnregister(w http.ResponseWriter, r *http.Request) {
	appName := chi.URLParam(r, "appName")
	if appName == "" {
		http.Error(w, "appName is required", http.StatusBadRequest)
		return
	}

	if err := s.store.ClearWebPush(appName); err != nil {
		s.logger.Error("Failed to clear webpush endpoint", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	s.logger.Info("Cleared webpush subscription", "app", appName)

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status":  "unregistered",
		"appName": appName,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Failed to encode response", "error", err)
	}
}
