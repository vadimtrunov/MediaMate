package prowlarr

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vadimtrunov/MediaMate/internal/httpclient"
)

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &Client{
		baseURL: server.URL,
		apiKey:  "test-api-key",
		http:    httpclient.New(httpclient.DefaultConfig(), logger),
	}
}

func TestAddApplication(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/applications" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			t.Errorf("expected X-Api-Key=test-api-key, got %s", r.Header.Get("X-Api-Key"))
		}

		var app Application
		if err := json.NewDecoder(r.Body).Decode(&app); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if app.Name != "Radarr" {
			t.Errorf("expected name Radarr, got %s", app.Name)
		}
		if app.Implementation != "Radarr" {
			t.Errorf("expected implementation Radarr, got %s", app.Implementation)
		}
		if app.SyncLevel != "fullSync" {
			t.Errorf("expected syncLevel fullSync, got %s", app.SyncLevel)
		}

		w.WriteHeader(http.StatusCreated)
	}))

	app := Application{
		Name:           "Radarr",
		Implementation: "Radarr",
		ConfigContract: "RadarrSettings",
		SyncLevel:      "fullSync",
		Fields: []Field{
			{Name: "prowlarrUrl", Value: "http://prowlarr:9696"},
			{Name: "baseUrl", Value: "http://radarr:7878"},
			{Name: "apiKey", Value: "radarr-api-key"},
		},
	}
	err := client.AddApplication(context.Background(), app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListApplications(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/applications" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			t.Errorf("expected X-Api-Key=test-api-key, got %s", r.Header.Get("X-Api-Key"))
		}

		json.NewEncoder(w).Encode([]Application{
			{
				ID:             1,
				Name:           "Radarr",
				Implementation: "Radarr",
				ConfigContract: "RadarrSettings",
				SyncLevel:      "fullSync",
				Fields: []Field{
					{Name: "baseUrl", Value: "http://radarr:7878"},
				},
			},
			{
				ID:             2,
				Name:           "Sonarr",
				Implementation: "Sonarr",
				ConfigContract: "SonarrSettings",
				SyncLevel:      "fullSync",
				Fields: []Field{
					{Name: "baseUrl", Value: "http://sonarr:8989"},
				},
			},
		})
	}))

	apps, err := client.ListApplications(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("expected 2 apps, got %d", len(apps))
	}
	if apps[0].Name != "Radarr" {
		t.Errorf("expected first app Radarr, got %s", apps[0].Name)
	}
	if apps[1].Name != "Sonarr" {
		t.Errorf("expected second app Sonarr, got %s", apps[1].Name)
	}
	if apps[0].ID != 1 {
		t.Errorf("expected first app ID 1, got %d", apps[0].ID)
	}
}

func TestAddDownloadClient(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/downloadclient" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			t.Errorf("expected X-Api-Key=test-api-key, got %s", r.Header.Get("X-Api-Key"))
		}

		var dc DownloadClient
		if err := json.NewDecoder(r.Body).Decode(&dc); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if dc.Name != "Transmission" {
			t.Errorf("expected name Transmission, got %s", dc.Name)
		}
		if dc.Implementation != "Transmission" {
			t.Errorf("expected implementation Transmission, got %s", dc.Implementation)
		}
		if dc.Protocol != "torrent" {
			t.Errorf("expected protocol torrent, got %s", dc.Protocol)
		}
		if !dc.Enable {
			t.Error("expected download client to be enabled")
		}

		w.WriteHeader(http.StatusCreated)
	}))

	dc := DownloadClient{
		Name:           "Transmission",
		Implementation: "Transmission",
		ConfigContract: "TransmissionSettings",
		Enable:         true,
		Protocol:       "torrent",
		Priority:       1,
		Fields: []Field{
			{Name: "host", Value: "localhost"},
			{Name: "port", Value: 9091},
		},
	}
	err := client.AddDownloadClient(context.Background(), dc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListDownloadClients(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/downloadclient" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			t.Errorf("expected X-Api-Key=test-api-key, got %s", r.Header.Get("X-Api-Key"))
		}

		json.NewEncoder(w).Encode([]DownloadClient{
			{
				ID:             1,
				Name:           "Transmission",
				Implementation: "Transmission",
				ConfigContract: "TransmissionSettings",
				Enable:         true,
				Protocol:       "torrent",
				Priority:       1,
				Fields: []Field{
					{Name: "host", Value: "localhost"},
					{Name: "port", Value: float64(9091)},
				},
			},
		})
	}))

	clients, err := client.ListDownloadClients(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(clients))
	}
	if clients[0].Name != "Transmission" {
		t.Errorf("expected client name Transmission, got %s", clients[0].Name)
	}
	if clients[0].Implementation != "Transmission" {
		t.Errorf("expected implementation Transmission, got %s", clients[0].Implementation)
	}
	if !clients[0].Enable {
		t.Error("expected client to be enabled")
	}
	if clients[0].Protocol != "torrent" {
		t.Errorf("expected protocol torrent, got %s", clients[0].Protocol)
	}
}

