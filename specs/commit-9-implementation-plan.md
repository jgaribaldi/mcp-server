# Commit 9 Implementation Plan: Incremental File System Resource Development

## Overview
This specification defines the incremental implementation approach for commit 9 (first example resource) following the coding and committing guidelines from CLAUDE.md. The implementation is broken down into 8 small, incremental commits that maintain a runnable state throughout development.

## Reference
This implementation plan is based on the detailed specification in `specs/commit-9-first-resource.md` and follows the git-driven development approach outlined in CLAUDE.md.

## Incremental Commit Plan

### Commit 9.1: File System Resource Foundation
**Goal**: Create basic directory structure and error definitions

**Changes**:
- Create `internal/resources/files/` directory
- Create `internal/resources/files/errors.go` with file system specific error types:
  - ErrFileNotFound, ErrFilePermissionDenied, ErrInvalidFilePath
  - ErrDirectoryTraversal, ErrFileTooBig, ErrUnsupportedFileType, ErrDirectoryNotAllowed

**Runnable State**: Server starts normally, no new functionality added yet. Error types are available for import.

**Test Requirements**:
- Create `internal/resources/files/errors_test.go`
- Test error creation and string representations
- Verify error types implement error interface correctly

**Validation Criteria**:
- All code compiles without warnings
- Server starts successfully
- Error types are properly defined and testable
- Tests pass with proper coverage

---

### Commit 9.2: Path Validation Security Layer
**Goal**: Implement core security validation for file paths

**Changes**:
- Create `internal/resources/files/validator.go` with FilePathValidator struct
- Implement path traversal prevention methods
- Add directory validation against allow-lists
- Add MIME type detection utilities
- Add file extension validation

**Runnable State**: Validator can be instantiated and used independently. Security validation functions are available.

**Test Requirements**:
- Create `internal/resources/files/validator_test.go`
- Comprehensive path validation tests
- Security attack prevention tests (directory traversal, path injection)
- MIME type detection accuracy tests
- Edge cases and boundary condition tests

**Validation Criteria**:
- All security validations properly prevent attacks
- Validator can be created and used without dependencies
- MIME type detection works for common file types
- All tests pass with comprehensive coverage

---

### Commit 9.3: File System Resource Core
**Goal**: Implement the main FileSystemResource struct

**Changes**:
- Create `internal/resources/files/resource.go` with FileSystemResource struct
- Implement mcp.Resource interface methods (URI, Name, Description, MimeType, Handler)
- Add file metadata handling (size, modification time, permissions)
- Basic file content reading with error handling
- Integration with validator from 9.2

**Runnable State**: Resource can be created and provides metadata. File information can be extracted.

**Test Requirements**:
- Create `internal/resources/files/resource_test.go`
- Test resource creation and interface compliance
- Test metadata extraction accuracy
- Test basic file operations and error handling
- Test validator integration

**Validation Criteria**:
- Resource properly implements mcp.Resource interface
- Metadata extraction is accurate and efficient
- Error handling is comprehensive and secure
- Integration with validator works correctly

---

### Commit 9.4: Resource Handler Implementation
**Goal**: Add content access handler with security integration

**Changes**:
- Create `internal/resources/files/handler.go` with FileSystemResourceHandler struct
- Implement mcp.ResourceHandler interface
- Implement Read() method with validator integration
- Add audit logging for file access operations
- Comprehensive error handling for file operations

**Runnable State**: Complete resource can read file content securely with full audit trail.

**Test Requirements**:
- Create `internal/resources/files/handler_test.go`
- Test handler functionality and interface compliance
- Test security validation integration
- Test audit logging accuracy
- Test error handling scenarios (permission denied, file not found)

**Validation Criteria**:
- Handler properly implements mcp.ResourceHandler interface
- Security validation prevents unauthorized access
- Audit logging captures all access attempts
- Error handling provides appropriate feedback

---

### Commit 9.5: Resource Factory Pattern
**Goal**: Implement factory for resource creation and configuration

