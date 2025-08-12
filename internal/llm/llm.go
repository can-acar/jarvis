package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	config *types.ModelConfig
}

// OllamaRequest represents a request to Ollama API
type OllamaRequest struct {
	Model    string `json:"model"`
	Prompt   string `json:"prompt"`
	Stream   bool   `json:"stream"`
	System   string `json:"system,omitempty"`
	Context  []int  `json:"context,omitempty"`
}

// OllamaResponse represents a response from Ollama API
type OllamaResponse struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	Context   []int  `json:"context,omitempty"`
	Error     string `json:"error,omitempty"`
}

// NewLLMClient creates a new LLM client with the configured model
func NewLLMClient() (*LLMClient, error) {
	cfg := common.Get()
	if cfg.ModelConfig == nil {
		return nil, fmt.Errorf("no model configuration found")
	}
	
	return &LLMClient{
		config: cfg.ModelConfig,
	}, nil
}

// ProcessRequest processes a user request using the configured LLM
func (c *LLMClient) ProcessRequest(ctx context.Context, request *types.AgentRequest) (*types.AgentResponse, error) {
	// Build the prompt with system context
	prompt := c.buildPrompt(request)
	
	// Determine model type and call appropriate handler
	if strings.HasPrefix(c.config.Model, "ollama:") {
		return c.processOllamaRequest(ctx, prompt, request)
	}
	
	return nil, fmt.Errorf("unsupported model type: %s", c.config.Model)
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

// processOllamaRequest handles requests to Ollama models
func (c *LLMClient) processOllamaRequest(ctx context.Context, prompt string, request *types.AgentRequest) (*types.AgentResponse, error) {
	// Extract model name (remove "ollama:" prefix)
	modelName := strings.TrimPrefix(c.config.Model, "ollama:")
	
	// Prepare Ollama request
	ollamaReq := OllamaRequest{
		Model:  modelName,
		Prompt: prompt,
		Stream: false,
	}
	
	if c.config.SystemPrompt != "" {
		ollamaReq.System = c.config.SystemPrompt
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
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "http://localhost:11434/api/generate", bytes.NewBuffer(reqBody))
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
	
	// Check for errors in response
	if ollamaResp.Error != "" {
		return &types.AgentResponse{
			Success: false,
			Error:   fmt.Sprintf("Ollama error: %s", ollamaResp.Error),
		}, nil
	}
	
	// Return successful response
	return &types.AgentResponse{
		Response: ollamaResp.Response,
		Success:  true,
		Context: map[string]interface{}{
			"model": ollamaResp.Model,
			"working_dir": request.WorkingDir,
		},
	}, nil
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