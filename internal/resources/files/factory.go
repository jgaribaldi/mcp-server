package files

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
	"mcp-server/internal/resources"
)

type FileSystemResourceFactory struct {
	name            string
	description     string
	version         string
	baseURI         string
	basePath        string
	allowedDirs     []string
	maxFileSize     int64
	allowedExts     []string
	blockedPatterns []string
	logger          *logger.Logger
}

type FileSystemFactoryConfig struct {
	Name               string
	Description        string
	Version            string
	BaseURI            string
	BasePath           string
	AllowedDirectories []string
	MaxFileSize        int64
	AllowedExtensions  []string
	BlockedPatterns    []string
	Logger            *logger.Logger
}

func NewFileSystemResourceFactory(config FileSystemFactoryConfig) (*FileSystemResourceFactory, error) {
	if config.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	if config.Name == "" {
		config.Name = "file-system"
	}

	if config.Description == "" {
		config.Description = "File system resource factory for secure file access"
	}

	if config.Version == "" {
		config.Version = "1.0.0"
	}

	if config.BaseURI == "" {
		config.BaseURI = "file://"
	}

	if config.MaxFileSize <= 0 {
		config.MaxFileSize = 10 * 1024 * 1024 // 10MB default
	}

	factory := &FileSystemResourceFactory{
		name:            config.Name,
		description:     config.Description,
		version:         config.Version,
		baseURI:         config.BaseURI,
		basePath:        config.BasePath,
		allowedDirs:     config.AllowedDirectories,
		maxFileSize:     config.MaxFileSize,
		allowedExts:     config.AllowedExtensions,
		blockedPatterns: config.BlockedPatterns,
		logger:          config.Logger,
	}

	// Validate factory configuration
	if err := factory.validateFactoryConfig(); err != nil {
		config.Logger.Error("factory configuration validation failed", "error", err)
		return nil, fmt.Errorf("invalid factory configuration: %w", err)
	}

	config.Logger.Info("file system resource factory created successfully",
		"name", factory.name,
		"version", factory.version,
		"allowed_dirs", len(factory.allowedDirs),
		"max_file_size", factory.maxFileSize,
	)

	return factory, nil
}

func (f *FileSystemResourceFactory) URI() string {
	return f.baseURI
}

func (f *FileSystemResourceFactory) Name() string {
	return f.name
}

func (f *FileSystemResourceFactory) Description() string {
	return f.description
}

func (f *FileSystemResourceFactory) MimeType() string {
	return "application/octet-stream" // Default for unknown file types
}

func (f *FileSystemResourceFactory) Version() string {
	return f.version
}

func (f *FileSystemResourceFactory) Tags() []string {
	return []string{"filesystem", "file", "resource", "secure"}
}

func (f *FileSystemResourceFactory) Capabilities() []string {
	return []string{
		"read-file",
		"validate-path",
		"security-validation",
		"audit-logging",
		"mime-detection",
	}
}

func (f *FileSystemResourceFactory) Create(ctx context.Context, config resources.ResourceConfig) (mcp.Resource, error) {
	f.logger.Debug("creating file system resource",
		"enabled", config.Enabled,
		"config_keys", len(config.Config),
	)

	if !config.Enabled {
		return nil, fmt.Errorf("resource creation disabled in configuration")
	}

	// Extract file path from configuration
	filePath, err := f.extractFilePath(config)
	if err != nil {
		f.logger.Error("failed to extract file path from configuration", "error", err)
		return nil, fmt.Errorf("invalid file path configuration: %w", err)
	}

	// Create validation configuration
	validationConfig := f.createValidationConfig(config)

	// Create resource configuration
	resourceConfig := FileSystemResourceConfig{
		FilePath:         filePath,
		Name:             f.extractResourceName(config, filePath),
		Description:      f.extractResourceDescription(config, filePath),
		ValidationConfig: validationConfig,
		Logger:          f.logger,
	}

	// Create the resource
	resource, err := NewFileSystemResource(resourceConfig)
	if err != nil {
		f.logger.Error("failed to create file system resource",
			"file_path", filePath,
			"error", err,
		)
		return nil, fmt.Errorf("resource creation failed: %w", err)
	}

	f.logger.Info("file system resource created successfully via factory",
		"uri", resource.URI(),
		"file_path", filePath,
		"name", resource.Name(),
	)

	return resource, nil
}

