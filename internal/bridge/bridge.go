package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"jarvis/internal/common"
	"jarvis/internal/types"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ToolBridge provides access to Jarvis MCP tools for the agent
type ToolBridge struct {
	availableTools map[string]ToolDefinition
	mcpServers     map[string]*MCPServerConnection
}

// MCPServerConnection represents a connection to an external MCP server
type MCPServerConnection struct {
	Config    types.MCPServerConfig
	Process   *exec.Cmd
	IsRunning bool
	Tools     map[string]ToolDefinition
}

// ToolDefinition describes an available tool
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Handler     func(context.Context, map[string]interface{}) (interface{}, error)
	IsMCPTool   bool   `json:"is_mcp_tool,omitempty"`
	MCPServer   string `json:"mcp_server,omitempty"`
}

// ToolCall represents a tool call request
type ToolCall struct {
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// NewToolBridge creates a new tool bridge with available MCP tools
func NewToolBridge() *ToolBridge {
	bridge := &ToolBridge{
		availableTools: make(map[string]ToolDefinition),
		mcpServers:     make(map[string]*MCPServerConnection),
	}

	// Register built-in tools
	bridge.registerFilesystemTools()
	bridge.registerTerminalTools()
	bridge.registerTextEditingTools()
	bridge.registerFetchTools()
	bridge.registerConfigTools()

	// Initialize external MCP servers
	bridge.initializeMCPServers()

	return bridge
}

// GetAvailableTools returns a list of available tools for the LLM
func (b *ToolBridge) GetAvailableTools() []ToolDefinition {
	tools := make([]ToolDefinition, 0, len(b.availableTools))
	for _, tool := range b.availableTools {
		tools = append(tools, tool)
	}
	return tools
}

// ExecuteTool executes a tool call and returns the result
func (b *ToolBridge) ExecuteTool(ctx context.Context, toolCall ToolCall) ToolResult {
	tool, exists := b.availableTools[toolCall.Name]
	if !exists {
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Tool '%s' not found", toolCall.Name),
		}
	}

	// Assess risk before execution
	riskLevel := b.assessOperationRisk(toolCall)
	if riskLevel == "critical" {
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Critical-risk operation detected. Manual confirmation required for: %s", toolCall.Name),
		}
	}

	// Log high-risk operations but allow them
	if riskLevel == "high" {
		log.Printf("Warning: High-risk operation detected: %s", toolCall.Name)
	}

	// Execute the tool
	result, err := tool.Handler(ctx, toolCall.Parameters)
	if err != nil {
		// Try alternative approaches on failure
		if alternativeResult := b.tryAlternativeApproaches(ctx, toolCall, err); alternativeResult != nil {
			return *alternativeResult
		}

		return ToolResult{
			Success: false,
			Error:   err.Error(),
		}
	}

	return ToolResult{
		Success: true,
		Result:  result,
	}
} // registerFilesystemTools registers filesystem-related tools
func (b *ToolBridge) registerFilesystemTools() {
	// List directory contents
	b.availableTools["list-directory"] = ToolDefinition{
		Name:        "list-directory",
		Description: "List contents of a directory",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Directory path to list",
				},
				"show_hidden": map[string]interface{}{
					"type":        "boolean",
					"description": "Include hidden files",
					"default":     false,
				},
			},
			"required": []string{"path"},
		},
		Handler: b.handleListDirectory,
	}

	// Read file contents
	b.availableTools["read-file"] = ToolDefinition{
		Name:        "read-file",
		Description: "Read contents of a file",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File path to read",
				},
				"lines": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of lines to read",
					"default":     1000,
				},
			},
			"required": []string{"path"},
		},
		Handler: b.handleReadFile,
	}

	// Get file info
	b.availableTools["file-info"] = ToolDefinition{
		Name:        "file-info",
		Description: "Get information about a file or directory",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File or directory path",
				},
			},
			"required": []string{"path"},
		},
		Handler: b.handleFileInfo,
	}
}

