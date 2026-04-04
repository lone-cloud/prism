package proton

import (
	"context"
	"database/sql"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"prism/service/config"
	"prism/service/credentials"
	"prism/service/util"

	"github.com/emersion/hydroxide/protonmail"
	"github.com/go-chi/chi/v5"
)

//go:embed templates/*.html
var Templates embed.FS

type authHandler struct {
	db          *sql.DB
	apiKey      string
	logger      *slog.Logger
	integration *Integration
	cfg         *config.Config
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
		RootURL:    protonAPIURL,
		AppVersion: protonAppVersion,
		Debug:      false,
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
		// Pre-2FA bearer is invalid for /keys/salts; refresh returns post-2FA tokens (hydroxide
		// does not apply them to the Client, so we use auth below for the salts request).
		auth.Scope = scope
		refreshed, err := c.AuthRefresh(auth)
		if err != nil {
			h.logger.Error("Failed to refresh session after 2FA", "error", err)
			util.JSONError(w, "Failed to complete authentication", http.StatusInternalServerError)
			return
		}
		auth = refreshed
	}

	credStore, err := credentials.NewStore(h.db, h.apiKey)
	if err != nil {
		h.logger.Error("Failed to initialize credentials store", "error", err)
		util.JSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	keySalts, err := listKeySaltsWithAuth(protonAPIURL, protonAppVersion, auth)
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

	if h.integration != nil {
		ctx := context.Background()
		if err := h.integration.monitor.Start(ctx, credStore, h.integration.Publisher); err != nil {
			h.logger.Error("Failed to start Proton monitor after auth", "error", err)
			util.JSONError(w, "Authentication failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		h.logger.Info("Proton monitor started")
		if h.integration.Handlers != nil {
			h.integration.Handlers.username = req.Email
		}
	}

	util.SetToast(w, "Proton Mail linked", "success")
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

	if h.integration != nil {
		h.integration.monitor.Stop()
	}

	util.SetToast(w, "Proton Mail unlinked", "success")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func listKeySaltsWithAuth(rootURL, appVersion string, auth *protonmail.Auth) (map[string][]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rootURL+"/keys/salts", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Pm-Appversion", appVersion)
	req.Header.Set("X-Pm-Apiversion", strconv.Itoa(protonmail.Version))
	req.Header.Set("X-Pm-Uid", auth.UID)
	req.Header.Set("Authorization", "Bearer "+auth.AccessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:101.0) Gecko/20100101 Firefox/101.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Code     int    `json:"Code"`
		Error    string `json:"Error"`
		KeySalts []struct {
			ID      string `json:"ID"`
			KeySalt string `json:"KeySalt"`
		} `json:"KeySalts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Code != 1000 {
		return nil, fmt.Errorf("[%d] %s", result.Code, result.Error)
	}

	salts := make(map[string][]byte, len(result.KeySalts))
	for _, ks := range result.KeySalts {
		if ks.KeySalt == "" {
			salts[ks.ID] = nil
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(ks.KeySalt)
		if err != nil {
			return nil, fmt.Errorf("failed to decode key salt for %s: %w", ks.ID, err)
		}
		salts[ks.ID] = decoded
	}
	return salts, nil
}

func RegisterRoutes(router *chi.Mux, handlers *Handlers, auth func(http.Handler) http.Handler, db *sql.DB, apiKey string, logger *slog.Logger, integration *Integration, cfg *config.Config) {
	if handlers == nil {
		return
	}

	handlers.DB = db
	handlers.APIKey = apiKey

	authH := &authHandler{
		db:          db,
		apiKey:      apiKey,
		logger:      logger,
		integration: integration,
		cfg:         cfg,
	}

	router.With(auth).Get("/fragment/proton", handlers.HandleFragment)

	router.Route("/api/v1/proton", func(r chi.Router) {
		r.Use(auth)
		r.Post("/mark-read", handlers.HandleMarkRead)
		r.Post("/archive", handlers.HandleArchive)
		r.Post("/trash", handlers.HandleTrash)
		r.Post("/auth", authH.handleAuth)
		r.Delete("/auth", authH.handleDelete)
	})
}
