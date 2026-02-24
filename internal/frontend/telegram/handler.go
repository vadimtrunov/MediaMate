package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/vadimtrunov/MediaMate/internal/metadata/tmdb"
)

const (
	unauthorizedMsg = "Sorry, you are not authorized to use this bot."
	errorMsg        = "An error occurred while processing your request. Please try again."
	resetMsg        = "Conversation reset. Send a message to start over."

	callbackPrefix = "sel:" // prefix for selection callback data

	minLineLen     = 3  // minimum line length for numbered list detection
	maxButtonLabel = 30 // max characters in inline keyboard button label
)

// handleMessage processes an incoming text message.
func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	userID := msg.From.ID
	chatID := msg.Chat.ID

	b.logger.Debug("received message",
		slog.Int64("user_id", userID),
	)

	if !b.sessions.isAllowed(userID) {
		b.sendText(chatID, unauthorizedMsg)
		return
	}

	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return
	}

	// Handle slash commands.
	if text == "/start" {
		b.sendText(chatID, "Welcome to MediaMate! Ask me about movies - search, recommend, download.")
		return
	}
	if text == "/reset" {
		b.sessions.reset(userID)
		b.sendText(chatID, resetMsg)
		return
	}

	// Show typing indicator.
	typing := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	b.api.Send(typing) //nolint:errcheck // best-effort typing indicator

	// Get or create per-user session.
	a := b.sessions.getOrCreate(userID, b.agentFactory)
	if a == nil {
		b.logger.Error("failed to create agent session", slog.Int64("user_id", userID))
		b.sendText(chatID, errorMsg)
		return
	}

	response, err := a.HandleMessage(ctx, text)
	if err != nil {
		b.logger.Error("agent error",
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		b.sendText(chatID, errorMsg)
		return
	}

	b.sendResponse(chatID, response)
}

// handleCallback processes inline keyboard callback queries.
func (b *Bot) handleCallback(ctx context.Context, cq *tgbotapi.CallbackQuery) {
	userID := cq.From.ID
	chatID := cq.Message.Chat.ID

	b.logger.Debug("received callback",
		slog.Int64("user_id", userID),
		slog.String("data", cq.Data),
	)

	// Acknowledge the callback immediately.
	callback := tgbotapi.NewCallback(cq.ID, "")
	b.api.Send(callback) //nolint:errcheck // best-effort ack

	if !b.sessions.isAllowed(userID) {
		return
	}

	// Parse selection callbacks like "sel:1" → user chose option 1.
	if !strings.HasPrefix(cq.Data, callbackPrefix) {
		return
	}
	choice := strings.TrimPrefix(cq.Data, callbackPrefix)

	// Show typing indicator.
	typing := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	b.api.Send(typing) //nolint:errcheck

	// Remove the inline keyboard from the original message.
	removeKB := tgbotapi.NewEditMessageReplyMarkup(chatID, cq.Message.MessageID, tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{},
	})
	b.api.Send(removeKB) //nolint:errcheck

	a := b.sessions.getOrCreate(userID, b.agentFactory)
	if a == nil {
		b.logger.Error("failed to create agent session", slog.Int64("user_id", userID))
		b.sendText(chatID, errorMsg)
		return
	}

	response, err := a.HandleMessage(ctx, choice)
	if err != nil {
		b.logger.Error("agent error on callback",
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		b.sendText(chatID, errorMsg)
		return
	}

	b.sendResponse(chatID, response)
}