// registerTerminalTools registers terminal-related tools
func (b *ToolBridge) registerTerminalTools() {
	// Execute command
	b.availableTools["execute-command"] = ToolDefinition{
		Name:        "execute-command",
		Description: "Execute a shell command",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "Command to execute",
				},
				"working_dir": map[string]interface{}{
					"type":        "string",
					"description": "Working directory for command execution",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Command timeout in seconds",
					"default":     30,
				},
			},
			"required": []string{"command"},
		},
		Handler: b.handleExecuteCommand,
	}
}

// registerTextEditingTools registers text editing tools
func (b *ToolBridge) registerTextEditingTools() {
	// Search in files
	b.availableTools["search-files"] = ToolDefinition{
		Name:        "search-files",
		Description: "Search for text patterns in files",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Text pattern to search for",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Directory or file path to search in",
				},
				"file_extensions": map[string]interface{}{
					"type":        "array",
					"description": "File extensions to include (e.g., [\".go\", \".js\"])",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"case_sensitive": map[string]interface{}{
					"type":        "boolean",
					"description": "Case sensitive search",
					"default":     false,
				},
			},
			"required": []string{"pattern", "path"},
		},
		Handler: b.handleSearchFiles,
	}
}

// registerFetchTools registers fetch-related tools
func (b *ToolBridge) registerFetchTools() {
	// Fetch URL
	b.availableTools["fetch-url"] = ToolDefinition{
		Name:        "fetch-url",
		Description: "Fetch content from a URL",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "URL to fetch",
				},
				"method": map[string]interface{}{
					"type":        "string",
					"description": "HTTP method (GET, POST, etc.)",
					"default":     "GET",
				},
			},
			"required": []string{"url"},
		},
		Handler: b.handleFetchURL,
	}
}

// registerConfigTools registers configuration tools
func (b *ToolBridge) registerConfigTools() {
	// Get config
	b.availableTools["get-config"] = ToolDefinition{
		Name:        "get-config",
		Description: "Get current Jarvis configuration",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: b.handleGetConfig,
	}
}

// Tool handler implementations

