package main

import (
	"fmt"

	"mcp-server/internal/config"
	"mcp-server/internal/logger"
	"mcp-server/internal/resources"
	"mcp-server/internal/resources/files"
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

func registerFileSystemResource(registry resources.ResourceRegistry, cfg *config.Config, log *logger.Logger) error {
	if !cfg.FileResource.Enabled {
		log.Info("File system resources disabled in configuration")
		return nil
	}

	log.Info("Registering file system resource factory")

	// Register factory with base directory URI
	baseURI := fmt.Sprintf("file://%s", cfg.FileResource.BaseDirectory)
	
	factoryConfig := files.FileSystemFactoryConfig{
		Name:               "file-system",
		Description:        "File system resource factory for secure file access",
		Version:            "1.0.0",
		BaseURI:            baseURI,
		BasePath:           cfg.FileResource.BaseDirectory,
		AllowedDirectories: cfg.FileResource.AllowedDirectories,
		MaxFileSize:        cfg.FileResource.MaxFileSize,
		AllowedExtensions:  cfg.FileResource.AllowedExtensions,
		BlockedPatterns:    cfg.FileResource.BlockedPatterns,
		Logger:            log,
	}

	factory, err := files.NewFileSystemResourceFactory(factoryConfig)
	if err != nil {
		log.Error("Failed to create file system resource factory", "error", err)
		return err
	}

	if err := registry.Register(baseURI, factory); err != nil {
		log.Error("Failed to register file system resource factory", "error", err)
		return err
	}

	log.Info("Successfully registered file system resource factory",
		"base_directory", cfg.FileResource.BaseDirectory,
		"max_file_size", cfg.FileResource.MaxFileSize,
		"allowed_extensions", cfg.FileResource.AllowedExtensions)
	return nil
}

func registerAllResources(registry resources.ResourceRegistry, cfg *config.Config, log *logger.Logger) error {
	log.Info("Registering all available resources")

	if err := registerFileSystemResource(registry, cfg, log); err != nil {
		return err
	}

	log.Info("Successfully registered all resources")
	return nil
}