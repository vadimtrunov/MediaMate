package notification

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vadimtrunov/MediaMate/internal/core"
)

var errTestEdit = errors.New("edit failed")

// mockTorrentClient implements core.TorrentClient for testing.
type mockTorrentClient struct {
	mu       sync.Mutex
	torrents []core.Torrent
	err      error
}

func (m *mockTorrentClient) List(_ context.Context) ([]core.Torrent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.torrents, m.err
}

func (m *mockTorrentClient) GetProgress(_ context.Context, _ string) (*core.TorrentProgress, error) {
	return nil, nil
}

func (m *mockTorrentClient) Pause(_ context.Context, _ string) error  { return nil }
func (m *mockTorrentClient) Resume(_ context.Context, _ string) error { return nil }
func (m *mockTorrentClient) Name() string                             { return "mock" }

func (m *mockTorrentClient) Remove(_ context.Context, _ string, _ bool) error {
	return nil
}

func (m *mockTorrentClient) setTorrents(torrents []core.Torrent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.torrents = torrents
}

// mockNotifier implements ProgressNotifier for testing.
type mockNotifier struct {
	mu      sync.Mutex
	sent    []progressSentMessage
	edited  []progressEditedMessage
	nextID  int
	sendErr error
	editErr error
}

type progressSentMessage struct {
	chatID int64
	text   string
}

type progressEditedMessage struct {
	chatID    int64
	messageID int
	text      string
}

func (m *mockNotifier) SendProgressMessage(
	_ context.Context, chatID int64, text string,
) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sendErr != nil {
		return 0, m.sendErr
	}
	m.nextID++
	m.sent = append(m.sent, progressSentMessage{chatID: chatID, text: text})

	return m.nextID, nil
}

func (m *mockNotifier) EditProgressMessage(
	_ context.Context, chatID int64, messageID int, text string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.editErr != nil {
		return m.editErr
	}
	m.edited = append(m.edited, progressEditedMessage{
		chatID: chatID, messageID: messageID, text: text,
	})

	return nil
}

func (m *mockNotifier) getSent() []progressSentMessage {
	m.mu.Lock()
	defer m.mu.Unlock()

	cp := make([]progressSentMessage, len(m.sent))
	copy(cp, m.sent)

	return cp
}

func (m *mockNotifier) getEdited() []progressEditedMessage {
	m.mu.Lock()
	defer m.mu.Unlock()

	cp := make([]progressEditedMessage, len(m.edited))
	copy(cp, m.edited)

	return cp
}

// stubFrontend implements core.Frontend for TestNotifyGrab.
type stubFrontend struct{}

func (stubFrontend) Start(_ context.Context) error                    { return nil }
func (stubFrontend) Stop(_ context.Context) error                     { return nil }
func (stubFrontend) Name() string                                     { return "stub" }
func (stubFrontend) SendMessage(_ context.Context, _, _ string) error { return nil }

func newTestTracker(
	tc core.TorrentClient, n ProgressNotifier, userIDs []int64,
) *Tracker {
	return NewTracker(tc, n, userIDs, time.Second, nil)
}

func TestTrackDownload(t *testing.T) {
	t.Parallel()

	tc := &mockTorrentClient{}
	notif := &mockNotifier{}
	tr := newTestTracker(tc, notif, []int64{100})

	tr.TrackDownload("abc123", "Dune", 2021)

	if tr.activeCount() != 1 {
		t.Fatalf("expected 1 tracked download, got %d", tr.activeCount())
	}

	// Duplicate should not be added.
	tr.TrackDownload("abc123", "Dune Part Two", 2024)

	if tr.activeCount() != 1 {
		t.Fatalf("expected 1 tracked download after duplicate, got %d", tr.activeCount())
	}

	// Verify original title is preserved.
	tr.mu.Lock()
	dl := tr.downloads["abc123"]
	tr.mu.Unlock()

	if dl.title != "Dune" {
		t.Errorf("expected title 'Dune', got %q", dl.title)
	}
}

func TestCompleteDownload(t *testing.T) {
	t.Parallel()

	tc := &mockTorrentClient{}
	notif := &mockNotifier{}
	tr := newTestTracker(tc, notif, []int64{100})

	tr.TrackDownload("abc123", "Dune", 2021)
	tr.CompleteDownload("abc123")

	if tr.activeCount() != 0 {
		t.Fatalf("expected 0 tracked downloads after complete, got %d", tr.activeCount())
	}

	// Completing a non-existent hash should not panic.
	tr.CompleteDownload("nonexistent")
}

