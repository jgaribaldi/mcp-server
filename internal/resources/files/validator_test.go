package files

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func createTestValidator(allowedDirs []string, maxSize int64, allowedExts []string) *FilePathValidator {
	cfg := ValidationConfig{
		AllowedDirectories: allowedDirs,
		MaxFileSize:        maxSize,
		AllowedExtensions:  allowedExts,
		BlockedPatterns:    []string{"secret", "private"},
	}
	return NewFilePathValidator(cfg, createTestLogger(&testing.T{}))
}

func TestNewFilePathValidator(t *testing.T) {
	cfg := ValidationConfig{
		AllowedDirectories: []string{"/tmp"},
		MaxFileSize:        1024,
		AllowedExtensions:  []string{".txt"},
		BlockedPatterns:    []string{"secret"},
	}
	log := createTestLogger(t)

	validator := NewFilePathValidator(cfg, log)

	if len(validator.allowedDirectories) != 1 {
		t.Errorf("Expected 1 allowed directory, got %d", len(validator.allowedDirectories))
	}

	if validator.maxFileSize != 1024 {
		t.Errorf("Expected max file size 1024, got %d", validator.maxFileSize)
	}

	if len(validator.allowedExtensions) != 1 {
		t.Errorf("Expected 1 allowed extension, got %d", len(validator.allowedExtensions))
	}
}

func TestCheckDirectoryTraversal(t *testing.T) {
	validator := createTestValidator(nil, 0, nil)

	tests := []struct {
		name     string
		path     string
		expected error
	}{
		{
			name:     "safe path",
			path:     "/tmp/test.txt",
			expected: nil,
		},
		{
			name:     "path with double dots",
			path:     "/tmp/../etc/passwd",
			expected: ErrDirectoryTraversal,
		},
		{
			name:     "path with forward slash traversal",
			path:     "/tmp/../secret/file.txt",
			expected: ErrDirectoryTraversal,
		},
		{
			name:     "path with backslash traversal",
			path:     "/tmp/..\\secret\\file.txt",
			expected: ErrDirectoryTraversal,
		},
		{
			name:     "path with hidden file reference",
			path:     "/tmp/./file.txt",
			expected: ErrDirectoryTraversal,
		},
		{
			name:     "clean path with dots in filename",
			path:     "/tmp/file.test.txt",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.CheckDirectoryTraversal(tt.path)
			if err != tt.expected {
				t.Errorf("Expected error %v, got %v", tt.expected, err)
			}
		})
	}
}

func TestCheckAllowedDirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "validator_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	validator := createTestValidator([]string{tmpDir}, 0, nil)

	tests := []struct {
		name     string
		path     string
		expected error
	}{
		{
			name:     "allowed directory",
			path:     filepath.Join(tmpDir, "test.txt"),
			expected: nil,
		},
		{
			name:     "disallowed directory",
			path:     "/etc/passwd",
			expected: ErrDirectoryNotAllowed,
		},
		{
			name:     "subdirectory of allowed",
			path:     filepath.Join(tmpDir, "subdir", "test.txt"),
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absPath, err := filepath.Abs(tt.path)
			if err != nil {
				t.Fatalf("Failed to get absolute path: %v", err)
			}
			err = validator.CheckAllowedDirectories(absPath)
			if err != tt.expected {
				t.Errorf("Expected error %v, got %v", tt.expected, err)
			}
		})
	}
}

func TestCheckBlockedPatterns(t *testing.T) {
	validator := createTestValidator(nil, 0, nil)

	tests := []struct {
		name     string
		path     string
		expected error
	}{
		{
			name:     "safe path",
			path:     "/tmp/test.txt",
			expected: nil,
		},
		{
			name:     "path with secret pattern",
			path:     "/tmp/secret.txt",
			expected: ErrInvalidFilePath,
		},
		{
			name:     "path with private pattern",
			path:     "/tmp/private.txt",
			expected: ErrInvalidFilePath,
		},
		{
			name:     "case insensitive pattern match",
			path:     "/tmp/SECRET.txt",
			expected: ErrInvalidFilePath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.CheckBlockedPatterns(tt.path)
			if err != tt.expected {
				t.Errorf("Expected error %v, got %v", tt.expected, err)
			}
		})
	}
}

