package main

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"prism/service/config"
	"prism/service/server"
	"prism/service/util"

	"github.com/joho/godotenv"
)

//go:embed public/*
var publicAssets embed.FS

var (
	version = "dev"
)

func init() {
	_ = godotenv.Load() //nolint:errcheck
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("Prism %s\n", version)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := util.NewLogger(cfg.VerboseLogging)
	logger.Info("Starting Prism", "version", version)

	if err := runServer(cfg, logger); err != nil {
		logger.Error("Fatal error", "error", err)
		os.Exit(1)
	}
}

func runServer(cfg *config.Config, logger *slog.Logger) error {
	srv, err := server.New(cfg, publicAssets)
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
