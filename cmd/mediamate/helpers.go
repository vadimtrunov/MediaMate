package main

import (
	"fmt"
	"log/slog"

	"github.com/charmbracelet/lipgloss"

	"github.com/vadimtrunov/MediaMate/internal/agent"
	"github.com/vadimtrunov/MediaMate/internal/backend/radarr"
	"github.com/vadimtrunov/MediaMate/internal/config"
	"github.com/vadimtrunov/MediaMate/internal/core"
	"github.com/vadimtrunov/MediaMate/internal/llm/claude"
	"github.com/vadimtrunov/MediaMate/internal/metadata/tmdb"
	"github.com/vadimtrunov/MediaMate/internal/torrent/qbittorrent"
)

// Lipgloss styles used across commands.
var (
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // red
	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	styleInfo    = lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // blue
	styleDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // gray

	styleUser      = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true) // cyan bold
	styleAssistant = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))            // white

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("5")).
			MarginBottom(1)
)

// loadConfig loads and validates the configuration file.
func loadConfig(path string) (*config.Config, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, fmt.Errorf("load configuration: %w", err)
	}
	return cfg, nil
}

// initServices creates all backend services and returns an Agent.
func initServices(cfg *config.Config, logger *slog.Logger) (*agent.Agent, error) {
	llmClient, err := initLLM(cfg, logger)
	if err != nil {
		return nil, err
	}
	tmdbClient := tmdb.New(cfg.TMDb.APIKey, logger)
	backend := initBackend(cfg, logger)

	torrentClient, err := initTorrent(cfg, logger)
	if err != nil {
		return nil, err
	}

	return agent.New(llmClient, tmdbClient, backend, torrentClient, logger), nil
}

func initLLM(cfg *config.Config, logger *slog.Logger) (core.LLMProvider, error) {
	switch cfg.LLM.Provider {
	case "claude":
		return claude.New(cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.BaseURL, logger), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s (only 'claude' is supported in this version)", cfg.LLM.Provider)
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
