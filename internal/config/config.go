package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	
	"gopkg.in/yaml.v3"
)

// TODO: refactor all functions following Single Responsibility Principle

const (
	// Default server settings
	DefaultServerHost     = "localhost"
	DefaultServerPort     = 3000
	
	// Default HTTP server timeouts
	DefaultReadTimeout    = 15 * time.Second
	DefaultWriteTimeout   = 15 * time.Second
	DefaultIdleTimeout    = 60 * time.Second
	DefaultMaxHeaderBytes = 1 << 20 // 1MB
	
	// Default MCP settings
	DefaultProtocolTimeout = 30 * time.Second
	DefaultMaxTools        = 100
	DefaultMaxResources    = 100
	DefaultDebugMode       = false
	DefaultEnableMetrics   = true
	DefaultBufferSize      = 4096
)

type Config struct {
	Server ServerConfig
	Logger LoggerConfig
	MCP    MCPConfig
}

type ServerConfig struct {
	Host           string
	Port           int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	MaxHeaderBytes int
}

type LoggerConfig struct {
	Level     string
	Format    string
	Service   string
	Version   string
	UseEmojis bool
}

type MCPConfig struct {
	ProtocolTimeout time.Duration
	MaxTools        int
	MaxResources    int
	DebugMode       bool
	EnableMetrics   bool
	BufferSize      int
	ResourceCache   ResourceCacheConfig
}

type ResourceCacheConfig struct {
	DefaultTimeout int  `json:"default_timeout_seconds"`
	MaxSize        int  `json:"max_size"`
	Enabled        bool `json:"enabled"`
}

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

type FileConfig struct {
	Server FileServerConfig `yaml:"server"`
	Logger FileLoggerConfig `yaml:"logger"`
	MCP    FileMCPConfig    `yaml:"mcp"`
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
	Level     string `yaml:"level"`
	Format    string `yaml:"format"`
	Service   string `yaml:"service"`
	Version   string `yaml:"version"`
	UseEmojis bool   `yaml:"use_emojis"`
}

type FileMCPConfig struct {
	ProtocolTimeout string              `yaml:"protocol_timeout"`
	MaxTools        int                 `yaml:"max_tools"`
	MaxResources    int                 `yaml:"max_resources"`
	DebugMode       bool                `yaml:"debug_mode"`
	EnableMetrics   bool                `yaml:"enable_metrics"`
	BufferSize      int                 `yaml:"buffer_size"`
	ResourceCache   FileResourceCacheConfig `yaml:"resource_cache"`
}

type FileResourceCacheConfig struct {
	DefaultTimeout int  `yaml:"default_timeout_seconds"`
	MaxSize        int  `yaml:"max_size"`
	Enabled        bool `yaml:"enabled"`
}

// getEnvBool gets environment variable as boolean with default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		switch strings.ToLower(value) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return defaultValue
}

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
	if os.Getenv("MCP_LOG_USE_EMOJIS") == "" {
		result.Logger.UseEmojis = file.Logger.UseEmojis
	}
	
	// Merge MCP config (only if not overridden by env vars)
	if file.MCP.ProtocolTimeout != "" && os.Getenv("MCP_PROTOCOL_TIMEOUT") == "" {
		if duration, err := time.ParseDuration(file.MCP.ProtocolTimeout); err == nil {
			result.MCP.ProtocolTimeout = duration
		}
	}
	if file.MCP.MaxTools != 0 && os.Getenv("MCP_MAX_TOOLS") == "" {
		result.MCP.MaxTools = file.MCP.MaxTools
	}
	if file.MCP.MaxResources != 0 && os.Getenv("MCP_MAX_RESOURCES") == "" {
		result.MCP.MaxResources = file.MCP.MaxResources
	}
	if os.Getenv("MCP_DEBUG_MODE") == "" {
		result.MCP.DebugMode = file.MCP.DebugMode
	}
	if os.Getenv("MCP_ENABLE_METRICS") == "" {
		result.MCP.EnableMetrics = file.MCP.EnableMetrics
	}
	if file.MCP.BufferSize != 0 && os.Getenv("MCP_BUFFER_SIZE") == "" {
		result.MCP.BufferSize = file.MCP.BufferSize
	}
	
	// Merge resource cache config (only if not overridden by env vars)
	if file.MCP.ResourceCache.DefaultTimeout != 0 && os.Getenv("MCP_RESOURCE_CACHE_TIMEOUT") == "" {
		result.MCP.ResourceCache.DefaultTimeout = file.MCP.ResourceCache.DefaultTimeout
	}
	if file.MCP.ResourceCache.MaxSize != 0 && os.Getenv("MCP_RESOURCE_CACHE_MAX_SIZE") == "" {
		result.MCP.ResourceCache.MaxSize = file.MCP.ResourceCache.MaxSize
	}
	if os.Getenv("MCP_RESOURCE_CACHE_ENABLED") == "" {
		result.MCP.ResourceCache.Enabled = file.MCP.ResourceCache.Enabled
	}
	
	return &result
}

