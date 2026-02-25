package notification

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vadimtrunov/MediaMate/internal/core"
)

// ProgressNotifier can send and update progress messages to users.
type ProgressNotifier interface {
	// SendProgressMessage sends a new progress message and returns the message ID.
	SendProgressMessage(ctx context.Context, chatID int64, text string) (int, error)

	// EditProgressMessage updates an existing progress message.
	EditProgressMessage(ctx context.Context, chatID int64, messageID int, text string) error
}

// Compile-time interface check.
var _ ProgressTracker = (*Tracker)(nil)

// progressThreshold is the minimum progress change (%) to trigger an update.
const progressThreshold = 2.0

// progressBarWidth is the character width of the ASCII progress bar.
const progressBarWidth = 15

// secondsPerMinute and secondsPerHour for ETA formatting.
const (
	secondsPerMinute = 60
	secondsPerHour   = 3600
)

// trackedDownload holds state for a single tracked torrent.
type trackedDownload struct {
	hash         string
	title        string
	year         int
	lastProgress float64
	lastStatus   string
	speed        int64
	eta          int64
}

// userProgress holds per-user progress message state.
type userProgress struct {
	messageID int
}

// Tracker polls qBittorrent for active downloads and updates Telegram messages.
type Tracker struct {
	torrent  core.TorrentClient
	notifier ProgressNotifier
	userIDs  []int64
	interval time.Duration

	mu        sync.Mutex
	downloads map[string]*trackedDownload
	users     map[int64]*userProgress

	logger *slog.Logger
}

// NewTracker creates a new progress tracker.
func NewTracker(
	torrent core.TorrentClient,
	notifier ProgressNotifier,
	userIDs []int64,
	interval time.Duration,
	logger *slog.Logger,
) *Tracker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Tracker{
		torrent:   torrent,
		notifier:  notifier,
		userIDs:   userIDs,
		interval:  interval,
		downloads: make(map[string]*trackedDownload),
		users:     make(map[int64]*userProgress),
		logger:    logger,
	}
}

// TrackDownload starts tracking a torrent download.
func (t *Tracker) TrackDownload(hash, title string, year int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.downloads[hash]; ok {
		return
	}
	t.downloads[hash] = &trackedDownload{
		hash:  hash,
		title: title,
		year:  year,
	}
	t.logger.Info("tracking download", slog.String("hash", hash), slog.String("title", title))
}

// CompleteDownload marks a download as complete and removes it from tracking.
func (t *Tracker) CompleteDownload(hash string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.downloads, hash)
	t.logger.Info("completed download", slog.String("hash", hash))
}

// Start runs the polling loop until ctx is canceled.
func (t *Tracker) Start(ctx context.Context) error {
	t.syncActiveDownloads(ctx)

	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			t.pollAndUpdate(ctx)
		}
	}
}

// syncActiveDownloads picks up already-running downloads on startup.
// Note: core.Torrent has no year metadata, so year is intentionally zero for picked-up downloads.
func (t *Tracker) syncActiveDownloads(ctx context.Context) {
	torrents, err := t.torrent.List(ctx)
	if err != nil {
		t.logger.Error("failed to list torrents on startup", slog.String("error", err.Error()))
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	for i := range torrents {
		tr := &torrents[i]
		if tr.Status != "downloading" {
			continue
		}
		if _, ok := t.downloads[tr.Hash]; ok {
			continue
		}
		t.downloads[tr.Hash] = &trackedDownload{
			hash:         tr.Hash,
			title:        tr.Name,
			lastProgress: tr.Progress,
			lastStatus:   tr.Status,
			speed:        tr.DownloadSpeed,
			eta:          tr.ETA,
		}
		t.logger.Info("picked up active download", slog.String("hash", tr.Hash), slog.String("name", tr.Name))
	}
}

// pollAndUpdate performs a single update cycle.
func (t *Tracker) pollAndUpdate(ctx context.Context) {
	active := t.activeCount()
	hasMessages := t.hasTrackedMessages()

	if active == 0 && !hasMessages {
		return
	}

	// No active downloads but user messages exist: send final "all complete" update.
	if active == 0 && hasMessages {
		t.sendUpdates(ctx)
		return
	}

	torrents, err := t.torrent.List(ctx)
	if err != nil {
		t.logger.Error("failed to list torrents", slog.String("error", err.Error()))
		return
	}

	torrentMap := buildTorrentMap(torrents)
	changed, completed, disappeared := t.applyUpdates(torrentMap)

	t.removeCompleted(completed, disappeared)

	if changed {
		t.sendUpdates(ctx)
	}
}

// activeCount returns the number of tracked downloads.
func (t *Tracker) activeCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	return len(t.downloads)
}

// hasTrackedMessages returns true if any user still has a tracked progress message.
func (t *Tracker) hasTrackedMessages() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return len(t.users) > 0
}

// buildTorrentMap creates a lookup map from hash to torrent.
func buildTorrentMap(torrents []core.Torrent) map[string]*core.Torrent {
	m := make(map[string]*core.Torrent, len(torrents))
	for i := range torrents {
		m[torrents[i].Hash] = &torrents[i]
	}
	return m
}

// applyUpdates checks each tracked download against fresh torrent data.
// Returns whether any update occurred, completed hashes (finished), and disappeared hashes (no longer reported).
func (t *Tracker) applyUpdates(
	torrentMap map[string]*core.Torrent,
) (bool, []string, []string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	changed := false
	var completed []string
	var disappeared []string

	for hash, dl := range t.downloads {
		tr, ok := torrentMap[hash]
		if !ok {
			disappeared = append(disappeared, hash)
			changed = true
			continue
		}
		if isFinished(tr) {
			completed = append(completed, hash)
			changed = true
			continue
		}
		if t.shouldUpdate(dl, tr) {
			dl.lastProgress = tr.Progress
			dl.lastStatus = tr.Status
			dl.speed = tr.DownloadSpeed
			dl.eta = tr.ETA
			changed = true
		}
	}
	return changed, completed, disappeared
}

