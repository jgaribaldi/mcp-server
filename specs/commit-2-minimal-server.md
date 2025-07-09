# Commit 2: Minimal Runnable Server

## Overview
Build upon the project foundation to create a minimal runnable HTTP server that can handle basic requests and serve as the foundation for the MCP protocol implementation.

## Objectives
- Create basic HTTP server infrastructure
- Implement health and readiness endpoints with proper semantics
- Extend configuration system for server settings
- Integrate HTTP server with main application lifecycle
- Maintain graceful shutdown capability
- Ensure comprehensive error handling and logging

## Prerequisites
- Commit 1 (Project Foundation) completed
- Go 1.22.1 available
- Project compiles and runs successfully

## Implementation Steps

### 1. Create HTTP Server Implementation
**File**: `internal/server/server.go`
- Basic HTTP server struct with configuration
- Server lifecycle management (start, stop, graceful shutdown)
- Request routing for health endpoints
- Proper error handling and logging for all operations
- Context-based cancellation support

### 2. Implement Health Endpoints
**Endpoint**: `/health`
- **Purpose**: Indicates if application is alive and functioning
- **Response**: 200 OK if basic application components are working
- **Checks**: Application process running, core services initialized
- **Usage**: Used by monitoring systems to detect if service needs restart

**Endpoint**: `/ready`
- **Purpose**: Indicates if application is ready to serve traffic
- **Response**: 200 OK only when service can handle requests
- **Checks**: All dependencies available, initialization complete, fully warmed up
- **Usage**: Used by load balancers to determine if traffic should be routed

### 3. Extend Configuration System
**File**: `internal/config/config.go` (extend existing)
- Add HTTP server configuration options
- Server timeouts (read, write, idle)
- Maximum header size and request limits
- TLS configuration preparation (for future use)
- Validation for all new configuration parameters

### 4. Update Main Application
**File**: `cmd/mcp-server/main.go` (modify existing)
- Initialize HTTP server with configuration
- Start server in background goroutine
- Integrate server shutdown with existing graceful shutdown
- Proper error handling for server startup failures
- Enhanced logging for server lifecycle events

### 5. Server Configuration Options
Environment variables to support:
- `MCP_SERVER_READ_TIMEOUT`: HTTP read timeout (default: 15s)
- `MCP_SERVER_WRITE_TIMEOUT`: HTTP write timeout (default: 15s)
- `MCP_SERVER_IDLE_TIMEOUT`: HTTP idle timeout (default: 60s)
- `MCP_SERVER_MAX_HEADER_BYTES`: Maximum header size (default: 1MB)

## Expected Outcomes
- Runnable HTTP server on configured host/port
- Working health check endpoints with correct semantics
- Extended configuration system supporting server settings
- Proper JSON logging for all HTTP operations
- Graceful shutdown with HTTP server cleanup
- Foundation ready for MCP protocol addition

## File Structure After Implementation
```
mcp-server/
├── .gitignore
├── CLAUDE.md
├── README.md
├── go.mod
├── cmd/
│   └── mcp-server/
│       └── main.go (modified)
├── internal/
│   ├── config/
│   │   └── config.go (extended)
│   ├── logger/
│   │   └── logger.go
│   ├── server/
│   │   └── server.go (new)
│   ├── tools/
│   └── resources/
└── specs/
    ├── step-0-git-init.md
    └── commit-1-project-foundation.md
```

## Error Handling Requirements
- **Server startup errors**: Port binding failures, permission issues
- **Request handling errors**: Malformed requests, timeout handling
- **Shutdown errors**: Connection cleanup, graceful termination
- **Configuration errors**: Invalid timeout values, port conflicts
- **Logging errors**: All HTTP operations properly logged with context

## Testing Requirements
- Server starts successfully on configured port
- Health endpoints respond with correct HTTP status codes
- Graceful shutdown works properly with active connections
- Configuration validation catches invalid parameters
- All error cases produce appropriate log messages

## Health Endpoint Specifications

### `/health` Endpoint
- **HTTP Method**: GET
- **Response Format**: JSON
- **Success Response**: 
  ```json
  {
    "status": "healthy",
    "timestamp": "2024-01-01T00:00:00Z",
    "service": "mcp-server",
    "version": "dev"
  }
  ```
- **Error Response**: 503 Service Unavailable if core components fail

### `/ready` Endpoint
- **HTTP Method**: GET
- **Response Format**: JSON
- **Success Response**:
  ```json
  {
    "status": "ready",
    "timestamp": "2024-01-01T00:00:00Z",
    "service": "mcp-server",
    "version": "dev"
  }
  ```
- **Error Response**: 503 Service Unavailable if not ready to serve traffic

## Success Criteria
- `go build ./cmd/mcp-server` succeeds
- `./mcp-server` starts HTTP server without errors
- `curl http://localhost:3000/health` returns 200 with proper JSON
- `curl http://localhost:3000/ready` returns 200 with proper JSON
- Server logs all HTTP requests in structured JSON format
- Server shuts down gracefully on SIGINT/SIGTERM
- No regression in existing functionality
- All error cases are handled with appropriate logging

## Next Steps
After successful implementation:
1. Create Commit 3 specification (MCP Protocol Dependencies)
2. Add MCP Go SDK dependencies
3. Begin MCP server core implementation

## Performance Considerations
- Request/response logging should not block request processing
- Health endpoints should respond quickly (< 100ms)
- Server should handle concurrent requests appropriately
- Graceful shutdown should complete within configured timeout