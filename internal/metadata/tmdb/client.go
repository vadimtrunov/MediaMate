package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/vadimtrunov/MediaMate/internal/httpclient"
)

const (
	defaultBaseURL = "https://api.themoviedb.org/3"
	cacheTTL       = 15 * time.Minute
	imageBaseURL   = "https://image.tmdb.org/t/p/"
)

// Client is a TMDb API v3 client.
type Client struct {
	baseURL string
	apiKey  string
	http    *httpclient.Client
	cache   *cache
	logger  *slog.Logger
}

// New creates a new TMDb client.
func New(apiKey string, logger *slog.Logger) *Client {
	return &Client{
		baseURL: defaultBaseURL,
		apiKey:  apiKey,
		http:    httpclient.New(httpclient.DefaultConfig(), logger),
		cache:   newCache(cacheTTL),
		logger:  logger,
	}
}

// NewForTest creates a TMDb client with a custom base URL for testing.
// Exported because it is used by cross-package tests (e.g. internal/agent).
func NewForTest(baseURL string, logger *slog.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  "test-key",
		http:    httpclient.New(httpclient.DefaultConfig(), logger),
		cache:   newCache(cacheTTL),
		logger:  logger,
	}
}

// SearchMovies searches for movies by title. year=0 means no year filter.
func (c *Client) SearchMovies(ctx context.Context, query string, year int) ([]Movie, error) {
	cacheKey := fmt.Sprintf("search:%s:%d", query, year)
	if cached, ok := c.cache.Get(cacheKey); ok {
		if movies, ok := cached.([]Movie); ok {
			return movies, nil
		}
	}

	params := url.Values{"query": {query}}
	if year > 0 {
		params.Set("year", strconv.Itoa(year))
	}

	var resp searchResponse
	if err := c.get(ctx, "/search/movie", params, &resp); err != nil {
		return nil, fmt.Errorf("search movies: %w", err)
	}

	c.cache.Set(cacheKey, resp.Results)
	return resp.Results, nil
}

// GetMovie retrieves full details for a movie by TMDb ID.
func (c *Client) GetMovie(ctx context.Context, id int) (*MovieDetails, error) {
	cacheKey := fmt.Sprintf("movie:%d", id)
	if cached, ok := c.cache.Get(cacheKey); ok {
		if details, ok := cached.(*MovieDetails); ok {
			return details, nil
		}
	}

	var details MovieDetails
	path := fmt.Sprintf("/movie/%d", id)
	if err := c.get(ctx, path, nil, &details); err != nil {
		return nil, fmt.Errorf("get movie %d: %w", id, err)
	}

	c.cache.Set(cacheKey, &details)
	return &details, nil
}

// GetRecommendations returns recommended movies based on a movie ID.
func (c *Client) GetRecommendations(ctx context.Context, movieID int) ([]Movie, error) {
	cacheKey := fmt.Sprintf("recs:%d", movieID)
	if cached, ok := c.cache.Get(cacheKey); ok {
		if movies, ok := cached.([]Movie); ok {
			return movies, nil
		}
	}

	var resp recommendationsResponse
	path := fmt.Sprintf("/movie/%d/recommendations", movieID)
	if err := c.get(ctx, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("get recommendations for %d: %w", movieID, err)
	}

	c.cache.Set(cacheKey, resp.Results)
	return resp.Results, nil
}

// GetSimilar returns movies similar to a given movie ID.
func (c *Client) GetSimilar(ctx context.Context, movieID int) ([]Movie, error) {
	cacheKey := fmt.Sprintf("similar:%d", movieID)
	if cached, ok := c.cache.Get(cacheKey); ok {
		if movies, ok := cached.([]Movie); ok {
			return movies, nil
		}
	}

	var resp recommendationsResponse
	path := fmt.Sprintf("/movie/%d/similar", movieID)
	if err := c.get(ctx, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("get similar for %d: %w", movieID, err)
	}

	c.cache.Set(cacheKey, resp.Results)
	return resp.Results, nil
}

// PosterURL returns the full URL for a poster path.
func PosterURL(posterPath, size string) string {
	if posterPath == "" {
		return ""
	}
	return imageBaseURL + size + posterPath
}

// get performs an authenticated GET request to the TMDb API and decodes the JSON response.
func (c *Client) get(ctx context.Context, path string, params url.Values, result any) error {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	q := u.Query()
	q.Set("api_key", c.apiKey)
	for k, vs := range params {
		for _, v := range vs {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("tmdb API error %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}
