package notification_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vadimtrunov/MediaMate/internal/notification"
)

func newTestHandler(t *testing.T, secret string) (*notification.WebhookHandler, *mockFrontend) {
	t.Helper()
	frontend := &mockFrontend{}
	svc := notification.NewService(frontend, nil, []int64{111}, nil)
	handler := notification.NewWebhookHandler(svc, secret, nil)
	return handler, frontend
}

func TestWebhookHandler_DownloadEvent(t *testing.T) {
	t.Parallel()
	handler, frontend := newTestHandler(t, "")

	body := `{"eventType":"Download","movie":{"title":"Dune","year":2021}}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/radarr", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if len(frontend.messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(frontend.messages))
	}
}

func TestWebhookHandler_NonDownloadEvent(t *testing.T) {
	t.Parallel()
	handler, frontend := newTestHandler(t, "")

	body := `{"eventType":"Grab","movie":{"title":"Dune","year":2021}}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/radarr", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if len(frontend.messages) != 0 {
		t.Errorf("expected 0 messages for non-download event, got %d", len(frontend.messages))
	}
}

func TestWebhookHandler_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	handler, _ := newTestHandler(t, "")

	req := httptest.NewRequest(http.MethodGet, "/webhooks/radarr", http.NoBody)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestWebhookHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	handler, _ := newTestHandler(t, "")

	req := httptest.NewRequest(http.MethodPost, "/webhooks/radarr", strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestWebhookHandler_ValidSecret(t *testing.T) {
	t.Parallel()
	handler, frontend := newTestHandler(t, "my-secret")

	body := `{"eventType":"Download","movie":{"title":"Test","year":2024}}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/radarr", strings.NewReader(body))
	req.Header.Set("X-Webhook-Secret", "my-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if len(frontend.messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(frontend.messages))
	}
}

func TestWebhookHandler_InvalidSecret(t *testing.T) {
	t.Parallel()
	handler, frontend := newTestHandler(t, "my-secret")

	body := `{"eventType":"Download","movie":{"title":"Test","year":2024}}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/radarr", strings.NewReader(body))
	req.Header.Set("X-Webhook-Secret", "wrong-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
	if len(frontend.messages) != 0 {
		t.Errorf("expected 0 messages on invalid secret, got %d", len(frontend.messages))
	}
}

func TestWebhookHandler_MissingSecret(t *testing.T) {
	t.Parallel()
	handler, frontend := newTestHandler(t, "my-secret")

	body := `{"eventType":"Download","movie":{"title":"Test","year":2024}}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/radarr", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
	if len(frontend.messages) != 0 {
		t.Errorf("expected 0 messages on missing secret, got %d", len(frontend.messages))
	}
}