func TestAddIndexerProxy(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/indexerproxy" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			t.Errorf("expected X-Api-Key=test-api-key, got %s", r.Header.Get("X-Api-Key"))
		}

		var proxy IndexerProxy
		if err := json.NewDecoder(r.Body).Decode(&proxy); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if proxy.Name != "FlareSolverr" {
			t.Errorf("expected name FlareSolverr, got %s", proxy.Name)
		}
		if proxy.Implementation != "FlareSolverr" {
			t.Errorf("expected implementation FlareSolverr, got %s", proxy.Implementation)
		}

		w.WriteHeader(http.StatusCreated)
	}))

	proxy := IndexerProxy{
		Name:           "FlareSolverr",
		Implementation: "FlareSolverr",
		ConfigContract: "FlareSolverrSettings",
		Fields: []Field{
			{Name: "host", Value: "http://flaresolverr:8191"},
		},
	}
	err := client.AddIndexerProxy(context.Background(), proxy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListIndexerProxies(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/indexerproxy" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			t.Errorf("expected X-Api-Key=test-api-key, got %s", r.Header.Get("X-Api-Key"))
		}

		json.NewEncoder(w).Encode([]IndexerProxy{
			{
				ID:             1,
				Name:           "FlareSolverr",
				Implementation: "FlareSolverr",
				ConfigContract: "FlareSolverrSettings",
				Fields: []Field{
					{Name: "host", Value: "http://flaresolverr:8191"},
				},
			},
			{
				ID:             2,
				Name:           "Socks5Proxy",
				Implementation: "Socks5",
				ConfigContract: "Socks5Settings",
				Fields: []Field{
					{Name: "host", Value: "proxy.example.com"},
					{Name: "port", Value: float64(1080)},
				},
			},
		})
	}))

	proxies, err := client.ListIndexerProxies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(proxies) != 2 {
		t.Fatalf("expected 2 proxies, got %d", len(proxies))
	}
	if proxies[0].Name != "FlareSolverr" {
		t.Errorf("expected first proxy FlareSolverr, got %s", proxies[0].Name)
	}
	if proxies[1].Name != "Socks5Proxy" {
		t.Errorf("expected second proxy Socks5Proxy, got %s", proxies[1].Name)
	}
	if proxies[0].ID != 1 {
		t.Errorf("expected first proxy ID 1, got %d", proxies[0].ID)
	}
}

func TestErrorHandlingGET(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "not found"}`))
	}))

	_, err := client.ListApplications(context.Background())
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should mention 404: %v", err)
	}
}

func TestErrorHandlingPOST(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "internal server error"}`))
	}))

	err := client.AddApplication(context.Background(), Application{Name: "Test"})
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention 500: %v", err)
	}
}
