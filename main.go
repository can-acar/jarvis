package main

import (
	"encoding/json"
	"fmt"
	"jarvis/internal/agent"
	"jarvis/internal/common"
	"jarvis/internal/config"
	"jarvis/internal/terminal"
	"jarvis/internal/textedit"
	"jarvis/internal/types"
	"log"
	"os"
	"path/filepath"
	"strings"

	fetching "jarvis/internal/fetch"
	"jarvis/internal/filesystem"

	"github.com/mark3labs/mcp-go/server"
	flag "github.com/spf13/pflag"
)

func main() {
	// Parse command line arguments
	args := parseCommandLineArgs()

	// Handle help and version flags
	if args.Help {
		printUsage()
		return
	}

	if args.Version {
		fmt.Println("Jarvis MCP Server v1.0.0")
		return
	}

	// Initialize configuration with command line arguments
	if err := initializeWithArgs(args); err != nil {
		log.Fatalf("Failed to initialize configuration: %v", err)
	}

	// Check if we should run in interactive mode
	if args.Interactive {
		if err := runInteractiveMode(); err != nil {
			log.Fatalf("Interactive mode failed: %v", err)
		}
		return
	}

	s := server.NewMCPServer(
		"jarvis",                          // Sunucu adı
		"1.0.0",                           // Versiyon
		server.WithToolCapabilities(true), // Tool desteği
		server.WithResourceCapabilities(true, true), // Resource desteği
		server.WithPromptCapabilities(true),         // Prompt desteği
		server.WithRecovery(),                       // Hata kurtarma
		server.WithLogging(),
	)

	config.RegisterConfigTools(s)         // Yapılandırma araçlarını kaydet
	terminal.RegisterTerminalTools(s)     // Terminal araçlarını kaydet
	filesystem.RegisterFilesystemTools(s) // Dosya sistemi araçlarını kaydet
	textedit.RegisterTextEditingTools(s)  // Metin düzenleme araçlarını kaydet
	fetching.RegisterFetchTools(s)        // Fetching araçlarını kaydet
	logStartupInfo()
	// Sunucuyu stdio üzerinden başlat
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Sunucu hatası: %v\n", err)
	}
}

// parseCommandLineArgs parses and validates command line arguments
func parseCommandLineArgs() types.CommandLineArgs {
	var args types.CommandLineArgs

	flag.StringVarP(&args.Model, "model", "m", "", "LLM model to use (e.g., ollama:qwen3)")
	flag.StringVarP(&args.ConfigFile, "config", "c", "", "Path to custom config file")
	flag.StringVar(&args.SystemPrompt, "system-prompt", "", "System prompt for the model")
	flag.BoolVarP(&args.Interactive, "interactive", "i", false, "Run in interactive agent mode")
	flag.BoolVarP(&args.Help, "help", "h", false, "Show help message")
	flag.BoolVarP(&args.Version, "version", "v", false, "Show version information")

	flag.Parse()

	return args
}

// printUsage displays usage information
func printUsage() {
	fmt.Println("Jarvis MCP Server - AI Assistant with System Tools")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  jarvis [OPTIONS]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -m, --model MODEL        LLM model to use (e.g., ollama:qwen3)")
	fmt.Println("  -c, --config FILE        Path to custom config file")
	fmt.Println("      --system-prompt TEXT System prompt for the model")
	fmt.Println("  -i, --interactive        Run in interactive agent mode")
	fmt.Println("  -h, --help               Show this help message")
	fmt.Println("  -v, --version            Show version information")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Start MCP server with default config (jarvis-config.json)")
	fmt.Println("  jarvis")
	fmt.Println()
	fmt.Println("  # Start MCP server with specific model")
	fmt.Println("  jarvis --model ollama:qwen3")
	fmt.Println()
	fmt.Println("  # Interactive agent mode with default config")
	fmt.Println("  jarvis --interactive")
	fmt.Println()
	fmt.Println("  # Custom configuration")
	fmt.Println("  jarvis -m ollama:qwen3 --config ~/.ollama-mcp.json -i")
}

