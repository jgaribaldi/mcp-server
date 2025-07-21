package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

type MockMCPServer struct {
	tools     map[string]Tool
	resources map[string]Resource
	impl      Implementation
	running   bool
}

func (m *MockMCPServer) Start(ctx context.Context, transport Transport) error {
	m.running = true
	return nil
}

func (m *MockMCPServer) Stop(ctx context.Context) error {
	m.running = false
	return nil
}

func (m *MockMCPServer) AddTool(tool Tool) error {
	if m.tools == nil {
		m.tools = make(map[string]Tool)
	}
	m.tools[tool.Name()] = tool
	return nil
}

func (m *MockMCPServer) AddResource(resource Resource) error {
	if m.resources == nil {
		m.resources = make(map[string]Resource)
	}
	m.resources[resource.URI()] = resource
	return nil
}

func (m *MockMCPServer) GetImplementation() Implementation {
	return m.impl
}

type MockTool struct {
	name        string
	description string
	parameters  json.RawMessage
	handler     ToolHandler
}

func (m *MockTool) Name() string              { return m.name }
func (m *MockTool) Description() string       { return m.description }
func (m *MockTool) Parameters() json.RawMessage { return m.parameters }
func (m *MockTool) Handler() ToolHandler      { return m.handler }

type MockToolHandler struct {
	handleFunc func(ctx context.Context, params json.RawMessage) (ToolResult, error)
}

func (m *MockToolHandler) Handle(ctx context.Context, params json.RawMessage) (ToolResult, error) {
	if m.handleFunc != nil {
		return m.handleFunc(ctx, params)
	}
	return &MockToolResult{}, nil
}

type MockToolResult struct {
	isError bool
	content []Content
	err     error
}

func (m *MockToolResult) IsError() bool       { return m.isError }
func (m *MockToolResult) GetContent() []Content { return m.content }
func (m *MockToolResult) GetError() error     { return m.err }

type MockResource struct {
	uri         string
	name        string
	description string
	mimeType    string
	handler     ResourceHandler
}

func (m *MockResource) URI() string            { return m.uri }
func (m *MockResource) Name() string           { return m.name }
func (m *MockResource) Description() string    { return m.description }
func (m *MockResource) MimeType() string       { return m.mimeType }
func (m *MockResource) Handler() ResourceHandler { return m.handler }

type MockResourceHandler struct{}

func (m *MockResourceHandler) Read(ctx context.Context, uri string) (ResourceContent, error) {
	return &MockResourceContent{}, nil
}

type MockResourceContent struct {
	content  []Content
	mimeType string
}

func (m *MockResourceContent) GetContent() []Content { return m.content }
func (m *MockResourceContent) GetMimeType() string   { return m.mimeType }

type MockTransport struct {
	readFunc  func() ([]byte, error)
	writeFunc func(data []byte) error
	closeFunc func() error
}

func (m *MockTransport) Read() ([]byte, error) {
	if m.readFunc != nil {
		return m.readFunc()
	}
	return []byte("mock data"), nil
}

func (m *MockTransport) Write(data []byte) error {
	if m.writeFunc != nil {
		return m.writeFunc(data)
	}
	return nil
}

func (m *MockTransport) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

type MockContent struct {
	contentType string
	text        string
	blob        []byte
}

func (m *MockContent) Type() string    { return m.contentType }
func (m *MockContent) GetText() string { return m.text }
func (m *MockContent) GetBlob() []byte { return m.blob }

func TestImplementation(t *testing.T) {
	impl := Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}

	if impl.Name != "test-server" {
		t.Errorf("Expected name 'test-server', got '%s'", impl.Name)
	}

	if impl.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", impl.Version)
	}
}

func TestCallToolParams(t *testing.T) {
	params := CallToolParams{
		Name:      "test-tool",
		Arguments: json.RawMessage(`{"arg1": "value1"}`),
	}

	if params.Name != "test-tool" {
		t.Errorf("Expected name 'test-tool', got '%s'", params.Name)
	}

	var args map[string]interface{}
	if err := json.Unmarshal(params.Arguments, &args); err != nil {
		t.Errorf("Failed to unmarshal arguments: %v", err)
	}

	if args["arg1"] != "value1" {
		t.Errorf("Expected arg1 'value1', got '%v'", args["arg1"])
	}
}

