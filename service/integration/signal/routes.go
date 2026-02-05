package signal

import (
	"net/http"

	"prism/service/config"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router *chi.Mux, cfg *config.Config, authMiddleware func(http.Handler) http.Handler) *Handlers {
	if !cfg.IsSignalEnabled() {
		return nil
	}

	client := NewClient(cfg.SignalSocket)
	linkDevice := NewLinkDevice(client, cfg.DeviceName)
	handlers := NewHandlers(client, linkDevice)

	router.With(authMiddleware).Get("/fragment/signal", handlers.HandleFragment)

	return handlers
}
