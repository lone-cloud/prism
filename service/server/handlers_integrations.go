package server

import (
	"fmt"
	"net/http"
)

func (s *Server) handleFragmentIntegrations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	html := ""

	if s.cfg.IsSignalEnabled() {
		html += `<div id="signal-integration" class="integration-container" hx-get="/fragment/signal" hx-trigger="load"><div class="loading"><div class="spinner"></div></div></div>`
	}

	if s.cfg.IsProtonEnabled() {
		html += `<div id="proton-integration" class="integration-container" hx-get="/fragment/proton" hx-trigger="load"><div class="loading"><div class="spinner"></div></div></div>`
	}

	if s.cfg.IsTelegramEnabled() {
		html += `<div id="telegram-integration" class="integration-container" hx-get="/fragment/telegram" hx-trigger="load"><div class="loading"><div class="spinner"></div></div></div>`
	}

	_, _ = fmt.Fprint(w, html)
}
