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

type mockToolHandler struct {
	handleFunc func(ctx context.Context, params json.RawMessage) (ToolResult, error)
}

func (m *mockToolHandler) Handle(ctx context.Context, params json.RawMessage) (ToolResult, error) {
	if m.handleFunc != nil {
		return m.handleFunc(ctx, params)
	}
	return &ToolResultImpl{Content: []Content{&TextContent{Text: "mock result"}}, IsErrorFlag: false}, nil
}

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

type mockResourceHandler struct {
	readFunc func(ctx context.Context, uri string) (ResourceContent, error)
}

func (m *mockResourceHandler) Read(ctx context.Context, uri string) (ResourceContent, error) {
	if m.readFunc != nil {
		return m.readFunc(ctx, uri)
	}
	return NewResourceContent("text/plain", &TextContent{Text: "mock resource content"}), nil
}

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

	if server.GetImplementation().Name != "test-server" {
		t.Errorf("Expected name 'test-server', got '%s'", server.GetImplementation().Name)
	}

	if server.GetImplementation().Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", server.GetImplementation().Version)
	}
}

func TestServerLifecycle(t *testing.T) {
	impl := Implementation{Name: "test-server", Version: "1.0.0"}
	cfg := (*config.Config)(nil)
	log := createTestLogger(t)
	server := NewServer(impl, cfg, log).(*Server)

	ctx := context.Background()
	transport := NewTestableStdioTransport(nil, nil)

	if err := server.Start(ctx, transport); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !server.running {
		t.Error("Expected server to be running")
	}

	if err := server.Start(ctx, transport); err == nil {
		t.Error("Expected error on double start")
	}

	if err := server.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if server.running {
		t.Error("Expected server to be stopped")
	}

	if err := server.Stop(ctx); err != nil {
		t.Errorf("Stop failed on already stopped server: %v", err)
	}
}

func TestServerToolManagement(t *testing.T) {
	impl := Implementation{Name: "test-server", Version: "1.0.0"}
	cfg := (*config.Config)(nil)
	log := createTestLogger(t)
	server := NewServer(impl, cfg, log).(*Server)

	handler := &mockToolHandler{}
	tool := &mockTool{
		name:        "test-tool",
		description: "A test tool",
		parameters:  json.RawMessage(`{"type": "object", "properties": {"input": {"type": "string", "description": "Input parameter"}}}`),
		handler:     handler,
	}

	if err := server.AddTool(tool); err != nil {
		t.Fatalf("AddTool failed: %v", err)
	}

	if len(server.tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(server.tools))
	}

	if server.tools["test-tool"] != tool {
		t.Error("Tool was not stored correctly")
	}

	ctx := context.Background()
	transport := NewTestableStdioTransport(nil, nil)
	if err := server.Start(ctx, transport); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	handler2 := &mockToolHandler{}
	tool2 := &mockTool{
		name:        "test-tool-2",
		description: "Another test tool",
		handler:     handler2,
	}

	if err := server.AddTool(tool2); err != nil {
		t.Fatalf("AddTool after start failed: %v", err)
	}

	if len(server.tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(server.tools))
	}

	if err := server.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestServerResourceManagement(t *testing.T) {
	impl := Implementation{Name: "test-server", Version: "1.0.0"}
	cfg := (*config.Config)(nil)
	log := createTestLogger(t)
	server := NewServer(impl, cfg, log).(*Server)

	handler := &mockResourceHandler{}
	resource := &mockResource{
		uri:         "file://test.txt",
		name:        "test-resource",
		description: "A test resource",
		mimeType:    "text/plain",
		handler:     handler,
	}

	if err := server.AddResource(resource); err != nil {
		t.Fatalf("AddResource failed: %v", err)
	}

	if len(server.resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(server.resources))
	}

	if server.resources["file://test.txt"] != resource {
		t.Error("Resource was not stored correctly")
	}

	ctx := context.Background()
	transport := NewTestableStdioTransport(nil, nil)
	if err := server.Start(ctx, transport); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

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

	if len(server.resources) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(server.resources))
	}

	if err := server.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestConcurrentAccess(t *testing.T) {
	impl := Implementation{Name: "test-server", Version: "1.0.0"}
	cfg := (*config.Config)(nil)
	log := createTestLogger(t)
	server := NewServer(impl, cfg, log).(*Server)

	ctx := context.Background()
	transport := NewTestableStdioTransport(nil, nil)

	if err := server.Start(ctx, transport); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

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

	wg.Wait()

	if len(server.tools) != numOps {
		t.Errorf("Expected %d tools, got %d", numOps, len(server.tools))
	}

	if len(server.resources) != numOps {
		t.Errorf("Expected %d resources, got %d", numOps, len(server.resources))
	}

	if err := server.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestToolHandlerAdapter(t *testing.T) {
	impl := Implementation{Name: "test-server", Version: "1.0.0"}
	cfg := (*config.Config)(nil)
	log := createTestLogger(t)
	server := NewServer(impl, cfg, log).(*Server)

	// Test successful tool execution
	t.Run("SuccessfulExecution", func(t *testing.T) {
		handler := &mockToolHandler{
			handleFunc: func(ctx context.Context, params json.RawMessage) (ToolResult, error) {
				return &ToolResultImpl{Content: []Content{&TextContent{Text: "success result"}}, IsErrorFlag: false}, nil
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
