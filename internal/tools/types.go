package tools

import (
	"context"
	"fmt"

	"mcp-server/internal/mcp"
	"mcp-server/internal/registry"
)

type ToolStatus = registry.LifecycleStatus

const (
	ToolStatusUnknown    = registry.StatusUnknown
	ToolStatusRegistered = registry.StatusRegistered
	ToolStatusLoaded     = registry.StatusLoaded
	ToolStatusActive     = registry.StatusActive
	ToolStatusError      = registry.StatusError
	ToolStatusDisabled   = registry.StatusDisabled
)

type ToolInfo struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Version      string            `json:"version"`
	Capabilities []string          `json:"capabilities"`
	Requirements map[string]string `json:"requirements"`
	Status       ToolStatus        `json:"status"`
}

type ToolConfig struct {
	Enabled    bool                   `json:"enabled"`
	Config     map[string]interface{} `json:"config"`
	Timeout    int                    `json:"timeout_seconds"`
	MaxRetries int                    `json:"max_retries"`
}

type ToolFactory interface {
	registry.BaseFactory
	Requirements() map[string]string
	Create(ctx context.Context, config ToolConfig) (mcp.Tool, error)
	Validate(config ToolConfig) error
}

type RegistryHealth struct {
	Status       string              `json:"status"`
	ToolCount    int                 `json:"tool_count"`
	ActiveTools  int                 `json:"active_tools"`
	ErrorTools   int                 `json:"error_tools"`
	LastCheck    string              `json:"last_check"`
	Errors       []string            `json:"errors,omitempty"`
	ToolStatuses map[string]string   `json:"tool_statuses"`
}

type ToolRegistry interface {
	Register(name string, factory ToolFactory) error
	Unregister(name string) error
	Get(name string) (mcp.Tool, error)
	GetFactory(name string) (ToolFactory, error)
	List() []ToolInfo

	LoadTools(ctx context.Context) error
	ValidateTools(ctx context.Context) error
	TransitionStatus(name string, newStatus ToolStatus) error
	RestartTool(ctx context.Context, name string) error

	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health() RegistryHealth
}

var (
	ErrToolNotFound        = registry.ErrEntityNotFound
	ErrToolAlreadyExists   = registry.ErrEntityAlreadyExists
	ErrInvalidToolName     = fmt.Errorf("invalid tool name")
	ErrToolValidation      = registry.ErrEntityValidation
	ErrRegistryNotRunning  = registry.ErrRegistryNotRunning
	ErrToolCreation        = registry.ErrEntityCreation
	ErrInvalidTransition   = registry.ErrInvalidTransition
	ErrTransitionNotAllowed = registry.ErrTransitionNotAllowed
	ErrToolRestart         = fmt.Errorf("tool restart failed")
	ErrRestartNotAllowed   = fmt.Errorf("tool restart not allowed")
)

type ToolValidationError = registry.ValidationError
type ToolValidationErrors = registry.ValidationErrors

type StatusTransition = registry.StatusTransition

var ValidStatusTransitions = registry.ValidStatusTransitions

func IsValidTransition(from, to ToolStatus) bool {
	return registry.IsValidTransition(from, to)
}

func GetAllowedTransitions(from ToolStatus) []ToolStatus {
	return registry.GetAllowedTransitions(from)
}