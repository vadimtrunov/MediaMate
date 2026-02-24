package main

import (
	"fmt"
	"log/slog"
	"net/url"

	"github.com/charmbracelet/lipgloss"

	"github.com/vadimtrunov/MediaMate/internal/agent"
	"github.com/vadimtrunov/MediaMate/internal/backend/radarr"
	"github.com/vadimtrunov/MediaMate/internal/config"
	"github.com/vadimtrunov/MediaMate/internal/core"
	"github.com/vadimtrunov/MediaMate/internal/llm/claude"
	"github.com/vadimtrunov/MediaMate/internal/mediaserver/jellyfin"
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

	mediaServer := initMediaServer(cfg, logger)

	return agent.New(llmClient, tmdbClient, backend, torrentClient, mediaServer, logger), nil
}

// initLLM creates an LLM provider client based on the configured provider name.
func initLLM(cfg *config.Config, logger *slog.Logger) (core.LLMProvider, error) {
	switch cfg.LLM.Provider {
	case "claude":
		return claude.New(cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.BaseURL, logger), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s (only 'claude' is supported in this version)", cfg.LLM.Provider)
	}
}

// initBackend creates a Radarr media backend client if configured, or returns nil.
func initBackend(cfg *config.Config, logger *slog.Logger) core.MediaBackend {
	if cfg.Radarr == nil {
		return nil
	}
	backend := radarr.New(
		cfg.Radarr.URL, cfg.Radarr.APIKey,
		cfg.Radarr.QualityProfile, cfg.Radarr.RootFolder,
		logger,
	)
	logger.Info("Radarr backend initialized", slog.String("url", sanitizeURL(cfg.Radarr.URL)))
	return backend
}

// initTorrent creates a qBittorrent client if configured, or returns nil.
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
	logger.Info("qBittorrent client initialized", slog.String("url", sanitizeURL(cfg.QBittorrent.URL)))
	return tc, nil
}

// initMediaServer creates a Jellyfin media server client if configured, or returns nil.
func initMediaServer(cfg *config.Config, logger *slog.Logger) core.MediaServer {
	if cfg.Jellyfin == nil {
		return nil
	}
	ms := jellyfin.New(cfg.Jellyfin.URL, cfg.Jellyfin.APIKey, logger)
	logger.Info("Jellyfin media server initialized", slog.String("url", sanitizeURL(cfg.Jellyfin.URL)))
	return ms
}

// sanitizeURL strips credentials, query params, and fragment from a URL for safe logging.
func sanitizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Scheme == "" {
		return "<redacted>"
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}
