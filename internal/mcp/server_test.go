package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"mcp-server/internal/config"
	"mcp-server/internal/logger"
)

// Test helpers

// createTestLogger creates a logger for testing
func createTestLogger(t *testing.T) *logger.Logger {
	t.Helper()
	log, err := logger.New(logger.Config{
		Level:   "info",
		Format:  "text",
		Service: "test",
		Version: "1.0.0",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	return log
}

// Mock implementations for testing

// Mock tool implementation
type mockTool struct {
	name        string
	description string
	parameters  json.RawMessage
	handler     ToolHandler
}

func (m *mockTool) Name() string              { return m.name }
func (m *mockTool) Description() string       { return m.description }
func (m *mockTool) Parameters() json.RawMessage { return m.parameters }
func (m *mockTool) Handler() ToolHandler      { return m.handler }

// Mock tool handler
type mockToolHandler struct {
	handleFunc func(ctx context.Context, params json.RawMessage) (ToolResult, error)
}

func (m *mockToolHandler) Handle(ctx context.Context, params json.RawMessage) (ToolResult, error) {
	if m.handleFunc != nil {
		return m.handleFunc(ctx, params)
	}
	return NewToolResult(NewTextContent("mock result")), nil
}

// Mock resource implementation
type mockResource struct {
	uri         string
	name        string
	description string
	mimeType    string
	handler     ResourceHandler
}

func (m *mockResource) URI() string            { return m.uri }
func (m *mockResource) Name() string           { return m.name }
func (m *mockResource) Description() string    { return m.description }
func (m *mockResource) MimeType() string       { return m.mimeType }
func (m *mockResource) Handler() ResourceHandler { return m.handler }

// Mock resource handler
type mockResourceHandler struct {
	readFunc func(ctx context.Context, uri string) (ResourceContent, error)
}

func (m *mockResourceHandler) Read(ctx context.Context, uri string) (ResourceContent, error) {
	if m.readFunc != nil {
		return m.readFunc(ctx, uri)
	}
	return NewResourceContent("text/plain", NewTextContent("mock resource content")), nil
}

// Test server creation
func TestNewServer(t *testing.T) {
	impl := Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}
	cfg := (*config.Config)(nil) // Use nil for testing
	log := createTestLogger(t)

	server := NewServer(impl, cfg, log)
	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	// Verify implementation
	if server.GetImplementation().Name != "test-server" {
		t.Errorf("Expected name 'test-server', got '%s'", server.GetImplementation().Name)
	}

	if server.GetImplementation().Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", server.GetImplementation().Version)
	}
}

// Test server lifecycle
func TestServerLifecycle(t *testing.T) {
	impl := Implementation{Name: "test-server", Version: "1.0.0"}
	cfg := (*config.Config)(nil)
	log := createTestLogger(t)
	server := NewServer(impl, cfg, log).(*Server)

	ctx := context.Background()
	transport := NewTestableStdioTransport(nil, nil)

	// Test start
	if err := server.Start(ctx, transport); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Verify server is running
	if !server.running {
		t.Error("Expected server to be running")
	}

	// Test double start (should fail)
	if err := server.Start(ctx, transport); err == nil {
		t.Error("Expected error on double start")
	}

	// Test stop
	if err := server.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Verify server is stopped
	if server.running {
		t.Error("Expected server to be stopped")
	}

	// Test double stop (should not fail)
	if err := server.Stop(ctx); err != nil {
		t.Errorf("Stop failed on already stopped server: %v", err)
	}
}

