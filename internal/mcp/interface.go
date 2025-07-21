package mcp

import (
	"context"
	"encoding/json"
)

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

type Tool interface {
	Name() string
	Description() string
	Parameters() json.RawMessage // JSON schema
	Handler() ToolHandler
}

type ToolHandler interface {
	Handle(ctx context.Context, params json.RawMessage) (ToolResult, error)
}

type ToolResult interface {
	IsError() bool
	GetContent() []Content
	GetError() error
}

type Resource interface {
	URI() string
	Name() string
	Description() string
	MimeType() string
	Handler() ResourceHandler
}

type ResourceHandler interface {
	Read(ctx context.Context, uri string) (ResourceContent, error)
}

type ResourceContent interface {
	GetContent() []Content
	GetMimeType() string
}

type Transport interface {
	Read() ([]byte, error)
	Write(data []byte) error
	Close() error
}

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

type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}
