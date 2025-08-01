package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
	"mcp-server/internal/tools/adapters"
)

// DefaultToolRegistry implements ToolRegistry
type DefaultToolRegistry struct {
	factories        map[string]ToolFactory
	circuitFactories map[string]*CircuitBreakerToolFactory
	tools            map[string]mcp.Tool
	toolInfo         map[string]ToolInfo
	logger           *logger.Logger
	config           *config.Config
	validator        *ToolValidator
	adapter          adapters.LibraryAdapter // Library adapter for MCP implementation
	mu               sync.RWMutex
	running          bool
	lastCheck        time.Time
}

// NewDefaultToolRegistry creates a new tool registry instance
func NewDefaultToolRegistry(cfg *config.Config, log *logger.Logger) ToolRegistry {
	return &DefaultToolRegistry{
		factories:        make(map[string]ToolFactory),
		circuitFactories: make(map[string]*CircuitBreakerToolFactory),
		tools:            make(map[string]mcp.Tool),
		toolInfo:         make(map[string]ToolInfo),
		logger:           log,
		config:           cfg,
		validator:        NewToolValidator(cfg, log),
		adapter:          nil, // No adapter for backward compatibility
	}
}

// NewDefaultToolRegistryWithAdapter creates a new tool registry instance with a library adapter
func NewDefaultToolRegistryWithAdapter(cfg *config.Config, log *logger.Logger, adapter adapters.LibraryAdapter) ToolRegistry {
	return &DefaultToolRegistry{
		factories:        make(map[string]ToolFactory),
		circuitFactories: make(map[string]*CircuitBreakerToolFactory),
		tools:            make(map[string]mcp.Tool),
		toolInfo:         make(map[string]ToolInfo),
		logger:           log,
		config:           cfg,
		validator:        NewToolValidator(cfg, log),
		adapter:          adapter,
	}
}

// Register implements ToolRegistry.Register
func (r *DefaultToolRegistry) Register(name string, factory ToolFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Info("registering tool factory",
		"name", name,
		"description", factory.GetDescription(),
		"version", factory.GetVersion(),
	)

	// Validate tool name
	if err := r.validator.ValidateName(name); err != nil {
		r.logger.Error("tool name validation failed",
			"name", name,
			"error", err,
		)
		return fmt.Errorf("%w: %v", ErrInvalidToolName, err)
	}

	// Check for duplicate registration
	if _, exists := r.factories[name]; exists {
		r.logger.Error("tool already registered",
			"name", name,
		)
		return fmt.Errorf("%w: %s", ErrToolAlreadyExists, name)
	}

	// Validate factory
	if err := r.validator.ValidateFactory(factory); err != nil {
		r.logger.Error("tool factory validation failed",
			"name", name,
			"error", err,
		)
		return fmt.Errorf("%w: %v", ErrToolValidation, err)
	}

	// Register factory
	r.factories[name] = factory

	// Create circuit breaker wrapper
	circuitConfig := DefaultCircuitBreakerConfig()
	circuitFactory := NewCircuitBreakerToolFactory(factory, circuitConfig)
	r.circuitFactories[name] = circuitFactory

	// Create tool info
	info := ToolInfo{
		Name:         factory.GetName(),
		Description:  factory.GetDescription(),
		Version:      factory.GetVersion(),
		Capabilities: factory.GetCapabilities(),
		Requirements: factory.Requirements(),
		Status:       ToolStatusRegistered,
	}
	r.toolInfo[name] = info

	r.logger.Info("tool factory registered successfully",
		"name", name,
		"capabilities", factory.GetCapabilities(),
	)

	return nil
}

// Unregister implements ToolRegistry.Unregister
func (r *DefaultToolRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Info("unregistering tool", "name", name)

	// Check if tool exists
	if _, exists := r.factories[name]; !exists {
		r.logger.Warn("attempted to unregister non-existent tool", "name", name)
		return fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}

	// Unregister from adapter if available
	if r.adapter != nil {
		if err := r.adapter.UnregisterTool(name); err != nil {
			r.logger.Error("failed to unregister tool from adapter",
				"name", name,
				"error", err,
			)
			// Continue with local unregistration even if adapter fails
		}
	}

	// Remove from all maps
	delete(r.factories, name)
	delete(r.tools, name)
	delete(r.toolInfo, name)

	r.logger.Info("tool unregistered successfully", "name", name)
	return nil
}

