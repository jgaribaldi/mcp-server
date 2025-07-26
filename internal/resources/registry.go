package resources

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
	"mcp-server/internal/registry"
)

type DefaultResourceRegistry struct {
	factories        map[string]ResourceFactory
	circuitFactories map[string]*registry.CircuitBreakerFactory[mcp.Resource]
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

func NewDefaultResourceRegistry(cfg *config.Config, log *logger.Logger) ResourceRegistry {
	return &DefaultResourceRegistry{
		factories:        make(map[string]ResourceFactory),
		circuitFactories: make(map[string]*registry.CircuitBreakerFactory[mcp.Resource]),
		resources:        make(map[string]mcp.Resource),
		resourceInfo:     make(map[string]ResourceInfo),
		cache:           make(map[string]CachedContent),
		logger:          log,
		config:          cfg,
		validator:       NewResourceValidator(cfg, log),
	}
}

func (r *DefaultResourceRegistry) validateRegistrationRequest(uri string, factory ResourceFactory) error {
	if err := r.validator.ValidateURI(uri); err != nil {
		r.logger.Error("resource URI validation failed",
			"uri", uri,
			"error", err,
		)
		return fmt.Errorf("%w: %v", ErrInvalidResourceURI, err)
	}

	if _, exists := r.factories[uri]; exists {
		r.logger.Error("resource already registered",
			"uri", uri,
		)
		return fmt.Errorf("%w: %s", ErrResourceAlreadyExists, uri)
	}

	if err := r.validator.ValidateFactory(factory); err != nil {
		r.logger.Error("resource factory validation failed",
			"uri", uri,
			"error", err,
		)
		return fmt.Errorf("%w: %v", ErrResourceValidation, err)
	}

	return nil
}

func (r *DefaultResourceRegistry) createCircuitBreakerFactory(uri string) *registry.CircuitBreakerFactory[mcp.Resource] {
	circuitConfig := registry.DefaultCircuitBreakerConfig()
	return registry.NewCircuitBreakerFactory[mcp.Resource](uri, circuitConfig)
}

func (r *DefaultResourceRegistry) storeFactoryInfo(uri string, factory ResourceFactory, circuitFactory *registry.CircuitBreakerFactory[mcp.Resource]) {
	r.factories[uri] = factory
	r.circuitFactories[uri] = circuitFactory

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
}

func (r *DefaultResourceRegistry) Register(uri string, factory ResourceFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Info("registering resource factory",
		"uri", uri,
		"name", factory.Name(),
		"description", factory.Description(),
		"version", factory.Version(),
	)

	if err := r.validateRegistrationRequest(uri, factory); err != nil {
		return err
	}

	circuitFactory := r.createCircuitBreakerFactory(uri)
	r.storeFactoryInfo(uri, factory, circuitFactory)

	r.logger.Info("resource factory registered successfully",
		"uri", uri,
		"capabilities", factory.Capabilities(),
	)

	return nil
}

func (r *DefaultResourceRegistry) Unregister(uri string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Info("unregistering resource", "uri", uri)

	if _, exists := r.factories[uri]; !exists {
		r.logger.Warn("attempted to unregister non-existent resource", "uri", uri)
		return fmt.Errorf("%w: %s", ErrResourceNotFound, uri)
	}

	delete(r.factories, uri)
	delete(r.circuitFactories, uri)
	delete(r.resources, uri)
	delete(r.resourceInfo, uri)

	r.cacheMu.Lock()
	delete(r.cache, uri)
	r.cacheMu.Unlock()

	r.logger.Info("resource unregistered successfully", "uri", uri)
	return nil
}

func (r *DefaultResourceRegistry) findExistingResource(uri string) (mcp.Resource, bool) {
	if resource, exists := r.resources[uri]; exists {
		return resource, true
	}
	return nil, false
}

func (r *DefaultResourceRegistry) createResourceInstance(ctx context.Context, factory ResourceFactory) (mcp.Resource, error) {
	resourceConfig := ResourceConfig{
		Enabled:       true,
		Config:        make(map[string]interface{}),
		CacheTimeout:  300,
		AccessControl: make(map[string]string),
	}

	return factory.Create(ctx, resourceConfig)
}

func (r *DefaultResourceRegistry) validateAndStoreResource(uri string, resource mcp.Resource) error {
	if err := r.validator.ValidateResource(resource); err != nil {
		r.logger.Error("created resource validation failed",
			"uri", uri,
			"error", err,
		)
		return fmt.Errorf("%w: %v", ErrResourceValidation, err)
	}

	r.mu.Lock()
	r.resources[uri] = resource
	
	if info, exists := r.resourceInfo[uri]; exists {
		if IsValidTransition(info.Status, ResourceStatusLoaded) {
			info.Status = ResourceStatusLoaded
			r.resourceInfo[uri] = info
		}
	}
	r.mu.Unlock()

	return nil
}

func (r *DefaultResourceRegistry) Get(uri string) (mcp.Resource, error) {
	r.mu.RLock()

	if resource, exists := r.findExistingResource(uri); exists {
		r.mu.RUnlock()
		return resource, nil
	}

	factory, exists := r.factories[uri]
	if !exists {
		r.mu.RUnlock()
		return nil, fmt.Errorf("%w: %s", ErrResourceNotFound, uri)
	}

	circuitFactory, circuitExists := r.circuitFactories[uri]
	if !circuitExists {
		r.mu.RUnlock()
		return nil, fmt.Errorf("%w: circuit breaker not found for %s", ErrResourceNotFound, uri)
	}

	r.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use circuit breaker to protect resource creation
	resource, err := circuitFactory.ExecuteWithContext(ctx, func(ctx context.Context) (mcp.Resource, error) {
		return r.createResourceInstance(ctx, factory)
	})
	if err != nil {
		r.logger.Error("resource creation failed",
			"uri", uri,
			"error", err,
		)
		return nil, fmt.Errorf("%w: %v", ErrResourceCreation, err)
	}

	if err := r.validateAndStoreResource(uri, resource); err != nil {
		return nil, err
	}

	r.logger.Info("resource instance created and cached",
		"uri", uri,
	)

	return resource, nil
}

func (r *DefaultResourceRegistry) GetFactory(uri string) (ResourceFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.factories[uri]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrResourceNotFound, uri)
	}

	return factory, nil
}

