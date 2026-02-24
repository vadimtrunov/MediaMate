package telegram

import (
	"fmt"
	"strings"
)

// mdV2Replacer escapes special characters for Telegram MarkdownV2.
var mdV2Replacer = strings.NewReplacer(
	`\`, `\\`,
	"_", "\\_",
	"*", "\\*",
	"[", "\\[",
	"]", "\\]",
	"(", "\\(",
	")", "\\)",
	"~", "\\~",
	"`", "\\`",
	">", "\\>",
	"#", "\\#",
	"+", "\\+",
	"-", "\\-",
	"=", "\\=",
	"|", "\\|",
	"{", "\\{",
	"}", "\\}",
	".", "\\.",
	"!", "\\!",
)

// EscapeMdV2 escapes a string for safe use in Telegram MarkdownV2.
func EscapeMdV2(s string) string {
	return mdV2Replacer.Replace(s)
}

// FormatBold returns MarkdownV2 bold text.
func FormatBold(s string) string {
	return "*" + EscapeMdV2(s) + "*"
}

// FormatItalic returns MarkdownV2 italic text.
func FormatItalic(s string) string {
	return "_" + EscapeMdV2(s) + "_"
}

// ProgressBar generates an ASCII progress bar.
// width is the total number of characters for the bar body.
func ProgressBar(percent float64, width int) string {
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
		strings.Repeat("█", filled),
		strings.Repeat("░", width-filled),
		percent,
	)
}
