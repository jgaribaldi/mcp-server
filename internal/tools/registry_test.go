package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
	"mcp-server/internal/tools/adapters"
)

// Mock implementations for testing

type mockTool struct {
	name        string
	description string
	parameters  json.RawMessage
	handler     mcp.ToolHandler
}

func (m *mockTool) Name() string                  { return m.name }
func (m *mockTool) Description() string           { return m.description }
func (m *mockTool) Parameters() json.RawMessage   { return m.parameters }
func (m *mockTool) Handler() mcp.ToolHandler      { return m.handler }

type mockToolHandler struct{}

func (m *mockToolHandler) Handle(ctx context.Context, params json.RawMessage) (mcp.ToolResult, error) {
	return nil, nil
}

type mockToolFactory struct {
	name         string
	description  string
	version      string
	capabilities []string
	requirements map[string]string
	createError  error
}

func (m *mockToolFactory) Name() string                       { return m.name }
func (m *mockToolFactory) Description() string                { return m.description }
func (m *mockToolFactory) Version() string                    { return m.version }
func (m *mockToolFactory) Capabilities() []string             { return m.capabilities }
func (m *mockToolFactory) Requirements() map[string]string    { return m.requirements }
func (m *mockToolFactory) Validate(config ToolConfig) error   { return nil }

func (m *mockToolFactory) Create(ctx context.Context, config ToolConfig) (mcp.Tool, error) {
	if m.createError != nil {
		return nil, m.createError
	}
	return &mockTool{
		name:        m.name,
		description: m.description,
		parameters:  json.RawMessage(`{"type": "object"}`),
		handler:     &mockToolHandler{},
	}, nil
}

func createTestRegistry() ToolRegistry {
	cfg := &config.Config{
		MCP: config.MCPConfig{
			MaxTools: 100,
		},
	}
	log, _ := logger.NewDefault()
	return NewDefaultToolRegistry(cfg, log)
}

func createTestFactory(name string) ToolFactory {
	return &mockToolFactory{
		name:         name,
		description:  "Test tool " + name,
		version:      "1.0.0",
		capabilities: []string{"test"},
		requirements: map[string]string{"runtime": "go"},
	}
}

func TestDefaultToolRegistry_Register(t *testing.T) {
	registry := createTestRegistry()
	factory := createTestFactory("test_tool")

	err := registry.Register("test_tool", factory)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify tool is in registry
	info := registry.List()
	if len(info) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(info))
	}

	if info[0].Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", info[0].Name)
	}

	if info[0].Status != ToolStatusRegistered {
		t.Errorf("Expected status '%s', got '%s'", ToolStatusRegistered, info[0].Status)
	}
}

func TestDefaultToolRegistry_RegisterDuplicate(t *testing.T) {
	registry := createTestRegistry()
	factory := createTestFactory("test_tool")

	// Register first time
	err := registry.Register("test_tool", factory)
	if err != nil {
		t.Fatalf("Expected no error on first registration, got: %v", err)
	}

	// Register second time should fail
	err = registry.Register("test_tool", factory)
	if err == nil {
		t.Fatal("Expected error on duplicate registration, got nil")
	}

	if !strings.Contains(err.Error(), "tool already exists") {
		t.Errorf("Expected tool already exists error, got: %v", err)
	}
}

func TestDefaultToolRegistry_RegisterInvalidName(t *testing.T) {
	registry := createTestRegistry()
	factory := createTestFactory("invalid-name!")

	err := registry.Register("invalid-name!", factory)
	if err == nil {
		t.Fatal("Expected error for invalid name, got nil")
	}

	if !strings.Contains(err.Error(), "invalid tool name") {
		t.Errorf("Expected invalid tool name error, got: %v", err)
	}
}