func (r *DefaultResourceRegistry) List() []ResourceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ResourceInfo, 0, len(r.resourceInfo))
	for _, info := range r.resourceInfo {
		result = append(result, info)
	}

	return result
}

func (r *DefaultResourceRegistry) loadSingleResource(ctx context.Context, uri string, factory ResourceFactory) error {
	resource, err := r.createResourceInstance(ctx, factory)
	if err != nil {
		r.handleResourceLoadError(uri, fmt.Sprintf("failed to create resource %s: %v", uri, err), err)
		return err
	}

	if err := r.validator.ValidateResource(resource); err != nil {
		r.handleResourceLoadError(uri, fmt.Sprintf("resource validation failed for %s: %v", uri, err), err)
		return err
	}

	r.validateAndStoreBulkResource(uri, resource)
	r.logger.Debug("resource loaded successfully", "uri", uri)
	return nil
}

func (r *DefaultResourceRegistry) handleResourceLoadError(uri string, errorMsg string, err error) {
	r.logger.Error("resource creation failed during load",
		"uri", uri,
		"error", err,
	)

	r.mu.Lock()
	if info, exists := r.resourceInfo[uri]; exists {
		if IsValidTransition(info.Status, ResourceStatusError) {
			info.Status = ResourceStatusError
			r.resourceInfo[uri] = info
		}
	}
	r.mu.Unlock()
}

func (r *DefaultResourceRegistry) validateAndStoreBulkResource(uri string, resource mcp.Resource) {
	r.mu.Lock()
	r.resources[uri] = resource
	if info, exists := r.resourceInfo[uri]; exists {
		info.Status = ResourceStatusLoaded
		r.resourceInfo[uri] = info
	}
	r.mu.Unlock()
}

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
		if err := r.loadSingleResource(ctx, uri, factory); err != nil {
			errors = append(errors, err.Error())
		} else {
			loaded++
		}
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

