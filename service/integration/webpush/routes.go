package webpush

import (
	"log/slog"
	"net/http"

	"prism/service/notification"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router *chi.Mux, store *notification.Store, logger *slog.Logger, authMiddleware func(http.Handler) http.Handler) {
	handlers := NewHandlers(store, logger)

	router.Route("/webpush/app", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/", handlers.HandleRegister)
		r.Delete("/{appName}", handlers.HandleUnregister)
	})
}
