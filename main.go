package main

import (
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
	fmt.Println("  # Start MCP server")
	fmt.Println("  jarvis --model ollama:qwen3")
	fmt.Println()
	fmt.Println("  # Interactive agent mode")
	fmt.Println("  jarvis --model ollama:qwen3 --interactive")
	fmt.Println("  jarvis -m ollama:qwen3 -i --system-prompt \"You are a helpful assistant\"")
	fmt.Println()
	fmt.Println("  # Custom configuration")
	fmt.Println("  jarvis -m ollama:qwen3 --config ~/.ollama-mcp.json -i")
}

// initializeWithArgs initializes the server configuration with command line arguments
func initializeWithArgs(args types.CommandLineArgs) error {
	// Initialize common configuration first
	common.Initialize()
	
	// Load custom config file if specified
	if args.ConfigFile != "" {
		if err := loadCustomConfigFile(args.ConfigFile); err != nil {
			return fmt.Errorf("failed to load config file %s: %v", args.ConfigFile, err)
		}
	}
	
	// Set model configuration from command line arguments
	if args.Model != "" || args.SystemPrompt != "" {
		modelConfig := &types.ModelConfig{
			Model:        args.Model,
			ConfigFile:   args.ConfigFile,
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
