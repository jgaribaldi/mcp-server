package tools

import (
	"context"
	"fmt"

	"mcp-server/internal/mcp"
)

// ToolStatus represents the current state of a tool
type ToolStatus string

const (
	ToolStatusUnknown    ToolStatus = "unknown"
	ToolStatusRegistered ToolStatus = "registered"
	ToolStatusLoaded     ToolStatus = "loaded"
	ToolStatusActive     ToolStatus = "active"
	ToolStatusError      ToolStatus = "error"
	ToolStatusDisabled   ToolStatus = "disabled"
)

// ToolInfo provides metadata about available tools
type ToolInfo struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Version      string            `json:"version"`
	Capabilities []string          `json:"capabilities"`
	Requirements map[string]string `json:"requirements"`
	Status       ToolStatus        `json:"status"`
}

// ToolConfig represents configuration for a specific tool
type ToolConfig struct {
	Enabled    bool                   `json:"enabled"`
	Config     map[string]interface{} `json:"config"`
	Timeout    int                    `json:"timeout_seconds"`
	MaxRetries int                    `json:"max_retries"`
}

// ToolFactory creates tool instances
type ToolFactory interface {
	Name() string
	Description() string
	Version() string
	Capabilities() []string
	Requirements() map[string]string
	Create(ctx context.Context, config ToolConfig) (mcp.Tool, error)
	Validate(config ToolConfig) error
}

// RegistryHealth represents the health status of the tool registry
type RegistryHealth struct {
	Status       string              `json:"status"`
	ToolCount    int                 `json:"tool_count"`
	ActiveTools  int                 `json:"active_tools"`
	ErrorTools   int                 `json:"error_tools"`
	LastCheck    string              `json:"last_check"`
	Errors       []string            `json:"errors,omitempty"`
	ToolStatuses map[string]string   `json:"tool_statuses"`
}

// ToolRegistry manages the collection of available MCP tools
type ToolRegistry interface {
	// Tool management
	Register(name string, factory ToolFactory) error
	Unregister(name string) error
	Get(name string) (mcp.Tool, error)
	GetFactory(name string) (ToolFactory, error)
	List() []ToolInfo

	// Tool lifecycle
	LoadTools(ctx context.Context) error
	ValidateTools(ctx context.Context) error
	TransitionStatus(name string, newStatus ToolStatus) error

	// Registry operations
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health() RegistryHealth
}

// Tool registry errors
var (
	ErrToolNotFound        = fmt.Errorf("tool not found")
	ErrToolAlreadyExists   = fmt.Errorf("tool already exists")
	ErrInvalidToolName     = fmt.Errorf("invalid tool name")
	ErrToolValidation      = fmt.Errorf("tool validation failed")
	ErrRegistryNotRunning  = fmt.Errorf("registry not running")
	ErrToolCreation        = fmt.Errorf("tool creation failed")
	ErrInvalidTransition   = fmt.Errorf("invalid status transition")
	ErrTransitionNotAllowed = fmt.Errorf("status transition not allowed")
)

// ToolValidationError represents a validation error with details
type ToolValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

func (e ToolValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s' (value: '%s'): %s", e.Field, e.Value, e.Message)
}

// StatusTransition represents a valid status transition
type StatusTransition struct {
	From ToolStatus
	To   ToolStatus
}

// ValidStatusTransitions defines the allowed status transitions
var ValidStatusTransitions = map[StatusTransition]bool{
	// From registered
	{ToolStatusRegistered, ToolStatusLoaded}:   true,
	{ToolStatusRegistered, ToolStatusError}:    true,
	{ToolStatusRegistered, ToolStatusDisabled}: true,
	
	// From loaded
	{ToolStatusLoaded, ToolStatusActive}:   true,
	{ToolStatusLoaded, ToolStatusError}:    true,
	{ToolStatusLoaded, ToolStatusDisabled}: true,
	
	// From active
	{ToolStatusActive, ToolStatusError}:    true,
	{ToolStatusActive, ToolStatusDisabled}: true,
	{ToolStatusActive, ToolStatusLoaded}:   true, // downgrade
	
	// From error
	{ToolStatusError, ToolStatusRegistered}: true, // restart
	{ToolStatusError, ToolStatusDisabled}:   true,
	
	// From disabled
	{ToolStatusDisabled, ToolStatusRegistered}: true, // enable
	{ToolStatusDisabled, ToolStatusError}:      true,
}

// IsValidTransition checks if a status transition is allowed
func IsValidTransition(from, to ToolStatus) bool {
	if from == to {
		return true // same status is always valid
	}
	return ValidStatusTransitions[StatusTransition{From: from, To: to}]
}

// GetAllowedTransitions returns all valid transitions from a given status
func GetAllowedTransitions(from ToolStatus) []ToolStatus {
	var allowed []ToolStatus
	
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

// ToolValidationErrors represents multiple validation errors
type ToolValidationErrors []ToolValidationError

func (e ToolValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	if len(e) == 1 {
		return e[0].Error()
	}
	return fmt.Sprintf("%d validation errors: %s (and %d more)", len(e), e[0].Error(), len(e)-1)
}

// Add appends a validation error
func (e *ToolValidationErrors) Add(field, value, message string) {
	*e = append(*e, ToolValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	})
}

// HasErrors returns true if there are validation errors
func (e ToolValidationErrors) HasErrors() bool {
	return len(e) > 0
}