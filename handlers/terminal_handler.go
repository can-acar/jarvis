package handlers

import (
	"context"
	"fmt"
	"jarvis/internal/common"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func HandleExecuteCommand(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	command, err := req.RequireString("command")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid command parameter: %v", err)), nil
	}

	// Sanitize the command
	command = common.SanitizeCommand(command)

	// Security check
	if common.IsCommandBlocked(command) {
		return mcp.NewToolResultError("Command contains blocked patterns"), nil
	}

	cfg := common.Get()
	shell := mcp.ParseString(req, "shell", cfg.DefaultShell)
	timeout := time.Duration(mcp.ParseFloat64(req, "timeout_seconds", 30)) * time.Second
	workingDir := mcp.ParseString(req, "working_dir", "")
	captureStderr := mcp.ParseBoolean(req, "capture_stderr", false)

	// Create context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Prepare command
	cmd := exec.CommandContext(cmdCtx, shell, "-c", command)

	if workingDir != "" && common.IsPathAllowed(workingDir) {
		cmd.Dir = workingDir
	}

	// Execute command
	var output []byte
	if captureStderr {
		stdout, err1 := cmd.Output()
		stderr := ""
		if err1 != nil {
			if exitErr, ok := err1.(*exec.ExitError); ok {
				stderr = string(exitErr.Stderr)
			}
		}

		result := fmt.Sprintf("STDOUT:\n%s\n\nSTDERR:\n%s", string(stdout), stderr)
		if err1 != nil {
			result += fmt.Sprintf("\n\nEXIT CODE: %v", err1)
		}
		return mcp.NewToolResultText(result), nil
	} else {
		output, err = cmd.CombinedOutput()
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Command failed: %v\nOutput: %s", err, string(output))), nil
	}

	return mcp.NewToolResultText(string(output)), nil
}

func HandleListProcesses(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filter := mcp.ParseString(req, "filter", "")
	includeThreads := mcp.ParseBoolean(req, "include_threads", false)

	var cmd *exec.Cmd
	if includeThreads {
		cmd = exec.Command("ps", "auxH")
	} else {
		cmd = exec.Command("ps", "aux")
	}

	output, err := cmd.Output()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list processes: %v", err)), nil
	}

	result := string(output)

	// Apply filter if specified
	if filter != "" {
		lines := strings.Split(result, "\n")
		var filteredLines []string

		// Keep header
		if len(lines) > 0 {
			filteredLines = append(filteredLines, lines[0])
		}

		// Filter processes
		for i := 1; i < len(lines); i++ {
			if strings.Contains(strings.ToLower(lines[i]), strings.ToLower(filter)) {
				filteredLines = append(filteredLines, lines[i])
			}
		}

		result = strings.Join(filteredLines, "\n")
	}

	return mcp.NewToolResultText(result), nil
}

func HandleKillProcess(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pid := int(mcp.ParseFloat64(req, "pid", 0))
	if pid <= 0 {
		return mcp.NewToolResultError("Invalid PID"), nil
	}

	force := mcp.ParseBoolean(req, "force", false)

	var cmd *exec.Cmd
	if force {
		cmd = exec.Command("kill", "-9", strconv.Itoa(pid))
	} else {
		cmd = exec.Command("kill", strconv.Itoa(pid))
	}

	err := cmd.Run()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to kill process %d: %v", pid, err)), nil
	}

	killType := "terminated"
	if force {
		killType = "force killed"
	}

	return mcp.NewToolResultText(fmt.Sprintf("Process %d %s", pid, killType)), nil
}

func HandleGetProcessInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pid := int(mcp.ParseFloat64(req, "pid", 0))
	if pid <= 0 {
		return mcp.NewToolResultError("Invalid PID"), nil
	}

	// Get detailed process information
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "pid,ppid,user,cpu,mem,vsz,rss,tty,stat,start,time,command")
	output, err := cmd.Output()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get process info for PID %d: %v", pid, err)), nil
	}

	result := string(output)

	// Try to get additional information from /proc if available
	cmd = exec.Command("cat", fmt.Sprintf("/proc/%d/status", pid))
	if statusOutput, err := cmd.Output(); err == nil {
		result += "\n\nProcess Status:\n" + string(statusOutput)
	}

	return mcp.NewToolResultText(result), nil
}

func HandleRunShellScript(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	script, err := req.RequireString("script")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid script parameter: %v", err)), nil
	}

	// Basic security check on script content
	if common.IsCommandBlocked(script) {
		return mcp.NewToolResultError("Script contains blocked command patterns"), nil
	}

	cfg := common.Get()
	shell := mcp.ParseString(req, "shell", cfg.DefaultShell)
	timeout := time.Duration(mcp.ParseFloat64(req, "timeout_seconds", 60)) * time.Second
	createTempFile := mcp.ParseBoolean(req, "create_temp_file", true)

	// Create context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd

	if createTempFile {
		// Create temporary script file
		tempFile, err := common.CreateTempScript(script, shell)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create temp script: %v", err)), nil
		}
		defer common.CleanupTempFile(tempFile)

		cmd = exec.CommandContext(cmdCtx, shell, tempFile)
	} else {
		// Execute directly
		cmd = exec.CommandContext(cmdCtx, shell, "-c", script)
	}

	// Execute script
	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Script execution failed: %v\nOutput: %s", err, string(output))), nil
	}

	return mcp.NewToolResultText(string(output)), nil
}

func HandleCheckCommandExists(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	command, err := req.RequireString("command")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid command parameter: %v", err)), nil
	}

	// Use 'which' command to check if command exists
	cmd := exec.Command("which", command)
	output, err := cmd.Output()

	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Command '%s' not found in PATH", command)), nil
	}

	path := strings.TrimSpace(string(output))
	return mcp.NewToolResultText(fmt.Sprintf("Command '%s' found at: %s", command, path)), nil
}

func HandleGetSystemInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var result strings.Builder

	// Get OS information
	if output, err := exec.Command("uname", "-a").Output(); err == nil {
		result.WriteString("System: " + strings.TrimSpace(string(output)) + "\n")
	}

	// Get uptime
	if output, err := exec.Command("uptime").Output(); err == nil {
		result.WriteString("Uptime: " + strings.TrimSpace(string(output)) + "\n")
	}

	// Get memory information
	if output, err := exec.Command("free", "-h").Output(); err == nil {
		result.WriteString("\nMemory:\n" + string(output))
	}

	// Get disk usage
	if output, err := exec.Command("df", "-h").Output(); err == nil {
		result.WriteString("\nDisk Usage:\n" + string(output))
	}

	// Get CPU information
	if output, err := exec.Command("nproc").Output(); err == nil {
		result.WriteString("\nCPU Cores: " + strings.TrimSpace(string(output)) + "\n")
	}

	// Get load average
	if output, err := exec.Command("cat", "/proc/loadavg").Output(); err == nil {
		result.WriteString("Load Average: " + strings.TrimSpace(string(output)) + "\n")
	}

	return mcp.NewToolResultText(result.String()), nil
}