func TestFormatSpeed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bps      int64
		expected string
	}{
		{"bytes", 500, "500 B/s"},
		{"kilobytes", 2048, "2.0 KB/s"},
		{"megabytes", 5 * 1024 * 1024, "5.0 MB/s"},
		{"zero", 0, "0 B/s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := formatSpeed(tt.bps)
			if got != tt.expected {
				t.Errorf("formatSpeed(%d) = %q, want %q", tt.bps, got, tt.expected)
			}
		})
	}
}

func TestFormatETA(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		seconds  int64
		expected string
	}{
		{"zero", 0, "<1 мин"},
		{"negative", -10, "<1 мин"},
		{"under_minute", 30, "<1 мин"},
		{"five_minutes", 300, "~5 мин"},
		{"two_hours_15_min", 2*3600 + 15*60, "~2 ч 15 мин"},
		{"exact_hour", 3600, "~1 ч"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := formatETA(tt.seconds)
			if got != tt.expected {
				t.Errorf("formatETA(%d) = %q, want %q", tt.seconds, got, tt.expected)
			}
		})
	}
}

func TestBuildProgressText_Active(t *testing.T) {
	t.Parallel()

	t.Run("with_year", func(t *testing.T) {
		t.Parallel()

		tc := &mockTorrentClient{}
		notif := &mockNotifier{}
		tr := newTestTracker(tc, notif, []int64{100})

		tr.TrackDownload("hash1", "Dune", 2021)

		text, count := tr.buildProgressText()

		if count != 1 {
			t.Errorf("expected count 1, got %d", count)
		}
		if !strings.Contains(text, "Dune (2021)") {
			t.Errorf("expected title with year, got: %s", text)
		}
		if !strings.Contains(text, "Активные загрузки:") {
			t.Errorf("expected header, got: %s", text)
		}
	})

	t.Run("without_year", func(t *testing.T) {
		t.Parallel()

		tc := &mockTorrentClient{}
		notif := &mockNotifier{}
		tr := newTestTracker(tc, notif, []int64{100})

		tr.TrackDownload("hash2", "Unknown Movie", 0)

		text, count := tr.buildProgressText()

		if count != 1 {
			t.Errorf("expected count 1, got %d", count)
		}
		if strings.Contains(text, "(0)") {
			t.Errorf("should not contain (0) for zero year, got: %s", text)
		}
		if !strings.Contains(text, "Unknown Movie") {
			t.Errorf("expected title without year, got: %s", text)
		}
	})
}

func TestBuildProgressText_Empty(t *testing.T) {
	t.Parallel()

	tc := &mockTorrentClient{}
	notif := &mockNotifier{}
	tr := newTestTracker(tc, notif, []int64{100})

	text, count := tr.buildProgressText()

	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}
	if text != "Все загрузки завершены!" {
		t.Errorf("expected completion message, got: %s", text)
	}
}

func TestPollAndUpdate_SendsNewMessage(t *testing.T) {
	t.Parallel()

	tc := &mockTorrentClient{
		torrents: []core.Torrent{
			{
				Hash:          "hash1",
				Name:          "Dune",
				Status:        "downloading",
				Progress:      50.0,
				DownloadSpeed: 1024 * 1024,
				ETA:           300,
			},
		},
	}
	notif := &mockNotifier{}
	tr := newTestTracker(tc, notif, []int64{100})

	tr.TrackDownload("hash1", "Dune", 2021)

	tr.pollAndUpdate(context.Background())

	sent := notif.getSent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(sent))
	}
	if sent[0].chatID != 100 {
		t.Errorf("expected chatID 100, got %d", sent[0].chatID)
	}
	if !strings.Contains(sent[0].text, "Dune") {
		t.Errorf("expected message to contain 'Dune', got: %s", sent[0].text)
	}
}