// initializeWithArgs initializes the server configuration with command line arguments
func initializeWithArgs(args types.CommandLineArgs) error {
	// Initialize common configuration first
	common.Initialize()

	// Handle config file logic
	configFile := args.ConfigFile
	if configFile == "" {
		// Use default config file if not specified
		configFile = "jarvis-config.json"
		args.ConfigFile = configFile

		// If model is specified but no config, auto-generate config file name
		if args.Model != "" {
			modelName := strings.ReplaceAll(args.Model, ":", "-")
			configFile = fmt.Sprintf("%s-config.json", modelName)
			args.ConfigFile = configFile
		}
	}

	// Load or create custom config file
	if configFile != "" {
		if err := loadOrCreateCustomConfigFile(configFile, args); err != nil {
			return fmt.Errorf("failed to load/create config file %s: %v", configFile, err)
		}
	}

	// Set model configuration from command line arguments
	if args.Model != "" || args.SystemPrompt != "" {
		modelConfig := &types.ModelConfig{
			Model:        args.Model,
			ConfigFile:   configFile,
			SystemPrompt: args.SystemPrompt,
		}

		if err := common.SetModelConfig(modelConfig); err != nil {
			return fmt.Errorf("failed to set model configuration: %v", err)
		}
	}

	return nil
}

// loadCustomConfigFile loads configuration from a custom JSON file
func loadCustomConfigFile(configPath string) error {
	// Expand home directory if path starts with ~
	if len(configPath) > 0 && configPath[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %v", err)
		}
		if len(configPath) == 1 {
			configPath = homeDir
		} else if configPath[1] == filepath.Separator {
			configPath = filepath.Join(homeDir, configPath[2:])
		}
	}

	return common.LoadFromCustomFile(configPath)
}

// loadOrCreateCustomConfigFile loads or creates a custom configuration file
func loadOrCreateCustomConfigFile(configPath string, args types.CommandLineArgs) error {
	// Expand home directory if path starts with ~
	if len(configPath) > 0 && configPath[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %v", err)
		}
		if len(configPath) == 1 {
			configPath = homeDir
		} else if configPath[1] == filepath.Separator {
			configPath = filepath.Join(homeDir, configPath[2:])
		}
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Config file doesn't exist, create a default one
		log.Printf("Config file %s does not exist, creating default configuration", configPath)

		if err := createDefaultConfigFile(configPath, args); err != nil {
			return fmt.Errorf("failed to create default config file: %v", err)
		}

		log.Printf("Created default config file: %s", configPath)
	}

	// Load the config file
	return common.LoadFromCustomFile(configPath)
}