// Get implements ToolRegistry.Get
func (r *DefaultToolRegistry) Get(name string) (mcp.Tool, error) {
	r.mu.RLock()

	// Check if tool instance exists
	if tool, exists := r.tools[name]; exists {
		r.mu.RUnlock()
		return tool, nil
	}

	// Check if factory exists
	factory, exists := r.factories[name]
	if !exists {
		r.mu.RUnlock()
		return nil, fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}

	// Release read lock for tool creation (may take time)
	r.mu.RUnlock()

	// Create tool instance
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get tool configuration (empty for now)
	toolConfig := ToolConfig{
		Enabled:    true,
		Config:     make(map[string]interface{}),
		Timeout:    30,
		MaxRetries: 3,
	}

	tool, err := factory.Create(ctx, toolConfig)
	if err != nil {
		r.logger.Error("tool creation failed",
			"name", name,
			"error", err,
		)
		return nil, fmt.Errorf("%w: %v", ErrToolCreation, err)
	}

	// Validate created tool
	if err := r.validator.ValidateTool(tool); err != nil {
		r.logger.Error("created tool validation failed",
			"name", name,
			"error", err,
		)
		return nil, fmt.Errorf("%w: %v", ErrToolValidation, err)
	}

	// Register with adapter if available
	if r.adapter != nil {
		if err := r.adapter.RegisterTool(tool); err != nil {
			r.logger.Error("failed to register tool with adapter",
				"name", name,
				"error", err,
			)
			// Continue even if adapter registration fails - this is business logic
			// We still store the tool locally for fallback
		}
	}

	// Store tool instance (need write lock)
	r.mu.Lock()
	r.tools[name] = tool
	
	// Update status using transition logic
	if info, exists := r.toolInfo[name]; exists {
		if IsValidTransition(info.Status, ToolStatusLoaded) {
			info.Status = ToolStatusLoaded
			r.toolInfo[name] = info
		}
	}
	r.mu.Unlock()

	r.logger.Info("tool instance created and cached",
		"name", name,
	)

	return tool, nil
}

// GetFactory implements ToolRegistry.GetFactory
func (r *DefaultToolRegistry) GetFactory(name string) (ToolFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.factories[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}

	return factory, nil
}

// List implements ToolRegistry.List
func (r *DefaultToolRegistry) List() []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ToolInfo, 0, len(r.toolInfo))
	for _, info := range r.toolInfo {
		result = append(result, info)
	}

	return result
}

// LoadTools implements ToolRegistry.LoadTools
func (r *DefaultToolRegistry) LoadTools(ctx context.Context) error {
	r.mu.RLock()
	factories := make(map[string]ToolFactory)
	for name, factory := range r.factories {
		factories[name] = factory
	}
	r.mu.RUnlock()

	r.logger.Info("loading tools",
		"count", len(factories),
	)

	var errors []string
	loaded := 0

	for name, factory := range factories {
		// Get tool configuration (empty for now)
		toolConfig := ToolConfig{
			Enabled:    true,
			Config:     make(map[string]interface{}),
			Timeout:    30,
			MaxRetries: 3,
		}

		// Create tool instance
		tool, err := factory.Create(ctx, toolConfig)
		if err != nil {
			errorMsg := fmt.Sprintf("failed to create tool %s: %v", name, err)
			errors = append(errors, errorMsg)
			r.logger.Error("tool creation failed during load",
				"name", name,
				"error", err,
			)

			// Update status to error using transition logic
			r.mu.Lock()
			if info, exists := r.toolInfo[name]; exists {
				if IsValidTransition(info.Status, ToolStatusError) {
					info.Status = ToolStatusError
					r.toolInfo[name] = info
				}
			}
			r.mu.Unlock()
			continue
		}

		// Validate tool
		if err := r.validator.ValidateTool(tool); err != nil {
			errorMsg := fmt.Sprintf("tool validation failed for %s: %v", name, err)
			errors = append(errors, errorMsg)
			r.logger.Error("tool validation failed during load",
				"name", name,
				"error", err,
			)

			// Update status to error using transition logic
			r.mu.Lock()
			if info, exists := r.toolInfo[name]; exists {
				if IsValidTransition(info.Status, ToolStatusError) {
					info.Status = ToolStatusError
					r.toolInfo[name] = info
				}
			}
			r.mu.Unlock()
			continue
		}

		// Store tool
		r.mu.Lock()
		r.tools[name] = tool
		if info, exists := r.toolInfo[name]; exists {
			info.Status = ToolStatusLoaded
			r.toolInfo[name] = info
		}
		r.mu.Unlock()

		loaded++
		r.logger.Debug("tool loaded successfully",
			"name", name,
		)
	}

	r.logger.Info("tool loading completed",
		"total", len(factories),
		"loaded", loaded,
		"errors", len(errors),
	)

	if len(errors) > 0 {
		return fmt.Errorf("failed to load %d tools: %v", len(errors), errors)
	}

	return nil
}

