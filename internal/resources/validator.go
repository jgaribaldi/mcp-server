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

type ResourceValidator struct {
	config *config.Config
	logger *logger.Logger
}

func NewResourceValidator(cfg *config.Config, log *logger.Logger) *ResourceValidator {
	return &ResourceValidator{
		config: cfg,
		logger: log,
	}
}

func (v *ResourceValidator) validateStringLength(value string, fieldName string, maxLength int) error {
	if len(value) > maxLength {
		return fmt.Errorf("%s too long: %d characters (max: %d)", fieldName, len(value), maxLength)
	}
	return nil
}

func (v *ResourceValidator) addValidationError(errors *ResourceValidationErrors, field, value, message string) {
	errors.Add(field, value, message)
}

func (v *ResourceValidator) logValidationResult(success bool, entityType, uri string, errorCount int) {
	if success {
		v.logger.Debug(fmt.Sprintf("%s validation passed", entityType), "uri", uri)
	} else {
		v.logger.Error(fmt.Sprintf("%s validation failed", entityType), "uri", uri, "errors", errorCount)
	}
}

func (v *ResourceValidator) validateBasicURI(uri string) error {
	if uri == "" {
		return fmt.Errorf("URI cannot be empty")
	}

	parsedURI, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("invalid URI format: %w", err)
	}

	if parsedURI.Scheme == "" {
		return fmt.Errorf("URI must have a scheme (e.g., file://, config://, api://)")
	}

	return nil
}

