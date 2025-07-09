package server

import (
	"fmt"
	"net/http"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
)

// Server represents the HTTP server
type Server struct {
	httpServer *http.Server
	logger     *logger.Logger
	config     *config.Config
	mux        *http.ServeMux
}

// New creates a new HTTP server instance
func New(cfg *config.Config, log *logger.Logger) *Server {
	mux := http.NewServeMux()

	server := &Server{
		logger: log,
		config: cfg,
		mux:    mux,
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
	// Placeholder - routes will be added in Steps 2.3 and 2.4
}