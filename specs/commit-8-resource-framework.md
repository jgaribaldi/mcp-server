# Commit 8: Resource Registration Framework

## Overview
This specification defines the implementation of the resource registration framework for the MCP server. The framework enables the server to manage MCP resources that can be accessed by clients, providing a structured approach to resource registration, lifecycle management, and access control.

## Background and Context
Resources in the MCP protocol represent data sources that clients can access, such as files, database queries, API endpoints, or configuration data. Unlike tools which perform actions, resources provide read-only access to information through standardized URI patterns.

The resource registry framework parallels the existing tool registry architecture, ensuring consistency in management patterns while addressing the unique requirements of resource access and content delivery.

## Architecture Overview

### Resource Lifecycle
Resources follow a structured lifecycle similar to tools:
```
registered → loaded → active → error/disabled
```

- **registered**: Resource factory is registered with metadata
- **loaded**: Resource instance is created and validated
- **active**: Resource is available for client access
- **error**: Resource has encountered failures
- **disabled**: Resource has been administratively disabled

### URI-Based Access Pattern
Resources are identified and accessed through URI patterns:
```
file:///path/to/document.txt
config://database/connection
api://external/service/data
custom://internal/metrics
```

## Core Components

### 1. Resource Types and Interfaces (`internal/resources/types.go`)

#### ResourceStatus Enumeration
```go
type ResourceStatus string

const (
    ResourceStatusUnknown    ResourceStatus = "unknown"
    ResourceStatusRegistered ResourceStatus = "registered"
    ResourceStatusLoaded     ResourceStatus = "loaded"
    ResourceStatusActive     ResourceStatus = "active"
    ResourceStatusError      ResourceStatus = "error"
    ResourceStatusDisabled   ResourceStatus = "disabled"
)
```

#### ResourceInfo Structure
```go
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
```

#### ResourceConfig Structure
```go
type ResourceConfig struct {
    Enabled       bool                   `json:"enabled"`
    Config        map[string]interface{} `json:"config"`
    CacheTimeout  int                    `json:"cache_timeout_seconds"`
    AccessControl map[string]string      `json:"access_control"`
}
```

#### ResourceFactory Interface
```go
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
```

#### ResourceRegistry Interface
```go
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
```

### 2. Registry Implementation (`internal/resources/registry.go`)

#### DefaultResourceRegistry Structure
```go
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
    mu              sync.RWMutex
    cacheMu         sync.RWMutex
}
```

#### Key Methods
- `Register(uri, factory)`: Register resource factory with URI validation
- `LoadResources(ctx)`: Create resource instances from factories
- `ValidateResources(ctx)`: Validate all registered resources
- `TransitionStatus(uri, status)`: Manage resource status transitions
- `RefreshResource(ctx, uri)`: Refresh cached resource content
- `Health()`: Return comprehensive health information

### 3. Factory Pattern (`internal/resources/factory.go`)

#### ResourceRegistryFactory Interface
```go
type ResourceRegistryFactory interface {
    CreateRegistry() (ResourceRegistry, error)
}
```

#### DefaultResourceRegistryFactory Implementation
- Creates registry with proper configuration
- Integrates with existing server infrastructure
- Provides fallback mechanisms for robustness

### 4. Validation (`internal/resources/validator.go`)

#### ResourceValidator Structure
```go
type ResourceValidator struct {
    config *config.Config
    logger *logger.Logger
}
```

#### Validation Functions
- URI format and pattern validation
- Access control policy enforcement
- Configuration parameter validation
- Resource content validation
- Cache expiration management

### 5. Circuit Breaker Integration (`internal/resources/circuit_breaker.go`)

#### CircuitBreakerResourceFactory Structure
```go
type CircuitBreakerResourceFactory struct {
    factory ResourceFactory
    breaker *gobreaker.CircuitBreaker[mcp.Resource]
}
```

#### Protection Mechanisms
- Resource creation failure protection
- Content access failure handling
- Automatic recovery and retry logic
- Failure threshold configuration

## Error Handling Strategy

### Resource-Specific Errors
```go
var (
    ErrResourceNotFound       = fmt.Errorf("resource not found")
    ErrResourceAlreadyExists  = fmt.Errorf("resource already exists")
    ErrInvalidResourceURI     = fmt.Errorf("invalid resource URI")
    ErrResourceValidation     = fmt.Errorf("resource validation failed")
    ErrResourceAccess         = fmt.Errorf("resource access denied")
    ErrResourceContent        = fmt.Errorf("resource content error")
    ErrCacheExpired          = fmt.Errorf("cached content expired")
)
```

