# Commit 9: First Example Resource

## Overview
This specification defines the implementation of the first concrete resource example - a file system resource that allows MCP clients to access files through the standardized resource protocol. This implementation serves as a reference for other resource types and demonstrates the practical usage of the resource framework established in Commit 8.

## Background and Context
The file system resource provides read-only access to files on the server's local filesystem through MCP URI patterns. This implementation emphasizes security through permission checking, audit logging, and path validation to prevent unauthorized access or directory traversal attacks.

## Architecture Overview

### URI Pattern
File system resources use the standard file URI scheme:
```
file:///absolute/path/to/file.txt
file:///etc/config/app.json
file:///var/log/application.log
```

### Security Model
- Only absolute paths are allowed (no relative paths)
- Path validation prevents directory traversal (../, ..\)
- Configurable access control lists for allowed directories
- All file access operations are audit logged
- Permission checking respects file system permissions

## Core Components

### 1. File System Resource (`internal/resources/files/resource.go`)

#### FileSystemResource Structure
```go
type FileSystemResource struct {
    URI         string
    FilePath    string
    MimeType    string
    Size        int64
    ModTime     time.Time
    Permissions os.FileMode
    config      ResourceConfig
    logger      *logger.Logger
    validator   *FilePathValidator
}
```

### 2. File System Resource Handler (`internal/resources/files/handler.go`)

#### FileSystemResourceHandler Structure
```go
type FileSystemResourceHandler struct {
    resource  *FileSystemResource
    logger    *logger.Logger
    validator *FilePathValidator
}
```

### 3. File System Resource Factory (`internal/resources/files/factory.go`)

#### FileSystemResourceFactory Structure
```go
type FileSystemResourceFactory struct {
    baseURI     string
    basePath    string
    allowedDirs []string
    maxFileSize int64
    logger      *logger.Logger
}
```

### 4. Path Validation (`internal/resources/files/validator.go`)

#### FilePathValidator Structure
```go
type FilePathValidator struct {
    allowedDirectories []string
    maxFileSize        int64
    allowedExtensions  []string
    blockedPatterns    []string
    logger            *logger.Logger
}
```

## Error Handling Strategy

### File System Specific Errors
```go
var (
    ErrFileNotFound         = fmt.Errorf("file not found")
    ErrFilePermissionDenied = fmt.Errorf("file permission denied")
    ErrInvalidFilePath      = fmt.Errorf("invalid file path")
    ErrDirectoryTraversal   = fmt.Errorf("directory traversal attempt")
    ErrFileTooBig          = fmt.Errorf("file size exceeds limit")
    ErrUnsupportedFileType = fmt.Errorf("unsupported file type")
    ErrDirectoryNotAllowed = fmt.Errorf("directory not in allowed list")
)
```

## Security Implementation

### Path Security
- Convert to absolute path
- Check for directory traversal
- Validate against allowed directories
- Respect file permissions

### Access Control Configuration
```yaml
file_resources:
  allowed_directories:
    - "/var/lib/app/data"
    - "/etc/app/configs"
    - "/tmp/app/uploads"
  max_file_size: 10485760  # 10MB
  allowed_extensions:
    - ".txt"
    - ".json"
    - ".xml"
    - ".log"
```

## Testing Strategy

### Unit Tests
- Path validation tests
- File access tests
- MIME type detection tests
- Error handling tests

### Integration Tests
- Resource registration
- MCP protocol integration
- Health monitoring

## Success Criteria

### Functional Requirements
1. File system resource successfully serves file content through MCP
2. Security validation prevents unauthorized file access
3. Audit logging captures all file access operations
4. Resource integrates with existing registry framework
5. Health monitoring provides accurate file system status

### Security Requirements
1. Directory traversal attacks are prevented
2. Only allowed directories are accessible
3. File permissions are respected
4. All access attempts are logged
5. Error messages don't leak sensitive path information