func (b *ToolBridge) handleListDirectory(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter is required")
	}

	showHidden := false
	if h, ok := params["show_hidden"].(bool); ok {
		showHidden = h
	}

	// Check if path is allowed
	if !common.IsPathAllowed(path) {
		return nil, fmt.Errorf("access to path '%s' is not allowed", path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %v", err)
	}

	var result []map[string]interface{}
	for _, entry := range entries {
		if !showHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		result = append(result, map[string]interface{}{
			"name":     entry.Name(),
			"is_dir":   entry.IsDir(),
			"size":     info.Size(),
			"mod_time": info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	return map[string]interface{}{
		"path":    path,
		"entries": result,
		"count":   len(result),
	}, nil
}

func (b *ToolBridge) handleReadFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter is required")
	}

	// Check if path is allowed
	if !common.IsPathAllowed(path) {
		return nil, fmt.Errorf("access to path '%s' is not allowed", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	lines := strings.Split(string(content), "\n")

	// Apply line limit if specified
	maxLines := 1000
	if l, ok := params["lines"].(float64); ok {
		maxLines = int(l)
	}

	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	return map[string]interface{}{
		"path":        path,
		"content":     strings.Join(lines, "\n"),
		"total_lines": len(strings.Split(string(content), "\n")),
		"lines_shown": len(lines),
		"truncated":   len(strings.Split(string(content), "\n")) > maxLines,
	}, nil
}

func (b *ToolBridge) handleFileInfo(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter is required")
	}

	// Check if path is allowed
	if !common.IsPathAllowed(path) {
		return nil, fmt.Errorf("access to path '%s' is not allowed", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	return map[string]interface{}{
		"name":     info.Name(),
		"path":     path,
		"is_dir":   info.IsDir(),
		"size":     info.Size(),
		"mode":     info.Mode().String(),
		"mod_time": info.ModTime().Format("2006-01-02 15:04:05"),
	}, nil
}

func (b *ToolBridge) handleExecuteCommand(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	command, ok := params["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command parameter is required")
	}

	// Check if command is blocked
	if common.IsCommandBlocked(command) {
		return nil, fmt.Errorf("command is blocked for security reasons")
	}

	// Get working directory
	workingDir := ""
	if wd, ok := params["working_dir"].(string); ok {
		workingDir = wd
		if !common.IsPathAllowed(workingDir) {
			return nil, fmt.Errorf("access to working directory '%s' is not allowed", workingDir)
		}
	}

	// Get timeout
	timeout := 30
	if t, ok := params["timeout"].(float64); ok {
		timeout = int(t)
	}

	// Execute the command using the actual terminal handler
	result, err := b.executeSystemCommand(command, workingDir, timeout)
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %v", err)
	}

	return result, nil
}

func (b *ToolBridge) handleSearchFiles(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	pattern, ok := params["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("pattern parameter is required")
	}

	searchPath, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter is required")
	}

	// Check if path is allowed
	if !common.IsPathAllowed(searchPath) {
		return nil, fmt.Errorf("access to path '%s' is not allowed", searchPath)
	}

	var results []map[string]interface{}

	// Get file extensions filter
	var extensions []string
	if exts, ok := params["file_extensions"].([]interface{}); ok {
		for _, ext := range exts {
			if extStr, ok := ext.(string); ok {
				extensions = append(extensions, extStr)
			}
		}
	}

	// Case sensitivity
	caseSensitive := false
	if cs, ok := params["case_sensitive"].(bool); ok {
		caseSensitive = cs
	}

	// Walk the directory
	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			return nil
		}

		// Check file extension filter
		if len(extensions) > 0 {
			ext := filepath.Ext(path)
			found := false
			for _, allowedExt := range extensions {
				if strings.EqualFold(ext, allowedExt) {
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		}

		// Read and search file
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip unreadable files
		}

		contentStr := string(content)
		searchPattern := pattern
		if !caseSensitive {
			contentStr = strings.ToLower(contentStr)
			searchPattern = strings.ToLower(pattern)
		}

		if strings.Contains(contentStr, searchPattern) {
			// Count occurrences
			count := strings.Count(contentStr, searchPattern)
			results = append(results, map[string]interface{}{
				"file":    path,
				"matches": count,
			})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("search failed: %v", err)
	}

	return map[string]interface{}{
		"pattern":     pattern,
		"search_path": searchPath,
		"results":     results,
		"total_files": len(results),
	}, nil
}

func (b *ToolBridge) handleFetchURL(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	url, ok := params["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url parameter is required")
	}

	// For now, return a placeholder - would need full fetch integration
	log.Printf("Tool bridge would fetch URL: %s", url)

	return map[string]interface{}{
		"url":    url,
		"status": "URL fetch not fully implemented in bridge mode",
	}, nil
}

func (b *ToolBridge) handleGetConfig(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	configJSON, err := common.GetJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %v", err)
	}

	return config, nil
}

// executeSystemCommand executes a system command with security controls
func (b *ToolBridge) executeSystemCommand(command, workingDir string, timeoutSec int) (map[string]interface{}, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// Sanitize command
	cleanCommand := common.SanitizeCommand(command)
	if cleanCommand != command {
		return nil, fmt.Errorf("command contains invalid characters")
	}

	// Prepare command execution
	var cmd *exec.Cmd

	// Use shell to execute complex commands
	shell := common.Get().DefaultShell
	if shell == "" {
		shell = "bash"
	}

	cmd = exec.CommandContext(ctx, shell, "-c", command)

	// Set working directory if provided
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// Execute command and capture output
	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	// Prepare result
	result := map[string]interface{}{
		"command":     command,
		"working_dir": workingDir,
		"duration":    fmt.Sprintf("%.2fs", duration.Seconds()),
		"exit_code":   0,
		"output":      string(output),
		"success":     err == nil,
	}

	// Handle command errors
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result["exit_code"] = exitError.ExitCode()
			result["error"] = fmt.Sprintf("Command failed with exit code %d", exitError.ExitCode())
		} else {
			result["error"] = fmt.Sprintf("Command execution error: %v", err)
		}
		result["success"] = false
	}

	// Log the execution for audit
	log.Printf("Command executed: %s | Success: %t | Duration: %s", command, result["success"], result["duration"])

	return result, nil
}

