package filesystem

import (
	"jarvis/handlers"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func RegisterFilesystemTools(s *server.MCPServer) {
	// read_file tool
	readFile := mcp.NewTool("read_file",
		mcp.WithDescription("Read contents from local filesystem with line-based pagination"),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path to read")),
		mcp.WithNumber("offset", mcp.Description("Line offset to start reading from (1-based)")),
		mcp.WithNumber("length", mcp.Description("Number of lines to read")),
		mcp.WithBoolean("show_line_numbers", mcp.Description("Show line numbers (default: false)")),
	)
	s.AddTool(readFile, handlers.HandleReadFile)

	// write_file tool
	writeFile := mcp.NewTool("write_file",
		mcp.WithDescription("Write file contents with options for rewrite or append mode"),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path to write")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Content to write")),
		mcp.WithBoolean("append", mcp.Description("Append to file instead of overwriting")),
		mcp.WithBoolean("create_backup", mcp.Description("Create backup before writing (default: false)")),
		mcp.WithString("encoding", mcp.Description("File encoding (default: utf-8)")),
	)
	s.AddTool(writeFile, handlers.HandleWriteFile)

	// create_directory tool
	createDir := mcp.NewTool("create_directory",
		mcp.WithDescription("Create a new directory or ensure it exists"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Directory path to create")),
		mcp.WithBoolean("create_parents", mcp.Description("Create parent directories if needed (default: true)")),
		mcp.WithString("permissions", mcp.Description("Directory permissions in octal (default: 0755)")),
	)
	s.AddTool(createDir, handlers.HandleCreateDirectory)

	// list_directory tool
	listDir := mcp.NewTool("list_directory",
		mcp.WithDescription("Get detailed listing of files and directories"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Directory path to list")),
		mcp.WithBoolean("include_hidden", mcp.Description("Include hidden files (default: false)")),
		mcp.WithBoolean("recursive", mcp.Description("List recursively (default: false)")),
		mcp.WithString("sort_by", mcp.Description("Sort by: name, size, modified (default: name)")),
	)
	s.AddTool(listDir, handlers.HandleListDirectory)

	// search_files tool
	searchFiles := mcp.NewTool("search_files",
		mcp.WithDescription("Find files by name using pattern matching"),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("Search pattern (supports wildcards)")),
		mcp.WithString("directory", mcp.Description("Directory to search in (default: current)")),
		mcp.WithBoolean("case_sensitive", mcp.Description("Case sensitive search (default: false)")),
		mcp.WithBoolean("include_directories", mcp.Description("Include directories in results (default: false)")),
		mcp.WithNumber("max_depth", mcp.Description("Maximum search depth (default: unlimited)")),
	)
	s.AddTool(searchFiles, handlers.HandleSearchFiles)

	// get_file_info tool
	getFileInfo := mcp.NewTool("get_file_info",
		mcp.WithDescription("Retrieve detailed metadata about a file or directory"),
		mcp.WithString("path", mcp.Required(), mcp.Description("File or directory path")),
		mcp.WithBoolean("include_checksum", mcp.Description("Calculate file checksum (default: false)")),
	)
	s.AddTool(getFileInfo, handlers.HandleGetFileInfo)

	copyFile := mcp.NewTool("copy_file",
		mcp.WithDescription("Copy a file or directory to another location"),
		mcp.WithString("source", mcp.Required(), mcp.Description("Source path")),
		mcp.WithString("destination", mcp.Required(), mcp.Description("Destination path")),
		mcp.WithBoolean("overwrite", mcp.Description("Overwrite destination if exists (default: false)")),
		mcp.WithBoolean("preserve_permissions", mcp.Description("Preserve file permissions (default: true)")),
	)
	s.AddTool(copyFile, handlers.HandleCopyFile)

	moveFile := mcp.NewTool("move_file",
		mcp.WithDescription("Move or rename files and directories"),
		mcp.WithString("source", mcp.Required(), mcp.Description("Source path")),
		mcp.WithString("destination", mcp.Required(), mcp.Description("Destination path")),
		mcp.WithBoolean("overwrite", mcp.Description("Overwrite destination if exists (default: false)")),
	)
	s.AddTool(moveFile, handlers.HandleMoveFile)

	// delete_file tool
	deleteFile := mcp.NewTool("delete_file",
		mcp.WithDescription("Delete a file or directory"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to delete")),
		mcp.WithBoolean("recursive", mcp.Description("Delete directories recursively (default: false)")),
		mcp.WithBoolean("create_backup", mcp.Description("Create backup before deletion (default: false)")),
	)
	s.AddTool(deleteFile, handlers.HandleDeleteFile)

	// find_in_files tool
	findInFiles := mcp.NewTool("find_in_files",
		mcp.WithDescription("Search for text patterns within file contents"),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("Text pattern to search for")),
		mcp.WithString("directory", mcp.Description("Directory to search in (default: current)")),
		mcp.WithString("file_pattern", mcp.Description("File name pattern to include (default: all files)")),
		mcp.WithBoolean("case_sensitive", mcp.Description("Case sensitive search (default: false)")),
		mcp.WithBoolean("regex", mcp.Description("Use regular expressions (default: false)")),
		mcp.WithNumber("context_lines", mcp.Description("Number of context lines around matches (default: 0)")),
	)
	s.AddTool(findInFiles, handlers.HandleFindInFiles)
}
