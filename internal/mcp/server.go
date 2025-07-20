package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"mcp-server/internal/config"
	"mcp-server/internal/logger"
)

// Server implements MCPServer using mark3labs/mcp-go
type Server struct {
	impl        Implementation
	logger      *logger.Logger
	config      *config.Config
	mcpServer   *server.MCPServer
	tools       map[string]Tool
	resources   map[string]Resource
	mu          sync.RWMutex
	running     bool
	transport   Transport
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

	s.mcpServer = server.NewMCPServer(
		s.impl.Name,
		s.impl.Version,
		server.WithToolCapabilities(false),
		server.WithResourceCapabilities(true, true),
		server.WithRecovery(),
	)

	s.transport = transport

	for _, tool := range s.tools {
		if err := s.registerTool(tool); err != nil {
			return fmt.Errorf("failed to register tool %s: %w", tool.Name(), err)
		}
	}

	for _, resource := range s.resources {
		if err := s.registerResource(resource); err != nil {
			return fmt.Errorf("failed to register resource %s: %w", resource.URI(), err)
		}
	}

	s.running = true

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

	// Note: mark3labs/mcp-go doesn't have explicit shutdown method
	// The server stops when context is cancelled or transport closes
	if s.transport != nil {
		if err := s.transport.Close(); err != nil {
			s.logger.Error("error closing transport", "error", err)
		}
	}

	s.mcpServer = nil
	s.transport = nil
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

	if s.running && s.mcpServer != nil {
		return s.registerResource(resource)
	}

	return nil
}

// GetImplementation implements MCPServer.GetImplementation
func (s *Server) GetImplementation() Implementation {
	return s.impl
}

// registerTool registers a tool with the underlying mark3labs MCP server
func (s *Server) registerTool(tool Tool) error {
	options := []mcp.ToolOption{
		mcp.WithDescription(tool.Description()),
	}

	// Parse and add parameters if available
	if tool.Parameters() != nil {
		var params map[string]interface{}
		if err := json.Unmarshal(tool.Parameters(), &params); err != nil {
			return fmt.Errorf("failed to parse tool parameters: %w", err)
		}

		// Add parameters to tool (simplified - real implementation would parse JSON schema)
		if properties, ok := params["properties"].(map[string]interface{}); ok {
			for name, prop := range properties {
				if propMap, ok := prop.(map[string]interface{}); ok {
					description := ""
					if desc, ok := propMap["description"].(string); ok {
						description = desc
					}
					required := false
					if req, ok := params["required"].([]interface{}); ok {
						for _, r := range req {
							if r == name {
								required = true
								break
							}
						}
					}

					if required {
						options = append(options, mcp.WithString(name, mcp.Required(), mcp.Description(description)))
					} else {
						options = append(options, mcp.WithString(name, mcp.Description(description)))
					}
				}
			}
		}
	}

	mcpTool := mcp.NewTool(tool.Name(), options...)

	handler := s.createToolHandlerAdapter(tool.Handler())

	s.mcpServer.AddTool(mcpTool, handler)

	return nil
}

// registerResource registers a resource with the underlying mark3labs MCP server
func (s *Server) registerResource(resource Resource) error {
	// Create mark3labs resource definition
	mcpResource := mcp.NewResource(
		resource.URI(),
		resource.Name(),
		mcp.WithResourceDescription(resource.Description()),
		mcp.WithMIMEType(resource.MimeType()),
	)

	// Create handler adapter
	handler := s.createResourceHandlerAdapter(resource.Handler())

	// Register with mark3labs server
	s.mcpServer.AddResource(mcpResource, handler)

	return nil
}

// createToolHandlerAdapter adapts our ToolHandler interface to mark3labs handler
func (s *Server) createToolHandlerAdapter(handler ToolHandler) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.Info("executing tool",
			"name", request.Params.Name,
		)

		// Convert arguments to our format
		var args json.RawMessage
		if request.Params.Arguments != nil {
			argsBytes, err := json.Marshal(request.Params.Arguments)
			if err != nil {
				s.logger.Error("failed to marshal tool arguments", "error", err)
				return mcp.NewToolResultError(fmt.Sprintf("Invalid arguments: %v", err)), nil
			}
			args = argsBytes
		}

		// Call our handler
		result, err := handler.Handle(ctx, args)
		if err != nil {
			s.logger.Error("tool execution failed", "error", err)
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Handle error results
		if result.IsError() {
			errorMsg := "Tool execution failed"
			if result.GetError() != nil {
				errorMsg = result.GetError().Error()
			}
			s.logger.Error("tool returned error", "error", errorMsg)
			return mcp.NewToolResultError(errorMsg), nil
		}

		// Convert successful result
		contents := result.GetContent()
		if len(contents) == 0 {
			return mcp.NewToolResultText(""), nil
		}

		// For now, return the first content item as text
		// In a full implementation, we'd handle multiple content types
		firstContent := contents[0]
		if firstContent.Type() == "text" {
			return mcp.NewToolResultText(firstContent.GetText()), nil
		}

		// For blob content, convert to text representation
		return mcp.NewToolResultText(firstContent.GetText()), nil
	}
}

// createResourceHandlerAdapter adapts our ResourceHandler interface to mark3labs handler
func (s *Server) createResourceHandlerAdapter(handler ResourceHandler) func(context.Context, mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		s.logger.Info("reading resource",
			"uri", request.Params.URI,
		)

		// Call our handler
		content, err := handler.Read(ctx, request.Params.URI)
		if err != nil {
			s.logger.Error("resource read failed", "error", err)
			return nil, err
		}

		// Convert our ResourceContent to mark3labs format
		var results []mcp.ResourceContents

		for _, c := range content.GetContent() {
			switch c.Type() {
			case "text":
				results = append(results, mcp.TextResourceContents{
					URI:      request.Params.URI,
					MIMEType: content.GetMimeType(),
					Text:     c.GetText(),
				})
			case "blob":
				// For blob content, encode as base64
				blob := c.GetBlob()
				encoded := base64.StdEncoding.EncodeToString(blob)
				results = append(results, mcp.BlobResourceContents{
					URI:      request.Params.URI,
					MIMEType: content.GetMimeType(),
					Blob:     encoded,
				})
			}
		}

		if len(results) == 0 {
			// Return empty content if no content items
			results = append(results, mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: content.GetMimeType(),
				Text:     "",
			})
		}

		return results, nil
	}
}

// Serve starts the MCP server with the configured transport
func (s *Server) Serve(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return fmt.Errorf("server not started")
	}

	if s.mcpServer == nil {
		return fmt.Errorf("MCP server not initialized")
	}

	s.logger.Info("serving MCP requests")

	// For now, we'll use stdio transport as that's what mark3labs supports
	// In a full implementation, we'd adapt our transport interface
	return server.ServeStdio(s.mcpServer)
}