func TestDefaultToolRegistry_Unregister(t *testing.T) {
	registry := createTestRegistry()
	factory := createTestFactory("test_tool")

	// Register tool
	err := registry.Register("test_tool", factory)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Unregister tool
	err = registry.Unregister("test_tool")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify tool is removed
	info := registry.List()
	if len(info) != 0 {
		t.Fatalf("Expected 0 tools, got %d", len(info))
	}
}

func TestDefaultToolRegistry_UnregisterNonExistent(t *testing.T) {
	registry := createTestRegistry()

	err := registry.Unregister("non_existent")
	if err == nil {
		t.Fatal("Expected error for non-existent tool, got nil")
	}

	if !strings.Contains(err.Error(), "tool not found") {
		t.Errorf("Expected tool not found error, got: %v", err)
	}
}

func TestDefaultToolRegistry_Get(t *testing.T) {
	registry := createTestRegistry()
	factory := createTestFactory("test_tool")

	// Register tool
	err := registry.Register("test_tool", factory)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Get tool (should create instance)
	tool, err := registry.Get("test_tool")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if tool.Name() != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", tool.Name())
	}

	// Get tool again (should return cached instance)
	tool2, err := registry.Get("test_tool")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if tool != tool2 {
		t.Error("Expected same tool instance on second get")
	}
}

func TestDefaultToolRegistry_GetNonExistent(t *testing.T) {
	registry := createTestRegistry()

	_, err := registry.Get("non_existent")
	if err == nil {
		t.Fatal("Expected error for non-existent tool, got nil")
	}

	if !strings.Contains(err.Error(), "tool not found") {
		t.Errorf("Expected tool not found error, got: %v", err)
	}
}

func TestDefaultToolRegistry_GetFactory(t *testing.T) {
	registry := createTestRegistry()
	factory := createTestFactory("test_tool")

	// Register tool
	err := registry.Register("test_tool", factory)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Get factory
	retrievedFactory, err := registry.GetFactory("test_tool")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if retrievedFactory.Name() != factory.Name() {
		t.Errorf("Expected factory name '%s', got '%s'", factory.Name(), retrievedFactory.Name())
	}
}

func TestDefaultToolRegistry_List(t *testing.T) {
	registry := createTestRegistry()

	// Register multiple tools
	tools := []string{"tool1", "tool2", "tool3"}
	for _, name := range tools {
		factory := createTestFactory(name)
		err := registry.Register(name, factory)
		if err != nil {
			t.Fatalf("Expected no error registering %s, got: %v", name, err)
		}
	}

	// List tools
	info := registry.List()
	if len(info) != len(tools) {
		t.Fatalf("Expected %d tools, got %d", len(tools), len(info))
	}

	// Check all tools are present
	nameSet := make(map[string]bool)
	for _, toolInfo := range info {
		nameSet[toolInfo.Name] = true
	}

	for _, name := range tools {
		if !nameSet[name] {
			t.Errorf("Tool '%s' not found in list", name)
		}
	}
}

func TestDefaultToolRegistry_StartStop(t *testing.T) {
	registry := createTestRegistry()
	ctx := context.Background()

	// Start registry
	err := registry.Start(ctx)
	if err != nil {
		t.Fatalf("Expected no error starting registry, got: %v", err)
	}

	// Check health
	health := registry.Health()
	if health.Status != "healthy" {
		t.Errorf("Expected healthy status, got '%s'", health.Status)
	}

	// Stop registry
	err = registry.Stop(ctx)
	if err != nil {
		t.Fatalf("Expected no error stopping registry, got: %v", err)
	}

	// Check health after stop
	health = registry.Health()
	if health.Status != "stopped" {
		t.Errorf("Expected stopped status, got '%s'", health.Status)
	}
}

