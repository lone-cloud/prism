package server

import (
	"encoding/json"
	"net/http"
	"time"

	"prism/service/util"
)

type healthResponse struct {
	Version  string             `json:"version"`
	Uptime   string             `json:"uptime"`
	Signal   *integrationHealth `json:"signal,omitempty"`
	Proton   *integrationHealth `json:"proton,omitempty"`
	Telegram *integrationHealth `json:"telegram,omitempty"`
}

type integrationHealth struct {
	Linked bool `json:"linked"`
}

func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(s.startTime)

	resp := healthResponse{
		Version: s.version,
		Uptime:  util.FormatUptime(uptime),
	}

	if s.integrations.Signal != nil && s.integrations.Signal.IsEnabled() {
		signalClient := s.integrations.Signal.GetHandlers().GetClient()
		account, _ := signalClient.GetLinkedAccount()
		resp.Signal = &integrationHealth{
			Linked: account != nil,
		}
	}

	if s.cfg.IsProtonEnabled() {
		resp.Proton = &integrationHealth{
			Linked: true,
		}
	}

	if s.cfg.IsTelegramEnabled() {
		resp.Telegram = &integrationHealth{
			Linked: true,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		s.logger.Error("Failed to encode health response", "error", err)
	}
}
