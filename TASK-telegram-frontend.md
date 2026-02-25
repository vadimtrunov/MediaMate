# Feature: Telegram Frontend (Phase 3.1 + 3.2)

## Task ID
telegram-frontend

## Description
Full Telegram frontend for MediaMate: bot with text message handling, inline keyboards,
typing indicators, multi-user sessions with whitelist auth, poster photos, MarkdownV2
formatting, ASCII progress bars, and callback buttons.

## Acceptance Criteria
- [x] Telegram bot starts via `mediamate bot` CLI command
- [x] Bot accepts text messages and responds via LLM agent
- [x] Unauthorized users get rejection message
- [x] Typing indicator shown during LLM processing
- [x] Inline keyboard for selection from multiple options
- [x] Callback buttons work for movie selection
- [x] Movie posters sent as photos when poster_url available
- [x] Responses formatted with MarkdownV2
- [x] Per-user isolated sessions (separate agent history)
- [x] Graceful shutdown via context/signals
- [x] Tests with mock bot API

## Plan
- [x] Step 1: Add tgbotapi v5 dependency
- [x] Step 2: Telegram bot core (bot.go) — Bot struct, Frontend interface, long polling, shutdown
- [x] Step 3: Session management (session.go) — per-user Agent, whitelist auth, mutex
- [x] Step 4: Message handler (handler.go) — text dispatch, typing indicator, error handling
- [x] Step 5: MarkdownV2 formatter (format.go) — safe escaping, response formatting
- [x] Step 6: Inline keyboard & callbacks (keyboard.go) — callback routing, selection buttons
- [x] Step 7: Rich media — poster photos via SendPhoto, ASCII progress bars
- [x] Step 8: CLI command (bot.go) — `mediamate bot` Cobra command
- [x] Step 9: Tests — formatter, session, handler with mocks
- [x] Step 10: Build, lint, test verification

## Constraints
- Library: github.com/go-telegram-bot-api/telegram-bot-api/v5
- Long polling (not webhook)
- MarkdownV2 parse mode
- In-memory sessions (no persistence)
- Follow existing patterns from agent, config, helpers.go

## Context
- core.Frontend interface: Start, Stop, SendMessage, Name
- agent.Agent: New(), HandleMessage(), Reset()
- config.TelegramConfig: BotToken, AllowedUserIDs
- cmd/mediamate/helpers.go: initServices pattern
- cmd/mediamate/chat.go: signal handling pattern

## Feedback Log

### Steps 1-10: Full implementation (2026-02-24)
- Result: OK
- Keyboard+callbacks merged into handler.go (simpler than separate file)
- Rich media (poster, progress bar) integrated into handler.go and format.go
- golangci-lint unavailable on ARM64 RPi — used go vet instead
- Race detector unsupported on this kernel VMA — tests run without -race

---
*Created: 2026-02-24*
*Status: complete*
