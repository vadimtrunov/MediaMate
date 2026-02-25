# Fix all CodeRabbit review comments on PR #23

## Overview
Address all CodeRabbit review findings (1 critical, 5 major, 2 minor, 7 nitpicks) on the download progress tracking PR to get CodeRabbit approval.

## Context
- PR: #23 feat: add live download progress tracking (Phase 5.2)
- Branch: feat/download-progress
- Files involved:
  - `cmd/mediamate/bot.go`
  - `internal/config/config.go`
  - `internal/frontend/telegram/bot.go`
  - `internal/notification/progress.go`
  - `internal/notification/progress_test.go`
  - `internal/notification/service.go`
  - `internal/core/interfaces.go`

## Development Approach
- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Fix context cancellation and redundant nil-check in bot.go

**Files:**
- Modify: `cmd/mediamate/bot.go`

- [x] Call `cancel()` immediately after `bot.Start(ctx)` returns (line 58), before waiting on `webhookErrCh`, to unblock the webhook server goroutine
- [x] Filter `context.Canceled` from webhook errors: `!errors.Is(webhookErr, context.Canceled)`
- [x] Remove the redundant `if torrentClient != nil` guard (lines 136-142) — after the error check and `cfg.QBittorrent != nil` guard, torrentClient cannot be nil
- [x] No new tests needed (startup wiring, covered by integration)

### Task 2: Fix progress interval validation in config.go

**Files:**
- Modify: `internal/config/config.go`

- [x] In `setDefaults()` (line 444): change `Interval == 0` to `Interval <= 0` so negative values also get the 15s default
- [x] In `validateOptionalServices()`: add validation `if c.Webhook.Progress.Enabled && c.Webhook.Progress.Interval <= 0` returning an error (this catches post-setDefaults edge cases)
- [x] Write test: negative interval in config triggers validation error (or gets defaulted)
- [x] Run project test suite — must pass before task 3

### Task 3: Handle "message is not modified" in telegram/bot.go

**Files:**
- Modify: `internal/frontend/telegram/bot.go`

- [x] In `EditProgressMessage`, check if error contains "message is not modified" and return nil (no-op)
- [x] Add `strings` import
- [x] No unit test needed (requires Telegram API mock; behavior is implicitly covered by progress tracker tests)

### Task 4: Fix progress tracking logic in progress.go

**Files:**
- Modify: `internal/notification/progress.go`

- [x] **Completion ordering**: move `t.removeCompleted(completed)` before `t.sendUpdates(ctx)` in `pollAndUpdate` so `buildProgressText` sees the correct active set
- [x] **Final update on zero active**: replace the early return when `activeCount() == 0` with a check that sends a final update if tracked user messages exist (so "Все загрузки завершены!" is delivered)
- [x] **Edit failure recovery**: in `sendToUser`, when `EditProgressMessage` fails, fall back to `SendProgressMessage` and update stored messageID
- [x] **Comment for syncActiveDownloads**: add a brief comment noting that `core.Torrent` has no year metadata, so year is intentionally zero for picked-up downloads
- [x] **Distinguish disappeared vs finished**: split `applyUpdates` to return separate `completed` and `disappeared` slices; update `removeCompleted` to log differently for each case
- [x] Write/update tests for the new behaviors
- [x] Run project test suite — must pass before task 5

### Task 5: Move ProgressNotifier interface out of core

**Files:**
- Modify: `internal/core/interfaces.go`
- Modify: `internal/notification/progress.go`
- Modify: `internal/frontend/telegram/bot.go`

- [ ] Define `ProgressNotifier` interface in `internal/notification/progress.go` (same signatures)
- [ ] Update `Tracker` field type from `core.ProgressNotifier` to local `ProgressNotifier`
- [ ] Update `NewTracker` parameter type accordingly
- [ ] Remove `ProgressNotifier` from `internal/core/interfaces.go`
- [ ] Remove the `core.ProgressNotifier` compile-time check from `telegram/bot.go` (compiler still enforces at call site in cmd/)
- [ ] Verify all references compile cleanly
- [ ] Run project test suite — must pass before task 6

### Task 6: Add warning log for missing downloadId in service.go

**Files:**
- Modify: `internal/notification/service.go`

- [ ] In `NotifyDownloadComplete`, when `s.tracker != nil` but `payload.DownloadID` is empty, log a warning with movie title context
- [ ] No new tests needed (logging-only change, service tests already cover the happy path)

### Task 7: Improve test coverage in progress_test.go

**Files:**
- Modify: `internal/notification/progress_test.go`

- [ ] **Captured messageID**: in `TestPollAndUpdate_EditsExistingMessage`, replace hardcoded `messageID != 1` with `sent[0]` captured ID comparison
- [ ] **List error path test**: add `TestPollAndUpdate_ListError` — set `mockTorrentClient.err`, call `pollAndUpdate`, assert no panic, state preserved, no sends/edits
- [ ] **Error path tests**: add `TestSendToUser_SendError` and `TestSendToUser_EditError` exercising `mockNotifier.sendErr` / `mockNotifier.editErr`, including the new edit-failure-recovery path
- [ ] **Accessor for internals**: add `GetDownloadInfo(hash) (title string, year int, ok bool)` method on Tracker; update `TestTrackDownload`, `TestSyncActiveDownloads`, `TestNotifyGrab` to use it instead of locking `tr.mu` directly
- [ ] Run project test suite — must pass before task 8

### Task 8: Verify acceptance criteria

- [ ] Run full test suite: `go test ./...`
- [ ] Run linter: `PATH="/usr/local/go/bin:/home/vadym/go/bin:$PATH" golangci-lint run`
- [ ] Verify no function exceeds 60 lines (revive function-length)
- [ ] Verify no line exceeds 140 chars (revive line-length-limit)

### Task 9: Update documentation

- [ ] Update README.md if user-facing changes
- [ ] Update CLAUDE.md if internal patterns changed
- [ ] Move this plan to `docs/plans/completed/`
