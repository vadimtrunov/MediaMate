package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/vadimtrunov/MediaMate/internal/config"
)

const version = "0.0.1"

func main() {
	// Parse command line flags
	configPath := flag.String("config", "configs/mediamate.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version and exit")
	validateConfig := flag.Bool("validate", false, "Validate configuration and exit")
	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("MediaMate v%s\n", version)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	logger := config.SetupLogger(cfg.App.LogLevel)
	logger.Info("MediaMate starting",
		slog.String("version", version),
		slog.String("config", *configPath),
	)

	// Validate config if requested
	if *validateConfig {
		logger.Info("Configuration is valid")
		return
	}

	// Log configuration (without sensitive data)
	logger.Debug("Configuration loaded",
		slog.String("llm_provider", cfg.LLM.Provider),
		slog.String("llm_model", cfg.LLM.Model),
		slog.Bool("radarr_configured", cfg.Radarr != nil),
		slog.Bool("sonarr_configured", cfg.Sonarr != nil),
		slog.Bool("jellyfin_configured", cfg.Jellyfin != nil),
		slog.Bool("telegram_configured", cfg.Telegram != nil),
	)

	logger.Info("MediaMate initialized successfully")
	// TODO: Start the application
}