### Validation Errors
- URI format validation
- Access control violations
- Configuration parameter errors
- Content type mismatches
- Cache coherency issues

## Health Monitoring Integration

### RegistryHealth Enhancement
```go
type RegistryHealth struct {
    Status            string              `json:"status"`
    ResourceCount     int                 `json:"resource_count"`
    ActiveResources   int                 `json:"active_resources"`
    ErrorResources    int                 `json:"error_resources"`
    CachedResources   int                 `json:"cached_resources"`
    CacheHitRate      float64            `json:"cache_hit_rate"`
    LastCheck         string             `json:"last_check"`
    Errors            []string           `json:"errors,omitempty"`
    ResourceStatuses  map[string]string  `json:"resource_statuses"`
    CircuitBreakers   map[string]string  `json:"circuit_breakers"`
}
```

### Health Endpoint Integration
- Resource registry health in `/health` endpoint
- Detailed resource information in `/resources/health` endpoint
- Resource metrics in `/metrics` endpoint

## Caching Strategy

### CachedContent Structure
```go
type CachedContent struct {
    Content    mcp.ResourceContent
    Timestamp  time.Time
    ExpiresAt  time.Time
    AccessCount int64
}
```

### Cache Management
- Configurable cache timeout per resource
- LRU eviction policy for memory management
- Cache hit/miss metrics collection
- Content validation on cache retrieval
- Automatic refresh for expired content

## Testing Strategy

### Unit Tests (`internal/resources/registry_test.go`)
- Resource registration and unregistration
- Status transition validation
- Health monitoring accuracy
- Circuit breaker protection
- Cache management functionality
- Error handling scenarios

### Validation Tests (`internal/resources/validator_test.go`)
- URI format validation
- Access control enforcement
- Configuration validation
- Content type verification
- Cache expiration logic

### Test Coverage Requirements
- All public interface methods
- Error conditions and edge cases
- Concurrent access scenarios
- Cache coherency validation
- Circuit breaker behavior

## Integration Points

### Server Integration
- Resource registry creation in server startup
- Health endpoint enhancement for resources
- Metrics collection for resource usage
- MCP server resource registration flow

### Configuration Integration
- Resource-specific configuration sections
- Cache configuration parameters
- Access control policy definitions
- Circuit breaker thresholds

### Logging Integration
- Structured logging for resource operations
- Access audit logging
- Performance metrics logging
- Error and failure logging

## Implementation Guidelines

### File Organization
```
internal/resources/
├── types.go              # Types, interfaces, and constants
├── registry.go           # Main registry implementation
├── factory.go            # Registry factory pattern
├── validator.go          # Validation logic
├── circuit_breaker.go    # Circuit breaker integration
├── registry_test.go      # Registry unit tests
└── validator_test.go     # Validator unit tests
```

### Code Quality Standards
- Follow Single Responsibility Principle
- Include meaningful comments for complex business logic
- Use dependency injection for testability
- Implement comprehensive error handling
- Maintain consistent naming conventions

### Performance Considerations
- Efficient URI pattern matching
- Optimized cache access patterns
- Minimal lock contention in concurrent scenarios
- Resource cleanup and memory management
- Circuit breaker performance impact

## Success Criteria

### Functional Requirements
1. Resource registry successfully manages resource lifecycle
2. URI-based resource identification and access works correctly
3. Circuit breaker protection prevents cascade failures
4. Health monitoring provides accurate resource status
5. Cache management improves resource access performance

### Non-Functional Requirements
1. All unit tests pass with comprehensive coverage
2. Resource registry integrates cleanly with existing server
3. Performance impact is minimal on server operations
4. Memory usage is bounded and predictable
5. Error handling provides clear diagnostics

### Integration Requirements
1. Health endpoints include resource registry status
2. Metrics endpoints track resource usage
3. Configuration system supports resource parameters
4. Logging provides adequate operational visibility
5. Circuit breaker protection is active and effective

## Future Extensibility

### Planned Enhancements
- Dynamic resource discovery and registration
- Resource content streaming for large data
- Advanced caching strategies (distributed cache)
- Resource access analytics and optimization
- Integration with external resource providers

### Extension Points
- Pluggable validation strategies
- Custom cache implementations
- Alternative circuit breaker configurations
- Resource content transformation pipelines
- Access control policy engines

This specification establishes the foundation for a robust, scalable resource management system that integrates seamlessly with the existing MCP server architecture while providing the flexibility for future enhancements.