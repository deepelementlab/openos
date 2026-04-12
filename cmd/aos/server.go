package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/agentos/aos/internal/config"
	"github.com/agentos/aos/internal/server"
)

func newServerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "Run the OpenOS API and control plane server",
		RunE:  runServer,
	}
}

func runServer(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if debugMode {
		cfg.Debug = true
	}
	logger := initLogger(cfg)
	defer logger.Sync()

	logger.Info("Starting OpenOS server",
		zap.String("mode", cfg.Mode),
		zap.Bool("debug", cfg.Debug),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	srv, err := server.NewServer(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(ctx); err != nil {
			errCh <- err
		}
	}()

	select {
	case sig := <-sigCh:
		logger.Info("shutdown signal", zap.String("signal", sig.String()))
		cancel()
	case err := <-errCh:
		logger.Error("server error", zap.Error(err))
		cancel()
		return err
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	logger.Info("shutdown complete")
	return nil
}

func initLogger(cfg *config.Config) *zap.Logger {
	var logger *zap.Logger
	var err error
	if cfg.Debug {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: %v\n", err)
		os.Exit(1)
	}
	return logger
}
