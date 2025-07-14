package adapters

import (
	"context"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
)

// LibraryAdapter abstracts the underlying MCP library implementation
// This allows us to switch between different MCP libraries (mark3labs, official SDK, etc.)
// without changing the business logic in our tool registry
type LibraryAdapter interface {
	// Tool management
	RegisterTool(tool mcp.Tool) error
	UnregisterTool(name string) error
	GetTool(name string) (mcp.Tool, error)
	ListTools() []string

	// Resource management  
	RegisterResource(resource mcp.Resource) error
	UnregisterResource(uri string) error
	GetResource(uri string) (mcp.Resource, error)
	ListResources() []string

	// Lifecycle management
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool

	// Health and status
	Health() AdapterHealth
}

// AdapterHealth represents the health status of the library adapter
type AdapterHealth struct {
	Status        string            `json:"status"`        // "healthy", "degraded", "unhealthy"
	Library       string            `json:"library"`       // e.g., "mark3labs", "official"
	Version       string            `json:"version"`       // Library version
	ToolCount     int               `json:"tool_count"`    // Number of registered tools
	ResourceCount int               `json:"resource_count"` // Number of registered resources
	LastCheck     string            `json:"last_check"`    // RFC3339 timestamp
	Errors        []string          `json:"errors,omitempty"` // Any errors encountered
	Details       map[string]string `json:"details,omitempty"` // Additional details
}

// AdapterFactory creates library adapter instances
type AdapterFactory interface {
	CreateAdapter(cfg *config.Config, log *logger.Logger) (LibraryAdapter, error)
	SupportedLibraries() []string
	DefaultLibrary() string
}