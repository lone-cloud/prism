package server

import (
	"net/http"

	"prism/service/notification"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleDeleteAppAction(w http.ResponseWriter, r *http.Request) {
	app := chi.URLParam(r, "appName")
	if app == "" {
		http.Error(w, "Missing app parameter", http.StatusBadRequest)
		return
	}

	if err := s.store.RemoveApp(app); err != nil {
		s.logger.Error("Failed to delete app", "error", err)
		http.Error(w, "Failed to delete app", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	s.handleFragmentApps(w, r)
}

func (s *Server) handleToggleChannelAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	app := r.FormValue("app")
	channel := r.FormValue("channel")

	if app == "" || channel == "" {
		http.Error(w, "Missing app or channel parameter", http.StatusBadRequest)
		return
	}

	if !s.dispatcher.IsValidChannel(notification.Channel(channel)) {
		http.Error(w, "Invalid channel", http.StatusBadRequest)
		return
	}

	if err := s.store.UpdateChannel(app, notification.Channel(channel)); err != nil {
		s.logger.Error("Failed to update channel", "error", err)
		http.Error(w, "Failed to update channel", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	s.handleFragmentApps(w, r)
}
