package echo

import (
	"context"
	"encoding/json"
	"fmt"

	"mcp-server/internal/mcp"
)

// EchoParams represents the parameters for the Echo tool
type EchoParams struct {
	Message   string `json:"message"`
	Prefix    string `json:"prefix,omitempty"`
	Suffix    string `json:"suffix,omitempty"`
	Uppercase bool   `json:"uppercase,omitempty"`
}

// EchoTool implements the mcp.Tool interface for the Echo tool
type EchoTool struct {
	service *EchoService
	handler *EchoHandler
}

// NewEchoTool creates a new EchoTool instance
func NewEchoTool() *EchoTool {
	service := NewEchoService()
	handler := NewEchoHandler(service)
	return &EchoTool{
		service: service,
		handler: handler,
	}
}

// Name implements mcp.Tool.Name
func (t *EchoTool) Name() string {
	return "echo"
}

// Description implements mcp.Tool.Description
func (t *EchoTool) Description() string {
	return "Simple text manipulation tool for testing and demonstration"
}

// Parameters implements mcp.Tool.Parameters
func (t *EchoTool) Parameters() json.RawMessage {
	schema := `{
		"type": "object",
		"properties": {
			"message": {
				"type": "string",
				"minLength": 1,
				"maxLength": 1000,
				"description": "The message to transform"
			},
			"prefix": {
				"type": "string",
				"maxLength": 100,
				"description": "Optional prefix to add before the message"
			},
			"suffix": {
				"type": "string",
				"maxLength": 100,
				"description": "Optional suffix to add after the message"
			},
			"uppercase": {
				"type": "boolean",
				"description": "Whether to convert the result to uppercase"
			}
		},
		"required": ["message"]
	}`
	return json.RawMessage(schema)
}

// Handler implements mcp.Tool.Handler
func (t *EchoTool) Handler() mcp.ToolHandler {
	return t.handler
}

// EchoHandler implements the mcp.ToolHandler interface
type EchoHandler struct {
	service *EchoService
}

// NewEchoHandler creates a new EchoHandler instance
func NewEchoHandler(service *EchoService) *EchoHandler {
	return &EchoHandler{
		service: service,
	}
}

// Handle implements mcp.ToolHandler.Handle
func (h *EchoHandler) Handle(ctx context.Context, params json.RawMessage) (mcp.ToolResult, error) {
	var echoParams EchoParams
	
	// Parse JSON parameters
	if err := json.Unmarshal(params, &echoParams); err != nil {
		return mcp.NewToolError(fmt.Errorf("invalid parameters: %w", err)), nil
	}
	
	// Validate parameters using business logic
	if err := h.service.ValidateAll(echoParams.Message, echoParams.Prefix, echoParams.Suffix); err != nil {
		return mcp.NewToolError(err), nil
	}
	
	// Transform message using business logic
	result := h.service.Transform(echoParams.Message, echoParams.Prefix, echoParams.Suffix, echoParams.Uppercase)
	
	// Create and return successful result
	content := mcp.NewTextContent(result)
	return mcp.NewToolResult(content), nil
}