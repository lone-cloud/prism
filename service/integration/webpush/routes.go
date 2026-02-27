package webpush

import (
	"log/slog"
	"net/http"

	"prism/service/subscription"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router *chi.Mux, store *subscription.Store, logger *slog.Logger, authMiddleware func(http.Handler) http.Handler) {
	handlers := NewHandlers(store, logger)

	router.Route("/api/v1/webpush/subscriptions", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/", handlers.HandleRegister)
		r.Delete("/{subscriptionId}", handlers.HandleUnregister)
	})
}
