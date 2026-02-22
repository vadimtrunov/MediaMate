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
