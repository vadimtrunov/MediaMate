package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vadimtrunov/MediaMate/internal/core"
	"github.com/vadimtrunov/MediaMate/internal/metadata/tmdb"
)

// mockMediaServer implements core.MediaServer for testing.
type mockMediaServer struct {
	available bool
	link      string
	availErr  error
	linkErr   error
}

func (m *mockMediaServer) IsAvailable(_ context.Context, _ string) (bool, error) {
	return m.available, m.availErr
}

func (m *mockMediaServer) GetLink(_ context.Context, _ string) (string, error) {
	return m.link, m.linkErr
}

func (m *mockMediaServer) GetLibraryItems(_ context.Context) ([]core.MediaItem, error) {
	return nil, nil
}

func (m *mockMediaServer) Name() string { return "mock" }

// newTestAgent creates an Agent with the given dependencies.
func newTestAgent(
	tmdbClient *tmdb.Client,
	backend core.MediaBackend,
	ms core.MediaServer,
) *Agent {
	llm := &mockLLM{responses: []*core.Response{{Content: "ok", Done: true}}}
	return New(llm, tmdbClient, backend, nil, ms, testLogger())
}

func TestToolSearchMovie_Errors(t *testing.T) {
	t.Parallel()

	t.Run("nil_tmdb", func(t *testing.T) {
		t.Parallel()
		a := newTestAgent(nil, nil, nil)
		_, err := a.toolSearchMovie(context.Background(), map[string]any{"query": "inception"})
		if err == nil || !strings.Contains(err.Error(), "TMDb client not configured") {
			t.Errorf("expected TMDb error, got %v", err)
		}
	})

	t.Run("missing_query", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		tc := tmdb.NewForTest(srv.URL, testLogger())
		a := newTestAgent(tc, nil, nil)
		_, err := a.toolSearchMovie(context.Background(), map[string]any{})
		if err == nil || !strings.Contains(err.Error(), "requires a 'query'") {
			t.Errorf("expected query error, got %v", err)
		}
	})

	t.Run("empty_query", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		tc := tmdb.NewForTest(srv.URL, testLogger())
		a := newTestAgent(tc, nil, nil)
		_, err := a.toolSearchMovie(context.Background(), map[string]any{"query": ""})
		if err == nil || !strings.Contains(err.Error(), "requires a 'query'") {
			t.Errorf("expected query error, got %v", err)
		}
	})

	t.Run("invalid_year", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		tc := tmdb.NewForTest(srv.URL, testLogger())
		a := newTestAgent(tc, nil, nil)
		_, err := a.toolSearchMovie(context.Background(), map[string]any{
			"query": "inception",
			"year":  "abc",
		})
		if err == nil || !strings.Contains(err.Error(), "must be a number") {
			t.Errorf("expected year error, got %v", err)
		}
	})
}

