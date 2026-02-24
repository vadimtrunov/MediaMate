package radarr

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vadimtrunov/MediaMate/internal/core"
	"github.com/vadimtrunov/MediaMate/internal/httpclient"
)

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &Client{
		baseURL:        server.URL,
		apiKey:         "test-api-key",
		http:           httpclient.New(httpclient.DefaultConfig(), slog.New(slog.NewTextHandler(io.Discard, nil))),
		qualityProfile: "HD-1080p",
		rootFolder:     "/movies",
		logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestSearch(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/movie/lookup" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("term") != "inception" {
			t.Errorf("unexpected term: %s", r.URL.Query().Get("term"))
		}

		movies := []radarrMovie{{
			ID: 1, Title: "Inception", Year: 2010, TmdbID: 27205,
			Overview: "A thief...",
			Ratings:  radarrRatings{Tmdb: radarrRating{Value: 8.4}},
		}}
		json.NewEncoder(w).Encode(movies)
	}))

	items, err := client.Search(context.Background(), "inception")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Title != "Inception" {
		t.Errorf("expected Inception, got %s", items[0].Title)
	}
	if items[0].Metadata["tmdbId"] != "27205" {
		t.Errorf("expected tmdbId 27205, got %s", items[0].Metadata["tmdbId"])
	}
}

func TestAdd(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v3/qualityprofile":
			json.NewEncoder(w).Encode([]radarrQualityProfile{
				{ID: 4, Name: "HD-1080p"},
			})
		case "/api/v3/movie":
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			var movie radarrMovie
			json.NewDecoder(r.Body).Decode(&movie)
			if movie.TmdbID != 27205 {
				t.Errorf("expected tmdbId 27205, got %d", movie.TmdbID)
			}
			if movie.QualityProfileID != 4 {
				t.Errorf("expected qualityProfileId 4, got %d", movie.QualityProfileID)
			}
			if movie.RootFolderPath != "/movies" {
				t.Errorf("expected rootFolderPath /movies, got %s", movie.RootFolderPath)
			}
			if movie.AddOptions == nil || !movie.AddOptions.SearchForMovie {
				t.Error("expected searchForMovie=true")
			}
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))

	item := core.MediaItem{
		Title: "Inception",
		Year:  2010,
		Type:  "movie",
		Metadata: map[string]string{
			"tmdbId": "27205",
		},
	}
	err := client.Add(context.Background(), item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetStatus(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/movie/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(radarrMovie{
			ID:      42,
			Title:   "Test Movie",
			HasFile: true,
		})
	}))

	status, err := client.GetStatus(context.Background(), "42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "downloaded" {
		t.Errorf("expected downloaded, got %s", status.Status)
	}
	if status.ItemID != "42" {
		t.Errorf("expected itemID 42, got %s", status.ItemID)
	}
}

func TestGetStatusWanted(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(radarrMovie{
			ID:        43,
			HasFile:   false,
			Monitored: true,
		})
	}))

	status, err := client.GetStatus(context.Background(), "43")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "wanted" {
		t.Errorf("expected wanted, got %s", status.Status)
	}
}

func TestListItems(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/movie" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]radarrMovie{
			{ID: 1, Title: "Movie A", TmdbID: 100},
			{ID: 2, Title: "Movie B", TmdbID: 200},
		})
	}))

	items, err := client.ListItems(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestAPIKeyHeader(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			t.Errorf("expected X-Api-Key=test-api-key, got %s", r.Header.Get("X-Api-Key"))
		}
		json.NewEncoder(w).Encode([]radarrMovie{})
	}))

	_, err := client.Search(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveQualityProfile(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v3/qualityprofile":
			json.NewEncoder(w).Encode([]radarrQualityProfile{
				{ID: 1, Name: "Any"},
				{ID: 4, Name: "HD-1080p"},
				{ID: 6, Name: "Ultra-HD"},
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))

	id, err := client.resolveQualityProfileID(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 4 {
		t.Errorf("expected profile ID 4, got %d", id)
	}
}

func TestResolveRootFolderDefault(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode([]radarrRootFolder{
			{ID: 1, Path: "/data/movies"},
		})
	}))
	client.rootFolder = "" // no configured root folder

	folder, err := client.resolveRootFolder(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if folder != "/data/movies" {
		t.Errorf("expected /data/movies, got %s", folder)
	}
}

