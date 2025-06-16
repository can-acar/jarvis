package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"jarvis/internal/common"
	"jarvis/internal/types"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func HandleEditBlock(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid path parameter: %v", err)), nil
	}

	if !common.IsPathAllowed(path) {
		return mcp.NewToolResultError("Access to this path is not allowed"), nil
	}

	startLine := int(mcp.ParseFloat64(req, "start_line", 1))
	endLine := int(mcp.ParseFloat64(req, "end_line", 1))
	replacement, err := req.RequireString("replacement")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid replacement parameter: %v", err)), nil
	}

	showDiff := mcp.ParseBoolean(req, "show_diff", true)
	createBackup := mcp.ParseBoolean(req, "create_backup", true)
	validateSyntax := mcp.ParseBoolean(req, "validate_syntax", false)

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to read file: %v", err)), nil
	}

	originalContent := string(content)
	lines := common.SplitLines(originalContent)

	// Validate line range
	if err := common.ValidateLineRange(startLine, endLine, len(lines)); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Create backup
	if createBackup {
		if _, err := common.CreateBackup(path); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create backup: %v", err)), nil
		}
	}

	// Get original section for diff
	startIdx := startLine - 1
	endIdx := endLine
	originalSection := common.JoinLines(lines[startIdx:endIdx])

	// Apply replacement
	newLines := make([]string, 0, len(lines)+(strings.Count(replacement, "\n")+1)-(endIdx-startIdx))
	newLines = append(newLines, lines[:startIdx]...)
	newLines = append(newLines, common.SplitLines(replacement)...)
	newLines = append(newLines, lines[endIdx:]...)

	newContent := common.JoinLines(newLines)

	// Validate syntax if requested
	if validateSyntax {
		if err := common.ValidateFileSyntax(path, newContent); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Syntax validation failed: %v", err)), nil
		}
	}

	// Write file
	err = os.WriteFile(path, []byte(newContent), 0644)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to write file: %v", err)), nil
	}

	result := fmt.Sprintf("Successfully edited lines %d-%d in %s", startLine, endLine, path)

	// Show diff if requested
	if showDiff {
		diff := common.GenerateCharacterDiff(originalSection, replacement)
		result += "\n\nDiff:\n" + diff
	}

	return mcp.NewToolResultText(result), nil
}

func HandleEditFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid path parameter: %v", err)), nil
	}

	if !common.IsPathAllowed(path) {
		return mcp.NewToolResultError("Access to this path is not allowed"), nil
	}

	operationsStr, err := req.RequireString("operations")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid operations parameter: %v", err)), nil
	}

	var operations []types.EditOperation
	if err := json.Unmarshal([]byte(operationsStr), &operations); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse operations: %v", err)), nil
	}

	createBackup := mcp.ParseBoolean(req, "create_backup", true)
	validateOperations := mcp.ParseBoolean(req, "validate_operations", true)
	showPreview := mcp.ParseBoolean(req, "show_preview", false)
	atomic := mcp.ParseBoolean(req, "atomic", true)

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to read file: %v", err)), nil
	}

	originalContent := string(content)
	lines := common.SplitLines(originalContent)

	// Validate operations
	if validateOperations {
		if err := common.ValidateEditOperations(lines, operations); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Operation validation failed: %v", err)), nil
		}
	}

	// Sort operations by start line (descending) to avoid line number shifts
	sortedOps := common.SortOperationsByLine(operations)

	// Preview mode
	if showPreview {
		preview := common.GenerateEditPreview(lines, sortedOps)
		return mcp.NewToolResultText(fmt.Sprintf("Preview of changes for %s:\n%s", path, preview)), nil
	}

	// Create backup
	if createBackup {
		if _, err := common.CreateBackup(path); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create backup: %v", err)), nil
		}
	}

	// Apply operations
	resultLines := make([]string, len(lines))
	copy(resultLines, lines)

	if atomic {
		// Apply all operations atomically
		for _, op := range sortedOps {
			startIdx := op.StartLine - 1
			endIdx := op.EndLine

			newLines := make([]string, 0, len(resultLines)+(strings.Count(op.Replacement, "\n")+1)-(endIdx-startIdx))
			newLines = append(newLines, resultLines[:startIdx]...)
			newLines = append(newLines, common.SplitLines(op.Replacement)...)
			newLines = append(newLines, resultLines[endIdx:]...)

			resultLines = newLines
		}

		// Write file once
		newContent := common.JoinLines(resultLines)
		if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to write file: %v", err)), nil
		}
	} else {
		// Apply operations one by one
		for i, op := range sortedOps {
			startIdx := op.StartLine - 1
			endIdx := op.EndLine

			newLines := make([]string, 0, len(resultLines)+(strings.Count(op.Replacement, "\n")+1)-(endIdx-startIdx))
			newLines = append(newLines, resultLines[:startIdx]...)
			newLines = append(newLines, common.SplitLines(op.Replacement)...)
			newLines = append(newLines, resultLines[endIdx:]...)

			resultLines = newLines

			// Write after each operation for non-atomic mode
			newContent := common.JoinLines(resultLines)
			if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to write file at operation %d: %v", i+1, err)), nil
			}
		}
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully applied %d operations to %s", len(operations), path)), nil
}