**Changes**:
- Create `internal/resources/files/factory.go` with FileSystemResourceFactory struct
- Implement resources.ResourceFactory interface
- Configuration management for allowed directories and limits
- Resource creation workflow with validation
- Factory-level error handling and validation

**Runnable State**: Factory can create configured file resources with proper validation.

**Test Requirements**:
- Create `internal/resources/files/factory_test.go`
- Test factory creation and interface compliance
- Test configuration validation
- Test resource instantiation through factory
- Test factory error handling

**Validation Criteria**:
- Factory properly implements resources.ResourceFactory interface
- Configuration validation is comprehensive
- Resource creation workflow is secure and efficient
- Factory integrates with all previous components

---

### Commit 9.6: Configuration System Integration
**Goal**: Extend configuration system for file resources

**Changes**:
- Extend `internal/config/config.go` to include file resource configuration types
- Add FileResourceConfig struct with allowed directories, size limits, extensions
- Update validation logic for file resource settings
- Add default configuration values
- Update config loading and merging logic

**Runnable State**: Configuration system supports file resource settings with validation.

**Test Requirements**:
- Update `internal/config/config_test.go` with file resource configuration tests
- Test configuration loading, validation, and merging
- Test default values and environment variable overrides
- Test configuration error handling

**Validation Criteria**:
- Configuration system properly supports file resource settings
- Validation provides helpful error messages
- Default values are appropriate for production use
- Environment variable integration works correctly

---

### Commit 9.7: Server Integration and Registration
**Goal**: Register file system resources in server startup

**Changes**:
- Update server initialization to include file resource factory registration
- Integrate with existing resource registry
- Add example file resource registration for demonstration
- Update server startup logging for file resources

**Runnable State**: Server starts with file resources registered and available through MCP protocol.

**Test Requirements**:
- Update server integration tests to include file resources
- Test resource registry integration
- Test MCP protocol resource access
- Test server startup with file resources

**Validation Criteria**:
- Server starts successfully with file resources
- Resources are accessible through MCP protocol
- Resource registry integration is seamless
- No regression in existing functionality

---

### Commit 9.8: Health Monitoring Integration
**Goal**: Add file resource health monitoring

**Changes**:
- Extend health endpoints to include file resource status
- Add file system specific health checks (directory accessibility, permission validation)
- Monitor file access patterns and error rates
- Integration with circuit breaker monitoring

**Runnable State**: Health endpoints show file resource status with detailed diagnostics.

**Test Requirements**:
- Update health monitoring tests to include file resources
- Test health check accuracy
- Test metrics collection for file operations
- Test health endpoint integration

**Validation Criteria**:
- Health endpoints accurately report file resource status
- Health checks detect common file system issues
- Metrics collection provides useful operational data
- Integration with existing health monitoring is seamless

## Implementation Guidelines

### Code Quality Standards
- Follow Single Responsibility Principle for all functions and structs
- Include meaningful comments only for complex business logic
- Use dependency injection for testability
- Implement comprehensive error handling with structured logging

### Security Focus
- Path traversal prevention implemented from the start
- Directory allow-listing enforced at all levels
- Audit logging for all file access operations
- Secure error messages that don't expose sensitive information

### Testing Strategy
- Unit tests for each component as implemented
- Security-focused tests for path validation and access control
- Integration tests for server and registry connectivity
- Performance tests for file operations

### Commit Message Guidelines
- Use descriptive commit messages focusing on functionality added
- Follow conventional commit format
- Do not include references to AI tools or automated generation
- Include reference to this implementation plan where appropriate

## Success Criteria

### Per-Commit Validation
- Each commit compiles without warnings
- All tests pass at each step
- Server starts and responds to health checks
- No regression in existing functionality
- Clear documentation of changes made

### Overall Implementation Success
- File system resource fully functional through MCP protocol
- Security validation prevents unauthorized access
- Comprehensive error handling and logging
- Integration with existing server infrastructure
- Full test coverage including security scenarios
- Performance meets production requirements

This incremental approach ensures that the file system resource implementation maintains the high quality standards established in the existing codebase while providing a solid foundation for future resource implementations.