func (r *DefaultResourceRegistry) validateSingleResource(uri string, resource mcp.Resource) error {
	if err := r.validator.ValidateResource(resource); err != nil {
		r.logger.Error("resource validation failed",
			"uri", uri,
			"error", err,
		)

		r.mu.Lock()
		if info, exists := r.resourceInfo[uri]; exists {
			if IsValidTransition(info.Status, ResourceStatusError) {
				info.Status = ResourceStatusError
				r.resourceInfo[uri] = info
			}
		}
		r.mu.Unlock()
		
		return fmt.Errorf("validation failed for resource %s: %v", uri, err)
	}

	r.mu.Lock()
	if info, exists := r.resourceInfo[uri]; exists {
		if IsValidTransition(info.Status, ResourceStatusActive) {
			info.Status = ResourceStatusActive
			r.resourceInfo[uri] = info
		}
	}
	r.mu.Unlock()

	return nil
}

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
		if err := r.validateSingleResource(uri, resource); err != nil {
			errors = append(errors, err.Error())
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

func (r *DefaultResourceRegistry) validateStatusTransition(uri string, currentStatus, newStatus ResourceStatus) error {
	if !IsValidTransition(currentStatus, newStatus) {
		r.logger.Error("invalid status transition attempted",
			"uri", uri,
			"current_status", string(currentStatus),
			"new_status", string(newStatus),
		)
		return fmt.Errorf("%w: cannot transition from %s to %s", 
			ErrInvalidTransition, string(currentStatus), string(newStatus))
	}

	if !r.running && (newStatus == ResourceStatusActive || newStatus == ResourceStatusLoaded) {
		r.logger.Error("cannot activate resource when registry is not running",
			"uri", uri,
			"new_status", string(newStatus),
		)
		return fmt.Errorf("%w: cannot transition to %s when registry is stopped", 
			ErrRegistryNotRunning, string(newStatus))
	}

	return nil
}

func (r *DefaultResourceRegistry) updateResourceStatus(uri string, newStatus ResourceStatus) {
	if info, exists := r.resourceInfo[uri]; exists {
		info.Status = newStatus
		r.resourceInfo[uri] = info
	}
}

func (r *DefaultResourceRegistry) handleStatusTransitionCleanup(uri string, newStatus ResourceStatus) {
	switch newStatus {
	case ResourceStatusDisabled:
		if _, exists := r.resources[uri]; exists {
			delete(r.resources, uri)
			r.logger.Debug("removed resource instance for disabled resource", "uri", uri)
		}
		r.cacheMu.Lock()
		delete(r.cache, uri)
		r.cacheMu.Unlock()
	case ResourceStatusError:
		if _, exists := r.resources[uri]; exists {
			delete(r.resources, uri)
			r.logger.Debug("removed resource instance for error resource", "uri", uri)
		}
		r.cacheMu.Lock()
		delete(r.cache, uri)
		r.cacheMu.Unlock()
	}
}

func (r *DefaultResourceRegistry) TransitionStatus(uri string, newStatus ResourceStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Info("transitioning resource status",
		"uri", uri,
		"new_status", string(newStatus),
	)

	info, exists := r.resourceInfo[uri]
	if !exists {
		r.logger.Error("attempted to transition status of non-existent resource",
			"uri", uri,
			"new_status", string(newStatus),
		)
		return fmt.Errorf("%w: %s", ErrResourceNotFound, uri)
	}

	currentStatus := info.Status
	
	if err := r.validateStatusTransition(uri, currentStatus, newStatus); err != nil {
		return err
	}

	r.updateResourceStatus(uri, newStatus)
	r.handleStatusTransitionCleanup(uri, newStatus)

	r.logger.Info("resource status transition completed successfully",
		"uri", uri,
		"previous_status", string(currentStatus),
		"new_status", string(newStatus),
	)

	return nil
}

func (r *DefaultResourceRegistry) validateRefreshRequest(uri string, info ResourceInfo) error {
	if info.Status != ResourceStatusActive && info.Status != ResourceStatusLoaded {
		r.logger.Error("resource refresh not allowed from current status",
			"uri", uri, "current_status", string(info.Status))
		return fmt.Errorf("%w: cannot refresh resource from status %s", 
			ErrRefreshNotAllowed, string(info.Status))
	}
	return nil
}

func (r *DefaultResourceRegistry) recreateResourceInstance(ctx context.Context, uri string, factory ResourceFactory) (mcp.Resource, error) {
	resource, err := r.createResourceInstance(ctx, factory)
	if err != nil {
		r.logger.Error("resource recreation failed during refresh", "uri", uri, "error", err)
		r.TransitionStatus(uri, ResourceStatusError)
		return nil, fmt.Errorf("%w: failed to recreate resource %s: %v", ErrResourceRefresh, uri, err)
	}

	if err := r.validator.ValidateResource(resource); err != nil {
		r.logger.Error("resource validation failed during refresh", "uri", uri, "error", err)
		r.TransitionStatus(uri, ResourceStatusError)
		return nil, fmt.Errorf("%w: resource validation failed for %s: %v", ErrResourceRefresh, uri, err)
	}

	return resource, nil
}

func (r *DefaultResourceRegistry) updateResourceAndCache(uri string, resource mcp.Resource) {
	r.mu.Lock()
	r.resources[uri] = resource
	r.mu.Unlock()

	r.cacheMu.Lock()
	delete(r.cache, uri)
	r.cacheMu.Unlock()
}

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

	if err := r.validateRefreshRequest(uri, info); err != nil {
		return err
	}

	resource, err := r.recreateResourceInstance(ctx, uri, factory)
	if err != nil {
		return err
	}

	r.updateResourceAndCache(uri, resource)

	r.logger.Info("resource refresh completed successfully", "uri", uri)
	return nil
}

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

func (r *DefaultResourceRegistry) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running {
		return nil
	}

	r.logger.Info("stopping resource registry")

	r.resources = make(map[string]mcp.Resource)
	r.cacheMu.Lock()
	r.cache = make(map[string]CachedContent)
	r.cacheMu.Unlock()
	
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

	for uri, info := range r.resourceInfo {
		health.ResourceStatuses[uri] = string(info.Status)
		
		switch info.Status {
		case ResourceStatusActive:
			health.ActiveResources++
		case ResourceStatusError:
			health.ErrorResources++
		}
	}

	for uri, cb := range r.circuitFactories {
		health.CircuitBreakers[uri] = cb.Status()
	}

	if !r.running {
		health.Status = "stopped"
	} else if health.ErrorResources > 0 {
		health.Status = "degraded"
	}

	if health.ErrorResources > 0 {
		health.Errors = append(health.Errors, 
			fmt.Sprintf("%d resources in error state", health.ErrorResources))
	}

	return health
}