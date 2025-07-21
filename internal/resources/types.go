package resources

import (
	"context"
	"fmt"
	"time"

	"mcp-server/internal/mcp"
)

// ResourceStatus represents the current state of a resource
type ResourceStatus string

const (
	ResourceStatusUnknown    ResourceStatus = "unknown"
	ResourceStatusRegistered ResourceStatus = "registered"
	ResourceStatusLoaded     ResourceStatus = "loaded"
	ResourceStatusActive     ResourceStatus = "active"
	ResourceStatusError      ResourceStatus = "error"
	ResourceStatusDisabled   ResourceStatus = "disabled"
)

// ResourceInfo provides metadata about available resources
type ResourceInfo struct {
	URI          string            `json:"uri"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	MimeType     string            `json:"mime_type"`
	Version      string            `json:"version"`
	Tags         []string          `json:"tags"`
	Capabilities []string          `json:"capabilities"`
	Status       ResourceStatus    `json:"status"`
	Metadata     map[string]string `json:"metadata"`
}

// ResourceConfig represents configuration for a specific resource
type ResourceConfig struct {
	Enabled       bool                   `json:"enabled"`
	Config        map[string]interface{} `json:"config"`
	CacheTimeout  int                    `json:"cache_timeout_seconds"`
	AccessControl map[string]string      `json:"access_control"`
}

// CachedContent represents cached resource content with metadata
type CachedContent struct {
	Content     mcp.ResourceContent
	Timestamp   time.Time
	ExpiresAt   time.Time
	AccessCount int64
}

// ResourceFactory creates resource instances
type ResourceFactory interface {
	URI() string
	Name() string
	Description() string
	MimeType() string
	Version() string
	Tags() []string
	Capabilities() []string
	Create(ctx context.Context, config ResourceConfig) (mcp.Resource, error)
	Validate(config ResourceConfig) error
}

// RegistryHealth represents the health status of the resource registry
type RegistryHealth struct {
	Status            string              `json:"status"`
	ResourceCount     int                 `json:"resource_count"`
	ActiveResources   int                 `json:"active_resources"`
	ErrorResources    int                 `json:"error_resources"`
	CachedResources   int                 `json:"cached_resources"`
	CacheHitRate      float64             `json:"cache_hit_rate"`
	LastCheck         string              `json:"last_check"`
	Errors            []string            `json:"errors,omitempty"`
	ResourceStatuses  map[string]string   `json:"resource_statuses"`
	CircuitBreakers   map[string]string   `json:"circuit_breakers"`
}

// ResourceRegistry manages the collection of available MCP resources
type ResourceRegistry interface {
	// Resource management
	Register(uri string, factory ResourceFactory) error
	Unregister(uri string) error
	Get(uri string) (mcp.Resource, error)
	GetFactory(uri string) (ResourceFactory, error)
	List() []ResourceInfo

	// Resource lifecycle
	LoadResources(ctx context.Context) error
	ValidateResources(ctx context.Context) error
	TransitionStatus(uri string, newStatus ResourceStatus) error
	RefreshResource(ctx context.Context, uri string) error

	// Registry operations
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health() RegistryHealth
}

// Resource registry errors
var (
	ErrResourceNotFound       = fmt.Errorf("resource not found")
	ErrResourceAlreadyExists  = fmt.Errorf("resource already exists")
	ErrInvalidResourceURI     = fmt.Errorf("invalid resource URI")
	ErrResourceValidation     = fmt.Errorf("resource validation failed")
	ErrResourceAccess         = fmt.Errorf("resource access denied")
	ErrResourceContent        = fmt.Errorf("resource content error")
	ErrCacheExpired          = fmt.Errorf("cached content expired")
	ErrRegistryNotRunning    = fmt.Errorf("registry not running")
	ErrResourceCreation      = fmt.Errorf("resource creation failed")
	ErrInvalidTransition     = fmt.Errorf("invalid status transition")
	ErrTransitionNotAllowed  = fmt.Errorf("status transition not allowed")
	ErrResourceRefresh       = fmt.Errorf("resource refresh failed")
	ErrRefreshNotAllowed     = fmt.Errorf("resource refresh not allowed")
)

// ResourceValidationError represents a validation error with details
type ResourceValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

func (e ResourceValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s' (value: '%s'): %s", e.Field, e.Value, e.Message)
}

// StatusTransition represents a valid status transition
type StatusTransition struct {
	From ResourceStatus
	To   ResourceStatus
}

// ValidStatusTransitions defines the allowed status transitions
var ValidStatusTransitions = map[StatusTransition]bool{
	// From registered
	{ResourceStatusRegistered, ResourceStatusLoaded}:   true,
	{ResourceStatusRegistered, ResourceStatusError}:    true,
	{ResourceStatusRegistered, ResourceStatusDisabled}: true,
	
	// From loaded
	{ResourceStatusLoaded, ResourceStatusActive}:   true,
	{ResourceStatusLoaded, ResourceStatusError}:    true,
	{ResourceStatusLoaded, ResourceStatusDisabled}: true,
	
	// From active
	{ResourceStatusActive, ResourceStatusError}:    true,
	{ResourceStatusActive, ResourceStatusDisabled}: true,
	{ResourceStatusActive, ResourceStatusLoaded}:   true, // downgrade
	
	// From error
	{ResourceStatusError, ResourceStatusRegistered}: true, // restart
	{ResourceStatusError, ResourceStatusDisabled}:   true,
	
	// From disabled
	{ResourceStatusDisabled, ResourceStatusRegistered}: true, // enable
	{ResourceStatusDisabled, ResourceStatusError}:      true,
}

// IsValidTransition checks if a status transition is allowed
func IsValidTransition(from, to ResourceStatus) bool {
	if from == to {
		return true // same status is always valid
	}
	return ValidStatusTransitions[StatusTransition{From: from, To: to}]
}

// GetAllowedTransitions returns all valid transitions from a given status
func GetAllowedTransitions(from ResourceStatus) []ResourceStatus {
	var allowed []ResourceStatus
	
	// Same status is always allowed
	allowed = append(allowed, from)
	
	// Check all possible transitions
	for transition := range ValidStatusTransitions {
		if transition.From == from {
			allowed = append(allowed, transition.To)
		}
	}
	
	return allowed
}

// ResourceValidationErrors represents multiple validation errors
type ResourceValidationErrors []ResourceValidationError

func (e ResourceValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	if len(e) == 1 {
		return e[0].Error()
	}
	return fmt.Sprintf("%d validation errors: %s (and %d more)", len(e), e[0].Error(), len(e)-1)
}

// Add appends a validation error
func (e *ResourceValidationErrors) Add(field, value, message string) {
	*e = append(*e, ResourceValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	})
}

// HasErrors returns true if there are validation errors
func (e ResourceValidationErrors) HasErrors() bool {
	return len(e) > 0
}