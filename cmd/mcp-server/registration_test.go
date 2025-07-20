package main

import (
	"errors"
	"testing"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
	"mcp-server/internal/tools"
)

func TestRegisterEchoTool(t *testing.T) {
	cfg := &config.Config{}
	log, err := logger.NewDefault()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	registry := tools.NewDefaultToolRegistry(cfg, log)
	
	err = registerEchoTool(registry, log)
	if err != nil {
		t.Fatalf("Expected successful registration, got error: %v", err)
	}
	
	toolList := registry.List()
	found := false
	for _, tool := range toolList {
		if tool.Name == "echo" {
			found = true
			
			if tool.Description != "Simple text manipulation tool for testing and demonstration" {
				t.Errorf("Expected description 'Simple text manipulation tool for testing and demonstration', got '%s'", tool.Description)
			}
			
			if tool.Version != "1.0.0" {
				t.Errorf("Expected version '1.0.0', got '%s'", tool.Version)
			}
			
			expectedCapabilities := []string{"text_processing", "demonstration"}
			if len(tool.Capabilities) != len(expectedCapabilities) {
				t.Errorf("Expected %d capabilities, got %d", len(expectedCapabilities), len(tool.Capabilities))
			} else {
				for i, cap := range expectedCapabilities {
					if i >= len(tool.Capabilities) || tool.Capabilities[i] != cap {
						t.Errorf("Expected capability '%s' at index %d, got '%s'", cap, i, tool.Capabilities[i])
					}
				}
			}
			
			expectedRequirements := map[string]string{"runtime": "go"}
			if len(tool.Requirements) != len(expectedRequirements) {
				t.Errorf("Expected %d requirements, got %d", len(expectedRequirements), len(tool.Requirements))
			} else {
				for key, expectedValue := range expectedRequirements {
					if actualValue, exists := tool.Requirements[key]; !exists {
						t.Errorf("Expected requirement '%s' not found", key)
					} else if actualValue != expectedValue {
						t.Errorf("Expected requirement '%s' = '%s', got '%s'", key, expectedValue, actualValue)
					}
				}
			}
			
			if tool.Status != tools.ToolStatusRegistered {
				t.Errorf("Expected status '%s', got '%s'", tools.ToolStatusRegistered, tool.Status)
			}
			
			break
		}
	}
	
	if !found {
		t.Fatal("Echo tool not found in registry list after registration")
	}
}

func TestRegisterEchoToolDuplicate(t *testing.T) {
	cfg := &config.Config{}
	log, err := logger.NewDefault()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	registry := tools.NewDefaultToolRegistry(cfg, log)
	
	err = registerEchoTool(registry, log)
	if err != nil {
		t.Fatalf("Expected successful first registration, got error: %v", err)
	}
	
	err = registerEchoTool(registry, log)
	if err == nil {
		t.Fatal("Expected error on duplicate registration, got nil")
	}
	
	if !errors.Is(err, tools.ErrToolAlreadyExists) {
		t.Errorf("Expected ErrToolAlreadyExists, got: %v", err)
	}
}

func TestRegisterAllTools(t *testing.T) {
	cfg := &config.Config{}
	log, err := logger.NewDefault()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	registry := tools.NewDefaultToolRegistry(cfg, log)
	
	err = registerAllTools(registry, log)
	if err != nil {
		t.Fatalf("Expected successful registration of all tools, got error: %v", err)
	}
	
	toolList := registry.List()
	if len(toolList) == 0 {
		t.Fatal("Expected at least one tool to be registered")
	}
	
	echoFound := false
	for _, tool := range toolList {
		if tool.Name == "echo" {
			echoFound = true
			break
		}
	}
	
	if !echoFound {
		t.Fatal("Echo tool not found after registering all tools")
	}
}

func TestEchoFactoryDirectly(t *testing.T) {
	cfg := &config.Config{}
	log, err := logger.NewDefault()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	registry := tools.NewDefaultToolRegistry(cfg, log)
	
	err = registerEchoTool(registry, log)
	if err != nil {
		t.Fatalf("Expected successful registration, got error: %v", err)
	}
	
	factory, err := registry.GetFactory("echo")
	if err != nil {
		t.Fatalf("Expected to get echo factory, got error: %v", err)
	}
	
	if factory.Name() != "echo" {
		t.Errorf("Expected factory name 'echo', got '%s'", factory.Name())
	}
	
	if factory.Description() != "Simple text manipulation tool for testing and demonstration" {
		t.Errorf("Expected factory description 'Simple text manipulation tool for testing and demonstration', got '%s'", factory.Description())
	}
	
	if factory.Version() != "1.0.0" {
		t.Errorf("Expected factory version '1.0.0', got '%s'", factory.Version())
	}
}