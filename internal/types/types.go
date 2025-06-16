package types

// EditOperation represents a single edit operation
type EditOperation struct {
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	Replacement string `json:"replacement"`
	Description string `json:"description,omitempty"`
}

// FileEditRequest represents multiple edits for a single file
type FileEditRequest struct {
	Path         string          `json:"path"`
	Operations   []EditOperation `json:"operations"`
	CreateBackup bool            `json:"create_backup,omitempty"`
}

// MultiFileEditRequest represents edits for multiple files
type MultiFileEditRequest struct {
	Files  []FileEditRequest `json:"files"`
	Atomic bool              `json:"atomic,omitempty"`
	DryRun bool              `json:"dry_run,omitempty"`
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	BlockedCommands    []string `json:"blockedCommands"`
	DefaultShell       string   `json:"defaultShell"`
	AllowedDirectories []string `json:"allowedDirectories"`
	FileReadLineLimit  int      `json:"fileReadLineLimit"`
	FileWriteLineLimit int      `json:"fileWriteLineLimit"`
	TelemetryEnabled   bool     `json:"telemetryEnabled"`
}

// HTTPRequestConfig represents HTTP request configuration
type HTTPRequestConfig struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body,omitempty"`
	Timeout   int               `json:"timeout,omitempty"`
	UserAgent string            `json:"user_agent,omitempty"`
	Validate  bool              `json:"validate,omitempty"`
}

// FileDownloadConfig represents file download configuration
type FileDownloadConfig struct {
	URL           string            `json:"url"`
	FilePath      string            `json:"filepath"`
	Headers       map[string]string `json:"headers,omitempty"`
	Overwrite     bool              `json:"overwrite,omitempty"`
	ValidateImage bool              `json:"validate_image,omitempty"`
	Format        string            `json:"format,omitempty"`
}

// CommandExecutionConfig represents command execution configuration
type CommandExecutionConfig struct {
	Command        string   `json:"command"`
	Shell          string   `json:"shell,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
	WorkingDir     string   `json:"working_dir,omitempty"`
	Environment    []string `json:"environment,omitempty"`
}

// OperationResult represents the result of any operation
type OperationResult struct {
	Success  bool                   `json:"success"`
	Message  string                 `json:"message"`
	Data     interface{}            `json:"data,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type TextInsertion struct {
	Line    int    `json:"line"`
	Content string `json:"content"`
	Before  bool   `json:"before,omitempty"` // If true, insert before the line, otherwise after
}
type TextInsertionRequest struct {
	Insertions []TextInsertion `json:"insertions"`
}
type TextInsertionResponse struct {
	Success  bool                   `json:"success"`
	Message  string                 `json:"message"`
	Data     string                 `json:"data,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
