package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"jarvis/internal/bridge"
	"jarvis/internal/common"
	"jarvis/internal/types"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LLMClient handles communication with LLM models
type LLMClient struct {
	config     *types.ModelConfig
	toolBridge *bridge.ToolBridge
}

// OllamaRequest represents a request to Ollama API
type OllamaRequest struct {
	Model    string                 `json:"model"`
	Messages []OllamaMessage        `json:"messages"`
	Tools    []OllamaToolDefinition `json:"tools,omitempty"`
	Stream   bool                   `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// OllamaMessage represents a message in chat format
type OllamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []OllamaToolCall `json:"tool_calls,omitempty"`
}

// OllamaToolDefinition represents a tool definition for Ollama
type OllamaToolDefinition struct {
	Type     string            `json:"type"`
	Function OllamaFunctionDef `json:"function"`
}

// OllamaFunctionDef represents a function definition
type OllamaFunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// OllamaToolCall represents a tool call from the model
type OllamaToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function OllamaFunctionCall `json:"function"`
}

// OllamaFunctionCall represents a function call
type OllamaFunctionCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// OllamaResponse represents a response from Ollama API
type OllamaResponse struct {
	Model   string        `json:"model"`
	Message OllamaMessage `json:"message"`
	Done    bool          `json:"done"`
	Error   string        `json:"error,omitempty"`
}

// NewLLMClient creates a new LLM client with the configured model
func NewLLMClient() (*LLMClient, error) {
	cfg := common.Get()
	if cfg.ModelConfig == nil {
		return nil, fmt.Errorf("no model configuration found")
	}

	return &LLMClient{
		config:     cfg.ModelConfig,
		toolBridge: bridge.NewToolBridge(),
	}, nil
}

// ProcessRequest processes a user request using the configured LLM
func (c *LLMClient) ProcessRequest(ctx context.Context, request *types.AgentRequest) (*types.AgentResponse, error) {
	// Always use function calling approach for proper LLM integration
	if strings.HasPrefix(c.config.Model, "ollama:") {
		return c.processOllamaRequestWithFunctionCalling(ctx, request)
	}

	return nil, fmt.Errorf("unsupported model type: %s", c.config.Model)
}

// ProcessRequestWithStreaming processes a user request with streaming support
func (c *LLMClient) ProcessRequestWithStreaming(ctx context.Context, request *types.AgentRequest, callback types.StreamCallback) error {
	if !strings.HasPrefix(c.config.Model, "ollama:") {
		return fmt.Errorf("unsupported model type for streaming: %s", c.config.Model)
	}

	return c.processOllamaRequestWithStreaming(ctx, request, callback)
}

// buildPrompt constructs the prompt with system context and user query
func (c *LLMClient) buildPrompt(request *types.AgentRequest) string {
	var promptBuilder strings.Builder

	// Add system prompt if configured
	if c.config.SystemPrompt != "" {
		promptBuilder.WriteString(c.config.SystemPrompt)
		promptBuilder.WriteString("\n\n")
	}

	// Add context information
	if request.WorkingDir != "" {
		promptBuilder.WriteString(fmt.Sprintf("Current working directory: %s\n", request.WorkingDir))
	}

	// Add available context
	if len(request.Context) > 0 {
		promptBuilder.WriteString("Context:\n")
		for key, value := range request.Context {
			promptBuilder.WriteString(fmt.Sprintf("- %s: %s\n", key, value))
		}
		promptBuilder.WriteString("\n")
	}

	// Add the user query
	promptBuilder.WriteString("User request: ")
	promptBuilder.WriteString(request.Query)

	return promptBuilder.String()
}

// Legacy function - now redirected to function calling approach
func (c *LLMClient) processRequestWithTools(ctx context.Context, request *types.AgentRequest) (*types.AgentResponse, error) {
	return c.processOllamaRequestWithFunctionCalling(ctx, request)
}

// buildToolAwarePrompt builds a prompt that includes tool capabilities
func (c *LLMClient) buildToolAwarePrompt(request *types.AgentRequest) string {
	var promptBuilder strings.Builder

	// Add system prompt
	if c.config.SystemPrompt != "" {
		promptBuilder.WriteString(c.config.SystemPrompt)
		promptBuilder.WriteString("\n\n")
	}

	// Add tool capabilities information
	promptBuilder.WriteString("Available Tools:\n")
	tools := c.toolBridge.GetAvailableTools()
	for _, tool := range tools {
		promptBuilder.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
	}
	promptBuilder.WriteString("\n")

	// Add context information
	if request.WorkingDir != "" {
		promptBuilder.WriteString(fmt.Sprintf("Current working directory: %s\n", request.WorkingDir))
	}

	// Add available context
	if len(request.Context) > 0 {
		promptBuilder.WriteString("Directory Context:\n")
		for key, value := range request.Context {
			promptBuilder.WriteString(fmt.Sprintf("- %s: %s\n", key, value))
		}
		promptBuilder.WriteString("\n")
	}

	// Add instructions for tool usage
	promptBuilder.WriteString("Instructions:\n")
	promptBuilder.WriteString("- You can perform file operations, directory listings, and searches\n")
	promptBuilder.WriteString("- Use the available tools when appropriate for the user's request\n")
	promptBuilder.WriteString("- Provide clear and helpful responses based on the tool results\n\n")

	// Add the user query
	promptBuilder.WriteString("User request: ")
	promptBuilder.WriteString(request.Query)

	return promptBuilder.String()
}

// Helper functions for pattern matching
func (c *LLMClient) matchesPattern(text string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

// extractDirectoryFromQuery extracts specific directory path from query
func (c *LLMClient) extractDirectoryFromQuery(query, workingDir string) string {
	// Look for common directory names in the query
	words := strings.Fields(query)

	for _, word := range words {
		// Remove common suffixes
		cleanWord := strings.TrimSuffix(word, "klasörü")
		cleanWord = strings.TrimSuffix(cleanWord, "klasörünün")
		cleanWord = strings.TrimSuffix(cleanWord, "folder")
		cleanWord = strings.TrimSuffix(cleanWord, "directory")
		cleanWord = strings.TrimSuffix(cleanWord, "dizinin")
		cleanWord = strings.TrimSuffix(cleanWord, "dizini")

		// Skip common words
		if cleanWord == "içinde" || cleanWord == "in" || cleanWord == "ne" ||
			cleanWord == "var" || cleanWord == "what" || cleanWord == "is" ||
			cleanWord == "neler" || cleanWord == "" {
			continue
		}

		// Check if this could be a directory name
		testPath := filepath.Join(workingDir, cleanWord)
		if info, err := os.Stat(testPath); err == nil && info.IsDir() {
			return testPath
		}
	}

	// Default to working directory if no specific directory found
	return workingDir
}

func (c *LLMClient) matchesFileExtensionPattern(query string) bool {
	extensions := []string{".go", ".js", ".py", ".java", ".txt", ".md", ".json", ".yaml", ".yml"}
	for _, ext := range extensions {
		if strings.Contains(query, ext) {
			return true
		}
	}
	// Also check for "extension" or "files" keywords - English and Turkish
	return c.matchesPattern(query, []string{"extension", "files with", "*.go", "*.js", "*.py", "uzantı", "uzantılı", "dosyalar"})
}

// Tool execution functions
func (c *LLMClient) executeDirectoryListing(ctx context.Context, path string) (interface{}, error) {
	toolCall := bridge.ToolCall{
		Name: "list-directory",
		Parameters: map[string]interface{}{
			"path":        path,
			"show_hidden": false,
		},
	}

	result := c.toolBridge.ExecuteTool(ctx, toolCall)
	if !result.Success {
		return nil, fmt.Errorf(result.Error)
	}

	return result.Result, nil
}

func (c *LLMClient) executeFileSearch(ctx context.Context, request *types.AgentRequest) (interface{}, error) {
	// Extract file extension from query
	query := strings.ToLower(request.Query)
	var extensions []string

	// Common file extensions
	extMap := map[string]string{
		"go":     ".go",
		"js":     ".js",
		"python": ".py",
		"py":     ".py",
		"java":   ".java",
		"txt":    ".txt",
		"md":     ".md",
		"json":   ".json",
		"yaml":   ".yaml",
		"yml":    ".yml",
	}

	for keyword, ext := range extMap {
		if strings.Contains(query, keyword) || strings.Contains(query, ext) {
			extensions = append(extensions, ext)
		}
	}

	if len(extensions) == 0 {
		extensions = []string{".go"} // Default to Go files
	}

	toolCall := bridge.ToolCall{
		Name: "search-files",
		Parameters: map[string]interface{}{
			"pattern":         "*", // Search all files
			"path":            request.WorkingDir,
			"file_extensions": extensions,
		},
	}

	result := c.toolBridge.ExecuteTool(ctx, toolCall)
	if !result.Success {
		return nil, fmt.Errorf(result.Error)
	}

	return result.Result, nil
}

func (c *LLMClient) executeFileRead(ctx context.Context, request *types.AgentRequest) (interface{}, error) {
	query := strings.ToLower(request.Query)
	var filePath string

	// Try to determine which file to read
	if strings.Contains(query, "main.go") {
		filePath = filepath.Join(request.WorkingDir, "main.go")
	} else if strings.Contains(query, "readme") {
		filePath = filepath.Join(request.WorkingDir, "README.md")
	} else if strings.Contains(query, "config") {
		filePath = filepath.Join(request.WorkingDir, "config.yaml")
	} else {
		return nil, fmt.Errorf("could not determine which file to read")
	}

	toolCall := bridge.ToolCall{
		Name: "read-file",
		Parameters: map[string]interface{}{
			"path":  filePath,
			"lines": 100, // Limit to first 100 lines
		},
	}

	result := c.toolBridge.ExecuteTool(ctx, toolCall)
	if !result.Success {
		return nil, fmt.Errorf(result.Error)
	}

	return result.Result, nil
}

// Response formatting functions
func (c *LLMClient) formatDirectoryResponse(result interface{}, query string) string {
	data, ok := result.(map[string]interface{})
	if !ok {
		return "Failed to format directory response"
	}

	path := data["path"].(string)
	entries := data["entries"].([]map[string]interface{})
	count := data["count"].(int)

	var response strings.Builder
	response.WriteString(fmt.Sprintf("📁 Directory contents of %s:\n\n", path))
	response.WriteString(fmt.Sprintf("Found %d items:\n\n", count))

	// Group by type
	var directories, files []map[string]interface{}
	for _, entry := range entries {
		if entry["is_dir"].(bool) {
			directories = append(directories, entry)
		} else {
			files = append(files, entry)
		}
	}

	if len(directories) > 0 {
		response.WriteString("📂 Directories:\n")
		for _, dir := range directories {
			response.WriteString(fmt.Sprintf("  - %s/\n", dir["name"].(string)))
		}
		response.WriteString("\n")
	}

	if len(files) > 0 {
		response.WriteString("📄 Files:\n")
		for _, file := range files {
			size := file["size"].(int64)
			response.WriteString(fmt.Sprintf("  - %s (%s)\n", file["name"].(string), c.formatFileSize(size)))
		}
	}

	return response.String()
}

func (c *LLMClient) formatFileSearchResponse(result interface{}, query string) string {
	data, ok := result.(map[string]interface{})
	if !ok {
		return "Failed to format file search response"
	}

	searchPath := data["search_path"].(string)
	results := data["results"].([]map[string]interface{})
	totalFiles := data["total_files"].(int)

	var response strings.Builder
	response.WriteString(fmt.Sprintf("🔍 File search results in %s:\n\n", searchPath))
	response.WriteString(fmt.Sprintf("Found %d matching files:\n\n", totalFiles))

	if totalFiles == 0 {
		response.WriteString("No files found matching the criteria.\n")
	} else {
		for i, result := range results {
			if i >= 20 { // Limit to first 20 results
				response.WriteString(fmt.Sprintf("... and %d more files\n", totalFiles-20))
				break
			}
			filePath := result["file"].(string)
			response.WriteString(fmt.Sprintf("  📄 %s\n", filePath))
		}
	}

	return response.String()
}

func (c *LLMClient) formatFileReadResponse(result interface{}, query string) string {
	data, ok := result.(map[string]interface{})
	if !ok {
		return "Failed to format file read response"
	}

	path := data["path"].(string)
	content := data["content"].(string)
	totalLines := data["total_lines"].(int)
	linesShown := data["lines_shown"].(int)
	truncated := data["truncated"].(bool)

	var response strings.Builder
	response.WriteString(fmt.Sprintf("📄 Contents of %s:\n\n", path))

	if truncated {
		response.WriteString(fmt.Sprintf("Showing first %d of %d lines:\n\n", linesShown, totalLines))
	} else {
		response.WriteString(fmt.Sprintf("Total lines: %d\n\n", totalLines))
	}

	response.WriteString("```\n")
	response.WriteString(content)
	response.WriteString("\n```\n")

	if truncated {
		response.WriteString(fmt.Sprintf("\n... (truncated, %d more lines available)", totalLines-linesShown))
	}

	return response.String()
}

