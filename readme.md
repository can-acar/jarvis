Go dilinde MCP tool server,  Kaynak: `https://mcpgolang.com/`. mcp-golang paketi kullanılacak.
Projenin adı jarvis-mcp-server. projedeki oluşturalacak tool'un içeriği
### Configuration
    - `get_config` =>  Get the complete server configuration as JSON (includes blockedCommands, defaultShell, allowedDirectories, fileReadLineLimit, fileWriteLineLimit, telemetryEnabled)
    - `set_config_value` =>  Set a specific configuration value by key. 
        Available settings: 
         `blockedCommands`: Array of shell commands that cannot be executed
         `defaultShell`: Shell to use for commands (e.g., bash, zsh, powershell) `allowedDirectories`: Array of filesystem paths the server can access for file operations (⚠️ terminal commands can still access files outside these directories) `fileReadLineLimit`: Maximum lines to read at once (default: 1000)
         `fileWriteLineLimit`: Maximum lines to write at once (default: 50)
         `telemetryEnabled`: Enable/disable telemetry (boolean)
### Terminal
    - `execute_command` => Execute a terminal command with configurable timeout and shell selection
    - `read_output` => Read new output from a running terminal session
    - `force_terminate` => Force terminate a running terminal session
    - `list_sessions` =>  List all active terminal sessions
    - `list_processes` => List all running processes with detailed information
    - `kill_process` => Terminate a running process by PID
### Filesystem
    - `read_file`=> Read contents from local filesystem or URLs with line-based pagination (supports positive/negative offset and length parameters)
    - `read_multiple_files` => Read multiple files simultaneously
    - `write_file` => Write file contents with options for rewrite or append mode (uses configurable line limits)
    - `create_directory` => Create a new directory or ensure it exists 
    - `list_directory` => Get detailed listing of files and directories
    - `move_file` => Move or rename files and directories
    - `search_files` => Find files by name using case-insensitive substring
    - `search_code` => Search for text/code patterns within file contents using ripgrep 
    - `get_file_info` => Retrieve detailed metadata about a file or 
### Text Editing
    - `edit_block` => Apply targeted text replacements with enhanced prompting for smaller edits (includes character-level diff feedback)
    - `edit_file` => Edit files with line-based replacements, supports multiple edits in one go 
    - `edit_multiple_files` => Edit multiple files simultaneously with line-based replacements 

### Fetching
    - `fetch_web` =>Fetch represents a structured HTTP request for fetching resources
    - `fetch_web_content` =>Fetch web content with options for headers, method, and body
    - `fetch_web_file` =>Fetch a file from a URL and save it locally
    - `fetch_web_image` =>Fetch an image from a URL and save it locally
    - `fetch_web_json` =>Fetch JSON data from a URL and parse it