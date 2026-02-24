package qbittorrent

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := New(server.URL, "admin", "password", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client
}

func loginAndHandle(t *testing.T, handler http.HandlerFunc) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/auth/login" {
			w.Write([]byte("Ok."))
			return
		}
		handler(w, r)
	}
}

func TestLogin(t *testing.T) {
	var loginCalled atomic.Bool
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/auth/login" {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			r.ParseForm()
			if r.FormValue("username") != "admin" {
				t.Errorf("expected username=admin, got %s", r.FormValue("username"))
			}
			if r.FormValue("password") != "password" {
				t.Errorf("expected password=password, got %s", r.FormValue("password"))
			}
			loginCalled.Store(true)
			w.Write([]byte("Ok."))
			return
		}
		json.NewEncoder(w).Encode([]qbitTorrent{})
	}))

	_, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !loginCalled.Load() {
		t.Error("login was not called")
	}
}

func TestList(t *testing.T) {
	client := newTestClient(t, loginAndHandle(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/torrents/info" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]qbitTorrent{
			{
				Hash:     "abc123",
				Name:     "Test Movie",
				Size:     1024 * 1024 * 1024,
				Progress: 0.75,
				State:    "downloading",
				DLSpeed:  1024 * 1024,
				ETA:      300,
			},
		})
	}))

	torrents, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(torrents) != 1 {
		t.Fatalf("expected 1 torrent, got %d", len(torrents))
	}
	if torrents[0].Hash != "abc123" {
		t.Errorf("expected hash abc123, got %s", torrents[0].Hash)
	}
	if torrents[0].Progress != 75 {
		t.Errorf("expected progress 75, got %f", torrents[0].Progress)
	}
	if torrents[0].Status != "downloading" {
		t.Errorf("expected downloading, got %s", torrents[0].Status)
	}
}

func TestGetProgress(t *testing.T) {
	client := newTestClient(t, loginAndHandle(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("hashes") != "abc123" {
			t.Errorf("expected hashes=abc123, got %s", r.URL.Query().Get("hashes"))
		}
		json.NewEncoder(w).Encode([]qbitTorrent{
			{
				Hash:       "abc123",
				Progress:   0.5,
				Downloaded: 512 * 1024 * 1024,
				TotalSize:  1024 * 1024 * 1024,
				DLSpeed:    2 * 1024 * 1024,
				ETA:        256,
			},
		})
	}))

	progress, err := client.GetProgress(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if progress.Progress != 50 {
		t.Errorf("expected progress 50, got %f", progress.Progress)
	}
	if progress.Downloaded != 512*1024*1024 {
		t.Errorf("unexpected downloaded: %d", progress.Downloaded)
	}
}

func TestGetProgressNotFound(t *testing.T) {
	client := newTestClient(t, loginAndHandle(t, func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode([]qbitTorrent{})
	}))

	_, err := client.GetProgress(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing torrent")
	}
}

