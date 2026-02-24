package telegram

import "testing"

func TestBuildSelectionKeyboard(t *testing.T) {
	b := &Bot{}

	t.Run("no numbered list", func(t *testing.T) {
		kb := b.buildSelectionKeyboard("Hello, how can I help you?")
		if kb != nil {
			t.Error("expected nil keyboard for non-list response")
		}
	})

	t.Run("single item not enough", func(t *testing.T) {
		kb := b.buildSelectionKeyboard("1. Dune (2021)")
		if kb != nil {
			t.Error("expected nil keyboard for single item")
		}
	})

	t.Run("numbered list detected", func(t *testing.T) {
		response := `Here are some results:
1. Dune (2021) - 8.0
2. Dune: Part Two (2024) - 8.3
3. Dune (1984) - 6.2`

		kb := b.buildSelectionKeyboard(response)
		if kb == nil {
			t.Fatal("expected keyboard for numbered list")
		}
		if len(kb.InlineKeyboard) != 3 {
			t.Errorf("expected 3 rows, got %d", len(kb.InlineKeyboard))
		}
		if kb.InlineKeyboard[0][0].CallbackData == nil {
			t.Fatal("expected callback data on button")
		}
		if *kb.InlineKeyboard[0][0].CallbackData != "sel:1" {
			t.Errorf("expected callback data 'sel:1', got %q", *kb.InlineKeyboard[0][0].CallbackData)
		}
	})

	t.Run("parenthesis format", func(t *testing.T) {
		response := "1) Arrival\n2) Annihilation"
		kb := b.buildSelectionKeyboard(response)
		if kb == nil {
			t.Fatal("expected keyboard for parenthesis-numbered list")
		}
		if len(kb.InlineKeyboard) != 2 {
			t.Errorf("expected 2 rows, got %d", len(kb.InlineKeyboard))
		}
	})

	t.Run("long label truncated", func(t *testing.T) {
		response := "1. A very long movie title that exceeds thirty characters in length\n2. Another movie"
		kb := b.buildSelectionKeyboard(response)
		if kb == nil {
			t.Fatal("expected keyboard")
		}
		label := kb.InlineKeyboard[0][0].Text
		if len(label) > 40 {
			t.Errorf("expected truncated label, got length %d: %q", len(label), label)
		}
	})
}

func TestSendPosterIfAvailable_NoPoster(_ *testing.T) {
	// Just verify no panic when bot API is nil — the method is best-effort.
	b := &Bot{}
	// No poster URL in response — should return without error.
	b.sendPosterIfAvailable(0, "Hello, no poster here")
}