func TestPollAndUpdate_EditsExistingMessage(t *testing.T) {
	t.Parallel()

	tc := &mockTorrentClient{
		torrents: []core.Torrent{
			{
				Hash:          "hash1",
				Name:          "Dune",
				Status:        "downloading",
				Progress:      30.0,
				DownloadSpeed: 1024 * 1024,
				ETA:           600,
			},
		},
	}
	notif := &mockNotifier{}
	tr := newTestTracker(tc, notif, []int64{100})

	tr.TrackDownload("hash1", "Dune", 2021)

	// First poll: sends new message.
	tr.pollAndUpdate(context.Background())

	sent := notif.getSent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(sent))
	}

	// Change progress enough to trigger update.
	tc.setTorrents([]core.Torrent{
		{
			Hash:          "hash1",
			Name:          "Dune",
			Status:        "downloading",
			Progress:      55.0,
			DownloadSpeed: 2 * 1024 * 1024,
			ETA:           300,
		},
	})

	// Second poll: should edit existing message.
	tr.pollAndUpdate(context.Background())

	edited := notif.getEdited()
	if len(edited) != 1 {
		t.Fatalf("expected 1 edited message, got %d", len(edited))
	}
	if edited[0].chatID != 100 {
		t.Errorf("expected chatID 100, got %d", edited[0].chatID)
	}
	if edited[0].messageID != 1 {
		t.Errorf("expected messageID 1, got %d", edited[0].messageID)
	}
}

func TestPollAndUpdate_CompletedRemoved(t *testing.T) {
	t.Parallel()

	tc := &mockTorrentClient{
		torrents: []core.Torrent{
			{
				Hash:          "hash1",
				Name:          "Dune",
				Status:        "downloading",
				Progress:      50.0,
				DownloadSpeed: 1024 * 1024,
				ETA:           300,
			},
		},
	}
	notif := &mockNotifier{}
	tr := newTestTracker(tc, notif, []int64{100})

	tr.TrackDownload("hash1", "Dune", 2021)

	// First poll to establish state.
	tr.pollAndUpdate(context.Background())

	// Torrent completes.
	tc.setTorrents([]core.Torrent{
		{
			Hash:     "hash1",
			Name:     "Dune",
			Status:   "seeding",
			Progress: 100.0,
		},
	})

	tr.pollAndUpdate(context.Background())

	if tr.activeCount() != 0 {
		t.Errorf("expected 0 active downloads after completion, got %d", tr.activeCount())
	}
}

func TestSyncActiveDownloads(t *testing.T) {
	t.Parallel()

	tc := &mockTorrentClient{
		torrents: []core.Torrent{
			{
				Hash:          "hash1",
				Name:          "Dune",
				Status:        "downloading",
				Progress:      25.0,
				DownloadSpeed: 1024,
				ETA:           600,
			},
			{
				Hash:     "hash2",
				Name:     "Seeded Movie",
				Status:   "seeding",
				Progress: 100.0,
			},
		},
	}
	notif := &mockNotifier{}
	tr := newTestTracker(tc, notif, []int64{100})

	tr.syncActiveDownloads(context.Background())

	if tr.activeCount() != 1 {
		t.Fatalf("expected 1 active download (only downloading), got %d", tr.activeCount())
	}

	tr.mu.Lock()
	dl, ok := tr.downloads["hash1"]
	tr.mu.Unlock()

	if !ok {
		t.Fatal("expected hash1 to be tracked")
	}
	if dl.title != "Dune" {
		t.Errorf("expected title 'Dune', got %q", dl.title)
	}
}

func TestPollAndUpdate_CompletionSendsFinishedMessage(t *testing.T) {
	t.Parallel()

	tc := &mockTorrentClient{
		torrents: []core.Torrent{
			{
				Hash:          "hash1",
				Name:          "Dune",
				Status:        "downloading",
				Progress:      50.0,
				DownloadSpeed: 1024 * 1024,
				ETA:           300,
			},
		},
	}
	notif := &mockNotifier{}
	tr := newTestTracker(tc, notif, []int64{100})

	tr.TrackDownload("hash1", "Dune", 2021)

	// First poll: sends progress message.
	tr.pollAndUpdate(context.Background())

	sent := notif.getSent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(sent))
	}

	// Torrent finishes.
	tc.setTorrents([]core.Torrent{
		{Hash: "hash1", Name: "Dune", Status: "seeding", Progress: 100.0},
	})

	// Second poll: completed removed first, then sendUpdates sees empty => "Все загрузки завершены!"
	tr.pollAndUpdate(context.Background())

	// Should have edited with completion message.
	edited := notif.getEdited()
	if len(edited) != 1 {
		t.Fatalf("expected 1 edited message, got %d", len(edited))
	}
	if !strings.Contains(edited[0].text, "Все загрузки завершены!") {
		t.Errorf("expected completion message in edit, got: %s", edited[0].text)
	}
}

