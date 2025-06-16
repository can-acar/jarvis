package handlers

// Package handlers contains the MCP tool handlers for configuration management.

import (
	"context"
	"fmt"
	"jarvis/internal/common"

	"github.com/mark3labs/mcp-go/mcp"
)

func HandleGetConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	configJSON, err := common.GetJSON()
	if err != nil {
		return mcp.NewToolResultError(common.FormatError(err, "get configuration")), nil
	}
	return mcp.NewToolResultText(configJSON), nil
}

func HandleSetConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := req.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid key parameter: %v", err)), nil
	}

	value, err := req.RequireString("value")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid value parameter: %v", err)), nil
	}

	if err := common.Set(key, value); err != nil {
		return mcp.NewToolResultError(common.FormatError(err, "set configuration")), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Configuration key '%s' set to '%s'", key, value)), nil
}

func HandleAddAllowedDirectory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	directory, err := req.RequireString("directory")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid directory parameter: %v", err)), nil
	}

	if err := common.AddAllowedDirectory(directory); err != nil {
		return mcp.NewToolResultError(common.FormatError(err, "add allowed directory")), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Directory '%s' added to allowed list", directory)), nil
}

func HandleRemoveAllowedDirectory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	directory, err := req.RequireString("directory")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid directory parameter: %v", err)), nil
	}

	err = common.RemoveAllowedDirectory(directory)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to remove allowed directory: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Directory '%s' removed from allowed list", directory)), nil
}

func HandleAddBlockedCommand(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pattern, err := req.RequireString("pattern")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid pattern parameter: %v", err)), nil
	}

	err = common.AddBlockedCommand(pattern)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add blocked command: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Command pattern '%s' added to blocked list", pattern)), nil
}

func HandleValidateConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := common.Validate(); err != nil {
		return mcp.NewToolResultError(common.FormatError(err, "validate configuration")), nil
	}
	return mcp.NewToolResultText("Configuration is valid"), nil
}

func HandleResetConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	common.Reset()
	return mcp.NewToolResultText("Configuration reset to default values"), nil
}
