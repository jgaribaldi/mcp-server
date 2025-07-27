package files

import (
	"context"
	"fmt"
	"os"
	"time"

	"mcp-server/internal/logger"
	"mcp-server/internal/mcp"
)

type FileSystemResourceHandler struct {
	resource *FileSystemResource
	logger   *logger.Logger
}

func NewFileSystemResourceHandler(resource *FileSystemResource, logger *logger.Logger) *FileSystemResourceHandler {
	return &FileSystemResourceHandler{
		resource: resource,
		logger:   logger,
	}
}

func (h *FileSystemResourceHandler) Read(ctx context.Context, uri string) (mcp.ResourceContent, error) {
	startTime := time.Now()
	
	h.logger.Info("file resource access initiated",
		"uri", uri,
		"resource_uri", h.resource.URI(),
	)

	// Validate the URI matches our resource
	if uri != h.resource.URI() {
		h.logger.Warn("file resource access denied - URI mismatch",
			"requested_uri", uri,
			"resource_uri", h.resource.URI(),
		)
		return nil, fmt.Errorf("%w: URI mismatch", ErrFilePermissionDenied)
	}

	// Validate file access through validator
	if err := h.resource.Validate(); err != nil {
		h.logger.Warn("file resource access denied - validation failed",
			"uri", uri,
			"error", err,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		return nil, h.convertValidationError(err)
	}

	// Read file content
	content, err := h.readFileContent(ctx)
	if err != nil {
		h.logger.Error("file resource read failed",
			"uri", uri,
			"error", err,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		return nil, err
	}

	h.logger.Info("file resource access successful",
		"uri", uri,
		"content_size", len(content.GetContent()),
		"mime_type", content.GetMimeType(),
		"duration_ms", time.Since(startTime).Milliseconds(),
	)

	return content, nil
}

func (h *FileSystemResourceHandler) readFileContent(ctx context.Context) (mcp.ResourceContent, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled during file read: %w", ctx.Err())
	default:
	}

	// Read file content with OS-level operations
	data, err := os.ReadFile(h.resource.GetFilePath())
	if err != nil {
		return nil, h.convertFileError(err)
	}

	// Create appropriate content based on MIME type
	var content mcp.Content
	mimeType := h.resource.MimeType()
	
	if h.isTextContent(mimeType) {
		content = &mcp.TextContent{Text: string(data)}
	} else {
		content = &mcp.BlobContent{Data: data}
	}

	// Create resource content
	resourceContent := &mcp.ResourceContentImpl{
		Content:  []mcp.Content{content},
		MimeType: mimeType,
	}

	return resourceContent, nil
}

func (h *FileSystemResourceHandler) isTextContent(mimeType string) bool {
	textTypes := map[string]bool{
		"text/plain":               true,
		"text/plain; charset=utf-8": true,
		"application/json":         true,
		"application/xml":          true,
		"application/x-yaml":       true,
		"text/csv":                true,
		"text/csv; charset=utf-8":  true,
		"text/html":               true,
		"text/html; charset=utf-8": true,
		"application/javascript":   true,
		"text/css":                true,
	}
	
	return textTypes[mimeType]
}

func (h *FileSystemResourceHandler) convertValidationError(err error) error {
	switch err {
	case ErrFileNotFound:
		return ErrFileNotFound
	case ErrFilePermissionDenied:
		return ErrFilePermissionDenied
	case ErrDirectoryTraversal:
		return ErrFilePermissionDenied // Don't expose traversal details
	case ErrDirectoryNotAllowed:
		return ErrFilePermissionDenied // Don't expose directory details
	case ErrFileTooBig:
		return ErrFileTooBig
	case ErrUnsupportedFileType:
		return ErrUnsupportedFileType
	case ErrInvalidFilePath:
		return ErrFilePermissionDenied // Don't expose path details
	default:
		// For unknown validation errors, return a generic permission error
		return ErrFilePermissionDenied
	}
}

func (h *FileSystemResourceHandler) convertFileError(err error) error {
	if os.IsNotExist(err) {
		return ErrFileNotFound
	}
	if os.IsPermission(err) {
		return ErrFilePermissionDenied
	}
	
	// For other file system errors, return a generic error without exposing details
	return fmt.Errorf("%w: file system error", ErrFilePermissionDenied)
}

func (h *FileSystemResourceHandler) GetResource() *FileSystemResource {
	return h.resource
}

func (h *FileSystemResourceHandler) CanHandle(uri string) bool {
	return uri == h.resource.URI()
}

func (h *FileSystemResourceHandler) String() string {
	return fmt.Sprintf("FileSystemResourceHandler{URI: %s, Path: %s}", 
		h.resource.URI(), h.resource.GetFilePath())
}