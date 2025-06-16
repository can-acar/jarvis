package common

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"jarvis/internal/types"
)

var (
	// Global configuration instance
	instance *types.ServerConfig
	mutex    sync.RWMutex
	once     sync.Once
)

const (
	DefaultShell           = "bash"
	DefaultFileReadLimit   = 1000
	DefaultFileWriteLimit  = 50
	DefaultTelemetryStatus = false
)

func Initialize() {
	once.Do(func() {
		instance = &types.ServerConfig{
			BlockedCommands:    []string{"rm -rf", "dd", "mkfs", "format", "del /f /s /q"},
			DefaultShell:       DefaultShell,
			AllowedDirectories: []string{"/home", "/tmp", "/var/log", "/opt/jarvis"},
			FileReadLineLimit:  DefaultFileReadLimit,
			FileWriteLineLimit: DefaultFileWriteLimit,
			TelemetryEnabled:   DefaultTelemetryStatus,
		}

		// Try to load from config file if exists
		loadFromFile()
	})
}

func Get() *types.ServerConfig {
	mutex.RLock()
	defer mutex.RUnlock()

	if instance == nil {
		Initialize()
	}

	// Return a copy to prevent external modification
	config := *instance
	return &config
}

// Set updates a configuration value (thread-safe)
func Set(key, value string) error {
	mutex.Lock()
	defer mutex.Unlock()

	if instance == nil {
		Initialize()
	}

	switch key {
	case "defaultShell":
		instance.DefaultShell = value
	case "telemetryEnabled":
		instance.TelemetryEnabled = value == "true"
	case "fileReadLineLimit":
		if limit, err := parseIntValue(value); err == nil {
			instance.FileReadLineLimit = limit
		} else {
			return fmt.Errorf("invalid fileReadLineLimit value: %s", value)
		}
	case "fileWriteLineLimit":
		if limit, err := parseIntValue(value); err == nil {
			instance.FileWriteLineLimit = limit
		} else {
			return fmt.Errorf("invalid fileWriteLineLimit value: %s", value)
		}
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	// Save to file after successful update
	saveToFile()
	return nil
}

// GetJSON returns the configuration as a JSON string
func GetJSON() (string, error) {
	config := Get()
	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config to JSON: %v", err)
	}
	return string(jsonData), nil
}

// AddAllowedDirectory adds a directory to the allowed list
func AddAllowedDirectory(dir string) error {
	mutex.Lock()
	defer mutex.Unlock()

	if instance == nil {
		Initialize()
	}

	// Check if already exists
	for _, existing := range instance.AllowedDirectories {
		if existing == dir {
			return nil // Already exists
		}
	}

	instance.AllowedDirectories = append(instance.AllowedDirectories, dir)
	saveToFile()
	return nil
}

// RemoveAllowedDirectory removes a directory from the allowed list
func RemoveAllowedDirectory(dir string) error {
	mutex.Lock()
	defer mutex.Unlock()

	if instance == nil {
		Initialize()
	}

	for i, existing := range instance.AllowedDirectories {
		if existing == dir {
			instance.AllowedDirectories = append(
				instance.AllowedDirectories[:i],
				instance.AllowedDirectories[i+1:]...,
			)
			saveToFile()
			return nil
		}
	}

	return fmt.Errorf("directory not found in allowed list: %s", dir)
}

// AddBlockedCommand adds a command pattern to the blocked list
func AddBlockedCommand(pattern string) error {
	mutex.Lock()
	defer mutex.Unlock()

	if instance == nil {
		Initialize()
	}

	// Check if already exists
	for _, existing := range instance.BlockedCommands {
		if existing == pattern {
			return nil // Already exists
		}
	}

	instance.BlockedCommands = append(instance.BlockedCommands, pattern)
	saveToFile()
	return nil
}

