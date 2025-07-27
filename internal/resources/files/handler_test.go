package files

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
)

func createTestResource(t *testing.T, filePath, content string) (*FileSystemResource, *logger.Logger) {
	t.Helper()
	
	log := createTestLogger(t)
	
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	config := FileSystemResourceConfig{
		FilePath:    filePath,
		Name:        "test-resource",
		Description: "Test resource for handler",
		ValidationConfig: ValidationConfig{
			AllowedDirectories: []string{filepath.Dir(filePath)},
			MaxFileSize:        10000,
			AllowedExtensions:  []string{".txt", ".json", ".xml"},
			BlockedPatterns:    []string{},
		},
		Logger: log,
	}
	
	resource, err := NewFileSystemResource(config)
	if err != nil {
		t.Fatalf("Failed to create test resource: %v", err)
	}
	
	return resource, log
}

func TestNewFileSystemResourceHandler(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "handler_test.txt")
	resource, log := createTestResource(t, testFile, "test content")

	handler := NewFileSystemResourceHandler(resource, log)

	if handler == nil {
		t.Fatal("NewFileSystemResourceHandler returned nil")
	}

	if handler.GetResource() != resource {
		t.Error("Handler resource doesn't match expected resource")
	}

	// Verify interface compliance
	var _ mcp.ResourceHandler = handler
}