func TestMockMCPServer(t *testing.T) {
	server := &MockMCPServer{
		impl: Implementation{
			Name:    "mock-server",
			Version: "1.0.0",
		},
	}

	ctx := context.Background()

	transport := &MockTransport{}
	if err := server.Start(ctx, transport); err != nil {
		t.Errorf("Failed to start server: %v", err)
	}

	if !server.running {
		t.Error("Expected server to be running")
	}

	if err := server.Stop(ctx); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}

	if server.running {
		t.Error("Expected server to be stopped")
	}

	impl := server.GetImplementation()
	if impl.Name != "mock-server" {
		t.Errorf("Expected implementation name 'mock-server', got '%s'", impl.Name)
	}
}

func TestToolManagement(t *testing.T) {
	server := &MockMCPServer{}

	tool := &MockTool{
		name:        "test-tool",
		description: "A test tool",
		parameters:  json.RawMessage(`{"type": "object"}`),
		handler:     &MockToolHandler{},
	}

	if err := server.AddTool(tool); err != nil {
		t.Errorf("Failed to add tool: %v", err)
	}

	if len(server.tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(server.tools))
	}

	if server.tools["test-tool"] != tool {
		t.Error("Tool was not stored correctly")
	}
}

func TestResourceManagement(t *testing.T) {
	server := &MockMCPServer{}

	resource := &MockResource{
		uri:         "file://test.txt",
		name:        "test-resource",
		description: "A test resource",
		mimeType:    "text/plain",
		handler:     &MockResourceHandler{},
	}

	if err := server.AddResource(resource); err != nil {
		t.Errorf("Failed to add resource: %v", err)
	}

	if len(server.resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(server.resources))
	}

	if server.resources["file://test.txt"] != resource {
		t.Error("Resource was not stored correctly")
	}
}

func TestToolHandler(t *testing.T) {
	handlerCalled := false
	handler := &MockToolHandler{
		handleFunc: func(ctx context.Context, params json.RawMessage) (ToolResult, error) {
			handlerCalled = true
			return &MockToolResult{
				content: []Content{
					&MockContent{
						contentType: "text",
						text:        "test result",
					},
				},
			}, nil
		},
	}

	ctx := context.Background()
	params := json.RawMessage(`{"test": "value"}`)

	result, err := handler.Handle(ctx, params)
	if err != nil {
		t.Errorf("Handler failed: %v", err)
	}

	if !handlerCalled {
		t.Error("Handler function was not called")
	}

	if result.IsError() {
		t.Error("Expected successful result")
	}

	content := result.GetContent()
	if len(content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(content))
	}

	if content[0].GetText() != "test result" {
		t.Errorf("Expected 'test result', got '%s'", content[0].GetText())
	}
}

func TestTransport(t *testing.T) {
	readCalled := false
	writeCalled := false
	closeCalled := false

	transport := &MockTransport{
		readFunc: func() ([]byte, error) {
			readCalled = true
			return []byte("test data"), nil
		},
		writeFunc: func(data []byte) error {
			writeCalled = true
			if string(data) != "output data" {
				t.Errorf("Expected 'output data', got '%s'", string(data))
			}
			return nil
		},
		closeFunc: func() error {
			closeCalled = true
			return nil
		},
	}

	data, err := transport.Read()
	if err != nil {
		t.Errorf("Read failed: %v", err)
	}
	if string(data) != "test data" {
		t.Errorf("Expected 'test data', got '%s'", string(data))
	}
	if !readCalled {
		t.Error("Read function was not called")
	}

	if err := transport.Write([]byte("output data")); err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if !writeCalled {
		t.Error("Write function was not called")
	}

	if err := transport.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
	if !closeCalled {
		t.Error("Close function was not called")
	}
}
