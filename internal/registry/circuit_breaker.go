package registry

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker/v2"
)

// CircuitBreakerConfig holds configuration for circuit breaker behavior
type CircuitBreakerConfig struct {
	MaxRequests uint32
	Interval    time.Duration
	Timeout     time.Duration
}

// DefaultCircuitBreakerConfig returns sensible defaults for entity creation circuit breaker
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxRequests: 3,                // Allow 3 requests in half-open state
		Interval:    10 * time.Second, // Reset failure count every 10 seconds
		Timeout:     30 * time.Second, // Stay open for 30 seconds before trying half-open
	}
}

// CircuitBreakerFactory wraps any factory with circuit breaker protection
type CircuitBreakerFactory[T any] struct {
	name    string
	breaker *gobreaker.CircuitBreaker[T]
}

// NewCircuitBreakerFactory creates a new circuit breaker wrapped factory
func NewCircuitBreakerFactory[T any](name string, config CircuitBreakerConfig) *CircuitBreakerFactory[T] {
	settings := gobreaker.Settings{
		Name:        fmt.Sprintf("factory_%s", name),
		MaxRequests: config.MaxRequests,
		Interval:    config.Interval,
		Timeout:     config.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			// State changes can be logged by the caller if needed
		},
	}

	breaker := gobreaker.NewCircuitBreaker[T](settings)

	return &CircuitBreakerFactory[T]{
		name:    name,
		breaker: breaker,
	}
}

// Execute runs the provided function with circuit breaker protection
func (cb *CircuitBreakerFactory[T]) Execute(fn func() (T, error)) (T, error) {
	result, err := cb.breaker.Execute(fn)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("circuit breaker protected operation failed: %w", err)
	}
	return result, nil
}

// ExecuteWithContext runs the provided function with circuit breaker protection and context
func (cb *CircuitBreakerFactory[T]) ExecuteWithContext(ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	result, err := cb.breaker.Execute(func() (T, error) {
		return fn(ctx)
	})
	if err != nil {
		var zero T
		return zero, fmt.Errorf("circuit breaker protected operation failed: %w", err)
	}
	return result, nil
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreakerFactory[T]) GetState() gobreaker.State {
	return cb.breaker.State()
}

// GetCounts returns the current counts from the circuit breaker
func (cb *CircuitBreakerFactory[T]) GetCounts() gobreaker.Counts {
	return cb.breaker.Counts()
}

// IsOpen returns true if the circuit breaker is open
func (cb *CircuitBreakerFactory[T]) IsOpen() bool {
	return cb.breaker.State() == gobreaker.StateOpen
}

// GetName returns the name of the circuit breaker
func (cb *CircuitBreakerFactory[T]) GetName() string {
	return cb.name
}

// Status returns a string representation of the circuit breaker status
func (cb *CircuitBreakerFactory[T]) Status() string {
	switch cb.breaker.State() {
	case gobreaker.StateClosed:
		return "closed"
	case gobreaker.StateHalfOpen:
		return "half-open"
	case gobreaker.StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// CircuitBreakerMetrics provides information about circuit breaker performance
type CircuitBreakerMetrics struct {
	Name         string            `json:"name"`
	State        string            `json:"state"`
	Requests     uint32            `json:"requests"`
	Successes    uint32            `json:"successes"`
	Failures     uint32            `json:"failures"`
	Timeouts     uint32            `json:"timeouts"`
	MaxRequests  uint32            `json:"max_requests"`
}

// GetMetrics returns current circuit breaker metrics
func (cb *CircuitBreakerFactory[T]) GetMetrics() CircuitBreakerMetrics {
	counts := cb.breaker.Counts()
	
	return CircuitBreakerMetrics{
		Name:         cb.name,
		State:        cb.Status(),
		Requests:     counts.Requests,
		Successes:    counts.TotalSuccesses,
		Failures:     counts.TotalFailures,
		Timeouts:     counts.ConsecutiveFailures,
		MaxRequests:  0, // gobreaker doesn't expose this directly
	}
}