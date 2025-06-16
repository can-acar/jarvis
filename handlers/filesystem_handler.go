package handlers

import (
	"context"
	"fmt"
	"jarvis/internal/common"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func HandleReadFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid path parameter: %v", err)), nil
	}

	if !common.IsPathAllowed(path) {
		return mcp.NewToolResultError("Access to this path is not allowed"), nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to read file: %v", err)), nil
	}

	lines := common.SplitLines(string(content))

	// Handle pagination
	offset := int(mcp.ParseFloat64(req, "offset", 1)) - 1 // Convert to 0-based
	length := int(mcp.ParseFloat64(req, "length", 0))
	showLineNumbers := mcp.ParseBoolean(req, "show_line_numbers", false)

	if offset < 0 {
		offset = 0
	}

	var resultLines []string

	if length > 0 {
		end := offset + length
		if end > len(lines) {
			end = len(lines)
		}
		if offset < len(lines) {
			resultLines = lines[offset:end]
		}
	} else {
		if offset < len(lines) {
			resultLines = lines[offset:]
		}
	}

	// Apply line read limit from config
	cfg := common.Get()
	if len(resultLines) > cfg.FileReadLineLimit {
		resultLines = resultLines[:cfg.FileReadLineLimit]
		resultLines = append(resultLines, "... (truncated due to line limit)")
	}

	// Add line numbers if requested
	if showLineNumbers && len(resultLines) > 0 {
		for i, line := range resultLines {
			if line != "... (truncated due to line limit)" {
				resultLines[i] = fmt.Sprintf("%d: %s", offset+i+1, line)
			}
		}
	}

	result := common.JoinLines(resultLines)
	return mcp.NewToolResultText(result), nil
}

func HandleWriteFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid path parameter: %v", err)), nil
	}

	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid content parameter: %v", err)), nil
	}

	if !common.IsPathAllowed(path) {
		return mcp.NewToolResultError("Access to this path is not allowed"), nil
	}

	append := mcp.ParseBoolean(req, "append", false)
	createBackup := mcp.ParseBoolean(req, "create_backup", false)

	// Create backup if requested and file exists
	if createBackup {
		if _, err := os.Stat(path); err == nil {
			backupPath, err := common.CreateBackup(path)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to create backup: %v", err)), nil
			}
			defer func() {
				// Log backup creation
				fmt.Printf("Backup created: %s\n", backupPath)
			}()
		}
	}

	// Ensure parent directory exists
	if err := common.EnsureDir(filepath.Dir(path)); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create parent directory: %v", err)), nil
	}

	flag := os.O_CREATE | os.O_WRONLY
	if append {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	file, err := os.OpenFile(path, flag, 0644)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to open file: %v", err)), nil
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to write file: %v", err)), nil
	}

	operation := "written"
	if append {
		operation = "appended"
	}

	return mcp.NewToolResultText(fmt.Sprintf("Content successfully %s to %s", operation, path)), nil
}

func HandleCreateDirectory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid path parameter: %v", err)), nil
	}

	if !common.IsPathAllowed(path) {
		return mcp.NewToolResultError("Access to this path is not allowed"), nil
	}

	createParents := mcp.ParseBoolean(req, "create_parents", true)
	permissions := mcp.ParseString(req, "permissions", "0755")

	// Parse permissions
	var perm os.FileMode = 0755
	if permissions != "0755" {
		fmt.Sscanf(permissions, "%o", &perm)
	}

	var createErr error
	if createParents {
		createErr = os.MkdirAll(path, perm)
	} else {
		createErr = os.Mkdir(path, perm)
	}

	if createErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create directory: %v", createErr)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Directory created: %s", path)), nil
}

func HandleListDirectory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid path parameter: %v", err)), nil
	}

	if !common.IsPathAllowed(path) {
		return mcp.NewToolResultError("Access to this path is not allowed"), nil
	}

	includeHidden := mcp.ParseBoolean(req, "include_hidden", false)
	recursive := mcp.ParseBoolean(req, "recursive", false)

	var result strings.Builder

	if recursive {
		err = filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip hidden files if not requested
			if !includeHidden && strings.HasPrefix(info.Name(), ".") && walkPath != path {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			relPath, _ := filepath.Rel(path, walkPath)
			if relPath == "." {
				relPath = filepath.Base(path)
			}

			result.WriteString(common.FormatFileInfo(relPath, info))
			return nil
		})
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to read directory: %v", err)), nil
		}

		for _, entry := range entries {
			// Skip hidden files if not requested
			if !includeHidden && strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				continue
			}

			result.WriteString(common.FormatFileInfo(entry.Name(), info))
		}
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list directory: %v", err)), nil
	}

	return mcp.NewToolResultText(result.String()), nil
}

func HandleSearchFiles(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pattern, err := req.RequireString("pattern")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid pattern parameter: %v", err)), nil
	}

	directory := mcp.ParseString(req, "directory", ".")
	if !common.IsPathAllowed(directory) {
		return mcp.NewToolResultError("Access to this directory is not allowed"), nil
	}

	caseSensitive := mcp.ParseBoolean(req, "case_sensitive", false)
	includeDirectories := mcp.ParseBoolean(req, "include_directories", false)
	maxDepth := int(mcp.ParseFloat64(req, "max_depth", -1))

	var matches []string

	err = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip problematic files
		}

		// Check depth limit
		if maxDepth >= 0 {
			relPath, _ := filepath.Rel(directory, path)
			depth := strings.Count(relPath, string(filepath.Separator))
			if depth > maxDepth {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Skip directories unless requested
		if info.IsDir() && !includeDirectories {
			return nil
		}

		// Match pattern
		name := info.Name()
		if !caseSensitive {
			name = strings.ToLower(name)
			pattern = strings.ToLower(pattern)
		}

		if matched, _ := filepath.Match(pattern, name); matched || strings.Contains(name, pattern) {
			matches = append(matches, path)
		}

		return nil
	})

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Search failed: %v", err)), nil
	}

	return mcp.NewToolResultText(strings.Join(matches, "\n")), nil
}

func HandleGetFileInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid path parameter: %v", err)), nil
	}

	if !common.IsPathAllowed(path) {
		return mcp.NewToolResultError("Access to this path is not allowed"), nil
	}

	includeChecksum := mcp.ParseBoolean(req, "include_checksum", false)

	info, err := os.Stat(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get file info: %v", err)), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Name: %s\n", info.Name()))
	result.WriteString(fmt.Sprintf("Size: %s (%d bytes)\n", common.FormatBytes(info.Size()), info.Size()))
	result.WriteString(fmt.Sprintf("Mode: %s\n", info.Mode().String()))
	result.WriteString(fmt.Sprintf("Modified: %s\n", info.ModTime().Format(time.RFC3339)))
	result.WriteString(fmt.Sprintf("Is Directory: %t\n", info.IsDir()))

	if !info.IsDir() {
		result.WriteString(fmt.Sprintf("Is Text File: %t\n", common.IsTextFile(path)))

		if includeChecksum {
			checksum, err := common.CalculateFileChecksum(path)
			if err == nil {
				result.WriteString(fmt.Sprintf("SHA256: %s\n", checksum))
			}
		}
	}

	return mcp.NewToolResultText(result.String()), nil
}

func HandleCopyFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	source, err := req.RequireString("source")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid source parameter: %v", err)), nil
	}

	destination, err := req.RequireString("destination")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid destination parameter: %v", err)), nil
	}

	if !common.IsPathAllowed(source) || !common.IsPathAllowed(destination) {
		return mcp.NewToolResultError("Access to one or both paths is not allowed"), nil
	}

	overwrite := mcp.ParseBoolean(req, "overwrite", false)

	// Check if destination exists
	if _, err := os.Stat(destination); err == nil && !overwrite {
		return mcp.NewToolResultError("Destination exists and overwrite is false"), nil
	}

	// Ensure destination directory exists
	if err := common.EnsureDir(filepath.Dir(destination)); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create destination directory: %v", err)), nil
	}

	// Copy file
	err = common.CopyFile(source, destination)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to copy file: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("File copied from %s to %s", source, destination)), nil
}

func HandleMoveFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	source, err := req.RequireString("source")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid source parameter: %v", err)), nil
	}

	destination, err := req.RequireString("destination")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid destination parameter: %v", err)), nil
	}

	if !common.IsPathAllowed(source) || !common.IsPathAllowed(destination) {
		return mcp.NewToolResultError("Access to one or both paths is not allowed"), nil
	}

	overwrite := mcp.ParseBoolean(req, "overwrite", false)

	// Check if destination exists
	if _, err := os.Stat(destination); err == nil && !overwrite {
		return mcp.NewToolResultError("Destination exists and overwrite is false"), nil
	}

	// Ensure destination directory exists
	if err := common.EnsureDir(filepath.Dir(destination)); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create destination directory: %v", err)), nil
	}

	// Move file
	err = os.Rename(source, destination)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to move file: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("File moved from %s to %s", source, destination)), nil
}

func HandleDeleteFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid path parameter: %v", err)), nil
	}

	if !common.IsPathAllowed(path) {
		return mcp.NewToolResultError("Access to this path is not allowed"), nil
	}

	recursive := mcp.ParseBoolean(req, "recursive", false)
	createBackup := mcp.ParseBoolean(req, "create_backup", false)

	// Create backup if requested
	if createBackup {
		if _, err := os.Stat(path); err == nil {
			backupPath, err := common.CreateBackup(path)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to create backup: %v", err)), nil
			}
			defer func() {
				fmt.Printf("Backup created: %s\n", backupPath)
			}()
		}
	}

	// Delete file or directory
	var deleteErr error
	if recursive {
		deleteErr = os.RemoveAll(path)
	} else {
		deleteErr = os.Remove(path)
	}

	if deleteErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete: %v", deleteErr)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Deleted: %s", path)), nil
}

func HandleFindInFiles(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pattern, err := req.RequireString("pattern")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid pattern parameter: %v", err)), nil
	}

	directory := mcp.ParseString(req, "directory", ".")
	if !common.IsPathAllowed(directory) {
		return mcp.NewToolResultError("Access to this directory is not allowed"), nil
	}

	filePattern := mcp.ParseString(req, "file_pattern", "*")
	caseSensitive := mcp.ParseBoolean(req, "case_sensitive", false)
	contextLines := int(mcp.ParseFloat64(req, "context_lines", 0))

	var results []string

	err = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Check file pattern
		if matched, _ := filepath.Match(filePattern, info.Name()); !matched {
			return nil
		}

		// Only search in text files
		if !common.IsTextFile(path) {
			return nil
		}

		// Search in file
		matches, err := common.SearchInFile(path, pattern, caseSensitive, contextLines)
		if err != nil {
			return nil // Skip files that can't be read
		}

		if len(matches) > 0 {
			results = append(results, fmt.Sprintf("=== %s ===", path))
			results = append(results, matches...)
			results = append(results, "")
		}

		return nil
	})

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Search failed: %v", err)), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText("No matches found"), nil
	}

	return mcp.NewToolResultText(strings.Join(results, "\n")), nil
}

// Helper functions
