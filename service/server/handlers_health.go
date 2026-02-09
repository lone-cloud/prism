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
		Uptime:  util.FormatUptime(uptime),
	}

	if s.integrations.Signal != nil && s.integrations.Signal.IsEnabled() {
		signalClient := s.integrations.Signal.GetHandlers().GetClient()
		account, _ := signalClient.GetLinkedAccount()
		if account != nil {
			resp.Signal = &integrationHealth{
				Linked:  true,
				Account: account.Number,
			}
		} else {
			resp.Signal = &integrationHealth{
				Linked: false,
			}
		}
	}

	if s.integrations.Telegram != nil && s.integrations.Telegram.IsEnabled() {
		telegramClient := s.integrations.Telegram.GetHandlers().GetClient()
		if telegramClient != nil {
			bot, err := telegramClient.GetMe()
			if err == nil {
				resp.Telegram = &integrationHealth{
					Linked:  true,
					Account: "@" + bot.Username,
				}
			}
		}
	}

	if s.integrations.Proton != nil && s.integrations.Proton.IsEnabled() {
		protonHandlers := s.integrations.Proton.GetHandlers()
		if protonHandlers != nil && protonHandlers.IsEnabled() {
			email, hasCredentials := protonHandlers.LoadFreshCredentials()
			if hasCredentials {
				resp.Proton = &integrationHealth{
					Linked:  true,
					Account: email,
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		s.logger.Error("Failed to encode health response", "error", err)
	}
}
