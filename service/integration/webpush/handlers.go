package webpush

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"prism/service/subscription"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	store  *subscription.Store
	logger *slog.Logger
}

func NewHandlers(store *subscription.Store, logger *slog.Logger) *Handlers {
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
		util.JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AppName == "" || req.PushEndpoint == "" {
		util.JSONError(w, "appName and pushEndpoint are required", http.StatusBadRequest)
		return
	}

	encryptedFieldCount := 0
	if req.P256dh != nil {
		encryptedFieldCount++
	}
	if req.Auth != nil {
		encryptedFieldCount++
	}
	if req.VapidPrivateKey != nil {
		encryptedFieldCount++
	}
	if encryptedFieldCount > 0 && encryptedFieldCount < 3 {
		util.JSONError(w, "p256dh, auth, and vapidPrivateKey must all be provided together", http.StatusBadRequest)
		return
	}

	if err := validatePushEndpoint(req.PushEndpoint, encryptedFieldCount == 3); err != nil {
		util.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	var webPush *subscription.WebPushSubscription
	if req.P256dh != nil && req.Auth != nil && req.VapidPrivateKey != nil {
		normalizedP256dh, err := normalizeP256DH(*req.P256dh)
		if err != nil {
			util.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}

		normalizedAuth, err := normalizeAuthSecret(*req.Auth)
		if err != nil {
			util.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}

		normalizedKey, err := normalizeVAPIDPrivateKey(*req.VapidPrivateKey)
		if err != nil {
			util.JSONError(w, err.Error(), http.StatusBadRequest)
			return
		}

		webPush = &subscription.WebPushSubscription{
			Endpoint:        req.PushEndpoint,
			P256dh:          normalizedP256dh,
			Auth:            normalizedAuth,
			VapidPrivateKey: normalizedKey,
		}
	} else {
		webPush = &subscription.WebPushSubscription{
			Endpoint: req.PushEndpoint,
		}
	}

	sub := subscription.Subscription{
		AppName: req.AppName,
		Channel: subscription.ChannelWebPush,
		WebPush: webPush,
	}

	subID, err := h.store.AddSubscription(sub)
	if err != nil {
		h.logger.Warn("Failed to add webpush subscription", "app", req.AppName, "error", err)
		util.LogAndError(w, h.logger, "Failed to add subscription", http.StatusInternalServerError, err)
		return
	}

	h.logger.Info("Added webpush subscription", "app", req.AppName, "subscriptionID", subID, "pushEndpoint", req.PushEndpoint)

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"appName":        req.AppName,
		"channel":        subscription.ChannelWebPush.String(),
		"subscriptionId": subID,
	}
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handlers) HandleUnregister(w http.ResponseWriter, r *http.Request) {
	subscriptionID := chi.URLParam(r, "subscriptionId")
	if subscriptionID == "" {
		util.JSONError(w, "subscriptionId is required", http.StatusBadRequest)
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
