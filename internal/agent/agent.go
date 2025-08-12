package agent

import (
	"bufio"
	"context"
	"fmt"
	"jarvis/internal/llm"
	"jarvis/internal/types"
	"os"
	"strings"
)

// Agent represents the interactive AI agent
type Agent struct {
	llmClient *llm.LLMClient
	workingDir string
}

// NewAgent creates a new agent instance
func NewAgent() (*Agent, error) {
	client, err := llm.NewLLMClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %v", err)
	}
	
	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = "/"
	}
	
	return &Agent{
		llmClient: client,
		workingDir: workingDir,
	}, nil
}

// StartInteractiveSession starts an interactive chat session with the agent
func (a *Agent) StartInteractiveSession() error {
	fmt.Println("🤖 Jarvis AI Agent - Interactive Mode")
	fmt.Println("Type 'exit', 'quit', or 'bye' to end the session")
	fmt.Println("Type 'help' for available commands")
	fmt.Printf("Working directory: %s\n", a.workingDir)
	fmt.Println(strings.Repeat("-", 60))
	
	scanner := bufio.NewScanner(os.Stdin)
	
	for {
		fmt.Print("jarvis >> ")
		
		if !scanner.Scan() {
			break
		}
		
		input := strings.TrimSpace(scanner.Text())
		
		// Handle exit commands
		if isExitCommand(input) {
			fmt.Println("👋 Goodbye!")
			break
		}
		
		// Handle empty input
		if input == "" {
			continue
		}
		
		// Handle built-in commands
		if handled := a.handleBuiltinCommand(input); handled {
			continue
		}
		
		// Process request with LLM
		if err := a.processUserInput(input); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		}
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("input error: %v", err)
	}
	
	return nil
}

// processUserInput processes user input through the LLM
func (a *Agent) processUserInput(input string) error {
	ctx := context.Background()
	
	// Gather directory context
	dirContext := llm.GetDirectoryContext(a.workingDir)
	
	// Create agent request
	request := &types.AgentRequest{
		Query:      input,
		Context:    dirContext,
		WorkingDir: a.workingDir,
		UseTools:   true,
	}
	
	// Process with LLM
	response, err := a.llmClient.ProcessRequest(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to process request: %v", err)
	}
	
	if !response.Success {
		return fmt.Errorf("agent error: %s", response.Error)
	}
	
	// Display response
	fmt.Printf("\n🤖 %s\n\n", response.Response)
	
	// Show tools used if any
	if len(response.ToolsUsed) > 0 {
		fmt.Printf("🔧 Tools used: %s\n", strings.Join(response.ToolsUsed, ", "))
	}
	
	return nil
}

// handleBuiltinCommand handles built-in agent commands
func (a *Agent) handleBuiltinCommand(input string) bool {
	switch strings.ToLower(input) {
	case "help":
		a.showHelp()
		return true
	case "pwd":
		fmt.Printf("Current directory: %s\n", a.workingDir)
		return true
	case "cd":
		a.changeDirectory("")
		return true
	case "context":
		a.showContext()
		return true
	case "clear":
		a.clearScreen()
		return true
	default:
		// Handle cd with directory
		if strings.HasPrefix(strings.ToLower(input), "cd ") {
			dir := strings.TrimSpace(input[3:])
			a.changeDirectory(dir)
			return true
		}
		return false
	}
}

// showHelp displays available commands
func (a *Agent) showHelp() {
	fmt.Println("\n📖 Available Commands:")
	fmt.Println("  help          - Show this help message")
	fmt.Println("  pwd           - Show current working directory")
	fmt.Println("  cd [dir]      - Change working directory")
	fmt.Println("  context       - Show current directory context")
	fmt.Println("  clear         - Clear the screen")
	fmt.Println("  exit/quit/bye - Exit the session")
	fmt.Println("\n💡 Examples:")
	fmt.Println("  Analyze current directory")
	fmt.Println("  List all Go files in this project")
	fmt.Println("  What kind of project is this?")
	fmt.Println("  Explain the main.go file")
	fmt.Println("  Show me the project structure")
	fmt.Println()
}

// changeDirectory changes the working directory
func (a *Agent) changeDirectory(dir string) {
	if dir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("❌ Failed to get home directory: %v\n", err)
			return
		}
		dir = homeDir
	}
	
	// Handle relative paths
	if !strings.HasPrefix(dir, "/") {
		dir = fmt.Sprintf("%s/%s", a.workingDir, dir)
	}
	
	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Printf("❌ Directory does not exist: %s\n", dir)
		return
	}
	
	// Change directory
	if err := os.Chdir(dir); err != nil {
		fmt.Printf("❌ Failed to change directory: %v\n", err)
		return
	}
	
	// Update working directory
	newDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("❌ Failed to get current directory: %v\n", err)
		return
	}
	
	a.workingDir = newDir
	fmt.Printf("📁 Changed to: %s\n", a.workingDir)
}

// showContext displays current directory context
func (a *Agent) showContext() {
	fmt.Println("\n📋 Current Context:")
	context := llm.GetDirectoryContext(a.workingDir)
	
	for key, value := range context {
		fmt.Printf("  %s: %s\n", key, value)
	}
	fmt.Println()
}

// clearScreen clears the terminal screen
func (a *Agent) clearScreen() {
	fmt.Print("\033[2J\033[H")
}

// isExitCommand checks if the input is an exit command
func isExitCommand(input string) bool {
	lower := strings.ToLower(strings.TrimSpace(input))
	return lower == "exit" || lower == "quit" || lower == "bye"
}

// ProcessSingleQuery processes a single query without interactive mode
func (a *Agent) ProcessSingleQuery(query string) error {
	ctx := context.Background()
	
	// Gather directory context
	dirContext := llm.GetDirectoryContext(a.workingDir)
	
	// Create agent request
	request := &types.AgentRequest{
		Query:      query,
		Context:    dirContext,
		WorkingDir: a.workingDir,
		UseTools:   true,
	}
	
	// Process with LLM
	response, err := a.llmClient.ProcessRequest(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to process request: %v", err)
	}
	
	if !response.Success {
		return fmt.Errorf("agent error: %s", response.Error)
	}
	
	// Display response
	fmt.Printf("%s\n", response.Response)
	
	return nil
}