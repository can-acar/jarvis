package terminal

import (
	"jarvis/handlers"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTerminalTools registers all terminal-related MCP tools
func RegisterTerminalTools(s *server.MCPServer) {
	// execute_command tool
	executeCmd := mcp.NewTool("execute_command",
		mcp.WithDescription("Execute a terminal command with configurable timeout and shell selection"),
		mcp.WithString("command", mcp.Required(), mcp.Description("Command to execute")),
		mcp.WithString("shell", mcp.Description("Shell to use (default: from config)")),
		mcp.WithNumber("timeout_seconds", mcp.Description("Timeout in seconds (default: 30)")),
		mcp.WithString("working_dir", mcp.Description("Working directory for command execution")),
		mcp.WithBoolean("capture_stderr", mcp.Description("Capture stderr separately (default: false)")),
	)
	s.AddTool(executeCmd, handlers.HandleExecuteCommand)

	// list_processes tool
	listProcesses := mcp.NewTool("list_processes",
		mcp.WithDescription("List all running processes with detailed information"),
		mcp.WithString("filter", mcp.Description("Filter processes by name pattern")),
		mcp.WithBoolean("include_threads", mcp.Description("Include thread information (default: false)")),
	)
	s.AddTool(listProcesses, handlers.HandleListProcesses)

	// kill_process tool
	killProcess := mcp.NewTool("kill_process",
		mcp.WithDescription("Terminate a running process by PID"),
		mcp.WithNumber("pid", mcp.Required(), mcp.Description("Process ID to terminate")),
		mcp.WithBoolean("force", mcp.Description("Force kill with SIGKILL (default: false)")),
	)
	s.AddTool(killProcess, handlers.HandleKillProcess)

	// get_process_info tool
	getProcessInfo := mcp.NewTool("get_process_info",
		mcp.WithDescription("Get detailed information about a specific process"),
		mcp.WithNumber("pid", mcp.Required(), mcp.Description("Process ID to query")),
	)
	s.AddTool(getProcessInfo, handlers.HandleGetProcessInfo)

	// run_shell_script tool
	runScript := mcp.NewTool("run_shell_script",
		mcp.WithDescription("Execute a multi-line shell script"),
		mcp.WithString("script", mcp.Required(), mcp.Description("Shell script content")),
		mcp.WithString("shell", mcp.Description("Shell interpreter (default: from config)")),
		mcp.WithNumber("timeout_seconds", mcp.Description("Timeout in seconds (default: 60)")),
		mcp.WithBoolean("create_temp_file", mcp.Description("Create temporary script file (default: true)")),
	)
	s.AddTool(runScript, handlers.HandleRunShellScript)

	// check_command_exists tool
	checkCommand := mcp.NewTool("check_command_exists",
		mcp.WithDescription("Check if a command or program exists in the system PATH"),
		mcp.WithString("command", mcp.Required(), mcp.Description("Command name to check")),
	)
	s.AddTool(checkCommand, handlers.HandleCheckCommandExists)

	// get_system_info tool
	getSystemInfo := mcp.NewTool("get_system_info",
		mcp.WithDescription("Get system information including OS, CPU, memory, and disk usage"),
	)
	s.AddTool(getSystemInfo, handlers.HandleGetSystemInfo)
}
