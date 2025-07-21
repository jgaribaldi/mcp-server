package resources

import (
	"strings"
	"testing"
	"time"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
)

func createTestValidator() *ResourceValidator {
	cfg := &config.Config{}
	log, _ := logger.NewDefault()
	return NewResourceValidator(cfg, log)
}

func TestResourceValidator_ValidateURI(t *testing.T) {
	validator := createTestValidator()

	tests := []struct {
		name        string
		uri         string
		expectError bool
		errorSubstr string
	}{
		{
			name:        "valid file URI",
			uri:         "file:///path/to/file.txt",
			expectError: false,
		},
		{
			name:        "valid config URI",
			uri:         "config://database/connection",
			expectError: false,
		},
		{
			name:        "valid api URI",
			uri:         "api://service.example.com/data",
			expectError: false,
		},
		{
			name:        "valid custom URI",
			uri:         "custom://internal/metrics",
			expectError: false,
		},
		{
			name:        "valid http URI",
			uri:         "http://example.com/resource",
			expectError: false,
		},
		{
			name:        "valid https URI",
			uri:         "https://example.com/resource",
			expectError: false,
		},
		{
			name:        "empty URI",
			uri:         "",
			expectError: true,
			errorSubstr: "URI cannot be empty",
		},
		{
			name:        "invalid URI format",
			uri:         "not-a-valid-uri",
			expectError: true,
			errorSubstr: "URI must have a scheme",
		},
		{
			name:        "unsupported scheme",
			uri:         "ftp://example.com/file",
			expectError: true,
			errorSubstr: "unsupported URI scheme",
		},
		{
			name:        "file URI without path",
			uri:         "file://",
			expectError: true,
			errorSubstr: "file URI must have a path",
		},
		{
			name:        "file URI with path traversal",
			uri:         "file:///path/../etc/passwd",
			expectError: true,
			errorSubstr: "path traversal not allowed",
		},
		{
			name:        "config URI without host or path",
			uri:         "config://",
			expectError: true,
			errorSubstr: "config URI must have host or path",
		},
		{
			name:        "api URI without host",
			uri:         "api:///path",
			expectError: true,
			errorSubstr: "api URI must have a host",
		},
		{
			name:        "URI too long",
			uri:         "file:///" + strings.Repeat("a", 2050),
			expectError: true,
			errorSubstr: "URI too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateURI(tt.uri)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for URI '%s', got nil", tt.uri)
				} else if !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorSubstr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for URI '%s', got: %v", tt.uri, err)
				}
			}
		})
	}
}

func TestResourceValidator_ValidateName(t *testing.T) {
	validator := createTestValidator()

	tests := []struct {
		name        string
		resourceName string
		expectError bool
		errorSubstr string
	}{
		{
			name:         "valid name",
			resourceName: "Valid Resource Name",
			expectError:  false,
		},
		{
			name:         "name with hyphens and underscores",
			resourceName: "Valid-Resource_Name",
			expectError:  false,
		},
		{
			name:         "alphanumeric name",
			resourceName: "Resource123",
			expectError:  false,
		},
		{
			name:        "empty name",
			resourceName: "",
			expectError: true,
			errorSubstr: "name cannot be empty",
		},
		{
			name:        "name too long",
			resourceName: strings.Repeat("a", 256),
			expectError: true,
			errorSubstr: "name too long",
		},
		{
			name:        "name with invalid characters",
			resourceName: "Invalid@Name!",
			expectError: true,
			errorSubstr: "name contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateName(tt.resourceName)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for name '%s', got nil", tt.resourceName)
				} else if !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorSubstr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for name '%s', got: %v", tt.resourceName, err)
				}
			}
		})
	}
}

func TestResourceValidator_ValidateMimeType(t *testing.T) {
	validator := createTestValidator()

	tests := []struct {
		name        string
		mimeType    string
		expectError bool
		errorSubstr string
	}{
		{
			name:        "valid text mime type",
			mimeType:    "text/plain",
			expectError: false,
		},
		{
			name:        "valid application mime type",
			mimeType:    "application/json",
			expectError: false,
		},
		{
			name:        "valid image mime type",
			mimeType:    "image/png",
			expectError: false,
		},
		{
			name:        "empty mime type",
			mimeType:    "",
			expectError: true,
			errorSubstr: "MIME type cannot be empty",
		},
		{
			name:        "invalid format - no slash",
			mimeType:    "textplain",
			expectError: true,
			errorSubstr: "invalid MIME type format",
		},
		{
			name:        "invalid format - multiple slashes",
			mimeType:    "text/plain/extra",
			expectError: true,
			errorSubstr: "invalid MIME type format",
		},
		{
			name:        "empty main type",
			mimeType:    "/plain",
			expectError: true,
			errorSubstr: "MIME type parts cannot be empty",
		},
		{
			name:        "empty sub type",
			mimeType:    "text/",
			expectError: true,
			errorSubstr: "MIME type parts cannot be empty",
		},
		{
			name:        "unsupported main type",
			mimeType:    "unknown/subtype",
			expectError: true,
			errorSubstr: "unsupported MIME type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateMimeType(tt.mimeType)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for MIME type '%s', got nil", tt.mimeType)
				} else if !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorSubstr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for MIME type '%s', got: %v", tt.mimeType, err)
				}
			}
		})
	}
}