// ValidateTools implements ToolRegistry.ValidateTools
func (r *DefaultToolRegistry) ValidateTools(ctx context.Context) error {
	r.mu.RLock()
	tools := make(map[string]mcp.Tool)
	for name, tool := range r.tools {
		tools[name] = tool
	}
	r.mu.RUnlock()

	r.logger.Info("validating tools",
		"count", len(tools),
	)

	var errors []string

	for name, tool := range tools {
		if err := r.validator.ValidateTool(tool); err != nil {
			errorMsg := fmt.Sprintf("validation failed for tool %s: %v", name, err)
			errors = append(errors, errorMsg)
			r.logger.Error("tool validation failed",
				"name", name,
				"error", err,
			)

			// Update status to error using transition logic
			r.mu.Lock()
			if info, exists := r.toolInfo[name]; exists {
				if IsValidTransition(info.Status, ToolStatusError) {
					info.Status = ToolStatusError
					r.toolInfo[name] = info
				}
			}
			r.mu.Unlock()
		} else {
			// Update status to active using transition logic
			r.mu.Lock()
			if info, exists := r.toolInfo[name]; exists {
				if IsValidTransition(info.Status, ToolStatusActive) {
					info.Status = ToolStatusActive
					r.toolInfo[name] = info
				}
			}
			r.mu.Unlock()
		}
	}

	r.logger.Info("tool validation completed",
		"total", len(tools),
		"errors", len(errors),
	)

	if len(errors) > 0 {
		return fmt.Errorf("validation failed for %d tools: %v", len(errors), errors)
	}

	return nil
}

// Start implements ToolRegistry.Start
func (r *DefaultToolRegistry) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return fmt.Errorf("registry is already running")
	}

	r.logger.Info("starting tool registry")

	// Start adapter if available
	if r.adapter != nil {
		if err := r.adapter.Start(ctx); err != nil {
			r.logger.Error("failed to start adapter", "error", err)
			return fmt.Errorf("failed to start adapter: %w", err)
		}
		r.logger.Info("adapter started successfully")
	}

	r.running = true
	r.lastCheck = time.Now()

	r.logger.Info("tool registry started successfully")
	return nil
}

// Stop implements ToolRegistry.Stop
func (r *DefaultToolRegistry) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running {
		return nil
	}

	r.logger.Info("stopping tool registry")

	// Stop adapter if available
	if r.adapter != nil {
		if err := r.adapter.Stop(ctx); err != nil {
			r.logger.Error("failed to stop adapter", "error", err)
			// Continue with registry shutdown even if adapter fails
		} else {
			r.logger.Info("adapter stopped successfully")
		}
	}

	// Clear all tools
	r.tools = make(map[string]mcp.Tool)
	
	// Update all statuses to disabled using transition logic
	for name, info := range r.toolInfo {
		if IsValidTransition(info.Status, ToolStatusDisabled) {
			info.Status = ToolStatusDisabled
			r.toolInfo[name] = info
		}
	}

	r.running = false

	r.logger.Info("tool registry stopped")
	return nil
}

