package echo

import (
	"context"
	"encoding/json"
	"fmt"

	"mcp-server/internal/mcp"
)

type EchoParams struct {
	Message   string `json:"message"`
	Prefix    string `json:"prefix,omitempty"`
	Suffix    string `json:"suffix,omitempty"`
	Uppercase bool   `json:"uppercase,omitempty"`
}

type EchoTool struct {
	service *EchoService
	handler *EchoHandler
}

func NewEchoTool() *EchoTool {
	service := NewEchoService()
	handler := NewEchoHandler(service)
	return &EchoTool{
		service: service,
		handler: handler,
	}
}

func (t *EchoTool) Name() string {
	return "echo"
}

func (t *EchoTool) Description() string {
	return "Simple text manipulation tool for testing and demonstration"
}

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

func (t *EchoTool) Handler() mcp.ToolHandler {
	return t.handler
}

type EchoHandler struct {
	service *EchoService
}

func NewEchoHandler(service *EchoService) *EchoHandler {
	return &EchoHandler{
		service: service,
	}
}

func (h *EchoHandler) Handle(ctx context.Context, params json.RawMessage) (mcp.ToolResult, error) {
	var echoParams EchoParams
	
	if err := json.Unmarshal(params, &echoParams); err != nil {
		return mcp.NewToolError(fmt.Errorf("invalid parameters: %w", err)), nil
	}
	
	if err := h.service.ValidateAll(echoParams.Message, echoParams.Prefix, echoParams.Suffix); err != nil {
		return mcp.NewToolError(err), nil
	}
	
	result := h.service.Transform(echoParams.Message, echoParams.Prefix, echoParams.Suffix, echoParams.Uppercase)
	
	content := mcp.NewTextContent(result)
	return mcp.NewToolResult(content), nil
}