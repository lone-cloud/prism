package server

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"prism/service/config"
	"prism/service/delivery"
	"prism/service/integration"
	"prism/service/subscription"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	_ "modernc.org/sqlite"
)

//go:embed templates/*.html
var fragmentTemplates embed.FS

type Server struct {
	startTime    time.Time
	publicAssets embed.FS
	cfg          *config.Config
	store        *subscription.Store
	publisher    *delivery.Publisher
	integrations *integration.Integrations
	logger       *slog.Logger
	router       *chi.Mux
	httpServer   *http.Server
	indexTmpl    *template.Template
	fragmentTmpl *template.Template
	version      string
}

func openDB(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db, nil
}

func New(cfg *config.Config, publicAssets embed.FS, version string, logger *slog.Logger) (*Server, error) {
	db, err := openDB(cfg.StoragePath)
	if err != nil {
		return nil, err
	}

	store, err := subscription.NewStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	fragmentTmpl := template.New("")
	fragmentTmpl, err = fragmentTmpl.ParseFS(fragmentTemplates, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse fragment templates: %w", err)
	}

	integrations, fragmentTmpl, err := integration.Initialize(cfg, store, logger, fragmentTmpl)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.ParseFS(publicAssets, "public/index.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse index template: %w", err)
	}

	s := &Server{
		cfg:          cfg,
		store:        store,
		publisher:    integrations.Publisher,
		integrations: integrations,
		logger:       logger,
		startTime:    time.Now(),
		version:      version,
		indexTmpl:    tmpl,
		fragmentTmpl: fragmentTmpl,
		publicAssets: publicAssets,
	}

	s.setupRoutes()
	return s, nil
}

func (s *Server) setupRoutes() {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(loggingMiddleware(s.logger))
	r.Use(securityHeadersMiddleware())
	r.Use(middleware.Timeout(5 * time.Second))
	r.Use(middleware.Compress(5))
	r.Use(middleware.StripSlashes)
	r.Use(rateLimitMiddleware(s.cfg.RateLimit))
	r.Use(maxBodySizeMiddleware(1 << 20))

	r.Get("/health", s.handleHealthCheck)
	r.Head("/health", s.handleHealthCheck)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if err := s.indexTmpl.Execute(w, map[string]string{"Version": s.version}); err != nil {
			util.LogAndError(w, s.logger, "Internal Server Error", http.StatusInternalServerError, err)
		}
	})

	integration.RegisterAll(s.integrations, r, s.cfg, s.logger, authMiddleware)

	r.With(authMiddleware(s.cfg.APIKey)).Get("/fragment/apps", s.handleFragmentApps)
	r.With(authMiddleware(s.cfg.APIKey)).Get("/fragment/integrations", s.handleFragmentIntegrations)

	r.Route("/apps", func(r chi.Router) {
		r.Use(authMiddleware(s.cfg.APIKey))
		r.Delete("/{appName}", s.handleDeleteApp)
		r.Post("/{appName}/subscriptions", s.handleCreateSubscription)
		r.Delete("/{appName}/subscriptions/{subscriptionId}", s.handleDeleteSubscription)
	})

	r.With(authMiddleware(s.cfg.APIKey)).Get("/api/v1/apps", s.handleGetApps)
	r.With(authMiddleware(s.cfg.APIKey)).Get("/api/v1/health", s.handleHealth)

	r.With(authMiddleware(s.cfg.APIKey)).Post("/{appName}", s.handleNtfyPublish)

	publicFS, err := fs.Sub(s.publicAssets, "public")
	if err != nil {
		s.logger.Error("Failed to create public assets sub-filesystem", "error", err)
	} else {
		r.Handle("/*", http.FileServer(http.FS(publicFS)))
	}

	s.router = r
}

func (s *Server) Start(ctx context.Context) error {
	s.integrations.Start(ctx, s.logger)

	addr := fmt.Sprintf(":%d", s.cfg.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	msg := fmt.Sprintf("Prism running on:\n  Local: http://localhost:%d", s.cfg.Port)
	if lanIP := util.GetLANIP(); lanIP != "" {
		msg += fmt.Sprintf("\n  Network: http://%s:%d", lanIP, s.cfg.Port)
	}
	s.logger.Info(msg)

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		return s.Shutdown()
	case err := <-errCh:
		return err
	}
}

func (s *Server) Shutdown() error {
	s.logger.Info("Shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown http server: %w", err)
	}

	if err := s.store.DB.Close(); err != nil {
		return fmt.Errorf("failed to close store: %w", err)
	}

	return nil
}