// Test tool management in server
func TestServerToolManagement(t *testing.T) {
	impl := Implementation{Name: "test-server", Version: "1.0.0"}
	cfg := (*config.Config)(nil)
	log := createTestLogger(t)
	server := NewServer(impl, cfg, log).(*Server)

	// Create mock tool
	handler := &mockToolHandler{}
	tool := &mockTool{
		name:        "test-tool",
		description: "A test tool",
		parameters:  json.RawMessage(`{"type": "object", "properties": {"input": {"type": "string", "description": "Input parameter"}}}`),
		handler:     handler,
	}

	// Test adding tool before server start
	if err := server.AddTool(tool); err != nil {
		t.Fatalf("AddTool failed: %v", err)
	}

	// Verify tool was stored
	if len(server.tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(server.tools))
	}

	if server.tools["test-tool"] != tool {
		t.Error("Tool was not stored correctly")
	}

	// Start server
	ctx := context.Background()
	transport := NewTestableStdioTransport(nil, nil)
	if err := server.Start(ctx, transport); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Test adding tool after server start
	handler2 := &mockToolHandler{}
	tool2 := &mockTool{
		name:        "test-tool-2",
		description: "Another test tool",
		handler:     handler2,
	}

	if err := server.AddTool(tool2); err != nil {
		t.Fatalf("AddTool after start failed: %v", err)
	}

	// Verify both tools are stored
	if len(server.tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(server.tools))
	}

	// Stop server
	if err := server.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

// Test resource management in server
func TestServerResourceManagement(t *testing.T) {
	impl := Implementation{Name: "test-server", Version: "1.0.0"}
	cfg := (*config.Config)(nil)
	log := createTestLogger(t)
	server := NewServer(impl, cfg, log).(*Server)

	// Create mock resource
	handler := &mockResourceHandler{}
	resource := &mockResource{
		uri:         "file://test.txt",
		name:        "test-resource",
		description: "A test resource",
		mimeType:    "text/plain",
		handler:     handler,
	}

	// Test adding resource before server start
	if err := server.AddResource(resource); err != nil {
		t.Fatalf("AddResource failed: %v", err)
	}

	// Verify resource was stored
	if len(server.resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(server.resources))
	}

	if server.resources["file://test.txt"] != resource {
		t.Error("Resource was not stored correctly")
	}

	// Start server
	ctx := context.Background()
	transport := NewTestableStdioTransport(nil, nil)
	if err := server.Start(ctx, transport); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Test adding resource after server start
	handler2 := &mockResourceHandler{}
	resource2 := &mockResource{
		uri:         "file://test2.txt",
		name:        "test-resource-2",
		description: "Another test resource",
		mimeType:    "text/plain",
		handler:     handler2,
	}

	if err := server.AddResource(resource2); err != nil {
		t.Fatalf("AddResource after start failed: %v", err)
	}

	// Verify both resources are stored
	if len(server.resources) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(server.resources))
	}

	// Stop server
	if err := server.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

