package tools

import (
	"fmt"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
	"mcp-server/internal/tools/adapters"
)

// RegistryFactory creates tool registry instances with appropriate library adapters
type RegistryFactory interface {
	CreateRegistry() (ToolRegistry, error)
	GetSupportedLibraries() []string
	GetDefaultLibrary() string
}

// DefaultRegistryFactory implements RegistryFactory using hardcoded mark3labs adapter
type DefaultRegistryFactory struct {
	config *config.Config
	logger *logger.Logger
}

// NewRegistryFactory creates a new registry factory instance
func NewRegistryFactory(cfg *config.Config, log *logger.Logger) RegistryFactory {
	return &DefaultRegistryFactory{
		config: cfg,
		logger: log,
	}
}

// CreateRegistry implements RegistryFactory.CreateRegistry
func (f *DefaultRegistryFactory) CreateRegistry() (ToolRegistry, error) {
	f.logger.Info("creating tool registry with mark3labs adapter")

	// Create mark3labs adapter (hardcoded for now)
	adapter := adapters.NewMark3LabsAdapter(f.config, f.logger)
	if adapter == nil {
		return nil, fmt.Errorf("failed to create mark3labs adapter")
	}

	// Create registry with adapter
	registry := NewDefaultToolRegistryWithAdapter(f.config, f.logger, adapter)
	if registry == nil {
		return nil, fmt.Errorf("failed to create tool registry")
	}

	f.logger.Info("tool registry created successfully with mark3labs adapter")
	return registry, nil
}

// GetSupportedLibraries implements RegistryFactory.GetSupportedLibraries
func (f *DefaultRegistryFactory) GetSupportedLibraries() []string {
	// For now, only mark3labs is supported
	// In the future, we'll add official SDK when available
	return []string{"mark3labs"}
}

// GetDefaultLibrary implements RegistryFactory.GetDefaultLibrary
func (f *DefaultRegistryFactory) GetDefaultLibrary() string {
	// Use mark3labs as default until official SDK is stable
	return "mark3labs"
}