package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type healthResponse struct {
	Version  string             `json:"version"`
	Uptime   string             `json:"uptime"`
	Signal   *integrationHealth `json:"signal,omitempty"`
	Proton   *integrationHealth `json:"proton,omitempty"`
	Telegram *integrationHealth `json:"telegram,omitempty"`
}

type integrationHealth struct {
	Linked  bool   `json:"linked"`
	Account string `json:"account,omitempty"`
}

func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(s.startTime)

	resp := healthResponse{
		Version: s.version,
		Uptime:  formatUptime(uptime),
	}

	if s.integrations.Signal != nil && s.integrations.Signal.IsEnabled() {
		linked, account := s.integrations.Signal.Health()
		resp.Signal = &integrationHealth{Linked: linked, Account: account}
	}

	if s.integrations.Telegram != nil && s.integrations.Telegram.IsEnabled() {
		if linked, account := s.integrations.Telegram.Health(); linked {
			resp.Telegram = &integrationHealth{Linked: true, Account: account}
		}
	}

	if s.integrations.Proton != nil && s.integrations.Proton.IsEnabled() {
		if linked, account := s.integrations.Proton.Health(); linked {
			resp.Proton = &integrationHealth{Linked: true, Account: account}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		s.logger.Error("Failed to encode health response", "error", err)
	}
}

func formatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	return strings.Join(parts, " ")
}
