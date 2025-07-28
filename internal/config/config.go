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

const (
	DefaultServerHost     = "localhost"
	DefaultServerPort     = 3000
	
	DefaultReadTimeout    = 15 * time.Second
	DefaultWriteTimeout   = 15 * time.Second
	DefaultIdleTimeout    = 60 * time.Second
	DefaultMaxHeaderBytes = 1 << 20 // 1MB
	
	DefaultProtocolTimeout = 30 * time.Second
	DefaultMaxTools        = 100
	DefaultMaxResources    = 100
	DefaultDebugMode       = false
	DefaultEnableMetrics   = true
	DefaultBufferSize      = 4096
	
	DefaultFileResourceEnabled     = false
	DefaultFileResourceBaseDir     = "/tmp/mcp-files"
	DefaultFileResourceMaxSize     = 10 * 1024 * 1024 // 10MB
	DefaultFileResourceCacheTimeout = 5 * time.Minute
)

type Config struct {
	Server       ServerConfig
	Logger       LoggerConfig
	MCP          MCPConfig
	FileResource FileResourceConfig
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

type FileResourceConfig struct {
	Enabled            bool          `json:"enabled"`
	BaseDirectory      string        `json:"base_directory"`
	AllowedDirectories []string      `json:"allowed_directories"`
	MaxFileSize        int64         `json:"max_file_size_bytes"`
	AllowedExtensions  []string      `json:"allowed_extensions"`
	BlockedPatterns    []string      `json:"blocked_patterns"`
	CacheTimeout       time.Duration `json:"cache_timeout"`
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
	Server       FileServerConfig       `yaml:"server"`
	Logger       FileLoggerConfig       `yaml:"logger"`
	MCP          FileMCPConfig          `yaml:"mcp"`
	FileResource FileFileResourceConfig `yaml:"file_resource"`
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

type FileFileResourceConfig struct {
	Enabled            bool     `yaml:"enabled"`
	BaseDirectory      string   `yaml:"base_directory"`
	AllowedDirectories []string `yaml:"allowed_directories"`
	MaxFileSize        int64    `yaml:"max_file_size_bytes"`
	AllowedExtensions  []string `yaml:"allowed_extensions"`
	BlockedPatterns    []string `yaml:"blocked_patterns"`
	CacheTimeout       string   `yaml:"cache_timeout"`
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

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		if value == "" {
			return []string{}
		}
		return strings.Split(value, ",")
	}
	return defaultValue
}

func loadFromFile() (*FileConfig, error) {
	configPath := getEnv("MCP_CONFIG_FILE", "")
	if configPath == "" {
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
		return nil, nil
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

func loadFromEnvironment() *Config {
	return &Config{
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
		FileResource: FileResourceConfig{
			Enabled:       getEnvBool("MCP_FILE_RESOURCE_ENABLED", DefaultFileResourceEnabled),
			BaseDirectory: getEnv("MCP_FILE_RESOURCE_BASE_DIR", DefaultFileResourceBaseDir),
			AllowedDirectories: getEnvStringSlice("MCP_FILE_RESOURCE_ALLOWED_DIRS", []string{DefaultFileResourceBaseDir}),
			MaxFileSize:        getEnvInt64("MCP_FILE_RESOURCE_MAX_SIZE", DefaultFileResourceMaxSize),
			AllowedExtensions:  getEnvStringSlice("MCP_FILE_RESOURCE_ALLOWED_EXTS", []string{".txt", ".md", ".json", ".yaml", ".yml"}),
			BlockedPatterns:    getEnvStringSlice("MCP_FILE_RESOURCE_BLOCKED_PATTERNS", []string{".*", "~*", "*.tmp"}),
			CacheTimeout:       getEnvDuration("MCP_FILE_RESOURCE_CACHE_TIMEOUT", DefaultFileResourceCacheTimeout),
		},
	}
}

func mergeServerConfig(base *ServerConfig, file *FileServerConfig) {
	if file.Host != "" && os.Getenv("MCP_SERVER_HOST") == "" {
		base.Host = file.Host
	}
	if file.Port != 0 && os.Getenv("MCP_SERVER_PORT") == "" {
		base.Port = file.Port
	}
	if file.ReadTimeout != "" && os.Getenv("MCP_SERVER_READ_TIMEOUT") == "" {
		if duration, err := time.ParseDuration(file.ReadTimeout); err == nil {
			base.ReadTimeout = duration
		}
	}
	if file.WriteTimeout != "" && os.Getenv("MCP_SERVER_WRITE_TIMEOUT") == "" {
		if duration, err := time.ParseDuration(file.WriteTimeout); err == nil {
			base.WriteTimeout = duration
		}
	}
	if file.IdleTimeout != "" && os.Getenv("MCP_SERVER_IDLE_TIMEOUT") == "" {
		if duration, err := time.ParseDuration(file.IdleTimeout); err == nil {
			base.IdleTimeout = duration
		}
	}
	if file.MaxHeaderBytes != 0 && os.Getenv("MCP_SERVER_MAX_HEADER_BYTES") == "" {
		base.MaxHeaderBytes = file.MaxHeaderBytes
	}
}

func mergeLoggerConfig(base *LoggerConfig, file *FileLoggerConfig) {
	if file.Level != "" && os.Getenv("MCP_LOG_LEVEL") == "" {
		base.Level = file.Level
	}
	if file.Format != "" && os.Getenv("MCP_LOG_FORMAT") == "" {
		base.Format = file.Format
	}
	if file.Service != "" && os.Getenv("MCP_SERVICE_NAME") == "" {
		base.Service = file.Service
	}
	if file.Version != "" && os.Getenv("MCP_VERSION") == "" {
		base.Version = file.Version
	}
	if os.Getenv("MCP_LOG_USE_EMOJIS") == "" {
		base.UseEmojis = file.UseEmojis
	}
}

func mergeMCPConfig(base *MCPConfig, file *FileMCPConfig) {
	if file.ProtocolTimeout != "" && os.Getenv("MCP_PROTOCOL_TIMEOUT") == "" {
		if duration, err := time.ParseDuration(file.ProtocolTimeout); err == nil {
			base.ProtocolTimeout = duration
		}
	}
	if file.MaxTools != 0 && os.Getenv("MCP_MAX_TOOLS") == "" {
		base.MaxTools = file.MaxTools
	}
	if file.MaxResources != 0 && os.Getenv("MCP_MAX_RESOURCES") == "" {
		base.MaxResources = file.MaxResources
	}
	if os.Getenv("MCP_DEBUG_MODE") == "" {
		base.DebugMode = file.DebugMode
	}
	if os.Getenv("MCP_ENABLE_METRICS") == "" {
		base.EnableMetrics = file.EnableMetrics
	}
	if file.BufferSize != 0 && os.Getenv("MCP_BUFFER_SIZE") == "" {
		base.BufferSize = file.BufferSize
	}
}

func mergeResourceCacheConfig(base *ResourceCacheConfig, file *FileResourceCacheConfig) {
	if file.DefaultTimeout != 0 && os.Getenv("MCP_RESOURCE_CACHE_TIMEOUT") == "" {
		base.DefaultTimeout = file.DefaultTimeout
	}
	if file.MaxSize != 0 && os.Getenv("MCP_RESOURCE_CACHE_MAX_SIZE") == "" {
		base.MaxSize = file.MaxSize
	}
	if os.Getenv("MCP_RESOURCE_CACHE_ENABLED") == "" {
		base.Enabled = file.Enabled
	}
}

func mergeFileResourceConfig(base *FileResourceConfig, file *FileFileResourceConfig) {
	if os.Getenv("MCP_FILE_RESOURCE_ENABLED") == "" {
		base.Enabled = file.Enabled
	}
	if file.BaseDirectory != "" && os.Getenv("MCP_FILE_RESOURCE_BASE_DIR") == "" {
		base.BaseDirectory = file.BaseDirectory
	}
	if len(file.AllowedDirectories) > 0 && os.Getenv("MCP_FILE_RESOURCE_ALLOWED_DIRS") == "" {
		base.AllowedDirectories = file.AllowedDirectories
	}
	if file.MaxFileSize != 0 && os.Getenv("MCP_FILE_RESOURCE_MAX_SIZE") == "" {
		base.MaxFileSize = file.MaxFileSize
	}
	if len(file.AllowedExtensions) > 0 && os.Getenv("MCP_FILE_RESOURCE_ALLOWED_EXTS") == "" {
		base.AllowedExtensions = file.AllowedExtensions
	}
	if len(file.BlockedPatterns) > 0 && os.Getenv("MCP_FILE_RESOURCE_BLOCKED_PATTERNS") == "" {
		base.BlockedPatterns = file.BlockedPatterns
	}
	if file.CacheTimeout != "" && os.Getenv("MCP_FILE_RESOURCE_CACHE_TIMEOUT") == "" {
		if duration, err := time.ParseDuration(file.CacheTimeout); err == nil {
			base.CacheTimeout = duration
		}
	}
}

func mergeConfigs(base *Config, file *FileConfig) *Config {
	if file == nil {
		return base
	}
	
	result := *base
	
	mergeServerConfig(&result.Server, &file.Server)
	mergeLoggerConfig(&result.Logger, &file.Logger)
	mergeMCPConfig(&result.MCP, &file.MCP)
	mergeResourceCacheConfig(&result.MCP.ResourceCache, &file.MCP.ResourceCache)
	mergeFileResourceConfig(&result.FileResource, &file.FileResource)
	
	return &result
}

func validateServerConfig(cfg *ServerConfig) ValidationErrors {
	var errors ValidationErrors
	
	if cfg.Host == "" {
		errors = append(errors, "server host cannot be empty (hint: use 'localhost' for local development)")
	}
	
	if cfg.Port < 1 || cfg.Port > 65535 {
		errors = append(errors, fmt.Sprintf("server port must be between 1 and 65535, got %d (hint: use 3000 for development, 8080 for production)", cfg.Port))
	}
	
	if cfg.ReadTimeout < 0 {
		errors = append(errors, fmt.Sprintf("server read timeout cannot be negative, got %v (hint: use 15s or larger)", cfg.ReadTimeout))
	} else if cfg.ReadTimeout > 5*time.Minute {
		errors = append(errors, fmt.Sprintf("server read timeout is very large: %v (hint: typically 15s-60s)", cfg.ReadTimeout))
	}
	
	if cfg.WriteTimeout < 0 {
		errors = append(errors, fmt.Sprintf("server write timeout cannot be negative, got %v (hint: use 15s or larger)", cfg.WriteTimeout))
	} else if cfg.WriteTimeout > 5*time.Minute {
		errors = append(errors, fmt.Sprintf("server write timeout is very large: %v (hint: typically 15s-60s)", cfg.WriteTimeout))
	}
	
	if cfg.IdleTimeout < 0 {
		errors = append(errors, fmt.Sprintf("server idle timeout cannot be negative, got %v (hint: use 60s or larger)", cfg.IdleTimeout))
	}
	
	if cfg.ReadTimeout > 0 && cfg.IdleTimeout > 0 && cfg.ReadTimeout >= cfg.IdleTimeout {
		errors = append(errors, fmt.Sprintf("read timeout (%v) should be less than idle timeout (%v)", cfg.ReadTimeout, cfg.IdleTimeout))
	}
	
	if cfg.WriteTimeout > 0 && cfg.IdleTimeout > 0 && cfg.WriteTimeout >= cfg.IdleTimeout {
		errors = append(errors, fmt.Sprintf("write timeout (%v) should be less than idle timeout (%v)", cfg.WriteTimeout, cfg.IdleTimeout))
	}
	
	if cfg.MaxHeaderBytes < 1 {
		errors = append(errors, fmt.Sprintf("server max header bytes must be positive, got %d (hint: use 1048576 for 1MB)", cfg.MaxHeaderBytes))
	} else if cfg.MaxHeaderBytes > 10*1024*1024 {
		errors = append(errors, fmt.Sprintf("server max header bytes is very large: %d (hint: typically 1MB-8MB)", cfg.MaxHeaderBytes))
	}
	
	return errors
}

func validateLoggerConfig(cfg *LoggerConfig) ValidationErrors {
	var errors ValidationErrors
	
	normalizedLevel := strings.ToLower(cfg.Level)
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[normalizedLevel] {
		errors = append(errors, fmt.Sprintf("invalid log level: %s (valid options: debug, info, warn, error)", cfg.Level))
	}
	
	validFormats := map[string]bool{"json": true, "text": true, "console": true}
	if !validFormats[cfg.Format] {
		errors = append(errors, fmt.Sprintf("invalid log format: %s (valid options: json, text, console)", cfg.Format))
	}
	
	return errors
}

func validateMCPConfig(cfg *MCPConfig) ValidationErrors {
	var errors ValidationErrors
	
	if cfg.ProtocolTimeout < 0 {
		errors = append(errors, fmt.Sprintf("MCP protocol timeout cannot be negative, got %v (hint: use 30s or larger)", cfg.ProtocolTimeout))
	} else if cfg.ProtocolTimeout > 10*time.Minute {
		errors = append(errors, fmt.Sprintf("MCP protocol timeout is very large: %v (hint: typically 30s-5m)", cfg.ProtocolTimeout))
	}
	
	if cfg.MaxTools < 1 {
		errors = append(errors, fmt.Sprintf("MCP max tools must be positive, got %d (hint: use 10-1000)", cfg.MaxTools))
	} else if cfg.MaxTools > 10000 {
		errors = append(errors, fmt.Sprintf("MCP max tools is very large: %d (hint: typically 10-1000)", cfg.MaxTools))
	}
	
	if cfg.MaxResources < 1 {
		errors = append(errors, fmt.Sprintf("MCP max resources must be positive, got %d (hint: use 10-1000)", cfg.MaxResources))
	} else if cfg.MaxResources > 10000 {
		errors = append(errors, fmt.Sprintf("MCP max resources is very large: %d (hint: typically 10-1000)", cfg.MaxResources))
	}
	
	if cfg.BufferSize < 1024 {
		errors = append(errors, fmt.Sprintf("MCP buffer size too small: %d (hint: use 4096 or larger)", cfg.BufferSize))
	} else if cfg.BufferSize > 1024*1024 {
		errors = append(errors, fmt.Sprintf("MCP buffer size very large: %d (hint: typically 4KB-64KB)", cfg.BufferSize))
	}
	
	return errors
}

func validateResourceCacheConfig(cfg *ResourceCacheConfig) ValidationErrors {
	var errors ValidationErrors
	
	if cfg.DefaultTimeout < 0 {
		errors = append(errors, fmt.Sprintf("resource cache default timeout cannot be negative: %d", cfg.DefaultTimeout))
	} else if cfg.DefaultTimeout > 86400 {
		errors = append(errors, fmt.Sprintf("resource cache default timeout too large: %d seconds (hint: typically 300-3600 seconds)", cfg.DefaultTimeout))
	}
	
	if cfg.MaxSize < 0 {
		errors = append(errors, fmt.Sprintf("resource cache max size cannot be negative: %d", cfg.MaxSize))
	} else if cfg.MaxSize > 100000 {
		errors = append(errors, fmt.Sprintf("resource cache max size very large: %d (hint: typically 100-10000)", cfg.MaxSize))
	}
	
	return errors
}

func validateFileResourceConfig(cfg *FileResourceConfig) ValidationErrors {
	var errors ValidationErrors
	
	if cfg.BaseDirectory == "" {
		errors = append(errors, "file resource base directory cannot be empty (hint: use '/tmp/mcp-files' for development)")
	} else if strings.Contains(cfg.BaseDirectory, "..") {
		errors = append(errors, "file resource base directory cannot contain '..' (hint: use absolute paths only)")
	}
	
	if cfg.MaxFileSize < 0 {
		errors = append(errors, fmt.Sprintf("file resource max file size cannot be negative: %d", cfg.MaxFileSize))
	} else if cfg.MaxFileSize > 100*1024*1024 {
		errors = append(errors, fmt.Sprintf("file resource max file size very large: %d bytes (hint: typically 1MB-50MB)", cfg.MaxFileSize))
	}
	
	if cfg.CacheTimeout < 0 {
		errors = append(errors, fmt.Sprintf("file resource cache timeout cannot be negative: %v", cfg.CacheTimeout))
	} else if cfg.CacheTimeout > time.Hour {
		errors = append(errors, fmt.Sprintf("file resource cache timeout very large: %v (hint: typically 1m-30m)", cfg.CacheTimeout))
	}
	
	for _, dir := range cfg.AllowedDirectories {
		if strings.Contains(dir, "..") {
			errors = append(errors, fmt.Sprintf("allowed directory cannot contain '..': %s (hint: use absolute paths only)", dir))
		}
	}
	
	return errors
}

func validateConfig(cfg *Config) error {
	var allErrors ValidationErrors
	
	allErrors = append(allErrors, validateServerConfig(&cfg.Server)...)
	allErrors = append(allErrors, validateLoggerConfig(&cfg.Logger)...)
	allErrors = append(allErrors, validateMCPConfig(&cfg.MCP)...)
	allErrors = append(allErrors, validateResourceCacheConfig(&cfg.MCP.ResourceCache)...)
	allErrors = append(allErrors, validateFileResourceConfig(&cfg.FileResource)...)
	
	if len(allErrors) > 0 {
		return allErrors
	}
	return nil
}

func Load() (*Config, error) {
	cfg := loadFromEnvironment()
	
	fileConfig, err := loadFromFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config file: %v\n", err)
	}
	
	cfg = mergeConfigs(cfg, fileConfig)
	
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return cfg, nil
}

func (c *Config) Validate() error {
	return validateConfig(c)
}

func (c *Config) String() string {
	return fmt.Sprintf(`Configuration Summary:
Server: %s:%d (timeouts: read=%v, write=%v, idle=%v)
Logger: level=%s, format=%s, service=%s
MCP: timeout=%v, tools=%d, resources=%d, debug=%v
Resource Cache: enabled=%v, timeout=%ds, max_size=%d
File Resource: enabled=%v, base_dir=%s, max_size=%d, cache_timeout=%v`,
		c.Server.Host, c.Server.Port,
		c.Server.ReadTimeout, c.Server.WriteTimeout, c.Server.IdleTimeout,
		c.Logger.Level, c.Logger.Format, c.Logger.Service,
		c.MCP.ProtocolTimeout, c.MCP.MaxTools, c.MCP.MaxResources, c.MCP.DebugMode,
		c.MCP.ResourceCache.Enabled, c.MCP.ResourceCache.DefaultTimeout, c.MCP.ResourceCache.MaxSize,
		c.FileResource.Enabled, c.FileResource.BaseDirectory, c.FileResource.MaxFileSize, c.FileResource.CacheTimeout)
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