// createDefaultConfigFile creates a default configuration file
func createDefaultConfigFile(configPath string, args types.CommandLineArgs) error {
	// Create default configuration
	defaultConfig := &types.ServerConfig{
		BlockedCommands:    []string{"rm -rf", "shutdown", "reboot", "dd", "mkfs", "format"},
		DefaultShell:       "bash",
		AllowedDirectories: []string{"/tmp", "/var/log", "/home", "/Users"},
		FileReadLineLimit:  2000,
		FileWriteLineLimit: 100,
		TelemetryEnabled:   false,
		ModelConfig: &types.ModelConfig{
			Model:        args.Model,
			SystemPrompt: getDefaultSystemPrompt(),
		},
		MCP: []types.MCPServerConfig{
			{
				Name:    "example-filesystem",
				Command: "filesystem-mcp-server",
				Args:    []string{"--stdio"},
				Environment: map[string]string{
					"PATH": os.Getenv("PATH"),
				},
				Enabled: false,
			},
			{
				Name:    "example-database",
				Command: "database-mcp-server",
				Args:    []string{"--stdio", "--db-path", "/tmp/example.db"},
				Environment: map[string]string{
					"PATH": os.Getenv("PATH"),
				},
				Enabled: false,
			},
		},
	}

	// Set system prompt if provided
	if args.SystemPrompt != "" {
		defaultConfig.ModelConfig.SystemPrompt = args.SystemPrompt
	}

	// Convert to JSON with pretty formatting
	configJSON, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// Write config file
	if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// getDefaultSystemPrompt returns the default system prompt
func getDefaultSystemPrompt() string {
	return `You are Jarvis, a highly capable AI assistant with access to system tools. Be helpful, accurate, and secure in all operations.

## CRITICAL: ALWAYS USE TOOLS FOR ACTIONS
- When user asks to create, build, run, or execute something: USE THE AVAILABLE TOOLS
- NEVER just list steps or provide instructions - ACTUALLY EXECUTE THE COMMANDS
- Use execute-command tool for any bash/shell commands
- Use file creation tools for creating files
- Use directory tools for folder operations

## Interactive Response Protocol:
- ALWAYS start your response with "Anladım!" (I understand!) when user gives a command
- Then explain your understanding of what the user wants to accomplish
- Share your thoughts about the best approach to take
- Mention any potential considerations or alternatives
- MOST IMPORTANT: Then proceed with executing the tools - DO NOT just list what should be done

## Execution Rules:
- If user says "create project", "build app", "make folder" etc. -> USE TOOLS IMMEDIATELY
- Break complex tasks into multiple tool calls
- Execute commands step by step
- Show progress as you go

## Response Format Example:
"Anladım! Sen /Users/r00t/Projects altında .NET Core CQRS projesi oluşturmak istiyorsun. Şimdi adım adım gerçekleştiriyorum:

[execute-command: mkdir -p /Users/r00t/Projects/PragmaticApi]
[execute-command: cd /Users/r00t/Projects/PragmaticApi && dotnet new webapi -n PragmaticApi]
[execute-command: cd /Users/r00t/Projects/PragmaticApi && dotnet new sln]

Bu şekilde projen hazırlanıyor..."

## Fail-Safe and Recovery Protocol:
- If a tool/command fails, ALWAYS try alternative approaches before giving up
- Suggest multiple solutions when the first approach doesn't work
- Use progressive fallback strategies (e.g., if mkdir fails, try with different permissions or alternative paths)
- Never stop at the first error - explore different methods to achieve the user's goal

## User Confirmation Protocol:
- For DESTRUCTIVE operations (delete, remove, format, overwrite), ALWAYS ask for explicit user confirmation first
- For SYSTEM-CRITICAL operations (shutdown, reboot, user management), ALWAYS ask for confirmation
- For operations affecting MULTIPLE FILES or LARGE DIRECTORIES, ask for confirmation
- Use clear, specific descriptions of what will be done before asking for confirmation

## Confirmation Examples:
❗ "I'm about to delete 15 files in /tmp/project. This action cannot be undone. Do you want to proceed? (yes/no)"
❗ "I will create a new user account 'testuser' with sudo privileges. Confirm? (yes/no)"
❗ "This will modify 8 configuration files. Review and confirm? (yes/no)"

## Alternative Strategy Examples:
- If 'rm file.txt' fails → try 'rm -f file.txt' → try 'sudo rm file.txt' → try moving to trash
- If 'mkdir /restricted/path' fails → try 'mkdir -p /restricted/path' → try creating in alternative location
- If direct file edit fails → try backup and edit → try temporary file approach

Always be proactive in finding solutions and transparent about risks.`
}

// runInteractiveMode starts the interactive agent mode
func runInteractiveMode() error {
	// Check if model is configured
	cfg := common.Get()
	if cfg.ModelConfig == nil || cfg.ModelConfig.Model == "" {
		return fmt.Errorf("no model configured. Use --model to specify a model (e.g., --model ollama:qwen3)")
	}

	// Create and start the agent
	agent, err := agent.NewAgent()
	if err != nil {
		return fmt.Errorf("failed to create agent: %v", err)
	}

	return agent.StartInteractiveSession()
}

// logStartupInfo logs server startup information
func logStartupInfo() {
	cfg := common.Get()

	log.Printf("Server Configuration:")
	log.Printf("  Default Shell: %s", cfg.DefaultShell)
	log.Printf("  Allowed Directories: %v", cfg.AllowedDirectories)
	log.Printf("  File Read Line Limit: %d", cfg.FileReadLineLimit)
	log.Printf("  File Write Line Limit: %d", cfg.FileWriteLineLimit)
	log.Printf("  Blocked Commands: %v", cfg.BlockedCommands)
	log.Printf("  Telemetry: %t", cfg.TelemetryEnabled)

	if cfg.ModelConfig != nil {
		log.Printf("  Model Configuration:")
		log.Printf("    Model: %s", cfg.ModelConfig.Model)
		if cfg.ModelConfig.ConfigFile != "" {
			log.Printf("    Config File: %s", cfg.ModelConfig.ConfigFile)
		}
		if cfg.ModelConfig.SystemPrompt != "" {
			log.Printf("    System Prompt: %s", cfg.ModelConfig.SystemPrompt)
		}
	}

	log.Println("Jarvis MCP Server initialized successfully")
}
