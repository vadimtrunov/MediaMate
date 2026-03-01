package mcp

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vadimtrunov/MediaMate/internal/core"
)

// mockBackend implements core.MediaBackend for testing.
type mockBackend struct {
	addErr    error
	statusErr error
	status    *core.MediaStatus
	addedItem core.MediaItem
}

func (m *mockBackend) Search(_ context.Context, _ string) ([]core.MediaItem, error) {
	return nil, nil
}

func (m *mockBackend) Add(_ context.Context, item core.MediaItem) error {
	m.addedItem = item
	return m.addErr
}

func (m *mockBackend) GetStatus(_ context.Context, _ string) (*core.MediaStatus, error) {
	return m.status, m.statusErr
}
func (m *mockBackend) ListItems(_ context.Context) ([]core.MediaItem, error) { return nil, nil }
func (m *mockBackend) Type() string                                          { return "radarr" }

// mockTorrent implements core.TorrentClient for testing.
type mockTorrent struct {
	torrents []core.Torrent
	listErr  error
}

func (m *mockTorrent) List(_ context.Context) ([]core.Torrent, error) {
	return m.torrents, m.listErr
}

func (m *mockTorrent) GetProgress(_ context.Context, _ string) (*core.TorrentProgress, error) {
	return nil, nil
}
func (m *mockTorrent) Pause(_ context.Context, _ string) error          { return nil }
func (m *mockTorrent) Resume(_ context.Context, _ string) error         { return nil }
func (m *mockTorrent) Remove(_ context.Context, _ string, _ bool) error { return nil }
func (m *mockTorrent) Name() string                                     { return "qbittorrent" }

// mockMediaServer implements core.MediaServer for testing.
type mockMediaServer struct {
	available    bool
	availableErr error
	link         string
	linkErr      error
}

func (m *mockMediaServer) IsAvailable(_ context.Context, _ string) (bool, error) {
	return m.available, m.availableErr
}

func (m *mockMediaServer) GetLink(_ context.Context, _ string) (string, error) {
	return m.link, m.linkErr
}

func (m *mockMediaServer) GetLibraryItems(_ context.Context) ([]core.MediaItem, error) {
	return nil, nil
}
func (m *mockMediaServer) Name() string { return "jellyfin" }

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func callTool(t *testing.T, srv *Server, toolName string, args map[string]any) *mcpsdk.CallToolResult {
	t.Helper()
	ctx := context.Background()

	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()

	_, err := srv.MCPServer().Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}

	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("call tool %s: %v", toolName, err)
	}
	return result
}

func TestListDownloads(t *testing.T) {
	torrents := []core.Torrent{
		{Hash: "abc123", Name: "Movie.2024", Progress: 50.5, Status: "downloading"},
	}
	srv := NewServer(Deps{
		Torrent: &mockTorrent{torrents: torrents},
	}, discardLogger)

	result := callTool(t, srv, "list_downloads", map[string]any{})

	if result.IsError {
		t.Fatal("expected success, got error")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Content))
	}
	text, ok := result.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	var got []core.Torrent
	if err := json.Unmarshal([]byte(text.Text), &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(got) != 1 || got[0].Hash != "abc123" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestCheckAvailability(t *testing.T) {
	srv := NewServer(Deps{
		MediaServer: &mockMediaServer{available: true},
	}, discardLogger)

	result := callTool(t, srv, "check_availability", map[string]any{"title": "Inception"})

	if result.IsError {
		t.Fatal("expected success, got error")
	}
	text := result.Content[0].(*mcpsdk.TextContent)

	var got map[string]any
	if err := json.Unmarshal([]byte(text.Text), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["available"] != true {
		t.Errorf("expected available=true, got %v", got["available"])
	}
	if got["title"] != "Inception" {
		t.Errorf("expected title=Inception, got %v", got["title"])
	}
}

func TestGetWatchLink(t *testing.T) {
	srv := NewServer(Deps{
		MediaServer: &mockMediaServer{link: "http://jellyfin/movie/123"},
	}, discardLogger)

	result := callTool(t, srv, "get_watch_link", map[string]any{"title": "Inception"})

	if result.IsError {
		t.Fatal("expected success, got error")
	}
	text := result.Content[0].(*mcpsdk.TextContent)

	var got map[string]any
	if err := json.Unmarshal([]byte(text.Text), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["link"] != "http://jellyfin/movie/123" {
		t.Errorf("expected link, got %v", got["link"])
	}
}

func TestDownloadMovie(t *testing.T) {
	backend := &mockBackend{}
	srv := NewServer(Deps{Backend: backend}, discardLogger)

	result := callTool(t, srv, "download_movie", map[string]any{
		"tmdb_id": 27205,
		"title":   "Inception",
	})

	if result.IsError {
		t.Fatal("expected success, got error")
	}
	if backend.addedItem.Title != "Inception" {
		t.Errorf("expected title Inception, got %s", backend.addedItem.Title)
	}
	if backend.addedItem.Metadata["tmdbId"] != "27205" {
		t.Errorf("expected tmdbId 27205, got %s", backend.addedItem.Metadata["tmdbId"])
	}
}

func TestGetDownloadStatus(t *testing.T) {
	backend := &mockBackend{
		status: &core.MediaStatus{
			ItemID: "123",
			Status: "downloading",
		},
	}
	srv := NewServer(Deps{Backend: backend}, discardLogger)

	result := callTool(t, srv, "get_download_status", map[string]any{"radarr_id": 123})

	if result.IsError {
		t.Fatal("expected success, got error")
	}
	text := result.Content[0].(*mcpsdk.TextContent)

	var got core.MediaStatus
	if err := json.Unmarshal([]byte(text.Text), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Status != "downloading" {
		t.Errorf("expected downloading, got %s", got.Status)
	}
}

func TestToolError_NilDependency(t *testing.T) {
	srv := NewServer(Deps{}, discardLogger)

	tests := []struct {
		tool string
		args map[string]any
	}{
		{"list_downloads", map[string]any{}},
		{"check_availability", map[string]any{"title": "Test"}},
		{"get_watch_link", map[string]any{"title": "Test"}},
		{"download_movie", map[string]any{"tmdb_id": 1, "title": "Test"}},
		{"get_download_status", map[string]any{"radarr_id": 1}},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			result := callTool(t, srv, tt.tool, tt.args)
			if !result.IsError {
				t.Errorf("expected error for %s with nil dependency", tt.tool)
			}
		})
	}
}

func TestToolError_MissingArgs(t *testing.T) {
	srv := NewServer(Deps{
		MediaServer: &mockMediaServer{},
	}, discardLogger)

	result := callTool(t, srv, "check_availability", map[string]any{})

	if !result.IsError {
		t.Fatal("expected error for missing title argument")
	}
}
