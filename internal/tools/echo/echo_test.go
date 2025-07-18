package echo

import (
	"strings"
	"testing"
)

func TestNewEchoService(t *testing.T) {
	service := NewEchoService()
	if service == nil {
		t.Fatal("NewEchoService should return a valid service instance")
	}
}

func TestEchoService_Transform(t *testing.T) {
	service := NewEchoService()

	tests := []struct {
		name      string
		message   string
		prefix    string
		suffix    string
		uppercase bool
		expected  string
	}{
		{
			name:     "message only",
			message:  "hello world",
			prefix:   "",
			suffix:   "",
			uppercase: false,
			expected: "hello world",
		},
		{
			name:     "message with prefix",
			message:  "world",
			prefix:   "hello ",
			suffix:   "",
			uppercase: false,
			expected: "hello world",
		},
		{
			name:     "message with suffix",
			message:  "hello",
			prefix:   "",
			suffix:   " world",
			uppercase: false,
			expected: "hello world",
		},
		{
			name:     "message with prefix and suffix",
			message:  "beautiful",
			prefix:   "hello ",
			suffix:   " world",
			uppercase: false,
			expected: "hello beautiful world",
		},
		{
			name:     "message only with uppercase",
			message:  "hello world",
			prefix:   "",
			suffix:   "",
			uppercase: true,
			expected: "HELLO WORLD",
		},
		{
			name:     "message with prefix and suffix and uppercase",
			message:  "beautiful",
			prefix:   "hello ",
			suffix:   " world",
			uppercase: true,
			expected: "HELLO BEAUTIFUL WORLD",
		},
		{
			name:     "empty prefix and suffix",
			message:  "test",
			prefix:   "",
			suffix:   "",
			uppercase: false,
			expected: "test",
		},
		{
			name:     "special characters",
			message:  "test@#$%",
			prefix:   "[",
			suffix:   "]",
			uppercase: false,
			expected: "[test@#$%]",
		},
		{
			name:     "unicode characters",
			message:  "héllo wørld",
			prefix:   "→ ",
			suffix:   " ←",
			uppercase: false,
			expected: "→ héllo wørld ←",
		},
		{
			name:     "unicode with uppercase",
			message:  "héllo wørld",
			prefix:   "→ ",
			suffix:   " ←",
			uppercase: true,
			expected: "→ HÉLLO WØRLD ←",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.Transform(tt.message, tt.prefix, tt.suffix, tt.uppercase)
			if result != tt.expected {
				t.Errorf("Transform() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestEchoService_Validate(t *testing.T) {
	service := NewEchoService()

	tests := []struct {
		name        string
		message     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid short message",
			message:     "hello",
			expectError: false,
		},
		{
			name:        "valid single character",
			message:     "a",
			expectError: false,
		},
		{
			name:        "valid medium message",
			message:     strings.Repeat("a", 500),
			expectError: false,
		},
		{
			name:        "valid maximum length message",
			message:     strings.Repeat("a", 1000),
			expectError: false,
		},
		{
			name:        "empty message",
			message:     "",
			expectError: true,
			errorMsg:    "message cannot be empty",
		},
		{
			name:        "message too long",
			message:     strings.Repeat("a", 1001),
			expectError: true,
			errorMsg:    "message too long: 1001 characters (maximum 1000)",
		},
		{
			name:        "message much too long",
			message:     strings.Repeat("a", 2000),
			expectError: true,
			errorMsg:    "message too long: 2000 characters (maximum 1000)",
		},
		{
			name:        "valid message with special characters",
			message:     "hello@#$%^&*()world",
			expectError: false,
		},
		{
			name:        "valid message with unicode",
			message:     "héllo wørld 🌍",
			expectError: false,
		},
		{
			name:        "valid message with newlines",
			message:     "hello\nworld\ntest",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.Validate(tt.message)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Validate() expected error but got none")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("Validate() error = %q, expected %q", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestEchoService_ValidatePrefix(t *testing.T) {
	service := NewEchoService()

	tests := []struct {
		name        string
		prefix      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty prefix",
			prefix:      "",
			expectError: false,
		},
		{
			name:        "short prefix",
			prefix:      "hello ",
			expectError: false,
		},
		{
			name:        "maximum length prefix",
			prefix:      strings.Repeat("a", 100),
			expectError: false,
		},
		{
			name:        "prefix too long",
			prefix:      strings.Repeat("a", 101),
			expectError: true,
			errorMsg:    "prefix too long: 101 characters (maximum 100)",
		},
		{
			name:        "prefix much too long",
			prefix:      strings.Repeat("a", 200),
			expectError: true,
			errorMsg:    "prefix too long: 200 characters (maximum 100)",
		},
		{
			name:        "prefix with special characters",
			prefix:      "[DEBUG] ",
			expectError: false,
		},
		{
			name:        "prefix with unicode",
			prefix:      "→ ",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidatePrefix(tt.prefix)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidatePrefix() expected error but got none")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("ValidatePrefix() error = %q, expected %q", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePrefix() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestEchoService_ValidateSuffix(t *testing.T) {
	service := NewEchoService()

	tests := []struct {
		name        string
		suffix      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty suffix",
			suffix:      "",
			expectError: false,
		},
		{
			name:        "short suffix",
			suffix:      " [END]",
			expectError: false,
		},
		{
			name:        "maximum length suffix",
			suffix:      strings.Repeat("a", 100),
			expectError: false,
		},
		{
			name:        "suffix too long",
			suffix:      strings.Repeat("a", 101),
			expectError: true,
			errorMsg:    "suffix too long: 101 characters (maximum 100)",
		},
		{
			name:        "suffix much too long",
			suffix:      strings.Repeat("a", 200),
			expectError: true,
			errorMsg:    "suffix too long: 200 characters (maximum 100)",
		},
		{
			name:        "suffix with special characters",
			suffix:      " !!!",
			expectError: false,
		},
		{
			name:        "suffix with unicode",
			suffix:      " ←",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateSuffix(tt.suffix)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateSuffix() expected error but got none")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("ValidateSuffix() error = %q, expected %q", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateSuffix() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestEchoService_ValidateAll(t *testing.T) {
	service := NewEchoService()

	tests := []struct {
		name        string
		message     string
		prefix      string
		suffix      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "all valid parameters",
			message:     "hello world",
			prefix:      ">>> ",
			suffix:      " <<<",
			expectError: false,
		},
		{
			name:        "valid with empty prefix and suffix",
			message:     "hello world",
			prefix:      "",
			suffix:      "",
			expectError: false,
		},
		{
			name:        "maximum length parameters",
			message:     strings.Repeat("a", 1000),
			prefix:      strings.Repeat("b", 100),
			suffix:      strings.Repeat("c", 100),
			expectError: false,
		},
		{
			name:        "invalid message empty",
			message:     "",
			prefix:      "valid",
			suffix:      "valid",
			expectError: true,
			errorMsg:    "message cannot be empty",
		},
		{
			name:        "invalid message too long",
			message:     strings.Repeat("a", 1001),
			prefix:      "valid",
			suffix:      "valid",
			expectError: true,
			errorMsg:    "message too long: 1001 characters (maximum 1000)",
		},
		{
			name:        "invalid prefix too long",
			message:     "valid message",
			prefix:      strings.Repeat("a", 101),
			suffix:      "valid",
			expectError: true,
			errorMsg:    "prefix too long: 101 characters (maximum 100)",
		},
		{
			name:        "invalid suffix too long",
			message:     "valid message",
			prefix:      "valid",
			suffix:      strings.Repeat("a", 101),
			expectError: true,
			errorMsg:    "suffix too long: 101 characters (maximum 100)",
		},
		{
			name:        "multiple validation errors - message error reported first",
			message:     "",
			prefix:      strings.Repeat("a", 101),
			suffix:      strings.Repeat("b", 101),
			expectError: true,
			errorMsg:    "message cannot be empty",
		},
		{
			name:        "unicode characters in all fields",
			message:     "héllo wørld 🌍",
			prefix:      "→ ",
			suffix:      " ←",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateAll(tt.message, tt.prefix, tt.suffix)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateAll() expected error but got none")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("ValidateAll() error = %q, expected %q", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateAll() unexpected error: %v", err)
				}
			}
		})
	}
}