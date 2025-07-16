package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
	"mcp-server/internal/tools"
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

// ToolsHealthResponse represents detailed tool health information
type ToolsHealthResponse struct {
	Status    string                    `json:"status"`
	Timestamp string                    `json:"timestamp"`
	Summary   ToolHealthSummary         `json:"summary"`
	Tools     map[string]ToolHealthInfo `json:"tools"`
}

// ToolHealthSummary provides overall tool health statistics
type ToolHealthSummary struct {
	Total      int `json:"total"`
	Active     int `json:"active"`
	Loaded     int `json:"loaded"`
	Registered int `json:"registered"`
	Error      int `json:"error"`
	Disabled   int `json:"disabled"`
}

// ToolHealthInfo provides detailed information about a specific tool
type ToolHealthInfo struct {
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	Description  string    `json:"description"`
	Version      string    `json:"version"`
	Capabilities []string  `json:"capabilities"`
	LastCheck    string    `json:"last_check"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// MetricsResponse represents registry metrics information
type MetricsResponse struct {
	Status           string            `json:"status"`
	Timestamp        string            `json:"timestamp"`
	Registry         RegistryMetrics   `json:"registry"`
	Adapter          AdapterMetrics    `json:"adapter"`
	Tools            ToolMetrics       `json:"tools"`
	Performance      PerformanceMetrics `json:"performance"`
}

// RegistryMetrics represents registry-specific metrics
type RegistryMetrics struct {
	TotalTools       int     `json:"total_tools"`
	ActiveTools      int     `json:"active_tools"`
	ErrorTools       int     `json:"error_tools"`
	LoadedTools      int     `json:"loaded_tools"`
	RegisteredTools  int     `json:"registered_tools"`
	SuccessRate      float64 `json:"success_rate"`
	ErrorRate        float64 `json:"error_rate"`
	UptimeSeconds    int64   `json:"uptime_seconds"`
}

// AdapterMetrics represents adapter-specific metrics
type AdapterMetrics struct {
	Library         string  `json:"library"`
	Version         string  `json:"version"`
	Running         bool    `json:"running"`
	ToolCount       int     `json:"tool_count"`
	ResourceCount   int     `json:"resource_count"`
	SuccessRate     float64 `json:"success_rate"`
}

// ToolMetrics represents tool execution metrics
type ToolMetrics struct {
	TotalExecutions int64   `json:"total_executions"`
	SuccessfulRuns  int64   `json:"successful_runs"`
	FailedRuns      int64   `json:"failed_runs"`
	AverageLatency  float64 `json:"average_latency_ms"`
}

// PerformanceMetrics represents performance statistics
type PerformanceMetrics struct {
	RequestsPerSecond float64 `json:"requests_per_second"`
	P95LatencyMs      float64 `json:"p95_latency_ms"`
	P99LatencyMs      float64 `json:"p99_latency_ms"`
	MemoryUsageMB     float64 `json:"memory_usage_mb"`
}

// Server represents the HTTP server
type Server struct {
	httpServer *http.Server
	mcpServer  mcp.MCPServer
	registry   tools.ToolRegistry
	logger     *logger.Logger
	config     *config.Config
	mux        *http.ServeMux
	startTime  time.Time
}

// New creates a new HTTP server instance
func New(cfg *config.Config, log *logger.Logger) *Server {
	mux := http.NewServeMux()

	// Create tool registry using factory
	registryFactory := tools.NewRegistryFactory(cfg, log)
	registry, err := registryFactory.CreateRegistry()
	if err != nil {
		log.Error("failed to create tool registry", "error", err)
		// Fall back to default registry for robustness
		registry = tools.NewDefaultToolRegistry(cfg, log)
	}

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
		registry:  registry,
		startTime: time.Now(),
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
	s.mux.HandleFunc("/metrics", s.handleMetrics)
	s.mux.HandleFunc("/tools/health", s.handleToolsHealth)
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
	s.logger.Info("Starting MCP server and tool registry")
	
	// Start tool registry first
	if err := s.registry.Start(ctx); err != nil {
		return fmt.Errorf("failed to start tool registry: %w", err)
	}
	s.logger.Info("Tool registry started successfully")
	
	// Create stdio transport for MCP server
	transport := mcp.NewStdioTransport()
	
	// Start the MCP server
	if err := s.mcpServer.Start(ctx, transport); err != nil {
		// If MCP server fails to start, stop the registry
		if stopErr := s.registry.Stop(ctx); stopErr != nil {
			s.logger.Error("failed to stop registry after MCP server start failure", "error", stopErr)
		}
		return fmt.Errorf("failed to start MCP server: %w", err)
	}
	
	s.logger.Info("MCP server started successfully")
	return nil
}

// StopMCP stops the MCP server
func (s *Server) StopMCP(ctx context.Context) error {
	s.logger.Info("Stopping MCP server and tool registry")
	
	// Stop MCP server first
	if err := s.mcpServer.Stop(ctx); err != nil {
		s.logger.Error("failed to stop MCP server", "error", err)
		// Continue to stop registry even if MCP server fails to stop
	} else {
		s.logger.Info("MCP server stopped successfully")
	}
	
	// Stop tool registry
	if err := s.registry.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop tool registry: %w", err)
	}
	s.logger.Info("Tool registry stopped successfully")
	
	return nil
}

// IsMCPRunning returns true if the MCP server is running
func (s *Server) IsMCPRunning() bool {
	// This is a simple implementation - in a real scenario we might need
	// to track the running state more carefully
	return s.mcpServer != nil
}

// Data Collection Functions - Single responsibility: gather raw data

// collectRegistryData gathers registry health and tool information
func (s *Server) collectRegistryData() (tools.RegistryHealth, []tools.ToolInfo) {
	return s.registry.Health(), s.registry.List()
}

// collectPerformanceData gathers runtime performance statistics
func (s *Server) collectPerformanceData() runtime.MemStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return memStats
}

// collectUptimeData calculates server uptime
func (s *Server) collectUptimeData() time.Duration {
	return time.Since(s.startTime)
}

// Metrics Calculation Functions - Single responsibility: calculate specific metrics

// calculateRegistryMetrics computes registry-specific metrics
func (s *Server) calculateRegistryMetrics(health tools.RegistryHealth, toolList []tools.ToolInfo, uptime time.Duration) RegistryMetrics {
	var successRate, errorRate float64
	if health.ToolCount > 0 {
		successRate = float64(health.ActiveTools) / float64(health.ToolCount) * 100.0
		errorRate = float64(health.ErrorTools) / float64(health.ToolCount) * 100.0
	}
	
	loadedTools := 0
	registeredTools := 0
	for _, tool := range toolList {
		switch tool.Status {
		case "loaded":
			loadedTools++
		case "registered":
			registeredTools++
		}
	}
	
	return RegistryMetrics{
		TotalTools:      health.ToolCount,
		ActiveTools:     health.ActiveTools,
		ErrorTools:      health.ErrorTools,
		LoadedTools:     loadedTools,
		RegisteredTools: registeredTools,
		SuccessRate:     successRate,
		ErrorRate:       errorRate,
		UptimeSeconds:   int64(uptime.Seconds()),
	}
}

// calculateAdapterMetrics computes adapter-specific metrics
func (s *Server) calculateAdapterMetrics(health tools.RegistryHealth, successRate float64) AdapterMetrics {
	return AdapterMetrics{
		Library:       "mark3labs",
		Version:       "0.33.0",
		Running:       true,
		ToolCount:     health.ToolCount,
		ResourceCount: 0, // No resources tracked yet
		SuccessRate:   successRate,
	}
}

// calculateToolMetrics computes tool execution metrics
func (s *Server) calculateToolMetrics(health tools.RegistryHealth) ToolMetrics {
	return ToolMetrics{
		TotalExecutions: 0,
		SuccessfulRuns:  0,
		FailedRuns:      int64(health.ErrorTools),
		AverageLatency:  0.0,
	}
}

// calculatePerformanceMetrics computes performance statistics
func (s *Server) calculatePerformanceMetrics(memStats runtime.MemStats) PerformanceMetrics {
	memoryUsageMB := float64(memStats.Alloc) / 1024 / 1024
	
	return PerformanceMetrics{
		RequestsPerSecond: 0.0,
		P95LatencyMs:      0.0,
		P99LatencyMs:      0.0,
		MemoryUsageMB:     memoryUsageMB,
	}
}

// determineOverallHealth determines overall server health status
func (s *Server) determineOverallHealth(registryHealth tools.RegistryHealth) string {
	if registryHealth.Status == "stopped" {
		return "degraded"
	}
	if registryHealth.Status == "degraded" || registryHealth.ErrorTools > 0 {
		return "degraded"
	}
	return "healthy"
}

// Business Logic Function - Single responsibility: orchestrate and build response

// buildMetricsResponse collects data and constructs the complete metrics response
func (s *Server) buildMetricsResponse() MetricsResponse {
	registryHealth, toolList := s.collectRegistryData()
	memStats := s.collectPerformanceData()
	uptime := s.collectUptimeData()
	
	registryMetrics := s.calculateRegistryMetrics(registryHealth, toolList, uptime)
	adapterMetrics := s.calculateAdapterMetrics(registryHealth, registryMetrics.SuccessRate)
	toolMetrics := s.calculateToolMetrics(registryHealth)
	perfMetrics := s.calculatePerformanceMetrics(memStats)
	
	return MetricsResponse{
		Status:      s.determineOverallHealth(registryHealth),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Registry:    registryMetrics,
		Adapter:     adapterMetrics,
		Tools:       toolMetrics,
		Performance: perfMetrics,
	}
}

// Tool Health Functions - Single responsibility: tool health data collection and aggregation

// collectToolHealthData gathers detailed tool health information
func (s *Server) collectToolHealthData() (tools.RegistryHealth, []tools.ToolInfo) {
	return s.registry.Health(), s.registry.List()
}

// buildToolHealthSummary calculates summary statistics from tool list
func (s *Server) buildToolHealthSummary(toolList []tools.ToolInfo) ToolHealthSummary {
	summary := ToolHealthSummary{Total: len(toolList)}
	
	for _, tool := range toolList {
		switch tool.Status {
		case tools.ToolStatusActive:
			summary.Active++
		case tools.ToolStatusLoaded:
			summary.Loaded++
		case tools.ToolStatusRegistered:
			summary.Registered++
		case tools.ToolStatusError:
			summary.Error++
		case tools.ToolStatusDisabled:
			summary.Disabled++
		}
	}
	
	return summary
}

// buildToolHealthDetails creates detailed tool health information map
func (s *Server) buildToolHealthDetails(toolList []tools.ToolInfo, registryHealth tools.RegistryHealth) map[string]ToolHealthInfo {
	toolDetails := make(map[string]ToolHealthInfo)
	
	for _, tool := range toolList {
		details := ToolHealthInfo{
			Name:         tool.Name,
			Status:       string(tool.Status),
			Description:  tool.Description,
			Version:      tool.Version,
			Capabilities: tool.Capabilities,
			LastCheck:    registryHealth.LastCheck,
		}
		
		if tool.Status == tools.ToolStatusError {
			details.ErrorMessage = "Tool failed validation or creation"
		}
		
		toolDetails[tool.Name] = details
	}
	
	return toolDetails
}

// determineToolsOverallHealth determines overall tools health status
func (s *Server) determineToolsOverallHealth(summary ToolHealthSummary, registryHealth tools.RegistryHealth) string {
	if registryHealth.Status == "stopped" {
		return "stopped"
	}
	if summary.Error > 0 {
		return "degraded"
	}
	if summary.Active == 0 && summary.Total > 0 {
		return "degraded"
	}
	return "healthy"
}

// buildToolsHealthResponse orchestrates tools health data collection and response building
func (s *Server) buildToolsHealthResponse() ToolsHealthResponse {
	registryHealth, toolList := s.collectToolHealthData()
	summary := s.buildToolHealthSummary(toolList)
	toolDetails := s.buildToolHealthDetails(toolList, registryHealth)
	
	return ToolsHealthResponse{
		Status:    s.determineToolsOverallHealth(summary, registryHealth),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Summary:   summary,
		Tools:     toolDetails,
	}
}

// HTTP Handler Function - Single responsibility: HTTP request/response handling

// handleToolsHealth handles detailed tools health requests
func (s *Server) handleToolsHealth(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("tools health requested",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
	)

	response := s.buildToolsHealthResponse()

	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.Marshal(response)
	if err != nil {
		s.logger.Error("failed to marshal tools health response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)

	s.logger.Info("tools health request completed successfully",
		"status", response.Status,
		"total_tools", response.Summary.Total,
		"active_tools", response.Summary.Active,
		"error_tools", response.Summary.Error,
	)
}

// handleMetrics handles metrics endpoint requests
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("metrics requested",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
	)

	response := s.buildMetricsResponse()

	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.Marshal(response)
	if err != nil {
		s.logger.Error("failed to marshal metrics response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)

	s.logger.Info("metrics request completed successfully",
		"status", response.Status,
		"uptime_seconds", response.Registry.UptimeSeconds,
		"memory_mb", response.Performance.MemoryUsageMB,
	)
}