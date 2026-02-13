package proton

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"log/slog"
	"net/http"

	"prism/service/credentials"
	"prism/service/notification"
	"prism/service/util"

	"github.com/emersion/hydroxide/protonmail"
	"github.com/go-chi/chi/v5"
)

//go:embed templates/*.html
var templates embed.FS

func GetTemplates() embed.FS {
	return templates
}

type authHandler struct {
	db          *sql.DB
	apiKey      string
	logger      *slog.Logger
	integration *Integration
	dispatcher  *notification.Dispatcher
}

type protonAuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TOTP     string `json:"totp"`
}

func (h *authHandler) handleAuth(w http.ResponseWriter, r *http.Request) {
	var req protonAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		util.JSONError(w, "email and password are required", http.StatusBadRequest)
		return
	}

	c := &protonmail.Client{
		RootURL:    "https://mail.proton.me/api",
		AppVersion: "Other",
	}
	authInfo, err := c.AuthInfo(req.Email)
	if err != nil {
		h.logger.Error("Failed to get auth info", "error", err)
		util.JSONError(w, "Failed to get auth info", http.StatusInternalServerError)
		return
	}

	auth, err := c.Auth(req.Email, req.Password, authInfo)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		util.JSONError(w, "Incorrect login credentials. Please try again", http.StatusBadRequest)
		return
	}

	if auth.TwoFactor.Enabled != 0 {
		if req.TOTP == "" {
			util.JSONError(w, "2FA enabled: TOTP code required", http.StatusBadRequest)
			return
		}
		if auth.TwoFactor.TOTP != 1 {
			util.JSONError(w, "Only TOTP is supported as a 2FA method", http.StatusBadRequest)
			return
		}
		scope, err := c.AuthTOTP(req.TOTP)
		if err != nil {
			h.logger.Error("TOTP verification failed", "error", err)
			util.JSONError(w, "Invalid 2FA code. Please try again", http.StatusBadRequest)
			return
		}
		auth.Scope = scope
	}

	credStore, err := credentials.NewStore(h.db, h.apiKey)
	if err != nil {
		h.logger.Error("Failed to initialize credentials store", "error", err)
		util.JSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	keySalts, err := c.ListKeySalts()
	if err != nil {
		h.logger.Error("Failed to get key salts", "error", err)
		util.JSONError(w, "Failed to complete authentication", http.StatusInternalServerError)
		return
	}

	creds := &credentials.ProtonCredentials{
		Email:        req.Email,
		Password:     req.Password,
		UID:          auth.UID,
		AccessToken:  auth.AccessToken,
		RefreshToken: auth.RefreshToken,
		Scope:        auth.Scope,
		KeySalts:     keySalts,
	}

	if err := credStore.SaveProton(creds); err != nil {
		h.logger.Error("Failed to save Proton credentials", "error", err)
		util.JSONError(w, "Failed to save credentials", http.StatusInternalServerError)
		return
	}

	if h.dispatcher != nil {
		if err := h.dispatcher.RegisterApp(prismTopic); err != nil {
			h.logger.Warn("Failed to auto-create Proton Mail app mapping", "error", err)
		} else {
			h.logger.Info("Created Proton Mail app mapping")
		}
	}

	if h.integration != nil {
		ctx := context.Background()
		if err := h.integration.monitor.Start(ctx, credStore); err != nil {
			h.logger.Error("Failed to start Proton monitor after auth", "error", err)
			util.JSONError(w, "Authentication failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		h.logger.Info("Proton monitor started")
		if h.integration.handlers != nil {
			h.integration.handlers.username = req.Email
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (h *authHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	credStore, err := credentials.NewStore(h.db, h.apiKey)
	if err != nil {
		h.logger.Error("Failed to initialize credentials store", "error", err)
		util.JSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := credStore.DeleteIntegration(credentials.IntegrationProton); err != nil {
		h.logger.Error("Failed to delete integration", "error", err)
		util.JSONError(w, "Failed to delete integration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func RegisterRoutes(router *chi.Mux, handlers *Handlers, auth func(http.Handler) http.Handler, db *sql.DB, apiKey string, logger *slog.Logger, integration *Integration) {
	if handlers == nil {
		return
	}

	handlers.SetDB(db, apiKey)

	authH := &authHandler{
		db:          db,
		apiKey:      apiKey,
		logger:      logger,
		integration: integration,
		dispatcher:  integration.dispatcher,
	}

	router.With(auth).Get("/fragment/proton", handlers.HandleFragment)

	router.Route("/api/proton", func(r chi.Router) {
		r.Use(auth)
		r.Post("/mark-read", handlers.HandleMarkRead)
		r.Post("/archive", handlers.HandleArchive)
		r.Post("/auth", authH.handleAuth)
		r.Delete("/auth", authH.handleDelete)
	})
}
