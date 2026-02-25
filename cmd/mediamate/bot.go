package main

import (
	"context"
	"errors"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/vadimtrunov/MediaMate/internal/agent"
	"github.com/vadimtrunov/MediaMate/internal/config"
	"github.com/vadimtrunov/MediaMate/internal/frontend/telegram"
	"github.com/vadimtrunov/MediaMate/internal/notification"
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

// runBot initializes services and starts the Telegram bot with an optional webhook server.
func runBot() error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	if cfg.Telegram == nil {
		return errors.New(
			"telegram configuration is required: set telegram.bot_token in config or MEDIAMATE_TELEGRAM_BOT_TOKEN env var",
		)
	}

	logger := config.SetupLogger(cfg.App.LogLevel)

	bot, err := initTelegramBot(cfg, logger)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Start webhook server in background if configured.
	webhookErrCh := make(chan error, 1)
	if cfg.Webhook != nil {
		webhookSrv := initWebhookServer(cfg, bot, logger)
		go func() {
			err := webhookSrv.Start(ctx)
			if err != nil {
				logger.Error("webhook server stopped", slog.String("error", err.Error()))
			}
			webhookErrCh <- err
		}()
	} else {
		close(webhookErrCh)
	}

	logger.Info("telegram bot starting")
	botErr := bot.Start(ctx)

	// Surface webhook error if bot exited cleanly.
	if webhookErr := <-webhookErrCh; webhookErr != nil {
		if botErr == nil {
			return webhookErr
		}
		logger.Error("webhook server error", slog.String("error", webhookErr.Error()))
	}
	return botErr
}

// initTelegramBot creates and returns a Telegram bot instance.
func initTelegramBot(cfg *config.Config, logger *slog.Logger) (*telegram.Bot, error) {
	factory := func() *agent.Agent {
		a, factoryErr := initServices(cfg, logger)
		if factoryErr != nil {
			logger.Error("failed to create agent for session", "error", factoryErr)
			return nil
		}
		return a
	}

	return telegram.New(
		cfg.Telegram.BotToken,
		cfg.Telegram.AllowedUserIDs,
		factory,
		logger,
	)
}

// initWebhookServer creates a webhook notification server.
func initWebhookServer(cfg *config.Config, bot *telegram.Bot, logger *slog.Logger) *notification.Server {
	mediaServer := initMediaServer(cfg, logger)

	svc := notification.NewService(bot, mediaServer, cfg.Telegram.AllowedUserIDs, logger)
	handler := notification.NewWebhookHandler(svc, cfg.Webhook.Secret, logger)

	return notification.NewServer(cfg.Webhook.Port, handler, logger)
}
