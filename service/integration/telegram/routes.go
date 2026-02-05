package telegram

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router *chi.Mux, handlers *Handlers, auth func(http.Handler) http.Handler) {
	if handlers == nil {
		return
	}

	router.With(auth).Get("/fragment/telegram", handlers.HandleFragment)
}
