package notification_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/vadimtrunov/MediaMate/internal/core"
	"github.com/vadimtrunov/MediaMate/internal/notification"
)

// mockFrontend records SendMessage calls.
type mockFrontend struct {
	messages []sentMessage
	sendErr  error
}

type sentMessage struct {
	userID  string
	message string
}

func (m *mockFrontend) Start(_ context.Context) error { return nil }
func (m *mockFrontend) Stop(_ context.Context) error  { return nil }
func (m *mockFrontend) Name() string                  { return "mock" }
func (m *mockFrontend) SendMessage(_ context.Context, userID, msg string) error {
	m.messages = append(m.messages, sentMessage{userID: userID, message: msg})
	return m.sendErr
}

// mockMediaServer returns a fixed link.
type mockMediaServer struct {
	link   string
	getErr error
}

func (m *mockMediaServer) IsAvailable(_ context.Context, _ string) (bool, error) { return true, nil }
func (m *mockMediaServer) GetLink(_ context.Context, _ string) (string, error) {
	return m.link, m.getErr
}

func (m *mockMediaServer) GetLibraryItems(_ context.Context) ([]core.MediaItem, error) {
	return nil, nil
}
func (m *mockMediaServer) Name() string { return "mock" }

func TestNotifyDownloadComplete_WithJellyfinLink(t *testing.T) {
	t.Parallel()
	frontend := &mockFrontend{}
	ms := &mockMediaServer{link: "http://jellyfin:8096/web/index.html#!/details?id=abc123"}
	svc := notification.NewService(frontend, ms, []int64{111, 222}, nil)

	payload := &notification.RadarrWebhookPayload{
		EventType: notification.EventDownload,
		Movie:     notification.RadarrMovie{Title: "Dune", Year: 2021},
	}

	err := svc.NotifyDownloadComplete(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(frontend.messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(frontend.messages))
	}

	for i, uid := range []string{"111", "222"} {
		if frontend.messages[i].userID != uid {
			t.Errorf("message %d: expected userID %s, got %s", i, uid, frontend.messages[i].userID)
		}
	}

	msg := frontend.messages[0].message
	if msg == "" {
		t.Fatal("message should not be empty")
	}
	if !strings.Contains(msg, "Dune") || !strings.Contains(msg, "2021") || !strings.Contains(msg, "jellyfin") {
		t.Errorf("message missing expected content: %s", msg)
	}
}

func TestNotifyDownloadComplete_WithoutJellyfin(t *testing.T) {
	t.Parallel()
	frontend := &mockFrontend{}
	svc := notification.NewService(frontend, nil, []int64{111}, nil)

	payload := &notification.RadarrWebhookPayload{
		EventType: notification.EventDownload,
		Movie:     notification.RadarrMovie{Title: "Avatar", Year: 2009},
	}

	err := svc.NotifyDownloadComplete(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(frontend.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(frontend.messages))
	}

	msg := frontend.messages[0].message
	if strings.Contains(msg, "Jellyfin") || strings.Contains(msg, "http://") {
		t.Errorf("message should not contain link when jellyfin is nil: %s", msg)
	}
	if !strings.Contains(msg, "Avatar") || !strings.Contains(msg, "2009") {
		t.Errorf("message missing expected content: %s", msg)
	}
}

func TestNotifyDownloadComplete_JellyfinError(t *testing.T) {
	t.Parallel()
	frontend := &mockFrontend{}
	ms := &mockMediaServer{getErr: fmt.Errorf("connection refused")}
	svc := notification.NewService(frontend, ms, []int64{111}, nil)

	payload := &notification.RadarrWebhookPayload{
		EventType: notification.EventDownload,
		Movie:     notification.RadarrMovie{Title: "Matrix", Year: 1999},
	}

	err := svc.NotifyDownloadComplete(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(frontend.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(frontend.messages))
	}

	msg := frontend.messages[0].message
	if !strings.Contains(msg, "Matrix") {
		t.Errorf("message should contain movie title: %s", msg)
	}
}

func TestNotifyDownloadComplete_SendError(t *testing.T) {
	t.Parallel()
	frontend := &mockFrontend{sendErr: fmt.Errorf("bot token expired")}
	svc := notification.NewService(frontend, nil, []int64{111}, nil)

	payload := &notification.RadarrWebhookPayload{
		EventType: notification.EventDownload,
		Movie:     notification.RadarrMovie{Title: "Test", Year: 2024},
	}

	err := svc.NotifyDownloadComplete(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error when send fails")
	}
}

func TestNewService_PanicsOnNilFrontend(t *testing.T) {
	t.Parallel()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for nil frontend")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "frontend must not be nil") {
			t.Errorf("unexpected panic value: %v", r)
		}
	}()
	notification.NewService(nil, nil, nil, nil)
}

func TestNotifyDownloadComplete_NoRecipients(t *testing.T) {
	t.Parallel()
	frontend := &mockFrontend{}
	svc := notification.NewService(frontend, nil, []int64{}, nil)

	payload := &notification.RadarrWebhookPayload{
		EventType: notification.EventDownload,
		Movie:     notification.RadarrMovie{Title: "Test", Year: 2024},
	}

	err := svc.NotifyDownloadComplete(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(frontend.messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(frontend.messages))
	}
}

func TestNotifyDownloadComplete_SpecialCharsPreserved(t *testing.T) {
	t.Parallel()
	frontend := &mockFrontend{}
	svc := notification.NewService(frontend, nil, []int64{111}, nil)

	payload := &notification.RadarrWebhookPayload{
		EventType: notification.EventDownload,
		Movie:     notification.RadarrMovie{Title: "Test*Movie_[Special]", Year: 2024},
	}

	err := svc.NotifyDownloadComplete(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := frontend.messages[0].message
	if !strings.Contains(msg, "Test*Movie_[Special]") {
		t.Errorf("plain-text message should preserve special characters verbatim: %s", msg)
	}
}

func TestNotifyDownloadComplete_FallbackToRemoteMovie(t *testing.T) {
	t.Parallel()
	frontend := &mockFrontend{}
	svc := notification.NewService(frontend, nil, []int64{111}, nil)

	payload := &notification.RadarrWebhookPayload{
		EventType:   notification.EventDownload,
		RemoteMovie: notification.RadarrMovie{Title: "Interstellar", Year: 2014},
	}

	err := svc.NotifyDownloadComplete(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := frontend.messages[0].message
	if !strings.Contains(msg, "Interstellar") || !strings.Contains(msg, "2014") {
		t.Errorf("message should use remoteMovie data: %s", msg)
	}
}