func (c *LLMClient) formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	} else {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
}

// processOllamaRequestWithFunctionCalling processes request with proper function calling
func (c *LLMClient) processOllamaRequestWithFunctionCalling(ctx context.Context, request *types.AgentRequest) (*types.AgentResponse, error) {
	modelName := strings.TrimPrefix(c.config.Model, "ollama:")

	// Convert Jarvis tools to Ollama tool format
	tools := c.convertToOllamaTools()

	// Build system message
	systemMessage := c.buildSystemMessage(request)

	// Build user message
	userMessage := c.buildUserMessage(request)

	// Prepare Ollama request with function calling
	ollamaReq := OllamaRequest{
		Model: modelName,
		Messages: []OllamaMessage{
			{
				Role:    "system",
				Content: systemMessage,
			},
			{
				Role:    "user",
				Content: userMessage,
			},
		},
		Tools:  tools,
		Stream: false,
	}

	// Marshal request
	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return &types.AgentResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to marshal request: %v", err),
		}, nil
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "http://localhost:11434/api/chat", bytes.NewBuffer(reqBody))
	if err != nil {
		return &types.AgentResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create HTTP request: %v", err),
		}, nil
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return &types.AgentResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to send request to Ollama: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &types.AgentResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to read response: %v", err),
		}, nil
	}

	// Parse Ollama response
	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return &types.AgentResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse response: %v", err),
		}, nil
	}

	// Check for errors
	if ollamaResp.Error != "" {
		return &types.AgentResponse{
			Success: false,
			Error:   fmt.Sprintf("Ollama error: %s", ollamaResp.Error),
		}, nil
	}

	// Process tool calls if any
	if len(ollamaResp.Message.ToolCalls) > 0 {
		return c.handleToolCalls(ctx, ollamaResp.Message.ToolCalls, request)
	}

	// Return direct response if no tool calls
	return &types.AgentResponse{
		Response: ollamaResp.Message.Content,
		Success:  true,
		Context: map[string]interface{}{
			"model":       ollamaResp.Model,
			"working_dir": request.WorkingDir,
		},
	}, nil
}

