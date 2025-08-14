package types

// StreamCallback represents a callback function for streaming responses
type StreamCallback func(chunk string, isComplete bool) error

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
	BlockedCommands    []string          `json:"blockedCommands"`
	DefaultShell       string            `json:"defaultShell"`
	AllowedDirectories []string          `json:"allowedDirectories"`
	FileReadLineLimit  int               `json:"fileReadLineLimit"`
	FileWriteLineLimit int               `json:"fileWriteLineLimit"`
	TelemetryEnabled   bool              `json:"telemetryEnabled"`
	ModelConfig        *ModelConfig      `json:"modelConfig,omitempty"`
	MCP                []MCPServerConfig `json:"mcp,omitempty"`
}

// ModelConfig represents the LLM model configuration
type ModelConfig struct {
	Model        string `json:"model"`
	ConfigFile   string `json:"configFile,omitempty"`
	SystemPrompt string `json:"systemPrompt,omitempty"`
}

// MCPServerConfig represents configuration for external MCP servers
type MCPServerConfig struct {
	Name        string            `json:"name"`
	Command     string            `json:"command"`
	Args        []string          `json:"args,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	WorkingDir  string            `json:"workingDir,omitempty"`
	Enabled     bool              `json:"enabled"`
}

// CommandLineArgs represents parsed command line arguments
type CommandLineArgs struct {
	Model        string
	ConfigFile   string
	SystemPrompt string
	Interactive  bool
	Help         bool
	Version      bool
}

// AgentRequest represents a user request to the agent
type AgentRequest struct {
	Query      string            `json:"query"`
	Context    map[string]string `json:"context,omitempty"`
	WorkingDir string            `json:"working_dir,omitempty"`
	UseTools   bool              `json:"use_tools,omitempty"`
	StreamMode bool              `json:"stream_mode,omitempty"`
	History    []Message         `json:"history,omitempty"`
}

// Message represents a conversation message
type Message struct {
	Role      string     `json:"role"` // "user", "assistant", "system"
	Content   string     `json:"content"`
	Timestamp int64      `json:"timestamp"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool call made during conversation
type ToolCall struct {
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
	Result     interface{}            `json:"result,omitempty"`
	Success    bool                   `json:"success"`
}

// AgentResponse represents the agent's response
type AgentResponse struct {
	Response             string                 `json:"response"`
	ToolsUsed            []string               `json:"tools_used,omitempty"`
	Context              map[string]interface{} `json:"context,omitempty"`
	Success              bool                   `json:"success"`
	Error                string                 `json:"error,omitempty"`
	RequiresConfirmation bool                   `json:"requires_confirmation,omitempty"`
	ConfirmationMessage  string                 `json:"confirmation_message,omitempty"`
	PendingAction        interface{}            `json:"pending_action,omitempty"`
}

// ConfirmationRequest represents a request for user confirmation
type ConfirmationRequest struct {
	Message     string      `json:"message"`
	Action      string      `json:"action"`
	RiskLevel   string      `json:"risk_level"` // "low", "medium", "high", "critical"
	Details     interface{} `json:"details,omitempty"`
	AutoConfirm bool        `json:"auto_confirm,omitempty"`
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
