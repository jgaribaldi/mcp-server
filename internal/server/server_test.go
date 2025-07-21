package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
	"mcp-server/internal/tools"
)

// =============================================================================
// Mock implementations for testing
// =============================================================================

type MockToolRegistry struct {
	health   tools.RegistryHealth
	toolList []tools.ToolInfo
}

func (m *MockToolRegistry) Register(name string, factory tools.ToolFactory) error {
	return nil
}

func (m *MockToolRegistry) Unregister(name string) error {
	return nil
}

func (m *MockToolRegistry) Get(name string) (mcp.Tool, error) {
	return nil, nil
}

func (m *MockToolRegistry) GetFactory(name string) (tools.ToolFactory, error) {
	return nil, nil
}

func (m *MockToolRegistry) List() []tools.ToolInfo {
	return m.toolList
}

func (m *MockToolRegistry) LoadTools(ctx context.Context) error {
	return nil
}

func (m *MockToolRegistry) ValidateTools(ctx context.Context) error {
	return nil
}

func (m *MockToolRegistry) TransitionStatus(name string, newStatus tools.ToolStatus) error {
	for i, tool := range m.toolList {
		if tool.Name == name {
			m.toolList[i].Status = newStatus
			return nil
		}
	}
	return tools.ErrToolNotFound
}

func (m *MockToolRegistry) RestartTool(ctx context.Context, name string) error {
	for i, tool := range m.toolList {
		if tool.Name == name {
			// Check if restart is allowed from current status
			if !tools.IsValidTransition(tool.Status, tools.ToolStatusRegistered) {
				return tools.ErrRestartNotAllowed
			}
			m.toolList[i].Status = tools.ToolStatusLoaded
			return nil
		}
	}
	return tools.ErrToolNotFound
}

func (m *MockToolRegistry) Start(ctx context.Context) error {
	return nil
}

func (m *MockToolRegistry) Stop(ctx context.Context) error {
	return nil
}

func (m *MockToolRegistry) Health() tools.RegistryHealth {
	return m.health
}

// =============================================================================
// Test setup factory functions
// =============================================================================

func createTestServer() *Server {
	cfg := &config.Config{
		Logger: config.LoggerConfig{
			Service: "test-service",
			Version: "test-version",
		},
	}
	
	loggerCfg := logger.Config{
		Level:     "info",
		Format:    "console",
		Service:   cfg.Logger.Service,
		Version:   cfg.Logger.Version,
		UseEmojis: false,
	}
	log, _ := logger.New(loggerCfg)
	
	server := &Server{
		logger:    log,
		config:    cfg,
		startTime: time.Now().Add(-1 * time.Hour),
	}
	
	return server
}

func createMockToolRegistry() *MockToolRegistry {
	return &MockToolRegistry{
		health: tools.RegistryHealth{
			Status:    "healthy",
			LastCheck: time.Now().UTC().Format(time.RFC3339),
		},
		toolList: []tools.ToolInfo{},
	}
}

func createMockToolRegistryWithTools(toolList []tools.ToolInfo) *MockToolRegistry {
	return &MockToolRegistry{
		health: tools.RegistryHealth{
			Status:    "healthy",
			LastCheck: time.Now().UTC().Format(time.RFC3339),
		},
		toolList: toolList,
	}
}

func createMockToolRegistryWithHealth(health tools.RegistryHealth, toolList []tools.ToolInfo) *MockToolRegistry {
	return &MockToolRegistry{
		health:   health,
		toolList: toolList,
	}
}

// =============================================================================
// Test data builder functions
// =============================================================================

func buildHealthyToolList() []tools.ToolInfo {
	return []tools.ToolInfo{
		{
			Name:         "test-tool-1",
			Status:       tools.ToolStatusActive,
			Description:  "Test tool 1",
			Version:      "1.0.0",
			Capabilities: []string{"read", "write"},
		},
		{
			Name:         "test-tool-2",
			Status:       tools.ToolStatusLoaded,
			Description:  "Test tool 2",
			Version:      "2.0.0",
			Capabilities: []string{"execute"},
		},
	}
}