func (f *FileSystemResourceFactory) Validate(config resources.ResourceConfig) error {
	f.logger.Debug("validating resource configuration")

	if !config.Enabled {
		return nil // Skip validation for disabled resources
	}

	// Validate file path
	filePath, err := f.extractFilePath(config)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	// Create temporary validator for configuration validation
	validationConfig := f.createValidationConfig(config)
	validator := NewFilePathValidator(validationConfig, f.logger)

	// Validate the file path
	if err := validator.ValidatePath(filePath); err != nil {
		f.logger.Warn("configuration validation failed",
			"file_path", filePath,
			"error", err,
		)
		return fmt.Errorf("path validation failed: %w", err)
	}

	f.logger.Debug("resource configuration validation successful", "file_path", filePath)
	return nil
}

func (f *FileSystemResourceFactory) validateFactoryConfig() error {
	if f.name == "" {
		return fmt.Errorf("factory name cannot be empty")
	}

	if f.description == "" {
		return fmt.Errorf("factory description cannot be empty")
	}

	if f.version == "" {
		return fmt.Errorf("factory version cannot be empty")
	}

	if f.baseURI == "" {
		return fmt.Errorf("factory base URI cannot be empty")
	}

	if f.maxFileSize <= 0 {
		return fmt.Errorf("max file size must be positive, got %d", f.maxFileSize)
	}

	if f.maxFileSize > 100*1024*1024 { // 100MB limit
		return fmt.Errorf("max file size too large: %d bytes (limit: 100MB)", f.maxFileSize)
	}

	// Validate allowed directories exist if specified
	for _, dir := range f.allowedDirs {
		if dir == "" {
			return fmt.Errorf("allowed directory cannot be empty")
		}
		if !filepath.IsAbs(dir) {
			return fmt.Errorf("allowed directory must be absolute path: %s", dir)
		}
	}

	// Validate allowed extensions format
	for _, ext := range f.allowedExts {
		if ext == "" {
			return fmt.Errorf("allowed extension cannot be empty")
		}
		if !strings.HasPrefix(ext, ".") {
			return fmt.Errorf("allowed extension must start with dot: %s", ext)
		}
	}

	return nil
}

func (f *FileSystemResourceFactory) extractFilePath(config resources.ResourceConfig) (string, error) {
	// Try to get file path from config
	if pathVal, exists := config.Config["file_path"]; exists {
		if path, ok := pathVal.(string); ok {
			if path == "" {
				return "", fmt.Errorf("file path cannot be empty")
			}
			return path, nil
		}
		return "", fmt.Errorf("file path must be a string")
	}

	// Try alternative key names
	if pathVal, exists := config.Config["path"]; exists {
		if path, ok := pathVal.(string); ok {
			if path == "" {
				return "", fmt.Errorf("file path cannot be empty")
			}
			return path, nil
		}
		return "", fmt.Errorf("path must be a string")
	}

	return "", fmt.Errorf("file path not specified in configuration (use 'file_path' or 'path' key)")
}

func (f *FileSystemResourceFactory) extractResourceName(config resources.ResourceConfig, filePath string) string {
	if nameVal, exists := config.Config["name"]; exists {
		if name, ok := nameVal.(string); ok && name != "" {
			return name
		}
	}

	// Default to filename
	return filepath.Base(filePath)
}

func (f *FileSystemResourceFactory) extractResourceDescription(config resources.ResourceConfig, filePath string) string {
	if descVal, exists := config.Config["description"]; exists {
		if desc, ok := descVal.(string); ok && desc != "" {
			return desc
		}
	}

	// Default description
	return fmt.Sprintf("File resource: %s", filepath.Base(filePath))
}

func (f *FileSystemResourceFactory) createValidationConfig(config resources.ResourceConfig) ValidationConfig {
	validationConfig := ValidationConfig{
		AllowedDirectories: f.allowedDirs,
		MaxFileSize:        f.maxFileSize,
		AllowedExtensions:  f.allowedExts,
		BlockedPatterns:    f.blockedPatterns,
	}

	// Override with resource-specific settings if provided
	if dirsVal, exists := config.Config["allowed_directories"]; exists {
		if dirs, ok := dirsVal.([]string); ok {
			validationConfig.AllowedDirectories = dirs
		}
	}

	if sizeVal, exists := config.Config["max_file_size"]; exists {
		if size, ok := sizeVal.(int64); ok && size > 0 {
			validationConfig.MaxFileSize = size
		} else if sizeFloat, ok := sizeVal.(float64); ok && sizeFloat > 0 {
			validationConfig.MaxFileSize = int64(sizeFloat)
		}
	}

	if extsVal, exists := config.Config["allowed_extensions"]; exists {
		if exts, ok := extsVal.([]string); ok {
			validationConfig.AllowedExtensions = exts
		}
	}

	if patternsVal, exists := config.Config["blocked_patterns"]; exists {
		if patterns, ok := patternsVal.([]string); ok {
			validationConfig.BlockedPatterns = patterns
		}
	}

	return validationConfig
}