// convertToOllamaTools converts bridge tools to Ollama tool format
func (c *LLMClient) convertToOllamaTools() []OllamaToolDefinition {
	bridgeTools := c.toolBridge.GetAvailableTools()
	ollamaTools := make([]OllamaToolDefinition, len(bridgeTools))

	for i, tool := range bridgeTools {
		ollamaTools[i] = OllamaToolDefinition{
			Type: "function",
			Function: OllamaFunctionDef{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		}
	}

	return ollamaTools
}

// buildSystemMessage creates system message with context
func (c *LLMClient) buildSystemMessage(request *types.AgentRequest) string {
	var systemBuilder strings.Builder

	// Add configured system prompt
	if c.config.SystemPrompt != "" {
		systemBuilder.WriteString(c.config.SystemPrompt)
		systemBuilder.WriteString("\n\n")
	}

	// Add conversation history context if available
	if historyContext, exists := request.Context["conversation_history"]; exists {
		systemBuilder.WriteString(historyContext)
		systemBuilder.WriteString("\n")
	}

	// Add VERY strong function calling examples
	systemBuilder.WriteString("ZORUNLU FUNCTION CALLING ÖRNEKLERI:\n")
	systemBuilder.WriteString("User: klasör oluştur\n")
	systemBuilder.WriteString("Assistant: [function call: execute-command with args: mkdir yeni-klasor]\n\n")
	systemBuilder.WriteString("User: .net projesi kur\n")
	systemBuilder.WriteString("Assistant: [function call: execute-command with args: mkdir NetProject && cd NetProject && dotnet new webapi]\n\n")
	systemBuilder.WriteString("User: devam edelim\n")
	systemBuilder.WriteString("Assistant: [function call: execute-command with next step from conversation history]\n\n")
	systemBuilder.WriteString("IMPORTANT: You MUST call functions, NOT just explain! Always use execute-command function!\n\n")

	// Emphasize execute-command tool
	systemBuilder.WriteString("PRIMARY TOOL: execute-command\n")
	systemBuilder.WriteString("- Use for: mkdir, dotnet, npm, cd, mv, cp, etc.\n")
	systemBuilder.WriteString("- ALWAYS use this when user wants action\n\n")

	// Add current context
	if request.WorkingDir != "" {
		systemBuilder.WriteString(fmt.Sprintf("Current directory: %s\n", request.WorkingDir))
	}

	return systemBuilder.String()
}

// buildUserMessage creates user message
func (c *LLMClient) buildUserMessage(request *types.AgentRequest) string {
	userQuery := request.Query

	// Add explicit function calling instruction
	if strings.Contains(strings.ToLower(userQuery), "projesi") ||
		strings.Contains(strings.ToLower(userQuery), "klasör") ||
		strings.Contains(strings.ToLower(userQuery), "oluştur") ||
		strings.Contains(strings.ToLower(userQuery), "devam") {
		userQuery += "\n\n[IMPORTANT: You MUST use function calls to complete this request. Call execute-command function immediately.]"
	}

	return userQuery
}

// handleToolCalls processes tool calls from the LLM
func (c *LLMClient) handleToolCalls(ctx context.Context, toolCalls []OllamaToolCall, request *types.AgentRequest) (*types.AgentResponse, error) {
	var toolsUsed []string
	var toolResults []map[string]interface{}

	// Execute all tool calls first
	for _, toolCall := range toolCalls {
		// Arguments are already in the correct format
		args := toolCall.Function.Arguments

		// Execute tool via bridge
		bridgeCall := bridge.ToolCall{
			Name:       toolCall.Function.Name,
			Parameters: args,
		}

		result := c.toolBridge.ExecuteTool(ctx, bridgeCall)
		toolsUsed = append(toolsUsed, toolCall.Function.Name)

		toolResults = append(toolResults, map[string]interface{}{
			"tool":    toolCall.Function.Name,
			"success": result.Success,
			"result":  result.Result,
			"error":   result.Error,
		})
	}

	// Now ask LLM to provide a conversational response about what it did
	followUpPrompt := fmt.Sprintf(`You just executed these tools: %v

Tool results:
%s

Now provide a conversational response following the Interactive Response Protocol:
1. Start with "Anladım!" 
2. Explain what you understood from the user's request
3. Describe what you accomplished with the tools
4. Mention any relevant details or considerations

User's original request was: %s`,
		toolsUsed,
		c.formatToolResultsForLLM(toolResults),
		request.Query)

	// Create follow-up request to LLM
	modelName := strings.TrimPrefix(c.config.Model, "ollama:")
	followUpReq := OllamaRequest{
		Model: modelName,
		Messages: []OllamaMessage{
			{
				Role:    "system",
				Content: c.config.SystemPrompt,
			},
			{
				Role:    "user",
				Content: followUpPrompt,
			},
		},
		Stream: false,
	}

	// Get conversational response from LLM
	reqBody, err := json.Marshal(followUpReq)
	if err != nil {
		// Fallback to basic response if LLM follow-up fails
		return &types.AgentResponse{
			Response:  c.formatBasicToolResponse(toolResults),
			ToolsUsed: toolsUsed,
			Success:   true,
			Context: map[string]interface{}{
				"working_dir": request.WorkingDir,
				"tools_used":  toolsUsed,
			},
		}, nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "http://localhost:11434/api/chat", bytes.NewBuffer(reqBody))
	if err != nil {
		return &types.AgentResponse{
			Response:  c.formatBasicToolResponse(toolResults),
			ToolsUsed: toolsUsed,
			Success:   true,
			Context: map[string]interface{}{
				"working_dir": request.WorkingDir,
				"tools_used":  toolsUsed,
			},
		}, nil
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return &types.AgentResponse{
			Response:  c.formatBasicToolResponse(toolResults),
			ToolsUsed: toolsUsed,
			Success:   true,
			Context: map[string]interface{}{
				"working_dir": request.WorkingDir,
				"tools_used":  toolsUsed,
			},
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &types.AgentResponse{
			Response:  c.formatBasicToolResponse(toolResults),
			ToolsUsed: toolsUsed,
			Success:   true,
			Context: map[string]interface{}{
				"working_dir": request.WorkingDir,
				"tools_used":  toolsUsed,
			},
		}, nil
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return &types.AgentResponse{
			Response:  c.formatBasicToolResponse(toolResults),
			ToolsUsed: toolsUsed,
			Success:   true,
			Context: map[string]interface{}{
				"working_dir": request.WorkingDir,
				"tools_used":  toolsUsed,
			},
		}, nil
	}

	// Return the conversational response from LLM
	response := ollamaResp.Message.Content
	if response == "" {
		response = c.formatBasicToolResponse(toolResults)
	}

	return &types.AgentResponse{
		Response:  response,
		ToolsUsed: toolsUsed,
		Success:   true,
		Context: map[string]interface{}{
			"working_dir": request.WorkingDir,
			"tools_used":  toolsUsed,
		},
	}, nil
}

// formatToolResult formats tool execution results for display
func (c *LLMClient) formatToolResult(toolName string, result interface{}) string {
	switch toolName {
	case "list-directory":
		return c.formatDirectoryResponse(result, "")
	case "read-file":
		return c.formatFileReadResponse(result, "")
	case "search-files":
		return c.formatFileSearchResponse(result, "")
	case "execute-command":
		return c.formatCommandResponse(result)
	default:
		// Generic JSON formatting
		jsonBytes, _ := json.MarshalIndent(result, "", "  ")
		return fmt.Sprintf("🔧 %s result:\n```json\n%s\n```", toolName, string(jsonBytes))
	}
}

// formatCommandResponse formats command execution results
func (c *LLMClient) formatCommandResponse(result interface{}) string {
	data, ok := result.(map[string]interface{})
	if !ok {
		return "Failed to format command response"
	}

	command := data["command"].(string)
	success := data["success"].(bool)
	duration := data["duration"].(string)
	output := data["output"].(string)

	var response strings.Builder

	if success {
		response.WriteString(fmt.Sprintf("✅ Command executed successfully in %s:\n", duration))
		response.WriteString(fmt.Sprintf("```bash\n%s\n```\n\n", command))

		if output != "" {
			response.WriteString("Output:\n")
			response.WriteString("```\n")
			response.WriteString(output)
			response.WriteString("```\n")
		} else {
			response.WriteString("Command completed with no output.\n")
		}
	} else {
		response.WriteString(fmt.Sprintf("❌ Command failed in %s:\n", duration))
		response.WriteString(fmt.Sprintf("```bash\n%s\n```\n\n", command))

		if errorMsg, ok := data["error"].(string); ok {
			response.WriteString(fmt.Sprintf("Error: %s\n", errorMsg))
		}

		if output != "" {
			response.WriteString("Output:\n")
			response.WriteString("```\n")
			response.WriteString(output)
			response.WriteString("```\n")
		}

		if exitCode, ok := data["exit_code"].(int); ok && exitCode != 0 {
			response.WriteString(fmt.Sprintf("Exit code: %d\n", exitCode))
		}
	}

	return response.String()
}

// processOllamaRequest handles legacy requests (fallback to function calling)
func (c *LLMClient) processOllamaRequest(ctx context.Context, prompt string, request *types.AgentRequest) (*types.AgentResponse, error) {
	// Convert to modern function calling approach
	legacyRequest := &types.AgentRequest{
		Query:      prompt,
		Context:    request.Context,
		WorkingDir: request.WorkingDir,
		UseTools:   request.UseTools,
	}
	return c.processOllamaRequestWithFunctionCalling(ctx, legacyRequest)
}

// GetDirectoryContext gathers context about the current directory
func GetDirectoryContext(dirPath string) map[string]string {
	context := make(map[string]string)

	// Get absolute path
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		context["error"] = fmt.Sprintf("Failed to get absolute path: %v", err)
		return context
	}

	context["absolute_path"] = absPath

	// Check if it's a git repository
	gitDir := filepath.Join(absPath, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		context["git_repository"] = "true"

		// Try to get git branch (simplified)
		// In a real implementation, you would use git commands
		context["version_control"] = "git"
	}

	// List directory contents (limited)
	entries, err := os.ReadDir(absPath)
	if err != nil {
		context["read_error"] = fmt.Sprintf("Failed to read directory: %v", err)
		return context
	}

	var files, dirs []string
	fileCount, dirCount := 0, 0

	for i, entry := range entries {
		if i >= 10 { // Limit to first 10 entries
			break
		}

		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
			dirCount++
		} else {
			files = append(files, entry.Name())
			fileCount++
		}
	}

	context["file_count"] = fmt.Sprintf("%d", fileCount)
	context["directory_count"] = fmt.Sprintf("%d", dirCount)

	if len(files) > 0 {
		context["sample_files"] = strings.Join(files, ", ")
	}

	if len(dirs) > 0 {
		context["sample_directories"] = strings.Join(dirs, ", ")
	}

	// Check for common project files
	projectFiles := []string{"package.json", "go.mod", "Cargo.toml", "requirements.txt", "pom.xml", "Makefile", "README.md"}
	var foundProjectFiles []string

	for _, pf := range projectFiles {
		if _, err := os.Stat(filepath.Join(absPath, pf)); err == nil {
			foundProjectFiles = append(foundProjectFiles, pf)
		}
	}

	if len(foundProjectFiles) > 0 {
		context["project_files"] = strings.Join(foundProjectFiles, ", ")
	}

	return context
}

