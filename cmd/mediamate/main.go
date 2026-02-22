package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/vadimtrunov/MediaMate/internal/agent"
	"github.com/vadimtrunov/MediaMate/internal/backend/radarr"
	"github.com/vadimtrunov/MediaMate/internal/config"
	"github.com/vadimtrunov/MediaMate/internal/core"
	"github.com/vadimtrunov/MediaMate/internal/llm/claude"
	"github.com/vadimtrunov/MediaMate/internal/metadata/tmdb"
	"github.com/vadimtrunov/MediaMate/internal/torrent/qbittorrent"
)

const version = "0.1.0"

func main() {
	configPath := flag.String("config", "configs/mediamate.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version and exit")
	validateConfig := flag.Bool("validate", false, "Validate configuration and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("MediaMate v%s\n", version)
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	logger := config.SetupLogger(cfg.App.LogLevel)
	logger.Info("MediaMate starting",
		slog.String("version", version),
		slog.String("config", *configPath),
	)

	if *validateConfig {
		logger.Info("Configuration is valid")
		return
	}

	if err := run(cfg, logger); err != nil {
		logger.Error("fatal error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(cfg *config.Config, logger *slog.Logger) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	llmClient := initLLM(cfg, logger)
	tmdbClient := tmdb.New(cfg.TMDb.APIKey, logger)
	backend := initBackend(cfg, logger)

	torrentClient, err := initTorrent(cfg, logger)
	if err != nil {
		return err
	}

	a := agent.New(llmClient, tmdbClient, backend, torrentClient, logger)

	logger.Info("MediaMate ready")
	fmt.Println("MediaMate ready. Type your message (or 'quit' to exit, '/reset' to clear history).")
	runInteractiveLoop(ctx, a)
	return nil
}

func initLLM(cfg *config.Config, logger *slog.Logger) core.LLMProvider {
	switch cfg.LLM.Provider {
	case "claude":
		return claude.New(cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.BaseURL, logger)
	default:
		logger.Error("unsupported LLM provider (only 'claude' is supported in this version)",
			slog.String("provider", cfg.LLM.Provider),
		)
		os.Exit(1)
		return nil
	}
}

func initBackend(cfg *config.Config, logger *slog.Logger) core.MediaBackend {
	if cfg.Radarr == nil {
		return nil
	}
	backend := radarr.New(
		cfg.Radarr.URL, cfg.Radarr.APIKey,
		cfg.Radarr.QualityProfile, cfg.Radarr.RootFolder,
		logger,
	)
	logger.Info("Radarr backend initialized", slog.String("url", cfg.Radarr.URL))
	return backend
}

func initTorrent(cfg *config.Config, logger *slog.Logger) (core.TorrentClient, error) {
	if cfg.QBittorrent == nil {
		return nil, nil
	}
	tc, err := qbittorrent.New(
		cfg.QBittorrent.URL, cfg.QBittorrent.Username, cfg.QBittorrent.Password,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("create qBittorrent client: %w", err)
	}
	logger.Info("qBittorrent client initialized", slog.String("url", cfg.QBittorrent.URL))
	return tc, nil
}

func runInteractiveLoop(ctx context.Context, a *agent.Agent) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("\n> ")
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			fmt.Print("> ")
			continue
		}
		if input == "quit" || input == "exit" {
			fmt.Println("Goodbye!")
			return
		}
		if input == "/reset" {
			a.Reset()
			fmt.Println("Conversation reset.")
			fmt.Print("\n> ")
			continue
		}

		response, err := a.HandleMessage(ctx, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			fmt.Printf("\n%s\n", response)
		}
		fmt.Print("\n> ")
	}
}