func Load() (*Config, error) {
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
			Level:     getEnv("MCP_LOG_LEVEL", "info"),
			Format:    getEnv("MCP_LOG_FORMAT", "console"),
			Service:   getEnv("MCP_SERVICE_NAME", "mcp-server"),
			Version:   getEnv("MCP_VERSION", "dev"),
			UseEmojis: getEnvBool("MCP_LOG_USE_EMOJIS", true),
		},
		MCP: MCPConfig{
			ProtocolTimeout: getEnvDuration("MCP_PROTOCOL_TIMEOUT", DefaultProtocolTimeout),
			MaxTools:        getEnvInt("MCP_MAX_TOOLS", DefaultMaxTools),
			MaxResources:    getEnvInt("MCP_MAX_RESOURCES", DefaultMaxResources),
			DebugMode:       getEnvBool("MCP_DEBUG_MODE", DefaultDebugMode),
			EnableMetrics:   getEnvBool("MCP_ENABLE_METRICS", DefaultEnableMetrics),
			BufferSize:      getEnvInt("MCP_BUFFER_SIZE", DefaultBufferSize),
			ResourceCache: ResourceCacheConfig{
				DefaultTimeout: getEnvInt("MCP_RESOURCE_CACHE_TIMEOUT", 300),
				MaxSize:        getEnvInt("MCP_RESOURCE_CACHE_MAX_SIZE", 1000),
				Enabled:        getEnvBool("MCP_RESOURCE_CACHE_ENABLED", true),
			},
		},
	}
	
	fileConfig, err := loadConfigFile()
	if err != nil {
		// Log warning but don't fail - env vars might be sufficient
		// Note: We can't use logger here as it's not initialized yet
		fmt.Fprintf(os.Stderr, "Warning: failed to load config file: %v\n", err)
	}
	
	cfg = mergeFileConfig(cfg, fileConfig)
	
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return cfg, nil
}

