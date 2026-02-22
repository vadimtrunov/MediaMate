package httpclient

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestDo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := New(DefaultConfig(), testLogger())
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDo_RetryOn500(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	cfg := Config{
		MaxRetries: 3,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
		Timeout:    5 * time.Second,
	}
	client := New(cfg, testLogger())
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if calls.Load() != 3 {
		t.Errorf("expected 3 calls, got %d", calls.Load())
	}
}

func TestDo_RetryOn429WithRetryAfter(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := Config{
		MaxRetries: 3,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
		Timeout:    5 * time.Second,
	}
	client := New(cfg, testLogger())
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDo_ExhaustedRetries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := Config{
		MaxRetries: 2,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
		Timeout:    5 * time.Second,
	}
	client := New(cfg, testLogger())
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)

	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
		t.Fatal("expected error after exhausted retries")
	}
}

func TestDo_NoRetryOn400(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	cfg := Config{
		MaxRetries: 3,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
		Timeout:    5 * time.Second,
	}
	client := New(cfg, testLogger())
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	if calls.Load() != 1 {
		t.Errorf("expected 1 call (no retry), got %d", calls.Load())
	}
}

func TestDo_PostWithBody(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		n := calls.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if string(body) != "test body" {
			t.Errorf("body not replayed: got %q", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := Config{
		MaxRetries: 3,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
		Timeout:    5 * time.Second,
	}
	client := New(cfg, testLogger())

	bodyStr := "test body"
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, strings.NewReader(bodyStr))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(bodyStr)), nil
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		code   int
		expect bool
	}{
		{200, false},
		{201, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
	}
	for _, tt := range tests {
		if got := shouldRetry(tt.code); got != tt.expect {
			t.Errorf("shouldRetry(%d) = %v, want %v", tt.code, got, tt.expect)
		}
	}
}