func buildDegradedToolList() []tools.ToolInfo {
	return []tools.ToolInfo{
		{
			Name:         "test-tool-1",
			Status:       tools.ToolStatusActive,
			Description:  "Test tool 1",
			Version:      "1.0.0",
			Capabilities: []string{"read"},
		},
		{
			Name:         "test-tool-2",
			Status:       tools.ToolStatusError,
			Description:  "Test tool 2",
			Version:      "2.0.0",
			Capabilities: []string{"write"},
		},
	}
}

func buildRegisteredOnlyToolList() []tools.ToolInfo {
	return []tools.ToolInfo{
		{
			Name:         "test-tool-1",
			Status:       tools.ToolStatusRegistered,
			Description:  "Test tool 1",
			Version:      "1.0.0",
			Capabilities: []string{"read"},
		},
	}
}

func buildRegistryHealthData(status string) tools.RegistryHealth {
	return tools.RegistryHealth{
		Status:    status,
		LastCheck: time.Now().UTC().Format(time.RFC3339),
	}
}

// =============================================================================
// HTTP test helper functions
// =============================================================================

func executeToolsHealthRequest(server *Server) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "/tools/health", nil)
	w := httptest.NewRecorder()
	server.handleToolsHealth(w, req)
	return w
}

func validateJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedCode int) {
	if w.Code != expectedCode {
		t.Errorf("expected status code %d, got %d", expectedCode, w.Code)
	}
	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

func parseToolsHealthResponse(t *testing.T, w *httptest.ResponseRecorder) ToolsHealthResponse {
	var response ToolsHealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	return response
}

func validateToolsHealthResponse(t *testing.T, response ToolsHealthResponse, expectedStatus string, expectedToolCount int) {
	if response.Status != expectedStatus {
		t.Errorf("expected status %s, got %s", expectedStatus, response.Status)
	}
	if response.Summary.Total != expectedToolCount {
		t.Errorf("expected total tools %d, got %d", expectedToolCount, response.Summary.Total)
	}
	if len(response.Tools) != expectedToolCount {
		t.Errorf("expected %d tool details, got %d", expectedToolCount, len(response.Tools))
	}
	if _, err := time.Parse(time.RFC3339, response.Timestamp); err != nil {
		t.Errorf("invalid timestamp format: %s", response.Timestamp)
	}
}

func validateToolDetails(t *testing.T, response ToolsHealthResponse, expectedTools []tools.ToolInfo) {
	for _, expectedTool := range expectedTools {
		toolDetail, exists := response.Tools[expectedTool.Name]
		if !exists {
			t.Errorf("missing tool detail for %s", expectedTool.Name)
			continue
		}
		if toolDetail.Name != expectedTool.Name {
			t.Errorf("expected tool name %s, got %s", expectedTool.Name, toolDetail.Name)
		}
		if toolDetail.Status != string(expectedTool.Status) {
			t.Errorf("expected tool status %s, got %s", string(expectedTool.Status), toolDetail.Status)
		}
		if toolDetail.Description != expectedTool.Description {
			t.Errorf("expected description %s, got %s", expectedTool.Description, toolDetail.Description)
		}
		if toolDetail.Version != expectedTool.Version {
			t.Errorf("expected version %s, got %s", expectedTool.Version, toolDetail.Version)
		}
		if expectedTool.Status == tools.ToolStatusError {
			if toolDetail.ErrorMessage == "" {
				t.Errorf("expected error message for tool %s", expectedTool.Name)
			}
		}
	}
}

// =============================================================================
// Tools health endpoint tests (split by responsibility)
// =============================================================================

func TestHandleToolsHealth_HealthyTools(t *testing.T) {
	server := createTestServer()
	toolList := buildHealthyToolList()
	health := buildRegistryHealthData("healthy")
	server.toolRegistry = createMockToolRegistryWithHealth(health, toolList)

	w := executeToolsHealthRequest(server)
	validateJSONResponse(t, w, http.StatusOK)

	response := parseToolsHealthResponse(t, w)
	validateToolsHealthResponse(t, response, "healthy", len(toolList))
	validateToolDetails(t, response, toolList)
}

func TestHandleToolsHealth_DegradedTools(t *testing.T) {
	server := createTestServer()
	toolList := buildDegradedToolList()
	health := buildRegistryHealthData("healthy")
	server.toolRegistry = createMockToolRegistryWithHealth(health, toolList)

	w := executeToolsHealthRequest(server)
	validateJSONResponse(t, w, http.StatusOK)

	response := parseToolsHealthResponse(t, w)
	validateToolsHealthResponse(t, response, "degraded", len(toolList))
	validateToolDetails(t, response, toolList)
}

func TestHandleToolsHealth_StoppedRegistry(t *testing.T) {
	server := createTestServer()
	toolList := []tools.ToolInfo{}
	health := buildRegistryHealthData("stopped")
	server.toolRegistry = createMockToolRegistryWithHealth(health, toolList)

	w := executeToolsHealthRequest(server)
	validateJSONResponse(t, w, http.StatusOK)

	response := parseToolsHealthResponse(t, w)
	validateToolsHealthResponse(t, response, "stopped", 0)
}

func TestHandleToolsHealth_NoActiveTools(t *testing.T) {
	server := createTestServer()
	toolList := buildRegisteredOnlyToolList()
	health := buildRegistryHealthData("healthy")
	server.toolRegistry = createMockToolRegistryWithHealth(health, toolList)

	w := executeToolsHealthRequest(server)
	validateJSONResponse(t, w, http.StatusOK)

	response := parseToolsHealthResponse(t, w)
	validateToolsHealthResponse(t, response, "degraded", len(toolList))
	validateToolDetails(t, response, toolList)
}

func TestHandleToolsHealth_HTTPHeaders(t *testing.T) {
	server := createTestServer()
	server.toolRegistry = createMockToolRegistry()

	w := executeToolsHealthRequest(server)
	validateJSONResponse(t, w, http.StatusOK)
}

func TestHandleToolsHealth_JSONMarshaling(t *testing.T) {
	server := createTestServer()
	server.toolRegistry = createMockToolRegistry()

	w := executeToolsHealthRequest(server)
	response := parseToolsHealthResponse(t, w)

	if response.Status == "" {
		t.Error("response status should not be empty")
	}
	if response.Timestamp == "" {
		t.Error("response timestamp should not be empty")
	}
	if response.Summary.Total < 0 {
		t.Error("summary total should not be negative")
	}
	if response.Tools == nil {
		t.Error("tools map should not be nil")
	}
}

// =============================================================================
// Tool health summary calculation tests (split by responsibility)
// =============================================================================

func buildMixedStatusToolList() []tools.ToolInfo {
	return []tools.ToolInfo{
		{Status: tools.ToolStatusActive},
		{Status: tools.ToolStatusLoaded},
		{Status: tools.ToolStatusRegistered},
		{Status: tools.ToolStatusError},
		{Status: tools.ToolStatusDisabled},
		{Status: tools.ToolStatusActive},
	}
}

func buildAllActiveToolList() []tools.ToolInfo {
	return []tools.ToolInfo{
		{Status: tools.ToolStatusActive},
		{Status: tools.ToolStatusActive},
		{Status: tools.ToolStatusActive},
	}
}

func validateToolHealthSummary(t *testing.T, result, expected ToolHealthSummary) {
	if result.Total != expected.Total {
		t.Errorf("expected Total %d, got %d", expected.Total, result.Total)
	}
	if result.Active != expected.Active {
		t.Errorf("expected Active %d, got %d", expected.Active, result.Active)
	}
	if result.Loaded != expected.Loaded {
		t.Errorf("expected Loaded %d, got %d", expected.Loaded, result.Loaded)
	}
	if result.Registered != expected.Registered {
		t.Errorf("expected Registered %d, got %d", expected.Registered, result.Registered)
	}
	if result.Error != expected.Error {
		t.Errorf("expected Error %d, got %d", expected.Error, result.Error)
	}
	if result.Disabled != expected.Disabled {
		t.Errorf("expected Disabled %d, got %d", expected.Disabled, result.Disabled)
	}
}

func TestBuildToolHealthSummary_StatusCounting(t *testing.T) {
	server := createTestServer()
	toolList := buildMixedStatusToolList()
	expected := ToolHealthSummary{
		Total:      6,
		Active:     2,
		Loaded:     1,
		Registered: 1,
		Error:      1,
		Disabled:   1,
	}

	result := server.buildToolHealthSummary(toolList)
	validateToolHealthSummary(t, result, expected)
}

func TestBuildToolHealthSummary_EmptyInput(t *testing.T) {
	server := createTestServer()
	toolList := []tools.ToolInfo{}
	expected := ToolHealthSummary{
		Total:      0,
		Active:     0,
		Loaded:     0,
		Registered: 0,
		Error:      0,
		Disabled:   0,
	}

	result := server.buildToolHealthSummary(toolList)
	validateToolHealthSummary(t, result, expected)
}

func TestBuildToolHealthSummary_AllActiveTools(t *testing.T) {
	server := createTestServer()
	toolList := buildAllActiveToolList()
	expected := ToolHealthSummary{
		Total:      3,
		Active:     3,
		Loaded:     0,
		Registered: 0,
		Error:      0,
		Disabled:   0,
	}

	result := server.buildToolHealthSummary(toolList)
	validateToolHealthSummary(t, result, expected)
}

// =============================================================================
// Tool health determination tests (split by logic type)
// =============================================================================

func TestDetermineToolsOverallHealth_Logic(t *testing.T) {
	server := createTestServer()

	// Test healthy with active tools
	summary := ToolHealthSummary{Total: 2, Active: 2, Error: 0}
	registryHealth := tools.RegistryHealth{Status: "healthy"}
	result := server.determineToolsOverallHealth(summary, registryHealth)
	if result != "healthy" {
		t.Errorf("expected healthy, got %s", result)
	}

	// Test degraded with error tools
	summary = ToolHealthSummary{Total: 2, Active: 1, Error: 1}
	registryHealth = tools.RegistryHealth{Status: "healthy"}
	result = server.determineToolsOverallHealth(summary, registryHealth)
	if result != "degraded" {
		t.Errorf("expected degraded, got %s", result)
	}

	// Test degraded with no active tools
	summary = ToolHealthSummary{Total: 2, Active: 0, Registered: 2, Error: 0}
	registryHealth = tools.RegistryHealth{Status: "healthy"}
	result = server.determineToolsOverallHealth(summary, registryHealth)
	if result != "degraded" {
		t.Errorf("expected degraded, got %s", result)
	}
}

func TestDetermineToolsOverallHealth_EdgeCases(t *testing.T) {
	server := createTestServer()

	// Test stopped registry
	summary := ToolHealthSummary{Total: 1, Active: 1, Error: 0}
	registryHealth := tools.RegistryHealth{Status: "stopped"}
	result := server.determineToolsOverallHealth(summary, registryHealth)
	if result != "stopped" {
		t.Errorf("expected stopped, got %s", result)
	}

	// Test healthy with no tools
	summary = ToolHealthSummary{Total: 0, Active: 0, Error: 0}
	registryHealth = tools.RegistryHealth{Status: "healthy"}
	result = server.determineToolsOverallHealth(summary, registryHealth)
	if result != "healthy" {
		t.Errorf("expected healthy, got %s", result)
	}
}

// =============================================================================
// Tool validation logic tests (split by transition type)
// =============================================================================

func TestIsValidTransition_AllowedTransitions(t *testing.T) {
	allowedTransitions := []struct {
		from tools.ToolStatus
		to   tools.ToolStatus
	}{
		{tools.ToolStatusActive, tools.ToolStatusActive},
		{tools.ToolStatusRegistered, tools.ToolStatusLoaded},
		{tools.ToolStatusRegistered, tools.ToolStatusError},
		{tools.ToolStatusRegistered, tools.ToolStatusDisabled},
		{tools.ToolStatusLoaded, tools.ToolStatusActive},
		{tools.ToolStatusLoaded, tools.ToolStatusError},
		{tools.ToolStatusLoaded, tools.ToolStatusDisabled},
		{tools.ToolStatusActive, tools.ToolStatusError},
		{tools.ToolStatusActive, tools.ToolStatusDisabled},
		{tools.ToolStatusActive, tools.ToolStatusLoaded},
		{tools.ToolStatusError, tools.ToolStatusRegistered},
		{tools.ToolStatusError, tools.ToolStatusDisabled},
		{tools.ToolStatusDisabled, tools.ToolStatusRegistered},
		{tools.ToolStatusDisabled, tools.ToolStatusError},
	}

	for _, transition := range allowedTransitions {
		result := tools.IsValidTransition(transition.from, transition.to)
		if !result {
			t.Errorf("IsValidTransition(%s, %s) should be true",
				string(transition.from), string(transition.to))
		}
	}
}

func TestIsValidTransition_DisallowedTransitions(t *testing.T) {
	disallowedTransitions := []struct {
		from tools.ToolStatus
		to   tools.ToolStatus
	}{
		{tools.ToolStatusRegistered, tools.ToolStatusActive},
		{tools.ToolStatusLoaded, tools.ToolStatusRegistered},
		{tools.ToolStatusActive, tools.ToolStatusRegistered},
		{tools.ToolStatusError, tools.ToolStatusLoaded},
		{tools.ToolStatusError, tools.ToolStatusActive},
		{tools.ToolStatusDisabled, tools.ToolStatusLoaded},
		{tools.ToolStatusDisabled, tools.ToolStatusActive},
	}

	for _, transition := range disallowedTransitions {
		result := tools.IsValidTransition(transition.from, transition.to)
		if result {
			t.Errorf("IsValidTransition(%s, %s) should be false",
				string(transition.from), string(transition.to))
		}
	}
}

func validateAllowedTransitions(t *testing.T, from tools.ToolStatus, expected []tools.ToolStatus) {
	result := tools.GetAllowedTransitions(from)
	
	if len(result) != len(expected) {
		t.Errorf("GetAllowedTransitions(%s) returned %d statuses, expected %d",
			string(from), len(result), len(expected))
		t.Errorf("Got: %v", result)
		t.Errorf("Expected: %v", expected)
		return
	}

	expectedMap := make(map[tools.ToolStatus]bool)
	for _, status := range expected {
		expectedMap[status] = true
	}

	for _, status := range result {
		if !expectedMap[status] {
			t.Errorf("GetAllowedTransitions(%s) contains unexpected status: %s",
				string(from), string(status))
		}
		delete(expectedMap, status)
	}

	if len(expectedMap) > 0 {
		t.Errorf("GetAllowedTransitions(%s) missing expected statuses: %v",
			string(from), expectedMap)
	}
}

func TestGetAllowedTransitions_ByStatus(t *testing.T) {
	// Test transitions from registered status
	expected := []tools.ToolStatus{
		tools.ToolStatusRegistered,
		tools.ToolStatusLoaded,
		tools.ToolStatusError,
		tools.ToolStatusDisabled,
	}
	validateAllowedTransitions(t, tools.ToolStatusRegistered, expected)

	// Test transitions from loaded status
	expected = []tools.ToolStatus{
		tools.ToolStatusLoaded,
		tools.ToolStatusActive,
		tools.ToolStatusError,
		tools.ToolStatusDisabled,
	}
	validateAllowedTransitions(t, tools.ToolStatusLoaded, expected)

	// Test transitions from active status
	expected = []tools.ToolStatus{
		tools.ToolStatusActive,
		tools.ToolStatusError,
		tools.ToolStatusDisabled,
		tools.ToolStatusLoaded,
	}
	validateAllowedTransitions(t, tools.ToolStatusActive, expected)

	// Test transitions from error status
	expected = []tools.ToolStatus{
		tools.ToolStatusError,
		tools.ToolStatusRegistered,
		tools.ToolStatusDisabled,
	}
	validateAllowedTransitions(t, tools.ToolStatusError, expected)

	// Test transitions from disabled status
	expected = []tools.ToolStatus{
		tools.ToolStatusDisabled,
		tools.ToolStatusRegistered,
		tools.ToolStatusError,
	}
	validateAllowedTransitions(t, tools.ToolStatusDisabled, expected)
}

// =============================================================================
// Tool restart tests (split by scenario type)
// =============================================================================

func createMockToolRegistryWithSingleTool(toolName string, status tools.ToolStatus) *MockToolRegistry {
	return &MockToolRegistry{
		health: tools.RegistryHealth{
			Status:    "healthy",
			LastCheck: time.Now().UTC().Format(time.RFC3339),
		},
		toolList: []tools.ToolInfo{
			{
				Name:   toolName,
				Status: status,
			},
		},
	}
}

func validateRestartSuccess(t *testing.T, registry *MockToolRegistry, toolName string, expectedStatus tools.ToolStatus) {
	toolList := registry.List()
	found := false
	for _, tool := range toolList {
		if tool.Name == toolName {
			found = true
			if tool.Status != expectedStatus {
				t.Errorf("expected final status %s, got %s",
					string(expectedStatus), string(tool.Status))
			}
			break
		}
	}
	if !found {
		t.Errorf("tool %s not found after restart", toolName)
	}
}

func TestRestartTool_SuccessfulRestart(t *testing.T) {
	server := createTestServer()

	// Test restart from error status
	registry := createMockToolRegistryWithSingleTool("test-tool", tools.ToolStatusError)
	server.toolRegistry = registry
	err := server.toolRegistry.RestartTool(context.Background(), "test-tool")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	validateRestartSuccess(t, registry, "test-tool", tools.ToolStatusLoaded)

	// Test restart from disabled status
	registry = createMockToolRegistryWithSingleTool("test-tool", tools.ToolStatusDisabled)
	server.toolRegistry = registry
	err = server.toolRegistry.RestartTool(context.Background(), "test-tool")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	validateRestartSuccess(t, registry, "test-tool", tools.ToolStatusLoaded)
}

func TestRestartTool_InvalidStatus(t *testing.T) {
	server := createTestServer()
	registry := createMockToolRegistryWithSingleTool("test-tool", tools.ToolStatusActive)
	server.toolRegistry = registry

	err := server.toolRegistry.RestartTool(context.Background(), "test-tool")
	if err == nil {
		t.Error("expected error but got none")
	}
	if !errors.Is(err, tools.ErrRestartNotAllowed) {
		t.Errorf("expected ErrRestartNotAllowed, got %v", err)
	}
}

func TestRestartTool_NonExistentTool(t *testing.T) {
	server := createTestServer()
	server.toolRegistry = createMockToolRegistry()

	err := server.toolRegistry.RestartTool(context.Background(), "non-existent")
	if err == nil {
		t.Error("expected error but got none")
	}
	if !errors.Is(err, tools.ErrToolNotFound) {
		t.Errorf("expected ErrToolNotFound, got %v", err)
	}
}

func TestRestartTool_StatusValidation(t *testing.T) {
	server := createTestServer()
	registry := createMockToolRegistryWithSingleTool("test-tool", tools.ToolStatusError)
	server.toolRegistry = registry

	err := server.toolRegistry.RestartTool(context.Background(), "test-tool")
	if err != nil {
		t.Fatalf("unexpected error during restart: %v", err)
	}

	toolList := server.toolRegistry.List()
	for _, tool := range toolList {
		if tool.Name == "test-tool" {
			if tool.Status != tools.ToolStatusLoaded {
				t.Errorf("expected final status %s, got %s",
					string(tools.ToolStatusLoaded), string(tool.Status))
			}
			break
		}
	}
}
