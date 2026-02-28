package main

import (
	"context"
	"errors"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

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

	// Start webhook server and progress tracker in background if configured.
	webhookErrCh := startWebhookIfConfigured(ctx, cfg, bot, logger)

	logger.Info("telegram bot starting")
	botErr := bot.Start(ctx)
	cancel() // Unblock webhook server goroutine waiting on ctx.

	// Surface webhook error if bot exited cleanly.
	if webhookErr := <-webhookErrCh; webhookErr != nil && !errors.Is(webhookErr, context.Canceled) {
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

// startWebhookIfConfigured launches the webhook server and progress tracker in the background.
// Returns a channel that will receive the webhook server error (or be closed if webhooks are disabled).
func startWebhookIfConfigured(
	ctx context.Context, cfg *config.Config, bot *telegram.Bot, logger *slog.Logger,
) <-chan error {
	errCh := make(chan error, 1)
	if cfg.Webhook == nil {
		close(errCh)
		return errCh
	}

	srv, tracker := initWebhookServer(cfg, bot, logger)
	go func() {
		err := srv.Start(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("webhook server stopped", slog.String("error", err.Error()))
		}
		errCh <- err
	}()

	if tracker != nil {
		go func() {
			if err := tracker.Start(ctx); err != nil && ctx.Err() == nil {
				logger.Error("progress tracker stopped", slog.String("error", err.Error()))
			}
		}()
	}
	return errCh
}

// initWebhookServer creates a webhook notification server and an optional progress tracker.
func initWebhookServer(
	cfg *config.Config, bot *telegram.Bot, logger *slog.Logger,
) (*notification.Server, *notification.Tracker) {
	mediaServer := initMediaServer(cfg, logger)
	svc := notification.NewService(bot, mediaServer, cfg.Telegram.AllowedUserIDs, logger)
	handler := notification.NewWebhookHandler(svc, cfg.Webhook.Secret, logger)
	srv := notification.NewServer(cfg.Webhook.Port, handler, logger)

	var tracker *notification.Tracker
	if cfg.Webhook.Progress.Enabled && cfg.QBittorrent != nil {
		torrentClient, err := initTorrent(cfg, logger)
		if err != nil {
			logger.Error("failed to init torrent client for progress tracking",
				slog.String("error", err.Error()))
			return srv, nil
		}
		interval := time.Duration(cfg.Webhook.Progress.Interval) * time.Second
		tracker = notification.NewTracker(
			torrentClient, bot, cfg.Telegram.AllowedUserIDs, interval, logger,
		)
		svc.SetTracker(tracker)
	}
	return srv, tracker
}
