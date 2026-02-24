package main

import (
	"context"
	"errors"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/vadimtrunov/MediaMate/internal/agent"
	"github.com/vadimtrunov/MediaMate/internal/config"
	"github.com/vadimtrunov/MediaMate/internal/frontend/telegram"
)

// newBotCmd returns the "bot" subcommand for running the Telegram bot.
func newBotCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bot",
		Short: "Start the Telegram bot",
		Long:  "Start the MediaMate Telegram bot for interactive conversation via Telegram.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runBot()
		},
	}
}

// runBot initializes services and starts the Telegram bot.
func runBot() error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	if cfg.Telegram == nil {
		return errors.New("telegram configuration is required: set telegram.bot_token in config or MEDIAMATE_TELEGRAM_BOT_TOKEN env var")
	}

	logger := config.SetupLogger(cfg.App.LogLevel)

	// AgentFactory creates a fresh agent for each user session.
	factory := func() *agent.Agent {
		a, factoryErr := initServices(cfg, logger)
		if factoryErr != nil {
			logger.Error("failed to create agent for session", "error", factoryErr)
			return nil
		}
		return a
	}

	bot, err := telegram.New(
		cfg.Telegram.BotToken,
		cfg.Telegram.AllowedUserIDs,
		factory,
		logger,
	)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger.Info("telegram bot starting")
	return bot.Start(ctx)
}