func TestCheckFileExtension(t *testing.T) {
	validator := createTestValidator(nil, 0, []string{".txt", ".json"})

	tests := []struct {
		name     string
		path     string
		expected error
	}{
		{
			name:     "allowed txt extension",
			path:     "/tmp/test.txt",
			expected: nil,
		},
		{
			name:     "allowed json extension",
			path:     "/tmp/data.json",
			expected: nil,
		},
		{
			name:     "disallowed extension",
			path:     "/tmp/script.exe",
			expected: ErrUnsupportedFileType,
		},
		{
			name:     "no extension",
			path:     "/tmp/test",
			expected: ErrUnsupportedFileType,
		},
		{
			name:     "case insensitive extension",
			path:     "/tmp/test.TXT",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.CheckFileExtension(tt.path)
			if err != tt.expected {
				t.Errorf("Expected error %v, got %v", tt.expected, err)
			}
		})
	}
}

func TestCheckFileSize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "validator_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	smallFile := filepath.Join(tmpDir, "small.txt")
	if err := os.WriteFile(smallFile, []byte("small"), 0644); err != nil {
		t.Fatalf("Failed to create small file: %v", err)
	}

	largeFile := filepath.Join(tmpDir, "large.txt")
	largeData := strings.Repeat("x", 2000)
	if err := os.WriteFile(largeFile, []byte(largeData), 0644); err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	validator := createTestValidator(nil, 1000, nil)

	tests := []struct {
		name     string
		path     string
		expected error
	}{
		{
			name:     "file within size limit",
			path:     smallFile,
			expected: nil,
		},
		{
			name:     "file exceeds size limit",
			path:     largeFile,
			expected: ErrFileTooBig,
		},
		{
			name:     "non-existent file",
			path:     filepath.Join(tmpDir, "missing.txt"),
			expected: ErrFileNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.CheckFileSize(tt.path)
			if err != tt.expected {
				t.Errorf("Expected error %v, got %v", tt.expected, err)
			}
		})
	}
}

func TestDetectMimeType(t *testing.T) {
	validator := createTestValidator(nil, 0, nil)

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "text file",
			path:     "/tmp/test.txt",
			expected: "text/plain; charset=utf-8",
		},
		{
			name:     "json file",
			path:     "/tmp/data.json",
			expected: "application/json",
		},
		{
			name:     "xml file",
			path:     "/tmp/config.xml",
			expected: "application/xml",
		},
		{
			name:     "yaml file",
			path:     "/tmp/config.yaml",
			expected: "application/x-yaml",
		},
		{
			name:     "yml file",
			path:     "/tmp/config.yml",
			expected: "application/x-yaml",
		},
		{
			name:     "csv file",
			path:     "/tmp/data.csv",
			expected: "text/csv; charset=utf-8",
		},
		{
			name:     "html file",
			path:     "/tmp/page.html",
			expected: "text/html; charset=utf-8",
		},
		{
			name:     "unknown extension",
			path:     "/tmp/binary.unknown",
			expected: "application/octet-stream",
		},
		{
			name:     "case insensitive",
			path:     "/tmp/test.TXT",
			expected: "text/plain; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mimeType := validator.DetectMimeType(tt.path)
			if mimeType != tt.expected {
				t.Errorf("Expected MIME type %s, got %s", tt.expected, mimeType)
			}
		})
	}
}

func TestGetFileInfo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "validator_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "test content"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	validator := createTestValidator([]string{tmpDir}, 1000, []string{".txt"})

	info, err := validator.GetFileInfo(testFile)
	if err != nil {
		t.Fatalf("GetFileInfo failed: %v", err)
	}

	if info.Path != testFile {
		t.Errorf("Expected path %s, got %s", testFile, info.Path)
	}

	if info.Size != int64(len(testContent)) {
		t.Errorf("Expected size %d, got %d", len(testContent), info.Size)
	}

	if info.MimeType != "text/plain; charset=utf-8" {
		t.Errorf("Expected MIME type text/plain; charset=utf-8, got %s", info.MimeType)
	}

	if time.Since(info.ModTime) > time.Minute {
		t.Errorf("ModTime seems incorrect: %v", info.ModTime)
	}
}

func TestValidateFile_Integration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "validator_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create valid test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	validator := createTestValidator([]string{tmpDir}, 1000, []string{".txt"})

	// Should pass all validations
	if err := validator.ValidateFile(testFile); err != nil {
		t.Errorf("Valid file failed validation: %v", err)
	}

	// Test with traversal attack - should be caught by directory traversal check
	traversalPath := "/tmp/../etc/passwd"
	if err := validator.ValidateFile(traversalPath); err != ErrDirectoryTraversal {
		t.Errorf("Expected directory traversal error, got %v", err)
	}
}

func BenchmarkValidatePath(b *testing.B) {
	validator := createTestValidator([]string{"/tmp"}, 1000, []string{".txt"})
	path := "/tmp/test.txt"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidatePath(path)
	}
}

func BenchmarkDetectMimeType(b *testing.B) {
	validator := createTestValidator(nil, 0, nil)
	path := "/tmp/test.json"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.DetectMimeType(path)
	}
}