// IsCommandBlocked checks if a command contains any blocked patterns
func IsCommandBlocked(command string) bool {
	config := Get()

	for _, blocked := range config.BlockedCommands {
		if contains(command, blocked) {
			return true
		}
	}
	return false
}

// IsPathAllowed checks if a path is within allowed directories

// Validate checks if the current configuration is valid
func Validate() error {
	config := Get()

	if config.DefaultShell == "" {
		return fmt.Errorf("defaultShell cannot be empty")
	}

	if config.FileReadLineLimit < 1 {
		return fmt.Errorf("fileReadLineLimit must be positive")
	}

	if config.FileWriteLineLimit < 1 {
		return fmt.Errorf("fileWriteLineLimit must be positive")
	}

	if len(config.AllowedDirectories) == 0 {
		return fmt.Errorf("at least one allowed directory must be specified")
	}

	return nil
}

// Reset resets the configuration to default values
func Reset() {
	mutex.Lock()
	defer mutex.Unlock()

	instance = nil
	Initialize()
	saveToFile()
}

// Helper functions

func parseIntValue(value string) (int, error) {
	var result int
	_, err := fmt.Sscanf(value, "%d", &result)
	return result, err
}

func contains(text, pattern string) bool {
	return len(text) >= len(pattern) &&
		(text == pattern ||
			text[:len(pattern)] == pattern ||
			text[len(text)-len(pattern):] == pattern ||
			indexOf(text, pattern) != -1)
}

func indexOf(text, pattern string) int {
	for i := 0; i <= len(text)-len(pattern); i++ {
		if text[i:i+len(pattern)] == pattern {
			return i
		}
	}
	return -1
}

// Configuration file management

func getConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "jarvis-mcp.json"
	}
	return filepath.Join(homeDir, ".jarvis-mcp.json")
}

func loadFromFile() {
	configPath := getConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return // Use defaults if config file doesn't exist
	}

	var fileConfig types.ServerConfig
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return // Use defaults if config file is invalid
	}

	// Merge with defaults (keep existing values, add missing ones)
	if len(fileConfig.BlockedCommands) > 0 {
		instance.BlockedCommands = fileConfig.BlockedCommands
	}
	if fileConfig.DefaultShell != "" {
		instance.DefaultShell = fileConfig.DefaultShell
	}
	if len(fileConfig.AllowedDirectories) > 0 {
		instance.AllowedDirectories = fileConfig.AllowedDirectories
	}
	if fileConfig.FileReadLineLimit > 0 {
		instance.FileReadLineLimit = fileConfig.FileReadLineLimit
	}
	if fileConfig.FileWriteLineLimit > 0 {
		instance.FileWriteLineLimit = fileConfig.FileWriteLineLimit
	}
	instance.TelemetryEnabled = fileConfig.TelemetryEnabled
}

func saveToFile() {
	configPath := getConfigPath()
	data, err := json.MarshalIndent(instance, "", "  ")
	if err != nil {
		return // Silently fail if can't marshal
	}

	os.WriteFile(configPath, data, 0644)
}

// GenerateCharacterDiff creates a character-level diff between two strings
func GenerateCharacterDiff(original, replacement string) string {
	if original == replacement {
		return "No changes"
	}

	var diff strings.Builder
	diff.WriteString("- Original:\n")
	diff.WriteString(original)
	diff.WriteString("\n+ Replacement:\n")
	diff.WriteString(replacement)
	diff.WriteString("\n")

	return diff.String()
}

// OperationsOverlap checks if two edit operations overlap
func OperationsOverlap(op1, op2 types.EditOperation) bool {
	return !(op1.EndLine < op2.StartLine || op2.EndLine < op1.StartLine)
}