func TestResourceValidator_ValidateFactory(t *testing.T) {
	validator := createTestValidator()

	// Valid factory
	validFactory := &mockResourceFactory{
		uri:          "file:///test/resource.txt",
		name:         "Test Resource",
		description:  "A test resource",
		mimeType:     "text/plain",
		version:      "1.0.0",
		tags:         []string{"test"},
		capabilities: []string{"read"},
	}

	err := validator.ValidateFactory(validFactory)
	if err != nil {
		t.Errorf("Expected no error for valid factory, got: %v", err)
	}

	// Factory with invalid URI
	invalidURIFactory := &mockResourceFactory{
		uri:          "invalid-uri",
		name:         "Test Resource",
		description:  "A test resource",
		mimeType:     "text/plain",
		version:      "1.0.0",
		tags:         []string{"test"},
		capabilities: []string{"read"},
	}

	err = validator.ValidateFactory(invalidURIFactory)
	if err == nil {
		t.Error("Expected error for factory with invalid URI, got nil")
	}

	// Factory with empty name
	emptyNameFactory := &mockResourceFactory{
		uri:          "file:///test/resource.txt",
		name:         "",
		description:  "A test resource",
		mimeType:     "text/plain",
		version:      "1.0.0",
		tags:         []string{"test"},
		capabilities: []string{"read"},
	}

	err = validator.ValidateFactory(emptyNameFactory)
	if err == nil {
		t.Error("Expected error for factory with empty name, got nil")
	}

	// Factory with empty description
	emptyDescFactory := &mockResourceFactory{
		uri:          "file:///test/resource.txt",
		name:         "Test Resource",
		description:  "",
		mimeType:     "text/plain",
		version:      "1.0.0",
		tags:         []string{"test"},
		capabilities: []string{"read"},
	}

	err = validator.ValidateFactory(emptyDescFactory)
	if err == nil {
		t.Error("Expected error for factory with empty description, got nil")
	}

	// Factory with no capabilities
	noCapabilitiesFactory := &mockResourceFactory{
		uri:          "file:///test/resource.txt",
		name:         "Test Resource",
		description:  "A test resource",
		mimeType:     "text/plain",
		version:      "1.0.0",
		tags:         []string{"test"},
		capabilities: []string{},
	}

	err = validator.ValidateFactory(noCapabilitiesFactory)
	if err == nil {
		t.Error("Expected error for factory with no capabilities, got nil")
	}
}

func TestResourceValidator_ValidateResource(t *testing.T) {
	validator := createTestValidator()

	// Valid resource
	validResource := &mockResource{
		uri:         "file:///test/resource.txt",
		name:        "Test Resource",
		description: "A test resource",
		mimeType:    "text/plain",
		handler:     &mockResourceHandler{},
	}

	err := validator.ValidateResource(validResource)
	if err != nil {
		t.Errorf("Expected no error for valid resource, got: %v", err)
	}

	// Resource with invalid URI
	invalidURIResource := &mockResource{
		uri:         "invalid-uri",
		name:        "Test Resource",
		description: "A test resource",
		mimeType:    "text/plain",
		handler:     &mockResourceHandler{},
	}

	err = validator.ValidateResource(invalidURIResource)
	if err == nil {
		t.Error("Expected error for resource with invalid URI, got nil")
	}

	// Resource with nil handler
	nilHandlerResource := &mockResource{
		uri:         "file:///test/resource.txt",
		name:        "Test Resource",
		description: "A test resource",
		mimeType:    "text/plain",
		handler:     nil,
	}

	err = validator.ValidateResource(nilHandlerResource)
	if err == nil {
		t.Error("Expected error for resource with nil handler, got nil")
	}
}

func TestResourceValidator_ValidateConfig(t *testing.T) {
	validator := createTestValidator()

	// Valid config
	validConfig := ResourceConfig{
		Enabled:       true,
		Config:        map[string]interface{}{"key": "value"},
		CacheTimeout:  300,
		AccessControl: map[string]string{"role": "admin"},
	}

	err := validator.ValidateConfig(validConfig)
	if err != nil {
		t.Errorf("Expected no error for valid config, got: %v", err)
	}

	// Config with negative cache timeout
	negativeCacheConfig := ResourceConfig{
		Enabled:       true,
		Config:        map[string]interface{}{"key": "value"},
		CacheTimeout:  -1,
		AccessControl: map[string]string{"role": "admin"},
	}

	err = validator.ValidateConfig(negativeCacheConfig)
	if err == nil {
		t.Error("Expected error for config with negative cache timeout, got nil")
	}

	// Config with too large cache timeout
	largeCacheConfig := ResourceConfig{
		Enabled:       true,
		Config:        map[string]interface{}{"key": "value"},
		CacheTimeout:  100000,
		AccessControl: map[string]string{"role": "admin"},
	}

	err = validator.ValidateConfig(largeCacheConfig)
	if err == nil {
		t.Error("Expected error for config with too large cache timeout, got nil")
	}

	// Config with empty access control key
	emptyACKeyConfig := ResourceConfig{
		Enabled:       true,
		Config:        map[string]interface{}{"key": "value"},
		CacheTimeout:  300,
		AccessControl: map[string]string{"": "admin"},
	}

	err = validator.ValidateConfig(emptyACKeyConfig)
	if err == nil {
		t.Error("Expected error for config with empty access control key, got nil")
	}

	// Config with nil config value
	nilValueConfig := ResourceConfig{
		Enabled:       true,
		Config:        map[string]interface{}{"key": nil},
		CacheTimeout:  300,
		AccessControl: map[string]string{"role": "admin"},
	}

	err = validator.ValidateConfig(nilValueConfig)
	if err == nil {
		t.Error("Expected error for config with nil value, got nil")
	}
}

