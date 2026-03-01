package agent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/vadimtrunov/MediaMate/internal/core"
)

// mockLLM implements core.LLMProvider for testing.
type mockLLM struct {
	responses []*core.Response
	calls     int
}

func (m *mockLLM) Chat(_ context.Context, _ []core.Message, _ []core.Tool) (*core.Response, error) {
	if m.calls >= len(m.responses) {
		return nil, fmt.Errorf("no more responses")
	}
	resp := m.responses[m.calls]
	m.calls++
	return resp, nil
}

func (m *mockLLM) Name() string { return "mock" }

func (m *mockLLM) Close() error { return nil }

// mockBackend implements core.MediaBackend for testing.
type mockBackend struct {
	addCalled  bool
	addedItem  core.MediaItem
	statusResp *core.MediaStatus
	searchResp []core.MediaItem
	listResp   []core.MediaItem
}

func (m *mockBackend) Search(_ context.Context, _ string) ([]core.MediaItem, error) {
	return m.searchResp, nil
}

func (m *mockBackend) Add(_ context.Context, item core.MediaItem) error {
	m.addCalled = true
	m.addedItem = item
	return nil
}

func (m *mockBackend) GetStatus(_ context.Context, _ string) (*core.MediaStatus, error) {
	if m.statusResp != nil {
		return m.statusResp, nil
	}
	return &core.MediaStatus{ItemID: "1", Status: "wanted"}, nil
}

func (m *mockBackend) ListItems(_ context.Context) ([]core.MediaItem, error) {
	return m.listResp, nil
}

func (m *mockBackend) Type() string { return "mock" }

// mockTorrent implements core.TorrentClient for testing.
type mockTorrent struct {
	torrents []core.Torrent
}

func (m *mockTorrent) List(_ context.Context) ([]core.Torrent, error) {
	return m.torrents, nil
}

func (m *mockTorrent) GetProgress(_ context.Context, hash string) (*core.TorrentProgress, error) {
	return &core.TorrentProgress{Hash: hash, Progress: 50}, nil
}

func (m *mockTorrent) Pause(_ context.Context, _ string) error          { return nil }
func (m *mockTorrent) Resume(_ context.Context, _ string) error         { return nil }
func (m *mockTorrent) Remove(_ context.Context, _ string, _ bool) error { return nil }
func (m *mockTorrent) Name() string                                     { return "mock" }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHandleMessage_SimpleResponse(t *testing.T) {
	llm := &mockLLM{
		responses: []*core.Response{
			{Content: "Hello! How can I help?", Done: true},
		},
	}

	a := New(llm, nil, nil, nil, nil, testLogger())
	resp, err := a.HandleMessage(context.Background(), "Hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Hello! How can I help?" {
		t.Errorf("expected greeting, got %s", resp)
	}
}

func TestHandleMessage_WithToolCall(t *testing.T) {
	llm := &mockLLM{
		responses: []*core.Response{
			{
				Content: "Let me search for that.",
				ToolCalls: []core.ToolCall{
					{ID: "call_1", Name: "search_movie", Arguments: map[string]any{"query": "inception"}},
				},
			},
			{Content: "I found Inception (2010)!", Done: true},
		},
	}

	// tmdb is nil — tool will return error, but LLM gets the error as tool result and continues
	a := New(llm, nil, nil, nil, nil, testLogger())

	resp, err := a.HandleMessage(context.Background(), "Find inception")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "I found Inception (2010)!" {
		t.Errorf("unexpected response: %s", resp)
	}
	if llm.calls != 2 {
		t.Errorf("expected 2 LLM calls, got %d", llm.calls)
	}
}

func TestHandleMessage_DownloadMovie(t *testing.T) {
	backend := &mockBackend{}
	llm := &mockLLM{
		responses: []*core.Response{
			{
				ToolCalls: []core.ToolCall{
					{ID: "call_1", Name: "download_movie", Arguments: map[string]any{
						"tmdb_id": float64(27205),
						"title":   "Inception",
					}},
				},
			},
			{Content: "Added Inception to downloads!", Done: true},
		},
	}

	a := New(llm, nil, backend, nil, nil, testLogger())
	resp, err := a.HandleMessage(context.Background(), "Download inception")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Added Inception to downloads!" {
		t.Errorf("unexpected response: %s", resp)
	}
	if !backend.addCalled {
		t.Error("expected backend.Add to be called")
	}
	if backend.addedItem.Metadata["tmdbId"] != "27205" {
		t.Errorf("expected tmdbId 27205, got %s", backend.addedItem.Metadata["tmdbId"])
	}
}

