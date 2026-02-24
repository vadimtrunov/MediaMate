package jellyfin

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
		baseURL: server.URL,
		apiKey:  "test-api-key",
		http:    httpclient.New(httpclient.DefaultConfig(), slog.New(slog.NewTextHandler(io.Discard, nil))),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestIsAvailableFound(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Items" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("SearchTerm") != "Inception" {
			t.Errorf("unexpected SearchTerm: %s", r.URL.Query().Get("SearchTerm"))
		}
		if r.Header.Get("X-Emby-Token") != "test-api-key" {
			t.Errorf("expected X-Emby-Token=test-api-key, got %s", r.Header.Get("X-Emby-Token"))
		}

		json.NewEncoder(w).Encode(jellyfinItemsResponse{
			Items: []jellyfinItem{
				{ID: "abc-123", Name: "Inception", ProductionYear: 2010},
			},
			TotalRecordCount: 1,
		})
	}))

	available, err := client.IsAvailable(context.Background(), "Inception")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !available {
		t.Errorf("expected available=true, got false")
	}
}

func TestIsAvailableNotFound(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(jellyfinItemsResponse{
			Items:            []jellyfinItem{},
			TotalRecordCount: 0,
		})
	}))

	available, err := client.IsAvailable(context.Background(), "NonExistentMovie")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if available {
		t.Errorf("expected available=false, got true")
	}
}

func TestGetLink(t *testing.T) {
	var serverURL string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(jellyfinItemsResponse{
			Items: []jellyfinItem{
				{ID: "abc-123", Name: "Inception", ProductionYear: 2010},
			},
			TotalRecordCount: 1,
		})
	}))
	serverURL = client.baseURL

	link, err := client.GetLink(context.Background(), "Inception")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := serverURL + "/web/index.html#!/details?id=abc-123"
	if link != expected {
		t.Errorf("expected link %s, got %s", expected, link)
	}
}

func TestGetLinkNotFound(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(jellyfinItemsResponse{
			Items:            []jellyfinItem{},
			TotalRecordCount: 0,
		})
	}))

	_, err := client.GetLink(context.Background(), "NonExistentMovie")
	if err == nil {
		t.Fatal("expected error for not found item")
	}
}

func TestGetLibraryItems(t *testing.T) {
	var serverURL string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Items" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		json.NewEncoder(w).Encode(jellyfinItemsResponse{
			Items: []jellyfinItem{
				{
					ID:              "item-1",
					Name:            "Inception",
					ProductionYear:  2010,
					Overview:        "A thief who steals corporate secrets.",
					CommunityRating: 8.4,
					ImageTags:       map[string]string{"Primary": "tag-abc"},
				},
				{
					ID:              "item-2",
					Name:            "The Matrix",
					ProductionYear:  1999,
					Overview:        "A computer hacker learns about reality.",
					CommunityRating: 8.7,
					ImageTags:       map[string]string{},
				},
			},
			TotalRecordCount: 2,
		})
	}))
	serverURL = client.baseURL

	items, err := client.GetLibraryItems(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	assertLibraryItem(t, items[0], "Inception", 2010, 8.4, serverURL+"/Items/item-1/Images/Primary")
	assertLibraryItem(t, items[1], "The Matrix", 1999, 8.7, "")
}

func assertLibraryItem(t *testing.T, item core.MediaItem, title string, year int, rating float64, posterURL string) {
	t.Helper()
	if item.Title != title {
		t.Errorf("expected title %s, got %s", title, item.Title)
	}
	if item.Year != year {
		t.Errorf("expected year %d, got %d", year, item.Year)
	}
	if item.Rating != rating {
		t.Errorf("expected rating %v, got %v", rating, item.Rating)
	}
	if item.PosterURL != posterURL {
		t.Errorf("expected PosterURL %s, got %s", posterURL, item.PosterURL)
	}
}

func TestAuthHeader(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Emby-Token") != "test-api-key" {
			t.Errorf("expected X-Emby-Token=test-api-key, got %s", r.Header.Get("X-Emby-Token"))
		}
		json.NewEncoder(w).Encode(jellyfinItemsResponse{
			Items:            []jellyfinItem{},
			TotalRecordCount: 0,
		})
	}))

	_, err := client.IsAvailable(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestErrorHandling(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "not found"}`))
	}))

	_, err := client.IsAvailable(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should mention 404: %v", err)
	}
}
