package mcp

import (
	"context"
	"encoding/json"
)

// MCPServer represents the core MCP server functionality
type MCPServer interface {
	// Server lifecycle
	Start(ctx context.Context, transport Transport) error
	Stop(ctx context.Context) error

	// Tool and resource management
	AddTool(tool Tool) error
	AddResource(resource Resource) error

	// Server information
	GetImplementation() Implementation
}

// Tool represents an MCP tool that can be called by clients
type Tool interface {
	Name() string
	Description() string
	Parameters() json.RawMessage // JSON schema
	Handler() ToolHandler
}

// ToolHandler processes tool execution requests
type ToolHandler interface {
	Handle(ctx context.Context, params json.RawMessage) (ToolResult, error)
}

// ToolResult represents the result of a tool execution
type ToolResult interface {
	IsError() bool
	GetContent() []Content
	GetError() error
}

// Resource represents an MCP resource that can be accessed by clients
type Resource interface {
	URI() string
	Name() string
	Description() string
	MimeType() string
	Handler() ResourceHandler
}

// ResourceHandler processes resource access requests
type ResourceHandler interface {
	Read(ctx context.Context, uri string) (ResourceContent, error)
}

// ResourceContent represents the content of a resource
type ResourceContent interface {
	GetContent() []Content
	GetMimeType() string
}

// Transport handles communication between client and server
type Transport interface {
	Read() ([]byte, error)
	Write(data []byte) error
	Close() error
}

// Content represents MCP content (text, blob, etc.)
type Content interface {
	Type() string
	GetText() string
	GetBlob() []byte
}

// Implementation contains server metadata
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// CallToolParams represents parameters for tool calls
type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}