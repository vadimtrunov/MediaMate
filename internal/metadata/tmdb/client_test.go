package tmdb

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vadimtrunov/MediaMate/internal/httpclient"
)

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &Client{
		baseURL: server.URL,
		apiKey:  "test-key",
		http:    httpclient.New(httpclient.DefaultConfig(), slog.New(slog.NewTextHandler(io.Discard, nil))),
		cache:   newCache(cacheTTL),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestSearchMovies(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/movie" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("api_key") != "test-key" {
			t.Error("missing api_key")
		}
		if r.URL.Query().Get("query") != "inception" {
			t.Errorf("unexpected query: %s", r.URL.Query().Get("query"))
		}

		resp := searchResponse{
			Page: 1,
			Results: []Movie{
				{ID: 27205, Title: "Inception", VoteAverage: 8.4, ReleaseDate: "2010-07-16"},
			},
			TotalResults: 1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))

	movies, err := client.SearchMovies(context.Background(), "inception", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(movies) != 1 {
		t.Fatalf("expected 1 movie, got %d", len(movies))
	}
	if movies[0].Title != "Inception" {
		t.Errorf("expected Inception, got %s", movies[0].Title)
	}
	if movies[0].ID != 27205 {
		t.Errorf("expected ID 27205, got %d", movies[0].ID)
	}
}

func TestSearchMoviesWithYear(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("year") != "2010" {
			t.Errorf("expected year=2010, got %s", r.URL.Query().Get("year"))
		}
		json.NewEncoder(w).Encode(searchResponse{Page: 1, Results: []Movie{}})
	}))

	_, err := client.SearchMovies(context.Background(), "inception", 2010)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetMovie(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/movie/550" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		details := MovieDetails{
			ID:          550,
			Title:       "Fight Club",
			Overview:    "A ticking-Loss-bomb insomniac...",
			ReleaseDate: "1999-10-15",
			VoteAverage: 8.4,
			Runtime:     139,
			IMDbID:      "tt0137523",
			Genres:      []Genre{{ID: 18, Name: "Drama"}},
		}
		json.NewEncoder(w).Encode(details)
	}))

	details, err := client.GetMovie(context.Background(), 550)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details.Title != "Fight Club" {
		t.Errorf("expected Fight Club, got %s", details.Title)
	}
	if details.Runtime != 139 {
		t.Errorf("expected runtime 139, got %d", details.Runtime)
	}
}

func TestGetRecommendations(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/movie/550/recommendations" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := recommendationsResponse{
			Page:    1,
			Results: []Movie{{ID: 680, Title: "Pulp Fiction"}},
		}
		json.NewEncoder(w).Encode(resp)
	}))

	movies, err := client.GetRecommendations(context.Background(), 550)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(movies) != 1 {
		t.Fatalf("expected 1 movie, got %d", len(movies))
	}
	if movies[0].Title != "Pulp Fiction" {
		t.Errorf("expected Pulp Fiction, got %s", movies[0].Title)
	}
}

func TestGetSimilar(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/movie/550/similar" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := recommendationsResponse{
			Page:    1,
			Results: []Movie{{ID: 11, Title: "Star Wars"}},
		}
		json.NewEncoder(w).Encode(resp)
	}))

	movies, err := client.GetSimilar(context.Background(), 550)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(movies) != 1 {
		t.Fatalf("expected 1 movie, got %d", len(movies))
	}
}

func TestSearchMoviesCaching(t *testing.T) {
	calls := 0
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		json.NewEncoder(w).Encode(searchResponse{
			Page:    1,
			Results: []Movie{{ID: 1, Title: "Test"}},
		})
	}))

	// First call hits the server
	_, err := client.SearchMovies(context.Background(), "test", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call should hit cache
	_, err = client.SearchMovies(context.Background(), "test", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if calls != 1 {
		t.Errorf("expected 1 server call (cache hit), got %d", calls)
	}
}

func TestAPIError(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status_message": "Invalid API key"}`))
	}))

	_, err := client.SearchMovies(context.Background(), "test", 0)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestPosterURL(t *testing.T) {
	tests := []struct {
		path   string
		size   string
		expect string
	}{
		{"/abc123.jpg", "w500", "https://image.tmdb.org/t/p/w500/abc123.jpg"},
		{"", "w500", ""},
		{"/poster.jpg", "original", "https://image.tmdb.org/t/p/original/poster.jpg"},
	}
	for _, tt := range tests {
		got := PosterURL(tt.path, tt.size)
		if got != tt.expect {
			t.Errorf("PosterURL(%q, %q) = %q, want %q", tt.path, tt.size, got, tt.expect)
		}
	}
}