func TestDefaultToolRegistry_LoadTools(t *testing.T) {
	registry := createTestRegistry()
	ctx := context.Background()

	// Start registry
	err := registry.Start(ctx)
	if err != nil {
		t.Fatalf("Expected no error starting registry, got: %v", err)
	}

	// Register tools
	tools := []string{"tool1", "tool2"}
	for _, name := range tools {
		factory := createTestFactory(name)
		err := registry.Register(name, factory)
		if err != nil {
			t.Fatalf("Expected no error registering %s, got: %v", name, err)
		}
	}

	// Load tools
	err = registry.LoadTools(ctx)
	if err != nil {
		t.Fatalf("Expected no error loading tools, got: %v", err)
	}

	// Verify tools are loaded
	info := registry.List()
	for _, toolInfo := range info {
		if toolInfo.Status != ToolStatusLoaded {
			t.Errorf("Expected tool '%s' to be loaded, got status '%s'", toolInfo.Name, toolInfo.Status)
		}
	}
}

func TestDefaultToolRegistry_ValidateTools(t *testing.T) {
	registry := createTestRegistry()
	ctx := context.Background()

	// Start registry and register tools
	err := registry.Start(ctx)
	if err != nil {
		t.Fatalf("Expected no error starting registry, got: %v", err)
	}

	factory := createTestFactory("test_tool")
	err = registry.Register("test_tool", factory)
	if err != nil {
		t.Fatalf("Expected no error registering tool, got: %v", err)
	}

	err = registry.LoadTools(ctx)
	if err != nil {
		t.Fatalf("Expected no error loading tools, got: %v", err)
	}

	// Validate tools
	err = registry.ValidateTools(ctx)
	if err != nil {
		t.Fatalf("Expected no error validating tools, got: %v", err)
	}

	// Check tool status
	info := registry.List()
	if len(info) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(info))
	}

	if info[0].Status != ToolStatusActive {
		t.Errorf("Expected tool status to be active, got '%s'", info[0].Status)
	}
}

func TestDefaultToolRegistry_ConcurrentAccess(t *testing.T) {
	registry := createTestRegistry()
	ctx := context.Background()

	err := registry.Start(ctx)
	if err != nil {
		t.Fatalf("Expected no error starting registry, got: %v", err)
	}

	// Number of concurrent operations
	concurrency := 50
	var wg sync.WaitGroup
	errors := make(chan error, concurrency*3)

	// Concurrent registrations
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func(i int) {
			defer wg.Done()
			toolName := fmt.Sprintf("tool_%d", i)
			factory := createTestFactory(toolName)
			if err := registry.Register(toolName, factory); err != nil {
				errors <- err
			}
		}(i)
	}

	// Concurrent reads while registering
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				registry.List()
				time.Sleep(time.Microsecond)
			}
		}()
	}

	// Concurrent gets after some delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		wg.Add(concurrency)
		for i := 0; i < concurrency; i++ {
			go func(i int) {
				defer wg.Done()
				toolName := fmt.Sprintf("tool_%d", i)
				_, err := registry.Get(toolName)
				if err != nil {
					errors <- err
				}
			}(i)
		}
	}()

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
	}

	// Verify final state
	info := registry.List()
	if len(info) != concurrency {
		t.Errorf("Expected %d tools after concurrent operations, got %d", concurrency, len(info))
	}
}

func TestDefaultToolRegistry_Health(t *testing.T) {
	registry := createTestRegistry()
	ctx := context.Background()

	// Health when stopped
	health := registry.Health()
	if health.Status != "stopped" {
		t.Errorf("Expected stopped status, got '%s'", health.Status)
	}

	// Start and add tools
	err := registry.Start(ctx)
	if err != nil {
		t.Fatalf("Expected no error starting registry, got: %v", err)
	}

	factory := createTestFactory("test_tool")
	err = registry.Register("test_tool", factory)
	if err != nil {
		t.Fatalf("Expected no error registering tool, got: %v", err)
	}

	err = registry.LoadTools(ctx)
	if err != nil {
		t.Fatalf("Expected no error loading tools, got: %v", err)
	}

	err = registry.ValidateTools(ctx)
	if err != nil {
		t.Fatalf("Expected no error validating tools, got: %v", err)
	}

	// Health when running with active tools
	health = registry.Health()
	if health.Status != "healthy" {
		t.Errorf("Expected healthy status, got '%s'", health.Status)
	}

	if health.ToolCount != 1 {
		t.Errorf("Expected 1 tool count, got %d", health.ToolCount)
	}

	if health.ActiveTools != 1 {
		t.Errorf("Expected 1 active tool, got %d", health.ActiveTools)
	}

	if health.ErrorTools != 0 {
		t.Errorf("Expected 0 error tools, got %d", health.ErrorTools)
	}

	// Verify tool status in health
	if status, exists := health.ToolStatuses["test_tool"]; !exists {
		t.Error("Expected test_tool in health status")
	} else if status != string(ToolStatusActive) {
		t.Errorf("Expected active status for test_tool, got '%s'", status)
	}
}