// TransitionStatus implements ToolRegistry.TransitionStatus
func (r *DefaultToolRegistry) TransitionStatus(name string, newStatus ToolStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Info("transitioning tool status",
		"name", name,
		"new_status", string(newStatus),
	)

	// Check if tool exists
	info, exists := r.toolInfo[name]
	if !exists {
		r.logger.Error("attempted to transition status of non-existent tool",
			"name", name,
			"new_status", string(newStatus),
		)
		return fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}

	currentStatus := info.Status
	
	// Validate transition
	if !IsValidTransition(currentStatus, newStatus) {
		r.logger.Error("invalid status transition attempted",
			"name", name,
			"current_status", string(currentStatus),
			"new_status", string(newStatus),
		)
		return fmt.Errorf("%w: cannot transition from %s to %s", 
			ErrInvalidTransition, string(currentStatus), string(newStatus))
	}

	// Check registry state for certain transitions
	if !r.running && (newStatus == ToolStatusActive || newStatus == ToolStatusLoaded) {
		r.logger.Error("cannot activate tool when registry is not running",
			"name", name,
			"new_status", string(newStatus),
		)
		return fmt.Errorf("%w: cannot transition to %s when registry is stopped", 
			ErrRegistryNotRunning, string(newStatus))
	}

	// Update tool status
	info.Status = newStatus
	r.toolInfo[name] = info

	// Handle special transitions
	switch newStatus {
	case ToolStatusDisabled:
		// Remove from active tools map
		if _, exists := r.tools[name]; exists {
			delete(r.tools, name)
			r.logger.Debug("removed tool instance for disabled tool", "name", name)
		}
	case ToolStatusError:
		// Remove from active tools map but keep factory
		if _, exists := r.tools[name]; exists {
			delete(r.tools, name)
			r.logger.Debug("removed tool instance for error tool", "name", name)
		}
	}

	r.logger.Info("tool status transition completed successfully",
		"name", name,
		"previous_status", string(currentStatus),
		"new_status", string(newStatus),
	)

	return nil
}

// Restart validation functions

func (r *DefaultToolRegistry) validateRestartPreconditions(name string) (ToolInfo, ToolFactory, error) {
	info, exists := r.toolInfo[name]
	if !exists {
		r.logger.Error("attempted to restart non-existent tool", "name", name)
		return ToolInfo{}, nil, fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}

	if !IsValidTransition(info.Status, ToolStatusRegistered) {
		r.logger.Error("tool restart not allowed from current status",
			"name", name, "current_status", string(info.Status))
		return ToolInfo{}, nil, fmt.Errorf("%w: cannot restart tool from status %s", 
			ErrRestartNotAllowed, string(info.Status))
	}

	if !r.running {
		r.logger.Error("cannot restart tool when registry is not running", "name", name)
		return ToolInfo{}, nil, fmt.Errorf("%w: cannot restart tool when registry is stopped", 
			ErrRegistryNotRunning)
	}

	factory, exists := r.factories[name]
	if !exists {
		r.logger.Error("tool factory not found for restart", "name", name)
		return ToolInfo{}, nil, fmt.Errorf("%w: factory not found for tool %s", ErrToolNotFound, name)
	}

	return info, factory, nil
}

// Tool management functions

func (r *DefaultToolRegistry) cleanupExistingTool(name string) {
	if _, exists := r.tools[name]; exists {
		delete(r.tools, name)
		r.logger.Debug("removed existing tool instance for restart", "name", name)
	}
}

func (r *DefaultToolRegistry) createToolInstance(ctx context.Context, factory ToolFactory) (mcp.Tool, error) {
	toolConfig := ToolConfig{
		Enabled:    true,
		Config:     make(map[string]interface{}),
		Timeout:    30,
		MaxRetries: 3,
	}
	return factory.Create(ctx, toolConfig)
}

