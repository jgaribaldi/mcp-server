package files

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
	"mcp-server/internal/resources"
)

func createTestFactoryConfig(log *logger.Logger) FileSystemFactoryConfig {
	tmpDir, _ := os.MkdirTemp("", "factory_test")
	
	return FileSystemFactoryConfig{
		Name:               "test-factory",
		Description:        "Test file system factory",
		Version:            "1.0.0",
		BaseURI:            "file://",
		AllowedDirectories: []string{tmpDir},
		MaxFileSize:        1024,
		AllowedExtensions:  []string{".txt", ".json"},
		BlockedPatterns:    []string{"secret"},
		Logger:            log,
	}
}

func createTestFactoryResourceConfig(filePath string) resources.ResourceConfig {
	return resources.ResourceConfig{
		Enabled: true,
		Config: map[string]interface{}{
			"file_path": filePath,
			"name":      "test-resource",
			"description": "Test resource created by factory",
		},
		CacheTimeout:  300,
		AccessControl: make(map[string]string),
	}
}

func TestNewFileSystemResourceFactory(t *testing.T) {
	log := createTestLogger(t)
	config := createTestFactoryConfig(log)
	defer os.RemoveAll(config.AllowedDirectories[0])

	factory, err := NewFileSystemResourceFactory(config)
	if err != nil {
		t.Fatalf("NewFileSystemResourceFactory failed: %v", err)
	}

	if factory == nil {
		t.Fatal("NewFileSystemResourceFactory returned nil")
	}

	// Verify interface compliance
	var _ resources.ResourceFactory = factory

	// Test factory properties
	if factory.Name() != "test-factory" {
		t.Errorf("Expected name 'test-factory', got '%s'", factory.Name())
	}

	if factory.Version() != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", factory.Version())
	}

	if factory.URI() != "file://" {
		t.Errorf("Expected URI 'file://', got '%s'", factory.URI())
	}
}

func TestNewFileSystemResourceFactory_Defaults(t *testing.T) {
	log := createTestLogger(t)
	config := FileSystemFactoryConfig{
		Logger: log,
		// Leave other fields empty to test defaults
	}

	factory, err := NewFileSystemResourceFactory(config)
	if err != nil {
		t.Fatalf("NewFileSystemResourceFactory with defaults failed: %v", err)
	}

	if factory.Name() != "file-system" {
		t.Errorf("Expected default name 'file-system', got '%s'", factory.Name())
	}

	if factory.Version() != "1.0.0" {
		t.Errorf("Expected default version '1.0.0', got '%s'", factory.Version())
	}

	if factory.URI() != "file://" {
		t.Errorf("Expected default URI 'file://', got '%s'", factory.URI())
	}

	if factory.GetMaxFileSize() != 10*1024*1024 {
		t.Errorf("Expected default max file size 10MB, got %d", factory.GetMaxFileSize())
	}
}

func TestNewFileSystemResourceFactory_ValidationErrors(t *testing.T) {
	log := createTestLogger(t)

	tests := []struct {
		name   string
		config FileSystemFactoryConfig
	}{
		{
			name: "nil logger",
			config: FileSystemFactoryConfig{
				Name: "test",
			},
		},
		{
			name: "too large max file size",
			config: FileSystemFactoryConfig{
				Logger:      log,
				MaxFileSize: 200 * 1024 * 1024, // 200MB
			},
		},
		{
			name: "relative allowed directory",
			config: FileSystemFactoryConfig{
				Logger:             log,
				AllowedDirectories: []string{"relative/path"},
			},
		},
		{
			name: "invalid extension format",
			config: FileSystemFactoryConfig{
				Logger:            log,
				AllowedExtensions: []string{"txt"}, // Missing dot
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewFileSystemResourceFactory(tt.config)
			if err == nil {
				t.Errorf("Expected error for %s", tt.name)
			}
		})
	}
}

