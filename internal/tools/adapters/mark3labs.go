package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	mcpintf "mcp-server/internal/mcp"
	"mcp-server/internal/config"
	"mcp-server/internal/logger"
)

type Mark3LabsAdapter struct {
	logger      *logger.Logger
	config      *config.Config
	mcpServer   *server.MCPServer
	tools       map[string]mcpintf.Tool
	resources   map[string]mcpintf.Resource
	mu          sync.RWMutex
	running     bool
	lastCheck   time.Time
}

func NewMark3LabsAdapter(cfg *config.Config, log *logger.Logger) *Mark3LabsAdapter {
	return &Mark3LabsAdapter{
		logger:    log,
		config:    cfg,
		tools:     make(map[string]mcpintf.Tool),
		resources: make(map[string]mcpintf.Resource),
		lastCheck: time.Now(),
	}
}

func (a *Mark3LabsAdapter) RegisterTool(tool mcpintf.Tool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.logger.Info("registering tool with mark3labs adapter",
		"name", tool.Name(),
		"description", tool.Description(),
	)

	// Store tool in our registry
	if _, exists := a.tools[tool.Name()]; exists {
		return fmt.Errorf("tool '%s' already exists", tool.Name())
	}

	a.tools[tool.Name()] = tool

	// Register with mark3labs server if running
	if a.running && a.mcpServer != nil {
		return a.registerToolWithServer(tool)
	}

	return nil
}

func (a *Mark3LabsAdapter) UnregisterTool(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.logger.Info("unregistering tool from mark3labs adapter", "name", name)

	if _, exists := a.tools[name]; !exists {
		return fmt.Errorf("tool '%s' not found", name)
	}

	delete(a.tools, name)

	// Note: mark3labs/mcp-go doesn't have explicit tool unregistration
	// Tools are managed by the server instance lifecycle

	return nil
}

func (a *Mark3LabsAdapter) GetTool(name string) (mcpintf.Tool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	tool, exists := a.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}

	return tool, nil
}

func (a *Mark3LabsAdapter) ListTools() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	tools := make([]string, 0, len(a.tools))
	for name := range a.tools {
		tools = append(tools, name)
	}
	return tools
}

func (a *Mark3LabsAdapter) RegisterResource(resource mcpintf.Resource) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.logger.Info("registering resource with mark3labs adapter",
		"uri", resource.URI(),
		"name", resource.Name(),
	)

	if _, exists := a.resources[resource.URI()]; exists {
		return fmt.Errorf("resource '%s' already exists", resource.URI())
	}

	a.resources[resource.URI()] = resource

	if a.running && a.mcpServer != nil {
		return a.registerResourceWithServer(resource)
	}

	return nil
}

func (a *Mark3LabsAdapter) UnregisterResource(uri string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.logger.Info("unregistering resource from mark3labs adapter", "uri", uri)

	if _, exists := a.resources[uri]; !exists {
		return fmt.Errorf("resource '%s' not found", uri)
	}

	delete(a.resources, uri)

	// Note: mark3labs/mcp-go doesn't have explicit resource unregistration
	// Resources are managed by the server instance lifecycle

	return nil
}

func (a *Mark3LabsAdapter) GetResource(uri string) (mcpintf.Resource, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	resource, exists := a.resources[uri]
	if !exists {
		return nil, fmt.Errorf("resource '%s' not found", uri)
	}

	return resource, nil
}

func (a *Mark3LabsAdapter) ListResources() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	resources := make([]string, 0, len(a.resources))
	for uri := range a.resources {
		resources = append(resources, uri)
	}
	return resources
}

