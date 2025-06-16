package textedit

import (
	"jarvis/handlers"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTextEditingTools registers all text editing MCP tools
func RegisterTextEditingTools(s *server.MCPServer) {
	// edit_block - Enhanced with character-level diff feedback
	editBlock := mcp.NewTool("edit_block",
		mcp.WithDescription("Apply targeted text replacements with enhanced prompting for smaller edits (includes character-level diff feedback)"),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path to edit")),
		mcp.WithNumber("start_line", mcp.Required(), mcp.Description("Starting line number (1-based)")),
		mcp.WithNumber("end_line", mcp.Required(), mcp.Description("Ending line number (1-based)")),
		mcp.WithString("replacement", mcp.Required(), mcp.Description("Replacement text")),
		mcp.WithBoolean("show_diff", mcp.Description("Show character-level diff feedback (default: true)")),
		mcp.WithBoolean("create_backup", mcp.Description("Create backup before editing (default: true)")),
		mcp.WithBoolean("validate_syntax", mcp.Description("Validate syntax for known file types (default: false)")),
	)
	s.AddTool(editBlock, handlers.HandleEditBlock)

	// edit_file - Line-based replacements with multiple edits
	editFile := mcp.NewTool("edit_file",
		mcp.WithDescription("Edit files with line-based replacements, supports multiple edits in one go"),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path to edit")),
		mcp.WithString("operations", mcp.Required(), mcp.Description("JSON array of edit operations: [{\"start_line\": 1, \"end_line\": 3, \"replacement\": \"new text\", \"description\": \"optional\"}]")),
		mcp.WithBoolean("create_backup", mcp.Description("Create backup before editing (default: true)")),
		mcp.WithBoolean("validate_operations", mcp.Description("Validate operations before applying (default: true)")),
		mcp.WithBoolean("show_preview", mcp.Description("Show preview of changes (default: false)")),
		mcp.WithBoolean("atomic", mcp.Description("Apply all operations atomically (default: true)")),
	)
	s.AddTool(editFile, handlers.HandleEditFile)

	// edit_multiple_files - Edit multiple files simultaneously
	editMultipleFiles := mcp.NewTool("edit_multiple_files",
		mcp.WithDescription("Edit multiple files simultaneously with line-based replacements"),
		mcp.WithString("files", mcp.Required(), mcp.Description("JSON array of file edit requests: [{\"path\": \"file.txt\", \"operations\": [...], \"create_backup\": true}]")),
		mcp.WithBoolean("atomic", mcp.Description("All operations succeed or all fail (default: true)")),
		mcp.WithBoolean("dry_run", mcp.Description("Preview changes without applying them (default: false)")),
		mcp.WithBoolean("continue_on_error", mcp.Description("Continue processing files even if one fails (ignored if atomic=true)")),
		mcp.WithBoolean("validate_all", mcp.Description("Validate all operations before starting (default: true)")),
	)
	s.AddTool(editMultipleFiles, handlers.HandleEditMultipleFiles)

	// replace_text - Simple find and replace
	replaceText := mcp.NewTool("replace_text",
		mcp.WithDescription("Find and replace text in a file with optional regex support"),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path to edit")),
		mcp.WithString("find", mcp.Required(), mcp.Description("Text to find")),
		mcp.WithString("replace", mcp.Required(), mcp.Description("Replacement text")),
		mcp.WithBoolean("regex", mcp.Description("Use regular expressions (default: false)")),
		mcp.WithBoolean("case_sensitive", mcp.Description("Case sensitive search (default: true)")),
		mcp.WithBoolean("whole_word", mcp.Description("Match whole words only (default: false)")),
		mcp.WithNumber("max_replacements", mcp.Description("Maximum number of replacements (default: unlimited)")),
		mcp.WithBoolean("create_backup", mcp.Description("Create backup before editing (default: true)")),
	)
	s.AddTool(replaceText, handlers.HandleReplaceText)

	// insert_text - Insert text at specific positions
	insertText := mcp.NewTool("insert_text",
		mcp.WithDescription("Insert text at specific line positions in a file"),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path to edit")),
		mcp.WithString("insertions", mcp.Required(), mcp.Description("JSON array of insertions: [{\"line\": 5, \"text\": \"new line\", \"position\": \"before|after\"}]")),
		mcp.WithBoolean("create_backup", mcp.Description("Create backup before editing (default: true)")),
		mcp.WithBoolean("adjust_line_numbers", mcp.Description("Automatically adjust subsequent line numbers (default: true)")),
	)
	s.AddTool(insertText, handlers.HandleInsertText)

	// format_code - Format code files
	formatCode := mcp.NewTool("format_code",
		mcp.WithDescription("Format code files using appropriate formatters"),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path to format")),
		mcp.WithString("formatter", mcp.Description("Specific formatter to use (auto-detected if not specified)")),
		mcp.WithBoolean("create_backup", mcp.Description("Create backup before formatting (default: true)")),
		mcp.WithString("config_file", mcp.Description("Path to formatter configuration file")),
	)
	s.AddTool(formatCode, handlers.HandleFormatCode)
}
