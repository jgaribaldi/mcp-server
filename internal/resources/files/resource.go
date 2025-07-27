package files

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
)

type FileSystemResource struct {
	uri         string
	filePath    string
	name        string
	description string
	mimeType    string
	size        int64
	modTime     time.Time
	permissions os.FileMode
	validator   *FilePathValidator
	logger      *logger.Logger
	handler     mcp.ResourceHandler
}

type FileSystemResourceConfig struct {
	FilePath           string
	Name               string
	Description        string
	ValidationConfig   ValidationConfig
	Logger            *logger.Logger
}

func NewFileSystemResource(config FileSystemResourceConfig) (*FileSystemResource, error) {
	if config.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	validator := NewFilePathValidator(config.ValidationConfig, config.Logger)

	fileInfo, err := validator.GetFileInfo(config.FilePath)
	if err != nil {
		config.Logger.Error("failed to get file info during resource creation", 
			"path", config.FilePath, "error", err)
		return nil, fmt.Errorf("failed to create file system resource: %w", err)
	}

	uri, err := generateFileURI(config.FilePath)
	if err != nil {
		config.Logger.Error("failed to generate file URI", 
			"path", config.FilePath, "error", err)
		return nil, fmt.Errorf("failed to generate file URI: %w", err)
	}

	name := config.Name
	if name == "" {
		name = filepath.Base(config.FilePath)
	}

	description := config.Description
	if description == "" {
		description = fmt.Sprintf("File resource: %s", filepath.Base(config.FilePath))
	}

	resource := &FileSystemResource{
		uri:         uri,
		filePath:    config.FilePath,
		name:        name,
		description: description,
		mimeType:    fileInfo.MimeType,
		size:        fileInfo.Size,
		modTime:     fileInfo.ModTime,
		permissions: fileInfo.Permissions,
		validator:   validator,
		logger:      config.Logger,
	}

	// Create handler after resource is initialized
	resource.handler = NewFileSystemResourceHandler(resource, config.Logger)

	config.Logger.Info("file system resource created successfully",
		"uri", uri,
		"path", config.FilePath,
		"size", fileInfo.Size,
		"mime_type", fileInfo.MimeType,
	)

	return resource, nil
}

func generateFileURI(filePath string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Convert to file:// URI
	fileURL := &url.URL{
		Scheme: "file",
		Path:   absPath,
	}

	return fileURL.String(), nil
}

func (r *FileSystemResource) URI() string {
	return r.uri
}

func (r *FileSystemResource) Name() string {
	return r.name
}

func (r *FileSystemResource) Description() string {
	return r.description
}

func (r *FileSystemResource) MimeType() string {
	return r.mimeType
}

func (r *FileSystemResource) Handler() mcp.ResourceHandler {
	return r.handler
}

func (r *FileSystemResource) GetFilePath() string {
	return r.filePath
}

func (r *FileSystemResource) GetSize() int64 {
	return r.size
}

func (r *FileSystemResource) GetModTime() time.Time {
	return r.modTime
}

func (r *FileSystemResource) GetPermissions() os.FileMode {
	return r.permissions
}

func (r *FileSystemResource) RefreshMetadata() error {
	r.logger.Debug("refreshing file metadata", "path", r.filePath)

	fileInfo, err := r.validator.GetFileInfo(r.filePath)
	if err != nil {
		r.logger.Error("failed to refresh file metadata", 
			"path", r.filePath, "error", err)
		return fmt.Errorf("failed to refresh metadata: %w", err)
	}

	// Update metadata
	r.mimeType = fileInfo.MimeType
	r.size = fileInfo.Size
	r.modTime = fileInfo.ModTime
	r.permissions = fileInfo.Permissions

	r.logger.Debug("file metadata refreshed successfully",
		"path", r.filePath,
		"size", r.size,
		"mime_type", r.mimeType,
	)

	return nil
}

func (r *FileSystemResource) Validate() error {
	return r.validator.ValidateFile(r.filePath)
}

func (r *FileSystemResource) IsAccessible() bool {
	return r.validator.ValidateFile(r.filePath) == nil
}

func (r *FileSystemResource) String() string {
	return fmt.Sprintf("FileSystemResource{URI: %s, Path: %s, Size: %d, MimeType: %s}", 
		r.uri, r.filePath, r.size, r.mimeType)
}

