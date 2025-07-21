package resources

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
)

// Mock implementations for testing

type mockResourceContent struct {
	content  []mcp.Content
	mimeType string
}

func (m *mockResourceContent) GetContent() []mcp.Content {
	return m.content
}

func (m *mockResourceContent) GetMimeType() string {
	return m.mimeType
}

type mockContent struct {
	contentType string
	text        string
	blob        []byte
}

func (m *mockContent) Type() string {
	return m.contentType
}

func (m *mockContent) GetText() string {
	return m.text
}

func (m *mockContent) GetBlob() []byte {
	return m.blob
}

type mockResourceHandler struct{}

func (m *mockResourceHandler) Read(ctx context.Context, uri string) (mcp.ResourceContent, error) {
	content := &mockContent{
		contentType: "text",
		text:        "mock content for " + uri,
	}
	return &mockResourceContent{
		content:  []mcp.Content{content},
		mimeType: "text/plain",
	}, nil
}

type mockResource struct {
	uri         string
	name        string
	description string
	mimeType    string
	handler     mcp.ResourceHandler
}

func (m *mockResource) URI() string                        { return m.uri }
func (m *mockResource) Name() string                       { return m.name }
func (m *mockResource) Description() string                { return m.description }
func (m *mockResource) MimeType() string                   { return m.mimeType }
func (m *mockResource) Handler() mcp.ResourceHandler       { return m.handler }

type mockResourceFactory struct {
	uri          string
	name         string
	description  string
	mimeType     string
	version      string
	tags         []string
	capabilities []string
	createError  error
}

func (m *mockResourceFactory) URI() string                                                        { return m.uri }
func (m *mockResourceFactory) Name() string                                                       { return m.name }
func (m *mockResourceFactory) Description() string                                                { return m.description }
func (m *mockResourceFactory) MimeType() string                                                   { return m.mimeType }
func (m *mockResourceFactory) Version() string                                                    { return m.version }
func (m *mockResourceFactory) Tags() []string                                                     { return m.tags }
func (m *mockResourceFactory) Capabilities() []string                                             { return m.capabilities }
func (m *mockResourceFactory) Validate(config ResourceConfig) error                               { return nil }

func (m *mockResourceFactory) Create(ctx context.Context, config ResourceConfig) (mcp.Resource, error) {
	if m.createError != nil {
		return nil, m.createError
	}
	return &mockResource{
		uri:         m.uri,
		name:        m.name,
		description: m.description,
		mimeType:    m.mimeType,
		handler:     &mockResourceHandler{},
	}, nil
}

func createTestResourceRegistry() ResourceRegistry {
	cfg := &config.Config{
		MCP: config.MCPConfig{
			MaxResources: 100,
		},
	}
	log, _ := logger.NewDefault()
	return NewDefaultResourceRegistry(cfg, log)
}

func createTestResourceFactory(uri string) ResourceFactory {
	// Create a safe name without URI characters
	safeName := "Test Resource"
	return &mockResourceFactory{
		uri:          uri,
		name:         safeName,
		description:  "Test resource for " + uri,
		mimeType:     "text/plain",
		version:      "1.0.0",
		tags:         []string{"test"},
		capabilities: []string{"read"},
	}
}

func TestDefaultResourceRegistry_Register(t *testing.T) {
	registry := createTestResourceRegistry()
	factory := createTestResourceFactory("file:///test/resource.txt")

	err := registry.Register("file:///test/resource.txt", factory)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify resource is in registry
	info := registry.List()
	if len(info) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(info))
	}

	if info[0].URI != "file:///test/resource.txt" {
		t.Errorf("Expected resource URI 'file:///test/resource.txt', got '%s'", info[0].URI)
	}

	if info[0].Status != ResourceStatusRegistered {
		t.Errorf("Expected status '%s', got '%s'", ResourceStatusRegistered, info[0].Status)
	}
}

