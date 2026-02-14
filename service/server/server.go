package server

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"prism/service/config"
	"prism/service/integration"
	"prism/service/notification"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

//go:embed templates/*.html
var fragmentTemplates embed.FS

type Server struct {
	startTime    time.Time
	publicAssets embed.FS
	cfg          *config.Config
	store        *notification.Store
	dispatcher   *notification.Dispatcher
	integrations *integration.Integrations
	logger       *slog.Logger
	router       *chi.Mux
	httpServer   *http.Server
	indexTmpl    *template.Template
	fragmentTmpl *template.Template
	version      string
}

func New(cfg *config.Config, publicAssets embed.FS, version string) (*Server, error) {
	logger := util.NewLogger(cfg.VerboseLogging)

	store, err := notification.NewStore(cfg.StoragePath)
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
		dispatcher:   integrations.Dispatcher,
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
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/favicon.webp", http.StatusMovedPermanently)
	})

	integration.RegisterAll(s.integrations, r, s.cfg, s.store, s.logger, authMiddleware)

	r.With(authMiddleware(s.cfg.APIKey)).Get("/fragment/apps", s.handleFragmentApps)
	r.With(authMiddleware(s.cfg.APIKey)).Get("/fragment/integrations", s.handleFragmentIntegrations)

	r.Route("/action", func(r chi.Router) {
		r.Use(authMiddleware(s.cfg.APIKey))
		r.Delete("/app/{appName}", s.handleDeleteAppAction)
		r.Post("/toggle-channel", s.handleToggleChannelAction)
	})

	r.Route("/api/v1/admin", func(r chi.Router) {
		r.Use(authMiddleware(s.cfg.APIKey))
		r.Get("/mappings", s.handleGetMappings)
		r.Post("/mappings", s.handleCreateMapping)
		r.Delete("/mappings/{appName}", s.handleDeleteMapping)
		r.Put("/mappings/{appName}/channel", s.handleUpdateChannel)
		r.Get("/stats", s.handleGetStats)
	})

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
	s.integrations.Start(ctx, s.cfg, s.logger)

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

	if err := s.store.Close(); err != nil {
		return fmt.Errorf("failed to close store: %w", err)
	}

	return nil
}