func HandleEditMultipleFiles(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filesStr, err := req.RequireString("files")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid files parameter: %v", err)), nil
	}

	var fileRequests []types.FileEditRequest
	if err := json.Unmarshal([]byte(filesStr), &fileRequests); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse files: %v", err)), nil
	}

	atomic := mcp.ParseBoolean(req, "atomic", true)
	dryRun := mcp.ParseBoolean(req, "dry_run", false)
	continueOnError := mcp.ParseBoolean(req, "continue_on_error", false)
	validateAll := mcp.ParseBoolean(req, "validate_all", true)

	var results []string
	var errors []string

	// Validate all files and operations first if requested
	if validateAll || atomic {
		for i, fileReq := range fileRequests {
			if !common.IsPathAllowed(fileReq.Path) {
				err := fmt.Sprintf("Access to path %s (file %d) is not allowed", fileReq.Path, i+1)
				if atomic {
					return mcp.NewToolResultError(err), nil
				}
				errors = append(errors, err)
				continue
			}

			// Check if file exists and is readable
			content, err := os.ReadFile(fileReq.Path)
			if err != nil {
				errMsg := fmt.Sprintf("File %s (file %d) is not accessible: %v", fileReq.Path, i+1, err)
				if atomic {
					return mcp.NewToolResultError(errMsg), nil
				}
				errors = append(errors, errMsg)
				continue
			}

			lines := common.SplitLines(string(content))
			if err := common.ValidateEditOperations(lines, fileReq.Operations); err != nil {
				errMsg := fmt.Sprintf("Invalid operations in file %s: %v", fileReq.Path, err)
				if atomic {
					return mcp.NewToolResultError(errMsg), nil
				}
				errors = append(errors, errMsg)
			}
		}

		if atomic && len(errors) > 0 {
			return mcp.NewToolResultError("Validation failed: " + strings.Join(errors, "; ")), nil
		}
	}

	// Process each file
	for i, fileReq := range fileRequests {
		if !common.IsPathAllowed(fileReq.Path) {
			errMsg := fmt.Sprintf("Access to path %s (file %d) is not allowed", fileReq.Path, i+1)
			if atomic {
				return mcp.NewToolResultError(errMsg), nil
			}
			errors = append(errors, errMsg)
			if !continueOnError {
				break
			}
			continue
		}

		content, err := os.ReadFile(fileReq.Path)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to read file %s: %v", fileReq.Path, err)
			if atomic {
				return mcp.NewToolResultError(errMsg), nil
			}
			errors = append(errors, errMsg)
			if !continueOnError {
				break
			}
			continue
		}

		lines := common.SplitLines(string(content))

		if dryRun {
			preview := common.GenerateEditPreview(lines, fileReq.Operations)
			results = append(results, fmt.Sprintf("File: %s\n%s", fileReq.Path, preview))
			continue
		}

		// Create backup if requested
		if fileReq.CreateBackup {
			if _, err := common.CreateBackup(fileReq.Path); err != nil {
				errMsg := fmt.Sprintf("Failed to create backup for %s: %v", fileReq.Path, err)
				if atomic {
					return mcp.NewToolResultError(errMsg), nil
				}
				errors = append(errors, errMsg)
				if !continueOnError {
					break
				}
				continue
			}
		}

		// Sort operations by start line (descending)
		sortedOps := common.SortOperationsByLine(fileReq.Operations)

		// Apply operations
		resultLines := make([]string, len(lines))
		copy(resultLines, lines)

		for _, op := range sortedOps {
			startIdx := op.StartLine - 1
			endIdx := op.EndLine

			newLines := make([]string, 0, len(resultLines)+(strings.Count(op.Replacement, "\n")+1)-(endIdx-startIdx))
			newLines = append(newLines, resultLines[:startIdx]...)
			newLines = append(newLines, common.SplitLines(op.Replacement)...)
			newLines = append(newLines, resultLines[endIdx:]...)

			resultLines = newLines
		}

		// Write file
		newContent := common.JoinLines(resultLines)
		err = os.WriteFile(fileReq.Path, []byte(newContent), 0644)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to write file %s: %v", fileReq.Path, err)
			if atomic {
				return mcp.NewToolResultError(errMsg), nil
			}
			errors = append(errors, errMsg)
			if !continueOnError {
				break
			}
			continue
		}

		results = append(results, fmt.Sprintf("Successfully applied %d operations to %s", len(fileReq.Operations), fileReq.Path))
	}

	// Prepare result
	var result strings.Builder
	if dryRun {
		result.WriteString("DRY RUN - Preview of changes:\n\n")
	}

	for _, res := range results {
		result.WriteString(res + "\n")
	}

	if len(errors) > 0 {
		result.WriteString("\nErrors encountered:\n")
		for _, err := range errors {
			result.WriteString("- " + err + "\n")
		}
	}

	return mcp.NewToolResultText(result.String()), nil
}