func (f *FileSystemResourceFactory) GetAllowedDirectories() []string {
	return f.allowedDirs
}

func (f *FileSystemResourceFactory) GetMaxFileSize() int64 {
	return f.maxFileSize
}

func (f *FileSystemResourceFactory) GetAllowedExtensions() []string {
	return f.allowedExts
}

func (f *FileSystemResourceFactory) GetBlockedPatterns() []string {
	return f.blockedPatterns
}

func (f *FileSystemResourceFactory) SupportsPath(path string) bool {
	if f.basePath != "" {
		absBasePath, err := filepath.Abs(f.basePath)
		if err != nil {
			return false
		}
		
		absPath, err := filepath.Abs(path)
		if err != nil {
			return false
		}
		
		rel, err := filepath.Rel(absBasePath, absPath)
		if err != nil {
			return false
		}
		
		return !strings.HasPrefix(rel, "..")
	}

	// If no base path specified, use allowed directories
	if len(f.allowedDirs) == 0 {
		return true // No restrictions
	}

	for _, allowedDir := range f.allowedDirs {
		allowedAbs, err := filepath.Abs(allowedDir)
		if err != nil {
			continue
		}
		
		pathAbs, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		
		rel, err := filepath.Rel(allowedAbs, pathAbs)
		if err != nil {
			continue
		}
		
		if !strings.HasPrefix(rel, "..") {
			return true
		}
	}

	return false
}

type FileSystemHealthCheck struct {
	Status              string                  `json:"status"`
	Timestamp           string                  `json:"timestamp"`
	DirectoryHealth     map[string]DirectoryStatus `json:"directory_health"`
	PermissionHealth    map[string]bool         `json:"permission_health"`
	DiskSpaceWarnings   []string               `json:"disk_space_warnings,omitempty"`
	ConfigurationIssues []string               `json:"configuration_issues,omitempty"`
	OverallAccessible   bool                   `json:"overall_accessible"`
}

type DirectoryStatus struct {
	Exists     bool   `json:"exists"`
	Readable   bool   `json:"readable"`
	Writable   bool   `json:"writable"`
	ErrorMsg   string `json:"error_msg,omitempty"`
}

type DirectoryHealthChecker struct {
	logger *logger.Logger
}

func NewDirectoryHealthChecker(log *logger.Logger) *DirectoryHealthChecker {
	return &DirectoryHealthChecker{logger: log}
}

func (d *DirectoryHealthChecker) CheckDirectories(basePath string, allowedDirs []string) map[string]DirectoryStatus {
	directoryHealth := make(map[string]DirectoryStatus)
	
	if basePath != "" {
		directoryHealth[basePath] = d.checkSingleDirectory(basePath)
	}
	
	for _, dir := range allowedDirs {
		directoryHealth[dir] = d.checkSingleDirectory(dir)
	}
	
	return directoryHealth
}

func (d *DirectoryHealthChecker) CheckPermissions(directoryResults map[string]DirectoryStatus) map[string]bool {
	permissionHealth := make(map[string]bool)
	
	for path, status := range directoryResults {
		permissionHealth[path] = status.Readable && status.Writable
	}
	
	return permissionHealth
}

func (d *DirectoryHealthChecker) checkSingleDirectory(dirPath string) DirectoryStatus {
	status := DirectoryStatus{
		Exists:   false,
		Readable: false,
		Writable: false,
	}

	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			status.ErrorMsg = "directory does not exist"
		} else {
			status.ErrorMsg = fmt.Sprintf("stat error: %v", err)
		}
		return status
	}

	status.Exists = true

	if !info.IsDir() {
		status.ErrorMsg = "path exists but is not a directory"
		return status
	}

	if d.testReadAccess(dirPath) {
		status.Readable = true
	} else {
		status.ErrorMsg = "directory not readable"
	}

	if d.testWriteAccess(dirPath) {
		status.Writable = true
	} else if status.ErrorMsg == "" {
		status.ErrorMsg = "directory not writable"
	}

	return status
}