// isFinished returns true if the torrent is done downloading.
func isFinished(tr *core.Torrent) bool {
	return tr.Status == "seeding" || tr.Status == "completed" || tr.Progress >= 100
}

// shouldUpdate returns true if the download changed enough to warrant an update.
func (t *Tracker) shouldUpdate(dl *trackedDownload, tr *core.Torrent) bool {
	progressDelta := tr.Progress - dl.lastProgress
	if progressDelta < 0 {
		progressDelta = -progressDelta
	}
	return progressDelta >= progressThreshold || dl.lastStatus != tr.Status
}

// removeCompleted deletes finished and disappeared downloads from tracking.
func (t *Tracker) removeCompleted(completed, disappeared []string) {
	if len(completed) == 0 && len(disappeared) == 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, h := range completed {
		delete(t.downloads, h)
		t.logger.Info("download finished", slog.String("hash", h))
	}
	for _, h := range disappeared {
		delete(t.downloads, h)
		t.logger.Warn("download disappeared from torrent client", slog.String("hash", h))
	}
}

// sendUpdates sends or edits progress messages for all tracked users.
func (t *Tracker) sendUpdates(ctx context.Context) {
	text, remaining := t.buildProgressText()

	for _, uid := range t.userIDs {
		t.sendToUser(ctx, uid, text)
	}

	// Reset message IDs when all downloads complete so the next batch gets a new message.
	if remaining == 0 {
		t.mu.Lock()
		t.users = make(map[int64]*userProgress)
		t.mu.Unlock()
	}
}

// sendToUser sends or edits a progress message for one user.
func (t *Tracker) sendToUser(ctx context.Context, chatID int64, text string) {
	t.mu.Lock()
	up, ok := t.users[chatID]
	t.mu.Unlock()

	if !ok || up.messageID == 0 {
		msgID, err := t.notifier.SendProgressMessage(ctx, chatID, text)
		if err != nil {
			t.logger.Error("failed to send progress message",
				slog.Int64("chat_id", chatID), slog.String("error", err.Error()))
			return
		}
		t.mu.Lock()
		t.users[chatID] = &userProgress{messageID: msgID}
		t.mu.Unlock()
		return
	}

	if err := t.notifier.EditProgressMessage(ctx, chatID, up.messageID, text); err != nil {
		t.logger.Warn("failed to edit progress message, falling back to new message",
			slog.Int64("chat_id", chatID), slog.String("error", err.Error()))

		newID, sendErr := t.notifier.SendProgressMessage(ctx, chatID, text)
		if sendErr != nil {
			t.logger.Error("failed to send fallback progress message",
				slog.Int64("chat_id", chatID), slog.String("error", sendErr.Error()))
			return
		}
		t.mu.Lock()
		t.users[chatID] = &userProgress{messageID: newID}
		t.mu.Unlock()
	}
}

// buildProgressText generates the combined progress summary and returns the active download count.
func (t *Tracker) buildProgressText() (string, int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.downloads) == 0 {
		return "Все загрузки завершены!", 0
	}

	hashes := make([]string, 0, len(t.downloads))
	for h := range t.downloads {
		hashes = append(hashes, h)
	}
	sort.Strings(hashes)

	var b strings.Builder
	b.WriteString("Активные загрузки:\n")

	for _, h := range hashes {
		writeDownloadLine(&b, t.downloads[h])
	}
	return b.String(), len(t.downloads)
}

// writeDownloadLine appends a single download's progress to the builder.
func writeDownloadLine(b *strings.Builder, dl *trackedDownload) {
	b.WriteByte('\n')
	if dl.year > 0 {
		fmt.Fprintf(b, "%s (%d)\n", dl.title, dl.year)
	} else {
		fmt.Fprintf(b, "%s\n", dl.title)
	}
	bar := progressBar(dl.lastProgress, progressBarWidth)
	fmt.Fprintf(b, "%s | %s | %s\n", bar, formatSpeed(dl.speed), formatETA(dl.eta))
}

// progressBar generates an ASCII progress bar.
func progressBar(percent float64, width int) string {
	if width < 1 {
		width = 20
	}
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return fmt.Sprintf("[%s%s] %.1f%%",
		strings.Repeat("\u2588", filled),
		strings.Repeat("\u2591", width-filled),
		percent,
	)
}

// formatSpeed converts bytes/sec to a human-readable string.
func formatSpeed(bytesPerSec int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
	)
	switch {
	case bytesPerSec >= mb:
		return fmt.Sprintf("%.1f MB/s", float64(bytesPerSec)/float64(mb))
	case bytesPerSec >= kb:
		return fmt.Sprintf("%.1f KB/s", float64(bytesPerSec)/float64(kb))
	default:
		return fmt.Sprintf("%d B/s", bytesPerSec)
	}
}

// formatETA converts seconds to a human-readable ETA string.
func formatETA(seconds int64) string {
	switch {
	case seconds < secondsPerMinute:
		return "<1 мин"
	case seconds < secondsPerHour:
		return fmt.Sprintf("~%d мин", seconds/secondsPerMinute)
	default:
		h := seconds / secondsPerHour
		m := (seconds % secondsPerHour) / secondsPerMinute
		if m == 0 {
			return fmt.Sprintf("~%d ч", h)
		}
		return fmt.Sprintf("~%d ч %d мин", h, m)
	}
}
