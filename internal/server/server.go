package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Service   string `json:"service"`
	Version   string `json:"version"`
}

// ReadyResponse represents the readiness check response
type ReadyResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Service   string `json:"service"`
	Version   string `json:"version"`
}

// Server represents the HTTP server
type Server struct {
	httpServer *http.Server
	mcpServer  mcp.MCPServer
	logger     *logger.Logger
	config     *config.Config
	mux        *http.ServeMux
}

// New creates a new HTTP server instance
func New(cfg *config.Config, log *logger.Logger) *Server {
	mux := http.NewServeMux()

	// Create MCP server instance
	mcpImpl := mcp.Implementation{
		Name:    cfg.Logger.Service,
		Version: cfg.Logger.Version,
	}
	mcpSrv := mcp.NewServer(mcpImpl, cfg, log)

	server := &Server{
		logger:    log,
		config:    cfg,
		mux:       mux,
		mcpServer: mcpSrv,
		httpServer: &http.Server{
			Addr:           fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
			Handler:        mux,
			ReadTimeout:    cfg.Server.ReadTimeout,
			WriteTimeout:   cfg.Server.WriteTimeout,
			IdleTimeout:    cfg.Server.IdleTimeout,
			MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
		},
	}

	// Setup routes (placeholder for now)
	server.setupRoutes()

	return server
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/ready", s.handleReady)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Log the health check request
	s.logger.Info("health check requested",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
	)

	// Create health response
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Service:   s.config.Logger.Service,
		Version:   s.config.Logger.Version,
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Marshal response to JSON
	jsonData, err := json.Marshal(response)
	if err != nil {
		s.logger.Error("failed to marshal health response",
			"error", err,
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Write successful response
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)

	s.logger.Info("health check completed successfully",
		"status", response.Status,
		"timestamp", response.Timestamp,
	)
}

// handleReady handles readiness check requests
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	// Log the readiness check request
	s.logger.Info("readiness check requested",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
	)

	// Create readiness response
	response := ReadyResponse{
		Status:    "ready",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Service:   s.config.Logger.Service,
		Version:   s.config.Logger.Version,
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Marshal response to JSON
	jsonData, err := json.Marshal(response)
	if err != nil {
		s.logger.Error("failed to marshal readiness response",
			"error", err,
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Write successful response
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)

	s.logger.Info("readiness check completed successfully",
		"status", response.Status,
		"timestamp", response.Timestamp,
	)
}

// ListenAndServe starts the HTTP server
func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Close immediately closes the HTTP server
func (s *Server) Close() error {
	return s.httpServer.Close()
}

// StartMCP starts the MCP server
func (s *Server) StartMCP(ctx context.Context) error {
	s.logger.Info("Starting MCP server")
	
	// Create stdio transport for MCP server
	transport := mcp.NewStdioTransport()
	
	// Start the MCP server
	if err := s.mcpServer.Start(ctx, transport); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}
	
	s.logger.Info("MCP server started successfully")
	return nil
}

// StopMCP stops the MCP server
func (s *Server) StopMCP(ctx context.Context) error {
	s.logger.Info("Stopping MCP server")
	
	if err := s.mcpServer.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop MCP server: %w", err)
	}
	
	s.logger.Info("MCP server stopped successfully")
	return nil
}

// IsMCPRunning returns true if the MCP server is running
func (s *Server) IsMCPRunning() bool {
	// This is a simple implementation - in a real scenario we might need
	// to track the running state more carefully
	return s.mcpServer != nil
}