package resources

import (
	"mcp-server/internal/config"
	"mcp-server/internal/logger"
)

type ResourceRegistryFactory interface {
	CreateRegistry() (ResourceRegistry, error)
}

type DefaultResourceRegistryFactory struct {
	config *config.Config
	logger *logger.Logger
}

func NewRegistryFactory(cfg *config.Config, log *logger.Logger) ResourceRegistryFactory {
	return &DefaultResourceRegistryFactory{
		config: cfg,
		logger: log,
	}
}

func (f *DefaultResourceRegistryFactory) CreateRegistry() (ResourceRegistry, error) {
	f.logger.Info("creating resource registry",
		"max_resources", f.config.MCP.MaxResources,
	)

	registry := NewDefaultResourceRegistry(f.config, f.logger)
	
	f.logger.Info("resource registry created successfully")
	return registry, nil
}