// Mock adapter for testing business logic
type mockLibraryAdapter struct {
	tools             map[string]mcp.Tool
	resources         map[string]mcp.Resource
	running           bool
	startError        error
	stopError         error
	registerError     error
	unregisterError   error
	healthStatus      string
	mu                sync.RWMutex
}

func newMockAdapter() *mockLibraryAdapter {
	return &mockLibraryAdapter{
		tools:        make(map[string]mcp.Tool),
		resources:    make(map[string]mcp.Resource),
		healthStatus: "healthy",
	}
}

func (m *mockLibraryAdapter) RegisterTool(tool mcp.Tool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.registerError != nil {
		return m.registerError
	}
	m.tools[tool.Name()] = tool
	return nil
}

func (m *mockLibraryAdapter) UnregisterTool(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.unregisterError != nil {
		return m.unregisterError
	}
	delete(m.tools, name)
	return nil
}

func (m *mockLibraryAdapter) GetTool(name string) (mcp.Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tool, exists := m.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool not found")
	}
	return tool, nil
}

func (m *mockLibraryAdapter) ListTools() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var names []string
	for name := range m.tools {
		names = append(names, name)
	}
	return names
}

func (m *mockLibraryAdapter) RegisterResource(resource mcp.Resource) error   { return nil }
func (m *mockLibraryAdapter) UnregisterResource(uri string) error            { return nil }
func (m *mockLibraryAdapter) GetResource(uri string) (mcp.Resource, error)   { return nil, nil }
func (m *mockLibraryAdapter) ListResources() []string                        { return nil }

func (m *mockLibraryAdapter) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.startError != nil {
		return m.startError
	}
	m.running = true
	return nil
}

func (m *mockLibraryAdapter) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopError != nil {
		return m.stopError
	}
	m.running = false
	return nil
}

func (m *mockLibraryAdapter) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

func (m *mockLibraryAdapter) Health() adapters.AdapterHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return adapters.AdapterHealth{
		Status:        m.healthStatus,
		Library:       "mock",
		Version:       "1.0.0",
		ToolCount:     len(m.tools),
		ResourceCount: len(m.resources),
		LastCheck:     time.Now().Format(time.RFC3339),
		Errors:        []string{},
	}
}

// Helper to create registry with mock adapter
func createTestRegistryWithAdapter(adapter *mockLibraryAdapter) ToolRegistry {
	cfg := &config.Config{}
	log, _ := logger.NewDefault() // Use default logger for tests
	return NewDefaultToolRegistryWithAdapter(cfg, log, adapter)
}

