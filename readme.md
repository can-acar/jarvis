# Jarvis MCP Server

A Model Context Protocol (MCP) server implementation that provides AI assistant capabilities with comprehensive system tools.

## Overview

This MCP server enables seamless integration with AI models through the Model Context Protocol, offering a standardized way to interact with AI assistants and manage context effectively. Built with Go for high performance and reliability.

## Features

- **MCP Protocol Support**: Full implementation of the Model Context Protocol using mark3labs/mcp-go
- **Security-First Design**: Configurable command blocking and directory access controls
- **System Integration**: Terminal command execution with safety controls
- **File System Operations**: Secure file reading, writing, and manipulation
- **Text Editing Tools**: Advanced text editing capabilities with line-based operations
- **HTTP Fetch Tools**: Built-in web content fetching capabilities
- **Configuration Management**: Runtime configuration updates via MCP tools
- **Go Implementation**: High-performance server written in Go with robust error handling

## Installation

### Prerequisites

- Go 1.24.4 or later
- Git

### Setup

1. Clone the repository:
```bash
git clone <repository-url>
cd jarvis
```

2. Install dependencies:
```bash
go mod download
```

3. Build the project:
```bash
go build -o jarvis main.go
```

## Usage

### Starting the Server

```bash
./jarvis
```

The server will start and communicate via stdio using the MCP protocol.

### Configuration

The server uses a `config.yaml` file for configuration:

```yaml
# Security settings
blockedCommands:
  - rm
  - shutdown
  
# Default shell for command execution
defaultShell: bash

# Allowed directories for file operations
allowedDirectories:
  - /tmp
  - /var/log

# File operation limits
fileReadLineLimit: 1000
fileWriteLineLimit: 50

# Telemetry
telemetryEnabled: false
```

### Available Tools

The server provides the following MCP tools:

#### Configuration Tools
- `get-config` - Retrieve current server configuration
- `set-config` - Update server configuration values

#### Terminal Tools  
- `execute-command` - Execute shell commands with security controls
- `get-command-history` - Retrieve command execution history

#### File System Tools
- `read-file` - Read file contents with pagination support
- `write-file` - Write content to files
- `list-directory` - List directory contents
- `create-directory` - Create new directories
- `delete-file` - Delete files and directories
- `move-file` - Move/rename files and directories
- `file-info` - Get file metadata and information

#### Text Editing Tools
- `edit-file` - Perform complex text editing operations
- `search-replace` - Search and replace text in files
- `batch-edit` - Apply multiple edits to files

#### Fetch Tools
- `fetch-url` - Fetch content from web URLs
- `download-file` - Download files from remote sources

## Development

### Project Structure

```
jarvis/
├── main.go                    # Main server entry point
├── config.yaml               # Server configuration
├── go.mod                    # Go module definition
├── go.sum                    # Go module checksums
├── handlers/                 # MCP tool handlers
│   ├── config_handler.go     # Configuration management
│   ├── terminal_handler.go   # Terminal operations
│   ├── filesystem_handler.go # File system operations
│   ├── textediting_handler.go # Text editing tools
│   └── fetch_handler.go      # HTTP fetch operations
└── internal/                 # Internal packages
    ├── common/               # Shared utilities
    ├── config/               # Configuration management
    ├── terminal/             # Terminal utilities
    ├── filesystem/           # File system utilities
    ├── textedit/             # Text editing utilities
    ├── fetch/                # Fetch utilities
    └── types/                # Type definitions
```

### Building and Testing

```bash
# Build the project
go build -o jarvis main.go

# Run tests
go test ./...

# Run with verbose logging
go run main.go

# Build for different platforms
GOOS=windows GOARCH=amd64 go build -o jarvis.exe main.go
GOOS=linux GOARCH=amd64 go build -o jarvis-linux main.go
```

### Adding Custom Tools

To add new MCP tools:

1. Create a handler function in the appropriate handler file:
```go
func HandleNewTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Implementation here
    return mcp.NewToolResultText("Result"), nil
}
```

2. Register the tool in the corresponding internal package:
```go
func RegisterNewTools(s *server.MCPServer) {
    s.AddTool("new-tool", "Description", map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "param": map[string]interface{}{
                "type": "string",
                "description": "Parameter description",
            },
        },
        "required": []string{"param"},
    }, handlers.HandleNewTool)
}
```

3. Call the registration function in `main.go`.

### Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes and add tests
4. Ensure code follows Go conventions: `go fmt ./...`
5. Run tests: `go test ./...`
6. Commit your changes: `git commit -m 'Add feature'`
7. Push to the branch: `git push origin feature-name`
8. Submit a pull request

## Security Considerations

- Commands are sanitized and checked against blocked patterns
- File system access is restricted to allowed directories
- Command execution includes timeout controls
- All operations are logged for audit purposes
- Configuration can restrict dangerous operations

## Dependencies

- **mark3labs/mcp-go**: MCP protocol implementation for Go
- **google/uuid**: UUID generation
- **spf13/cast**: Type casting utilities

## API Documentation

### MCP Protocol Implementation

The server implements the following MCP capabilities:

- **Tools**: Execute system operations (file I/O, terminal commands, etc.)
- **Resources**: Access to file system resources with proper access controls
- **Prompts**: Template-based prompt management

### Tool Categories

1. **System Tools**: Terminal command execution, process management
2. **File Tools**: File operations with security boundaries
3. **Text Tools**: Advanced text manipulation and editing
4. **Network Tools**: HTTP requests and content fetching
5. **Config Tools**: Runtime configuration management

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For support and questions:

- Create an issue on GitHub
- Check the documentation and code comments
- Review existing issues and discussions

## Performance Notes

- Built with Go for high concurrency and performance
- Efficient file operations with configurable limits
- Memory-conscious design for large file handling
- Optimized for MCP protocol communication over stdio