func (r *DefaultToolRegistry) registerToolWithAdapter(tool mcp.Tool, name string) {
	if r.adapter != nil {
		if err := r.adapter.RegisterTool(tool); err != nil {
			r.logger.Error("failed to register restarted tool with adapter",
				"name", name, "error", err)
		}
	}
}

// Status management functions

func (r *DefaultToolRegistry) transitionToRegistered(name string) error {
	if info, exists := r.toolInfo[name]; exists {
		info.Status = ToolStatusRegistered
		r.toolInfo[name] = info
		r.logger.Info("tool transitioned to registered status for restart", "name", name)
	}
	return nil
}

func (r *DefaultToolRegistry) transitionToLoaded(name string) error {
	if info, exists := r.toolInfo[name]; exists && IsValidTransition(info.Status, ToolStatusLoaded) {
		info.Status = ToolStatusLoaded
		r.toolInfo[name] = info
	}
	return nil
}

func (r *DefaultToolRegistry) transitionToError(name string) error {
	if info, exists := r.toolInfo[name]; exists && IsValidTransition(info.Status, ToolStatusError) {
		info.Status = ToolStatusError
		r.toolInfo[name] = info
	}
	return nil
}

// Core restart orchestration

func (r *DefaultToolRegistry) restartToolCore(ctx context.Context, name string, factory ToolFactory) error {
	r.cleanupExistingTool(name)
	
	if err := r.transitionToRegistered(name); err != nil {
		return err
	}

	tool, err := r.createToolInstance(ctx, factory)
	if err != nil {
		r.logger.Error("tool recreation failed during restart", "name", name, "error", err)
		r.transitionToError(name)
		return fmt.Errorf("%w: failed to recreate tool %s: %v", ErrToolRestart, name, err)
	}

	if err := r.validator.ValidateTool(tool); err != nil {
		r.logger.Error("tool validation failed during restart", "name", name, "error", err)
		r.transitionToError(name)
		return fmt.Errorf("%w: tool validation failed for %s: %v", ErrToolRestart, name, err)
	}

	r.registerToolWithAdapter(tool, name)
	r.tools[name] = tool
	
	return r.transitionToLoaded(name)
}

// RestartTool implements ToolRegistry.RestartTool
func (r *DefaultToolRegistry) RestartTool(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Info("restarting tool", "name", name)

	info, factory, err := r.validateRestartPreconditions(name)
	if err != nil {
		return err
	}

	if err := r.restartToolCore(ctx, name, factory); err != nil {
		return err
	}

	r.logger.Info("tool restart completed successfully",
		"name", name, "status", string(info.Status))

	return nil
}

// Health implements ToolRegistry.Health
func (r *DefaultToolRegistry) Health() RegistryHealth {
	r.mu.RLock()
	defer r.mu.RUnlock()

	health := RegistryHealth{
		Status:       "healthy",
		ToolCount:    len(r.toolInfo),
		ActiveTools:  0,
		ErrorTools:   0,
		LastCheck:    r.lastCheck.Format(time.RFC3339),
		Errors:       []string{},
		ToolStatuses: make(map[string]string),
	}

	// Count tool statuses
	for name, info := range r.toolInfo {
		health.ToolStatuses[name] = string(info.Status)
		
		switch info.Status {
		case ToolStatusActive:
			health.ActiveTools++
		case ToolStatusError:
			health.ErrorTools++
		}
	}

	// Check adapter health if available
	if r.adapter != nil {
		adapterHealth := r.adapter.Health()
		if adapterHealth.Status != "healthy" {
			health.Status = "degraded"
			health.Errors = append(health.Errors, 
				fmt.Sprintf("adapter status: %s", adapterHealth.Status))
		}
	}

	// Determine overall status
	if !r.running {
		health.Status = "stopped"
	} else if health.ErrorTools > 0 {
		health.Status = "degraded"
	}

	// Add error details if any
	if health.ErrorTools > 0 {
		health.Errors = append(health.Errors, 
			fmt.Sprintf("%d tools in error state", health.ErrorTools))
	}

	return health
}