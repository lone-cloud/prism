package server

import (
	"net/http"

	"github.com/lone-cloud/prism/internal/notification"
)

func (s *Server) handleDeleteEndpointAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	endpoint := r.FormValue("endpoint")
	if endpoint == "" {
		http.Error(w, "Missing endpoint parameter", http.StatusBadRequest)
		return
	}

	if err := s.store.RemoveEndpoint(endpoint); err != nil {
		s.logger.Error("Failed to delete endpoint", "error", err)
		http.Error(w, "Failed to delete endpoint", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	s.handleFragmentEndpoints(w, r)
}

func (s *Server) handleToggleChannelAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	endpoint := r.FormValue("endpoint")
	channel := r.FormValue("channel")

	if endpoint == "" || channel == "" {
		http.Error(w, "Missing endpoint or channel parameter", http.StatusBadRequest)
		return
	}

	if channel != string(notification.ChannelSignal) && channel != string(notification.ChannelWebhook) {
		http.Error(w, "Invalid channel", http.StatusBadRequest)
		return
	}

	if err := s.store.UpdateChannel(endpoint, notification.Channel(channel)); err != nil {
		s.logger.Error("Failed to update channel", "error", err)
		http.Error(w, "Failed to update channel", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	s.handleFragmentEndpoints(w, r)
}