func TestFileSystemResourceFactory_InterfaceMethods(t *testing.T) {
	log := createTestLogger(t)
	config := createTestFactoryConfig(log)
	defer os.RemoveAll(config.AllowedDirectories[0])

	factory, err := NewFileSystemResourceFactory(config)
	if err != nil {
		t.Fatalf("NewFileSystemResourceFactory failed: %v", err)
	}

	// Test Description
	if factory.Description() == "" {
		t.Error("Description should not be empty")
	}

	// Test MimeType
	if factory.MimeType() != "application/octet-stream" {
		t.Errorf("Expected default MIME type 'application/octet-stream', got '%s'", factory.MimeType())
	}

	// Test Tags
	tags := factory.Tags()
	expectedTags := []string{"filesystem", "file", "resource", "secure"}
	if len(tags) != len(expectedTags) {
		t.Errorf("Expected %d tags, got %d", len(expectedTags), len(tags))
	}

	// Test Capabilities
	capabilities := factory.Capabilities()
	if len(capabilities) == 0 {
		t.Error("Capabilities should not be empty")
	}

	expectedCapabilities := []string{"read-file", "validate-path", "security-validation", "audit-logging", "mime-detection"}
	for _, expected := range expectedCapabilities {
		found := false
		for _, capability := range capabilities {
			if capability == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected capability '%s' not found", expected)
		}
	}
}

func TestFileSystemResourceFactory_Create_Success(t *testing.T) {
	log := createTestLogger(t)
	config := createTestFactoryConfig(log)
	defer os.RemoveAll(config.AllowedDirectories[0])

	factory, err := NewFileSystemResourceFactory(config)
	if err != nil {
		t.Fatalf("NewFileSystemResourceFactory failed: %v", err)
	}

	// Create test file
	testFile := filepath.Join(config.AllowedDirectories[0], "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	resourceConfig := createTestFactoryResourceConfig(testFile)
	ctx := context.Background()

	resource, err := factory.Create(ctx, resourceConfig)
	if err != nil {
		t.Fatalf("Factory Create failed: %v", err)
	}

	if resource == nil {
		t.Fatal("Factory Create returned nil resource")
	}

	// Verify interface compliance
	var _ mcp.Resource = resource

	// Test resource properties
	if resource.Name() != "test-resource" {
		t.Errorf("Expected resource name 'test-resource', got '%s'", resource.Name())
	}

	if !strings.Contains(resource.URI(), "test.txt") {
		t.Errorf("Expected URI to contain 'test.txt', got '%s'", resource.URI())
	}
}

func TestFileSystemResourceFactory_Create_DisabledResource(t *testing.T) {
	log := createTestLogger(t)
	config := createTestFactoryConfig(log)
	defer os.RemoveAll(config.AllowedDirectories[0])

	factory, err := NewFileSystemResourceFactory(config)
	if err != nil {
		t.Fatalf("NewFileSystemResourceFactory failed: %v", err)
	}

	resourceConfig := resources.ResourceConfig{
		Enabled: false, // Disabled
		Config: map[string]interface{}{
			"file_path": "/some/path",
		},
	}

	ctx := context.Background()
	_, err = factory.Create(ctx, resourceConfig)
	if err == nil {
		t.Error("Expected error for disabled resource")
	}

	if !strings.Contains(err.Error(), "disabled") {
		t.Errorf("Expected error about disabled resource, got: %v", err)
	}
}

func TestFileSystemResourceFactory_Create_ConfigurationErrors(t *testing.T) {
	log := createTestLogger(t)
	config := createTestFactoryConfig(log)
	defer os.RemoveAll(config.AllowedDirectories[0])

	factory, err := NewFileSystemResourceFactory(config)
	if err != nil {
		t.Fatalf("NewFileSystemResourceFactory failed: %v", err)
	}

	tests := []struct {
		name           string
		resourceConfig resources.ResourceConfig
	}{
		{
			name: "missing file path",
			resourceConfig: resources.ResourceConfig{
				Enabled: true,
				Config:  map[string]interface{}{},
			},
		},
		{
			name: "empty file path",
			resourceConfig: resources.ResourceConfig{
				Enabled: true,
				Config: map[string]interface{}{
					"file_path": "",
				},
			},
		},
		{
			name: "invalid file path type",
			resourceConfig: resources.ResourceConfig{
				Enabled: true,
				Config: map[string]interface{}{
					"file_path": 123,
				},
			},
		},
		{
			name: "non-existent file",
			resourceConfig: resources.ResourceConfig{
				Enabled: true,
				Config: map[string]interface{}{
					"file_path": "/nonexistent/file.txt",
				},
			},
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := factory.Create(ctx, tt.resourceConfig)
			if err == nil {
				t.Errorf("Expected error for %s", tt.name)
			}
		})
	}
}

