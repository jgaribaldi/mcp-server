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

func TestHandleToolsHealth(t *testing.T) {
	tests := []struct {
		name           string
		registryHealth tools.RegistryHealth
		toolList       []tools.ToolInfo
		expectedStatus string
		expectedCode   int
	}{
		{
			name: "healthy tools",
			registryHealth: tools.RegistryHealth{
				Status:    "healthy",
				LastCheck: time.Now().UTC().Format(time.RFC3339),
			},
			toolList: []tools.ToolInfo{
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
			},
			expectedStatus: "healthy",
			expectedCode:   http.StatusOK,
		},
		{
			name: "degraded tools with errors",
			registryHealth: tools.RegistryHealth{
				Status:    "healthy",
				LastCheck: time.Now().UTC().Format(time.RFC3339),
			},
			toolList: []tools.ToolInfo{
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
			},
			expectedStatus: "degraded",
			expectedCode:   http.StatusOK,
		},
		{
			name: "stopped registry",
			registryHealth: tools.RegistryHealth{
				Status:    "stopped",
				LastCheck: time.Now().UTC().Format(time.RFC3339),
			},
			toolList:       []tools.ToolInfo{},
			expectedStatus: "stopped",
			expectedCode:   http.StatusOK,
		},
		{
			name: "no active tools",
			registryHealth: tools.RegistryHealth{
				Status:    "healthy",
				LastCheck: time.Now().UTC().Format(time.RFC3339),
			},
			toolList: []tools.ToolInfo{
				{
					Name:         "test-tool-1",
					Status:       tools.ToolStatusRegistered,
					Description:  "Test tool 1",
					Version:      "1.0.0",
					Capabilities: []string{"read"},
				},
			},
			expectedStatus: "degraded",
			expectedCode:   http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createTestServer()
			server.toolRegistry = &MockToolRegistry{
				health:   tt.registryHealth,
				toolList: tt.toolList,
			}

			req := httptest.NewRequest("GET", "/tools/health", nil)
			w := httptest.NewRecorder()

			server.handleToolsHealth(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status code %d, got %d", tt.expectedCode, w.Code)
			}

			if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			var response ToolsHealthResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if response.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, response.Status)
			}

			if response.Summary.Total != len(tt.toolList) {
				t.Errorf("expected total tools %d, got %d", len(tt.toolList), response.Summary.Total)
			}

			if len(response.Tools) != len(tt.toolList) {
				t.Errorf("expected %d tool details, got %d", len(tt.toolList), len(response.Tools))
			}

			for _, expectedTool := range tt.toolList {
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

			if _, err := time.Parse(time.RFC3339, response.Timestamp); err != nil {
				t.Errorf("invalid timestamp format: %s", response.Timestamp)
			}
		})
	}
}

func TestBuildToolHealthSummary(t *testing.T) {
	server := createTestServer()

	tests := []struct {
		name     string
		toolList []tools.ToolInfo
		expected ToolHealthSummary
	}{
		{
			name: "mixed tool statuses",
			toolList: []tools.ToolInfo{
				{Status: tools.ToolStatusActive},
				{Status: tools.ToolStatusLoaded},
				{Status: tools.ToolStatusRegistered},
				{Status: tools.ToolStatusError},
				{Status: tools.ToolStatusDisabled},
				{Status: tools.ToolStatusActive},
			},
			expected: ToolHealthSummary{
				Total:      6,
				Active:     2,
				Loaded:     1,
				Registered: 1,
				Error:      1,
				Disabled:   1,
			},
		},
		{
			name:     "empty tool list",
			toolList: []tools.ToolInfo{},
			expected: ToolHealthSummary{
				Total:      0,
				Active:     0,
				Loaded:     0,
				Registered: 0,
				Error:      0,
				Disabled:   0,
			},
		},
		{
			name: "all active tools",
			toolList: []tools.ToolInfo{
				{Status: tools.ToolStatusActive},
				{Status: tools.ToolStatusActive},
				{Status: tools.ToolStatusActive},
			},
			expected: ToolHealthSummary{
				Total:      3,
				Active:     3,
				Loaded:     0,
				Registered: 0,
				Error:      0,
				Disabled:   0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.buildToolHealthSummary(tt.toolList)

			if result.Total != tt.expected.Total {
				t.Errorf("expected Total %d, got %d", tt.expected.Total, result.Total)
			}
			if result.Active != tt.expected.Active {
				t.Errorf("expected Active %d, got %d", tt.expected.Active, result.Active)
			}
			if result.Loaded != tt.expected.Loaded {
				t.Errorf("expected Loaded %d, got %d", tt.expected.Loaded, result.Loaded)
			}
			if result.Registered != tt.expected.Registered {
				t.Errorf("expected Registered %d, got %d", tt.expected.Registered, result.Registered)
			}
			if result.Error != tt.expected.Error {
				t.Errorf("expected Error %d, got %d", tt.expected.Error, result.Error)
			}
			if result.Disabled != tt.expected.Disabled {
				t.Errorf("expected Disabled %d, got %d", tt.expected.Disabled, result.Disabled)
			}
		})
	}
}