// initializeMCPServers initializes external MCP servers from configuration
func (b *ToolBridge) initializeMCPServers() {
	cfg := common.Get()
	if len(cfg.MCP) == 0 {
		log.Println("No external MCP servers configured")
		return
	}

	for _, mcpConfig := range cfg.MCP {
		if !mcpConfig.Enabled {
			log.Printf("MCP server %s is disabled, skipping", mcpConfig.Name)
			continue
		}

		log.Printf("Initializing MCP server: %s", mcpConfig.Name)

		connection := &MCPServerConnection{
			Config:    mcpConfig,
			IsRunning: false,
			Tools:     make(map[string]ToolDefinition),
		}

		// Start the MCP server process
		if err := b.startMCPServer(connection); err != nil {
			log.Printf("Failed to start MCP server %s: %v", mcpConfig.Name, err)
			continue
		}

		// Discover tools from the MCP server
		if err := b.discoverMCPTools(connection); err != nil {
			log.Printf("Failed to discover tools from MCP server %s: %v", mcpConfig.Name, err)
			continue
		}

		b.mcpServers[mcpConfig.Name] = connection
		log.Printf("Successfully initialized MCP server %s with %d tools", mcpConfig.Name, len(connection.Tools))
	}
}

// startMCPServer starts an external MCP server process
func (b *ToolBridge) startMCPServer(connection *MCPServerConnection) error {
	config := connection.Config

	// Create command
	cmd := exec.Command(config.Command, config.Args...)

	// Set working directory if specified
	if config.WorkingDir != "" {
		cmd.Dir = config.WorkingDir
	}

	// Set environment variables
	if len(config.Environment) > 0 {
		env := os.Environ()
		for key, value := range config.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
	}

	// For now, we'll start the process but not manage stdio pipes
	// In a full implementation, we'd set up JSON-RPC communication
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server process: %v", err)
	}

	connection.Process = cmd
	connection.IsRunning = true

	// Wait a moment for the server to start
	time.Sleep(100 * time.Millisecond)

	return nil
}

// discoverMCPTools discovers available tools from an MCP server
func (b *ToolBridge) discoverMCPTools(connection *MCPServerConnection) error {
	// In a full implementation, this would use JSON-RPC to query the MCP server
	// For now, we'll add some example tools based on the server name

	serverName := connection.Config.Name

	// Add example tools based on server type
	switch {
	case strings.Contains(strings.ToLower(serverName), "filesystem"):
		b.addMCPFilesystemTools(connection)
	case strings.Contains(strings.ToLower(serverName), "database"):
		b.addMCPDatabaseTools(connection)
	case strings.Contains(strings.ToLower(serverName), "web"):
		b.addMCPWebTools(connection)
	default:
		log.Printf("Unknown MCP server type: %s, no default tools added", serverName)
	}

	return nil
}

// addMCPFilesystemTools adds filesystem-related MCP tools
func (b *ToolBridge) addMCPFilesystemTools(connection *MCPServerConnection) {
	serverName := connection.Config.Name

	// Example MCP filesystem tool
	toolName := fmt.Sprintf("mcp_%s_list_files", strings.ReplaceAll(serverName, "-", "_"))
	connection.Tools[toolName] = ToolDefinition{
		Name:        toolName,
		Description: fmt.Sprintf("List files using %s MCP server", serverName),
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Directory path to list",
				},
			},
			"required": []string{"path"},
		},
		Handler:   b.createMCPToolHandler(connection, toolName),
		IsMCPTool: true,
		MCPServer: serverName,
	}

	// Register the tool in the main tools map
	b.availableTools[toolName] = connection.Tools[toolName]
}

