package echo

import (
	"context"
	"fmt"

	"mcp-server/internal/mcp"
	"mcp-server/internal/tools"
)

type EchoFactory struct{}

func NewEchoFactory() *EchoFactory {
	return &EchoFactory{}
}

func (f *EchoFactory) Name() string {
	return "echo"
}

func (f *EchoFactory) Description() string {
	return "Simple text manipulation tool for testing and demonstration"
}

func (f *EchoFactory) Version() string {
	return "1.0.0"
}

func (f *EchoFactory) Capabilities() []string {
	return []string{"text_processing", "demonstration"}
}

func (f *EchoFactory) Requirements() map[string]string {
	return map[string]string{
		"runtime": "go",
	}
}

func (f *EchoFactory) Create(ctx context.Context, config tools.ToolConfig) (mcp.Tool, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("echo tool is disabled in configuration")
	}

	tool := NewEchoTool()
	if tool == nil {
		return nil, fmt.Errorf("failed to create echo tool instance")
	}

	return tool, nil
}

func (f *EchoFactory) Validate(config tools.ToolConfig) error {
	if config.Timeout < 0 {
		return fmt.Errorf("invalid timeout value: %d (must be non-negative)", config.Timeout)
	}

	if config.MaxRetries < 0 {
		return fmt.Errorf("invalid max retries value: %d (must be non-negative)", config.MaxRetries)
	}

	return nil
}