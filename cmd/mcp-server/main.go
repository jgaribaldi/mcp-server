package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
	"mcp-server/internal/server"
)

const (
	ExitCodeOK    = 0
	ExitCodeError = 1
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(ExitCodeError)
	}

	// Setup logging
	log, err := setupLogging(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(ExitCodeError)
	}

	// Setup servers
	srv, err := setupServers(cfg, log)
	if err != nil {
		log.Error("Failed to setup servers", "error", err)
		os.Exit(ExitCodeError)
	}

	// Run servers and wait for shutdown signal
	if err := runServers(srv, log); err != nil {
		log.Error("Server startup failed", "error", err)
		os.Exit(ExitCodeError)
	}

	// Perform graceful shutdown
	gracefulShutdown(srv, log)

	os.Exit(ExitCodeOK)
}

// setupLogging initializes the logger with the given configuration
func setupLogging(cfg *config.Config) (*logger.Logger, error) {
	return logger.New(logger.Config{
		Level:   cfg.Logger.Level,
		Format:  cfg.Logger.Format,
		Service: cfg.Logger.Service,
		Version: cfg.Logger.Version,
	})
}

// setupServers initializes all servers (HTTP, etc.)
func setupServers(cfg *config.Config, log *logger.Logger) (*server.Server, error) {
	log.Info("Setting up servers",
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
		"version", cfg.Logger.Version)

	// Initialize HTTP server
	srv := server.New(cfg, log)

	return srv, nil
}

// runServers starts all servers and waits for shutdown signal
func runServers(srv *server.Server, log *logger.Logger) error {
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server in background goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		log.Info("Starting HTTP server")
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server failed", "error", err)
			serverErrChan <- err
		}
	}()

	log.Info("All servers are running")

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		log.Info("Received shutdown signal", "signal", sig.String())
		return nil
	case err := <-serverErrChan:
		return err
	}
}

// gracefulShutdown performs graceful shutdown of all servers
func gracefulShutdown(srv *server.Server, log *logger.Logger) {
	// Give some time for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server gracefully
	log.Info("Shutting down HTTP server...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Error during server shutdown", "error", err)
		// Force close if graceful shutdown fails
		if closeErr := srv.Close(); closeErr != nil {
			log.Error("Error force closing server", "error", closeErr)
		}
	} else {
		log.Info("HTTP server stopped gracefully")
	}
}