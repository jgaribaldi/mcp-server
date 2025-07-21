package resources

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
)

// ResourceValidator validates resource configurations and instances
type ResourceValidator struct {
	config *config.Config
	logger *logger.Logger
}

// NewResourceValidator creates a new resource validator
func NewResourceValidator(cfg *config.Config, log *logger.Logger) *ResourceValidator {
	return &ResourceValidator{
		config: cfg,
		logger: log,
	}
}

// ValidateURI validates resource URI format and pattern
func (v *ResourceValidator) ValidateURI(uri string) error {
	v.logger.Debug("validating resource URI", "uri", uri)

	if uri == "" {
		return fmt.Errorf("URI cannot be empty")
	}

	// Parse URI to validate format
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("invalid URI format: %w", err)
	}

	// Validate scheme
	if parsedURI.Scheme == "" {
		return fmt.Errorf("URI must have a scheme (e.g., file://, config://, api://)")
	}

	// Validate supported schemes
	supportedSchemes := map[string]bool{
		"file":   true,
		"config": true,
		"api":    true,
		"custom": true,
		"http":   true,
		"https":  true,
	}

	if !supportedSchemes[parsedURI.Scheme] {
		return fmt.Errorf("unsupported URI scheme: %s (supported: file, config, api, custom, http, https)", parsedURI.Scheme)
	}

	// Validate URI length
	if len(uri) > 2048 {
		return fmt.Errorf("URI too long: %d characters (max: 2048)", len(uri))
	}

	// Additional scheme-specific validation
	switch parsedURI.Scheme {
	case "file":
		if parsedURI.Path == "" {
			return fmt.Errorf("file URI must have a path")
		}
		// Validate no dangerous path traversal
		if strings.Contains(parsedURI.Path, "..") {
			return fmt.Errorf("path traversal not allowed in file URI")
		}
	case "config":
		if parsedURI.Host == "" && parsedURI.Path == "" {
			return fmt.Errorf("config URI must have host or path")
		}
	case "api":
		if parsedURI.Host == "" {
			return fmt.Errorf("api URI must have a host")
		}
	}

	v.logger.Debug("resource URI validation passed", "uri", uri, "scheme", parsedURI.Scheme)
	return nil
}

