# Jarvis MCP Server Development Guidelines

This document provides guidelines and information for developing and maintaining the Jarvis MCP Server project.

## Build and Configuration Instructions

### Prerequisites
- Go 1.24.4 or later
- The github.com/mark3labs/mcp-go package (v0.32.0)

### Building the Project
1. Clone the repository
2. Install dependencies:
   ```
   go mod download
   ```
3. Build the project:
   ```
   go build -o jarvis.exe
   ```

### Configuration
The application uses a `common.yaml` file in the root directory for configuration. The following settings can be configured:

- `blockedCommands`: Array of shell commands that cannot be executed
- `defaultShell`: Shell to use for commands (e.g., bash, zsh, powershell)
- `allowedDirectories`: Array of filesystem paths the server can access for file operations
- `fileReadLineLimit`: Maximum lines to read at once (default: 1000)
- `fileWriteLineLimit`: Maximum lines to write at once (default: 50)
- `telemetryEnabled`: Enable/disable telemetry (boolean)

Example configuration:
```yaml
blockedCommands:
  - rm
  - shutdown
defaultShell: bash
allowedDirectories:
  - /tmp
  - /var/log
fileReadLineLimit: 1000
fileWriteLineLimit: 50
telemetryEnabled: false
```

## Testing Information

### Running Tests
To run all tests in the project:
```
go test ./...
```

To run tests for a specific package:
```
go test ./internal/config
```

To run tests with verbose output:
```
go test -v ./...
```

### Adding New Tests
1. Create a new file with the naming convention `*_test.go` in the same package as the code you're testing.
2. Import the testing package: `import "testing"`
3. Write test functions with the naming convention `TestXxx` where `Xxx` is the name of the function or feature you're testing.
4. Use the `t.Error()`, `t.Errorf()`, `t.Fatal()`, or `t.Fatalf()` methods to report test failures.

Example test structure:
```go
package mypackage

import (
    "testing"
)

func TestMyFunction(t *testing.T) {
    result := MyFunction()
    expected := "expected result"
    if result != expected {
        t.Errorf("MyFunction() = %v, want %v", result, expected)
    }
}
```

## Project Structure and Conventions

### Package Organization
- `main.go`: Entry point of the application
- `internal/`: Contains all internal packages
  - `config/`: Configuration handling
  - `terminal/`: Terminal command execution
  - `filesystem/`: File system operations
  - `textedit/`: Text editing operations
  - `fetch/`: Web fetching operations
  - `common/`: Common utilities

### Code Style
- The project uses standard Go code style and conventions
- Comments are written in Turkish and English
- Each tool is registered with the MCP server using the `RegisterXxxTools` functions
- Tool handlers follow the naming convention `handleXxx`

### Tool Categories
The project provides several categories of tools:

1. **Configuration Tools**
   - `get_config`: Get the complete server configuration as JSON
   - `set_config_value`: Set a specific configuration value by key

2. **Terminal Tools**
   - `execute_command`: Execute a terminal command with configurable timeout and shell selection
   - `read_output`: Read new output from a running terminal session
   - `force_terminate`: Force terminate a running terminal session
   - `list_sessions`: List all active terminal sessions
   - `list_processes`: List all running processes with detailed information
   - `kill_process`: Terminate a running process by PID

3. **Filesystem Tools**
   - `read_file`: Read contents from local filesystem or URLs with line-based pagination
   - `read_multiple_files`: Read multiple files simultaneously
   - `write_file`: Write file contents with options for rewrite or append mode
   - `create_directory`: Create a new directory or ensure it exists
   - `list_directory`: Get detailed listing of files and directories
   - `move_file`: Move or rename files and directories
   - `search_files`: Find files by name using case-insensitive substring
   - `search_code`: Search for text/code patterns within file contents using ripgrep
   - `get_file_info`: Retrieve detailed metadata about a file

4. **Text Editing Tools**
   - `edit_block`: Apply targeted text replacements with enhanced prompting for smaller edits
   - `edit_file`: Edit files with line-based replacements, supports multiple edits in one go
   - `edit_multiple_files`: Edit multiple files simultaneously with line-based replacements

5. **Fetching Tools**
   - `fetch_web`: Fetch represents a structured HTTP request for fetching resources
   - `fetch_web_content`: Fetch web content with options for headers, method, and body
   - `fetch_web_file`: Fetch a file from a URL and save it locally
   - `fetch_web_image`: Fetch an image from a URL and save it locally
   - `fetch_web_json`: Fetch JSON data from a URL and parse it

### Security Considerations
- The application implements security measures such as blocking dangerous commands
- File operations are restricted to allowed directories
- Terminal commands have configurable timeouts