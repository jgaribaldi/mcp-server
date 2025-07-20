# MCP Server in Go - Production Ready Implementation

## Project Overview

This is a Model Context Protocol (MCP) server implementation in Go 1.22.1, designed for production deployment. The MCP server exposes tools and resources to LLM applications through a standardized protocol, enabling secure integration between AI models and external data sources.

**Key Features:**
- Production-ready MCP server with comprehensive error handling
- Structured logging throughout all components
- Docker containerization support
- Health monitoring and metrics
- Scalable architecture following Go best practices

## Development Setup

### Prerequisites
- Go 1.22.1 (fixed version)
- Git for version control
- Docker (for containerization)

### Initial Setup
```bash
# Initialize git repository
git init

# Initialize Go module
go mod init mcp-server

# Install dependencies
go mod tidy
```

### Development Environment
```bash
# Build for development
go build -o bin/mcp-server ./cmd/mcp-server

# Run in development mode
./bin/mcp-server

# Run with debug logging
DEBUG=true ./bin/mcp-server
```

## Project Structure

This project follows the standard Go project layout optimized for production deployment:

```
mcp-server/
├── cmd/
│   └── mcp-server/          # Main application entry point
│       └── main.go
├── internal/                # Private application code
│   ├── server/             # MCP server implementation
│   │   ├── server.go
│   │   └── handlers.go
│   ├── config/             # Configuration management
│   │   └── config.go
│   ├── tools/              # MCP tools implementation
│   │   ├── registry.go     # Tool registration and management
│   │   ├── echo/           # Echo tool (simple example)
│   │   │   ├── echo.go
│   │   │   └── echo_test.go
│   │   ├── filesystem/     # File system operations tool
│   │   │   ├── filesystem.go
│   │   │   └── filesystem_test.go
│   │   └── weather/        # Weather information tool
│   │       ├── weather.go
│   │       └── weather_test.go
│   ├── resources/          # MCP resources implementation
│   │   ├── registry.go
│   │   └── files/
│   └── logger/             # Structured logging
│       └── logger.go
├── pkg/                    # Exportable library code
│   └── mcplib/            # Reusable MCP utilities
├── configs/               # Configuration files
│   ├── development.yaml
│   ├── production.yaml
│   └── docker.yaml
├── deployments/           # Deployment configurations
│   ├── docker/
│   │   ├── Dockerfile
│   │   └── docker-compose.yml
│   └── kubernetes/
│       ├── deployment.yaml
│       └── service.yaml
├── scripts/               # Build and deployment scripts
│   ├── build.sh
│   └── deploy.sh
├── specs/                 # Implementation specifications
│   ├── step-0-git-init.md          # Git repository initialization spec
│   ├── commit-1-project-foundation.md  # Project foundation spec
│   ├── commit-2-minimal-server.md      # Minimal runnable server spec
│   ├── commit-3-mcp-dependencies.md    # MCP protocol dependencies spec
│   ├── commit-4-mcp-server-core.md     # MCP server core spec
│   ├── commit-5-configuration.md       # Configuration management spec
│   ├── commit-6-tool-framework.md      # Tool registration framework spec
│   ├── commit-7-first-tool.md          # First example tool spec
│   ├── commit-8-resource-framework.md  # Resource registration framework spec
│   ├── commit-9-first-resource.md      # First example resource spec
│   ├── commit-10-health-diagnostics.md # Health and diagnostics spec
│   ├── commit-11-docker-config.md      # Docker configuration spec
│   ├── commit-12-production-deploy.md  # Production deployment spec
│   ├── commit-13-monitoring.md         # Monitoring and metrics spec
│   ├── commit-14-testing.md            # Comprehensive testing spec
│   └── commit-15-production-hardening.md # Production hardening spec
├── tests/                 # Integration tests
│   └── integration/
└── docs/                  # Documentation
    └── api.md
```

### Tools Directory Structure Rationale

**Why `/internal/tools/`:**
1. **Private Implementation**: Tools contain business logic specific to this MCP server and shouldn't be imported by other projects
2. **Organized by Function**: Each tool gets its own package for clear separation of concerns
3. **Follows Go Standards**: The `/internal` directory is the standard location for private application code
4. **Scalable**: New tools can be added as separate packages without affecting existing ones
5. **Testable**: Each tool package can have its own tests alongside the implementation

**Alternatives Considered:**
- `/pkg/tools/` - Rejected because tools are server-specific, not reusable libraries
- `/cmd/tools/` - Rejected because this is for executable commands, not MCP tool implementations
- `/tools/` - Rejected because it doesn't follow Go project layout standards

### Specs Directory Structure

