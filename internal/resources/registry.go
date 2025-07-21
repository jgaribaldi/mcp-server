package resources

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
)

// DefaultResourceRegistry implements ResourceRegistry
type DefaultResourceRegistry struct {
	factories        map[string]ResourceFactory
	circuitFactories map[string]*CircuitBreakerResourceFactory
	resources        map[string]mcp.Resource
	resourceInfo     map[string]ResourceInfo
	cache           map[string]CachedContent
	logger          *logger.Logger
	config          *config.Config
	validator       *ResourceValidator
	running         bool
	startTime       time.Time
	lastCheck       time.Time
	cacheHits       int64
	cacheMisses     int64
	mu              sync.RWMutex
	cacheMu         sync.RWMutex
}

// NewDefaultResourceRegistry creates a new resource registry instance
func NewDefaultResourceRegistry(cfg *config.Config, log *logger.Logger) ResourceRegistry {
	return &DefaultResourceRegistry{
		factories:        make(map[string]ResourceFactory),
		circuitFactories: make(map[string]*CircuitBreakerResourceFactory),
		resources:        make(map[string]mcp.Resource),
		resourceInfo:     make(map[string]ResourceInfo),
		cache:           make(map[string]CachedContent),
		logger:          log,
		config:          cfg,
		validator:       NewResourceValidator(cfg, log),
	}
}

// Register implements ResourceRegistry.Register
func (r *DefaultResourceRegistry) Register(uri string, factory ResourceFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Info("registering resource factory",
		"uri", uri,
		"name", factory.Name(),
		"description", factory.Description(),
		"version", factory.Version(),
	)

	// Validate resource URI
	if err := r.validator.ValidateURI(uri); err != nil {
		r.logger.Error("resource URI validation failed",
			"uri", uri,
			"error", err,
		)
		return fmt.Errorf("%w: %v", ErrInvalidResourceURI, err)
	}

	// Check for duplicate registration
	if _, exists := r.factories[uri]; exists {
		r.logger.Error("resource already registered",
			"uri", uri,
		)
		return fmt.Errorf("%w: %s", ErrResourceAlreadyExists, uri)
	}

	// Validate factory
	if err := r.validator.ValidateFactory(factory); err != nil {
		r.logger.Error("resource factory validation failed",
			"uri", uri,
			"error", err,
		)
		return fmt.Errorf("%w: %v", ErrResourceValidation, err)
	}

	// Register factory
	r.factories[uri] = factory

	// Create circuit breaker wrapper
	circuitConfig := DefaultCircuitBreakerConfig()
	circuitFactory := NewCircuitBreakerResourceFactory(factory, circuitConfig)
	r.circuitFactories[uri] = circuitFactory

	// Create resource info
	info := ResourceInfo{
		URI:          factory.URI(),
		Name:         factory.Name(),
		Description:  factory.Description(),
		MimeType:     factory.MimeType(),
		Version:      factory.Version(),
		Tags:         factory.Tags(),
		Capabilities: factory.Capabilities(),
		Status:       ResourceStatusRegistered,
		Metadata:     make(map[string]string),
	}
	r.resourceInfo[uri] = info

	r.logger.Info("resource factory registered successfully",
		"uri", uri,
		"capabilities", factory.Capabilities(),
	)

	return nil
}

// Unregister implements ResourceRegistry.Unregister
func (r *DefaultResourceRegistry) Unregister(uri string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Info("unregistering resource", "uri", uri)

	// Check if resource exists
	if _, exists := r.factories[uri]; !exists {
		r.logger.Warn("attempted to unregister non-existent resource", "uri", uri)
		return fmt.Errorf("%w: %s", ErrResourceNotFound, uri)
	}

	// Remove from all maps
	delete(r.factories, uri)
	delete(r.circuitFactories, uri)
	delete(r.resources, uri)
	delete(r.resourceInfo, uri)

	// Clear cache
	r.cacheMu.Lock()
	delete(r.cache, uri)
	r.cacheMu.Unlock()

	r.logger.Info("resource unregistered successfully", "uri", uri)
	return nil
}