func TestPause(t *testing.T) {
	client := newTestClient(t, loginAndHandle(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/torrents/pause" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		r.ParseForm()
		if r.FormValue("hashes") != "abc123" {
			t.Errorf("expected hashes=abc123, got %s", r.FormValue("hashes"))
		}
		w.WriteHeader(http.StatusOK)
	}))

	err := client.Pause(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResume(t *testing.T) {
	client := newTestClient(t, loginAndHandle(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/torrents/resume" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))

	err := client.Resume(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemove(t *testing.T) {
	client := newTestClient(t, loginAndHandle(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/torrents/delete" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		r.ParseForm()
		if r.FormValue("deleteFiles") != "true" {
			t.Errorf("expected deleteFiles=true, got %s", r.FormValue("deleteFiles"))
		}
		w.WriteHeader(http.StatusOK)
	}))

	err := client.Remove(context.Background(), "abc123", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSessionExpiry(t *testing.T) {
	var loginCalls atomic.Int32
	var requestCalls atomic.Int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/auth/login" {
			loginCalls.Add(1)
			w.Write([]byte("Ok."))
			return
		}
		n := requestCalls.Add(1)
		if n == 1 {
			// First request: simulate session expiry
			w.WriteHeader(http.StatusForbidden)
			return
		}
		// Second request after re-login: success
		json.NewEncoder(w).Encode([]qbitTorrent{})
	}))

	_, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loginCalls.Load() != 2 {
		t.Errorf("expected 2 login calls (initial + re-login), got %d", loginCalls.Load())
	}
}

func TestStateMapping(t *testing.T) {
	tests := []struct {
		state  string
		expect string
	}{
		{"downloading", "downloading"},
		{"forcedDL", "downloading"},
		{"stalledDL", "downloading"},
		{"metaDL", "downloading"},
		{"queuedDL", "downloading"},
		{"uploading", "seeding"},
		{"forcedUP", "seeding"},
		{"stalledUP", "seeding"},
		{"pausedDL", "paused"},
		{"pausedUP", "paused"},
		{"error", "error"},
		{"missingFiles", "error"},
		{"unknown", "error"},
	}
	for _, tt := range tests {
		got := mapState(tt.state)
		if got != tt.expect {
			t.Errorf("mapState(%q) = %q, want %q", tt.state, got, tt.expect)
		}
	}
}

func TestETAInfinity(t *testing.T) {
	client := newTestClient(t, loginAndHandle(t, func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode([]qbitTorrent{
			{Hash: "abc", ETA: etaInfinity, Progress: 0.1, State: "downloading"},
		})
	}))

	torrents, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if torrents[0].ETA != 0 {
		t.Errorf("expected ETA 0 for infinity, got %d", torrents[0].ETA)
	}
}

func TestGetPreferences(t *testing.T) {
	client := newTestClient(t, loginAndHandle(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/app/preferences" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(Preferences{
			SavePath:        "/downloads",
			TempPath:        "/downloads/temp",
			TempPathEnabled: true,
			WebUIPort:       8080,
		})
	}))

	prefs, err := client.GetPreferences(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prefs.SavePath != "/downloads" {
		t.Errorf("expected save_path /downloads, got %s", prefs.SavePath)
	}
	if prefs.TempPath != "/downloads/temp" {
		t.Errorf("expected temp_path /downloads/temp, got %s", prefs.TempPath)
	}
	if !prefs.TempPathEnabled {
		t.Error("expected temp_path_enabled true")
	}
	if prefs.WebUIPort != 8080 {
		t.Errorf("expected web_ui_port 8080, got %d", prefs.WebUIPort)
	}
}

func TestSetPreferences(t *testing.T) {
	client := newTestClient(t, loginAndHandle(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/app/setPreferences" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		r.ParseForm()
		jsonStr := r.FormValue("json")
		if jsonStr == "" {
			t.Fatal("expected json form value, got empty")
		}
		var got map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &got); err != nil {
			t.Fatalf("failed to parse json form value: %v", err)
		}
		if got["save_path"] != "/new/downloads" {
			t.Errorf("expected save_path /new/downloads, got %v", got["save_path"])
		}
		if got["web_ui_port"] != float64(9090) {
			t.Errorf("expected web_ui_port 9090, got %v", got["web_ui_port"])
		}
		w.WriteHeader(http.StatusOK)
	}))

	err := client.SetPreferences(context.Background(), map[string]any{
		"save_path":   "/new/downloads",
		"web_ui_port": 9090,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetPreferencesNilMap(t *testing.T) {
	client := newTestClient(t, loginAndHandle(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/app/setPreferences" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		r.ParseForm()
		jsonStr := r.FormValue("json")
		if jsonStr != "{}" {
			t.Errorf("expected empty JSON object for nil prefs, got %s", jsonStr)
		}
		w.WriteHeader(http.StatusOK)
	}))

	if err := client.SetPreferences(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
