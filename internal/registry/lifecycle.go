package registry

import "fmt"

// LifecycleStatus represents the current state of a managed entity (tool or resource)
type LifecycleStatus string

const (
	StatusUnknown    LifecycleStatus = "unknown"
	StatusRegistered LifecycleStatus = "registered"
	StatusLoaded     LifecycleStatus = "loaded"
	StatusActive     LifecycleStatus = "active"
	StatusError      LifecycleStatus = "error"
	StatusDisabled   LifecycleStatus = "disabled"
)

// StatusTransition represents a valid status transition
type StatusTransition struct {
	From LifecycleStatus
	To   LifecycleStatus
}

// ValidStatusTransitions defines the allowed status transitions
var ValidStatusTransitions = map[StatusTransition]bool{
	// From registered
	{StatusRegistered, StatusLoaded}:   true,
	{StatusRegistered, StatusError}:    true,
	{StatusRegistered, StatusDisabled}: true,
	
	// From loaded
	{StatusLoaded, StatusActive}:   true,
	{StatusLoaded, StatusError}:    true,
	{StatusLoaded, StatusDisabled}: true,
	
	// From active
	{StatusActive, StatusError}:    true,
	{StatusActive, StatusDisabled}: true,
	{StatusActive, StatusLoaded}:   true, // downgrade
	
	// From error
	{StatusError, StatusRegistered}: true, // restart
	{StatusError, StatusDisabled}:   true,
	
	// From disabled
	{StatusDisabled, StatusRegistered}: true, // enable
	{StatusDisabled, StatusError}:      true,
}

// IsValidTransition checks if a status transition is allowed
func IsValidTransition(from, to LifecycleStatus) bool {
	if from == to {
		return true // same status is always valid
	}
	return ValidStatusTransitions[StatusTransition{From: from, To: to}]
}

// GetAllowedTransitions returns all valid transitions from a given status
func GetAllowedTransitions(from LifecycleStatus) []LifecycleStatus {
	var allowed []LifecycleStatus
	
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

// LifecycleEntity represents the basic interface for entities that have lifecycle management
type LifecycleEntity interface {
	GetStatus() LifecycleStatus
	GetName() string
	GetDescription() string
	GetVersion() string
}

// RegistryHealth represents the health status of a registry
type RegistryHealth struct {
	Status         string            `json:"status"`
	EntityCount    int               `json:"entity_count"`
	ActiveEntities int               `json:"active_entities"`
	ErrorEntities  int               `json:"error_entities"`
	LastCheck      string            `json:"last_check"`
	Errors         []string          `json:"errors,omitempty"`
	EntityStatuses map[string]string `json:"entity_statuses"`
}

// Common registry errors
var (
	ErrEntityNotFound        = fmt.Errorf("entity not found")
	ErrEntityAlreadyExists   = fmt.Errorf("entity already exists")
	ErrEntityValidation      = fmt.Errorf("entity validation failed")
	ErrRegistryNotRunning    = fmt.Errorf("registry not running")
	ErrEntityCreation        = fmt.Errorf("entity creation failed")
	ErrInvalidTransition     = fmt.Errorf("invalid status transition")
	ErrTransitionNotAllowed  = fmt.Errorf("status transition not allowed")
)