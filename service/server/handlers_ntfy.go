package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"prism/service/delivery"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleNtfyPublish(w http.ResponseWriter, r *http.Request) {
	appName := chi.URLParam(r, "appName")
	decodedAppName, err := url.PathUnescape(appName)
	if err != nil {
		util.JSONError(w, "Invalid app name", http.StatusBadRequest)
		return
	}
	appName = decodedAppName
	if appName == "" || strings.Contains(appName, "/") {
		util.JSONError(w, "Invalid app name", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		util.JSONError(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	message, title, imageURL := parseNtfyPayload(r, body)

	if message == "" {
		util.JSONError(w, "Message required", http.StatusBadRequest)
		return
	}

	if title == appName {
		title = ""
	}

	notif := delivery.Notification{
		Title:    title,
		Message:  message,
		ImageURL: imageURL,
	}

	if err := s.publisher.Publish(appName, notif); err != nil {
		util.LogAndError(w, s.logger, "Failed to send notification", http.StatusInternalServerError, err, "app", appName)
		return
	}

	s.logger.Debug("Sent ntfy message", "app", appName, "title", title, "message", message, "image", imageURL)

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

func parseNtfyPayload(r *http.Request, body []byte) (string, string, string) {
	var message, title, imageURL string

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		mediaType = ""
	}

	switch mediaType {
	case "application/json":
		var payload struct {
			Title   string `json:"title"`
			Message string `json:"message"`
			Attach  string `json:"attach"`
			Image   string `json:"image"`
		}
		if err := json.Unmarshal(body, &payload); err == nil {
			message = payload.Message
			title = payload.Title
			imageURL = firstNonEmpty(payload.Attach, payload.Image)
		} else {
			message = string(body)
		}
	case "application/x-www-form-urlencoded":
		r.Body = io.NopCloser(bytes.NewReader(body))
		if err := r.ParseForm(); err == nil {
			message = r.PostForm.Get("message")
			if message == "" {
				message = string(body)
			}
			title = firstNonEmpty(r.PostForm.Get("title"), r.PostForm.Get("t"))
		} else {
			message = string(body)
		}
	default:
		message = string(body)
	}

	if title == "" {
		title = firstNonEmpty(r.Header.Get("X-Title"), r.Header.Get("Title"), r.Header.Get("t"))
	}

	if imageURL == "" {
		imageURL = firstNonEmpty(r.Header.Get("X-Attach"), r.Header.Get("Attach"))
	}

	return message, title, imageURL
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
