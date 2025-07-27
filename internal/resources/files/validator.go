package files

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mcp-server/internal/logger"
)

type FilePathValidator struct {
	allowedDirectories []string
	maxFileSize        int64
	allowedExtensions  []string
	blockedPatterns    []string
	logger            *logger.Logger
}

type ValidationConfig struct {
	AllowedDirectories []string
	MaxFileSize        int64
	AllowedExtensions  []string
	BlockedPatterns    []string
}

func NewFilePathValidator(cfg ValidationConfig, log *logger.Logger) *FilePathValidator {
	return &FilePathValidator{
		allowedDirectories: cfg.AllowedDirectories,
		maxFileSize:        cfg.MaxFileSize,
		allowedExtensions:  cfg.AllowedExtensions,
		blockedPatterns:    cfg.BlockedPatterns,
		logger:            log,
	}
}

func (v *FilePathValidator) ValidatePath(path string) error {
	if err := v.CheckDirectoryTraversal(path); err != nil {
		v.logger.Warn("directory traversal attempt blocked", "path", path)
		return err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		v.logger.Error("failed to resolve absolute path", "path", path, "error", err)
		return fmt.Errorf("%w: %v", ErrInvalidFilePath, err)
	}

	if err := v.CheckAllowedDirectories(absPath); err != nil {
		v.logger.Warn("access to disallowed directory blocked", "path", absPath)
		return err
	}

	if err := v.CheckBlockedPatterns(absPath); err != nil {
		v.logger.Warn("blocked pattern detected in path", "path", absPath)
		return err
	}

	if err := v.CheckFileExtension(absPath); err != nil {
		v.logger.Warn("unsupported file extension", "path", absPath)
		return err
	}

	return nil
}

func (v *FilePathValidator) CheckDirectoryTraversal(path string) error {
	cleanPath := filepath.Clean(path)
	
	if strings.Contains(cleanPath, "..") {
		return ErrDirectoryTraversal
	}

	if strings.Contains(path, "../") || strings.Contains(path, "..\\") {
		return ErrDirectoryTraversal
	}

	if strings.Contains(path, "/.") || strings.Contains(path, "\\.") {
		return ErrDirectoryTraversal
	}

	return nil
}

func (v *FilePathValidator) CheckAllowedDirectories(absPath string) error {
	if len(v.allowedDirectories) == 0 {
		return nil
	}

	for _, allowedDir := range v.allowedDirectories {
		allowedAbs, err := filepath.Abs(allowedDir)
		if err != nil {
			v.logger.Error("failed to resolve allowed directory", "dir", allowedDir, "error", err)
			continue
		}

		rel, err := filepath.Rel(allowedAbs, absPath)
		if err != nil {
			continue
		}

		if !strings.HasPrefix(rel, "..") {
			return nil
		}
	}

	return ErrDirectoryNotAllowed
}

func (v *FilePathValidator) CheckBlockedPatterns(path string) error {
	for _, pattern := range v.blockedPatterns {
		if strings.Contains(strings.ToLower(path), strings.ToLower(pattern)) {
			return ErrInvalidFilePath
		}
	}
	return nil
}

func (v *FilePathValidator) CheckFileExtension(path string) error {
	if len(v.allowedExtensions) == 0 {
		return nil
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return ErrUnsupportedFileType
	}

	for _, allowedExt := range v.allowedExtensions {
		if strings.ToLower(allowedExt) == ext {
			return nil
		}
	}

	return ErrUnsupportedFileType
}

func (v *FilePathValidator) CheckFileSize(path string) error {
	if v.maxFileSize <= 0 {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrFileNotFound
		}
		if os.IsPermission(err) {
			return ErrFilePermissionDenied
		}
		return fmt.Errorf("%w: %v", ErrInvalidFilePath, err)
	}

	if info.Size() > v.maxFileSize {
		return ErrFileTooBig
	}

	return nil
}

func (v *FilePathValidator) ValidateFile(path string) error {
	if err := v.ValidatePath(path); err != nil {
		return err
	}

	if err := v.CheckFileSize(path); err != nil {
		return err
	}

	return nil
}

func (v *FilePathValidator) DetectMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	
	mimeType := mime.TypeByExtension(ext)
	if mimeType != "" {
		return mimeType
	}

	switch ext {
	case ".txt", ".md", ".log":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".yaml", ".yml":
		return "application/x-yaml"
	case ".csv":
		return "text/csv"
	case ".html", ".htm":
		return "text/html"
	case ".js":
		return "application/javascript"
	case ".css":
		return "text/css"
	default:
		return "application/octet-stream"
	}
}

type FileInfo struct {
	Path        string
	Size        int64
	ModTime     time.Time
	Permissions os.FileMode
	MimeType    string
}

func (v *FilePathValidator) GetFileInfo(path string) (*FileInfo, error) {
	if err := v.ValidateFile(path); err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		if os.IsPermission(err) {
			return nil, ErrFilePermissionDenied
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidFilePath, err)
	}

	if info.IsDir() {
		return nil, ErrInvalidFilePath
	}

	return &FileInfo{
		Path:        path,
		Size:        info.Size(),
		ModTime:     info.ModTime(),
		Permissions: info.Mode(),
		MimeType:    v.DetectMimeType(path),
	}, nil
}