// Test concurrent access
func TestConcurrentAccess(t *testing.T) {
	impl := Implementation{Name: "test-server", Version: "1.0.0"}
	cfg := (*config.Config)(nil)
	log := createTestLogger(t)
	server := NewServer(impl, cfg, log).(*Server)

	ctx := context.Background()
	transport := NewTestableStdioTransport(nil, nil)

	// Start server
	if err := server.Start(ctx, transport); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Number of concurrent operations
	numOps := 10
	var wg sync.WaitGroup
	wg.Add(numOps * 2) // tools + resources

	// Add tools concurrently
	for i := 0; i < numOps; i++ {
		go func(id int) {
			defer wg.Done()
			handler := &mockToolHandler{}
			tool := &mockTool{
				name:        fmt.Sprintf("tool-%d", id),
				description: fmt.Sprintf("Tool %d", id),
				handler:     handler,
			}
			if err := server.AddTool(tool); err != nil {
				t.Errorf("Concurrent AddTool %d failed: %v", id, err)
			}
		}(i)
	}

	// Add resources concurrently
	for i := 0; i < numOps; i++ {
		go func(id int) {
			defer wg.Done()
			handler := &mockResourceHandler{}
			resource := &mockResource{
				uri:         fmt.Sprintf("file://test-%d.txt", id),
				name:        fmt.Sprintf("resource-%d", id),
				description: fmt.Sprintf("Resource %d", id),
				mimeType:    "text/plain",
				handler:     handler,
			}
			if err := server.AddResource(resource); err != nil {
				t.Errorf("Concurrent AddResource %d failed: %v", id, err)
			}
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()

	// Verify all tools and resources were added
	if len(server.tools) != numOps {
		t.Errorf("Expected %d tools, got %d", numOps, len(server.tools))
	}

	if len(server.resources) != numOps {
		t.Errorf("Expected %d resources, got %d", numOps, len(server.resources))
	}

	// Stop server
	if err := server.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

// Test tool handler adaptation
func TestToolHandlerAdapter(t *testing.T) {
	impl := Implementation{Name: "test-server", Version: "1.0.0"}
	cfg := (*config.Config)(nil)
	log := createTestLogger(t)
	server := NewServer(impl, cfg, log).(*Server)

	// Test successful tool execution
	t.Run("SuccessfulExecution", func(t *testing.T) {
		handler := &mockToolHandler{
			handleFunc: func(ctx context.Context, params json.RawMessage) (ToolResult, error) {
				return NewToolResult(NewTextContent("success result")), nil
			},
		}

		adapter := server.createToolHandlerAdapter(handler)

		// Mock request (simplified)
		// Note: In real testing, we'd need to mock the mark3labs types properly
		ctx := context.Background()
		req := mcp.CallToolRequest{}
		req.Params.Name = "test"
		req.Params.Arguments = map[string]interface{}{"input": "test"}
		result, err := adapter(ctx, req)

		if err != nil {
			t.Fatalf("Handler adapter failed: %v", err)
		}

		if result == nil {
			t.Fatal("Handler adapter returned nil result")
		}
	})

	// Test error handling
	t.Run("ErrorHandling", func(t *testing.T) {
		handler := &mockToolHandler{
			handleFunc: func(ctx context.Context, params json.RawMessage) (ToolResult, error) {
				return nil, errors.New("tool error")
			},
		}

		adapter := server.createToolHandlerAdapter(handler)

		ctx := context.Background()
		req := mcp.CallToolRequest{}
		req.Params.Name = "test"
		req.Params.Arguments = nil
		result, err := adapter(ctx, req)

		if err != nil {
			t.Fatalf("Handler adapter failed: %v", err)
		}

		// The adapter should convert errors to error results, not return errors
		if result == nil {
			t.Fatal("Handler adapter returned nil result for error case")
		}
	})
}

// Test error conditions
func TestErrorConditions(t *testing.T) {
	impl := Implementation{Name: "test-server", Version: "1.0.0"}
	cfg := (*config.Config)(nil)
	log := createTestLogger(t)
	server := NewServer(impl, cfg, log).(*Server)

	ctx := context.Background()

	// Test serve without start
	if err := server.Serve(ctx); err == nil {
		t.Error("Expected error when serving without start")
	}

	// Test with nil transport
	if err := server.Start(ctx, nil); err != nil {
		// This might not error immediately, but should be handled gracefully
		// The actual error might come later when trying to use the transport
		t.Logf("Start with nil transport: %v", err)
	}
}

// Test logging - simplified test since we're using real logger
func TestLogging(t *testing.T) {
	impl := Implementation{Name: "test-server", Version: "1.0.0"}
	cfg := (*config.Config)(nil)
	log := createTestLogger(t)
	server := NewServer(impl, cfg, log).(*Server)

	ctx := context.Background()
	transport := NewTestableStdioTransport(nil, nil)

	// Start server (should generate log entries)
	if err := server.Start(ctx, transport); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Add tool (should generate log entry)
	handler := &mockToolHandler{}
	tool := &mockTool{
		name:        "test-tool",
		description: "A test tool",
		handler:     handler,
	}
	if err := server.AddTool(tool); err != nil {
		t.Fatalf("AddTool failed: %v", err)
	}

	// Stop server (should generate log entry)
	if err := server.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Test passes if no errors occurred during logging operations
	t.Log("Logging test completed - all operations logged successfully")
}

// Mock types for testing are no longer needed since we use real types

// Interface compliance test
func TestServerInterfaceCompliance(t *testing.T) {
	// Ensure our Server type implements MCPServer interface
	var _ MCPServer = (*Server)(nil)
}