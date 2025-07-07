# Commit 1: Project Foundation

## Overview
Establish the foundational structure for the MCP server project, including Go module initialization, directory structure, structured logging, and basic error handling.

## Objectives
- Initialize Go module with proper naming
- Create standard Go project directory structure
- Set up structured logging using Go's slog package
- Add comprehensive .gitignore for Go projects
- Create user-facing README.md with feature placeholder
- Implement basic error handling patterns
- Ensure the result is a valid, runnable Go project

## Prerequisites
- Git repository initialized (Step 0 complete)
- Go 1.22.1 installed and available
- Current working directory: `/Users/juli/workspace/mcp-server`

## Implementation Steps

### 1. Initialize Go Module
```bash
go mod init mcp-server
```

### 2. Create Directory Structure
```bash
mkdir -p cmd/mcp-server
mkdir -p internal/server
mkdir -p internal/config
mkdir -p internal/tools
mkdir -p internal/resources
mkdir -p internal/logger
mkdir -p pkg/mcplib
mkdir -p configs
mkdir -p deployments/docker
mkdir -p deployments/kubernetes
mkdir -p scripts
mkdir -p tests/integration
mkdir -p docs
```

### 3. Create Structured Logger
**File**: `internal/logger/logger.go`
- Set up slog with JSON formatting for production
- Support different log levels (DEBUG, INFO, WARN, ERROR)
- Include contextual fields (service name, version)
- Handle logger initialization errors gracefully

### 4. Create Basic Configuration
**File**: `internal/config/config.go`
- Define configuration struct with validation
- Support environment variable loading
- Include default values for all settings
- Implement configuration validation with detailed error messages

### 5. Create Minimal Main Application
**File**: `cmd/mcp-server/main.go`
- Basic application entry point
- Initialize logger with error handling
- Load configuration with validation
- Implement graceful shutdown signal handling
- Exit with proper error codes

### 6. Create .gitignore
**File**: `.gitignore`
- Standard Go project ignores
- Binary outputs, build artifacts
- IDE and editor files
- Environment and configuration files
- Test coverage reports

### 7. Create Project README
**File**: `README.md`
- Project title and description
- Features section with TBD placeholder (user-focused capabilities)
- Quick start guide
- Installation instructions
- Development setup
- Contributing guidelines

## Expected Outcomes
- `go.mod` file created with module name `mcp-server`
- Complete directory structure following Go best practices
- Structured logging system ready for use
- Configuration management system in place
- User-facing README with feature placeholder
- Runnable Go application (even if minimal)
- Proper error handling throughout all components

## File Structure After Implementation
```
mcp-server/
├── .gitignore
├── CLAUDE.md
├── README.md
├── go.mod
├── cmd/
│   └── mcp-server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── logger/
│   │   └── logger.go
│   ├── server/
│   ├── tools/
│   └── resources/
├── pkg/
│   └── mcplib/
├── configs/
├── deployments/
├── scripts/
├── tests/
├── docs/
└── specs/
    └── step-0-git-init.md
```

## Error Handling Requirements
- **Logger initialization**: Handle slog setup failures gracefully
- **Configuration loading**: Validate all config parameters with clear error messages
- **Application startup**: Proper error codes and logging for startup failures
- **Signal handling**: Graceful shutdown with cleanup and error logging
- **Module initialization**: Handle go.mod creation and validation errors

## Testing Requirements
- Application must compile without errors: `go build ./cmd/mcp-server`
- Application must run without crashing: `./mcp-server`
- Logger must produce structured JSON output
- Configuration must load from environment variables
- Graceful shutdown must work with SIGINT/SIGTERM

## Success Criteria
- `go build ./cmd/mcp-server` succeeds
- `./mcp-server` runs without immediate crash
- Structured logs are produced in JSON format
- Configuration can be loaded from environment
- Application responds to shutdown signals
- All error cases are handled with appropriate logging
- README.md accurately represents current capabilities
- Code follows Go 1.22.1 standards and best practices

## Next Steps
After successful implementation:
1. Create Commit 2 specification (Minimal Runnable Server)
2. Implement basic HTTP server functionality
3. Add MCP protocol dependencies