func TestPollAndUpdate_FinalUpdateWhenZeroActive(t *testing.T) {
	t.Parallel()

	tc := &mockTorrentClient{
		torrents: []core.Torrent{
			{
				Hash:     "hash1",
				Name:     "Dune",
				Status:   "downloading",
				Progress: 50.0,
			},
		},
	}
	notif := &mockNotifier{}
	tr := newTestTracker(tc, notif, []int64{100})

	tr.TrackDownload("hash1", "Dune", 2021)

	// First poll establishes message.
	tr.pollAndUpdate(context.Background())

	// Complete externally and clear torrents.
	tr.CompleteDownload("hash1")
	tc.setTorrents(nil)

	// activeCount is 0, but user messages still exist => should still poll and send final update.
	tr.pollAndUpdate(context.Background())

	// After final update, user messages should be cleared.
	if tr.hasTrackedMessages() {
		t.Error("expected user messages to be cleared after final update")
	}
}

func TestPollAndUpdate_DisappearedDownload(t *testing.T) {
	t.Parallel()

	tc := &mockTorrentClient{
		torrents: []core.Torrent{
			{
				Hash:          "hash1",
				Name:          "Dune",
				Status:        "downloading",
				Progress:      50.0,
				DownloadSpeed: 1024 * 1024,
				ETA:           300,
			},
		},
	}
	notif := &mockNotifier{}
	tr := newTestTracker(tc, notif, []int64{100})

	tr.TrackDownload("hash1", "Dune", 2021)

	// First poll to establish state.
	tr.pollAndUpdate(context.Background())

	// Torrent disappears entirely from client.
	tc.setTorrents(nil)

	tr.pollAndUpdate(context.Background())

	if tr.activeCount() != 0 {
		t.Errorf("expected 0 active downloads after disappearance, got %d", tr.activeCount())
	}
}

func TestSendToUser_EditFailureFallback(t *testing.T) {
	t.Parallel()

	notif := &mockNotifier{}
	tc := &mockTorrentClient{}
	tr := newTestTracker(tc, notif, []int64{100})

	// Simulate an existing user message.
	tr.mu.Lock()
	tr.users[100] = &userProgress{messageID: 42}
	tr.mu.Unlock()

	// Set edit to fail.
	notif.mu.Lock()
	notif.editErr = errTestEdit
	notif.mu.Unlock()

	tr.sendToUser(context.Background(), 100, "test message")

	// Edit failed, so fallback to send.
	sent := notif.getSent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 fallback send, got %d", len(sent))
	}
	if sent[0].text != "test message" {
		t.Errorf("expected 'test message', got %q", sent[0].text)
	}

	// Message ID should be updated to the new one.
	tr.mu.Lock()
	newID := tr.users[100].messageID
	tr.mu.Unlock()

	if newID == 42 {
		t.Error("expected messageID to be updated after fallback send")
	}
}

func TestNotifyGrab(t *testing.T) {
	t.Parallel()

	tc := &mockTorrentClient{}
	notif := &mockNotifier{}
	tr := newTestTracker(tc, notif, []int64{100})

	frontend := &stubFrontend{}
	svc := NewService(frontend, nil, []int64{100}, nil)
	svc.SetTracker(tr)

	handler := NewWebhookHandler(svc, "", nil)

	body := `{
		"eventType": "Grab",
		"movie": {"title": "Dune", "year": 2021},
		"downloadId": "abc123hash"
	}`
	req := httptest.NewRequest(
		http.MethodPost, "/webhooks/radarr", strings.NewReader(body),
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	if tr.activeCount() != 1 {
		t.Fatalf("expected 1 tracked download after grab, got %d", tr.activeCount())
	}

	tr.mu.Lock()
	dl, ok := tr.downloads["abc123hash"]
	tr.mu.Unlock()

	if !ok {
		t.Fatal("expected download with hash 'abc123hash' to be tracked")
	}
	if dl.title != "Dune" {
		t.Errorf("expected title 'Dune', got %q", dl.title)
	}
	if dl.year != 2021 {
		t.Errorf("expected year 2021, got %d", dl.year)
	}
}