func TestDetermineToolsOverallHealth(t *testing.T) {
	server := createTestServer()

	tests := []struct {
		name           string
		summary        ToolHealthSummary
		registryHealth tools.RegistryHealth
		expected       string
	}{
		{
			name: "healthy with active tools",
			summary: ToolHealthSummary{
				Total:  2,
				Active: 2,
				Error:  0,
			},
			registryHealth: tools.RegistryHealth{Status: "healthy"},
			expected:       "healthy",
		},
		{
			name: "degraded with error tools",
			summary: ToolHealthSummary{
				Total:  2,
				Active: 1,
				Error:  1,
			},
			registryHealth: tools.RegistryHealth{Status: "healthy"},
			expected:       "degraded",
		},
		{
			name: "degraded with no active tools",
			summary: ToolHealthSummary{
				Total:      2,
				Active:     0,
				Registered: 2,
				Error:      0,
			},
			registryHealth: tools.RegistryHealth{Status: "healthy"},
			expected:       "degraded",
		},
		{
			name: "stopped registry",
			summary: ToolHealthSummary{
				Total:  1,
				Active: 1,
				Error:  0,
			},
			registryHealth: tools.RegistryHealth{Status: "stopped"},
			expected:       "stopped",
		},
		{
			name: "healthy with no tools",
			summary: ToolHealthSummary{
				Total:  0,
				Active: 0,
				Error:  0,
			},
			registryHealth: tools.RegistryHealth{Status: "healthy"},
			expected:       "healthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.determineToolsOverallHealth(tt.summary, tt.registryHealth)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		name     string
		from     tools.ToolStatus
		to       tools.ToolStatus
		expected bool
	}{
		{"same status", tools.ToolStatusActive, tools.ToolStatusActive, true},
		{"registered to loaded", tools.ToolStatusRegistered, tools.ToolStatusLoaded, true},
		{"registered to error", tools.ToolStatusRegistered, tools.ToolStatusError, true},
		{"registered to disabled", tools.ToolStatusRegistered, tools.ToolStatusDisabled, true},
		{"loaded to active", tools.ToolStatusLoaded, tools.ToolStatusActive, true},
		{"loaded to error", tools.ToolStatusLoaded, tools.ToolStatusError, true},
		{"loaded to disabled", tools.ToolStatusLoaded, tools.ToolStatusDisabled, true},
		{"active to error", tools.ToolStatusActive, tools.ToolStatusError, true},
		{"active to disabled", tools.ToolStatusActive, tools.ToolStatusDisabled, true},
		{"active to loaded", tools.ToolStatusActive, tools.ToolStatusLoaded, true},
		{"error to registered", tools.ToolStatusError, tools.ToolStatusRegistered, true},
		{"error to disabled", tools.ToolStatusError, tools.ToolStatusDisabled, true},
		{"disabled to registered", tools.ToolStatusDisabled, tools.ToolStatusRegistered, true},
		{"disabled to error", tools.ToolStatusDisabled, tools.ToolStatusError, true},
		{"invalid: registered to active", tools.ToolStatusRegistered, tools.ToolStatusActive, false},
		{"invalid: loaded to registered", tools.ToolStatusLoaded, tools.ToolStatusRegistered, false},
		{"invalid: active to registered", tools.ToolStatusActive, tools.ToolStatusRegistered, false},
		{"invalid: error to loaded", tools.ToolStatusError, tools.ToolStatusLoaded, false},
		{"invalid: error to active", tools.ToolStatusError, tools.ToolStatusActive, false},
		{"invalid: disabled to loaded", tools.ToolStatusDisabled, tools.ToolStatusLoaded, false},
		{"invalid: disabled to active", tools.ToolStatusDisabled, tools.ToolStatusActive, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tools.IsValidTransition(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("IsValidTransition(%s, %s) = %v, expected %v", 
					string(tt.from), string(tt.to), result, tt.expected)
			}
		})
	}
}

