package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"prism/service/notification"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleNtfyPublish(w http.ResponseWriter, r *http.Request) {
	topic := chi.URLParam(r, "endpoint")
	if topic == "" {
		topic = chi.URLParam(r, "topic")
	}
	topic, _ = url.QueryUnescape(topic)
	if topic == "" || strings.Contains(topic, "/") {
		http.Error(w, "Invalid topic", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	message := string(body)
	if message == "" {
		http.Error(w, "Message required", http.StatusBadRequest)
		return
	}
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
			if m := values.Get("message"); m != "" {
				message = m
			}
			if title == "" {
				if t := values.Get("title"); t != "" {
					title = t
				} else if t := values.Get("t"); t != "" {
					title = t
				}
			}
		}
	}

	if title == topic {
		title = ""
	}

	notif := notification.Notification{
		Title:   title,
		Message: message,
	}

	if err := s.dispatcher.Send(topic, notif); err != nil {
		s.logger.Error("Failed to send notification", "app", topic, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	s.logger.Debug("Sent ntfy message", "topic", topic, "preview", truncate(message, 50))

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

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
