package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	
	"gopkg.in/yaml.v3"
)

const (
	// Default server settings
	DefaultServerHost     = "localhost"
	DefaultServerPort     = 3000
	
	// Default HTTP server timeouts
	DefaultReadTimeout    = 15 * time.Second
	DefaultWriteTimeout   = 15 * time.Second
	DefaultIdleTimeout    = 60 * time.Second
	DefaultMaxHeaderBytes = 1 << 20 // 1MB
)

// Config holds all configuration for the MCP server
type Config struct {
	Server ServerConfig
	Logger LoggerConfig
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Host           string
	Port           int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	MaxHeaderBytes int
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level   string
	Format  string
	Service string
	Version string
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []string

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	if len(ve) == 1 {
		return ve[0]
	}
	return fmt.Sprintf("multiple validation errors: %s", strings.Join(ve, "; "))
}

// FileConfig represents configuration loaded from YAML files
type FileConfig struct {
	Server FileServerConfig `yaml:"server"`
	Logger FileLoggerConfig `yaml:"logger"`
}

type FileServerConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	ReadTimeout    string `yaml:"read_timeout"`
	WriteTimeout   string `yaml:"write_timeout"`
	IdleTimeout    string `yaml:"idle_timeout"`
	MaxHeaderBytes int    `yaml:"max_header_bytes"`
}

type FileLoggerConfig struct {
	Level   string `yaml:"level"`
	Format  string `yaml:"format"`
	Service string `yaml:"service"`
	Version string `yaml:"version"`
}

// loadConfigFile attempts to load configuration from YAML files
func loadConfigFile() (*FileConfig, error) {
	configPath := getEnv("MCP_CONFIG_FILE", "")
	if configPath == "" {
		// Try default locations
		candidates := []string{
			"configs/development.yaml",
			"configs/production.yaml",
			"configs/docker.yaml",
		}
		
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				configPath = candidate
				break
			}
		}
	}
	
	if configPath == "" {
		return nil, nil // No config file found, not an error
	}
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}
	
	var fileConfig FileConfig
	if err := yaml.Unmarshal(data, &fileConfig); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}
	
	return &fileConfig, nil
}

// mergeFileConfig merges file configuration with base config, respecting environment variable precedence
func mergeFileConfig(base *Config, file *FileConfig) *Config {
	if file == nil {
		return base
	}
	
	result := *base // Copy base config
	
	// Merge server config (only if not overridden by env vars)
	if file.Server.Host != "" && os.Getenv("MCP_SERVER_HOST") == "" {
		result.Server.Host = file.Server.Host
	}
	if file.Server.Port != 0 && os.Getenv("MCP_SERVER_PORT") == "" {
		result.Server.Port = file.Server.Port
	}
	if file.Server.ReadTimeout != "" && os.Getenv("MCP_SERVER_READ_TIMEOUT") == "" {
		if duration, err := time.ParseDuration(file.Server.ReadTimeout); err == nil {
			result.Server.ReadTimeout = duration
		}
	}
	if file.Server.WriteTimeout != "" && os.Getenv("MCP_SERVER_WRITE_TIMEOUT") == "" {
		if duration, err := time.ParseDuration(file.Server.WriteTimeout); err == nil {
			result.Server.WriteTimeout = duration
		}
	}
	if file.Server.IdleTimeout != "" && os.Getenv("MCP_SERVER_IDLE_TIMEOUT") == "" {
		if duration, err := time.ParseDuration(file.Server.IdleTimeout); err == nil {
			result.Server.IdleTimeout = duration
		}
	}
	if file.Server.MaxHeaderBytes != 0 && os.Getenv("MCP_SERVER_MAX_HEADER_BYTES") == "" {
		result.Server.MaxHeaderBytes = file.Server.MaxHeaderBytes
	}
	
	// Merge logger config (only if not overridden by env vars)
	if file.Logger.Level != "" && os.Getenv("MCP_LOG_LEVEL") == "" {
		result.Logger.Level = file.Logger.Level
	}
	if file.Logger.Format != "" && os.Getenv("MCP_LOG_FORMAT") == "" {
		result.Logger.Format = file.Logger.Format
	}
	if file.Logger.Service != "" && os.Getenv("MCP_SERVICE_NAME") == "" {
		result.Logger.Service = file.Logger.Service
	}
	if file.Logger.Version != "" && os.Getenv("MCP_VERSION") == "" {
		result.Logger.Version = file.Logger.Version
	}
	
	return &result
}