// processOllamaRequestWithStreaming processes a request with streaming support
func (c *LLMClient) processOllamaRequestWithStreaming(ctx context.Context, request *types.AgentRequest, callback types.StreamCallback) error {
	// Get model name from config
	modelName := strings.TrimPrefix(c.config.Model, "ollama:")

	// Build system and user messages
	systemMessage := "You are Jarvis, a highly capable AI assistant with access to system tools. Be helpful, accurate, and secure in all operations."
	if c.config.SystemPrompt != "" {
		systemMessage = c.config.SystemPrompt
	}

	userMessage := c.buildPrompt(request)

	// Get available tools from bridge
	availableTools := c.toolBridge.GetAvailableTools()

	// Convert tools to Ollama format
	var tools []OllamaToolDefinition
	for _, tool := range availableTools {
		tools = append(tools, OllamaToolDefinition{
			Type: "function",
			Function: OllamaFunctionDef{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}

	// Create streaming request
	ollamaReq := OllamaRequest{
		Model: modelName,
		Messages: []OllamaMessage{
			{
				Role:    "system",
				Content: systemMessage,
			},
			{
				Role:    "user",
				Content: userMessage,
			},
		},
		Tools:  tools,
		Stream: true, // Enable streaming
	}

	// Marshal request
	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "http://localhost:11434/api/chat", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Process streaming response
	var fullResponse strings.Builder
	var toolCalls []OllamaToolCall

	scanner := json.NewDecoder(resp.Body)

	for {
		var chunk OllamaResponse
		if err := scanner.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode streaming response: %v", err)
		}

		if chunk.Error != "" {
			return fmt.Errorf("ollama error: %s", chunk.Error)
		}

		// Process message content
		if chunk.Message.Content != "" {
			fullResponse.WriteString(chunk.Message.Content)
			// Send chunk to callback
			if err := callback(chunk.Message.Content, false); err != nil {
				return fmt.Errorf("callback error: %v", err)
			}
		}

		// Collect tool calls
		if len(chunk.Message.ToolCalls) > 0 {
			toolCalls = append(toolCalls, chunk.Message.ToolCalls...)
		}

		// Check if done
		if chunk.Done {
			break
		}
	}

	// Execute tool calls if any
	if len(toolCalls) > 0 {
		callback("\n\n🔧 Executing tools...\n", false)

		for _, toolCall := range toolCalls {
			// Execute tool through bridge
			bridgeCall := bridge.ToolCall{
				Name:       toolCall.Function.Name,
				Parameters: toolCall.Function.Arguments,
			}

			callback(fmt.Sprintf("\n🛠️  %s", toolCall.Function.Name), false)

			result := c.toolBridge.ExecuteTool(ctx, bridgeCall)

			if result.Success {
				callback(" ✅\n", false)
				// You could send tool result back to LLM here for a complete conversation
			} else {
				callback(fmt.Sprintf(" ❌ Error: %s\n", result.Error), false)
			}
		}
	}

	// Send completion signal
	return callback("", true)
}

// formatToolResultsForLLM formats tool results for LLM consumption
func (c *LLMClient) formatToolResultsForLLM(results []map[string]interface{}) string {
	var formatted strings.Builder

	for i, result := range results {
		if i > 0 {
			formatted.WriteString("\n\n")
		}

		tool := result["tool"].(string)
		success := result["success"].(bool)

		formatted.WriteString(fmt.Sprintf("Tool: %s\n", tool))
		formatted.WriteString(fmt.Sprintf("Success: %t\n", success))

		if success {
			if resultData := result["result"]; resultData != nil {
				jsonBytes, _ := json.MarshalIndent(resultData, "", "  ")
				formatted.WriteString(fmt.Sprintf("Result: %s\n", string(jsonBytes)))
			}
		} else {
			if errorMsg := result["error"]; errorMsg != nil {
				formatted.WriteString(fmt.Sprintf("Error: %s\n", errorMsg))
			}
		}
	}

	return formatted.String()
}

// formatBasicToolResponse provides fallback formatting for tool results
func (c *LLMClient) formatBasicToolResponse(results []map[string]interface{}) string {
	var response strings.Builder

	for i, result := range results {
		if i > 0 {
			response.WriteString("\n\n")
		}

		tool := result["tool"].(string)
		success := result["success"].(bool)

		if success {
			resultStr := c.formatToolResult(tool, result["result"])
			response.WriteString(resultStr)
		} else {
			response.WriteString(fmt.Sprintf("❌ Tool %s failed: %s", tool, result["error"]))
		}
	}

	return response.String()
}
