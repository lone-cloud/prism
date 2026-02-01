package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lone-cloud/prism/internal/config"
	"github.com/lone-cloud/prism/internal/notification"
	"github.com/lone-cloud/prism/internal/proton"
	"github.com/lone-cloud/prism/internal/signal"
	"github.com/lone-cloud/prism/internal/util"
)

type Server struct {
	cfg           *config.Config
	store         *notification.Store
	dispatcher    *notification.Dispatcher
	protonMonitor *proton.Monitor
	actionHandler *proton.ActionHandler
	signalDaemon  *signal.Daemon
	logger        *slog.Logger
	router        *chi.Mux
	httpServer    *http.Server
	startTime     time.Time
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
	actionHandler := proton.NewActionHandler(protonMonitor, cfg, logger)

	signalDaemon := signal.NewDaemon(cfg.SignalCLIBinaryPath, cfg.SignalCLIDataPath, cfg.SignalCLISocketPath)

	s := &Server{
		cfg:           cfg,
		store:         store,
		dispatcher:    dispatcher,
		protonMonitor: protonMonitor,
		actionHandler: actionHandler,
		signalDaemon:  signalDaemon,
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
	r.Use(securityHeadersMiddleware(s.cfg.AllowInsecureHTTP))
	r.Use(middleware.Timeout(5 * time.Second))
	r.Use(middleware.Compress(5))
	r.Use(middleware.StripSlashes)
	r.Use(rateLimitMiddleware(s.cfg.RateLimit, s.cfg.AllowInsecureHTTP))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public/index.html")
	})
	r.Handle("/*", http.StripPrefix("/", http.FileServer(http.Dir("./public"))))

	r.Route("/fragment", func(r chi.Router) {
		r.Use(authMiddleware(s.cfg.APIKey, s.cfg.AllowInsecureHTTP))
		r.Get("/health", s.handleFragmentHealth)
		r.Get("/signal-info", s.handleFragmentSignalInfo)
		r.Get("/endpoints", s.handleFragmentEndpoints)
	})

	r.Route("/action", func(r chi.Router) {
		r.Use(authMiddleware(s.cfg.APIKey, s.cfg.AllowInsecureHTTP))
		r.Delete("/delete-endpoint", s.handleDeleteEndpointAction)
		r.Post("/toggle-channel", s.handleToggleChannelAction)
	})

	r.Route("/admin", func(r chi.Router) {
		r.Use(authMiddleware(s.cfg.APIKey, s.cfg.AllowInsecureHTTP))
		r.Get("/mappings", s.handleGetMappings)
		r.Post("/mappings", s.handleCreateMapping)
		r.Delete("/mappings/{endpoint}", s.handleDeleteMapping)
		r.Put("/mappings/{endpoint}/channel", s.handleUpdateChannel)
		r.Get("/stats", s.handleGetStats)
	})

	r.Route("/ntfy", func(r chi.Router) {
		r.Use(authMiddleware(s.cfg.APIKey, s.cfg.AllowInsecureHTTP))
		r.Post("/{endpoint}", s.handleNtfyPublish)
	})

	r.Route("/webhook", func(r chi.Router) {
		r.Use(authMiddleware(s.cfg.APIKey, s.cfg.AllowInsecureHTTP))
		r.Post("/register", s.handleWebhookRegister)
	})

	r.Route("/api/webhook", func(r chi.Router) {
		r.Use(authMiddleware(s.cfg.APIKey, s.cfg.AllowInsecureHTTP))
		r.Post("/register", s.handleWebhookRegister)
	})

	r.Route("/api/proton-mail", func(r chi.Router) {
		r.Use(authMiddleware(s.cfg.APIKey, s.cfg.AllowInsecureHTTP))
		r.Post("/mark-read", s.handleProtonMarkRead)
	})

	r.Route("/proton", func(r chi.Router) {
		r.Use(authMiddleware(s.cfg.APIKey, s.cfg.AllowInsecureHTTP))
		r.Post("/mark-read", s.handleProtonMarkRead)
	})

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

	s.logger.Info("")
	s.logger.Info("Prism running on:")
	s.logger.Info(fmt.Sprintf("  Local:   http://localhost:%d", s.cfg.Port))

	if lanIP := util.GetLANIP(); lanIP != "" {
		s.logger.Debug(fmt.Sprintf("  Network: http://%s:%d", lanIP, s.cfg.Port))
	}
	s.logger.Info("")

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
