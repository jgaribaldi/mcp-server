package resources

import (
	"mcp-server/internal/config"
	"mcp-server/internal/logger"
)

// ResourceRegistryFactory creates resource registry instances
type ResourceRegistryFactory interface {
	CreateRegistry() (ResourceRegistry, error)
}

// DefaultResourceRegistryFactory implements ResourceRegistryFactory
type DefaultResourceRegistryFactory struct {
	config *config.Config
	logger *logger.Logger
}

// NewRegistryFactory creates a new resource registry factory
func NewRegistryFactory(cfg *config.Config, log *logger.Logger) ResourceRegistryFactory {
	return &DefaultResourceRegistryFactory{
		config: cfg,
		logger: log,
	}
}

// CreateRegistry implements ResourceRegistryFactory.CreateRegistry
func (f *DefaultResourceRegistryFactory) CreateRegistry() (ResourceRegistry, error) {
	f.logger.Info("creating resource registry",
		"max_resources", f.config.MCP.MaxResources,
	)

	// Create the registry
	registry := NewDefaultResourceRegistry(f.config, f.logger)
	
	f.logger.Info("resource registry created successfully")
	return registry, nil
}