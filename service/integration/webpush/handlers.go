package webpush

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"

	"prism/service/notification"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	store  *notification.Store
	logger *slog.Logger
}

func NewHandlers(store *notification.Store, logger *slog.Logger) *Handlers {
	return &Handlers{
		store:  store,
		logger: logger,
	}
}

type registerRequest struct {
	AppName         string  `json:"appName"`
	PushEndpoint    string  `json:"pushEndpoint"`
	P256dh          *string `json:"p256dh,omitempty"`
	Auth            *string `json:"auth,omitempty"`
	VapidPrivateKey *string `json:"vapidPrivateKey,omitempty"`
}

func (h *Handlers) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AppName == "" || req.PushEndpoint == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "appName and pushEndpoint are required"})
		return
	}

	if _, err := url.Parse(req.PushEndpoint); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid pushEndpoint URL"})
		return
	}

	existing, err := h.store.GetApp(req.AppName)
	if err != nil {
		util.LogAndError(w, h.logger, "Internal server error", http.StatusInternalServerError, err)
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
		if err := h.store.UpdateWebPush(req.AppName, webPush); err != nil {
			util.LogAndError(w, h.logger, "Internal server error", http.StatusInternalServerError, err)
			return
		}
		h.logger.Info("Updated webpush for existing endpoint", "app", req.AppName, "pushEndpoint", req.PushEndpoint)
	} else {
		channel := notification.ChannelWebPush
		if err := h.store.Register(req.AppName, &channel, nil, webPush); err != nil {
			h.logger.Error("Failed to register webpush endpoint", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		h.logger.Info("Registered new webpush endpoint", "app", req.AppName, "pushEndpoint", req.PushEndpoint)
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"appName": req.AppName,
		"channel": notification.ChannelWebPush.String(),
	}
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handlers) HandleUnregister(w http.ResponseWriter, r *http.Request) {
	appName := chi.URLParam(r, "appName")
	if appName == "" {
		http.Error(w, "appName is required", http.StatusBadRequest)
		return
	}

	if err := h.store.ClearWebPush(appName); err != nil {
		h.logger.Error("Failed to clear webpush endpoint", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Cleared webpush subscription", "app", appName)

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status":  "unregistered",
		"appName": appName,
	}
	_ = json.NewEncoder(w).Encode(response)
}
