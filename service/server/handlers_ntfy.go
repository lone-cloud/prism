package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"prism/service/notification"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleNtfyPublish(w http.ResponseWriter, r *http.Request) {
	appName := chi.URLParam(r, "appName")
	if appName == "" {
		appName = chi.URLParam(r, "endpoint")
	}
	appName, _ = url.QueryUnescape(appName)
	if appName == "" || strings.Contains(appName, "/") {
		http.Error(w, "Invalid app name", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	var message, title string
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		var payload struct {
			Title   string `json:"title"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &payload); err == nil {
			message = payload.Message
			title = payload.Title
		} else {
			message = string(body)
		}
	} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		if values, err := url.ParseQuery(string(body)); err == nil {
			message = values.Get("message")
			if message == "" {
				message = string(body)
			}
			if title == "" {
				if t := values.Get("title"); t != "" {
					title = t
				} else if t := values.Get("t"); t != "" {
					title = t
				}
			}
		} else {
			message = string(body)
		}
	} else {
		message = string(body)
	}

	if title == "" {
		title = r.Header.Get("X-Title")
		if title == "" {
			title = r.Header.Get("Title")
		}
		if title == "" {
			title = r.Header.Get("t")
		}
	}

	if message == "" {
		http.Error(w, "Message required", http.StatusBadRequest)
		return
	}

	if title == appName {
		title = ""
	}

	notif := notification.Notification{
		Title:   title,
		Message: message,
	}

	if err := s.dispatcher.Send(appName, notif); err != nil {
		util.LogAndError(w, s.logger, "Failed to send notification", http.StatusInternalServerError, err, "app", appName)
		return
	}

	s.logger.Debug("Sent ntfy message", "app", appName, "preview", truncate(message, 50))

	now := time.Now()
	w.Header().Set("Content-Type", "application/json")
	response := map[string]any{
		"id":      now.UnixNano(),
		"time":    now.Unix(),
		"event":   "message",
		"topic":   appName,
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
