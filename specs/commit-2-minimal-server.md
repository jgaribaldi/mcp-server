# Commit 2: Minimal Runnable Server

## Overview
Build upon the project foundation to create a minimal runnable HTTP server. This implementation is broken down into small, atomic steps where each step produces compilable code that passes tests and can be committed independently.

## Objectives
- Create HTTP server infrastructure through incremental steps
- Ensure each step is independently committable and testable
- Maintain working state throughout development
- Build foundation for MCP protocol implementation

## Prerequisites
- Commit 1 (Project Foundation) completed
- Go 1.22.1 available
- Project compiles and runs successfully

## Implementation Steps

### Step 2.1: Add HTTP Server Configuration
**Objective**: Extend configuration system to support HTTP server settings
**Files Modified**: `internal/config/config.go`

**Changes**:
- Add HTTP server timeout fields to `ServerConfig` struct
- Add new environment variables for HTTP timeouts
- Extend validation to check HTTP configuration parameters
- Add helper functions for duration parsing

**New Configuration Fields**:
- `ReadTimeout time.Duration` (default: 15s)
- `WriteTimeout time.Duration` (default: 15s)
- `IdleTimeout time.Duration` (default: 60s)
- `MaxHeaderBytes int` (default: 1MB)

**Environment Variables**:
- `MCP_SERVER_READ_TIMEOUT` (default: "15s")
- `MCP_SERVER_WRITE_TIMEOUT` (default: "15s") 
- `MCP_SERVER_IDLE_TIMEOUT` (default: "60s")
- `MCP_SERVER_MAX_HEADER_BYTES` (default: "1048576")

**Success Criteria**:
- Code compiles successfully
- Configuration loads with new HTTP settings
- Validation works for invalid timeout values
- No functional changes to existing behavior

### Step 2.2: Create Basic HTTP Server Structure
**Objective**: Create HTTP server struct and basic setup without starting server
**Files Created**: `internal/server/server.go`

**Changes**:
- Create `Server` struct with HTTP server configuration
- Add constructor function `New()` that creates server instance
- Include basic setup methods (routes setup placeholder)
- Add imports for HTTP functionality

**Server Struct**:
```go
type Server struct {
    httpServer *http.Server
    logger     *logger.Logger
    config     *config.Config
    mux        *http.ServeMux
}
```

**Methods**:
- `New(cfg *config.Config, log *logger.Logger) *Server`
- `setupRoutes()` (placeholder, no routes yet)

**Success Criteria**:
- Code compiles successfully
- Server struct can be instantiated
- No HTTP endpoints active yet
- No changes to main application

### Step 2.3: Add Health Endpoint
**Objective**: Add `/health` endpoint to server
**Files Modified**: `internal/server/server.go`

**Changes**:
- Add `/health` route to `setupRoutes()` method
- Implement `handleHealth()` method
- Return JSON response with health status

**Health Endpoint Response**:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T00:00:00Z",
  "service": "mcp-server",
  "version": "dev"
}
```

**Success Criteria**:
- Code compiles successfully
- Health endpoint handler is registered
- JSON response is properly formatted
- Server still not integrated with main application

### Step 2.4: Add Readiness Endpoint
**Objective**: Add `/ready` endpoint to server
**Files Modified**: `internal/server/server.go`

**Changes**:
- Add `/ready` route to `setupRoutes()` method
- Implement `handleReady()` method
- Return JSON response with readiness status

**Ready Endpoint Response**:
```json
{
  "status": "ready",
  "timestamp": "2024-01-01T00:00:00Z",
  "service": "mcp-server",
  "version": "dev"
}
```

**Success Criteria**:
- Code compiles successfully
- Ready endpoint handler is registered
- JSON response is properly formatted
- Both health and ready endpoints available
- Server still not integrated with main application

### Step 2.5: Integrate Server with Main Application
**Objective**: Connect HTTP server with main application lifecycle
**Files Modified**: `cmd/mcp-server/main.go`

**Changes**:
- Import server package
- Initialize HTTP server in main function
- Start server in background goroutine
- Integrate server shutdown with existing graceful shutdown
- Add server lifecycle logging

**Integration Points**:
- Server creation after logger initialization
- Server start in background goroutine
- Server shutdown in graceful shutdown handler
- Error handling for server startup failures

**Success Criteria**:
- Code compiles successfully
- HTTP server starts on configured port
- Health and ready endpoints are accessible
- Graceful shutdown stops HTTP server
- All operations logged with structured logging

## Testing Requirements for Each Step

### Step 2.1 Testing:
- Configuration loads successfully with new fields
- Default values are applied correctly
- Validation catches invalid timeout values
- Environment variables override defaults

### Step 2.2 Testing:
- Server struct can be instantiated
- Constructor accepts configuration and logger
- No HTTP endpoints are accessible yet
- No errors during server creation

### Step 2.3 Testing:
- Health endpoint returns 200 status
- Response is valid JSON with correct structure
- Timestamp is properly formatted
- Logging works for health requests

### Step 2.4 Testing:
- Ready endpoint returns 200 status
- Response is valid JSON with correct structure
- Both health and ready endpoints work
- Logging works for ready requests

### Step 2.5 Testing:
- Server starts on configured port
- `curl http://localhost:3000/health` returns success
- `curl http://localhost:3000/ready` returns success
- Server shuts down gracefully
- All HTTP operations are logged

## Error Handling Requirements

Each step must include:
- Proper error handling for its specific functionality
- Structured logging for all operations
- Graceful degradation where applicable
- Clear error messages for debugging

## Commit Strategy

Each step should be committed individually:
1. **Step 2.1**: "Add HTTP server configuration support"
2. **Step 2.2**: "Create basic HTTP server structure"
3. **Step 2.3**: "Add health endpoint"
4. **Step 2.4**: "Add readiness endpoint"
5. **Step 2.5**: "Integrate HTTP server with main application"

## Success Criteria for Complete Commit 2

- All 5 steps completed and committed
- HTTP server runs on configured port
- Health and ready endpoints respond correctly
- Graceful shutdown works with HTTP server
- All operations logged with structured JSON
- No regression in existing functionality
- Foundation ready for MCP protocol implementation

## Next Steps
After successful completion:
1. Create Commit 3 specification (MCP Protocol Dependencies)
2. Add MCP Go SDK dependencies
3. Begin MCP server core implementation