func HandleReplaceText(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid path parameter: %v", err)), nil
	}

	if !common.IsPathAllowed(path) {
		return mcp.NewToolResultError("Access to this path is not allowed"), nil
	}

	find, err := req.RequireString("find")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid find parameter: %v", err)), nil
	}

	replace, err := req.RequireString("replace")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid replace parameter: %v", err)), nil
	}

	regex := mcp.ParseBoolean(req, "regex", false)
	caseSensitive := mcp.ParseBoolean(req, "case_sensitive", true)
	wholeWord := mcp.ParseBoolean(req, "whole_word", false)
	maxReplacements := int(mcp.ParseFloat64(req, "max_replacements", -1))
	createBackup := mcp.ParseBoolean(req, "create_backup", true)

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to read file: %v", err)), nil
	}

	originalContent := string(content)

	// Create backup
	if createBackup {
		if _, err := common.CreateBackup(path); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create backup: %v", err)), nil
		}
	}

	// Perform replacement
	newContent, count, err := common.ReplaceText(originalContent, find, replace, regex, caseSensitive, wholeWord, maxReplacements)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to replace text: %v", err)), nil
	}

	// Write file
	err = os.WriteFile(path, []byte(newContent), 0644)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to write file: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Replaced %d occurrences in %s", count, path)), nil
}

func HandleInsertText(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid path parameter: %v", err)), nil
	}

	if !common.IsPathAllowed(path) {
		return mcp.NewToolResultError("Access to this path is not allowed"), nil
	}

	insertionsStr, err := req.RequireString("insertions")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid insertions parameter: %v", err)), nil
	}

	var insertions = &[]types.TextInsertion{}
	if err := json.Unmarshal([]byte(insertionsStr), insertions); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse insertions: %v", err)), nil
	}

	createBackup := mcp.ParseBoolean(req, "create_backup", true)
	adjustLineNumbers := mcp.ParseBoolean(req, "adjust_line_numbers", true)

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to read file: %v", err)), nil
	}

	originalContent := string(content)

	// Create backup
	if createBackup {
		if _, err := common.CreateBackup(path); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create backup: %v", err)), nil
		}
	}

	// Apply insertions
	newContent, err := common.ApplyTextInsertions(originalContent, *insertions, adjustLineNumbers)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to apply insertions: %v", err)), nil
	}

	// Write file
	err = os.WriteFile(path, []byte(newContent), 0644)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to write file: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Applied %d insertions to %s", len(*insertions), path)), nil
}

func HandleFormatCode(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid path parameter: %v", err)), nil
	}

	if !common.IsPathAllowed(path) {
		return mcp.NewToolResultError("Access to this path is not allowed"), nil
	}

	formatter := mcp.ParseString(req, "formatter", "")
	createBackup := mcp.ParseBoolean(req, "create_backup", true)
	configFile := mcp.ParseString(req, "config_file", "")

	// Create backup
	if createBackup {
		if _, err := common.CreateBackup(path); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create backup: %v", err)), nil
		}
	}

	// Format code
	err = common.FormatCodeFile(path, formatter, configFile)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format code: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Code formatted successfully: %s", path)), nil
}