func TestDefaultResourceRegistry_RegisterDuplicate(t *testing.T) {
	registry := createTestResourceRegistry()
	factory := createTestResourceFactory("file:///test/resource.txt")

	// Register first time
	err := registry.Register("file:///test/resource.txt", factory)
	if err != nil {
		t.Fatalf("Expected no error on first registration, got: %v", err)
	}

	// Register second time should fail
	err = registry.Register("file:///test/resource.txt", factory)
	if err == nil {
		t.Fatal("Expected error on duplicate registration, got nil")
	}

	if !strings.Contains(err.Error(), "resource already exists") {
		t.Errorf("Expected resource already exists error, got: %v", err)
	}
}

func TestDefaultResourceRegistry_RegisterInvalidURI(t *testing.T) {
	registry := createTestResourceRegistry()
	factory := createTestResourceFactory("invalid-uri!")

	err := registry.Register("invalid-uri!", factory)
	if err == nil {
		t.Fatal("Expected error for invalid URI, got nil")
	}

	if !strings.Contains(err.Error(), "invalid resource URI") {
		t.Errorf("Expected invalid resource URI error, got: %v", err)
	}
}

func TestDefaultResourceRegistry_Unregister(t *testing.T) {
	registry := createTestResourceRegistry()
	factory := createTestResourceFactory("file:///test/resource.txt")

	// Register resource
	err := registry.Register("file:///test/resource.txt", factory)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Unregister resource
	err = registry.Unregister("file:///test/resource.txt")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify resource is removed
	info := registry.List()
	if len(info) != 0 {
		t.Fatalf("Expected 0 resources, got %d", len(info))
	}
}

func TestDefaultResourceRegistry_UnregisterNonExistent(t *testing.T) {
	registry := createTestResourceRegistry()

	err := registry.Unregister("file:///non/existent.txt")
	if err == nil {
		t.Fatal("Expected error for non-existent resource, got nil")
	}

	if !strings.Contains(err.Error(), "resource not found") {
		t.Errorf("Expected resource not found error, got: %v", err)
	}
}

func TestDefaultResourceRegistry_Get(t *testing.T) {
	registry := createTestResourceRegistry()
	factory := createTestResourceFactory("file:///test/resource.txt")

	// Register resource
	err := registry.Register("file:///test/resource.txt", factory)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Get resource (should create instance)
	resource, err := registry.Get("file:///test/resource.txt")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if resource.URI() != "file:///test/resource.txt" {
		t.Errorf("Expected resource URI 'file:///test/resource.txt', got '%s'", resource.URI())
	}

	// Get resource again (should return cached instance)
	resource2, err := registry.Get("file:///test/resource.txt")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if resource != resource2 {
		t.Error("Expected same resource instance on second get")
	}
}

func TestDefaultResourceRegistry_GetNonExistent(t *testing.T) {
	registry := createTestResourceRegistry()

	_, err := registry.Get("file:///non/existent.txt")
	if err == nil {
		t.Fatal("Expected error for non-existent resource, got nil")
	}

	if !strings.Contains(err.Error(), "resource not found") {
		t.Errorf("Expected resource not found error, got: %v", err)
	}
}

func TestDefaultResourceRegistry_GetFactory(t *testing.T) {
	registry := createTestResourceRegistry()
	factory := createTestResourceFactory("file:///test/resource.txt")

	// Register resource
	err := registry.Register("file:///test/resource.txt", factory)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Get factory
	retrievedFactory, err := registry.GetFactory("file:///test/resource.txt")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if retrievedFactory.URI() != factory.URI() {
		t.Errorf("Expected factory URI '%s', got '%s'", factory.URI(), retrievedFactory.URI())
	}
}

func TestDefaultResourceRegistry_List(t *testing.T) {
	registry := createTestResourceRegistry()

	// Register multiple resources
	resources := []string{"file:///test1.txt", "config://database/connection", "api://service/data"}
	for _, uri := range resources {
		factory := createTestResourceFactory(uri)
		err := registry.Register(uri, factory)
		if err != nil {
			t.Fatalf("Expected no error registering %s, got: %v", uri, err)
		}
	}

	// List resources
	info := registry.List()
	if len(info) != len(resources) {
		t.Fatalf("Expected %d resources, got %d", len(resources), len(info))
	}

	// Check all resources are present
	uriSet := make(map[string]bool)
	for _, resourceInfo := range info {
		uriSet[resourceInfo.URI] = true
	}

	for _, uri := range resources {
		if !uriSet[uri] {
			t.Errorf("Resource '%s' not found in list", uri)
		}
	}
}

