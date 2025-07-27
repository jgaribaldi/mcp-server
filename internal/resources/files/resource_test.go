package files

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
)

func createTestFile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", filePath, err)
	}
	return filePath
}

func createTestResourceConfig(filePath string, log *logger.Logger) FileSystemResourceConfig {
	return FileSystemResourceConfig{
		FilePath:    filePath,
		Name:        "test-resource",
		Description: "Test file resource",
		ValidationConfig: ValidationConfig{
			AllowedDirectories: []string{filepath.Dir(filePath)},
			MaxFileSize:        10000,
			AllowedExtensions:  []string{".txt", ".json"},
			BlockedPatterns:    []string{},
		},
		Logger: log,
	}
}

func TestNewFileSystemResource(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "resource_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log := createTestLogger(t)
	testFile := createTestFile(t, tmpDir, "test.txt", "test content")

	config := createTestResourceConfig(testFile, log)
	resource, err := NewFileSystemResource(config)

	if err != nil {
		t.Fatalf("NewFileSystemResource failed: %v", err)
	}

	if resource == nil {
		t.Fatal("NewFileSystemResource returned nil")
	}

	// Verify interface compliance
	var _ mcp.Resource = resource

	// Test URI generation
	if !strings.HasPrefix(resource.URI(), "file://") {
		t.Errorf("Expected URI to start with file://, got %s", resource.URI())
	}

	if !strings.Contains(resource.URI(), "test.txt") {
		t.Errorf("Expected URI to contain filename, got %s", resource.URI())
	}
}

func TestFileSystemResource_InterfaceMethods(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "resource_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log := createTestLogger(t)
	testFile := createTestFile(t, tmpDir, "interface_test.txt", "interface test content")

	config := createTestResourceConfig(testFile, log)
	config.Name = "custom-name"
	config.Description = "custom description"

	resource, err := NewFileSystemResource(config)
	if err != nil {
		t.Fatalf("NewFileSystemResource failed: %v", err)
	}

	// Test Name method
	if resource.Name() != "custom-name" {
		t.Errorf("Expected name 'custom-name', got '%s'", resource.Name())
	}

	// Test Description method
	if resource.Description() != "custom description" {
		t.Errorf("Expected description 'custom description', got '%s'", resource.Description())
	}

	// Test MimeType method
	expectedMimeType := "text/plain; charset=utf-8"
	if resource.MimeType() != expectedMimeType {
		t.Errorf("Expected MIME type '%s', got '%s'", expectedMimeType, resource.MimeType())
	}

	// Test Handler method
	handler := resource.Handler()
	if handler == nil {
		t.Error("Handler() returned nil")
	}

	// Verify handler type
	if _, ok := handler.(*FileSystemResourceHandler); !ok {
		t.Errorf("Expected FileSystemResourceHandler, got %T", handler)
	}
}

func TestFileSystemResource_DefaultNameAndDescription(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "resource_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log := createTestLogger(t)
	testFile := createTestFile(t, tmpDir, "defaults.txt", "test content")

	config := createTestResourceConfig(testFile, log)
	// Leave name and description empty to test defaults
	config.Name = ""
	config.Description = ""

	resource, err := NewFileSystemResource(config)
	if err != nil {
		t.Fatalf("NewFileSystemResource failed: %v", err)
	}

	// Test default name (should be filename)
	if resource.Name() != "defaults.txt" {
		t.Errorf("Expected default name 'defaults.txt', got '%s'", resource.Name())
	}

	// Test default description
	expectedDesc := "File resource: defaults.txt"
	if resource.Description() != expectedDesc {
		t.Errorf("Expected default description '%s', got '%s'", expectedDesc, resource.Description())
	}
}

func TestFileSystemResource_Metadata(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "resource_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log := createTestLogger(t)
	content := "metadata test content"
	testFile := createTestFile(t, tmpDir, "metadata.txt", content)

	config := createTestResourceConfig(testFile, log)
	resource, err := NewFileSystemResource(config)
	if err != nil {
		t.Fatalf("NewFileSystemResource failed: %v", err)
	}

	// Test size
	if resource.GetSize() != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), resource.GetSize())
	}

	// Test file path
	if resource.GetFilePath() != testFile {
		t.Errorf("Expected file path %s, got %s", testFile, resource.GetFilePath())
	}

	// Test modification time (should be recent)
	if time.Since(resource.GetModTime()) > time.Minute {
		t.Errorf("ModTime seems incorrect: %v", resource.GetModTime())
	}

	// Test permissions
	expectedMode := os.FileMode(0644)
	if resource.GetPermissions() != expectedMode {
		t.Errorf("Expected permissions %v, got %v", expectedMode, resource.GetPermissions())
	}
}

