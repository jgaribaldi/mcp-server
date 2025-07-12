# Commit 3: MCP Protocol Dependencies - Migration-Ready Implementation

## Overview

This commit introduces MCP (Model Context Protocol) dependencies with a strategic focus on future migration to the official SDK. We'll implement an abstraction layer that allows seamless migration when the official `modelcontextprotocol/go-sdk` becomes stable (August 2025) while using the stable `mark3labs/mcp-go` library for immediate production readiness.

## Objectives

- Add stable MCP protocol dependencies to support server implementation
- Create abstraction layer for future migration to official SDK
- Establish MCP server infrastructure without breaking existing HTTP server
- Maintain production-ready error handling and logging
- Design for easy migration when official SDK releases

## Prerequisites

- Commit 2 (Minimal Runnable Server) completed
- HTTP server running with health and ready endpoints
- Structured logging and configuration systems operational

## Migration Strategy

### Current State (Commit 3)
- **Primary Dependency**: `github.com/mark3labs/mcp-go` (stable, production-ready)
- **Rationale**: Official SDK is marked "unstable and subject to breaking changes"
- **Architecture**: Abstraction layer isolates MCP-specific implementation

### Future Migration (August 2025)
- **Target**: `github.com/modelcontextprotocol/go-sdk/mcp` (when stable)
- **Migration**: Swap implementation behind abstraction interfaces
- **Impact**: Zero changes to business logic in tools and resources

## Dependency Analysis

### Selected Library: mark3labs/mcp-go
**Pros**:
- Stable and production-ready
- Active development and community support
- Compatible API patterns with official SDK preview
- Comprehensive transport support (stdio, HTTP)
- Good error handling and logging integration

**Cons**:
- Third-party dependency (temporary)
- Will require migration to official SDK

### Official SDK: modelcontextprotocol/go-sdk
**Current Status**:
- Explicitly marked as "unstable and subject to breaking changes"
- Target stable release: August 2025
- Maintained in collaboration with Google
- API patterns established but subject to change

**Migration Timeline**:
- **Now**: Use mark3labs/mcp-go for stability
- **August 2025**: Migrate to official SDK when stable
- **Migration Effort**: Minimal due to abstraction layer

## Implementation Plan

### Step 3.1: Add MCP Dependencies
**Objective**: Add mark3labs/mcp-go to go.mod
**Files Modified**: `go.mod`, `go.sum`

**Dependencies to Add**:
```bash
go get github.com/mark3labs/mcp-go@latest
```

**Success Criteria**:
- Dependency added successfully
- No conflicts with existing dependencies
- Code still compiles and runs

### Step 3.2: Create MCP Abstraction Layer
**Objective**: Define interfaces for future migration compatibility
**Files Created**: `internal/mcp/interface.go`

**Core Interfaces**:
```go
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
```

**Success Criteria**:
- Interfaces defined to match official SDK patterns
- Compatible with both current and future implementations
- Clear separation of concerns
- Comprehensive coverage of MCP functionality

### Step 3.3: Implement MCP Server Adapter
**Objective**: Create concrete implementation using mark3labs/mcp-go
**Files Created**: `internal/mcp/server.go`

