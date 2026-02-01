package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/lone-cloud/prism/internal/config"
	"github.com/lone-cloud/prism/internal/server"
	"github.com/lone-cloud/prism/internal/util"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
)

func init() {
	_ = godotenv.Load() //nolint:errcheck // .env is optional
}

var rootCmd = &cobra.Command{
	Use:   "prism",
	Short: "Privacy-preserving push notifications via Signal",
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Prism server",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServer()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Prism %s (%s)\n", version, commit)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
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