**Purpose of `/specs/`:**
The specs directory contains detailed implementation specifications for each step of the MCP server development process. Before implementing any changes (Step 0, Commit 1, etc.), a corresponding .md file must be created in the specs directory describing the change in detail.

**Workflow:**
1. **Create Specification**: Write detailed spec file before implementation
2. **Review Specification**: Review and approve the spec before coding
3. **Implement According to Spec**: Code exactly according to the specification
4. **Commit Changes**: Commit implementation with reference to the spec

**Benefits:**
- **Planning**: Forces careful planning before implementation
- **Documentation**: Creates comprehensive documentation of each change
- **Review**: Enables spec review before coding begins
- **Consistency**: Ensures consistent implementation across all changes
- **Traceability**: Links each commit to its detailed specification

## Building and Testing

### Development Build
```bash
# Build for current platform
go build -o bin/mcp-server ./cmd/mcp-server

# Build with race detection
go build -race -o bin/mcp-server ./cmd/mcp-server

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -o bin/mcp-server-linux ./cmd/mcp-server
```

### Production Build
```bash
# Build optimized binary
go build -ldflags="-w -s" -o bin/mcp-server ./cmd/mcp-server

# Build with version information
VERSION=$(git describe --tags --always)
go build -ldflags="-w -s -X main.version=$VERSION" -o bin/mcp-server ./cmd/mcp-server
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with detailed coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run integration tests
go test ./tests/integration/...

# Run benchmarks
go test -bench=. ./...
```

### Code Quality Tools
```bash
# Format code
go fmt ./...

# Lint code (requires golangci-lint)
golangci-lint run

# Vet code
go vet ./...

# Check for security issues (requires gosec)
gosec ./...
```

### Code Quality Standards

This project follows strict coding standards to ensure maintainability, readability, and reliability:

#### Single Responsibility Principle
All functions should adhere to the Single Responsibility Principle - each function should do one thing and one thing only. This makes code easier to test, understand, and maintain.

**Good Example:**
```go
func (s *EchoService) Transform(message, prefix, suffix string, uppercase bool) string {
    result := prefix + message + suffix
    if uppercase {
        result = strings.ToUpper(result)
    }
    return result
}

func (s *EchoService) Validate(message string) error {
    if message == "" {
        return fmt.Errorf("message cannot be empty")
    }
    return nil
}
```

**Avoid:**
```go
// Function doing too many things
func (s *EchoService) ProcessMessage(message, prefix, suffix string, uppercase bool) (string, error) {
    // Validation logic
    if message == "" {
        return "", fmt.Errorf("message cannot be empty")
    }
    
    // Transformation logic
    result := prefix + message + suffix
    if uppercase {
        result = strings.ToUpper(result)
    }
    
    // Logging logic
    log.Printf("Processed message: %s", result)
    
    return result, nil
}
```

#### Comment Guidelines
Include comments only when they add value for human developers. Avoid obvious comments that simply restate what the code does.

**Include comments for:**
- Complex business logic that may not be immediately clear
- Non-obvious algorithmic choices or performance optimizations
- External API integrations or protocol-specific implementations
- Workarounds for known issues or limitations

**Good Example:**
```go
// Circuit breaker pattern: fail fast when service is consistently failing
// to prevent cascade failures across the system
if s.breaker.State() == gobreaker.StateOpen {
    return nil, ErrServiceUnavailable
}

func (s *EchoService) ValidateAll(message, prefix, suffix string) error {
    // Validate each parameter independently to provide specific error messages
    if err := s.Validate(message); err != nil {
        return err
    }
    return nil
}
```

**Avoid obvious comments:**
```go
// Create new echo service - AVOID: obvious from function name
func NewEchoService() *EchoService {
    return &EchoService{}
}

// Return the name - AVOID: obvious from return statement
func (t *EchoTool) Name() string {
    return "echo"
}

// Check if message is empty - AVOID: obvious from condition
if message == "" {
    return fmt.Errorf("message cannot be empty")
}
```

**Code Organization Principles:**
- Separate business logic from infrastructure concerns
- Use dependency injection for testability
- Keep functions small and focused
- Use meaningful variable and function names that reduce the need for comments
- Group related functionality into cohesive packages

## Production Deployment

### Docker Containerization
```dockerfile
# Multi-stage build for optimized image
FROM golang:1.22.1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o mcp-server ./cmd/mcp-server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/mcp-server .
COPY --from=builder /app/configs ./configs
EXPOSE 8080
CMD ["./mcp-server"]
```

