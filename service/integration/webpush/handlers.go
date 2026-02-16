package webpush

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"

	"prism/service/notification"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

func generateSubscriptionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

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
		util.JSONError(w, "appName and pushEndpoint are required", http.StatusBadRequest)
		return
	}

	if _, err := url.Parse(req.PushEndpoint); err != nil {
		util.JSONError(w, "Invalid pushEndpoint URL", http.StatusBadRequest)
		return
	}

	subID, err := generateSubscriptionID()
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

	sub := notification.Subscription{
		ID:      subID,
		AppName: req.AppName,
		Channel: notification.ChannelWebPush,
		WebPush: webPush,
	}

	if err := h.store.AddSubscription(sub); err != nil {
		util.LogAndError(w, h.logger, "Failed to add subscription", http.StatusInternalServerError, err)
		return
	}

	h.logger.Info("Added webpush subscription", "app", req.AppName, "subscriptionID", subID, "pushEndpoint", req.PushEndpoint)

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"appName":        req.AppName,
		"channel":        notification.ChannelWebPush.String(),
		"subscriptionId": subID,
	}
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handlers) HandleUnregister(w http.ResponseWriter, r *http.Request) {
	subscriptionID := chi.URLParam(r, "subscriptionId")
	if subscriptionID == "" {
		http.Error(w, "subscriptionId is required", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteSubscription(subscriptionID); err != nil {
		util.LogAndError(w, h.logger, "Failed to delete subscription", http.StatusInternalServerError, err)
		return
	}

	h.logger.Info("Deleted webpush subscription", "subscriptionID", subscriptionID)

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status":         "deleted",
		"subscriptionId": subscriptionID,
	}
	_ = json.NewEncoder(w).Encode(response)
}
