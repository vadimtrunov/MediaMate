package notification

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/vadimtrunov/MediaMate/internal/core"
)

// Service sends notifications to users when media events occur.
type Service struct {
	frontend    core.Frontend
	mediaServer core.MediaServer
	userIDs     []int64
	logger      *slog.Logger
}

// NewService creates a notification service.
// frontend is required; mediaServer may be nil (Jellyfin links will be skipped).
func NewService(
	frontend core.Frontend,
	mediaServer core.MediaServer,
	userIDs []int64,
	logger *slog.Logger,
) *Service {
	if frontend == nil {
		panic("notification.NewService: frontend must not be nil")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		frontend:    frontend,
		mediaServer: mediaServer,
		userIDs:     userIDs,
		logger:      logger,
	}
}

// NotifyDownloadComplete sends a Telegram message about a downloaded movie.
func (s *Service) NotifyDownloadComplete(ctx context.Context, payload *RadarrWebhookPayload) error {
	if payload == nil {
		return fmt.Errorf("nil Radarr payload")
	}
	title := payload.MovieTitle()
	year := payload.MovieYear()

	msg := s.buildDownloadMessage(ctx, title, year)

	s.logger.Info("sending download notification",
		slog.String("title", title),
		slog.Int("year", year),
		slog.Int("recipients", len(s.userIDs)),
	)

	if len(s.userIDs) == 0 {
		s.logger.Warn("no recipients configured, download notification will not be sent",
			slog.String("title", title),
		)
		return nil
	}

	var firstErr error
	for _, uid := range s.userIDs {
		userID := strconv.FormatInt(uid, 10)
		if err := s.frontend.SendMessage(ctx, userID, msg); err != nil {
			s.logger.Error("failed to send notification",
				slog.String("user_id", userID),
				slog.String("error", err.Error()),
			)
			if firstErr == nil {
				firstErr = fmt.Errorf("send to %s: %w", userID, err)
			}
		}
	}
	return firstErr
}

// buildDownloadMessage creates the notification text with an optional Jellyfin link.
// Messages are sent as plain text (no ParseMode), so no markdown escaping is applied.
func (s *Service) buildDownloadMessage(ctx context.Context, title string, year int) string {
	link := s.getJellyfinLink(ctx, title)
	if link != "" {
		return fmt.Sprintf("🎬 %s (%d) готов к просмотру!\n%s", title, year, link)
	}
	return fmt.Sprintf("🎬 %s (%d) готов к просмотру!", title, year)
}

// getJellyfinLink returns a Jellyfin watch link, or empty string on failure.
func (s *Service) getJellyfinLink(ctx context.Context, title string) string {
	if s.mediaServer == nil {
		return ""
	}
	link, err := s.mediaServer.GetLink(ctx, title)
	if err != nil {
		s.logger.Warn("jellyfin link unavailable, sending without link",
			slog.String("title", title),
			slog.String("error", err.Error()),
		)
		return ""
	}
	return link
}