func TestFileSystemResourceHandler_Read_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		filename    string
		content     string
		expectText  bool
		expectedContent string
	}{
		{
			name:            "text file",
			filename:        "test.txt",
			content:         "Hello, World!",
			expectText:      true,
			expectedContent: "Hello, World!",
		},
		{
			name:            "JSON file",
			filename:        "data.json",
			content:         `{"key": "value"}`,
			expectText:      true,
			expectedContent: `{"key": "value"}`,
		},
		{
			name:            "XML file",
			filename:        "config.xml",
			content:         `<root><item>value</item></root>`,
			expectText:      true,
			expectedContent: `<root><item>value</item></root>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tt.filename)
			resource, _ := createTestResource(t, testFile, tt.content)
			handler := resource.Handler()

			ctx := context.Background()
			content, err := handler.Read(ctx, resource.URI())

			if err != nil {
				t.Fatalf("Read failed: %v", err)
			}

			if content == nil {
				t.Fatal("Read returned nil content")
			}

			// Verify interface compliance
			var _ mcp.ResourceContent = content

			// Check content
			contentItems := content.GetContent()
			if len(contentItems) != 1 {
				t.Fatalf("Expected 1 content item, got %d", len(contentItems))
			}

			contentItem := contentItems[0]
			if tt.expectText {
				if contentItem.Type() != "text" {
					t.Errorf("Expected text content, got %s", contentItem.Type())
				}
				if contentItem.GetText() != tt.expectedContent {
					t.Errorf("Expected content '%s', got '%s'", tt.expectedContent, contentItem.GetText())
				}
			} else {
				if contentItem.Type() != "blob" {
					t.Errorf("Expected blob content, got %s", contentItem.Type())
				}
			}

			// Check MIME type
			if content.GetMimeType() == "" {
				t.Error("MIME type should not be empty")
			}
		})
	}
}

func TestFileSystemResourceHandler_Read_BinaryContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file with binary content (not in allowed extensions, so will be treated as binary)
	testFile := filepath.Join(tmpDir, "binary.txt")
	binaryData := []byte{0x00, 0x01, 0xFF, 0xFE, 0x48, 0x65, 0x6C, 0x6C, 0x6F} // Binary data with "Hello"
	
	log := createTestLogger(t)
	if err := os.WriteFile(testFile, binaryData, 0644); err != nil {
		t.Fatalf("Failed to create binary test file: %v", err)
	}

	config := FileSystemResourceConfig{
		FilePath:    testFile,
		ValidationConfig: ValidationConfig{
			AllowedDirectories: []string{tmpDir},
			MaxFileSize:        10000,
			AllowedExtensions:  []string{".txt"},
		},
		Logger: log,
	}

	resource, err := NewFileSystemResource(config)
	if err != nil {
		t.Fatalf("Failed to create resource: %v", err)
	}

	handler := resource.Handler()
	ctx := context.Background()
	content, err := handler.Read(ctx, resource.URI())

	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	contentItems := content.GetContent()
	if len(contentItems) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(contentItems))
	}

	// Since it's a .txt file, it should be treated as text even with binary data
	contentItem := contentItems[0]
	if contentItem.Type() != "text" {
		t.Errorf("Expected text content type for .txt file, got %s", contentItem.Type())
	}
}

func TestFileSystemResourceHandler_Read_URIMismatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "mismatch.txt")
	resource, _ := createTestResource(t, testFile, "test content")
	handler := resource.Handler()

	ctx := context.Background()
	wrongURI := "file:///wrong/path/file.txt"
	_, err = handler.Read(ctx, wrongURI)

	if err == nil {
		t.Fatal("Expected error for URI mismatch")
	}

	if !strings.Contains(err.Error(), "URI mismatch") {
		t.Errorf("Expected URI mismatch error, got: %v", err)
	}
}

func TestFileSystemResourceHandler_Read_FileNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "will_be_deleted.txt")
	resource, _ := createTestResource(t, testFile, "test content")
	handler := resource.Handler()

	// Delete the file after creating the resource
	if err := os.Remove(testFile); err != nil {
		t.Fatalf("Failed to remove test file: %v", err)
	}

	ctx := context.Background()
	_, err = handler.Read(ctx, resource.URI())

	if err == nil {
		t.Fatal("Expected error for missing file")
	}

	if err != ErrFileNotFound {
		t.Errorf("Expected ErrFileNotFound, got: %v", err)
	}
}

func TestFileSystemResourceHandler_Read_PermissionDenied(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "no_permission.txt")
	resource, _ := createTestResource(t, testFile, "test content")
	handler := resource.Handler()

	// Remove read permissions
	if err := os.Chmod(testFile, 0000); err != nil {
		t.Fatalf("Failed to change file permissions: %v", err)
	}
	defer os.Chmod(testFile, 0644) // Restore permissions for cleanup

	ctx := context.Background()
	_, err = handler.Read(ctx, resource.URI())

	if err == nil {
		t.Fatal("Expected error for permission denied")
	}

	if err != ErrFilePermissionDenied {
		t.Errorf("Expected ErrFilePermissionDenied, got: %v", err)
	}
}

func TestFileSystemResourceHandler_Read_ContextCancellation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "context_test.txt")
	resource, _ := createTestResource(t, testFile, "test content")
	handler := resource.Handler()

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = handler.Read(ctx, resource.URI())

	if err == nil {
		t.Fatal("Expected error for cancelled context")
	}

	if !strings.Contains(err.Error(), "context cancelled") {
		t.Errorf("Expected context cancelled error, got: %v", err)
	}
}

func TestFileSystemResourceHandler_Read_ValidationFailure(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file with disallowed extension
	testFile := filepath.Join(tmpDir, "disallowed.exe")
	
	log := createTestLogger(t)
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := FileSystemResourceConfig{
		FilePath:    testFile,
		ValidationConfig: ValidationConfig{
			AllowedDirectories: []string{tmpDir},
			MaxFileSize:        10000,
			AllowedExtensions:  []string{".txt", ".json"}, // .exe not allowed
		},
		Logger: log,
	}

	resource, err := NewFileSystemResource(config)
	if err != nil {
		// This might fail at resource creation due to validation
		if err == ErrUnsupportedFileType {
			return // Expected behavior
		}
		t.Fatalf("Unexpected error creating resource: %v", err)
	}

	handler := resource.Handler()
	ctx := context.Background()
	_, err = handler.Read(ctx, resource.URI())

	if err == nil {
		t.Fatal("Expected validation error")
	}

	// Should return a permission error (security - don't expose validation details)
	if err != ErrFilePermissionDenied && err != ErrUnsupportedFileType {
		t.Errorf("Expected ErrFilePermissionDenied or ErrUnsupportedFileType, got: %v", err)
	}
}

func TestFileSystemResourceHandler_CanHandle(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "can_handle.txt")
	resource, _ := createTestResource(t, testFile, "test content")
	
	// Cast to access the specific handler type
	handler := resource.Handler().(*FileSystemResourceHandler)

	// Test with correct URI
	if !handler.CanHandle(resource.URI()) {
		t.Error("Handler should be able to handle its own resource URI")
	}

	// Test with incorrect URI
	wrongURI := "file:///wrong/path/file.txt"
	if handler.CanHandle(wrongURI) {
		t.Error("Handler should not handle wrong URI")
	}
}

func TestFileSystemResourceHandler_String(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "string_test.txt")
	resource, _ := createTestResource(t, testFile, "test content")
	
	// Cast to access the specific handler type
	handler := resource.Handler().(*FileSystemResourceHandler)

	str := handler.String()

	// Verify string contains expected components
	if !strings.Contains(str, "FileSystemResourceHandler") {
		t.Errorf("String should contain 'FileSystemResourceHandler', got: %s", str)
	}

	if !strings.Contains(str, resource.URI()) {
		t.Errorf("String should contain URI, got: %s", str)
	}

	if !strings.Contains(str, testFile) {
		t.Errorf("String should contain file path, got: %s", str)
	}
}

func TestFileSystemResourceHandler_IsTextContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	resource, log := createTestResource(t, testFile, "test content")
	handler := NewFileSystemResourceHandler(resource, log)

	tests := []struct {
		mimeType string
		expected bool
	}{
		{"text/plain", true},
		{"text/plain; charset=utf-8", true},
		{"application/json", true},
		{"application/xml", true},
		{"application/x-yaml", true},
		{"text/csv", true},
		{"text/html", true},
		{"application/javascript", true},
		{"text/css", true},
		{"application/octet-stream", false},
		{"image/png", false},
		{"video/mp4", false},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			result := handler.isTextContent(tt.mimeType)
			if result != tt.expected {
				t.Errorf("isTextContent(%s) = %v, expected %v", tt.mimeType, result, tt.expected)
			}
		})
	}
}

func TestFileSystemResourceHandler_ErrorConversion(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "handler_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "error_test.txt")
	resource, log := createTestResource(t, testFile, "test content")
	handler := NewFileSystemResourceHandler(resource, log)

	// Test validation error conversion
	validationErrors := map[error]error{
		ErrFileNotFound:         ErrFileNotFound,
		ErrFilePermissionDenied: ErrFilePermissionDenied,
		ErrDirectoryTraversal:   ErrFilePermissionDenied,
		ErrDirectoryNotAllowed:  ErrFilePermissionDenied,
		ErrFileTooBig:          ErrFileTooBig,
		ErrUnsupportedFileType: ErrUnsupportedFileType,
		ErrInvalidFilePath:     ErrFilePermissionDenied,
	}

	for inputErr, expectedErr := range validationErrors {
		result := handler.convertValidationError(inputErr)
		if result != expectedErr {
			t.Errorf("convertValidationError(%v) = %v, expected %v", inputErr, result, expectedErr)
		}
	}

	// Test file error conversion
	notExistErr := &os.PathError{Op: "open", Path: "missing", Err: os.ErrNotExist}
	result := handler.convertFileError(notExistErr)
	if result != ErrFileNotFound {
		t.Errorf("convertFileError(not exist) = %v, expected %v", result, ErrFileNotFound)
	}

	permissionErr := &os.PathError{Op: "open", Path: "noperm", Err: os.ErrPermission}
	result = handler.convertFileError(permissionErr)
	if result != ErrFilePermissionDenied {
		t.Errorf("convertFileError(permission) = %v, expected %v", result, ErrFilePermissionDenied)
	}
}