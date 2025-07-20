package main

import (
	"mcp-server/internal/logger"
	"mcp-server/internal/tools"
	"mcp-server/internal/tools/echo"
)

func registerEchoTool(registry tools.ToolRegistry, log *logger.Logger) error {
	log.Info("Registering Echo tool")
	
	echoFactory := echo.NewEchoFactory()
	if err := registry.Register("echo", echoFactory); err != nil {
		log.Error("Failed to register Echo tool", "error", err)
		return err
	}
	
	log.Info("Successfully registered Echo tool")
	return nil
}

func registerAllTools(registry tools.ToolRegistry, log *logger.Logger) error {
	log.Info("Registering all available tools")
	
	if err := registerEchoTool(registry, log); err != nil {
		return err
	}
	
	log.Info("Successfully registered all tools")
	return nil
}