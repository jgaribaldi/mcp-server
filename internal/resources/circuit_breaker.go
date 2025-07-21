package resources

import (
	"context"
	"fmt"
	"time"

	"mcp-server/internal/mcp"
)

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	MaxFailures    uint32
	Timeout        time.Duration
	RetryTimeout   time.Duration
	ResetTimeout   time.Duration
}

// DefaultCircuitBreakerConfig returns default circuit breaker configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxFailures:  5,
		Timeout:      30 * time.Second,
		RetryTimeout: 60 * time.Second,
		ResetTimeout: 300 * time.Second,
	}
}

// CircuitState represents the current state of a circuit breaker
type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half-open"
)

// CircuitBreakerResourceFactory wraps a ResourceFactory with circuit breaker protection
type CircuitBreakerResourceFactory struct {
	factory       ResourceFactory
	config        CircuitBreakerConfig
	state         CircuitState
	failures      uint32
	lastFailTime  time.Time
	lastResetTime time.Time
}

// NewCircuitBreakerResourceFactory creates a new circuit breaker wrapped resource factory
func NewCircuitBreakerResourceFactory(factory ResourceFactory, config CircuitBreakerConfig) *CircuitBreakerResourceFactory {
	return &CircuitBreakerResourceFactory{
		factory: factory,
		config:  config,
		state:   CircuitClosed,
	}
}

// URI implements ResourceFactory.URI
func (cb *CircuitBreakerResourceFactory) URI() string {
	return cb.factory.URI()
}

// Name implements ResourceFactory.Name
func (cb *CircuitBreakerResourceFactory) Name() string {
	return cb.factory.Name()
}

// Description implements ResourceFactory.Description
func (cb *CircuitBreakerResourceFactory) Description() string {
	return cb.factory.Description()
}

// MimeType implements ResourceFactory.MimeType
func (cb *CircuitBreakerResourceFactory) MimeType() string {
	return cb.factory.MimeType()
}

// Version implements ResourceFactory.Version
func (cb *CircuitBreakerResourceFactory) Version() string {
	return cb.factory.Version()
}

// Tags implements ResourceFactory.Tags
func (cb *CircuitBreakerResourceFactory) Tags() []string {
	return cb.factory.Tags()
}

// Capabilities implements ResourceFactory.Capabilities
func (cb *CircuitBreakerResourceFactory) Capabilities() []string {
	return cb.factory.Capabilities()
}

// Validate implements ResourceFactory.Validate
func (cb *CircuitBreakerResourceFactory) Validate(config ResourceConfig) error {
	return cb.factory.Validate(config)
}

// Create implements ResourceFactory.Create with circuit breaker protection
func (cb *CircuitBreakerResourceFactory) Create(ctx context.Context, config ResourceConfig) (mcp.Resource, error) {
	// Check circuit breaker state
	if err := cb.checkState(); err != nil {
		return nil, err
	}

	// Create timeout context
	createCtx, cancel := context.WithTimeout(ctx, cb.config.Timeout)
	defer cancel()

	// Attempt to create resource
	resource, err := cb.factory.Create(createCtx, config)
	
	if err != nil {
		cb.recordFailure()
		return nil, fmt.Errorf("circuit breaker: resource creation failed: %w", err)
	}

	cb.recordSuccess()
	return resource, nil
}

// checkState checks the current circuit breaker state and updates it if necessary
func (cb *CircuitBreakerResourceFactory) checkState() error {
	now := time.Now()

	switch cb.state {
	case CircuitClosed:
		// Circuit is closed, allow requests
		return nil

	case CircuitOpen:
		// Check if we should transition to half-open
		if now.Sub(cb.lastFailTime) > cb.config.RetryTimeout {
			cb.state = CircuitHalfOpen
			return nil
		}
		return fmt.Errorf("circuit breaker is open: resource factory is temporarily unavailable")

	case CircuitHalfOpen:
		// Allow one request to test if the service has recovered
		return nil

	default:
		return fmt.Errorf("unknown circuit breaker state: %s", cb.state)
	}
}

// recordFailure records a failure and updates circuit breaker state
func (cb *CircuitBreakerResourceFactory) recordFailure() {
	cb.failures++
	cb.lastFailTime = time.Now()

	switch cb.state {
	case CircuitClosed:
		if cb.failures >= cb.config.MaxFailures {
			cb.state = CircuitOpen
		}
	case CircuitHalfOpen:
		// Failed in half-open state, go back to open
		cb.state = CircuitOpen
	}
}

// recordSuccess records a success and updates circuit breaker state
func (cb *CircuitBreakerResourceFactory) recordSuccess() {
	switch cb.state {
	case CircuitHalfOpen:
		// Success in half-open state, reset to closed
		cb.state = CircuitClosed
		cb.failures = 0
		cb.lastResetTime = time.Now()
	case CircuitClosed:
		// Gradually reduce failure count on success
		if cb.failures > 0 {
			cb.failures--
		}
	}
}

// Status returns the current circuit breaker status
func (cb *CircuitBreakerResourceFactory) Status() string {
	return string(cb.state)
}

// GetMetrics returns circuit breaker metrics
func (cb *CircuitBreakerResourceFactory) GetMetrics() CircuitBreakerMetrics {
	return CircuitBreakerMetrics{
		State:         cb.state,
		Failures:      cb.failures,
		MaxFailures:   cb.config.MaxFailures,
		LastFailTime:  cb.lastFailTime,
		LastResetTime: cb.lastResetTime,
	}
}

// CircuitBreakerMetrics contains metrics for a circuit breaker
type CircuitBreakerMetrics struct {
	State         CircuitState `json:"state"`
	Failures      uint32       `json:"failures"`
	MaxFailures   uint32       `json:"max_failures"`
	LastFailTime  time.Time    `json:"last_fail_time"`
	LastResetTime time.Time    `json:"last_reset_time"`
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreakerResourceFactory) Reset() {
	cb.state = CircuitClosed
	cb.failures = 0
	cb.lastResetTime = time.Now()
}

// IsOpen returns true if the circuit breaker is open
func (cb *CircuitBreakerResourceFactory) IsOpen() bool {
	return cb.state == CircuitOpen
}

// IsHalfOpen returns true if the circuit breaker is half-open
func (cb *CircuitBreakerResourceFactory) IsHalfOpen() bool {
	return cb.state == CircuitHalfOpen
}

// IsClosed returns true if the circuit breaker is closed
func (cb *CircuitBreakerResourceFactory) IsClosed() bool {
	return cb.state == CircuitClosed
}