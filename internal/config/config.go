package config

import (
	"jarvis/handlers"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterConfigTools registers all configuration-related MCP tools
func RegisterConfigTools(s *server.MCPServer) {
	// get_config tool
	getConfigTool := mcp.NewTool("get_config",
		mcp.WithDescription("Get the complete server configuration as JSON"),
	)
	s.AddTool(getConfigTool, handlers.HandleGetConfig)

	// set_config_value tool
	setConfigTool := mcp.NewTool("set_config_value",
		mcp.WithDescription("Set a specific configuration value by key"),
		mcp.WithString("key", mcp.Required(), mcp.Description("Configuration key to set")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Configuration value")),
	)
	s.AddTool(setConfigTool, handlers.HandleSetConfig)

	// add_allowed_directory tool
	addDirTool := mcp.NewTool("add_allowed_directory",
		mcp.WithDescription("Add a directory to the allowed directories list"),
		mcp.WithString("directory", mcp.Required(), mcp.Description("Directory path to allow")),
	)
	s.AddTool(addDirTool, handlers.HandleAddAllowedDirectory)

	// remove_allowed_directory tool
	removeDirTool := mcp.NewTool("remove_allowed_directory",
		mcp.WithDescription("Remove a directory from the allowed directories list"),
		mcp.WithString("directory", mcp.Required(), mcp.Description("Directory path to remove")),
	)
	s.AddTool(removeDirTool, handlers.HandleRemoveAllowedDirectory)

	// add_blocked_command tool
	addBlockedTool := mcp.NewTool("add_blocked_command",
		mcp.WithDescription("Add a command pattern to the blocked commands list"),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("Command pattern to block")),
	)
	s.AddTool(addBlockedTool, handlers.HandleAddBlockedCommand)

	// validate_config tool
	validateTool := mcp.NewTool("validate_config",
		mcp.WithDescription("Validate the current server configuration"),
	)
	s.AddTool(validateTool, handlers.HandleValidateConfig)

	// reset_config tool
	resetTool := mcp.NewTool("reset_config",
		mcp.WithDescription("Reset configuration to default values"),
	)
	s.AddTool(resetTool, handlers.HandleResetConfig)
}