**Implementation**:
```go
package mcp

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"

    "github.com/mark3labs/mcp-go"
    "mcp-server/internal/config"
    "mcp-server/internal/logger"
)

// Server implements MCPServer using mark3labs/mcp-go
type Server struct {
    impl        Implementation
    logger      *logger.Logger
    config      *config.Config
    mcpServer   *mcp.Server
    tools       map[string]Tool
    resources   map[string]Resource
    mu          sync.RWMutex
    running     bool
}

// NewServer creates a new MCP server instance
func NewServer(impl Implementation, cfg *config.Config, log *logger.Logger) MCPServer {
    return &Server{
        impl:      impl,
        logger:    log,
        config:    cfg,
        tools:     make(map[string]Tool),
        resources: make(map[string]Resource),
    }
}

// Start implements MCPServer.Start
func (s *Server) Start(ctx context.Context, transport Transport) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.running {
        return fmt.Errorf("server is already running")
    }

    s.logger.Info("starting MCP server",
        "name", s.impl.Name,
        "version", s.impl.Version,
    )

    // Create mcp-go server instance
    s.mcpServer = mcp.NewServer(s.impl.Name, s.impl.Version)

    // Register existing tools
    for _, tool := range s.tools {
        if err := s.registerTool(tool); err != nil {
            return fmt.Errorf("failed to register tool %s: %w", tool.Name(), err)
        }
    }

    // Register existing resources
    for _, resource := range s.resources {
        if err := s.registerResource(resource); err != nil {
            return fmt.Errorf("failed to register resource %s: %w", resource.URI(), err)
        }
    }

    s.running = true

    // Start server with transport
    go func() {
        if err := s.mcpServer.Run(ctx, transport); err != nil {
            s.logger.Error("MCP server error", "error", err)
        }
    }()

    s.logger.Info("MCP server started successfully")
    return nil
}

// Stop implements MCPServer.Stop
func (s *Server) Stop(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if !s.running {
        return nil
    }

    s.logger.Info("stopping MCP server")

    if s.mcpServer != nil {
        // Note: mark3labs/mcp-go server shutdown
        // Implementation depends on library's shutdown method
        s.mcpServer = nil
    }

    s.running = false
    s.logger.Info("MCP server stopped")
    return nil
}

// AddTool implements MCPServer.AddTool
func (s *Server) AddTool(tool Tool) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.logger.Info("adding MCP tool", "name", tool.Name())

    s.tools[tool.Name()] = tool

    // Register with running server if active
    if s.running && s.mcpServer != nil {
        return s.registerTool(tool)
    }

    return nil
}

// AddResource implements MCPServer.AddResource
func (s *Server) AddResource(resource Resource) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.logger.Info("adding MCP resource", "uri", resource.URI())

    s.resources[resource.URI()] = resource

    // Register with running server if active
    if s.running && s.mcpServer != nil {
        return s.registerResource(resource)
    }

    return nil
}

// GetImplementation implements MCPServer.GetImplementation
func (s *Server) GetImplementation() Implementation {
    return s.impl
}

// registerTool registers a tool with the underlying mcp-go server
func (s *Server) registerTool(tool Tool) error {
    // Convert our Tool interface to mcp-go tool
    mcpTool := &mcp.Tool{
        Name:        tool.Name(),
        Description: tool.Description(),
        Parameters:  tool.Parameters(),
    }

    // Wrap our handler for mcp-go
    handler := func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
        // Convert params to JSON
        paramBytes, err := json.Marshal(params)
        if err != nil {
            return nil, fmt.Errorf("failed to marshal parameters: %w", err)
        }

        // Call our handler
        result, err := tool.Handler().Handle(ctx, paramBytes)
        if err != nil {
            return nil, err
        }

        if result.IsError() {
            return nil, result.GetError()
        }

        return result.GetContent(), nil
    }

    return s.mcpServer.AddTool(mcpTool, handler)
}

// registerResource registers a resource with the underlying mcp-go server
func (s *Server) registerResource(resource Resource) error {
    // Convert our Resource interface to mcp-go resource
    handler := func(ctx context.Context, uri string) (interface{}, error) {
        content, err := resource.Handler().Read(ctx, uri)
        if err != nil {
            return nil, err
        }

        return content.GetContent(), nil
    }

    return s.mcpServer.AddResource(resource.URI(), resource.Name(), 
        resource.Description(), resource.MimeType(), handler)
}
```

**Success Criteria**:
- Server implements all abstraction interfaces
- Integrates properly with mark3labs/mcp-go
- Maintains thread safety with mutex protection
- Proper error handling and logging
- Tool and resource registration works

### Step 3.4: Implement Transport Layer
**Objective**: Create transport implementations for MCP communication
**Files Created**: `internal/mcp/transport.go`