func TestHandleMessage_ListDownloads(t *testing.T) {
	torrent := &mockTorrent{
		torrents: []core.Torrent{
			{Hash: "abc", Name: "Movie.mkv", Progress: 75, Status: "downloading"},
		},
	}
	llm := &mockLLM{
		responses: []*core.Response{
			{
				ToolCalls: []core.ToolCall{
					{ID: "call_1", Name: "list_downloads", Arguments: map[string]any{}},
				},
			},
			{Content: "You have 1 active download.", Done: true},
		},
	}

	a := New(llm, nil, nil, torrent, nil, testLogger())
	resp, err := a.HandleMessage(context.Background(), "What's downloading?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "You have 1 active download." {
		t.Errorf("unexpected response: %s", resp)
	}
}

func TestHandleMessage_MaxIterations(t *testing.T) {
	// LLM always returns a tool call, never a final response
	llm := &mockLLM{}
	for range maxToolIterations + 1 {
		llm.responses = append(llm.responses, &core.Response{
			ToolCalls: []core.ToolCall{
				{ID: "call", Name: "list_downloads", Arguments: map[string]any{}},
			},
		})
	}

	torrent := &mockTorrent{}
	a := New(llm, nil, nil, torrent, nil, testLogger())
	_, err := a.HandleMessage(context.Background(), "Loop forever")
	if err == nil {
		t.Fatal("expected error for max iterations")
	}
}

func TestHandleMessage_ToolError(t *testing.T) {
	llm := &mockLLM{
		responses: []*core.Response{
			{
				ToolCalls: []core.ToolCall{
					{ID: "call_1", Name: "download_movie", Arguments: map[string]any{
						"tmdb_id": float64(1),
						"title":   "Test",
					}},
				},
			},
			{Content: "Sorry, downloads are not available.", Done: true},
		},
	}

	// No backend configured — tool will return error
	a := New(llm, nil, nil, nil, nil, testLogger())
	resp, err := a.HandleMessage(context.Background(), "Download test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Sorry, downloads are not available." {
		t.Errorf("unexpected response: %s", resp)
	}

	// Verify error was passed back to LLM
	found := false
	for _, msg := range a.history {
		if msg.IsError && msg.ToolResultID == "call_1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error tool result in history")
	}
}

func TestReset(t *testing.T) {
	llm := &mockLLM{
		responses: []*core.Response{
			{Content: "Hello!", Done: true},
		},
	}

	a := New(llm, nil, nil, nil, nil, testLogger())
	_, _ = a.HandleMessage(context.Background(), "Hi")

	if len(a.history) <= 1 {
		t.Fatal("expected history to have messages")
	}

	a.Reset()
	if len(a.history) != 1 {
		t.Errorf("expected 1 message (system) after reset, got %d", len(a.history))
	}
	if a.history[0].Role != "system" {
		t.Errorf("expected system message, got %s", a.history[0].Role)
	}
}

func TestExtractIntArg(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		key     string
		want    int
		wantErr bool
	}{
		{"float64", map[string]any{"id": float64(42)}, "id", 42, false},
		{"int", map[string]any{"id": 42}, "id", 42, false},
		{"string", map[string]any{"id": "42"}, "id", 42, false},
		{"missing", map[string]any{}, "id", 0, true},
		{"invalid_string", map[string]any{"id": "abc"}, "id", 0, true},
		{"wrong_type", map[string]any{"id": true}, "id", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractIntArg(tt.args, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}