### Environment Configuration
```bash
# Production environment variables
export MCP_SERVER_PORT=8080
export MCP_SERVER_HOST=0.0.0.0
export MCP_LOG_LEVEL=info
export MCP_CONFIG_FILE=/app/configs/production.yaml

# Development environment variables
export MCP_SERVER_PORT=3000
export MCP_SERVER_HOST=localhost
export MCP_LOG_LEVEL=debug
export MCP_CONFIG_FILE=/app/configs/development.yaml
```

### Health Checks
```bash
# Health check endpoint
curl http://localhost:8080/health

# Readiness check
curl http://localhost:8080/ready

# Metrics endpoint
curl http://localhost:8080/metrics
```

### Scaling Considerations
- Stateless design for horizontal scaling
- Connection pooling for database resources
- Graceful shutdown handling
- Resource limits and quotas
- Load balancing configuration

## MCP Server Specifics

### Protocol Implementation
- JSON-RPC 2.0 over stdio/TCP
- Tool and resource registration
- Client capability negotiation
- Error handling and validation

### Tool Registration
```go
// Tool registration in internal/tools/registry.go
func RegisterTools(server *mcp.Server) error {
    // Register Echo tool
    if err := server.RegisterTool("echo", echo.New()); err != nil {
        return fmt.Errorf("failed to register echo tool: %w", err)
    }
    
    // Register filesystem tool
    if err := server.RegisterTool("filesystem", filesystem.New()); err != nil {
        return fmt.Errorf("failed to register filesystem tool: %w", err)
    }
    
    return nil
}
```

### Error Handling
- Structured error responses
- Request/response logging
- Circuit breaker patterns
- Timeout handling
- Retry mechanisms

### Security Best Practices
- Input validation and sanitization
- Rate limiting
- Authentication and authorization
- Secure configuration management
- Audit logging

## Tactical Implementation Plan

This implementation follows a git-driven development approach with small, incremental commits that maintain a runnable state throughout development. Each step requires creating a detailed specification before implementation:

### Step 0: Repository Initialization
**Specification**: Create `specs/step-0-git-init.md` before executing
```bash
git init
```

### Commit 1: Project Foundation
**Specification**: Create `specs/commit-1-project-foundation.md` before implementing
- Initialize Go module with `go mod init mcp-server`
- Create basic project structure (`cmd/`, `internal/`, `pkg/`, `configs/`)
- Set up structured logging with slog
- Add `.gitignore` for Go projects
- **Error Handling**: Logging initialization errors, module validation

### Commit 2: Minimal Runnable Server
**Specification**: Create `specs/commit-2-minimal-server.md` before implementing
- Create `cmd/mcp-server/main.go` with basic HTTP server
- Add configuration loading from environment variables
- Implement graceful shutdown with context cancellation
- Add startup and shutdown logging
- **Error Handling**: Server start/stop errors, configuration validation errors

### Commit 3: MCP Protocol Dependencies
**Specification**: Create `specs/commit-3-mcp-dependencies.md` before implementing
- Add MCP Go SDK dependencies to `go.mod`
- Create basic MCP server initialization in `internal/server/`
- Add connection error handling and timeout configuration
- **Error Handling**: Dependency loading errors, connection failures

### Commit 4: MCP Server Core
**Specification**: Create `specs/commit-4-mcp-server-core.md` before implementing
- Implement MCP server initialization and lifecycle management
- Add protocol negotiation and client capability handling
- Create health check endpoints (`/health`, `/ready`)
- **Error Handling**: Protocol errors, negotiation failures, health check errors

### Commit 5: Configuration Management
**Specification**: Create `specs/commit-5-configuration.md` before implementing
- Create `internal/config/config.go` with environment-based configuration
- Add validation for required configuration parameters
- Implement configuration loading with defaults and overrides
- **Error Handling**: Configuration validation errors, missing parameters, type conversion errors

### Commit 6: Tool Registration Framework
**Specification**: Create `specs/commit-6-tool-framework.md` before implementing
- Create `internal/tools/registry.go` for tool management
- Implement tool registration and discovery mechanisms
- Add tool validation and lifecycle management
- **Error Handling**: Tool registration errors, validation failures, lifecycle errors

### Commit 7: First Example Tool
**Specification**: Create `specs/commit-7-first-tool.md` before implementing
- Implement Echo tool in `internal/tools/echo/`
- Add tool execution with proper error handling
- Create tool-specific tests with error scenarios
- Add request/response logging for tool invocations
- **Error Handling**: Parameter validation errors, input processing failures

### Commit 8: Resource Registration Framework
**Specification**: Create `specs/commit-8-resource-framework.md` before implementing
- Create `internal/resources/registry.go` for resource management
- Implement resource registration and access control
- Add resource validation and caching mechanisms
- **Error Handling**: Resource registration errors, access control failures, caching errors

