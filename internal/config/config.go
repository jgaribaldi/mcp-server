package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the MCP server
type Config struct {
	Server ServerConfig
	Logger LoggerConfig
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Host string
	Port int
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level   string
	Format  string
	Service string
	Version string
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host: getEnv("MCP_SERVER_HOST", "localhost"),
			Port: getEnvInt("MCP_SERVER_PORT", 3000),
		},
		Logger: LoggerConfig{
			Level:   getEnv("MCP_LOG_LEVEL", "info"),
			Format:  getEnv("MCP_LOG_FORMAT", "json"),
			Service: getEnv("MCP_SERVICE_NAME", "mcp-server"),
			Version: getEnv("MCP_VERSION", "dev"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Host == "" {
		return fmt.Errorf("server host cannot be empty")
	}
	
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535, got %d", c.Server.Port)
	}

	validLevels := map[string]bool{
		"debug": true, "DEBUG": true,
		"info": true, "INFO": true,
		"warn": true, "WARN": true,
		"error": true, "ERROR": true,
	}
	if !validLevels[c.Logger.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logger.Level)
	}

	validFormats := map[string]bool{
		"json": true,
		"text": true,
	}
	if !validFormats[c.Logger.Format] {
		return fmt.Errorf("invalid log format: %s", c.Logger.Format)
	}

	return nil
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets environment variable as integer with default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}