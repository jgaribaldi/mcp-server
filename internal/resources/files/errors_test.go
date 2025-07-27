package files

import (
	"testing"
)

func TestFileErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrFileNotFound",
			err:      ErrFileNotFound,
			expected: "file not found",
		},
		{
			name:     "ErrFilePermissionDenied",
			err:      ErrFilePermissionDenied,
			expected: "file permission denied",
		},
		{
			name:     "ErrInvalidFilePath",
			err:      ErrInvalidFilePath,
			expected: "invalid file path",
		},
		{
			name:     "ErrDirectoryTraversal",
			err:      ErrDirectoryTraversal,
			expected: "directory traversal attempt",
		},
		{
			name:     "ErrFileTooBig",
			err:      ErrFileTooBig,
			expected: "file size exceeds limit",
		},
		{
			name:     "ErrUnsupportedFileType",
			err:      ErrUnsupportedFileType,
			expected: "unsupported file type",
		},
		{
			name:     "ErrDirectoryNotAllowed",
			err:      ErrDirectoryNotAllowed,
			expected: "directory not in allowed list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("Expected error message '%s', got '%s'", tt.expected, tt.err.Error())
			}
		})
	}
}

func TestErrorInterface(t *testing.T) {
	errors := []error{
		ErrFileNotFound,
		ErrFilePermissionDenied,
		ErrInvalidFilePath,
		ErrDirectoryTraversal,
		ErrFileTooBig,
		ErrUnsupportedFileType,
		ErrDirectoryNotAllowed,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("Error should not be nil")
		}
		
		if err.Error() == "" {
			t.Error("Error message should not be empty")
		}
	}
}

func TestErrorUniqueness(t *testing.T) {
	errors := map[string]error{
		"file not found":              ErrFileNotFound,
		"file permission denied":      ErrFilePermissionDenied,
		"invalid file path":           ErrInvalidFilePath,
		"directory traversal attempt": ErrDirectoryTraversal,
		"file size exceeds limit":     ErrFileTooBig,
		"unsupported file type":       ErrUnsupportedFileType,
		"directory not in allowed list": ErrDirectoryNotAllowed,
	}

	messages := make(map[string]bool)
	for message := range errors {
		if messages[message] {
			t.Errorf("Duplicate error message found: %s", message)
		}
		messages[message] = true
	}
}