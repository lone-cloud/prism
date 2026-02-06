package telegram

import (
	"embed"
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:embed templates/*.html
var templates embed.FS

func GetTemplates() embed.FS {
	return templates
}

func RegisterRoutes(router *chi.Mux, handlers *Handlers, auth func(http.Handler) http.Handler) {
	if handlers == nil {
		return
	}

	router.With(auth).Get("/fragment/telegram", handlers.HandleFragment)
}