func TestDefaultResourceRegistry_StartStop(t *testing.T) {
	registry := createTestResourceRegistry()
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

func TestDefaultResourceRegistry_LoadResources(t *testing.T) {
	registry := createTestResourceRegistry()
	ctx := context.Background()

	// Start registry
	err := registry.Start(ctx)
	if err != nil {
		t.Fatalf("Expected no error starting registry, got: %v", err)
	}

	// Register resources
	resources := []string{"file:///test1.txt", "file:///test2.txt"}
	for _, uri := range resources {
		factory := createTestResourceFactory(uri)
		err := registry.Register(uri, factory)
		if err != nil {
			t.Fatalf("Expected no error registering %s, got: %v", uri, err)
		}
	}

	// Load resources
	err = registry.LoadResources(ctx)
	if err != nil {
		t.Fatalf("Expected no error loading resources, got: %v", err)
	}

	// Verify resources are loaded
	info := registry.List()
	for _, resourceInfo := range info {
		if resourceInfo.Status != ResourceStatusLoaded {
			t.Errorf("Expected resource '%s' to be loaded, got status '%s'", resourceInfo.URI, resourceInfo.Status)
		}
	}
}

func TestDefaultResourceRegistry_ValidateResources(t *testing.T) {
	registry := createTestResourceRegistry()
	ctx := context.Background()

	// Start registry and register resources
	err := registry.Start(ctx)
	if err != nil {
		t.Fatalf("Expected no error starting registry, got: %v", err)
	}

	factory := createTestResourceFactory("file:///test/resource.txt")
	err = registry.Register("file:///test/resource.txt", factory)
	if err != nil {
		t.Fatalf("Expected no error registering resource, got: %v", err)
	}

	err = registry.LoadResources(ctx)
	if err != nil {
		t.Fatalf("Expected no error loading resources, got: %v", err)
	}

	// Validate resources
	err = registry.ValidateResources(ctx)
	if err != nil {
		t.Fatalf("Expected no error validating resources, got: %v", err)
	}

	// Check resource status
	info := registry.List()
	if len(info) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(info))
	}

	if info[0].Status != ResourceStatusActive {
		t.Errorf("Expected resource status to be active, got '%s'", info[0].Status)
	}
}

func TestDefaultResourceRegistry_TransitionStatus(t *testing.T) {
	registry := createTestResourceRegistry()
	ctx := context.Background()
	factory := createTestResourceFactory("file:///test/resource.txt")

	// Start registry and register resource
	err := registry.Start(ctx)
	if err != nil {
		t.Fatalf("Expected no error starting registry, got: %v", err)
	}

	err = registry.Register("file:///test/resource.txt", factory)
	if err != nil {
		t.Fatalf("Expected no error registering resource, got: %v", err)
	}

	// Test valid transition
	err = registry.TransitionStatus("file:///test/resource.txt", ResourceStatusLoaded)
	if err != nil {
		t.Fatalf("Expected no error transitioning status, got: %v", err)
	}

	// Verify status changed
	info := registry.List()
	if info[0].Status != ResourceStatusLoaded {
		t.Errorf("Expected status to be loaded, got '%s'", info[0].Status)
	}

	// Test invalid transition
	err = registry.TransitionStatus("file:///test/resource.txt", ResourceStatusUnknown)
	if err == nil {
		t.Fatal("Expected error for invalid transition, got nil")
	}

	if !strings.Contains(err.Error(), "invalid status transition") {
		t.Errorf("Expected invalid transition error, got: %v", err)
	}
}

