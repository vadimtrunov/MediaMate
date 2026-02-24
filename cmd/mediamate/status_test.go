package main

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestFormatSpeed(t *testing.T) {
	tests := []struct {
		name        string
		bytesPerSec int64
		want        string
	}{
		{"zero", 0, "0 B/s"},
		{"bytes", 512, "512 B/s"},
		{"kilobytes", 1024, "1.0 KB/s"},
		{"kilobytes_fractional", 1536, "1.5 KB/s"},
		{"megabytes", 1024 * 1024, "1.0 MB/s"},
		{"megabytes_fractional", 5 * 1024 * 1024, "5.0 MB/s"},
		{"just_below_kb", 1023, "1023 B/s"},
		{"just_below_mb", 1024*1024 - 1, "1024.0 KB/s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSpeed(tt.bytesPerSec)
			if got != tt.want {
				t.Errorf("formatSpeed(%d) = %q, want %q", tt.bytesPerSec, got, tt.want)
			}
		})
	}
}

func TestFormatETA(t *testing.T) {
	tests := []struct {
		name    string
		seconds int64
		want    string
	}{
		{"zero", 0, "∞"},
		{"negative", -1, "∞"},
		{"seconds_only", 45, "45s"},
		{"one_second", 1, "1s"},
		{"minutes_seconds", 125, "2m05s"},
		{"exact_minutes", 60, "1m00s"},
		{"hours_minutes", 3661, "1h01m"},
		{"many_hours", 36000, "10h00m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatETA(tt.seconds)
			if got != tt.want {
				t.Errorf("formatETA(%d) = %q, want %q", tt.seconds, got, tt.want)
			}
		})
	}
}

func TestStatusToColor(t *testing.T) {
	tests := []struct {
		status string
		want   lipgloss.Color
	}{
		{"downloading", lipgloss.Color("12")},
		{"seeding", lipgloss.Color("10")},
		{"paused", lipgloss.Color("11")},
		{"error", lipgloss.Color("9")},
		{"unknown", lipgloss.Color("8")},
		{"", lipgloss.Color("8")},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := statusToColor(tt.status)
			if got != tt.want {
				t.Errorf("statusToColor(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name    string
		percent float64
		width   int
	}{
		{"zero", 0, 30},
		{"half", 50, 30},
		{"full", 100, 30},
		{"over", 150, 30},
		{"small_width", 50, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := progressBar(tt.percent, tt.width)
			if got == "" {
				t.Error("progressBar returned empty string")
			}
		})
	}
}
