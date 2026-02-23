package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/vadimtrunov/MediaMate/internal/core"
)

func (a *Agent) toolSearchMovie(ctx context.Context, args map[string]any) (string, error) {
	if a.tmdb == nil {
		return "", fmt.Errorf("TMDb client not configured")
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("search_movie requires a 'query' string argument")
	}

	year := 0
	if _, ok := args["year"]; ok {
		var err error
		year, err = extractIntArg(args, "year")
		if err != nil {
			return "", err
		}
	}

	movies, err := a.tmdb.SearchMovies(ctx, query, year)
	if err != nil {
		return "", fmt.Errorf("tmdb search failed: %w", err)
	}

	result, err := json.Marshal(movies)
	if err != nil {
		return "", fmt.Errorf("marshal search results: %w", err)
	}
	return string(result), nil
}

func (a *Agent) toolGetMovieDetails(ctx context.Context, args map[string]any) (string, error) {
	if a.tmdb == nil {
		return "", fmt.Errorf("TMDb client not configured")
	}

	tmdbID, err := extractIntArg(args, "tmdb_id")
	if err != nil {
		return "", err
	}

	details, err := a.tmdb.GetMovie(ctx, tmdbID)
	if err != nil {
		return "", fmt.Errorf("tmdb get movie failed: %w", err)
	}

	result, err := json.Marshal(details)
	if err != nil {
		return "", fmt.Errorf("marshal movie details: %w", err)
	}
	return string(result), nil
}

func (a *Agent) toolDownloadMovie(ctx context.Context, args map[string]any) (string, error) {
	if a.backend == nil {
		return "", fmt.Errorf("no media backend configured for downloading")
	}

	tmdbID, err := extractIntArg(args, "tmdb_id")
	if err != nil {
		return "", err
	}

	title, _ := args["title"].(string)

	item := core.MediaItem{
		Title: title,
		Type:  "movie",
		Metadata: map[string]string{
			"tmdbId": strconv.Itoa(tmdbID),
		},
	}

	if err := a.backend.Add(ctx, item); err != nil {
		return "", fmt.Errorf("failed to add movie: %w", err)
	}

	return fmt.Sprintf(`{"status":"added","title":%q,"tmdb_id":%d}`, title, tmdbID), nil
}

func (a *Agent) toolGetDownloadStatus(ctx context.Context, args map[string]any) (string, error) {
	if a.backend == nil {
		return "", fmt.Errorf("no media backend configured")
	}

	radarrID, ok := args["radarr_id"].(string)
	if !ok || radarrID == "" {
		return "", fmt.Errorf("get_download_status requires a 'radarr_id' string argument")
	}

	status, err := a.backend.GetStatus(ctx, radarrID)
	if err != nil {
		return "", fmt.Errorf("get status failed: %w", err)
	}

	result, err := json.Marshal(status)
	if err != nil {
		return "", fmt.Errorf("marshal status: %w", err)
	}
	return string(result), nil
}

func (a *Agent) toolRecommendSimilar(ctx context.Context, args map[string]any) (string, error) {
	if a.tmdb == nil {
		return "", fmt.Errorf("TMDb client not configured")
	}

	tmdbID, err := extractIntArg(args, "tmdb_id")
	if err != nil {
		return "", err
	}

	movies, err := a.tmdb.GetRecommendations(ctx, tmdbID)
	if err != nil {
		return "", fmt.Errorf("tmdb recommendations failed: %w", err)
	}

	result, err := json.Marshal(movies)
	if err != nil {
		return "", fmt.Errorf("marshal recommendations: %w", err)
	}
	return string(result), nil
}

func (a *Agent) toolListDownloads(ctx context.Context, _ map[string]any) (string, error) {
	if a.torrent == nil {
		return "", fmt.Errorf("no torrent client configured")
	}

	torrents, err := a.torrent.List(ctx)
	if err != nil {
		return "", fmt.Errorf("list torrents failed: %w", err)
	}

	result, err := json.Marshal(torrents)
	if err != nil {
		return "", fmt.Errorf("marshal torrents: %w", err)
	}
	return string(result), nil
}

func extractIntArg(args map[string]any, key string) (int, error) {
	val, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("%s is required", key)
	}
	switch v := val.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case string:
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("%s must be a number: %w", key, err)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("%s must be a number, got %T", key, val)
	}
}