func TestFileSystemResource_RefreshMetadata(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "resource_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log := createTestLogger(t)
	testFile := createTestFile(t, tmpDir, "refresh.txt", "initial content")

	config := createTestResourceConfig(testFile, log)
	resource, err := NewFileSystemResource(config)
	if err != nil {
		t.Fatalf("NewFileSystemResource failed: %v", err)
	}

	initialSize := resource.GetSize()
	initialModTime := resource.GetModTime()

	// Wait a moment and modify the file
	time.Sleep(10 * time.Millisecond)
	newContent := "modified content with more text"
	if err := os.WriteFile(testFile, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Refresh metadata
	if err := resource.RefreshMetadata(); err != nil {
		t.Fatalf("RefreshMetadata failed: %v", err)
	}

	// Verify metadata was updated
	if resource.GetSize() == initialSize {
		t.Error("Size was not updated after refresh")
	}

	if resource.GetSize() != int64(len(newContent)) {
		t.Errorf("Expected size %d after refresh, got %d", len(newContent), resource.GetSize())
	}

	if !resource.GetModTime().After(initialModTime) {
		t.Error("ModTime was not updated after refresh")
	}
}

func TestFileSystemResource_Validate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "resource_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log := createTestLogger(t)
	testFile := createTestFile(t, tmpDir, "validate.txt", "validation test")

	config := createTestResourceConfig(testFile, log)
	resource, err := NewFileSystemResource(config)
	if err != nil {
		t.Fatalf("NewFileSystemResource failed: %v", err)
	}

	// Test validation of valid file
	if err := resource.Validate(); err != nil {
		t.Errorf("Validation failed for valid file: %v", err)
	}

	// Test IsAccessible for valid file
	if !resource.IsAccessible() {
		t.Error("IsAccessible returned false for valid file")
	}

	// Remove the file and test validation failure
	if err := os.Remove(testFile); err != nil {
		t.Fatalf("Failed to remove test file: %v", err)
	}

	if err := resource.Validate(); err == nil {
		t.Error("Expected validation to fail for missing file")
	}

	if resource.IsAccessible() {
		t.Error("IsAccessible returned true for missing file")
	}
}

func TestFileSystemResource_ErrorHandling(t *testing.T) {
	log := createTestLogger(t)

	// Test with nil logger
	config := FileSystemResourceConfig{
		FilePath: "/tmp/test.txt",
		Logger:   nil,
	}
	_, err := NewFileSystemResource(config)
	if err == nil {
		t.Error("Expected error for nil logger")
	}

	// Test with non-existent file
	config = FileSystemResourceConfig{
		FilePath: "/nonexistent/path/file.txt",
		ValidationConfig: ValidationConfig{
			AllowedDirectories: []string{"/nonexistent"},
			AllowedExtensions:  []string{".txt"},
		},
		Logger: log,
	}
	_, err = NewFileSystemResource(config)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test with invalid URI path
	tmpDir, err := os.MkdirTemp("", "resource_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := createTestFile(t, tmpDir, "uri_test.txt", "content")
	config = createTestResourceConfig(testFile, log)
	// Create resource successfully first
	resource, err := NewFileSystemResource(config)
	if err != nil {
		t.Fatalf("NewFileSystemResource failed: %v", err)
	}

	// Test refresh with missing file
	os.Remove(testFile)
	if err := resource.RefreshMetadata(); err == nil {
		t.Error("Expected RefreshMetadata to fail for missing file")
	}
}

func TestGenerateFileURI(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "absolute path",
			path:     "/tmp/test.txt",
			expected: "file:///tmp/test.txt",
		},
		{
			name:     "relative path",
			path:     "test.txt",
			expected: "file://", // Will contain absolute path
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri, err := generateFileURI(tt.path)
			if err != nil {
				t.Fatalf("generateFileURI failed: %v", err)
			}

			if !strings.HasPrefix(uri, "file://") {
				t.Errorf("Expected URI to start with file://, got %s", uri)
			}

			if tt.name == "absolute path" && uri != tt.expected {
				t.Errorf("Expected URI %s, got %s", tt.expected, uri)
			}

			if tt.name == "relative path" {
				// For relative paths, just verify it contains the filename
				if !strings.Contains(uri, "test.txt") {
					t.Errorf("Expected URI to contain filename, got %s", uri)
				}
			}
		})
	}
}

func TestFileSystemResource_String(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "resource_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log := createTestLogger(t)
	testFile := createTestFile(t, tmpDir, "string_test.txt", "string test content")

	config := createTestResourceConfig(testFile, log)
	resource, err := NewFileSystemResource(config)
	if err != nil {
		t.Fatalf("NewFileSystemResource failed: %v", err)
	}

	str := resource.String()

	// Verify string contains expected components
	if !strings.Contains(str, "FileSystemResource") {
		t.Errorf("String should contain 'FileSystemResource', got: %s", str)
	}

	if !strings.Contains(str, resource.URI()) {
		t.Errorf("String should contain URI, got: %s", str)
	}

	if !strings.Contains(str, testFile) {
		t.Errorf("String should contain file path, got: %s", str)
	}

	if !strings.Contains(str, resource.MimeType()) {
		t.Errorf("String should contain MIME type, got: %s", str)
	}
}

func TestFileSystemResourceHandler_Integration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "resource_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log := createTestLogger(t)
	testContent := "integration test content"
	testFile := createTestFile(t, tmpDir, "integration_test.txt", testContent)

	config := createTestResourceConfig(testFile, log)
	resource, err := NewFileSystemResource(config)
	if err != nil {
		t.Fatalf("NewFileSystemResource failed: %v", err)
	}

	handler := resource.Handler()
	if handler == nil {
		t.Fatal("Handler is nil")
	}

	// Test successful read
	ctx := context.Background()
	content, err := handler.Read(ctx, resource.URI())
	if err != nil {
		t.Fatalf("Handler Read failed: %v", err)
	}

	if content == nil {
		t.Fatal("Handler returned nil content")
	}

	// Verify content matches
	contentItems := content.GetContent()
	if len(contentItems) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(contentItems))
	}

	if contentItems[0].GetText() != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, contentItems[0].GetText())
	}
}

