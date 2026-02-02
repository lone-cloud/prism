package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"prism/internal/config"
	"prism/internal/server"
	"prism/internal/util"

	"github.com/joho/godotenv"
)

var (
	version = "dev"
	commit  = "unknown"
)

func init() {
	_ = godotenv.Load() //nolint:errcheck // .env is optional
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("Prism %s (%s)\n", version, commit)
		return
	}

	if err := runServer(); err != nil {
		logger := util.NewLogger(false)
		logger.Error("Fatal error", "error", err)
		os.Exit(1)
	}
}

func runServer() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger := util.NewLogger(cfg.VerboseLogging)
	logger.Info("Starting Prism", "version", version)

	srv, err := server.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Start(ctx)
	}()

	select {
	case <-ctx.Done():
		logger.Info("Received shutdown signal")
		return srv.Shutdown()
	case err := <-serverErr:
		return err
	}
}