// GenerateEditPreview creates a preview of edit operations
func GenerateEditPreview(lines []string, operations []types.EditOperation) string {
	var preview strings.Builder

	for _, op := range operations {
		preview.WriteString(fmt.Sprintf("Lines %d-%d:\n", op.StartLine, op.EndLine))
		preview.WriteString("- Original:\n")
		for i := op.StartLine - 1; i < op.EndLine && i < len(lines); i++ {
			preview.WriteString(fmt.Sprintf("  %d: %s\n", i+1, lines[i]))
		}
		preview.WriteString("+ Replacement:\n")
		replacementLines := strings.Split(op.Replacement, "\n")
		for i, line := range replacementLines {
			preview.WriteString(fmt.Sprintf("  %d: %s\n", op.StartLine+i, line))
		}
		if op.Description != "" {
			preview.WriteString(fmt.Sprintf("  Description: %s\n", op.Description))
		}
		preview.WriteString("\n")
	}

	return preview.String()
}

// SortOperationsByLine sorts edit operations by start line in descending order
// This prevents line number shifts during application
func SortOperationsByLine(operations []types.EditOperation) []types.EditOperation {
	sorted := make([]types.EditOperation, len(operations))
	copy(sorted, operations)

	// Simple bubble sort by start line (descending)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].StartLine < sorted[j].StartLine {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// ValidateEditOperations checks if edit operations are valid for given file content
func ValidateEditOperations(lines []string, operations []types.EditOperation) error {
	for i, op := range operations {
		if op.StartLine < 1 || op.EndLine < 1 {
			return fmt.Errorf("operation %d: line numbers must be positive", i+1)
		}
		if op.StartLine > len(lines) || op.EndLine > len(lines) {
			return fmt.Errorf("operation %d: line numbers exceed file length (%d lines)", i+1, len(lines))
		}
		if op.StartLine > op.EndLine {
			return fmt.Errorf("operation %d: start_line (%d) > end_line (%d)", i+1, op.StartLine, op.EndLine)
		}
	}

	// Check for overlapping operations
	for i := 0; i < len(operations); i++ {
		for j := i + 1; j < len(operations); j++ {
			if OperationsOverlap(operations[i], operations[j]) {
				return fmt.Errorf("operations %d and %d overlap", i+1, j+1)
			}
		}
	}

	return nil
}

// File utilities

// CreateBackup creates a timestamped backup of a file
func CreateBackup(filePath string) (string, error) {
	backupPath := filePath + ".backup." + fmt.Sprintf("%d", time.Now().Unix())

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read original file: %v", err)
	}

	err = os.WriteFile(backupPath, content, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create backup: %v", err)
	}

	return backupPath, nil
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(dirPath string) error {
	return os.MkdirAll(dirPath, 0755)
}

// GetFileExtension returns the file extension (with dot)
func GetFileExtension(filePath string) string {
	return filepath.Ext(filePath)
}

// IsTextFile checks if a file is likely a text file based on extension
func IsTextFile(filePath string) bool {
	textExtensions := []string{
		".txt", ".md", ".json", ".xml", ".yaml", ".yml",
		".go", ".py", ".js", ".ts", ".java", ".c", ".cpp",
		".h", ".hpp", ".cs", ".php", ".rb", ".rs", ".kt",
		".html", ".css", ".scss", ".sass", ".less",
		".sql", ".sh", ".bat", ".ps1", ".dockerfile",
		".cfg", ".conf", ".ini", ".toml", ".properties",
	}

	ext := strings.ToLower(GetFileExtension(filePath))
	for _, textExt := range textExtensions {
		if ext == textExt {
			return true
		}
	}
	return false
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// Web utilities

// BuildUserAgent creates a user agent string
func BuildUserAgent(appName, version string) string {
	if appName == "" {
		appName = "Jarvis-MCP"
	}
	if version == "" {
		version = "1.0.0"
	}
	return fmt.Sprintf("%s/%s", appName, version)
}

// CreateHTTPClient creates a configured HTTP client with timeout and redirect settings
func CreateHTTPClient(timeout time.Duration, followRedirects bool, maxRedirects int) *http.Client {
	client := &http.Client{
		Timeout: timeout,
	}

	if !followRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else if maxRedirects > 0 {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			return nil
		}
	}

	return client
}

// ValidateURL performs basic URL validation
func ValidateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("URL must start with http:// or https://")
	}

	return nil
}