// addMCPDatabaseTools adds database-related MCP tools
func (b *ToolBridge) addMCPDatabaseTools(connection *MCPServerConnection) {
	serverName := connection.Config.Name

	toolName := fmt.Sprintf("mcp_%s_query", strings.ReplaceAll(serverName, "-", "_"))
	connection.Tools[toolName] = ToolDefinition{
		Name:        toolName,
		Description: fmt.Sprintf("Execute database query using %s MCP server", serverName),
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "SQL query to execute",
				},
			},
			"required": []string{"query"},
		},
		Handler:   b.createMCPToolHandler(connection, toolName),
		IsMCPTool: true,
		MCPServer: serverName,
	}

	b.availableTools[toolName] = connection.Tools[toolName]
}

// addMCPWebTools adds web-related MCP tools
func (b *ToolBridge) addMCPWebTools(connection *MCPServerConnection) {
	serverName := connection.Config.Name

	toolName := fmt.Sprintf("mcp_%s_fetch", strings.ReplaceAll(serverName, "-", "_"))
	connection.Tools[toolName] = ToolDefinition{
		Name:        toolName,
		Description: fmt.Sprintf("Fetch web content using %s MCP server", serverName),
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "URL to fetch",
				},
			},
			"required": []string{"url"},
		},
		Handler:   b.createMCPToolHandler(connection, toolName),
		IsMCPTool: true,
		MCPServer: serverName,
	}

	b.availableTools[toolName] = connection.Tools[toolName]
}

// createMCPToolHandler creates a handler function for MCP tools
func (b *ToolBridge) createMCPToolHandler(connection *MCPServerConnection, toolName string) func(context.Context, map[string]interface{}) (interface{}, error) {
	return func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		// In a full implementation, this would send a JSON-RPC request to the MCP server
		log.Printf("Executing MCP tool %s on server %s with params: %v", toolName, connection.Config.Name, params)

		// For now, return a simulated response
		return map[string]interface{}{
			"mcp_server": connection.Config.Name,
			"tool":       toolName,
			"result":     "MCP tool execution simulated",
			"params":     params,
			"status":     "success",
		}, nil
	}
}

// StopMCPServers stops all running MCP servers
func (b *ToolBridge) StopMCPServers() {
	for name, connection := range b.mcpServers {
		if connection.IsRunning && connection.Process != nil {
			log.Printf("Stopping MCP server: %s", name)
			if err := connection.Process.Process.Kill(); err != nil {
				log.Printf("Failed to kill MCP server %s: %v", name, err)
			}
			connection.IsRunning = false
		}
	}
}

// assessOperationRisk assesses the risk level of a tool operation
func (b *ToolBridge) assessOperationRisk(toolCall ToolCall) string {
	// Check for destructive operations
	destructiveCommands := []string{"rm", "delete", "remove", "format", "dd", "shutdown", "reboot"}

	// Check command parameters for risky patterns
	if command, ok := toolCall.Parameters["command"].(string); ok {
		commandLower := strings.ToLower(command)

		// Safe paths - these should always be considered low risk
		safePaths := []string{"/users/r00t/projects", "/tmp", "/home", "/users/r00t"}
		for _, safePath := range safePaths {
			if strings.Contains(commandLower, safePath) {
				// Even with safe paths, still check for truly destructive commands
				if strings.Contains(commandLower, "rm -rf") ||
					strings.Contains(commandLower, "format") ||
					strings.Contains(commandLower, "dd if=") {
					return "high"
				}
				return "low" // Safe path operations are low risk
			}
		}

		// Critical risk operations
		if strings.Contains(commandLower, "rm -rf") ||
			strings.Contains(commandLower, "format") ||
			strings.Contains(commandLower, "dd if=") ||
			strings.Contains(commandLower, "shutdown") ||
			strings.Contains(commandLower, "reboot") {
			return "critical"
		}

		// High risk operations
		for _, destructive := range destructiveCommands {
			if strings.Contains(commandLower, destructive) {
				return "high"
			}
		}

		// Medium risk for system directories
		systemPaths := []string{"/etc", "/usr", "/var", "/sys", "/boot"}
		for _, sysPath := range systemPaths {
			if strings.Contains(commandLower, sysPath) {
				return "medium"
			}
		}
	}

	// Check file paths for risky locations
	if path, ok := toolCall.Parameters["path"].(string); ok {
		pathLower := strings.ToLower(path)
		systemPaths := []string{"/etc", "/usr", "/var", "/sys", "/boot", "/root"}

		for _, sysPath := range systemPaths {
			if strings.HasPrefix(pathLower, sysPath) {
				return "medium"
			}
		}
	}

	return "low"
}