func TestFileSystemResourceFactory_Validate(t *testing.T) {
	log := createTestLogger(t)
	config := createTestFactoryConfig(log)
	defer os.RemoveAll(config.AllowedDirectories[0])

	factory, err := NewFileSystemResourceFactory(config)
	if err != nil {
		t.Fatalf("NewFileSystemResourceFactory failed: %v", err)
	}

	// Create test file
	testFile := filepath.Join(config.AllowedDirectories[0], "validate.txt")
	if err := os.WriteFile(testFile, []byte("validate content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test valid configuration
	validConfig := createTestFactoryResourceConfig(testFile)
	if err := factory.Validate(validConfig); err != nil {
		t.Errorf("Validation failed for valid config: %v", err)
	}

	// Test disabled resource (should pass validation)
	disabledConfig := resources.ResourceConfig{
		Enabled: false,
		Config:  map[string]interface{}{},
	}
	if err := factory.Validate(disabledConfig); err != nil {
		t.Errorf("Validation failed for disabled resource: %v", err)
	}

	// Test invalid configurations
	invalidConfigs := []struct {
		name   string
		config resources.ResourceConfig
	}{
		{
			name: "missing file path",
			config: resources.ResourceConfig{
				Enabled: true,
				Config:  map[string]interface{}{},
			},
		},
		{
			name: "disallowed directory",
			config: resources.ResourceConfig{
				Enabled: true,
				Config: map[string]interface{}{
					"file_path": "/etc/passwd",
				},
			},
		},
	}

	for _, tc := range invalidConfigs {
		t.Run(tc.name, func(t *testing.T) {
			err := factory.Validate(tc.config)
			if err == nil {
				t.Errorf("Expected validation error for %s", tc.name)
			}
		})
	}
}

func TestFileSystemResourceFactory_ExtractFilePath(t *testing.T) {
	log := createTestLogger(t)
	config := createTestFactoryConfig(log)
	defer os.RemoveAll(config.AllowedDirectories[0])

	factory, err := NewFileSystemResourceFactory(config)
	if err != nil {
		t.Fatalf("NewFileSystemResourceFactory failed: %v", err)
	}

	tests := []struct {
		name     string
		config   resources.ResourceConfig
		expected string
		hasError bool
	}{
		{
			name: "file_path key",
			config: resources.ResourceConfig{
				Config: map[string]interface{}{
					"file_path": "/test/file.txt",
				},
			},
			expected: "/test/file.txt",
			hasError: false,
		},
		{
			name: "path key",
			config: resources.ResourceConfig{
				Config: map[string]interface{}{
					"path": "/test/file.txt",
				},
			},
			expected: "/test/file.txt",
			hasError: false,
		},
		{
			name: "missing path",
			config: resources.ResourceConfig{
				Config: map[string]interface{}{},
			},
			hasError: true,
		},
		{
			name: "invalid path type",
			config: resources.ResourceConfig{
				Config: map[string]interface{}{
					"file_path": 123,
				},
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := factory.extractFilePath(tt.config)
			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for %s", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				}
				if path != tt.expected {
					t.Errorf("Expected path '%s', got '%s'", tt.expected, path)
				}
			}
		})
	}
}

func TestFileSystemResourceFactory_ConfigurationOverrides(t *testing.T) {
	log := createTestLogger(t)
	config := createTestFactoryConfig(log)
	defer os.RemoveAll(config.AllowedDirectories[0])

	factory, err := NewFileSystemResourceFactory(config)
	if err != nil {
		t.Fatalf("NewFileSystemResourceFactory failed: %v", err)
	}

	// Test configuration with overrides
	resourceConfig := resources.ResourceConfig{
		Enabled: true,
		Config: map[string]interface{}{
			"file_path":           "/test/file.txt",
			"max_file_size":       2048.0, // Test float conversion
			"allowed_extensions":  []string{".xml", ".yaml"},
			"blocked_patterns":    []string{"private"},
		},
	}

	validationConfig := factory.createValidationConfig(resourceConfig)

	if validationConfig.MaxFileSize != 2048 {
		t.Errorf("Expected max file size 2048, got %d", validationConfig.MaxFileSize)
	}

	expectedExts := []string{".xml", ".yaml"}
	if len(validationConfig.AllowedExtensions) != len(expectedExts) {
		t.Errorf("Expected %d extensions, got %d", len(expectedExts), len(validationConfig.AllowedExtensions))
	}

	expectedPatterns := []string{"private"}
	if len(validationConfig.BlockedPatterns) != len(expectedPatterns) {
		t.Errorf("Expected %d patterns, got %d", len(expectedPatterns), len(validationConfig.BlockedPatterns))
	}
}

func TestFileSystemResourceFactory_SupportsPath(t *testing.T) {
	log := createTestLogger(t)
	
	tmpDir, err := os.MkdirTemp("", "supports_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := FileSystemFactoryConfig{
		Logger:             log,
		AllowedDirectories: []string{tmpDir},
	}

	factory, err := NewFileSystemResourceFactory(config)
	if err != nil {
		t.Fatalf("NewFileSystemResourceFactory failed: %v", err)
	}

	// Test allowed path
	allowedPath := filepath.Join(tmpDir, "test.txt")
	if !factory.SupportsPath(allowedPath) {
		t.Error("Factory should support allowed path")
	}

	// Test disallowed path
	disallowedPath := "/etc/passwd"
	if factory.SupportsPath(disallowedPath) {
		t.Error("Factory should not support disallowed path")
	}

	// Test with base path
	factoryWithBase := &FileSystemResourceFactory{
		basePath: tmpDir,
		logger:   log,
	}

	if !factoryWithBase.SupportsPath(allowedPath) {
		t.Error("Factory with base path should support allowed path")
	}

	if factoryWithBase.SupportsPath(disallowedPath) {
		t.Error("Factory with base path should not support disallowed path")
	}
}

func TestFileSystemResourceFactory_Getters(t *testing.T) {
	log := createTestLogger(t)
	config := createTestFactoryConfig(log)
	defer os.RemoveAll(config.AllowedDirectories[0])

	factory, err := NewFileSystemResourceFactory(config)
	if err != nil {
		t.Fatalf("NewFileSystemResourceFactory failed: %v", err)
	}

	// Test getters
	if len(factory.GetAllowedDirectories()) != 1 {
		t.Errorf("Expected 1 allowed directory, got %d", len(factory.GetAllowedDirectories()))
	}

	if factory.GetMaxFileSize() != 1024 {
		t.Errorf("Expected max file size 1024, got %d", factory.GetMaxFileSize())
	}

	if len(factory.GetAllowedExtensions()) != 2 {
		t.Errorf("Expected 2 allowed extensions, got %d", len(factory.GetAllowedExtensions()))
	}

	if len(factory.GetBlockedPatterns()) != 1 {
		t.Errorf("Expected 1 blocked pattern, got %d", len(factory.GetBlockedPatterns()))
	}
}

func TestFileSystemResourceFactory_String(t *testing.T) {
	log := createTestLogger(t)
	config := createTestFactoryConfig(log)
	defer os.RemoveAll(config.AllowedDirectories[0])

	factory, err := NewFileSystemResourceFactory(config)
	if err != nil {
		t.Fatalf("NewFileSystemResourceFactory failed: %v", err)
	}

	str := factory.String()

	// Verify string contains expected components
	if !strings.Contains(str, "FileSystemResourceFactory") {
		t.Errorf("String should contain 'FileSystemResourceFactory', got: %s", str)
	}

	if !strings.Contains(str, factory.Name()) {
		t.Errorf("String should contain factory name, got: %s", str)
	}

	if !strings.Contains(str, factory.Version()) {
		t.Errorf("String should contain factory version, got: %s", str)
	}
}