func ValidateFileSyntax(filePath string, content string) error {
	// Placeholder for syntax validation logic
	// This could be extended to use specific parsers for different file types
	if filePath == "" || content == "" {
		return fmt.Errorf("file path and content cannot be empty")
	}

	if !IsTextFile(filePath) {
		return fmt.Errorf("file is not a recognized text file type")
	}

	return nil

}

// GetContentType returns the content type of a file based on its extension
func GetContentType(filePath string) string {
	ext := GetFileExtension(filePath)
	switch ext {
	case ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	default:
		return "application/octet-stream"
	}
}

// GetContentTypeCategory returns the general category of a content type
func GetContentTypeCategory(contentType string) string {
	contentType = strings.ToLower(contentType)

	switch {
	case strings.HasPrefix(contentType, "text/"):
		return "text"
	case strings.HasPrefix(contentType, "image/"):
		return "image"
	case strings.HasPrefix(contentType, "audio/"):
		return "audio"
	case strings.HasPrefix(contentType, "video/"):
		return "video"
	case strings.HasPrefix(contentType, "application/json"):
		return "json"
	case strings.HasPrefix(contentType, "application/xml"):
		return "xml"
	case strings.HasPrefix(contentType, "application/"):
		return "application"
	default:
		return "unknown"
	}
}

// String utilities

// TruncateString truncates a string to maxLength with ellipsis
func TruncateString(str string, maxLength int) string {
	if len(str) <= maxLength {
		return str
	}
	if maxLength <= 3 {
		return str[:maxLength]
	}
	return str[:maxLength-3] + "..."
}

// SplitLines splits a string into lines, handling different line endings
func SplitLines(content string) []string {
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	return strings.Split(content, "\n")
}

// JoinLines joins lines with the appropriate line ending for the platform
func JoinLines(lines []string) string {
	return strings.Join(lines, "\n")
}

// Terminal utilities

// SanitizeCommand removes potentially dangerous characters from command strings
func SanitizeCommand(command string) string {
	// Remove null bytes and other control characters
	sanitized := strings.ReplaceAll(command, "\x00", "")
	sanitized = strings.ReplaceAll(sanitized, "\x01", "")
	sanitized = strings.ReplaceAll(sanitized, "\x02", "")
	sanitized = strings.ReplaceAll(sanitized, "\x03", "")

	return sanitized
}

// FormatBytes formats byte counts as human-readable strings
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Error utilities

// WrapError wraps an error with additional context
func WrapError(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %v", context, err)
}

// FormatError formats an error for user display
func FormatError(err error, operation string) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("Failed to %s: %v", operation, err)
}

// NewErrorResult creates a standardized error result
func NewErrorResult(message string, args ...interface{}) (interface{}, error) {
	if len(args) > 0 {
		return nil, fmt.Errorf(message, args...)
	}
	return nil, fmt.Errorf("%s", message)
}

// NewParameterError creates a standardized parameter error
func NewParameterError(param string, err error) (interface{}, error) {
	return nil, fmt.Errorf("Invalid %s parameter: %v", param, err)
}

// Timing utilities

// FormatDuration formats a duration as a human-readable string
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d)/float64(time.Millisecond))
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// Validation utilities

// ValidateLineRange checks if line range is valid for given content
func ValidateLineRange(startLine, endLine, totalLines int) error {
	if startLine < 1 {
		return fmt.Errorf("start line must be positive, got %d", startLine)
	}
	if endLine < 1 {
		return fmt.Errorf("end line must be positive, got %d", endLine)
	}
	if startLine > totalLines {
		return fmt.Errorf("start line %d exceeds file length (%d lines)", startLine, totalLines)
	}
	if endLine > totalLines {
		return fmt.Errorf("end line %d exceeds file length (%d lines)", endLine, totalLines)
	}
	if startLine > endLine {
		return fmt.Errorf("start line (%d) cannot be greater than end line (%d)", startLine, endLine)
	}
	return nil
}

