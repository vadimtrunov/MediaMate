package main

import (
	"context"
	"fmt"
	"os/signal"
	"strings"
	"syscall"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/vadimtrunov/MediaMate/internal/config"
	"github.com/vadimtrunov/MediaMate/internal/core"
)

const progressBarWidth = 30

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show download status",
		Long:  "Display the status of active downloads from your torrent client.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runStatus()
		},
	}
}

func runStatus() error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	logger := config.SetupLogger(cfg.App.LogLevel)

	tc, err := initTorrent(cfg, logger)
	if err != nil {
		return fmt.Errorf("connect to torrent client: %w", err)
	}
	if tc == nil {
		fmt.Println(styleDim.Render("No torrent client configured."))
		return nil
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	torrents, err := tc.List(ctx)
	if err != nil {
		return fmt.Errorf("list torrents: %w", err)
	}

	if len(torrents) == 0 {
		fmt.Println(styleDim.Render("No active downloads."))
		return nil
	}

	fmt.Println(styleHeader.Render("Downloads"))
	for i, t := range torrents {
		printTorrent(i+1, t)
	}
	return nil
}

func printTorrent(index int, t core.Torrent) {
	statusColor := statusToColor(t.Status)
	statusStyle := lipgloss.NewStyle().Foreground(statusColor)

	nameStyle := lipgloss.NewStyle().Bold(true)
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	fmt.Printf("%s %s  %s\n",
		label.Render(fmt.Sprintf("%d.", index)),
		nameStyle.Render(t.Name),
		statusStyle.Render(t.Status),
	)

	bar := progressBar(t.Progress, progressBarWidth)
	details := fmt.Sprintf("   %s  %s %s  %s %s",
		bar,
		label.Render("↓"),
		formatSpeed(t.DownloadSpeed),
		label.Render("↑"),
		formatSpeed(t.UploadSpeed),
	)
	if t.ETA > 0 {
		details += fmt.Sprintf("  %s %s", label.Render("ETA"), formatETA(t.ETA))
	}
	fmt.Println(details)
}

func statusToColor(status string) lipgloss.Color {
	switch status {
	case "downloading":
		return lipgloss.Color("12") // blue
	case "seeding":
		return lipgloss.Color("10") // green
	case "paused":
		return lipgloss.Color("11") // yellow
	case "error":
		return lipgloss.Color("9") // red
	default:
		return lipgloss.Color("8") // gray
	}
}

func progressBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	empty := width - filled

	bar := styleInfo.Render(strings.Repeat("█", filled)) +
		styleDim.Render(strings.Repeat("░", empty))
	return fmt.Sprintf("%s %s", bar, styleDim.Render(fmt.Sprintf("%.1f%%", percent)))
}

func formatSpeed(bytesPerSec int64) string {
	switch {
	case bytesPerSec >= 1024*1024:
		return fmt.Sprintf("%.1f MB/s", float64(bytesPerSec)/(1024*1024))
	case bytesPerSec >= 1024:
		return fmt.Sprintf("%.1f KB/s", float64(bytesPerSec)/1024)
	default:
		return fmt.Sprintf("%d B/s", bytesPerSec)
	}
}

func formatETA(seconds int64) string {
	if seconds <= 0 {
		return "∞"
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
