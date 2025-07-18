# Commit 7: First Example Tool - Echo Tool Implementation

## Overview
This specification defines the implementation of the Echo tool as the first example tool in the MCP server. The Echo tool provides simple string manipulation functionality while demonstrating clean architecture principles and proper separation of business logic from infrastructure concerns.

## Implementation Strategy
The Echo tool implementation will be broken into 4 incremental commits, each adding functional value while maintaining a runnable state:

### Commit 7.1: Echo Tool Foundation
**File**: `specs/commit-7.1-echo-foundation.md`

**Business Logic Implementation**:
- Create `internal/tools/echo/echo.go`
- Implement `EchoService` struct with pure business logic:
  ```go
  type EchoService struct{}
  
  func (s *EchoService) Transform(message, prefix, suffix string, uppercase bool) string
  func (s *EchoService) Validate(message string) error
  ```

**Core Functionality**:
- Message transformation: combine prefix + message + suffix
- Optional uppercase conversion
- Input validation (non-empty message, length limits)
- Error handling for invalid inputs

**Unit Tests**: `internal/tools/echo/echo_test.go`
- Test message transformation with various combinations
- Test validation logic with edge cases (empty, too long, special chars)
- Test error conditions
- **Focus on MCP server business logic only**

### Commit 7.2: Echo Tool MCP Integration
**File**: `specs/commit-7.2-echo-mcp-integration.md`

**MCP Interface Implementation**:
- Create `internal/tools/echo/tool.go`
- Implement `EchoTool` struct (implements `mcp.Tool`)
- Implement `EchoHandler` struct (implements `mcp.ToolHandler`)
- JSON schema for parameters:
  ```json
  {
    "type": "object",
    "properties": {
      "message": {"type": "string", "minLength": 1, "maxLength": 1000},
      "prefix": {"type": "string", "maxLength": 100},
      "suffix": {"type": "string", "maxLength": 100},
      "uppercase": {"type": "boolean"}
    },
    "required": ["message"]
  }
  ```

**Unit Tests**: `internal/tools/echo/tool_test.go`
- Test tool metadata (name: "echo", description, parameters)
- Test handler execution with valid JSON parameters
- Test parameter validation and error responses
- Test content formatting and MCP result structure
- **Focus on MCP server business logic only**

### Commit 7.3: Echo Tool Factory Implementation
**File**: `specs/commit-7.3-echo-factory.md`

**Factory Implementation**:
- Create `internal/tools/echo/factory.go`
- Implement `EchoFactory` struct (implements `tools.ToolFactory`)
- Factory metadata:
  - Name: "echo"
  - Description: "Simple text manipulation tool for testing and demonstration"
  - Version: "1.0.0"
  - Capabilities: ["text_processing", "demonstration"]
  - Requirements: {"runtime": "go"}

**No Unit Tests**: Factory is infrastructure code, not MCP server business logic

### Commit 7.4: Echo Tool Registry Integration
**File**: `specs/commit-7.4-echo-registry.md`

**Registry Integration**:
- Update `cmd/mcp-server/main.go` to register Echo tool
- Add Echo factory to default registry initialization
- Ensure proper error handling during registration

**Manual Verification**:
- Tool registration succeeds
- Tool appears in registry list with correct status
- Health endpoint shows Echo tool as active
- Circuit breaker protection works

## Technical Requirements

### Clean Architecture Principles
1. **Business Logic Separation**: `EchoService` contains pure business logic with no MCP dependencies
2. **Interface Adapters**: `EchoTool` and `EchoHandler` adapt business logic to MCP interfaces
3. **Dependency Direction**: Business logic doesn't depend on infrastructure
4. **Testability**: Business logic can be tested without mocking infrastructure

### Error Handling
- Input validation errors (empty message, length violations)
- JSON parameter parsing errors
- MCP protocol errors
- Registry integration errors
- Circuit breaker error scenarios

### Testing Strategy
- **Unit Tests Only**: Focus exclusively on MCP server business logic
- Test pure business logic (EchoService)
- Test MCP interface adapters (EchoTool, EchoHandler)
- **No factory tests, configuration tests, or infrastructure tests**

### Echo Tool Functionality
**Simple and Focused**:
- Input: message (required), prefix (optional), suffix (optional), uppercase (optional)
- Processing: string concatenation and optional case conversion
- Output: transformed text as MCP content
- Validation: message length (1-1000 chars), prefix/suffix length (0-100 chars)

### File Structure
```
internal/tools/echo/
├── echo.go          # Business logic (EchoService)
├── echo_test.go     # Business logic tests
├── tool.go          # MCP integration (EchoTool, EchoHandler)
├── tool_test.go     # MCP integration tests
└── factory.go       # Factory implementation (no tests)
```

## Success Criteria
1. Each commit maintains runnable server state
2. Unit test coverage for MCP server business logic only
3. Clean separation between business logic and infrastructure
4. Echo tool successfully registers and executes in MCP server
5. Health monitoring correctly reports Echo tool status
6. Circuit breaker protection active for Echo tool
7. All MCP business logic unit tests pass with proper error handling coverage

This specification ensures the Echo tool serves as a simple, well-architected example that demonstrates MCP server capabilities while following established patterns for future tool development.