func CalculateFileChecksum(filePath string) (string, error) {
	// Placeholder for checksum calculation logic
	// This could be implemented using a hash function like SHA-256
	return "", fmt.Errorf("checksum calculation not implemented")
}

func CopyFile(src, dst string) error {
	input, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer input.Close()

	output, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer output.Close()

	_, err = io.Copy(output, input)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}

func FormatFileInfo(name string, info os.FileInfo) string {
	sizeStr := FormatBytes(info.Size())
	if info.IsDir() {
		sizeStr = "<DIR>"
	}

	return fmt.Sprintf("%-10s %10s %s %s\n",
		info.Mode().String(),
		sizeStr,
		info.ModTime().Format("2006-01-02 15:04:05"),
		name)
}

func SearchInFile(filePath, pattern string, caseSensitive bool, contextLines int) ([]string, error) {
	var results []string

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if (caseSensitive && strings.Contains(line, pattern)) || (!caseSensitive && strings.Contains(strings.ToLower(line), strings.ToLower(pattern))) {
			start := i - contextLines
			if start < 0 {
				start = 0
			}
			end := i + contextLines + 1
			if end > len(lines) {
				end = len(lines)
			}
			results = append(results, fmt.Sprintf("Context for line %d:\n%s\n", i+1, strings.Join(lines[start:end], "\n")))
		}
	}

	return results, nil
}

func CreateTempScript(scriptContent string, dir string) (string, error) {
	if dir == "" {
		dir = os.TempDir()
	}

	tempFile, err := os.CreateTemp(dir, "script-*.sh")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary script file: %w", err)
	}

	_, err = tempFile.WriteString(scriptContent)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write script content: %w", err)
	}

	err = tempFile.Close()
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to close temporary script file: %w", err)
	}

	return tempFile.Name(), nil
}

func CleanupTempFile(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("failed to remove temporary file %s: %w", filePath, err)
	}

	return nil
}

func ParseInt64(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	return strconv.ParseInt(s, 10, 64)
}

func ConvertImageFormat(filePath, targetFormat string) (string, error) {
	// Open the source image
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open image: %v", err)
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %v", err)
	}

	// Create destination file path with new extension
	ext := "." + strings.ToLower(targetFormat)
	baseFilePath := strings.TrimSuffix(filePath, filepath.Ext(filePath))
	newPath := baseFilePath + ext

	// Create the destination file
	destFile, err := os.Create(newPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %v", err)
	}
	defer destFile.Close()

	// Encode the image to the desired format
	switch strings.ToLower(targetFormat) {
	case "jpg", "jpeg":
		err = jpeg.Encode(destFile, img, &jpeg.Options{Quality: 90})
	case "png":
		err = png.Encode(destFile, img)
	default:
		return "", fmt.Errorf("unsupported format: %s", targetFormat)
	}

	if err != nil {
		return "", fmt.Errorf("failed to encode image: %v", err)
	}

	return newPath, nil
}