func TestGetAllowedTransitions(t *testing.T) {
	tests := []struct {
		name     string
		from     tools.ToolStatus
		expected []tools.ToolStatus
	}{
		{
			name: "from registered",
			from: tools.ToolStatusRegistered,
			expected: []tools.ToolStatus{
				tools.ToolStatusRegistered,
				tools.ToolStatusLoaded,
				tools.ToolStatusError,
				tools.ToolStatusDisabled,
			},
		},
		{
			name: "from loaded",
			from: tools.ToolStatusLoaded,
			expected: []tools.ToolStatus{
				tools.ToolStatusLoaded,
				tools.ToolStatusActive,
				tools.ToolStatusError,
				tools.ToolStatusDisabled,
			},
		},
		{
			name: "from active",
			from: tools.ToolStatusActive,
			expected: []tools.ToolStatus{
				tools.ToolStatusActive,
				tools.ToolStatusError,
				tools.ToolStatusDisabled,
				tools.ToolStatusLoaded,
			},
		},
		{
			name: "from error",
			from: tools.ToolStatusError,
			expected: []tools.ToolStatus{
				tools.ToolStatusError,
				tools.ToolStatusRegistered,
				tools.ToolStatusDisabled,
			},
		},
		{
			name: "from disabled",
			from: tools.ToolStatusDisabled,
			expected: []tools.ToolStatus{
				tools.ToolStatusDisabled,
				tools.ToolStatusRegistered,
				tools.ToolStatusError,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tools.GetAllowedTransitions(tt.from)
			
			if len(result) != len(tt.expected) {
				t.Errorf("GetAllowedTransitions(%s) returned %d statuses, expected %d",
					string(tt.from), len(result), len(tt.expected))
				t.Errorf("Got: %v", result)
				t.Errorf("Expected: %v", tt.expected)
				return
			}

			// Check all expected statuses are present
			expectedMap := make(map[tools.ToolStatus]bool)
			for _, status := range tt.expected {
				expectedMap[status] = true
			}

			for _, status := range result {
				if !expectedMap[status] {
					t.Errorf("GetAllowedTransitions(%s) contains unexpected status: %s",
						string(tt.from), string(status))
				}
				delete(expectedMap, status)
			}

			if len(expectedMap) > 0 {
				t.Errorf("GetAllowedTransitions(%s) missing expected statuses: %v",
					string(tt.from), expectedMap)
			}
		})
	}
}

func TestRestartTool(t *testing.T) {
	tests := []struct {
		name                string
		toolName           string
		initialStatus      tools.ToolStatus
		expectedFinalStatus tools.ToolStatus
		expectError        bool
		expectedError      error
	}{
		{
			name:                "restart from error status",
			toolName:           "test-tool",
			initialStatus:      tools.ToolStatusError,
			expectedFinalStatus: tools.ToolStatusLoaded,
			expectError:        false,
		},
		{
			name:                "restart from disabled status", 
			toolName:           "test-tool",
			initialStatus:      tools.ToolStatusDisabled,
			expectedFinalStatus: tools.ToolStatusLoaded,
			expectError:        false,
		},
		{
			name:          "restart non-existent tool",
			toolName:     "non-existent",
			initialStatus: tools.ToolStatusError,
			expectError:   true,
			expectedError: tools.ErrToolNotFound,
		},
		{
			name:          "restart from invalid status",
			toolName:     "test-tool",
			initialStatus: tools.ToolStatusActive,
			expectError:   true,
			expectedError: tools.ErrRestartNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createTestServer()
			
			if tt.toolName == "test-tool" {
				server.toolRegistry = &MockToolRegistry{
					toolList: []tools.ToolInfo{
						{
							Name:   tt.toolName,
							Status: tt.initialStatus,
						},
					},
				}
			} else {
				server.toolRegistry = &MockToolRegistry{
					toolList: []tools.ToolInfo{},
				}
			}

			err := server.toolRegistry.RestartTool(context.Background(), tt.toolName)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.expectedError != nil && !errors.Is(err, tt.expectedError) {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}

				// Check final status for successful restarts
				toolList := server.toolRegistry.List()
				found := false
				for _, tool := range toolList {
					if tool.Name == tt.toolName {
						found = true
						if tool.Status != tt.expectedFinalStatus {
							t.Errorf("expected final status %s, got %s", 
								string(tt.expectedFinalStatus), string(tool.Status))
						}
						break
					}
				}
				if !found {
					t.Errorf("tool %s not found after restart", tt.toolName)
				}
			}
		})
	}
}

func TestRestartToolStatusTransitions(t *testing.T) {
	server := createTestServer()
	
	mockRegistry := &MockToolRegistry{
		toolList: []tools.ToolInfo{
			{
				Name:   "test-tool",
				Status: tools.ToolStatusError,
			},
		},
	}
	
	server.toolRegistry = mockRegistry

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