func TestErrorHandling(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "not found"}`))
	}))

	_, err := client.Search(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should mention 404: %v", err)
	}
}

func TestAddMissingTmdbId(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not make API call")
	}))

	item := core.MediaItem{Title: "Test", Metadata: map[string]string{}}
	err := client.Add(context.Background(), item)
	if err == nil {
		t.Fatal("expected error for missing tmdbId")
	}
}

func TestListQualityProfiles(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v3/qualityprofile" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			t.Errorf("expected X-Api-Key=test-api-key, got %s", r.Header.Get("X-Api-Key"))
		}
		json.NewEncoder(w).Encode([]QualityProfile{
			{ID: 1, Name: "HD-1080p"},
			{ID: 2, Name: "Ultra-HD"},
		})
	}))

	profiles, err := client.ListQualityProfiles(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profiles))
	}
	if profiles[0].ID != 1 || profiles[0].Name != "HD-1080p" {
		t.Errorf("expected first profile {1, HD-1080p}, got {%d, %s}", profiles[0].ID, profiles[0].Name)
	}
	if profiles[1].ID != 2 || profiles[1].Name != "Ultra-HD" {
		t.Errorf("expected second profile {2, Ultra-HD}, got {%d, %s}", profiles[1].ID, profiles[1].Name)
	}
}

func TestListRootFolders(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v3/rootfolder" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			t.Errorf("expected X-Api-Key=test-api-key, got %s", r.Header.Get("X-Api-Key"))
		}
		json.NewEncoder(w).Encode([]RootFolder{
			{ID: 1, Path: "/movies"},
		})
	}))

	folders, err := client.ListRootFolders(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(folders) != 1 {
		t.Fatalf("expected 1 folder, got %d", len(folders))
	}
	if folders[0].ID != 1 || folders[0].Path != "/movies" {
		t.Errorf("expected folder {1, /movies}, got {%d, %s}", folders[0].ID, folders[0].Path)
	}
}

func TestCreateRootFolder(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v3/rootfolder" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			t.Errorf("expected X-Api-Key=test-api-key, got %s", r.Header.Get("X-Api-Key"))
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["path"] != "/movies" {
			t.Errorf("expected path=/movies, got %s", body["path"])
		}

		json.NewEncoder(w).Encode(RootFolder{ID: 1, Path: "/movies"})
	}))

	folder, err := client.CreateRootFolder(context.Background(), "/movies")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if folder.ID != 1 {
		t.Errorf("expected folder ID 1, got %d", folder.ID)
	}
	if folder.Path != "/movies" {
		t.Errorf("expected folder path /movies, got %s", folder.Path)
	}
}

func assertDownloadClientRequest(t *testing.T, r *http.Request) {
	t.Helper()
	if r.Method != http.MethodPost {
		t.Errorf("expected POST, got %s", r.Method)
	}
	if r.URL.Path != "/api/v3/downloadclient" {
		t.Errorf("unexpected path: %s", r.URL.Path)
	}
	if r.Header.Get("X-Api-Key") != "test-api-key" {
		t.Errorf("expected X-Api-Key=test-api-key, got %s", r.Header.Get("X-Api-Key"))
	}
	var cfg DownloadClientConfig
	json.NewDecoder(r.Body).Decode(&cfg)
	if cfg.Name != "Transmission" {
		t.Errorf("expected name Transmission, got %s", cfg.Name)
	}
	if cfg.Implementation != "Transmission" {
		t.Errorf("expected implementation Transmission, got %s", cfg.Implementation)
	}
	if cfg.ConfigContract != "TransmissionSettings" {
		t.Errorf("expected config contract TransmissionSettings, got %s", cfg.ConfigContract)
	}
	if !cfg.Enable {
		t.Error("expected enable=true")
	}
	if cfg.Protocol != "torrent" {
		t.Errorf("expected protocol torrent, got %s", cfg.Protocol)
	}
	if cfg.Priority != 1 {
		t.Errorf("expected priority 1, got %d", cfg.Priority)
	}
	if len(cfg.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(cfg.Fields))
	}
	fields := make(map[string]any, len(cfg.Fields))
	for _, f := range cfg.Fields {
		fields[f.Name] = f.Value
	}
	if fields["host"] != "localhost" {
		t.Errorf("expected host=localhost, got %v", fields["host"])
	}
	if fields["port"] != float64(9091) {
		t.Errorf("expected port=9091, got %v", fields["port"])
	}
}

func TestAddDownloadClient(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertDownloadClientRequest(t, r)
		w.WriteHeader(http.StatusCreated)
	}))

	cfg := DownloadClientConfig{
		Name:           "Transmission",
		Implementation: "Transmission",
		ConfigContract: "TransmissionSettings",
		Enable:         true,
		Protocol:       "torrent",
		Priority:       1,
		Fields: []DownloadClientField{
			{Name: "host", Value: "localhost"},
			{Name: "port", Value: 9091},
		},
	}
	err := client.AddDownloadClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListDownloadClients(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v3/downloadclient" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			t.Errorf("expected X-Api-Key=test-api-key, got %s", r.Header.Get("X-Api-Key"))
		}
		json.NewEncoder(w).Encode([]DownloadClientConfig{
			{
				Name:           "Transmission",
				Implementation: "Transmission",
				ConfigContract: "TransmissionSettings",
				Enable:         true,
				Protocol:       "torrent",
				Priority:       1,
				Fields: []DownloadClientField{
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
