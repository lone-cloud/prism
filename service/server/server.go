package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"prism/service/config"
	"prism/service/notification"
	"prism/service/proton"
	"prism/service/signal"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	cfg              *config.Config
	store            *notification.Store
	dispatcher       *notification.Dispatcher
	protonMonitor    *proton.Monitor
	signalDaemon     *signal.Daemon
	linkDevice       *signal.LinkDevice
	logger           *slog.Logger
	router           *chi.Mux
	httpServer       *http.Server
	startTime        time.Time
	lastSignalLinked *bool // Track signal linked status for change detection
	lastAppsCount    *int  // Track apps count for change detection
}

func New(cfg *config.Config) (*Server, error) {
	logger := util.NewLogger(cfg.VerboseLogging)

	store, err := notification.NewStore(cfg.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	signalClient := signal.NewClient(cfg.SignalCLISocketPath)
	dispatcher := notification.NewDispatcher(store, signalClient, logger)

	protonMonitor := proton.NewMonitor(cfg, dispatcher, logger)

	signalDaemon := signal.NewDaemon(cfg.SignalCLIBinaryPath, cfg.SignalCLIDataPath, cfg.SignalCLISocketPath)
	linkDevice := signal.NewLinkDevice(signalClient, cfg.DeviceName)

	s := &Server{
		cfg:           cfg,
		store:         store,
		dispatcher:    dispatcher,
		protonMonitor: protonMonitor,
		signalDaemon:  signalDaemon,
		linkDevice:    linkDevice,
		logger:        logger,
		startTime:     time.Now(),
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

	r.Handle("/*", http.StripPrefix("/", http.FileServer(http.Dir("./public"))))

	r.Route("/fragment", func(r chi.Router) {
		r.Use(authMiddleware(s.cfg.APIKey))
		r.Get("/health", s.handleFragmentHealth)
		r.Get("/signal-info", s.handleFragmentSignalInfo)
		r.Get("/apps", s.handleFragmentApps)
	})

	r.Route("/action", func(r chi.Router) {
		r.Use(authMiddleware(s.cfg.APIKey))
		r.Delete("/app/{appName}", s.handleDeleteAppAction)
		r.Post("/toggle-channel", s.handleToggleChannelAction)
	})

	r.Route("/api/admin", func(r chi.Router) {
		r.Use(authMiddleware(s.cfg.APIKey))
		r.Get("/mappings", s.handleGetMappings)
		r.Post("/mappings", s.handleCreateMapping)
		r.Delete("/mappings/{appName}", s.handleDeleteMapping)
		r.Put("/mappings/{appName}/channel", s.handleUpdateChannel)
		r.Get("/stats", s.handleGetStats)
	})

	r.Route("/webpush/app", func(r chi.Router) {
		r.Use(authMiddleware(s.cfg.APIKey))
		r.Post("/", s.handleWebPushRegister)
		r.Delete("/{appName}", s.handleWebPushUnregister)
	})

	r.Route("/api/proton-mail", func(r chi.Router) {
		r.Use(authMiddleware(s.cfg.APIKey))
		r.Post("/mark-read", s.handleProtonMarkRead)
	})

	r.With(authMiddleware(s.cfg.APIKey)).Post("/{topic}", s.handleNtfyPublish)

	s.router = r
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting Signal daemon")
	if err := s.signalDaemon.Start(); err != nil {
		return fmt.Errorf("failed to start signal daemon: %w", err)
	}

	go func() {
		if err := s.protonMonitor.Start(ctx); err != nil && err != context.Canceled {
			s.logger.Error("Proton monitor error", "error", err)
		}
	}()

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
