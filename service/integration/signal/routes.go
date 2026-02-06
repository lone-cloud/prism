package signal

import (
	"embed"
	"log/slog"
	"net/http"

	"prism/service/config"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

//go:embed templates/*.html
var templates embed.FS

func GetTemplates() embed.FS {
	return templates
}

func RegisterRoutes(router *chi.Mux, cfg *config.Config, authMiddleware func(http.Handler) http.Handler, tmpl *util.TemplateRenderer, logger *slog.Logger) *Handlers {
	if !cfg.IsSignalEnabled() {
		return nil
	}

	client := NewClient(cfg.SignalSocket)
	linkDevice := NewLinkDevice(client, cfg.DeviceName)
	handlers := NewHandlers(client, linkDevice, tmpl, logger)

	router.With(authMiddleware).Get("/fragment/signal", handlers.HandleFragment)

	return handlers
}
