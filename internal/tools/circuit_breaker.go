package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker/v2"

	"mcp-server/internal/mcp"
)

// CircuitBreakerToolFactory wraps a ToolFactory with circuit breaker protection
type CircuitBreakerToolFactory struct {
	factory ToolFactory
	breaker *gobreaker.CircuitBreaker[mcp.Tool]
}

// CircuitBreakerConfig holds configuration for circuit breaker behavior
type CircuitBreakerConfig struct {
	MaxRequests uint32
	Interval    time.Duration
	Timeout     time.Duration
}

// DefaultCircuitBreakerConfig returns sensible defaults for tool creation circuit breaker
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxRequests: 3,                // Allow 3 requests in half-open state
		Interval:    10 * time.Second, // Reset failure count every 10 seconds
		Timeout:     30 * time.Second, // Stay open for 30 seconds before trying half-open
	}
}

// NewCircuitBreakerToolFactory creates a new circuit breaker wrapped tool factory
func NewCircuitBreakerToolFactory(factory ToolFactory, config CircuitBreakerConfig) *CircuitBreakerToolFactory {
	settings := gobreaker.Settings{
		Name:        fmt.Sprintf("tool_factory_%s", factory.Name()),
		MaxRequests: config.MaxRequests,
		Interval:    config.Interval,
		Timeout:     config.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			// State changes are handled by the registry for logging
		},
	}

	breaker := gobreaker.NewCircuitBreaker[mcp.Tool](settings)

	return &CircuitBreakerToolFactory{
		factory: factory,
		breaker: breaker,
	}
}

// Name implements ToolFactory.Name
func (cb *CircuitBreakerToolFactory) Name() string {
	return cb.factory.Name()
}

// Description implements ToolFactory.Description
func (cb *CircuitBreakerToolFactory) Description() string {
	return cb.factory.Description()
}

// Version implements ToolFactory.Version
func (cb *CircuitBreakerToolFactory) Version() string {
	return cb.factory.Version()
}

// Capabilities implements ToolFactory.Capabilities
func (cb *CircuitBreakerToolFactory) Capabilities() []string {
	return cb.factory.Capabilities()
}

// Requirements implements ToolFactory.Requirements
func (cb *CircuitBreakerToolFactory) Requirements() map[string]string {
	return cb.factory.Requirements()
}

// Validate implements ToolFactory.Validate
func (cb *CircuitBreakerToolFactory) Validate(config ToolConfig) error {
	return cb.factory.Validate(config)
}

// Create implements ToolFactory.Create with circuit breaker protection
func (cb *CircuitBreakerToolFactory) Create(ctx context.Context, config ToolConfig) (mcp.Tool, error) {
	tool, err := cb.breaker.Execute(func() (mcp.Tool, error) {
		return cb.factory.Create(ctx, config)
	})
	
	if err != nil {
		return nil, fmt.Errorf("circuit breaker protected tool creation failed: %w", err)
	}
	
	return tool, nil
}

// GetCircuitBreakerState returns the current state of the circuit breaker
func (cb *CircuitBreakerToolFactory) GetCircuitBreakerState() gobreaker.State {
	return cb.breaker.State()
}

// GetCircuitBreakerCounts returns the current counts from the circuit breaker
func (cb *CircuitBreakerToolFactory) GetCircuitBreakerCounts() gobreaker.Counts {
	return cb.breaker.Counts()
}

// IsCircuitBreakerOpen returns true if the circuit breaker is open
func (cb *CircuitBreakerToolFactory) IsCircuitBreakerOpen() bool {
	return cb.breaker.State() == gobreaker.StateOpen
}

// GetUnderlyingFactory returns the wrapped factory for introspection
func (cb *CircuitBreakerToolFactory) GetUnderlyingFactory() ToolFactory {
	return cb.factory
}