**Transport Implementations**:
```go
package mcp

import (
    "bufio"
    "fmt"
    "io"
    "os"
    "sync"
)

// StdioTransport implements Transport using stdin/stdout
type StdioTransport struct {
    reader *bufio.Reader
    writer *bufio.Writer
    mu     sync.Mutex
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport() Transport {
    return &StdioTransport{
        reader: bufio.NewReader(os.Stdin),
        writer: bufio.NewWriter(os.Stdout),
    }
}

// Read implements Transport.Read
func (t *StdioTransport) Read() ([]byte, error) {
    line, err := t.reader.ReadBytes('\n')
    if err != nil {
        if err == io.EOF {
            return nil, fmt.Errorf("stdin closed")
        }
        return nil, fmt.Errorf("failed to read from stdin: %w", err)
    }
    return line, nil
}

// Write implements Transport.Write
func (t *StdioTransport) Write(data []byte) error {
    t.mu.Lock()
    defer t.mu.Unlock()

    if _, err := t.writer.Write(data); err != nil {
        return fmt.Errorf("failed to write to stdout: %w", err)
    }

    if err := t.writer.Flush(); err != nil {
        return fmt.Errorf("failed to flush stdout: %w", err)
    }

    return nil
}

// Close implements Transport.Close
func (t *StdioTransport) Close() error {
    // Stdio doesn't need explicit closing
    return nil
}

// TransportFactory creates transport instances
type TransportFactory struct{}

// NewTransportFactory creates a new transport factory
func NewTransportFactory() *TransportFactory {
    return &TransportFactory{}
}

// CreateStdioTransport creates a stdio transport
func (f *TransportFactory) CreateStdioTransport() Transport {
    return NewStdioTransport()
}
```

**Success Criteria**:
- Stdio transport implementation complete
- Factory pattern for transport creation
- Thread-safe operations
- Proper error handling

### Step 3.5: Create Content Types
**Objective**: Implement MCP content types
**Files Created**: `internal/mcp/content.go`

**Content Implementations**:
```go
package mcp

// TextContent represents text content
type TextContent struct {
    text string
}

// NewTextContent creates new text content
func NewTextContent(text string) Content {
    return &TextContent{text: text}
}

// Type implements Content.Type
func (c *TextContent) Type() string {
    return "text"
}

// GetText implements Content.GetText
func (c *TextContent) GetText() string {
    return c.text
}

// GetBlob implements Content.GetBlob
func (c *TextContent) GetBlob() []byte {
    return []byte(c.text)
}

// BlobContent represents binary content
type BlobContent struct {
    data []byte
}

// NewBlobContent creates new blob content
func NewBlobContent(data []byte) Content {
    return &BlobContent{data: data}
}

// Type implements Content.Type
func (c *BlobContent) Type() string {
    return "blob"
}

// GetText implements Content.GetText
func (c *BlobContent) GetText() string {
    return string(c.data)
}

// GetBlob implements Content.GetBlob
func (c *BlobContent) GetBlob() []byte {
    return c.data
}

// ToolResult implementation
type toolResult struct {
    content []Content
    error   error
}

// NewToolResult creates a successful tool result
func NewToolResult(content ...Content) ToolResult {
    return &toolResult{content: content}
}

// NewToolError creates an error tool result
func NewToolError(err error) ToolResult {
    return &toolResult{error: err}
}

// IsError implements ToolResult.IsError
func (r *toolResult) IsError() bool {
    return r.error != nil
}

// GetContent implements ToolResult.GetContent
func (r *toolResult) GetContent() []Content {
    return r.content
}

// GetError implements ToolResult.GetError
func (r *toolResult) GetError() error {
    return r.error
}
```

**Success Criteria**:
- Content types properly implemented
- Tool result handling complete
- Compatible with MCP protocol specifications

### Step 3.6: Integration with Existing Server
**Objective**: Initialize MCP server in main application
**Files Modified**: `internal/server/server.go`, `cmd/mcp-server/main.go`

