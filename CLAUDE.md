# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Jarvis is a Model Context Protocol (MCP) server implementation written in Go that provides AI assistant capabilities with comprehensive system tools. It acts as a secure bridge between AI models and system operations, offering controlled access to terminal commands, file operations, text editing, and web fetching capabilities.

## Key Commands

### Build and Development
```bash
# Build the project
go build -o jarvis main.go

# Run the server directly with Go
go run main.go

# Run tests
go test ./...

# Download dependencies
go mod download

# Format code (Go standard)
go fmt ./...

# Cross-platform builds
GOOS=windows GOARCH=amd64 go build -o jarvis.exe main.go
GOOS=linux GOARCH=amd64 go build -o jarvis-linux main.go
```

### Running the Server

#### MCP Server Mode (Default)
```bash
# Start the MCP server (communicates via stdio)
./jarvis

# Start with specific LLM model
./jarvis --model ollama:qwen3

# Start with custom configuration file
./jarvis --config ~/.ollama-mcp.json

# Combined usage
./jarvis -m ollama:qwen3 --config ./custom-config.json --system-prompt "Custom prompt"
```

#### Interactive Agent Mode
```bash
# Start interactive agent with model
./jarvis --model ollama:qwen3 --interactive

# Interactive mode with system prompt
./jarvis -m ollama:qwen3 -i --system-prompt "You are Jarvis, a helpful AI assistant"

# Interactive mode with custom config
./jarvis -m ollama:qwen3 --config ~/.ollama-mcp.json --interactive

# Show help and version
./jarvis --help
./jarvis --version
```

#### Agent Usage Examples
Once in interactive mode (`jarvis >> `), you can use:
```
Analyze current directory
List all Go files in this project
What kind of project is this?
Explain the main.go file
Show me the project structure
```

## High-Level Architecture

### Core Components

**MCP Protocol Integration**: Built on `mark3labs/mcp-go` library, the server implements the Model Context Protocol specification with full support for tools, resources, and prompts.

**Security-First Design**: The server operates with configurable security boundaries:
- Command blocking system prevents execution of dangerous commands
- Directory access controls limit file operations to allowed paths
- All operations are validated and logged for audit purposes

**Modular Tool System**: Tools are organized into logical modules:
- **Config Tools** (`internal/config/`): Runtime configuration management
- **Terminal Tools** (`internal/terminal/`): Shell command execution with security controls  
- **Filesystem Tools** (`internal/filesystem/`): File operations with path validation
- **Text Editing Tools** (`internal/textedit/`): Advanced text manipulation and editing
- **Fetch Tools** (`internal/fetch/`): HTTP requests and file downloading
- **LLM Integration** (`internal/llm/`): Model communication and prompt handling
- **Agent Module** (`internal/agent/`): Interactive AI agent functionality

### Configuration System

The server uses a hybrid configuration approach:
- Default configuration hardcoded in `internal/common/common.go`
- Runtime configuration loaded from `config.yaml` (if present)
- Dynamic configuration updates via MCP tools
- Configuration persistence to `~/jarvis-mcp.json`

**Key Configuration Areas**:
- `blockedCommands`: Patterns of commands that are forbidden to execute
- `allowedDirectories`: Filesystem paths where operations are permitted
- `defaultShell`: Shell program used for command execution (default: bash)
- File operation limits: `fileReadLineLimit`, `fileWriteLineLimit`
- `telemetryEnabled`: Controls logging and telemetry features
- `modelConfig`: LLM model configuration including model name, system prompt, and config file path

### Request Flow

#### MCP Server Mode
1. **MCP Protocol**: Client sends tool request via stdio using MCP protocol
2. **Tool Routing**: Main server routes request to appropriate handler in `handlers/` directory
3. **Security Validation**: Internal packages validate permissions and sanitize inputs
4. **Operation Execution**: Core logic executes the requested operation
5. **Result Formatting**: Response formatted as MCP-compliant tool result
6. **Audit Logging**: Operation logged for security and debugging purposes

#### Interactive Agent Mode
1. **User Input**: User enters natural language query via interactive prompt
2. **Context Gathering**: Agent gathers current directory context and system information
3. **Prompt Construction**: System prompt + context + user query combined into LLM prompt
4. **Model Communication**: Request sent to configured LLM (e.g., Ollama API)
5. **Response Processing**: LLM response parsed and formatted for display
6. **Interactive Loop**: Process continues until user exits session

### Data Types and Interfaces

`internal/types/types.go` defines the core data structures:
- `ServerConfig`: Central configuration structure including optional model configuration
- `ModelConfig`: LLM model configuration with model name, config file path, and system prompt
- `CommandLineArgs`: Parsed command line arguments for model and configuration setup
- `AgentRequest`: User request structure for interactive agent mode
- `AgentResponse`: Agent response structure with success/error handling
- `EditOperation`: Represents text editing operations with line-based targeting
- `HTTPRequestConfig`: Configures web requests with headers, timeouts, validation
- `CommandExecutionConfig`: Defines command execution parameters
- `OperationResult`: Standardized response format for all operations

### Thread Safety and State Management

The server uses singleton pattern with mutex protection for configuration state (`internal/common/common.go`). All configuration updates are thread-safe and automatically persisted to disk.

## Development Notes

### Adding New Tools
1. Create handler function in appropriate `handlers/` file
2. Register tool in corresponding `internal/` package with schema definition
3. Call registration function in `main.go`
4. Follow existing patterns for parameter validation and error handling

### Security Considerations
- All user inputs are sanitized through `common.SanitizeCommand()`
- File paths validated through `common.IsPathAllowed()`
- Command execution checked against `common.IsCommandBlocked()`
- No operations bypass the security validation layer

### Testing Strategy
The codebase uses Go's standard testing framework. Test files should be created alongside source files with `_test.go` suffix.