// Get implements ResourceRegistry.Get
func (r *DefaultResourceRegistry) Get(uri string) (mcp.Resource, error) {
	r.mu.RLock()

	// Check if resource instance exists
	if resource, exists := r.resources[uri]; exists {
		r.mu.RUnlock()
		return resource, nil
	}

	// Check if factory exists
	factory, exists := r.factories[uri]
	if !exists {
		r.mu.RUnlock()
		return nil, fmt.Errorf("%w: %s", ErrResourceNotFound, uri)
	}

	// Release read lock for resource creation
	r.mu.RUnlock()

	// Create resource instance
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get resource configuration
	resourceConfig := ResourceConfig{
		Enabled:       true,
		Config:        make(map[string]interface{}),
		CacheTimeout:  300, // 5 minutes default
		AccessControl: make(map[string]string),
	}

	resource, err := factory.Create(ctx, resourceConfig)
	if err != nil {
		r.logger.Error("resource creation failed",
			"uri", uri,
			"error", err,
		)
		return nil, fmt.Errorf("%w: %v", ErrResourceCreation, err)
	}

	// Validate created resource
	if err := r.validator.ValidateResource(resource); err != nil {
		r.logger.Error("created resource validation failed",
			"uri", uri,
			"error", err,
		)
		return nil, fmt.Errorf("%w: %v", ErrResourceValidation, err)
	}

	// Store resource instance
	r.mu.Lock()
	r.resources[uri] = resource
	
	// Update status
	if info, exists := r.resourceInfo[uri]; exists {
		if IsValidTransition(info.Status, ResourceStatusLoaded) {
			info.Status = ResourceStatusLoaded
			r.resourceInfo[uri] = info
		}
	}
	r.mu.Unlock()

	r.logger.Info("resource instance created and cached",
		"uri", uri,
	)

	return resource, nil
}

// GetFactory implements ResourceRegistry.GetFactory
func (r *DefaultResourceRegistry) GetFactory(uri string) (ResourceFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.factories[uri]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrResourceNotFound, uri)
	}

	return factory, nil
}

// List implements ResourceRegistry.List
func (r *DefaultResourceRegistry) List() []ResourceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ResourceInfo, 0, len(r.resourceInfo))
	for _, info := range r.resourceInfo {
		result = append(result, info)
	}

	return result
}

// LoadResources implements ResourceRegistry.LoadResources
func (r *DefaultResourceRegistry) LoadResources(ctx context.Context) error {
	r.mu.RLock()
	factories := make(map[string]ResourceFactory)
	for uri, factory := range r.factories {
		factories[uri] = factory
	}
	r.mu.RUnlock()

	r.logger.Info("loading resources",
		"count", len(factories),
	)

	var errors []string
	loaded := 0

	for uri, factory := range factories {
		// Get resource configuration
		resourceConfig := ResourceConfig{
			Enabled:       true,
			Config:        make(map[string]interface{}),
			CacheTimeout:  300,
			AccessControl: make(map[string]string),
		}

		// Create resource instance
		resource, err := factory.Create(ctx, resourceConfig)
		if err != nil {
			errorMsg := fmt.Sprintf("failed to create resource %s: %v", uri, err)
			errors = append(errors, errorMsg)
			r.logger.Error("resource creation failed during load",
				"uri", uri,
				"error", err,
			)

			// Update status to error
			r.mu.Lock()
			if info, exists := r.resourceInfo[uri]; exists {
				if IsValidTransition(info.Status, ResourceStatusError) {
					info.Status = ResourceStatusError
					r.resourceInfo[uri] = info
				}
			}
			r.mu.Unlock()
			continue
		}

		// Validate resource
		if err := r.validator.ValidateResource(resource); err != nil {
			errorMsg := fmt.Sprintf("resource validation failed for %s: %v", uri, err)
			errors = append(errors, errorMsg)
			r.logger.Error("resource validation failed during load",
				"uri", uri,
				"error", err,
			)

			// Update status to error
			r.mu.Lock()
			if info, exists := r.resourceInfo[uri]; exists {
				if IsValidTransition(info.Status, ResourceStatusError) {
					info.Status = ResourceStatusError
					r.resourceInfo[uri] = info
				}
			}
			r.mu.Unlock()
			continue
		}

		// Store resource
		r.mu.Lock()
		r.resources[uri] = resource
		if info, exists := r.resourceInfo[uri]; exists {
			info.Status = ResourceStatusLoaded
			r.resourceInfo[uri] = info
		}
		r.mu.Unlock()

		loaded++
		r.logger.Debug("resource loaded successfully",
			"uri", uri,
		)
	}

	r.logger.Info("resource loading completed",
		"total", len(factories),
		"loaded", loaded,
		"errors", len(errors),
	)

	if len(errors) > 0 {
		return fmt.Errorf("failed to load %d resources: %v", len(errors), errors)
	}

	return nil
}