// Load loads configuration from environment variables and files with defaults
func Load() (*Config, error) {
	// Load base configuration with defaults
	cfg := &Config{
		Server: ServerConfig{
			Host:           getEnv("MCP_SERVER_HOST", DefaultServerHost),
			Port:           getEnvInt("MCP_SERVER_PORT", DefaultServerPort),
			ReadTimeout:    getEnvDuration("MCP_SERVER_READ_TIMEOUT", DefaultReadTimeout),
			WriteTimeout:   getEnvDuration("MCP_SERVER_WRITE_TIMEOUT", DefaultWriteTimeout),
			IdleTimeout:    getEnvDuration("MCP_SERVER_IDLE_TIMEOUT", DefaultIdleTimeout),
			MaxHeaderBytes: getEnvInt("MCP_SERVER_MAX_HEADER_BYTES", DefaultMaxHeaderBytes),
		},
		Logger: LoggerConfig{
			Level:   getEnv("MCP_LOG_LEVEL", "info"),
			Format:  getEnv("MCP_LOG_FORMAT", "json"),
			Service: getEnv("MCP_SERVICE_NAME", "mcp-server"),
			Version: getEnv("MCP_VERSION", "dev"),
		},
	}
	
	// Try to load configuration file
	fileConfig, err := loadConfigFile()
	if err != nil {
		// Log warning but don't fail - env vars might be sufficient
		// Note: We can't use logger here as it's not initialized yet
		fmt.Fprintf(os.Stderr, "Warning: failed to load config file: %v\n", err)
	}
	
	// Merge file config with environment variables taking precedence
	cfg = mergeFileConfig(cfg, fileConfig)
	
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return cfg, nil
}

// Validate validates the configuration with enhanced error reporting
func (c *Config) Validate() error {
	var errors ValidationErrors
	
	// Enhanced server validation
	if c.Server.Host == "" {
		errors = append(errors, "server host cannot be empty (hint: use 'localhost' for local development)")
	}
	
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errors = append(errors, fmt.Sprintf("server port must be between 1 and 65535, got %d (hint: use 3000 for development, 8080 for production)", c.Server.Port))
	} else if c.Server.Port < 1024 {
		// Note: This is a warning, not an error - privileged ports require root access
		// We don't add this to errors as it's not fatal, just noteworthy
	}
	
	// Enhanced timeout validation with cross-parameter checks
	if c.Server.ReadTimeout < 0 {
		errors = append(errors, fmt.Sprintf("server read timeout cannot be negative, got %v (hint: use 15s or larger)", c.Server.ReadTimeout))
	} else if c.Server.ReadTimeout > 5*time.Minute {
		errors = append(errors, fmt.Sprintf("server read timeout is very large: %v (hint: typically 15s-60s)", c.Server.ReadTimeout))
	}
	
	if c.Server.WriteTimeout < 0 {
		errors = append(errors, fmt.Sprintf("server write timeout cannot be negative, got %v (hint: use 15s or larger)", c.Server.WriteTimeout))
	} else if c.Server.WriteTimeout > 5*time.Minute {
		errors = append(errors, fmt.Sprintf("server write timeout is very large: %v (hint: typically 15s-60s)", c.Server.WriteTimeout))
	}
	
	if c.Server.IdleTimeout < 0 {
		errors = append(errors, fmt.Sprintf("server idle timeout cannot be negative, got %v (hint: use 60s or larger)", c.Server.IdleTimeout))
	}
	
	// Cross-parameter validation
	if c.Server.ReadTimeout > 0 && c.Server.IdleTimeout > 0 && c.Server.ReadTimeout >= c.Server.IdleTimeout {
		errors = append(errors, fmt.Sprintf("read timeout (%v) should be less than idle timeout (%v)", c.Server.ReadTimeout, c.Server.IdleTimeout))
	}
	
	if c.Server.WriteTimeout > 0 && c.Server.IdleTimeout > 0 && c.Server.WriteTimeout >= c.Server.IdleTimeout {
		errors = append(errors, fmt.Sprintf("write timeout (%v) should be less than idle timeout (%v)", c.Server.WriteTimeout, c.Server.IdleTimeout))
	}
	
	if c.Server.MaxHeaderBytes < 1 {
		errors = append(errors, fmt.Sprintf("server max header bytes must be positive, got %d (hint: use 1048576 for 1MB)", c.Server.MaxHeaderBytes))
	} else if c.Server.MaxHeaderBytes > 10*1024*1024 {
		errors = append(errors, fmt.Sprintf("server max header bytes is very large: %d (hint: typically 1MB-8MB)", c.Server.MaxHeaderBytes))
	}
	
	// Enhanced logger validation with case normalization
	normalizedLevel := strings.ToLower(c.Logger.Level)
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[normalizedLevel] {
		errors = append(errors, fmt.Sprintf("invalid log level: %s (valid options: debug, info, warn, error)", c.Logger.Level))
	}
	
	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[c.Logger.Format] {
		errors = append(errors, fmt.Sprintf("invalid log format: %s (valid options: json, text)", c.Logger.Format))
	}
	
	if len(errors) > 0 {
		return errors
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

// getEnvDuration gets environment variable as duration with default value
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}