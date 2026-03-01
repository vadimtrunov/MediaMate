package main

import (
	"github.com/spf13/cobra"

	"github.com/vadimtrunov/MediaMate/internal/config"
	mcpserver "github.com/vadimtrunov/MediaMate/internal/mcp"
	"github.com/vadimtrunov/MediaMate/internal/metadata/tmdb"
)

// newMCPServeCmd returns the hidden "mcp-serve" subcommand.
// It starts an MCP server over stdin/stdout, used internally by the
// claudecode LLM provider to expose MediaMate tools to Claude Code.
func newMCPServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "mcp-serve",
		Short:  "Start MCP server over stdio (internal)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(configPath)
			if err != nil {
				return err
			}

			logger := config.SetupLogger(cfg.App.LogLevel)

			backend := initBackend(cfg, logger)
			torrentClient, err := initTorrent(cfg, logger)
			if err != nil {
				return err
			}
			mediaServer := initMediaServer(cfg, logger)

			deps := mcpserver.Deps{
				Backend:     backend,
				Torrent:     torrentClient,
				MediaServer: mediaServer,
			}

			// TMDb is initialized only if configured.
			if cfg.TMDb.APIKey != "" {
				deps.TMDb = tmdb.New(cfg.TMDb.APIKey, logger)
			}

			srv := mcpserver.NewServer(deps, logger)
			return srv.ServeStdio(cmd.Context())
		},
	}
}