func (c *Config) Validate() error {
	var errors ValidationErrors
	
	if c.Server.Host == "" {
		errors = append(errors, "server host cannot be empty (hint: use 'localhost' for local development)")
	}
	
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errors = append(errors, fmt.Sprintf("server port must be between 1 and 65535, got %d (hint: use 3000 for development, 8080 for production)", c.Server.Port))
	} else if c.Server.Port < 1024 {
		// Note: This is a warning, not an error - privileged ports require root access
		// We don't add this to errors as it's not fatal, just noteworthy
	}
	
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
	
	normalizedLevel := strings.ToLower(c.Logger.Level)
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[normalizedLevel] {
		errors = append(errors, fmt.Sprintf("invalid log level: %s (valid options: debug, info, warn, error)", c.Logger.Level))
	}
	
	validFormats := map[string]bool{"json": true, "text": true, "console": true}
	if !validFormats[c.Logger.Format] {
		errors = append(errors, fmt.Sprintf("invalid log format: %s (valid options: json, text, console)", c.Logger.Format))
	}
	
	// MCP configuration validation
	if c.MCP.ProtocolTimeout < 0 {
		errors = append(errors, fmt.Sprintf("MCP protocol timeout cannot be negative, got %v (hint: use 30s or larger)", c.MCP.ProtocolTimeout))
	} else if c.MCP.ProtocolTimeout > 10*time.Minute {
		errors = append(errors, fmt.Sprintf("MCP protocol timeout is very large: %v (hint: typically 30s-5m)", c.MCP.ProtocolTimeout))
	}
	
	if c.MCP.MaxTools < 1 {
		errors = append(errors, fmt.Sprintf("MCP max tools must be positive, got %d (hint: use 10-1000)", c.MCP.MaxTools))
	} else if c.MCP.MaxTools > 10000 {
		errors = append(errors, fmt.Sprintf("MCP max tools is very large: %d (hint: typically 10-1000)", c.MCP.MaxTools))
	}
	
	if c.MCP.MaxResources < 1 {
		errors = append(errors, fmt.Sprintf("MCP max resources must be positive, got %d (hint: use 10-1000)", c.MCP.MaxResources))
	} else if c.MCP.MaxResources > 10000 {
		errors = append(errors, fmt.Sprintf("MCP max resources is very large: %d (hint: typically 10-1000)", c.MCP.MaxResources))
	}
	
	if c.MCP.BufferSize < 1024 {
		errors = append(errors, fmt.Sprintf("MCP buffer size too small: %d (hint: use 4096 or larger)", c.MCP.BufferSize))
	} else if c.MCP.BufferSize > 1024*1024 {
		errors = append(errors, fmt.Sprintf("MCP buffer size very large: %d (hint: typically 4KB-64KB)", c.MCP.BufferSize))
	}
	
	// Resource cache validation
	if c.MCP.ResourceCache.DefaultTimeout < 0 {
		errors = append(errors, fmt.Sprintf("resource cache default timeout cannot be negative: %d", c.MCP.ResourceCache.DefaultTimeout))
	} else if c.MCP.ResourceCache.DefaultTimeout > 86400 {
		errors = append(errors, fmt.Sprintf("resource cache default timeout too large: %d seconds (hint: typically 300-3600 seconds)", c.MCP.ResourceCache.DefaultTimeout))
	}
	
	if c.MCP.ResourceCache.MaxSize < 0 {
		errors = append(errors, fmt.Sprintf("resource cache max size cannot be negative: %d", c.MCP.ResourceCache.MaxSize))
	} else if c.MCP.ResourceCache.MaxSize > 100000 {
		errors = append(errors, fmt.Sprintf("resource cache max size very large: %d (hint: typically 100-10000)", c.MCP.ResourceCache.MaxSize))
	}
	
	if len(errors) > 0 {
		return errors
	}
	return nil
}

func (c *Config) String() string {
	return fmt.Sprintf(`Configuration Summary:
Server: %s:%d (timeouts: read=%v, write=%v, idle=%v)
Logger: level=%s, format=%s, service=%s
MCP: timeout=%v, tools=%d, resources=%d, debug=%v
Resource Cache: enabled=%v, timeout=%ds, max_size=%d`,
		c.Server.Host, c.Server.Port,
		c.Server.ReadTimeout, c.Server.WriteTimeout, c.Server.IdleTimeout,
		c.Logger.Level, c.Logger.Format, c.Logger.Service,
		c.MCP.ProtocolTimeout, c.MCP.MaxTools, c.MCP.MaxResources, c.MCP.DebugMode,
		c.MCP.ResourceCache.Enabled, c.MCP.ResourceCache.DefaultTimeout, c.MCP.ResourceCache.MaxSize)
}

func (c *Config) ToJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config to JSON: %w", err)
	}
	return string(data), nil
}

func (c *Config) ValidationReport() string {
	if err := c.Validate(); err == nil {
		return "Configuration validation: PASSED"
	} else {
		return fmt.Sprintf("Configuration validation: FAILED\n%v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
