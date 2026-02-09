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

func RegisterRoutes(router *chi.Mux, cfg *config.Config, authMiddleware func(http.Handler) http.Handler, tmpl *util.TemplateRenderer, logger *slog.Logger, client *Client) *Handlers {
	handlers := NewHandlers(client, tmpl, logger)

	router.With(authMiddleware).Get("/fragment/signal", handlers.HandleFragment)
	router.With(authMiddleware).Post("/api/signal/link", handlers.HandleLinkDevice)
	router.With(authMiddleware).Get("/api/signal/status", handlers.HandleLinkStatus)

	return handlers
}
