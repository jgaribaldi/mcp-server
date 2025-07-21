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
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(ExitCodeError)
	}

	log, err := setupLogging(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(ExitCodeError)
	}

	srv, err := setupServers(cfg, log)
	if err != nil {
		log.Error("Failed to setup servers", "error", err)
		os.Exit(ExitCodeError)
	}

	if err := runServers(srv, log); err != nil {
		log.Error("Server startup failed", "error", err)
		os.Exit(ExitCodeError)
	}

	gracefulShutdown(srv, log)

	os.Exit(ExitCodeOK)
}

// setupLogging initializes the logger with the given configuration
func setupLogging(cfg *config.Config) (*logger.Logger, error) {
	return logger.New(logger.Config{
		Level:     cfg.Logger.Level,
		Format:    cfg.Logger.Format,
		Service:   cfg.Logger.Service,
		Version:   cfg.Logger.Version,
		UseEmojis: cfg.Logger.UseEmojis,
	})
}

// setupServers initializes all servers (HTTP, etc.)
func setupServers(cfg *config.Config, log *logger.Logger) (*server.Server, error) {
	log.Info("Setting up servers",
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
		"version", cfg.Logger.Version)

	srv := server.New(cfg, log)

	if err := registerAllTools(srv.ToolRegistry(), log); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	ctx := context.Background()
	if err := srv.StartMCP(ctx); err != nil {
		return nil, fmt.Errorf("failed to start MCP server: %w", err)
	}

	return srv, nil
}

// runServers starts all servers and waits for shutdown signal
func runServers(srv *server.Server, log *logger.Logger) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	serverErrChan := make(chan error, 1)
	go func() {
		log.Info("Starting HTTP server")
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server failed", "error", err)
			serverErrChan <- err
		}
	}()

	log.Info("All servers are running",
		"http_endpoint", fmt.Sprintf("http://%s:%d", "localhost", 3000),
		"mcp_protocol", "stdio")

	select {
	case sig := <-sigChan:
		log.Info("Received shutdown signal", "signal", sig.String())
		return nil
	case err := <-serverErrChan:
		return err
	}
}

func gracefulShutdown(srv *server.Server, log *logger.Logger) {
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	log.Info("Shutting down MCP server...")
	if err := srv.StopMCP(shutdownCtx); err != nil {
		log.Error("Error during MCP server shutdown", "error", err)
	} else {
		log.Info("MCP server stopped gracefully")
	}

	log.Info("Shutting down HTTP server...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Error during HTTP server shutdown", "error", err)
		// Force close if graceful shutdown fails
		if closeErr := srv.Close(); closeErr != nil {
			log.Error("Error force closing HTTP server", "error", closeErr)
		}
	} else {
		log.Info("HTTP server stopped gracefully")
	}

	log.Info("All servers stopped gracefully")
}

