package resources

import (
	"context"
	"fmt"
	"time"

	"mcp-server/internal/mcp"
)

type CircuitBreakerConfig struct {
	MaxFailures    uint32
	Timeout        time.Duration
	RetryTimeout   time.Duration
	ResetTimeout   time.Duration
}

func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxFailures:  5,
		Timeout:      30 * time.Second,
		RetryTimeout: 60 * time.Second,
		ResetTimeout: 300 * time.Second,
	}
}

type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half-open"
)

type CircuitBreakerResourceFactory struct {
	factory       ResourceFactory
	config        CircuitBreakerConfig
	state         CircuitState
	failures      uint32
	lastFailTime  time.Time
	lastResetTime time.Time
}

func NewCircuitBreakerResourceFactory(factory ResourceFactory, config CircuitBreakerConfig) *CircuitBreakerResourceFactory {
	return &CircuitBreakerResourceFactory{
		factory: factory,
		config:  config,
		state:   CircuitClosed,
	}
}

func (cb *CircuitBreakerResourceFactory) URI() string {
	return cb.factory.URI()
}

func (cb *CircuitBreakerResourceFactory) Name() string {
	return cb.factory.Name()
}

func (cb *CircuitBreakerResourceFactory) Description() string {
	return cb.factory.Description()
}

func (cb *CircuitBreakerResourceFactory) MimeType() string {
	return cb.factory.MimeType()
}

func (cb *CircuitBreakerResourceFactory) Version() string {
	return cb.factory.Version()
}

func (cb *CircuitBreakerResourceFactory) Tags() []string {
	return cb.factory.Tags()
}

func (cb *CircuitBreakerResourceFactory) Capabilities() []string {
	return cb.factory.Capabilities()
}

func (cb *CircuitBreakerResourceFactory) Validate(config ResourceConfig) error {
	return cb.factory.Validate(config)
}

func (cb *CircuitBreakerResourceFactory) Create(ctx context.Context, config ResourceConfig) (mcp.Resource, error) {
	if err := cb.checkState(); err != nil {
		return nil, err
	}

	createCtx, cancel := context.WithTimeout(ctx, cb.config.Timeout)
	defer cancel()

	resource, err := cb.factory.Create(createCtx, config)
	
	if err != nil {
		cb.recordFailure()
		return nil, fmt.Errorf("circuit breaker: resource creation failed: %w", err)
	}

	cb.recordSuccess()
	return resource, nil
}

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

func (cb *CircuitBreakerResourceFactory) Status() string {
	return string(cb.state)
}

func (cb *CircuitBreakerResourceFactory) GetMetrics() CircuitBreakerMetrics {
	return CircuitBreakerMetrics{
		State:         cb.state,
		Failures:      cb.failures,
		MaxFailures:   cb.config.MaxFailures,
		LastFailTime:  cb.lastFailTime,
		LastResetTime: cb.lastResetTime,
	}
}

type CircuitBreakerMetrics struct {
	State         CircuitState `json:"state"`
	Failures      uint32       `json:"failures"`
	MaxFailures   uint32       `json:"max_failures"`
	LastFailTime  time.Time    `json:"last_fail_time"`
	LastResetTime time.Time    `json:"last_reset_time"`
}

func (cb *CircuitBreakerResourceFactory) Reset() {
	cb.state = CircuitClosed
	cb.failures = 0
	cb.lastResetTime = time.Now()
}

func (cb *CircuitBreakerResourceFactory) IsOpen() bool {
	return cb.state == CircuitOpen
}

func (cb *CircuitBreakerResourceFactory) IsHalfOpen() bool {
	return cb.state == CircuitHalfOpen
}

func (cb *CircuitBreakerResourceFactory) IsClosed() bool {
	return cb.state == CircuitClosed
}