// ValidateResources implements ResourceRegistry.ValidateResources
func (r *DefaultResourceRegistry) ValidateResources(ctx context.Context) error {
	r.mu.RLock()
	resources := make(map[string]mcp.Resource)
	for uri, resource := range r.resources {
		resources[uri] = resource
	}
	r.mu.RUnlock()

	r.logger.Info("validating resources",
		"count", len(resources),
	)

	var errors []string

	for uri, resource := range resources {
		if err := r.validator.ValidateResource(resource); err != nil {
			errorMsg := fmt.Sprintf("validation failed for resource %s: %v", uri, err)
			errors = append(errors, errorMsg)
			r.logger.Error("resource validation failed",
				"uri", uri,
				"error", err,
			)

			// Update status to error
			r.mu.Lock()
			if info, exists := r.resourceInfo[uri]; exists {
				if IsValidTransition(info.Status, ResourceStatusError) {
					info.Status = ResourceStatusError
					r.resourceInfo[uri] = info
				}
			}
			r.mu.Unlock()
		} else {
			// Update status to active
			r.mu.Lock()
			if info, exists := r.resourceInfo[uri]; exists {
				if IsValidTransition(info.Status, ResourceStatusActive) {
					info.Status = ResourceStatusActive
					r.resourceInfo[uri] = info
				}
			}
			r.mu.Unlock()
		}
	}

	r.logger.Info("resource validation completed",
		"total", len(resources),
		"errors", len(errors),
	)

	if len(errors) > 0 {
		return fmt.Errorf("validation failed for %d resources: %v", len(errors), errors)
	}

	return nil
}

// TransitionStatus implements ResourceRegistry.TransitionStatus
func (r *DefaultResourceRegistry) TransitionStatus(uri string, newStatus ResourceStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Info("transitioning resource status",
		"uri", uri,
		"new_status", string(newStatus),
	)

	// Check if resource exists
	info, exists := r.resourceInfo[uri]
	if !exists {
		r.logger.Error("attempted to transition status of non-existent resource",
			"uri", uri,
			"new_status", string(newStatus),
		)
		return fmt.Errorf("%w: %s", ErrResourceNotFound, uri)
	}

	currentStatus := info.Status
	
	// Validate transition
	if !IsValidTransition(currentStatus, newStatus) {
		r.logger.Error("invalid status transition attempted",
			"uri", uri,
			"current_status", string(currentStatus),
			"new_status", string(newStatus),
		)
		return fmt.Errorf("%w: cannot transition from %s to %s", 
			ErrInvalidTransition, string(currentStatus), string(newStatus))
	}

	// Check registry state for certain transitions
	if !r.running && (newStatus == ResourceStatusActive || newStatus == ResourceStatusLoaded) {
		r.logger.Error("cannot activate resource when registry is not running",
			"uri", uri,
			"new_status", string(newStatus),
		)
		return fmt.Errorf("%w: cannot transition to %s when registry is stopped", 
			ErrRegistryNotRunning, string(newStatus))
	}

	// Update resource status
	info.Status = newStatus
	r.resourceInfo[uri] = info

	// Handle special transitions
	switch newStatus {
	case ResourceStatusDisabled:
		// Remove from active resources map
		if _, exists := r.resources[uri]; exists {
			delete(r.resources, uri)
			r.logger.Debug("removed resource instance for disabled resource", "uri", uri)
		}
		// Clear cache
		r.cacheMu.Lock()
		delete(r.cache, uri)
		r.cacheMu.Unlock()
	case ResourceStatusError:
		// Remove from active resources map but keep factory
		if _, exists := r.resources[uri]; exists {
			delete(r.resources, uri)
			r.logger.Debug("removed resource instance for error resource", "uri", uri)
		}
		// Clear cache
		r.cacheMu.Lock()
		delete(r.cache, uri)
		r.cacheMu.Unlock()
	}

	r.logger.Info("resource status transition completed successfully",
		"uri", uri,
		"previous_status", string(currentStatus),
		"new_status", string(newStatus),
	)

	return nil
}