// ApplyJSONPath extracts data from a JSON object using a JSONPath expression
func ApplyJSONPath(data interface{}, jsonPath string) (interface{}, error) {
	if jsonPath == "" {
		return data, nil
	}

	// This is a simplified implementation
	// In a real implementation, you would use a JSONPath library
	parts := strings.Split(jsonPath, ".")
	current := data

	for _, part := range parts {
		if part == "$" || part == "" {
			continue // Root or empty segment
		}

		// Handle array indexing
		if strings.Contains(part, "[") && strings.Contains(part, "]") {
			key := part[:strings.Index(part, "[")]
			idxStr := part[strings.Index(part, "[")+1 : strings.Index(part, "]")]
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				return nil, fmt.Errorf("invalid array index in JSONPath: %s", part)
			}

			// Get the map value for the key
			if m, ok := current.(map[string]interface{}); ok {
				if val, exists := m[key]; exists {
					// Check if it's an array
					if arr, ok := val.([]interface{}); ok {
						if idx >= 0 && idx < len(arr) {
							current = arr[idx]
							continue
						}
						return nil, fmt.Errorf("array index out of bounds: %d", idx)
					}
					return nil, fmt.Errorf("not an array: %s", key)
				}
				return nil, fmt.Errorf("key not found: %s", key)
			}
			return nil, fmt.Errorf("not a map at path segment: %s", part)
		}

		// Handle regular object property
		if m, ok := current.(map[string]interface{}); ok {
			if val, exists := m[part]; exists {
				current = val
			} else {
				return nil, fmt.Errorf("key not found: %s", part)
			}
		} else {
			return nil, fmt.Errorf("not a map at path segment: %s", part)
		}
	}

	return current, nil
}

// FetchURLsBatch fetches multiple URLs in parallel with configurable parameters
func FetchURLsBatch(ctx context.Context, urlConfigs []types.HTTPRequestConfig, maxConcurrent, delayMs int, failFast, includeTiming bool) ([]types.OperationResult, error) {
	if len(urlConfigs) == 0 {
		return nil, fmt.Errorf("no URLs provided")
	}

	if maxConcurrent <= 0 {
		maxConcurrent = 5 // Default to 5 concurrent requests
	}

	results := make([]types.OperationResult, len(urlConfigs))

	// Simple implementation: process URLs sequentially
	// In a real implementation, you would use goroutines and channels for concurrency
	for i, config := range urlConfigs {
		// Check for context cancellation
		if ctx.Err() != nil {
			return results, ctx.Err()
		}

		// Validate URL
		if err := ValidateURL(config.URL); err != nil {
			results[i] = types.OperationResult{
				Success: false,
				Error:   fmt.Sprintf("Invalid URL: %v", err),
				Metadata: map[string]interface{}{
					"url": config.URL,
				},
			}
			if failFast {
				return results, fmt.Errorf("URL validation failed: %v", err)
			}
			continue
		}

		// Apply delay if specified
		if delayMs > 0 && i > 0 {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}

		// Prepare HTTP client
		timeout := 30 * time.Second
		if config.Timeout > 0 {
			timeout = time.Duration(config.Timeout) * time.Second
		}
		client := &http.Client{Timeout: timeout}

		// Prepare request
		var bodyReader io.Reader
		if config.Body != "" {
			bodyReader = strings.NewReader(config.Body)
		}

		method := config.Method
		if method == "" {
			method = "GET"
		}

		req, err := http.NewRequestWithContext(ctx, method, config.URL, bodyReader)
		if err != nil {
			results[i] = types.OperationResult{
				Success: false,
				Error:   fmt.Sprintf("Failed to create request: %v", err),
				Metadata: map[string]interface{}{
					"url": config.URL,
				},
			}
			if failFast {
				return results, fmt.Errorf("request creation failed: %v", err)
			}
			continue
		}

		// Set headers
		userAgent := config.UserAgent
		if userAgent == "" {
			userAgent = BuildUserAgent("Jarvis-MCP", "1.0.0")
		}
		req.Header.Set("User-Agent", userAgent)

		for key, value := range config.Headers {
			req.Header.Set(key, value)
		}

		// Execute request
		startTime := time.Now()
		resp, err := client.Do(req)
		duration := time.Since(startTime)

		if err != nil {
			results[i] = types.OperationResult{
				Success: false,
				Error:   fmt.Sprintf("Request failed: %v", err),
				Metadata: map[string]interface{}{
					"url":      config.URL,
					"duration": FormatDuration(duration),
				},
			}
			if failFast {
				return results, fmt.Errorf("request failed: %v", err)
			}
			continue
		}
		defer resp.Body.Close()

		// Read response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			results[i] = types.OperationResult{
				Success: false,
				Error:   fmt.Sprintf("Failed to read response: %v", err),
				Metadata: map[string]interface{}{
					"url":         config.URL,
					"status_code": resp.StatusCode,
					"duration":    FormatDuration(duration),
				},
			}
			if failFast {
				return results, fmt.Errorf("response reading failed: %v", err)
			}
			continue
		}

		// Prepare result
		metadata := map[string]interface{}{
			"url":            config.URL,
			"status_code":    resp.StatusCode,
			"content_type":   resp.Header.Get("Content-Type"),
			"content_length": resp.ContentLength,
		}

		if includeTiming {
			metadata["duration"] = FormatDuration(duration)
		}

		results[i] = types.OperationResult{
			Success:  resp.StatusCode < 400,
			Message:  fmt.Sprintf("Status: %s", resp.Status),
			Data:     string(body),
			Metadata: metadata,
		}
	}

	return results, nil
}