// tryAlternativeApproaches tries alternative approaches when a tool fails
func (b *ToolBridge) tryAlternativeApproaches(ctx context.Context, originalCall ToolCall, originalError error) *ToolResult {
	log.Printf("Original operation failed: %v. Trying alternatives...", originalError)

	// For execute-command tool, try different approaches
	if originalCall.Name == "execute-command" {
		if command, ok := originalCall.Parameters["command"].(string); ok {
			alternatives := b.generateCommandAlternatives(command)

			for i, altCommand := range alternatives {
				log.Printf("Trying alternative %d: %s", i+1, altCommand)

				// Create alternative tool call
				altCall := ToolCall{
					Name: "execute-command",
					Parameters: map[string]interface{}{
						"command": altCommand,
					},
				}

				// Copy other parameters
				for key, value := range originalCall.Parameters {
					if key != "command" {
						altCall.Parameters[key] = value
					}
				}

				// Try the alternative
				if tool, exists := b.availableTools["execute-command"]; exists {
					result, err := tool.Handler(ctx, altCall.Parameters)
					if err == nil {
						return &ToolResult{
							Success: true,
							Result:  result,
						}
					}
					log.Printf("Alternative %d failed: %v", i+1, err)
				}
			}
		}
	}

	// For file operations, try different approaches
	if originalCall.Name == "file-info" || originalCall.Name == "read-file" {
		if path, ok := originalCall.Parameters["path"].(string); ok {
			// Try with different permissions or alternative paths
			alternatives := []string{
				strings.TrimSpace(path),
				strings.ReplaceAll(path, "//", "/"),
			}

			for _, altPath := range alternatives {
				if altPath != path {
					altCall := ToolCall{
						Name:       originalCall.Name,
						Parameters: make(map[string]interface{}),
					}

					// Copy parameters and modify path
					for key, value := range originalCall.Parameters {
						altCall.Parameters[key] = value
					}
					altCall.Parameters["path"] = altPath

					if tool, exists := b.availableTools[originalCall.Name]; exists {
						result, err := tool.Handler(ctx, altCall.Parameters)
						if err == nil {
							return &ToolResult{
								Success: true,
								Result:  result,
							}
						}
					}
				}
			}
		}
	}

	// No successful alternative found
	return nil
}

// generateCommandAlternatives generates alternative commands to try
func (b *ToolBridge) generateCommandAlternatives(originalCommand string) []string {
	var alternatives []string

	cmdLower := strings.ToLower(originalCommand)

	// mkdir alternatives
	if strings.HasPrefix(cmdLower, "mkdir") && !strings.Contains(cmdLower, "-p") {
		alternatives = append(alternatives, strings.Replace(originalCommand, "mkdir", "mkdir -p", 1))
	}

	// ls alternatives
	if strings.HasPrefix(cmdLower, "ls") && !strings.Contains(cmdLower, "-la") {
		alternatives = append(alternatives, originalCommand+" -la")
	}

	// cat alternatives
	if strings.HasPrefix(cmdLower, "cat") {
		alternatives = append(alternatives, strings.Replace(originalCommand, "cat", "head -50", 1))
		alternatives = append(alternatives, strings.Replace(originalCommand, "cat", "less", 1))
	}

	// cp alternatives
	if strings.HasPrefix(cmdLower, "cp") && !strings.Contains(cmdLower, "-r") {
		alternatives = append(alternatives, strings.Replace(originalCommand, "cp", "cp -r", 1))
	}

	return alternatives
}
