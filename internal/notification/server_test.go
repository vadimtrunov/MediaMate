package notification_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/vadimtrunov/MediaMate/internal/notification"
)

func TestServer_StartAndStop(t *testing.T) {
	t.Parallel()

	frontend := &mockFrontend{}
	svc := notification.NewService(frontend, nil, []int64{111}, nil)
	handler := notification.NewWebhookHandler(svc, "", nil)
	srv := notification.NewServer(0, handler, nil) // port 0 = random

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	select {
	case <-srv.Ready():
	case err := <-errCh:
		t.Fatalf("server failed to start: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not become ready within timeout")
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("server returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop within timeout")
	}
}

func TestServer_AddrBeforeStart(t *testing.T) {
	t.Parallel()

	frontend := &mockFrontend{}
	svc := notification.NewService(frontend, nil, nil, nil)
	handler := notification.NewWebhookHandler(svc, "", nil)
	srv := notification.NewServer(0, handler, nil)

	if addr := srv.Addr(); addr != "" {
		t.Errorf("expected empty addr before start, got %q", addr)
	}
}

func TestServer_HealthEndpoint(t *testing.T) {
	t.Parallel()

	frontend := &mockFrontend{}
	svc := notification.NewService(frontend, nil, nil, nil)
	handler := notification.NewWebhookHandler(svc, "", nil)
	srv := notification.NewServer(0, handler, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	select {
	case <-srv.Ready():
	case err := <-errCh:
		t.Fatalf("server failed to start: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not become ready within timeout")
	}

	addr := srv.Addr()
	if addr == "" {
		t.Fatal("server address should not be empty after start")
	}

	healthURL := fmt.Sprintf("http://%s/health", addr)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, http.NoBody)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req) //nolint:gosec // test-only ephemeral URL
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("server returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop within timeout")
	}
}
