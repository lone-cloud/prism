package proton

import (
	"log/slog"
	"net/http"

	"prism/service/config"
	"prism/service/notification"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router *chi.Mux, cfg *config.Config, dispatcher *notification.Dispatcher, logger *slog.Logger, authMiddleware func(http.Handler) http.Handler) *Handlers {
	if !cfg.IsProtonEnabled() {
		return nil
	}

	monitor := NewMonitor(cfg, dispatcher, logger)
	handlers := NewHandlers(monitor, cfg.ProtonIMAPUsername, logger)

	router.With(authMiddleware).Get("/fragment/proton", handlers.HandleFragment)

	router.Route("/api/proton-mail", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/mark-read", handlers.HandleMarkRead)
	})

	return handlers
}
