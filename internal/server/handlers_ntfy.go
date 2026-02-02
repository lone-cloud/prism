package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"prism/internal/notification"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleNtfyPublish(w http.ResponseWriter, r *http.Request) {
	topic := chi.URLParam(r, "endpoint")
	if topic == "" || strings.Contains(topic, "/") {
		http.Error(w, "Invalid topic", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "Message required", http.StatusBadRequest)
		return
	}

	message := string(body)
	title := extractTitle(r, message)

	if title == topic {
		title = ""
	}

	fullEndpoint := s.cfg.EndpointPrefixNtfy + topic

	notif := notification.Notification{
		Title:   title,
		Message: message,
	}

	if err := s.dispatcher.Send(fullEndpoint, notif); err != nil {
		s.logger.Error("Failed to send notification", "endpoint", fullEndpoint, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	s.logger.Debug("Sent ntfy message", "topic", topic, "preview", truncate(message, 50))

	// Return ntfy-compatible response
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"id":      time.Now().UnixNano(),
		"time":    time.Now().Unix(),
		"event":   "message",
		"topic":   topic,
		"message": message,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Failed to encode response", "error", err)
	}
}

func extractTitle(r *http.Request, message string) string {
	// Extract title from headers (ntfy supports multiple headers)
	title := r.Header.Get("X-Title")
	if title == "" {
		title = r.Header.Get("Title")
	}
	if title == "" {
		title = r.Header.Get("t")
	}

	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		if values, err := url.ParseQuery(message); err == nil {
			if title == "" {
				if t := values.Get("title"); t != "" {
					title = t
				} else if t := values.Get("t"); t != "" {
					title = t
				}
			}
		}
	}

	return title
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
