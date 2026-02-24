package jellyfin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/vadimtrunov/MediaMate/internal/core"
	"github.com/vadimtrunov/MediaMate/internal/httpclient"
)

const (
	maxErrorBodyBytes = 4096
	libraryPageSize   = 500
)

// Client implements core.MediaServer for Jellyfin.
type Client struct {
	baseURL string
	apiKey  string
	http    *httpclient.Client
	logger  *slog.Logger
}

var _ core.MediaServer = (*Client)(nil)

// New creates a new Jellyfin client.
func New(baseURL, apiKey string, logger *slog.Logger) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    httpclient.New(httpclient.DefaultConfig(), logger),
		logger:  logger,
	}
}

// IsAvailable checks if a media item is available on the Jellyfin server.
func (c *Client) IsAvailable(ctx context.Context, itemName string) (bool, error) {
	params := url.Values{
		"SearchTerm":       {itemName},
		"IncludeItemTypes": {"Movie"},
		"Recursive":        {"true"},
		"Limit":            {"1"},
	}

	var resp jellyfinItemsResponse
	if err := c.get(ctx, "/Items", params, &resp); err != nil {
		return false, fmt.Errorf("jellyfin search: %w", err)
	}

	return resp.TotalRecordCount > 0, nil
}

// GetLink generates a direct link to watch the media on Jellyfin.
func (c *Client) GetLink(ctx context.Context, itemName string) (string, error) {
	params := url.Values{
		"SearchTerm":       {itemName},
		"IncludeItemTypes": {"Movie"},
		"Recursive":        {"true"},
		"Limit":            {"1"},
	}

	var resp jellyfinItemsResponse
	if err := c.get(ctx, "/Items", params, &resp); err != nil {
		return "", fmt.Errorf("jellyfin search: %w", err)
	}

	if resp.TotalRecordCount == 0 || len(resp.Items) == 0 {
		return "", fmt.Errorf("jellyfin search: item %q not found", itemName)
	}

	link := fmt.Sprintf("%s/web/index.html#!/details?id=%s", c.baseURL, resp.Items[0].ID)
	return link, nil
}

// GetLibraryItems gets all movie items in the Jellyfin library.
func (c *Client) GetLibraryItems(ctx context.Context) ([]core.MediaItem, error) {
	var (
		all        []core.MediaItem
		startIndex int
	)
	for {
		params := url.Values{
			"IncludeItemTypes": {"Movie"},
			"Recursive":        {"true"},
			"Fields":           {"Overview"},
			"Limit":            {fmt.Sprintf("%d", libraryPageSize)},
			"StartIndex":       {fmt.Sprintf("%d", startIndex)},
		}

		var resp jellyfinItemsResponse
		if err := c.get(ctx, "/Items", params, &resp); err != nil {
			return nil, fmt.Errorf("jellyfin list: %w", err)
		}

		for _, item := range resp.Items {
			all = append(all, c.toMediaItem(item))
		}
		startIndex += len(resp.Items)
		if startIndex >= resp.TotalRecordCount || len(resp.Items) == 0 {
			break
		}
	}
	return all, nil
}

// Name returns the server name.
func (c *Client) Name() string { return "jellyfin" }

// get performs an authenticated GET request to the Jellyfin API and decodes the JSON response.
func (c *Client) get(ctx context.Context, path string, params url.Values, result any) error {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if params != nil {
		u.RawQuery = params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-Emby-Token", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		return fmt.Errorf("jellyfin API error %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// toMediaItem converts a Jellyfin item to a core.MediaItem.
func (c *Client) toMediaItem(item jellyfinItem) core.MediaItem {
	var posterURL string
	if _, ok := item.ImageTags["Primary"]; ok {
		q := url.Values{"api_key": {c.apiKey}}
		posterURL = fmt.Sprintf("%s/Items/%s/Images/Primary?%s", c.baseURL, item.ID, q.Encode())
	}

	return core.MediaItem{
		ID:          item.ID,
		Title:       item.Name,
		Year:        item.ProductionYear,
		Type:        "movie",
		Description: item.Overview,
		PosterURL:   posterURL,
		Rating:      item.CommunityRating,
	}
}