func (v *ResourceValidator) validateURIScheme(uri string, parsedURI *url.URL) error {
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

	switch parsedURI.Scheme {
	case "file":
		if parsedURI.Path == "" {
			return fmt.Errorf("file URI must have a path")
		}
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

	return nil
}

func (v *ResourceValidator) validateURILength(uri string) error {
	if len(uri) > 2048 {
		return fmt.Errorf("URI too long: %d characters (max: 2048)", len(uri))
	}
	return nil
}

func (v *ResourceValidator) ValidateURI(uri string) error {
	v.logger.Debug("validating resource URI", "uri", uri)

	if err := v.validateBasicURI(uri); err != nil {
		return err
	}

	parsedURI, _ := url.Parse(uri)

	if err := v.validateURIScheme(uri, parsedURI); err != nil {
		return err
	}

	if err := v.validateURILength(uri); err != nil {
		return err
	}

	v.logger.Debug("resource URI validation passed", "uri", uri, "scheme", parsedURI.Scheme)
	return nil
}

func (v *ResourceValidator) ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if err := v.validateStringLength(name, "name", 255); err != nil {
		return err
	}

	validName := regexp.MustCompile(`^[a-zA-Z0-9\-_ ]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("name contains invalid characters (allowed: a-z, A-Z, 0-9, -, _, space)")
	}

	return nil
}

func (v *ResourceValidator) ValidateMimeType(mimeType string) error {
	if mimeType == "" {
		return fmt.Errorf("MIME type cannot be empty")
	}

	parts := strings.Split(mimeType, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid MIME type format: expected 'type/subtype'")
	}

	if parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("MIME type parts cannot be empty")
	}

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

func (v *ResourceValidator) validateFactoryBasics(factory ResourceFactory, errors *ResourceValidationErrors) {
	if err := v.ValidateURI(factory.URI()); err != nil {
		v.addValidationError(errors, "uri", factory.URI(), err.Error())
	}

	if err := v.ValidateName(factory.Name()); err != nil {
		v.addValidationError(errors, "name", factory.Name(), err.Error())
	}

	if factory.Description() == "" {
		v.addValidationError(errors, "description", "", "description cannot be empty")
	} else if err := v.validateStringLength(factory.Description(), "description", 1000); err != nil {
		v.addValidationError(errors, "description", factory.Description(), err.Error())
	}
}

func (v *ResourceValidator) validateFactoryMetadata(factory ResourceFactory, errors *ResourceValidationErrors) {
	if err := v.ValidateMimeType(factory.MimeType()); err != nil {
		v.addValidationError(errors, "mime_type", factory.MimeType(), err.Error())
	}

	if factory.Version() == "" {
		v.addValidationError(errors, "version", "", "version cannot be empty")
	} else if err := v.validateStringLength(factory.Version(), "version", 50); err != nil {
		v.addValidationError(errors, "version", factory.Version(), err.Error())
	}
}

func (v *ResourceValidator) validateFactoryCapabilities(factory ResourceFactory, errors *ResourceValidationErrors) {
	capabilities := factory.Capabilities()
	if len(capabilities) == 0 {
		v.addValidationError(errors, "capabilities", "", "at least one capability must be specified")
		return
	}

	for i, capability := range capabilities {
		if capability == "" {
			v.addValidationError(errors, "capabilities", fmt.Sprintf("[%d]", i), "capability cannot be empty")
		} else if err := v.validateStringLength(capability, "capability", 100); err != nil {
			v.addValidationError(errors, "capabilities", capability, err.Error())
		}
	}
}

func (v *ResourceValidator) validateFactoryTags(factory ResourceFactory, errors *ResourceValidationErrors) {
	tags := factory.Tags()
	for i, tag := range tags {
		if tag == "" {
			v.addValidationError(errors, "tags", fmt.Sprintf("[%d]", i), "tag cannot be empty")
		} else if err := v.validateStringLength(tag, "tag", 50); err != nil {
			v.addValidationError(errors, "tags", tag, err.Error())
		}
	}
}

func (v *ResourceValidator) ValidateFactory(factory ResourceFactory) error {
	v.logger.Debug("validating resource factory", "uri", factory.URI())

	var errors ResourceValidationErrors

	v.validateFactoryBasics(factory, &errors)
	v.validateFactoryMetadata(factory, &errors)
	v.validateFactoryCapabilities(factory, &errors)
	v.validateFactoryTags(factory, &errors)

	if errors.HasErrors() {
		v.logValidationResult(false, "resource factory", factory.URI(), len(errors))
		return errors
	}

	v.logValidationResult(true, "resource factory", factory.URI(), 0)
	return nil
}

func (v *ResourceValidator) validateResourceBasics(resource mcp.Resource, errors *ResourceValidationErrors) {
	if err := v.ValidateURI(resource.URI()); err != nil {
		v.addValidationError(errors, "uri", resource.URI(), err.Error())
	}

	if err := v.ValidateName(resource.Name()); err != nil {
		v.addValidationError(errors, "name", resource.Name(), err.Error())
	}

	if resource.Description() == "" {
		v.addValidationError(errors, "description", "", "description cannot be empty")
	}
}

func (v *ResourceValidator) validateResourceMetadata(resource mcp.Resource, errors *ResourceValidationErrors) {
	if err := v.ValidateMimeType(resource.MimeType()); err != nil {
		v.addValidationError(errors, "mime_type", resource.MimeType(), err.Error())
	}
}

func (v *ResourceValidator) validateResourceHandler(resource mcp.Resource, errors *ResourceValidationErrors) {
	if resource.Handler() == nil {
		v.addValidationError(errors, "handler", "", "resource handler cannot be nil")
	}
}

func (v *ResourceValidator) ValidateResource(resource mcp.Resource) error {
	v.logger.Debug("validating resource instance", "uri", resource.URI())

	var errors ResourceValidationErrors

	v.validateResourceBasics(resource, &errors)
	v.validateResourceMetadata(resource, &errors)
	v.validateResourceHandler(resource, &errors)

	if errors.HasErrors() {
		v.logValidationResult(false, "resource", resource.URI(), len(errors))
		return errors
	}

	v.logValidationResult(true, "resource", resource.URI(), 0)
	return nil
}

func (v *ResourceValidator) validateCacheTimeout(config ResourceConfig, errors *ResourceValidationErrors) {
	if config.CacheTimeout < 0 {
		v.addValidationError(errors, "cache_timeout", fmt.Sprintf("%d", config.CacheTimeout), 
			"cache timeout cannot be negative")
	} else if config.CacheTimeout > 86400 {
		v.addValidationError(errors, "cache_timeout", fmt.Sprintf("%d", config.CacheTimeout), 
			"cache timeout too large: maximum 86400 seconds (24 hours)")
	}
}

func (v *ResourceValidator) validateAccessControl(config ResourceConfig, errors *ResourceValidationErrors) {
	for key, value := range config.AccessControl {
		if key == "" {
			v.addValidationError(errors, "access_control", "", "access control key cannot be empty")
		}
		if value == "" {
			v.addValidationError(errors, "access_control", key, "access control value cannot be empty")
		}
	}
}

func (v *ResourceValidator) validateConfigMap(config ResourceConfig, errors *ResourceValidationErrors) {
	for key, value := range config.Config {
		if key == "" {
			v.addValidationError(errors, "config", "", "configuration key cannot be empty")
		}
		if value == nil {
			v.addValidationError(errors, "config", key, "configuration value cannot be nil")
		}
	}
}

func (v *ResourceValidator) ValidateConfig(config ResourceConfig) error {
	v.logger.Debug("validating resource configuration")

	var errors ResourceValidationErrors

	v.validateCacheTimeout(config, &errors)
	v.validateAccessControl(config, &errors)
	v.validateConfigMap(config, &errors)

	if errors.HasErrors() {
		v.logger.Error("resource configuration validation failed", "errors", len(errors))
		return errors
	}

	v.logger.Debug("resource configuration validation passed")
	return nil
}

func (v *ResourceValidator) ValidateCacheExpiration(cached CachedContent) error {
	now := time.Now()
	if now.After(cached.ExpiresAt) {
		return fmt.Errorf("cached content expired at %s", cached.ExpiresAt.Format("2006-01-02 15:04:05"))
	}
	return nil
}

func (v *ResourceValidator) validateContentMimeType(content mcp.ResourceContent) error {
	if err := v.ValidateMimeType(content.GetMimeType()); err != nil {
		return fmt.Errorf("invalid content MIME type: %w", err)
	}
	return nil
}

func (v *ResourceValidator) validateContentStructure(content mcp.ResourceContent) error {
	contentItems := content.GetContent()
	if len(contentItems) == 0 {
		return fmt.Errorf("resource content cannot be empty")
	}
	return nil
}

func (v *ResourceValidator) validateContentItem(item mcp.Content, index int) error {
	if item == nil {
		return fmt.Errorf("content item %d cannot be nil", index)
	}

	contentType := item.Type()
	if contentType == "" {
		return fmt.Errorf("content item %d must have a type", index)
	}

	switch contentType {
	case "text":
		if item.GetText() == "" {
			return fmt.Errorf("text content item %d cannot be empty", index)
		}
	case "blob":
		if len(item.GetBlob()) == 0 {
			return fmt.Errorf("blob content item %d cannot be empty", index)
		}
	default:
		return fmt.Errorf("unsupported content type: %s", contentType)
	}

	return nil
}

func (v *ResourceValidator) validateContentItems(content mcp.ResourceContent) error {
	contentItems := content.GetContent()
	for i, item := range contentItems {
		if err := v.validateContentItem(item, i); err != nil {
			return err
		}
	}
	return nil
}

func (v *ResourceValidator) ValidateResourceContent(content mcp.ResourceContent) error {
	v.logger.Debug("validating resource content")

	if err := v.validateContentMimeType(content); err != nil {
		return err
	}

	if err := v.validateContentStructure(content); err != nil {
		return err
	}

	if err := v.validateContentItems(content); err != nil {
		return err
	}

	contentItems := content.GetContent()
	v.logger.Debug("resource content validation passed", "items", len(contentItems))
	return nil
}