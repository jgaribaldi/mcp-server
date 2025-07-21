package tools

import (
	"context"
	"fmt"

	"github.com/sony/gobreaker/v2"

	"mcp-server/internal/mcp"
	"mcp-server/internal/registry"
)

type CircuitBreakerToolFactory struct {
	factory ToolFactory
	breaker *registry.CircuitBreakerFactory[mcp.Tool]
}

type CircuitBreakerConfig = registry.CircuitBreakerConfig

func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return registry.DefaultCircuitBreakerConfig()
}

func NewCircuitBreakerToolFactory(factory ToolFactory, config CircuitBreakerConfig) *CircuitBreakerToolFactory {
	breaker := registry.NewCircuitBreakerFactory[mcp.Tool](factory.GetName(), config)

	return &CircuitBreakerToolFactory{
		factory: factory,
		breaker: breaker,
	}
}

func (cb *CircuitBreakerToolFactory) GetName() string {
	return cb.factory.GetName()
}

func (cb *CircuitBreakerToolFactory) GetDescription() string {
	return cb.factory.GetDescription()
}

func (cb *CircuitBreakerToolFactory) GetVersion() string {
	return cb.factory.GetVersion()
}

func (cb *CircuitBreakerToolFactory) GetCapabilities() []string {
	return cb.factory.GetCapabilities()
}

func (cb *CircuitBreakerToolFactory) Requirements() map[string]string {
	return cb.factory.Requirements()
}

func (cb *CircuitBreakerToolFactory) Validate(config ToolConfig) error {
	return cb.factory.Validate(config)
}

func (cb *CircuitBreakerToolFactory) Create(ctx context.Context, config ToolConfig) (mcp.Tool, error) {
	tool, err := cb.breaker.ExecuteWithContext(ctx, func(ctx context.Context) (mcp.Tool, error) {
		return cb.factory.Create(ctx, config)
	})
	
	if err != nil {
		return nil, fmt.Errorf("circuit breaker protected tool creation failed: %w", err)
	}
	
	return tool, nil
}

func (cb *CircuitBreakerToolFactory) GetCircuitBreakerState() gobreaker.State {
	return cb.breaker.GetState()
}

func (cb *CircuitBreakerToolFactory) GetCircuitBreakerCounts() gobreaker.Counts {
	return cb.breaker.GetCounts()
}

func (cb *CircuitBreakerToolFactory) IsCircuitBreakerOpen() bool {
	return cb.breaker.IsOpen()
}

func (cb *CircuitBreakerToolFactory) GetUnderlyingFactory() ToolFactory {
	return cb.factory
}

func (cb *CircuitBreakerToolFactory) Status() string {
	return cb.breaker.Status()
}

func (cb *CircuitBreakerToolFactory) GetMetrics() registry.CircuitBreakerMetrics {
	return cb.breaker.GetMetrics()
}