### Commit 9: First Example Resource
**Specification**: Create `specs/commit-9-first-resource.md` before implementing
- Implement file system resource in `internal/resources/files/`
- Add resource access with proper permission checking
- Create resource-specific tests and error scenarios
- Add audit logging for resource access
- **Error Handling**: File system errors, permission denied, path validation errors

### Commit 10: Health and Diagnostics
**Specification**: Create `specs/commit-10-health-diagnostics.md` before implementing
- Enhance health check endpoint with dependency status
- Add metrics collection and Prometheus endpoint
- Implement diagnostic information gathering
- **Error Handling**: Health check failures, metrics collection errors, diagnostic errors

### Commit 11: Docker Configuration
**Specification**: Create `specs/commit-11-docker-config.md` before implementing
- Create `Dockerfile` with multi-stage build
- Add `docker-compose.yml` for development environment
- Configure container health checks and resource limits
- **Error Handling**: Container startup errors, resource limit violations, health check failures

### Commit 12: Production Deployment
**Specification**: Create `specs/commit-12-production-deploy.md` before implementing
- Create Kubernetes deployment and service manifests
- Add production configuration files
- Implement deployment scripts with error handling
- **Error Handling**: Deployment failures, configuration errors, service startup issues

### Commit 13: Monitoring and Metrics
**Specification**: Create `specs/commit-13-monitoring.md` before implementing
- Add comprehensive metrics collection (request counts, latencies, errors)
- Implement distributed tracing with OpenTelemetry
- Create alerting rules and dashboards
- **Error Handling**: Metrics collection failures, tracing errors, alerting issues

### Commit 14: Comprehensive Testing
**Specification**: Create `specs/commit-14-testing.md` before implementing
- Add unit tests for all components with error scenarios
- Create integration tests for MCP protocol compliance
- Implement performance and load testing
- **Error Handling**: Test failures, assertion errors, performance degradation

### Commit 15: Production Hardening
**Specification**: Create `specs/commit-15-production-hardening.md` before implementing
- Implement graceful shutdown with cleanup procedures
- Add circuit breakers and rate limiting
- Create backup and recovery procedures
- **Error Handling**: Shutdown errors, cleanup failures, recovery errors

**Key Principles:**
- **Specification First**: Create detailed spec file before any implementation
- Each commit maintains a runnable state with passing tests and no compiler warnings
- Error handling and logging are integral to each increment
- Tests are added alongside implementation
- Configuration and deployment considerations are included early
- Security and production readiness are built in, not added later
- **Traceability**: Every commit references its corresponding specification

**Commit Message Guidelines:**
- **No External Tool References**: Commit messages must NOT contain any references to AI tools, code assistants, or automated code generation tools
- **Clean Attribution**: Do not include "Generated with", "Co-Authored-By", or similar automated tool attributions
- **Professional Tone**: Use clear, descriptive commit messages that focus on the technical changes made
- **Standard Format**: Follow conventional commit format with clear summary and detailed description
- **Human Authorship**: All commits should appear as if written by human developers

## Maintenance and Operations

### Logging Configuration
```yaml
# Production logging configuration
logging:
  level: info
  format: json
  output: stdout
  fields:
    service: mcp-server
    version: "1.0.0"
```

### Metrics and Monitoring
- Prometheus metrics endpoint
- Grafana dashboards
- Alert rules for error rates and latencies
- Log aggregation and analysis

### Performance Optimization
- Connection pooling
- Request batching
- Caching strategies
- Memory optimization
- CPU profiling

### Security Hardening
- Regular dependency updates
- Security scanning
- Access control policies
- Audit logging
- Incident response procedures

### Backup and Recovery
- Configuration backup
- State persistence
- Disaster recovery procedures
- Data migration strategies

## Quick Start

1. **Initialize Repository**
   ```bash
   git init
   ```

2. **Setup Project**
   ```bash
   go mod init mcp-server
   mkdir -p cmd/mcp-server internal/{server,config,tools,resources,logger} configs deployments specs
   ```

3. **Run Development Server**
   ```bash
   go run cmd/mcp-server/main.go
   ```

4. **Test Health Endpoint**
   ```bash
   curl http://localhost:8080/health
   ```

5. **Build for Production**
   ```bash
   go build -ldflags="-w -s" -o bin/mcp-server ./cmd/mcp-server
   ```

This CLAUDE.md provides a comprehensive guide for implementing a production-ready MCP server in Go, with emphasis on proper error handling, structured logging, and incremental development through git commits.