func (a *Mark3LabsAdapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return fmt.Errorf("mark3labs adapter is already running")
	}

	a.logger.Info("starting mark3labs adapter")

	a.mcpServer = server.NewMCPServer(
		"mcp-server",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithResourceCapabilities(true, true),
		server.WithRecovery(),
	)

	for _, tool := range a.tools {
		if err := a.registerToolWithServer(tool); err != nil {
			a.logger.Error("failed to register tool with mark3labs server",
				"name", tool.Name(),
				"error", err,
			)
			return fmt.Errorf("failed to register tool %s: %w", tool.Name(), err)
		}
	}

	for _, resource := range a.resources {
		if err := a.registerResourceWithServer(resource); err != nil {
			a.logger.Error("failed to register resource with mark3labs server",
				"uri", resource.URI(),
				"error", err,
			)
			return fmt.Errorf("failed to register resource %s: %w", resource.URI(), err)
		}
	}

	a.running = true
	a.lastCheck = time.Now()

	a.logger.Info("mark3labs adapter started successfully")
	return nil
}

func (a *Mark3LabsAdapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	a.logger.Info("stopping mark3labs adapter")

	// Note: mark3labs/mcp-go doesn't have explicit shutdown method
	// The server is managed by its lifecycle
	a.mcpServer = nil
	a.running = false

	a.logger.Info("mark3labs adapter stopped")
	return nil
}

func (a *Mark3LabsAdapter) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
}

func (a *Mark3LabsAdapter) Health() AdapterHealth {
	a.mu.RLock()
	defer a.mu.RUnlock()

	status := "healthy"
	var errors []string

	if !a.running {
		status = "stopped"
	}

	return AdapterHealth{
		Status:        status,
		Library:       "mark3labs",
		Version:       "0.33.0",
		ToolCount:     len(a.tools),
		ResourceCount: len(a.resources),
		LastCheck:     a.lastCheck.Format(time.RFC3339),
		Errors:        errors,
		Details: map[string]string{
			"implementation": "mark3labs/mcp-go",
			"server_status":  fmt.Sprintf("running=%v", a.running),
		},
	}
}

func (a *Mark3LabsAdapter) registerToolWithServer(tool mcpintf.Tool) error {
	// This is a simplified implementation - in a full version we'd properly
	// parse the JSON schema and create appropriate mark3labs tool options
	mcpTool := mcp.NewTool(tool.Name(),
		mcp.WithDescription(tool.Description()),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Convert arguments to JSON RawMessage
		var args []byte
		if request.Params.Arguments != nil {
			var err error
			args, err = json.Marshal(request.Params.Arguments)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid arguments: %v", err)), nil
			}
		}
		
		result, err := tool.Handler().Handle(ctx, args)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if result.IsError() {
			errorMsg := "Tool execution failed"
			if result.GetError() != nil {
				errorMsg = result.GetError().Error()
			}
			return mcp.NewToolResultError(errorMsg), nil
		}

		contents := result.GetContent()
		if len(contents) == 0 {
			return mcp.NewToolResultText(""), nil
		}

		firstContent := contents[0]
		return mcp.NewToolResultText(firstContent.GetText()), nil
	}

	a.mcpServer.AddTool(mcpTool, handler)
	return nil
}

func (a *Mark3LabsAdapter) registerResourceWithServer(resource mcpintf.Resource) error {
	mcpResource := mcp.NewResource(
		resource.URI(),
		resource.Name(),
		mcp.WithResourceDescription(resource.Description()),
		mcp.WithMIMEType(resource.MimeType()),
	)

	handler := func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		content, err := resource.Handler().Read(ctx, request.Params.URI)
		if err != nil {
			return nil, err
		}

		var results []mcp.ResourceContents
		for _, c := range content.GetContent() {
			if c.Type() == "text" {
				results = append(results, mcp.TextResourceContents{
					URI:      request.Params.URI,
					MIMEType: content.GetMimeType(),
					Text:     c.GetText(),
				})
			}
		}

		if len(results) == 0 {
			results = append(results, mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: content.GetMimeType(),
				Text:     "",
			})
		}

		return results, nil
	}

	a.mcpServer.AddResource(mcpResource, handler)
	return nil
}