func TestToolSearchMovie_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/search/movie") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := map[string]any{
			"results": []map[string]any{
				{"id": 27205, "title": "Inception", "vote_average": 8.4},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()
	tc := tmdb.NewForTest(srv.URL, testLogger())
	a := newTestAgent(tc, nil, nil)
	result, err := a.toolSearchMovie(context.Background(), map[string]any{"query": "inception"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Inception") {
		t.Errorf("expected Inception in result, got %s", result)
	}
}

func TestToolGetMovieDetails(t *testing.T) {
	t.Parallel()

	t.Run("nil_tmdb", func(t *testing.T) {
		t.Parallel()
		a := newTestAgent(nil, nil, nil)
		_, err := a.toolGetMovieDetails(context.Background(), map[string]any{"tmdb_id": float64(550)})
		if err == nil || !strings.Contains(err.Error(), "TMDb client not configured") {
			t.Errorf("expected TMDb error, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			resp := map[string]any{
				"id": 550, "title": "Fight Club", "runtime": 139,
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()
		tc := tmdb.NewForTest(srv.URL, testLogger())
		a := newTestAgent(tc, nil, nil)
		result, err := a.toolGetMovieDetails(context.Background(), map[string]any{"tmdb_id": float64(550)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "Fight Club") {
			t.Errorf("expected Fight Club in result, got %s", result)
		}
	})
}

func TestToolGetDownloadStatus(t *testing.T) {
	t.Parallel()

	t.Run("nil_backend", func(t *testing.T) {
		t.Parallel()
		a := newTestAgent(nil, nil, nil)
		_, err := a.toolGetDownloadStatus(context.Background(), map[string]any{"radarr_id": float64(42)})
		if err == nil || !strings.Contains(err.Error(), "no media backend configured") {
			t.Errorf("expected backend error, got %v", err)
		}
	})

	t.Run("missing_arg", func(t *testing.T) {
		t.Parallel()
		a := newTestAgent(nil, &mockBackend{}, nil)
		_, err := a.toolGetDownloadStatus(context.Background(), map[string]any{})
		if err == nil || !strings.Contains(err.Error(), "radarr_id is required") {
			t.Errorf("expected arg error, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		backend := &mockBackend{
			statusResp: &core.MediaStatus{ItemID: "42", Status: "downloading", Progress: 75},
		}
		a := newTestAgent(nil, backend, nil)
		result, err := a.toolGetDownloadStatus(context.Background(), map[string]any{"radarr_id": float64(42)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "downloading") {
			t.Errorf("expected downloading in result, got %s", result)
		}
	})
}

func TestToolRecommendSimilar(t *testing.T) {
	t.Parallel()

	t.Run("nil_tmdb", func(t *testing.T) {
		t.Parallel()
		a := newTestAgent(nil, nil, nil)
		_, err := a.toolRecommendSimilar(context.Background(), map[string]any{"tmdb_id": float64(550)})
		if err == nil || !strings.Contains(err.Error(), "TMDb client not configured") {
			t.Errorf("expected TMDb error, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			resp := map[string]any{
				"results": []map[string]any{
					{"id": 680, "title": "Pulp Fiction"},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()
		tc := tmdb.NewForTest(srv.URL, testLogger())
		a := newTestAgent(tc, nil, nil)
		result, err := a.toolRecommendSimilar(context.Background(), map[string]any{"tmdb_id": float64(550)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "Pulp Fiction") {
			t.Errorf("expected Pulp Fiction in result, got %s", result)
		}
	})
}

func TestToolCheckAvailability(t *testing.T) {
	t.Parallel()

	t.Run("nil_server", func(t *testing.T) {
		t.Parallel()
		a := newTestAgent(nil, nil, nil)
		_, err := a.toolCheckAvailability(context.Background(), map[string]any{"title": "Dune"})
		if err == nil || !strings.Contains(err.Error(), "no media server configured") {
			t.Errorf("expected server error, got %v", err)
		}
	})

	t.Run("missing_title", func(t *testing.T) {
		t.Parallel()
		a := newTestAgent(nil, nil, &mockMediaServer{})
		_, err := a.toolCheckAvailability(context.Background(), map[string]any{})
		if err == nil || !strings.Contains(err.Error(), "requires a 'title'") {
			t.Errorf("expected title error, got %v", err)
		}
	})

	t.Run("empty_title", func(t *testing.T) {
		t.Parallel()
		a := newTestAgent(nil, nil, &mockMediaServer{})
		_, err := a.toolCheckAvailability(context.Background(), map[string]any{"title": ""})
		if err == nil || !strings.Contains(err.Error(), "requires a 'title'") {
			t.Errorf("expected title error, got %v", err)
		}
	})

	t.Run("success_available", func(t *testing.T) {
		t.Parallel()
		ms := &mockMediaServer{available: true}
		a := newTestAgent(nil, nil, ms)
		result, err := a.toolCheckAvailability(context.Background(), map[string]any{"title": "Dune"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, `"available":true`) {
			t.Errorf("expected available:true, got %s", result)
		}
	})

	t.Run("server_error", func(t *testing.T) {
		t.Parallel()
		ms := &mockMediaServer{availErr: fmt.Errorf("connection refused")}
		a := newTestAgent(nil, nil, ms)
		_, err := a.toolCheckAvailability(context.Background(), map[string]any{"title": "Dune"})
		if err == nil || !strings.Contains(err.Error(), "check availability failed") {
			t.Errorf("expected availability error, got %v", err)
		}
	})
}

func TestToolGetWatchLink(t *testing.T) {
	t.Parallel()

	t.Run("nil_server", func(t *testing.T) {
		t.Parallel()
		a := newTestAgent(nil, nil, nil)
		_, err := a.toolGetWatchLink(context.Background(), map[string]any{"title": "Dune"})
		if err == nil || !strings.Contains(err.Error(), "no media server configured") {
			t.Errorf("expected server error, got %v", err)
		}
	})

	t.Run("missing_title", func(t *testing.T) {
		t.Parallel()
		a := newTestAgent(nil, nil, &mockMediaServer{})
		_, err := a.toolGetWatchLink(context.Background(), map[string]any{})
		if err == nil || !strings.Contains(err.Error(), "requires a 'title'") {
			t.Errorf("expected title error, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ms := &mockMediaServer{link: "http://jellyfin/Items/123/Play"}
		a := newTestAgent(nil, nil, ms)
		result, err := a.toolGetWatchLink(context.Background(), map[string]any{"title": "Dune"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "http://jellyfin/Items/123/Play") {
			t.Errorf("expected watch link, got %s", result)
		}
	})

	t.Run("server_error", func(t *testing.T) {
		t.Parallel()
		ms := &mockMediaServer{linkErr: fmt.Errorf("not found")}
		a := newTestAgent(nil, nil, ms)
		_, err := a.toolGetWatchLink(context.Background(), map[string]any{"title": "Dune"})
		if err == nil || !strings.Contains(err.Error(), "get watch link failed") {
			t.Errorf("expected link error, got %v", err)
		}
	})
}

func TestToolDownloadMovie_Errors(t *testing.T) {
	t.Parallel()

	t.Run("nil_backend", func(t *testing.T) {
		t.Parallel()
		a := newTestAgent(nil, nil, nil)
		_, err := a.toolDownloadMovie(context.Background(), map[string]any{
			"tmdb_id": float64(1), "title": "X",
		})
		if err == nil || !strings.Contains(err.Error(), "no media backend configured") {
			t.Errorf("expected backend error, got %v", err)
		}
	})

	t.Run("missing_tmdb_id", func(t *testing.T) {
		t.Parallel()
		a := newTestAgent(nil, &mockBackend{}, nil)
		_, err := a.toolDownloadMovie(context.Background(), map[string]any{"title": "X"})
		if err == nil || !strings.Contains(err.Error(), "tmdb_id is required") {
			t.Errorf("expected arg error, got %v", err)
		}
	})

	t.Run("invalid_tmdb_id", func(t *testing.T) {
		t.Parallel()
		a := newTestAgent(nil, &mockBackend{}, nil)
		_, err := a.toolDownloadMovie(context.Background(), map[string]any{
			"tmdb_id": "abc", "title": "X",
		})
		if err == nil || !strings.Contains(err.Error(), "must be a number") {
			t.Errorf("expected number error, got %v", err)
		}
	})
}

func TestToolListDownloads_NilTorrent(t *testing.T) {
	t.Parallel()
	a := newTestAgent(nil, nil, nil)
	_, err := a.toolListDownloads(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "no torrent client configured") {
		t.Errorf("expected torrent error, got %v", err)
	}
}

func TestExecuteTool_UnknownTool(t *testing.T) {
	t.Parallel()
	a := newTestAgent(nil, nil, nil)
	_, err := a.executeTool(context.Background(), core.ToolCall{
		ID: "test", Name: "nonexistent_tool", Arguments: map[string]any{},
	})
	if err == nil || !strings.Contains(err.Error(), "unknown tool") {
		t.Errorf("expected unknown tool error, got %v", err)
	}
}

func TestExtractIntArg_FloatNonInteger(t *testing.T) {
	t.Parallel()
	_, err := extractIntArg(map[string]any{"id": float64(3.14)}, "id")
	if err == nil || !strings.Contains(err.Error(), "must be an integer") {
		t.Errorf("expected integer error, got %v", err)
	}
}
