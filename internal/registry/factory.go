package registry

import (
	"context"
	"fmt"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
)

// BaseFactory defines the common interface for all entity factories
type BaseFactory interface {
	GetName() string
	GetDescription() string
	GetVersion() string
	GetCapabilities() []string
}

// RegistryFactory creates registry instances
type RegistryFactory[T any] interface {
	CreateRegistry() (T, error)
	GetSupportedTypes() []string
	GetDefaultType() string
}

// BaseRegistryFactory provides common functionality for registry factories
type BaseRegistryFactory struct {
	config *config.Config
	logger *logger.Logger
}

// NewBaseRegistryFactory creates a new base registry factory
func NewBaseRegistryFactory(cfg *config.Config, log *logger.Logger) *BaseRegistryFactory {
	return &BaseRegistryFactory{
		config: cfg,
		logger: log,
	}
}

// GetConfig returns the configuration
func (f *BaseRegistryFactory) GetConfig() *config.Config {
	return f.config
}

// GetLogger returns the logger
func (f *BaseRegistryFactory) GetLogger() *logger.Logger {
	return f.logger
}

// EntityConfig represents common configuration for entities
type EntityConfig struct {
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config"`
}

// ValidateEntityConfig performs basic validation on entity configuration
func ValidateEntityConfig(config EntityConfig) error {
	var errors ValidationErrors
	
	// Validate configuration map
	if config.Config != nil {
		for key, value := range config.Config {
			if key == "" {
				errors.Add("config", key, "configuration key cannot be empty")
			}
			if value == nil {
				errors.Add("config", key, "configuration value cannot be nil")
			}
		}
	}
	
	if errors.HasErrors() {
		return errors
	}
	
	return nil
}

// LifecycleManager defines the interface for managing entity lifecycle
type LifecycleManager interface {
	// Entity lifecycle operations
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
	
	// Status management
	TransitionStatus(identifier string, newStatus LifecycleStatus) error
	GetStatus(identifier string) (LifecycleStatus, error)
	
	// Health monitoring
	Health() RegistryHealth
}

// BaseLifecycleManager provides common lifecycle management functionality
type BaseLifecycleManager struct {
	running   bool
	logger    *logger.Logger
	config    *config.Config
	validator *BaseValidator
}

// NewBaseLifecycleManager creates a new base lifecycle manager
func NewBaseLifecycleManager(cfg *config.Config, log *logger.Logger) *BaseLifecycleManager {
	return &BaseLifecycleManager{
		running:   false,
		logger:    log,
		config:    cfg,
		validator: NewBaseValidator(cfg, log),
	}
}

// Start implements basic start functionality
func (lm *BaseLifecycleManager) Start(ctx context.Context) error {
	if lm.running {
		return fmt.Errorf("lifecycle manager is already running")
	}
	
	lm.logger.Info("starting lifecycle manager")
	lm.running = true
	lm.logger.Info("lifecycle manager started successfully")
	
	return nil
}

// Stop implements basic stop functionality
func (lm *BaseLifecycleManager) Stop(ctx context.Context) error {
	if !lm.running {
		return nil
	}
	
	lm.logger.Info("stopping lifecycle manager")
	lm.running = false
	lm.logger.Info("lifecycle manager stopped")
	
	return nil
}

// IsRunning returns whether the lifecycle manager is running
func (lm *BaseLifecycleManager) IsRunning() bool {
	return lm.running
}

// ValidateStatusTransition validates a status transition request
func (lm *BaseLifecycleManager) ValidateStatusTransition(identifier string, currentStatus, newStatus LifecycleStatus) error {
	if !IsValidTransition(currentStatus, newStatus) {
		lm.logger.Error("invalid status transition attempted",
			"identifier", identifier,
			"current_status", string(currentStatus),
			"new_status", string(newStatus),
		)
		return fmt.Errorf("%w: cannot transition from %s to %s", 
			ErrInvalidTransition, string(currentStatus), string(newStatus))
	}

	if !lm.running && (newStatus == StatusActive || newStatus == StatusLoaded) {
		lm.logger.Error("cannot activate entity when lifecycle manager is not running",
			"identifier", identifier,
			"new_status", string(newStatus),
		)
		return fmt.Errorf("%w: cannot transition to %s when registry is stopped", 
			ErrRegistryNotRunning, string(newStatus))
	}

	return nil
}

// GetValidator returns the base validator
func (lm *BaseLifecycleManager) GetValidator() *BaseValidator {
	return lm.validator
}

// GetLogger returns the logger
func (lm *BaseLifecycleManager) GetLogger() *logger.Logger {
	return lm.logger
}

// GetConfig returns the configuration
func (lm *BaseLifecycleManager) GetConfig() *config.Config {
	return lm.config
}