func (d *DirectoryHealthChecker) testReadAccess(dirPath string) bool {
	_, err := os.ReadDir(dirPath)
	return err == nil
}

func (d *DirectoryHealthChecker) testWriteAccess(dirPath string) bool {
	info, err := os.Stat(dirPath)
	if err != nil {
		return false
	}

	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		uid := os.Getuid()
		gid := os.Getgid()
		mode := info.Mode()

		if uint32(uid) == stat.Uid {
			return mode&0200 != 0
		}

		if uint32(gid) == stat.Gid {
			return mode&0020 != 0
		}

		return mode&0002 != 0
	}

	tempFile := filepath.Join(dirPath, ".mcp_health_test")
	file, err := os.Create(tempFile)
	if err != nil {
		return false
	}
	file.Close()
	os.Remove(tempFile)
	return true
}

type ConfigurationValidator struct{}

func NewConfigurationValidator() *ConfigurationValidator {
	return &ConfigurationValidator{}
}

func (c *ConfigurationValidator) ValidateDirectoryConfig(basePath string, allowedDirs []string) []string {
	var issues []string
	
	if len(allowedDirs) == 0 && basePath == "" {
		issues = append(issues, "No allowed directories or base path configured - unrestricted access")
	}
	
	return issues
}

func (c *ConfigurationValidator) ValidateFileSizeConfig(maxFileSize int64) ([]string, []string) {
	var warnings []string
	var issues []string
	
	if maxFileSize <= 0 {
		issues = append(issues, "Invalid max file size configuration")
	}
	
	if maxFileSize > 100*1024*1024 { // 100MB
		warnings = append(warnings, fmt.Sprintf("Large max file size configured: %d bytes", maxFileSize))
	}
	
	return warnings, issues
}

type HealthStatusEvaluator struct{}

func NewHealthStatusEvaluator() *HealthStatusEvaluator {
	return &HealthStatusEvaluator{}
}

func (h *HealthStatusEvaluator) EvaluateOverallStatus(directoryHealth map[string]DirectoryStatus, configIssues []string) string {
	if len(configIssues) > 0 {
		return "degraded"
	}
	
	for _, status := range directoryHealth {
		if !status.Exists || !status.Readable {
			return "degraded"
		}
	}
	
	return "healthy"
}

func (h *HealthStatusEvaluator) DetermineAccessibility(directoryHealth map[string]DirectoryStatus) bool {
	for _, status := range directoryHealth {
		if !status.Exists || !status.Readable {
			return false
		}
	}
	return true
}

func (f *FileSystemResourceFactory) HealthCheck() FileSystemHealthCheck {
	dirChecker := NewDirectoryHealthChecker(f.logger)
	configValidator := NewConfigurationValidator()
	statusEvaluator := NewHealthStatusEvaluator()
	
	directoryHealth := dirChecker.CheckDirectories(f.basePath, f.allowedDirs)
	permissionHealth := dirChecker.CheckPermissions(directoryHealth)
	configIssues := configValidator.ValidateDirectoryConfig(f.basePath, f.allowedDirs)
	diskWarnings, sizeIssues := configValidator.ValidateFileSizeConfig(f.maxFileSize)
	
	allConfigIssues := append(configIssues, sizeIssues...)
	
	status := statusEvaluator.EvaluateOverallStatus(directoryHealth, allConfigIssues)
	accessible := statusEvaluator.DetermineAccessibility(directoryHealth)

	f.logger.Debug("file system factory health check completed",
		"status", status,
		"accessible", accessible,
		"directories_checked", len(directoryHealth),
		"config_issues", len(allConfigIssues),
	)

	return FileSystemHealthCheck{
		Status:              status,
		Timestamp:           time.Now().UTC().Format(time.RFC3339),
		DirectoryHealth:     directoryHealth,
		PermissionHealth:    permissionHealth,
		DiskSpaceWarnings:   diskWarnings,
		ConfigurationIssues: allConfigIssues,
		OverallAccessible:   accessible,
	}
}


func (f *FileSystemResourceFactory) String() string {
	return fmt.Sprintf("FileSystemResourceFactory{Name: %s, Version: %s, BaseURI: %s, AllowedDirs: %d}", 
		f.name, f.version, f.baseURI, len(f.allowedDirs))
}