package server

import (
	"encoding/json"
	"net/http"

	"prism/service/notification"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleDeleteApp(w http.ResponseWriter, r *http.Request) {
	app := chi.URLParam(r, "appName")
	if app == "" {
		http.Error(w, "Missing app parameter", http.StatusBadRequest)
		return
	}

	if err := s.store.RemoveApp(app); err != nil {
		util.LogAndError(w, s.logger, "Failed to delete app", http.StatusInternalServerError, err)
		return
	}

	util.SetToast(w, "App deleted", "success")
	s.handleFragmentApps(w, r)
}

func (s *Server) handleGetApps(w http.ResponseWriter, r *http.Request) {
	apps, err := s.store.GetAllApps()
	if err != nil {
		util.LogAndError(w, s.logger, "Failed to get apps", http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sanitizeApps(apps))
}

func sanitizeApps(apps []notification.App) []notification.App {
	sanitized := make([]notification.App, len(apps))
	for i, app := range apps {
		sanitized[i] = notification.App{
			AppName:       app.AppName,
			Subscriptions: make([]notification.Subscription, len(app.Subscriptions)),
		}
		for j, sub := range app.Subscriptions {
			sanitized[i].Subscriptions[j] = sub
			if sub.WebPush != nil {
				sanitized[i].Subscriptions[j].WebPush = &notification.WebPushSubscription{
					Endpoint: sub.WebPush.Endpoint,
				}
			}
		}
	}
	return sanitized
}