// CheckURLsStatus checks the status of multiple URLs
func CheckURLsStatus(ctx context.Context, urls []string, timeout time.Duration, followRedirects, checkSSL, includeHeaders bool) ([]types.OperationResult, error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("no URLs provided")
	}

	results := make([]types.OperationResult, len(urls))

	// Configure HTTP client
	client := &http.Client{
		Timeout: timeout,
	}

	// Disable redirect following if requested
	if !followRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// Process each URL
	for i, url := range urls {
		// Check for context cancellation
		if ctx.Err() != nil {
			return results, ctx.Err()
		}

		// Validate URL
		if err := ValidateURL(url); err != nil {
			results[i] = types.OperationResult{
				Success: false,
				Error:   fmt.Sprintf("Invalid URL: %v", err),
				Metadata: map[string]interface{}{
					"url": url,
				},
			}
			continue
		}

		// Prepare request
		req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
		if err != nil {
			results[i] = types.OperationResult{
				Success: false,
				Error:   fmt.Sprintf("Failed to create request: %v", err),
				Metadata: map[string]interface{}{
					"url": url,
				},
			}
			continue
		}

		req.Header.Set("User-Agent", BuildUserAgent("Jarvis-MCP", "1.0.0"))

		// Execute request
		startTime := time.Now()
		resp, err := client.Do(req)
		duration := time.Since(startTime)

		if err != nil {
			results[i] = types.OperationResult{
				Success: false,
				Error:   fmt.Sprintf("Request failed: %v", err),
				Metadata: map[string]interface{}{
					"url":      url,
					"duration": FormatDuration(duration),
				},
			}
			continue
		}

		// Prepare metadata
		metadata := map[string]interface{}{
			"url":          url,
			"status_code":  resp.StatusCode,
			"status":       resp.Status,
			"duration":     FormatDuration(duration),
			"content_type": resp.Header.Get("Content-Type"),
		}

		// Include headers if requested
		if includeHeaders {
			headers := make(map[string]string)
			for key, values := range resp.Header {
				headers[key] = strings.Join(values, ", ")
			}
			metadata["headers"] = headers
		}

		results[i] = types.OperationResult{
			Success:  resp.StatusCode < 400,
			Message:  fmt.Sprintf("Status: %s", resp.Status),
			Metadata: metadata,
		}

		resp.Body.Close()
	}

	return results, nil
}

