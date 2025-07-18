package echo

import (
	"context"
	"encoding/json"
	"testing"

	"mcp-server/internal/mcp"
)

func TestNewEchoTool(t *testing.T) {
	tool := NewEchoTool()
	if tool == nil {
		t.Fatal("NewEchoTool should return a valid tool instance")
	}
	
	if tool.service == nil {
		t.Fatal("EchoTool should have a valid service instance")
	}
	
	if tool.handler == nil {
		t.Fatal("EchoTool should have a valid handler instance")
	}
}

func TestEchoTool_Name(t *testing.T) {
	tool := NewEchoTool()
	name := tool.Name()
	
	expected := "echo"
	if name != expected {
		t.Errorf("Name() = %q, expected %q", name, expected)
	}
}

func TestEchoTool_Description(t *testing.T) {
	tool := NewEchoTool()
	description := tool.Description()
	
	expected := "Simple text manipulation tool for testing and demonstration"
	if description != expected {
		t.Errorf("Description() = %q, expected %q", description, expected)
	}
}

func TestEchoTool_Parameters(t *testing.T) {
	tool := NewEchoTool()
	params := tool.Parameters()
	
	if len(params) == 0 {
		t.Fatal("Parameters() should return non-empty JSON schema")
	}
	
	// Verify it's valid JSON
	var schema interface{}
	if err := json.Unmarshal(params, &schema); err != nil {
		t.Fatalf("Parameters() should return valid JSON: %v", err)
	}
}

func TestEchoTool_Handler(t *testing.T) {
	tool := NewEchoTool()
	handler := tool.Handler()
	
	if handler == nil {
		t.Fatal("Handler() should return a valid handler instance")
	}
	
	// Verify it implements the correct interface
	_, ok := handler.(mcp.ToolHandler)
	if !ok {
		t.Fatal("Handler() should return an instance that implements mcp.ToolHandler")
	}
}

func TestNewEchoHandler(t *testing.T) {
	service := NewEchoService()
	handler := NewEchoHandler(service)
	
	if handler == nil {
		t.Fatal("NewEchoHandler should return a valid handler instance")
	}
	
	if handler.service != service {
		t.Fatal("EchoHandler should use the provided service instance")
	}
}

func TestEchoHandler_Handle_ValidParameters(t *testing.T) {
	handler := NewEchoHandler(NewEchoService())
	ctx := context.Background()
	
	tests := []struct {
		name     string
		params   string
		expected string
	}{
		{
			name:     "message only",
			params:   `{"message": "hello"}`,
			expected: "hello",
		},
		{
			name:     "message with prefix",
			params:   `{"message": "world", "prefix": "hello "}`,
			expected: "hello world",
		},
		{
			name:     "message with suffix",
			params:   `{"message": "hello", "suffix": " world"}`,
			expected: "hello world",
		},
		{
			name:     "message with prefix and suffix",
			params:   `{"message": "beautiful", "prefix": "hello ", "suffix": " world"}`,
			expected: "hello beautiful world",
		},
		{
			name:     "message with uppercase",
			params:   `{"message": "hello world", "uppercase": true}`,
			expected: "HELLO WORLD",
		},
		{
			name:     "all parameters",
			params:   `{"message": "beautiful", "prefix": "hello ", "suffix": " world", "uppercase": true}`,
			expected: "HELLO BEAUTIFUL WORLD",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handler.Handle(ctx, json.RawMessage(tt.params))
			if err != nil {
				t.Fatalf("Handle() unexpected error: %v", err)
			}
			
			if result.IsError() {
				t.Fatalf("Handle() returned error result: %v", result.GetError())
			}
			
			content := result.GetContent()
			if len(content) != 1 {
				t.Fatalf("Handle() should return exactly one content item, got %d", len(content))
			}
			
			if content[0].Type() != "text" {
				t.Errorf("Handle() content type = %q, expected %q", content[0].Type(), "text")
			}
			
			if content[0].GetText() != tt.expected {
				t.Errorf("Handle() content text = %q, expected %q", content[0].GetText(), tt.expected)
			}
		})
	}
}

func TestEchoHandler_Handle_InvalidJSON(t *testing.T) {
	handler := NewEchoHandler(NewEchoService())
	ctx := context.Background()
	
	invalidParams := []string{
		`{invalid json}`,
		`{"message": }`,
		`{"message": "test", "prefix": }`,
		``,
		`null`,
	}
	
	for _, params := range invalidParams {
		t.Run("invalid_json", func(t *testing.T) {
			result, err := handler.Handle(ctx, json.RawMessage(params))
			if err != nil {
				t.Fatalf("Handle() should not return error, got: %v", err)
			}
			
			if !result.IsError() {
				t.Error("Handle() should return error result for invalid JSON")
			}
		})
	}
}

func TestEchoHandler_Handle_ValidationErrors(t *testing.T) {
	handler := NewEchoHandler(NewEchoService())
	ctx := context.Background()
	
	tests := []struct {
		name   string
		params string
	}{
		{
			name:   "empty message",
			params: `{"message": ""}`,
		},
		{
			name:   "missing message",
			params: `{"prefix": "hello"}`,
		},
		{
			name:   "message too long",
			params: `{"message": "` + string(make([]byte, 1001)) + `"}`,
		},
		{
			name:   "prefix too long",
			params: `{"message": "test", "prefix": "` + string(make([]byte, 101)) + `"}`,
		},
		{
			name:   "suffix too long",
			params: `{"message": "test", "suffix": "` + string(make([]byte, 101)) + `"}`,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handler.Handle(ctx, json.RawMessage(tt.params))
			if err != nil {
				t.Fatalf("Handle() should not return error, got: %v", err)
			}
			
			if !result.IsError() {
				t.Error("Handle() should return error result for validation failure")
			}
			
			if result.GetError() == nil {
				t.Error("Handle() error result should have an error message")
			}
		})
	}
}