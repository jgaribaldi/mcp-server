package files

import "fmt"

var (
	ErrFileNotFound         = fmt.Errorf("file not found")
	ErrFilePermissionDenied = fmt.Errorf("file permission denied")
	ErrInvalidFilePath      = fmt.Errorf("invalid file path")
	ErrDirectoryTraversal   = fmt.Errorf("directory traversal attempt")
	ErrFileTooBig          = fmt.Errorf("file size exceeds limit")
	ErrUnsupportedFileType = fmt.Errorf("unsupported file type")
	ErrDirectoryNotAllowed = fmt.Errorf("directory not in allowed list")
)