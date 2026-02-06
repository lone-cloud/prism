package proton

import (
	"embed"
	"log/slog"
	"net/http"

	"prism/service/config"
	"prism/service/notification"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

//go:embed templates/*.html
var templates embed.FS

func GetTemplates() embed.FS {
	return templates
}

func RegisterRoutes(router *chi.Mux, cfg *config.Config, dispatcher *notification.Dispatcher, logger *slog.Logger, authMiddleware func(http.Handler) http.Handler, tmpl *util.TemplateRenderer) *Handlers {
	if !cfg.IsProtonEnabled() {
		return nil
	}

	monitor := NewMonitor(cfg, dispatcher, logger)
	handlers := NewHandlers(monitor, cfg.ProtonIMAPUsername, logger, tmpl)

	router.With(authMiddleware).Get("/fragment/proton", handlers.HandleFragment)

	router.Route("/api/proton-mail", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/mark-read", handlers.HandleMarkRead)
	})

	return handlers
}