// ValidateName validates resource name
func (v *ResourceValidator) ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if len(name) > 255 {
		return fmt.Errorf("name too long: %d characters (max: 255)", len(name))
	}

	// Name should contain only alphanumeric characters, hyphens, underscores, and spaces
	validName := regexp.MustCompile(`^[a-zA-Z0-9\-_ ]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("name contains invalid characters (allowed: a-z, A-Z, 0-9, -, _, space)")
	}

	return nil
}

// ValidateMimeType validates MIME type format
func (v *ResourceValidator) ValidateMimeType(mimeType string) error {
	if mimeType == "" {
		return fmt.Errorf("MIME type cannot be empty")
	}

	// Basic MIME type format validation
	parts := strings.Split(mimeType, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid MIME type format: expected 'type/subtype'")
	}

	if parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("MIME type parts cannot be empty")
	}

	// Common MIME types validation
	validTypes := map[string]bool{
		"text":        true,
		"application": true,
		"image":       true,
		"audio":       true,
		"video":       true,
		"multipart":   true,
		"message":     true,
	}

	if !validTypes[parts[0]] {
		return fmt.Errorf("unsupported MIME type: %s (supported main types: text, application, image, audio, video, multipart, message)", parts[0])
	}

	return nil
}

// ValidateFactory validates a resource factory
func (v *ResourceValidator) ValidateFactory(factory ResourceFactory) error {
	v.logger.Debug("validating resource factory", "uri", factory.URI())

	var errors ResourceValidationErrors

	// Validate URI
	if err := v.ValidateURI(factory.URI()); err != nil {
		errors.Add("uri", factory.URI(), err.Error())
	}

	// Validate name
	if err := v.ValidateName(factory.Name()); err != nil {
		errors.Add("name", factory.Name(), err.Error())
	}

	// Validate description
	if factory.Description() == "" {
		errors.Add("description", "", "description cannot be empty")
	} else if len(factory.Description()) > 1000 {
		errors.Add("description", factory.Description(), 
			fmt.Sprintf("description too long: %d characters (max: 1000)", len(factory.Description())))
	}

	// Validate MIME type
	if err := v.ValidateMimeType(factory.MimeType()); err != nil {
		errors.Add("mime_type", factory.MimeType(), err.Error())
	}

	// Validate version
	if factory.Version() == "" {
		errors.Add("version", "", "version cannot be empty")
	} else if len(factory.Version()) > 50 {
		errors.Add("version", factory.Version(), 
			fmt.Sprintf("version too long: %d characters (max: 50)", len(factory.Version())))
	}

	// Validate capabilities
	capabilities := factory.Capabilities()
	if len(capabilities) == 0 {
		errors.Add("capabilities", "", "at least one capability must be specified")
	} else {
		for i, capability := range capabilities {
			if capability == "" {
				errors.Add("capabilities", fmt.Sprintf("[%d]", i), "capability cannot be empty")
			} else if len(capability) > 100 {
				errors.Add("capabilities", capability, 
					fmt.Sprintf("capability too long: %d characters (max: 100)", len(capability)))
			}
		}
	}

	// Validate tags
	tags := factory.Tags()
	for i, tag := range tags {
		if tag == "" {
			errors.Add("tags", fmt.Sprintf("[%d]", i), "tag cannot be empty")
		} else if len(tag) > 50 {
			errors.Add("tags", tag, 
				fmt.Sprintf("tag too long: %d characters (max: 50)", len(tag)))
		}
	}

	if errors.HasErrors() {
		v.logger.Error("resource factory validation failed", 
			"uri", factory.URI(), 
			"errors", len(errors))
		return errors
	}

	v.logger.Debug("resource factory validation passed", "uri", factory.URI())
	return nil
}

// ValidateResource validates a resource instance
func (v *ResourceValidator) ValidateResource(resource mcp.Resource) error {
	v.logger.Debug("validating resource instance", "uri", resource.URI())

	var errors ResourceValidationErrors

	// Validate URI
	if err := v.ValidateURI(resource.URI()); err != nil {
		errors.Add("uri", resource.URI(), err.Error())
	}

	// Validate name
	if err := v.ValidateName(resource.Name()); err != nil {
		errors.Add("name", resource.Name(), err.Error())
	}

	// Validate description
	if resource.Description() == "" {
		errors.Add("description", "", "description cannot be empty")
	}

	// Validate MIME type
	if err := v.ValidateMimeType(resource.MimeType()); err != nil {
		errors.Add("mime_type", resource.MimeType(), err.Error())
	}

	// Validate handler exists
	if resource.Handler() == nil {
		errors.Add("handler", "", "resource handler cannot be nil")
	}

	if errors.HasErrors() {
		v.logger.Error("resource validation failed", 
			"uri", resource.URI(), 
			"errors", len(errors))
		return errors
	}

	v.logger.Debug("resource validation passed", "uri", resource.URI())
	return nil
}

// ValidateConfig validates resource configuration
func (v *ResourceValidator) ValidateConfig(config ResourceConfig) error {
	v.logger.Debug("validating resource configuration")

	var errors ResourceValidationErrors

	// Validate cache timeout
	if config.CacheTimeout < 0 {
		errors.Add("cache_timeout", fmt.Sprintf("%d", config.CacheTimeout), 
			"cache timeout cannot be negative")
	} else if config.CacheTimeout > 86400 { // 24 hours
		errors.Add("cache_timeout", fmt.Sprintf("%d", config.CacheTimeout), 
			"cache timeout too large: maximum 86400 seconds (24 hours)")
	}

	// Validate access control
	for key, value := range config.AccessControl {
		if key == "" {
			errors.Add("access_control", "", "access control key cannot be empty")
		}
		if value == "" {
			errors.Add("access_control", key, "access control value cannot be empty")
		}
	}

	// Validate configuration parameters
	for key, value := range config.Config {
		if key == "" {
			errors.Add("config", "", "configuration key cannot be empty")
		}
		// Basic value validation
		if value == nil {
			errors.Add("config", key, "configuration value cannot be nil")
		}
	}

	if errors.HasErrors() {
		v.logger.Error("resource configuration validation failed", "errors", len(errors))
		return errors
	}

	v.logger.Debug("resource configuration validation passed")
	return nil
}

// ValidateCacheExpiration checks if cached content has expired
func (v *ResourceValidator) ValidateCacheExpiration(cached CachedContent) error {
	now := time.Now()
	if now.After(cached.ExpiresAt) {
		return fmt.Errorf("cached content expired at %s", cached.ExpiresAt.Format("2006-01-02 15:04:05"))
	}
	return nil
}

// ValidateResourceContent validates resource content
func (v *ResourceValidator) ValidateResourceContent(content mcp.ResourceContent) error {
	v.logger.Debug("validating resource content")

	// Validate MIME type
	if err := v.ValidateMimeType(content.GetMimeType()); err != nil {
		return fmt.Errorf("invalid content MIME type: %w", err)
	}

	// Validate content exists
	contentItems := content.GetContent()
	if len(contentItems) == 0 {
		return fmt.Errorf("resource content cannot be empty")
	}

	// Validate each content item
	for i, item := range contentItems {
		if item == nil {
			return fmt.Errorf("content item %d cannot be nil", i)
		}

		// Validate content type
		contentType := item.Type()
		if contentType == "" {
			return fmt.Errorf("content item %d must have a type", i)
		}

		// Type-specific validation
		switch contentType {
		case "text":
			if item.GetText() == "" {
				return fmt.Errorf("text content item %d cannot be empty", i)
			}
		case "blob":
			if len(item.GetBlob()) == 0 {
				return fmt.Errorf("blob content item %d cannot be empty", i)
			}
		default:
			return fmt.Errorf("unsupported content type: %s", contentType)
		}
	}

	v.logger.Debug("resource content validation passed", "items", len(contentItems))
	return nil
}