// ReplaceText replaces text in a string with various options
func ReplaceText(content, find, replace string, regex, caseSensitive, wholeWord bool, maxReplacements int) (string, int, error) {
	if find == "" {
		return content, 0, fmt.Errorf("find pattern cannot be empty")
	}

	// For simple string replacement
	if !regex && !wholeWord && caseSensitive {
		count := strings.Count(content, find)
		if maxReplacements > 0 && count > maxReplacements {
			count = maxReplacements
			// This is a simplified implementation
			// In a real implementation, you would need to limit the number of replacements
		}
		result := strings.Replace(content, find, replace, maxReplacements)
		return result, count, nil
	}

	// For case-insensitive string replacement
	if !regex && !wholeWord && !caseSensitive {
		findLower := strings.ToLower(find)
		count := 0
		result := ""
		remaining := content

		for {
			lowerRemaining := strings.ToLower(remaining)
			index := strings.Index(lowerRemaining, findLower)
			if index == -1 || (maxReplacements > 0 && count >= maxReplacements) {
				result += remaining
				break
			}

			result += remaining[:index] + replace
			remaining = remaining[index+len(find):]
			count++
		}

		return result, count, nil
	}

	// For more complex replacements (regex, whole word)
	// In a real implementation, you would use the regexp package
	return content, 0, fmt.Errorf("regex and whole word replacements not implemented in this simplified version")
}

// ApplyTextInsertions applies multiple text insertions to a string
func ApplyTextInsertions(content string, insertions []types.TextInsertion, adjustLineNumbers bool) (string, error) {
	if len(insertions) == 0 {
		return content, nil
	}

	lines := SplitLines(content)

	// Sort insertions by line number in descending order to avoid line number shifts
	// Simple bubble sort
	for i := 0; i < len(insertions)-1; i++ {
		for j := i + 1; j < len(insertions); j++ {
			if insertions[i].Line < insertions[j].Line {
				insertions[i], insertions[j] = insertions[j], insertions[i]
			}
		}
	}

	// Apply insertions
	for _, insertion := range insertions {
		line := insertion.Line

		// Validate line number
		if line < 1 || line > len(lines)+1 {
			return content, fmt.Errorf("invalid line number: %d (file has %d lines)", line, len(lines))
		}

		// Insert content
		insertContent := insertion.Content
		insertLines := SplitLines(insertContent)

		if insertion.Before {
			// Insert before the line
			newLines := make([]string, 0, len(lines)+len(insertLines))
			newLines = append(newLines, lines[:line-1]...)
			newLines = append(newLines, insertLines...)
			newLines = append(newLines, lines[line-1:]...)
			lines = newLines
		} else {
			// Insert after the line
			newLines := make([]string, 0, len(lines)+len(insertLines))
			newLines = append(newLines, lines[:line]...)
			newLines = append(newLines, insertLines...)
			newLines = append(newLines, lines[line:]...)
			lines = newLines
		}
	}

	return JoinLines(lines), nil
}

// FormatCodeFile formats a code file using the specified formatter
func FormatCodeFile(filePath, formatter, configFile string) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	// Determine formatter based on file extension if not specified
	if formatter == "" {
		ext := GetFileExtension(filePath)
		switch ext {
		case ".go":
			formatter = "gofmt"
		case ".py":
			formatter = "black"
		case ".js", ".ts", ".jsx", ".tsx", ".json":
			formatter = "prettier"
		case ".java":
			formatter = "google-java-format"
		case ".c", ".cpp", ".h", ".hpp":
			formatter = "clang-format"
		default:
			return fmt.Errorf("no default formatter for file type: %s", ext)
		}
	}

	// This is a placeholder implementation
	// In a real implementation, you would execute the formatter command
	return fmt.Errorf("code formatting not implemented in this simplified version")
}

func IsPathAllowed(path string) bool {
	config := Get()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	for _, allowedDir := range config.AllowedDirectories {
		allowedAbs, err := filepath.Abs(allowedDir)
		if err != nil {
			continue
		}

		if IsSubPath(absPath, allowedAbs) {
			return true
		}
	}

	return false
}

func IsSubPath(path, parent string) bool {
	// Ensure both paths end with separator for proper comparison
	if parent[len(parent)-1] != filepath.Separator {
		parent += string(filepath.Separator)
	}
	if path[len(path)-1] != filepath.Separator {
		path += string(filepath.Separator)
	}

	return len(path) >= len(parent) && path[:len(parent)] == parent
}