func TestResourceValidator_ValidateCacheExpiration(t *testing.T) {
	validator := createTestValidator()

	now := time.Now()

	// Valid (not expired) cache
	validCache := CachedContent{
		Content:     nil, // Not relevant for this test
		Timestamp:   now,
		ExpiresAt:   now.Add(5 * time.Minute),
		AccessCount: 1,
	}

	err := validator.ValidateCacheExpiration(validCache)
	if err != nil {
		t.Errorf("Expected no error for valid cache, got: %v", err)
	}

	// Expired cache
	expiredCache := CachedContent{
		Content:     nil, // Not relevant for this test
		Timestamp:   now.Add(-10 * time.Minute),
		ExpiresAt:   now.Add(-5 * time.Minute),
		AccessCount: 1,
	}

	err = validator.ValidateCacheExpiration(expiredCache)
	if err == nil {
		t.Error("Expected error for expired cache, got nil")
	}

	if !strings.Contains(err.Error(), "cached content expired") {
		t.Errorf("Expected 'cached content expired' error, got: %v", err)
	}
}

func TestResourceValidator_ValidateResourceContent(t *testing.T) {
	validator := createTestValidator()

	// Valid text content
	textContent := &mockContent{
		contentType: "text",
		text:        "Test content",
	}
	validTextResourceContent := &mockResourceContent{
		content:  []mcp.Content{textContent},
		mimeType: "text/plain",
	}

	err := validator.ValidateResourceContent(validTextResourceContent)
	if err != nil {
		t.Errorf("Expected no error for valid text content, got: %v", err)
	}

	// Valid blob content
	blobContent := &mockContent{
		contentType: "blob",
		blob:        []byte("binary data"),
	}
	validBlobResourceContent := &mockResourceContent{
		content:  []mcp.Content{blobContent},
		mimeType: "application/octet-stream",
	}

	err = validator.ValidateResourceContent(validBlobResourceContent)
	if err != nil {
		t.Errorf("Expected no error for valid blob content, got: %v", err)
	}

	// Content with invalid MIME type
	invalidMimeContent := &mockResourceContent{
		content:  []mcp.Content{textContent},
		mimeType: "invalid-mime",
	}

	err = validator.ValidateResourceContent(invalidMimeContent)
	if err == nil {
		t.Error("Expected error for content with invalid MIME type, got nil")
	}

	// Empty content
	emptyContent := &mockResourceContent{
		content:  []mcp.Content{},
		mimeType: "text/plain",
	}

	err = validator.ValidateResourceContent(emptyContent)
	if err == nil {
		t.Error("Expected error for empty content, got nil")
	}

	// Content with nil item
	nilItemContent := &mockResourceContent{
		content:  []mcp.Content{nil},
		mimeType: "text/plain",
	}

	err = validator.ValidateResourceContent(nilItemContent)
	if err == nil {
		t.Error("Expected error for content with nil item, got nil")
	}

	// Content with empty text
	emptyTextContent := &mockContent{
		contentType: "text",
		text:        "",
	}
	emptyTextResourceContent := &mockResourceContent{
		content:  []mcp.Content{emptyTextContent},
		mimeType: "text/plain",
	}

	err = validator.ValidateResourceContent(emptyTextResourceContent)
	if err == nil {
		t.Error("Expected error for content with empty text, got nil")
	}

	// Content with empty blob
	emptyBlobContent := &mockContent{
		contentType: "blob",
		blob:        []byte{},
	}
	emptyBlobResourceContent := &mockResourceContent{
		content:  []mcp.Content{emptyBlobContent},
		mimeType: "application/octet-stream",
	}

	err = validator.ValidateResourceContent(emptyBlobResourceContent)
	if err == nil {
		t.Error("Expected error for content with empty blob, got nil")
	}

	// Content with unsupported type
	unsupportedContent := &mockContent{
		contentType: "unsupported",
		text:        "test",
	}
	unsupportedResourceContent := &mockResourceContent{
		content:  []mcp.Content{unsupportedContent},
		mimeType: "text/plain",
	}

	err = validator.ValidateResourceContent(unsupportedResourceContent)
	if err == nil {
		t.Error("Expected error for content with unsupported type, got nil")
	}
}