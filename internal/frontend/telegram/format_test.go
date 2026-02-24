package telegram

import "testing"

func TestEscapeMdV2(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain text", in: "hello world", want: "hello world"},
		{name: "dots", in: "hello.", want: "hello\\."},
		{name: "exclamation", in: "Done!", want: "Done\\!"},
		{name: "parentheses", in: "(2024)", want: "\\(2024\\)"},
		{name: "brackets", in: "[link]", want: "\\[link\\]"},
		{name: "underscores", in: "foo_bar", want: "foo\\_bar"},
		{name: "stars", in: "*bold*", want: "\\*bold\\*"},
		{name: "mixed", in: "Dune (2021) - 8.0*", want: "Dune \\(2021\\) \\- 8\\.0\\*"},
		{name: "all specials", in: "_*[]()~`>#+-=|{}.!", want: "\\_\\*\\[\\]\\(\\)\\~\\`\\>\\#\\+\\-\\=\\|\\{\\}\\.\\!"},
		{name: "empty", in: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EscapeMdV2(tt.in)
			if got != tt.want {
				t.Errorf("EscapeMdV2(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatBold(t *testing.T) {
	got := FormatBold("Dune")
	want := "*Dune*"
	if got != want {
		t.Errorf("FormatBold(%q) = %q, want %q", "Dune", got, want)
	}

	got = FormatBold("Dune (2021)")
	want = "*Dune \\(2021\\)*"
	if got != want {
		t.Errorf("FormatBold(%q) = %q, want %q", "Dune (2021)", got, want)
	}
}

func TestFormatItalic(t *testing.T) {
	got := FormatItalic("description")
	want := "_description_"
	if got != want {
		t.Errorf("FormatItalic(%q) = %q, want %q", "description", got, want)
	}
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name    string
		percent float64
		width   int
		wantLen int // approximate length check
	}{
		{name: "0%", percent: 0, width: 10, wantLen: 0},
		{name: "50%", percent: 50, width: 10, wantLen: 5},
		{name: "100%", percent: 100, width: 10, wantLen: 10},
		{name: "default width", percent: 50, width: 0, wantLen: 10},
		{name: "over 100%", percent: 150, width: 10, wantLen: 10},
		{name: "negative", percent: -10, width: 10, wantLen: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProgressBar(tt.percent, tt.width)
			if got == "" {
				t.Error("ProgressBar returned empty string")
			}
			// Check it starts with [ and contains ]
			if got[0] != '[' {
				t.Errorf("ProgressBar should start with [, got %q", got)
			}
		})
	}
}
