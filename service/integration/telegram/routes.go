package telegram

import (
	"database/sql"
	"embed"
	"encoding/json"
	"log/slog"
	"net/http"

	"prism/service/credentials"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

//go:embed templates/*.html
var Templates embed.FS

type linkHandler struct {
	db       *sql.DB
	apiKey   string
	logger   *slog.Logger
	onUnlink func()
}

type telegramLinkRequest struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

func (h *linkHandler) handleLink(w http.ResponseWriter, r *http.Request) {
	var req telegramLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.BotToken == "" || req.ChatID == "" {
		util.JSONError(w, "bot_token and chat_id are required", http.StatusBadRequest)
		return
	}

	credStore, err := credentials.NewStore(h.db, h.apiKey)
	if err != nil {
		util.LogAndError(w, h.logger, "Failed to initialize credentials store", http.StatusInternalServerError, err)
		return
	}

	creds := &credentials.TelegramCredentials{
		BotToken: req.BotToken,
		ChatID:   req.ChatID,
	}

	if err := credStore.SaveTelegram(creds); err != nil {
		util.LogAndError(w, h.logger, "Failed to save Telegram credentials", http.StatusInternalServerError, err)
		return
	}

	util.SetToast(w, "Telegram linked", "success")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (h *linkHandler) handleUnlink(w http.ResponseWriter, r *http.Request) {
	credStore, err := credentials.NewStore(h.db, h.apiKey)
	if err != nil {
		util.LogAndError(w, h.logger, "Failed to initialize credentials store", http.StatusInternalServerError, err)
		return
	}

	if err := credStore.DeleteIntegration(credentials.IntegrationTelegram); err != nil {
		util.LogAndError(w, h.logger, "Failed to delete integration", http.StatusInternalServerError, err)
		return
	}

	h.onUnlink()
	util.SetToast(w, "Telegram unlinked", "success")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func RegisterRoutes(router *chi.Mux, handlers *Handlers, auth func(http.Handler) http.Handler, db *sql.DB, apiKey string, logger *slog.Logger, onUnlink func()) {
	if handlers == nil {
		return
	}

	handlers.DB = db
	handlers.APIKey = apiKey

	linkH := &linkHandler{db: db, apiKey: apiKey, logger: logger, onUnlink: onUnlink}

	router.With(auth).Get("/fragment/telegram", handlers.HandleFragment)
	router.With(auth).Post("/api/v1/telegram/link", linkH.handleLink)
	router.With(auth).Delete("/api/v1/telegram/link", linkH.handleUnlink)
}