func TestDefaultResourceRegistry_RefreshResource(t *testing.T) {
	registry := createTestResourceRegistry()
	ctx := context.Background()
	factory := createTestResourceFactory("file:///test/resource.txt")

	// Start registry and register resource
	err := registry.Start(ctx)
	if err != nil {
		t.Fatalf("Expected no error starting registry, got: %v", err)
	}

	err = registry.Register("file:///test/resource.txt", factory)
	if err != nil {
		t.Fatalf("Expected no error registering resource, got: %v", err)
	}

	// Load and activate resource
	err = registry.LoadResources(ctx)
	if err != nil {
		t.Fatalf("Expected no error loading resources, got: %v", err)
	}

	err = registry.ValidateResources(ctx)
	if err != nil {
		t.Fatalf("Expected no error validating resources, got: %v", err)
	}

	// Refresh resource
	err = registry.RefreshResource(ctx, "file:///test/resource.txt")
	if err != nil {
		t.Fatalf("Expected no error refreshing resource, got: %v", err)
	}
}

func TestDefaultResourceRegistry_ConcurrentAccess(t *testing.T) {
	registry := createTestResourceRegistry()
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
			uri := fmt.Sprintf("file:///test/resource_%d.txt", i)
			factory := createTestResourceFactory(uri)
			if err := registry.Register(uri, factory); err != nil {
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
				uri := fmt.Sprintf("file:///test/resource_%d.txt", i)
				_, err := registry.Get(uri)
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
		t.Errorf("Expected %d resources after concurrent operations, got %d", concurrency, len(info))
	}
}

func TestDefaultResourceRegistry_Health(t *testing.T) {
	registry := createTestResourceRegistry()
	ctx := context.Background()

	// Health when stopped
	health := registry.Health()
	if health.Status != "stopped" {
		t.Errorf("Expected stopped status, got '%s'", health.Status)
	}

	// Start and add resources
	err := registry.Start(ctx)
	if err != nil {
		t.Fatalf("Expected no error starting registry, got: %v", err)
	}

	factory := createTestResourceFactory("file:///test/resource.txt")
	err = registry.Register("file:///test/resource.txt", factory)
	if err != nil {
		t.Fatalf("Expected no error registering resource, got: %v", err)
	}

	err = registry.LoadResources(ctx)
	if err != nil {
		t.Fatalf("Expected no error loading resources, got: %v", err)
	}

	err = registry.ValidateResources(ctx)
	if err != nil {
		t.Fatalf("Expected no error validating resources, got: %v", err)
	}

	// Health when running with active resources
	health = registry.Health()
	if health.Status != "healthy" {
		t.Errorf("Expected healthy status, got '%s'", health.Status)
	}

	if health.ResourceCount != 1 {
		t.Errorf("Expected 1 resource count, got %d", health.ResourceCount)
	}

	if health.ActiveResources != 1 {
		t.Errorf("Expected 1 active resource, got %d", health.ActiveResources)
	}

	if health.ErrorResources != 0 {
		t.Errorf("Expected 0 error resources, got %d", health.ErrorResources)
	}

	// Verify resource status in health
	if status, exists := health.ResourceStatuses["file:///test/resource.txt"]; !exists {
		t.Error("Expected file:///test/resource.txt in health status")
	} else if status != string(ResourceStatusActive) {
		t.Errorf("Expected active status for test resource, got '%s'", status)
	}
}

func TestDefaultResourceRegistry_ErrorHandling(t *testing.T) {
	registry := createTestResourceRegistry()
	ctx := context.Background()

	// Test registration of factory that fails to create
	factory := &mockResourceFactory{
		uri:         "file:///test/failing.txt",
		name:        "Failing Resource",
		description: "A resource that fails to create",
		mimeType:    "text/plain",
		version:     "1.0.0",
		tags:        []string{"test"},
		capabilities: []string{"read"},
		createError: fmt.Errorf("creation failed"),
	}

	err := registry.Start(ctx)
	if err != nil {
		t.Fatalf("Expected no error starting registry, got: %v", err)
	}

	err = registry.Register("file:///test/failing.txt", factory)
	if err != nil {
		t.Fatalf("Expected no error registering factory, got: %v", err)
	}

	// Loading should handle the error gracefully
	err = registry.LoadResources(ctx)
	if err == nil {
		t.Error("Expected error loading resources with failing factory")
	}

	// Check that failing resource is marked as error
	info := registry.List()
	if len(info) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(info))
	}

	if info[0].Status != ResourceStatusError {
		t.Errorf("Expected error status for failing resource, got '%s'", info[0].Status)
	}
}