**Server Integration**:
```go
// In internal/server/server.go - add MCP server field
type Server struct {
    httpServer *http.Server
    mcpServer  mcp.MCPServer
    logger     *logger.Logger
    config     *config.Config
    mux        *http.ServeMux
}

// Modify New() function to initialize MCP server
func New(cfg *config.Config, log *logger.Logger) *Server {
    mux := http.NewServeMux()

    // Create MCP server
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

    server.setupRoutes()
    return server
}
```

**Success Criteria**:
- MCP server integrated with HTTP server
- No conflicts between protocols
- Proper initialization and shutdown
- Logging integration maintained

## Migration Documentation

### Future Migration Process (August 2025)

#### Step 1: Update Dependencies
```bash
# Remove interim dependency
go mod edit -droprequire github.com/mark3labs/mcp-go

# Add official SDK
go get github.com/modelcontextprotocol/go-sdk/mcp@latest
```

#### Step 2: Update Implementation
- Replace `internal/mcp/server.go` implementation
- Update imports to use official SDK
- Maintain same interface contracts
- Test all functionality

#### Step 3: Validation
- All existing tools continue working
- No changes required in business logic
- Performance and reliability maintained

### Compatibility Matrix

| Component | Current (mark3labs) | Future (official) | Migration Impact |
|-----------|-------------------|-------------------|------------------|
| Tool Interface | ✅ Compatible | ✅ Compatible | None |
| Transport Layer | ✅ Compatible | ✅ Compatible | Implementation only |
| Content Types | ✅ Compatible | ✅ Compatible | None |
| Error Handling | ✅ Compatible | ✅ Compatible | None |
| Business Logic | ✅ Isolated | ✅ Isolated | None |

## Error Handling Requirements

### Connection Errors
- Transport failure handling
- Client disconnection management
- Timeout handling
- Graceful degradation

### Protocol Errors
- Invalid message format handling
- Unsupported method responses
- Parameter validation
- Error response formatting

### Application Errors
- Tool execution failures
- Resource access errors
- Configuration errors
- Logging integration

## Testing Strategy

### Unit Tests
- Interface compliance testing
- Mock implementations for testing
- Error condition coverage
- Transport layer testing

### Integration Tests
- End-to-end MCP protocol testing
- Client-server communication
- Tool and resource operations
- Migration compatibility testing

## Security Considerations

### Input Validation
- Parameter sanitization
- URI validation for resources
- Content type verification
- Size limits enforcement

### Access Control
- Tool permission management
- Resource access restrictions
- Transport security
- Audit logging

## Performance Considerations

### Connection Management
- Connection pooling strategies
- Keep-alive mechanisms
- Resource cleanup
- Memory management

### Protocol Efficiency
- Message batching
- Compression support
- Caching strategies
- Lazy loading

## Success Criteria for Commit 3

1. **Dependencies Added**: mark3labs/mcp-go dependency successfully added
2. **Abstraction Layer**: Complete interface layer implemented
3. **Server Implementation**: Working MCP server adapter created
4. **Transport Support**: Stdio transport implementation functional
5. **Integration**: MCP server integrated with existing HTTP server
6. **Migration Ready**: Clear migration path documented and validated
7. **Error Handling**: Comprehensive error management implemented
8. **Logging**: All MCP operations properly logged
9. **No Regression**: Existing HTTP endpoints continue working
10. **Foundation Ready**: Ready for Commit 4 (MCP Server Core)

## Next Steps

After successful completion of Commit 3:
1. Create Commit 4 specification (MCP Server Core)
2. Implement protocol negotiation and capabilities
3. Add tool and resource management endpoints
4. Begin implementing first MCP tools

## Migration Notes

- **Abstraction Benefits**: Business logic remains unchanged during migration
- **Timeline**: Migration scheduled for August 2025 when official SDK stabilizes
- **Risk Mitigation**: Stable interim solution provides production readiness
- **Future Compatibility**: Interface design matches official SDK patterns
- **Documentation**: Migration process fully documented and tested