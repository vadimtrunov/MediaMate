package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/vadimtrunov/MediaMate/internal/agent"
	"github.com/vadimtrunov/MediaMate/internal/core"
)

// AgentFactory creates a new Agent instance for each user session.
type AgentFactory func() *agent.Agent

// Bot is the Telegram frontend for MediaMate.
// It implements the core.Frontend interface.
type Bot struct {
	api          *tgbotapi.BotAPI
	sessions     *sessionManager
	agentFactory AgentFactory
	logger       *slog.Logger
}

// compile-time checks.
var (
	_ core.Frontend         = (*Bot)(nil)
	_ core.ProgressNotifier = (*Bot)(nil)
)

// New creates a new Telegram Bot.
func New(token string, allowedUserIDs []int64, factory AgentFactory, logger *slog.Logger) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &Bot{
		api:          api,
		sessions:     newSessionManager(allowedUserIDs),
		agentFactory: factory,
		logger:       logger,
	}, nil
}

// Name returns the frontend name.
func (b *Bot) Name() string { return "telegram" }

// Start starts the long-polling loop. It blocks until ctx is canceled.
func (b *Bot) Start(ctx context.Context) error {
	b.logger.Info("telegram bot started",
		slog.String("username", b.api.Self.UserName),
	)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			b.api.StopReceivingUpdates()
			b.logger.Info("telegram bot stopped")
			return nil

		case update, ok := <-updates:
			if !ok {
				return nil
			}
			go b.handleUpdate(ctx, update)
		}
	}
}

// Stop stops the bot (no-op, Start returns when ctx is canceled).
func (b *Bot) Stop(_ context.Context) error {
	return nil
}

// SendMessage sends a text message to a Telegram user.
func (b *Bot) SendMessage(_ context.Context, userID, message string) error {
	var chatID int64
	if _, err := fmt.Sscanf(userID, "%d", &chatID); err != nil {
		return fmt.Errorf("invalid user ID %q: %w", userID, err)
	}

	msg := tgbotapi.NewMessage(chatID, message)
	_, err := b.api.Send(msg)
	return err
}

// SendProgressMessage sends a new plain-text message and returns its ID.
func (b *Bot) SendProgressMessage(_ context.Context, chatID int64, text string) (int, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	sent, err := b.api.Send(msg)
	if err != nil {
		return 0, fmt.Errorf("send progress message: %w", err)
	}
	return sent.MessageID, nil
}

// EditProgressMessage updates an existing message's text.
func (b *Bot) EditProgressMessage(_ context.Context, chatID int64, messageID int, text string) error {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	_, err := b.api.Send(edit)
	if err != nil {
		if strings.Contains(err.Error(), "message is not modified") {
			return nil
		}
		return fmt.Errorf("edit progress message: %w", err)
	}
	return nil
}

// handleUpdate dispatches an incoming Telegram update.
func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	switch {
	case update.CallbackQuery != nil:
		b.handleCallback(ctx, update.CallbackQuery)
	case update.Message != nil:
		b.handleMessage(ctx, update.Message)
	}
}
