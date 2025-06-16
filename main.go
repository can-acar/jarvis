package main

import (
	"fmt"
	"jarvis/internal/common"
	"jarvis/internal/config"
	"jarvis/internal/terminal"
	"jarvis/internal/textedit"
	"log"

	fetching "jarvis/internal/fetch"
	"jarvis/internal/filesystem"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
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

	log.Println("Jarvis MCP Server initialized successfully")
}