// RefreshResource implements ResourceRegistry.RefreshResource
func (r *DefaultResourceRegistry) RefreshResource(ctx context.Context, uri string) error {
	r.mu.RLock()
	factory, exists := r.factories[uri]
	if !exists {
		r.mu.RUnlock()
		return fmt.Errorf("%w: %s", ErrResourceNotFound, uri)
	}
	
	info, infoExists := r.resourceInfo[uri]
	if !infoExists {
		r.mu.RUnlock()
		return fmt.Errorf("%w: %s", ErrResourceNotFound, uri)
	}
	r.mu.RUnlock()

	r.logger.Info("refreshing resource", "uri", uri)

	// Check if refresh is allowed from current status
	if info.Status != ResourceStatusActive && info.Status != ResourceStatusLoaded {
		r.logger.Error("resource refresh not allowed from current status",
			"uri", uri, "current_status", string(info.Status))
		return fmt.Errorf("%w: cannot refresh resource from status %s", 
			ErrRefreshNotAllowed, string(info.Status))
	}

	// Create new resource instance
	resourceConfig := ResourceConfig{
		Enabled:       true,
		Config:        make(map[string]interface{}),
		CacheTimeout:  300,
		AccessControl: make(map[string]string),
	}

	resource, err := factory.Create(ctx, resourceConfig)
	if err != nil {
		r.logger.Error("resource recreation failed during refresh", "uri", uri, "error", err)
		r.TransitionStatus(uri, ResourceStatusError)
		return fmt.Errorf("%w: failed to recreate resource %s: %v", ErrResourceRefresh, uri, err)
	}

	if err := r.validator.ValidateResource(resource); err != nil {
		r.logger.Error("resource validation failed during refresh", "uri", uri, "error", err)
		r.TransitionStatus(uri, ResourceStatusError)
		return fmt.Errorf("%w: resource validation failed for %s: %v", ErrResourceRefresh, uri, err)
	}

	// Update resource and clear cache
	r.mu.Lock()
	r.resources[uri] = resource
	r.mu.Unlock()

	r.cacheMu.Lock()
	delete(r.cache, uri)
	r.cacheMu.Unlock()

	r.logger.Info("resource refresh completed successfully", "uri", uri)
	return nil
}

// Start implements ResourceRegistry.Start
func (r *DefaultResourceRegistry) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return fmt.Errorf("registry is already running")
	}

	r.logger.Info("starting resource registry")

	r.running = true
	r.startTime = time.Now()
	r.lastCheck = time.Now()

	r.logger.Info("resource registry started successfully")
	return nil
}

// Stop implements ResourceRegistry.Stop
func (r *DefaultResourceRegistry) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running {
		return nil
	}

	r.logger.Info("stopping resource registry")

	// Clear all resources and cache
	r.resources = make(map[string]mcp.Resource)
	r.cacheMu.Lock()
	r.cache = make(map[string]CachedContent)
	r.cacheMu.Unlock()
	
	// Update all statuses to disabled
	for uri, info := range r.resourceInfo {
		if IsValidTransition(info.Status, ResourceStatusDisabled) {
			info.Status = ResourceStatusDisabled
			r.resourceInfo[uri] = info
		}
	}

	r.running = false

	r.logger.Info("resource registry stopped")
	return nil
}

// Health implements ResourceRegistry.Health
func (r *DefaultResourceRegistry) Health() RegistryHealth {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.cacheMu.RLock()
	cacheSize := len(r.cache)
	totalRequests := r.cacheHits + r.cacheMisses
	var hitRate float64
	if totalRequests > 0 {
		hitRate = float64(r.cacheHits) / float64(totalRequests) * 100.0
	}
	r.cacheMu.RUnlock()

	health := RegistryHealth{
		Status:            "healthy",
		ResourceCount:     len(r.resourceInfo),
		ActiveResources:   0,
		ErrorResources:    0,
		CachedResources:   cacheSize,
		CacheHitRate:      hitRate,
		LastCheck:         r.lastCheck.Format(time.RFC3339),
		Errors:            []string{},
		ResourceStatuses:  make(map[string]string),
		CircuitBreakers:   make(map[string]string),
	}

	// Count resource statuses
	for uri, info := range r.resourceInfo {
		health.ResourceStatuses[uri] = string(info.Status)
		
		switch info.Status {
		case ResourceStatusActive:
			health.ActiveResources++
		case ResourceStatusError:
			health.ErrorResources++
		}
	}

	// Add circuit breaker statuses
	for uri, cb := range r.circuitFactories {
		health.CircuitBreakers[uri] = cb.Status()
	}

	// Determine overall status
	if !r.running {
		health.Status = "stopped"
	} else if health.ErrorResources > 0 {
		health.Status = "degraded"
	}

	// Add error details if any
	if health.ErrorResources > 0 {
		health.Errors = append(health.Errors, 
			fmt.Sprintf("%d resources in error state", health.ErrorResources))
	}

	return health
}