// sendResponse sends the agent's response, optionally with posters and keyboards.
func (b *Bot) sendResponse(chatID int64, response string) {
	// Try to detect and send movie posters embedded in the response.
	b.sendPosterIfAvailable(chatID, response)

	// Check if the response contains a numbered list (potential selection).
	if kb := b.buildSelectionKeyboard(response); kb != nil {
		msg := tgbotapi.NewMessage(chatID, EscapeMdV2(response))
		msg.ParseMode = tgbotapi.ModeMarkdownV2
		msg.ReplyMarkup = kb
		if _, err := b.api.Send(msg); err != nil {
			b.logger.Warn("failed to send markdown, retrying plain",
				slog.String("error", err.Error()),
			)
			b.sendPlainWithKeyboard(chatID, response, kb)
		}
		return
	}

	// Plain text response with MarkdownV2.
	msg := tgbotapi.NewMessage(chatID, EscapeMdV2(response))
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	if _, err := b.api.Send(msg); err != nil {
		b.logger.Warn("failed to send markdown, retrying plain",
			slog.String("error", err.Error()),
		)
		b.sendText(chatID, response)
	}
}

// sendText sends a plain text message (no parse mode).
func (b *Bot) sendText(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("failed to send message",
			slog.Int64("chat_id", chatID),
			slog.String("error", err.Error()),
		)
	}
}

// sendPlainWithKeyboard sends a plain-text message with inline keyboard.
func (b *Bot) sendPlainWithKeyboard(chatID int64, text string, kb *tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = kb
	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("failed to send message with keyboard",
			slog.Int64("chat_id", chatID),
			slog.String("error", err.Error()),
		)
	}
}

// buildSelectionKeyboard detects numbered items in a response and builds
// inline keyboard buttons. Returns nil if no numbered list is found.
// Looks for patterns like "1. Title", "10. Title", or "1) Title".
func (b *Bot) buildSelectionKeyboard(response string) *tgbotapi.InlineKeyboardMarkup {
	lines := strings.Split(response, "\n")
	var buttons []tgbotapi.InlineKeyboardButton

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) < minLineLen {
			continue
		}

		// Extract leading digits.
		i := 0
		for i < len(line) && line[i] >= '0' && line[i] <= '9' {
			i++
		}
		if i == 0 || i >= len(line)-1 {
			continue
		}

		num := line[:i]
		// Must be followed by ". " or ") ".
		if (line[i] == '.' || line[i] == ')') && i+1 < len(line) && line[i+1] == ' ' {
			label := line[i+2:]
			if len(label) > maxButtonLabel {
				label = label[:maxButtonLabel] + "…"
			}
			buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s. %s", num, label),
				callbackPrefix+num,
			))
		}
	}

	if len(buttons) < 2 {
		return nil
	}

	// Arrange buttons in rows of 1 each (cleaner on mobile).
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, btn := range buttons {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return &kb
}

// sendPosterIfAvailable tries to detect movie poster URLs in the response
// by checking the TMDb client for recently searched movies.
// This is a best-effort feature — if no poster is found, nothing is sent.
func (b *Bot) sendPosterIfAvailable(chatID int64, response string) {
	// Look for TMDb poster URL patterns in the response.
	// The agent may include poster URLs from tool results.
	const posterMarker = "https://image.tmdb.org/t/p/"

	idx := strings.Index(response, posterMarker)
	if idx < 0 {
		return
	}

	// Extract the URL (ends at whitespace or end of string).
	end := idx
	for end < len(response) && response[end] != ' ' && response[end] != '\n' && response[end] != ')' {
		end++
	}
	posterURL := response[idx:end]

	if posterURL == "" {
		return
	}

	// Send as photo — Telegram will fetch the URL itself.
	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(posterURL))
	if _, err := b.api.Send(photo); err != nil {
		b.logger.Debug("failed to send poster",
			slog.String("url", posterURL),
			slog.String("error", err.Error()),
		)
	}
}

// SendPoster sends a movie poster photo with a caption.
func (b *Bot) SendPoster(chatID int64, posterPath, caption string) {
	url := tmdb.PosterURL(posterPath, "w500")
	if url == "" {
		return
	}

	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(url))
	photo.Caption = caption
	if _, err := b.api.Send(photo); err != nil {
		b.logger.Debug("failed to send poster",
			slog.String("url", url),
			slog.String("error", err.Error()),
		)
	}
}