// Test adapter integration business logic
func TestDefaultToolRegistry_AdapterIntegration(t *testing.T) {
	adapter := newMockAdapter()
	registry := createTestRegistryWithAdapter(adapter)
	ctx := context.Background()

	// Start registry should start adapter
	err := registry.Start(ctx)
	if err != nil {
		t.Fatalf("Expected no error starting registry, got: %v", err)
	}

	if !adapter.IsRunning() {
		t.Error("Expected adapter to be running after registry start")
	}

	// Register tool should register with adapter
	factory := createTestFactory("test_tool")
	err = registry.Register("test_tool", factory)
	if err != nil {
		t.Fatalf("Expected no error registering tool, got: %v", err)
	}

	// Get tool should trigger adapter registration
	tool, err := registry.Get("test_tool")
	if err != nil {
		t.Fatalf("Expected no error getting tool, got: %v", err)
	}

	if tool.Name() != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", tool.Name())
	}

	// Check tool was registered with adapter
	adapterTools := adapter.ListTools()
	if len(adapterTools) != 1 {
		t.Errorf("Expected 1 tool in adapter, got %d", len(adapterTools))
	}

	// Unregister should remove from adapter
	err = registry.Unregister("test_tool")
	if err != nil {
		t.Fatalf("Expected no error unregistering tool, got: %v", err)
	}

	// Check tool was removed from adapter
	adapterTools = adapter.ListTools()
	if len(adapterTools) != 0 {
		t.Errorf("Expected 0 tools in adapter after unregister, got %d", len(adapterTools))
	}

	// Stop registry should stop adapter
	err = registry.Stop(ctx)
	if err != nil {
		t.Fatalf("Expected no error stopping registry, got: %v", err)
	}

	if adapter.IsRunning() {
		t.Error("Expected adapter to be stopped after registry stop")
	}
}

// Test adapter failure handling business logic
func TestDefaultToolRegistry_AdapterFailureHandling(t *testing.T) {
	adapter := newMockAdapter()
	registry := createTestRegistryWithAdapter(adapter)
	ctx := context.Background()

	// Test start failure
	adapter.startError = fmt.Errorf("adapter start failed")
	err := registry.Start(ctx)
	if err == nil {
		t.Error("Expected error when adapter start fails")
	}

	// Reset for successful start
	adapter.startError = nil
	err = registry.Start(ctx)
	if err != nil {
		t.Fatalf("Expected no error on successful start, got: %v", err)
	}

	// Test registration with adapter failure - should continue with local registration
	adapter.registerError = fmt.Errorf("adapter register failed")
	factory := createTestFactory("test_tool")
	err = registry.Register("test_tool", factory)
	if err != nil {
		t.Fatalf("Expected no error registering tool, got: %v", err)
	}

	// Tool should still be gettable even if adapter registration failed
	tool, err := registry.Get("test_tool")
	if err != nil {
		t.Fatalf("Expected to get tool even after adapter registration failure, got: %v", err)
	}

	if tool.Name() != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", tool.Name())
	}

	// Reset adapter error for unregistration test
	adapter.registerError = nil
	adapter.unregisterError = fmt.Errorf("adapter unregister failed")

	// Unregister should succeed locally even if adapter fails
	err = registry.Unregister("test_tool")
	if err != nil {
		t.Fatalf("Expected no error unregistering tool even with adapter failure, got: %v", err)
	}

	// Tool should be gone from registry
	_, err = registry.Get("test_tool")
	if err == nil {
		t.Error("Expected error getting unregistered tool")
	}
}

// Test health integration with adapter
func TestDefaultToolRegistry_HealthWithAdapter(t *testing.T) {
	adapter := newMockAdapter()
	registry := createTestRegistryWithAdapter(adapter)
	ctx := context.Background()

	// Start registry
	err := registry.Start(ctx)
	if err != nil {
		t.Fatalf("Expected no error starting registry, got: %v", err)
	}

	// Test healthy adapter
	health := registry.Health()
	if health.Status != "healthy" {
		t.Errorf("Expected healthy status with healthy adapter, got '%s'", health.Status)
	}

	// Test degraded adapter
	adapter.healthStatus = "degraded"
	health = registry.Health()
	if health.Status != "degraded" {
		t.Errorf("Expected degraded status with degraded adapter, got '%s'", health.Status)
	}

	if len(health.Errors) == 0 {
		t.Error("Expected errors in health when adapter is degraded")
	}
}