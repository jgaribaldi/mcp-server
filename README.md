# MCP Server in Go

A Model Context Protocol (MCP) server implementation in Go 1.22.1.

## Features

TBD - Features will be added as tools and resources are implemented.

## Quick Start

### Prerequisites

- Go 1.22.1 or later
- Git

### Installation

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd mcp-server
   ```

2. Build the server:
   ```bash
   go build -o mcp-server ./cmd/mcp-server
   ```

3. Run the server:
   ```bash
   ./mcp-server
   ```

### Configuration

The server can be configured using environment variables:

- `MCP_SERVER_HOST`: Server host (default: "localhost")
- `MCP_SERVER_PORT`: Server port (default: 3000)
- `MCP_LOG_LEVEL`: Log level (default: "info")
- `MCP_LOG_FORMAT`: Log format - "json" or "text" (default: "json")
- `MCP_SERVICE_NAME`: Service name for logging (default: "mcp-server")
- `MCP_VERSION`: Version for logging (default: "dev")

Example:
```bash
export MCP_SERVER_HOST=0.0.0.0
export MCP_SERVER_PORT=8080
export MCP_LOG_LEVEL=debug
./mcp-server
```

## Development

### Building

```bash
# Build for development
go build -o mcp-server ./cmd/mcp-server

# Build for production
go build -ldflags="-w -s" -o mcp-server ./cmd/mcp-server
```

### Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

### Code Quality

```bash
# Format code
go fmt ./...

# Vet code
go vet ./...
```

## Project Structure

This project follows standard Go project layout:

- `cmd/`: Main applications
- `internal/`: Private application code
- `pkg/`: Library code that can be used by external applications
- `configs/`: Configuration files
- `deployments/`: Deployment configurations
- `scripts/`: Build and deployment scripts
- `tests/`: Integration tests
- `docs/`: Documentation
- `specs/`: Implementation specifications

## Contributing

Please read the specifications in the `specs/` directory before making changes. Each change should have a corresponding specification that outlines the implementation details.

## License

TBD