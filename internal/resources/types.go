package resources

import (
	"context"
	"fmt"
	"time"

	"mcp-server/internal/mcp"
	"mcp-server/internal/registry"
)

type ResourceStatus = registry.LifecycleStatus

const (
	ResourceStatusUnknown    = registry.StatusUnknown
	ResourceStatusRegistered = registry.StatusRegistered
	ResourceStatusLoaded     = registry.StatusLoaded
	ResourceStatusActive     = registry.StatusActive
	ResourceStatusError      = registry.StatusError
	ResourceStatusDisabled   = registry.StatusDisabled
)

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

type ResourceConfig struct {
	Enabled       bool                   `json:"enabled"`
	Config        map[string]interface{} `json:"config"`
	CacheTimeout  int                    `json:"cache_timeout_seconds"`
	AccessControl map[string]string      `json:"access_control"`
}

type CachedContent struct {
	Content     mcp.ResourceContent
	Timestamp   time.Time
	ExpiresAt   time.Time
	AccessCount int64
}

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

type ResourceValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

func (e ResourceValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s' (value: '%s'): %s", e.Field, e.Value, e.Message)
}

// Use shared transition logic from registry package
var IsValidTransition = registry.IsValidTransition
var GetAllowedTransitions = registry.GetAllowedTransitions

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

func (e *ResourceValidationErrors) Add(field, value, message string) {
	*e = append(*e, ResourceValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	})
}

func (e ResourceValidationErrors) HasErrors() bool {
	return len(e) > 0
}
