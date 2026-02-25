package radarr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/vadimtrunov/MediaMate/internal/core"
	"github.com/vadimtrunov/MediaMate/internal/httpclient"
)

// maxErrBodySize caps the number of bytes read from error response bodies to prevent OOM.
const maxErrBodySize = 1 << 16 // 64 KB

// Client implements core.MediaBackend for Radarr.
type Client struct {
	baseURL        string
	apiKey         string
	http           *httpclient.Client
	qualityProfile string
	rootFolder     string
	logger         *slog.Logger
}

var _ core.MediaBackend = (*Client)(nil)

// New creates a new Radarr client.
func New(baseURL, apiKey, qualityProfile, rootFolder string, logger *slog.Logger) *Client {
	return &Client{
		baseURL:        strings.TrimRight(baseURL, "/"),
		apiKey:         apiKey,
		http:           httpclient.New(httpclient.DefaultConfig(), logger),
		qualityProfile: qualityProfile,
		rootFolder:     rootFolder,
		logger:         logger,
	}
}

// Search searches for movies using Radarr's lookup endpoint.
func (c *Client) Search(ctx context.Context, query string) ([]core.MediaItem, error) {
	params := url.Values{"term": {query}}
	var movies []radarrMovie
	if err := c.get(ctx, "/api/v3/movie/lookup", params, &movies); err != nil {
		return nil, fmt.Errorf("radarr search: %w", err)
	}

	items := make([]core.MediaItem, 0, len(movies))
	for _, m := range movies {
		items = append(items, toMediaItem(m))
	}
	return items, nil
}

// Add adds a movie to the Radarr library.
// Expects item.Metadata["tmdbId"] to be set.
func (c *Client) Add(ctx context.Context, item core.MediaItem) error {
	tmdbIDStr, ok := item.Metadata["tmdbId"]
	if !ok {
		return fmt.Errorf("item.Metadata[\"tmdbId\"] is required")
	}
	tmdbID, err := strconv.Atoi(tmdbIDStr)
	if err != nil {
		return fmt.Errorf("invalid tmdbId %q: %w", tmdbIDStr, err)
	}

	qualityProfileID, err := c.resolveQualityProfileID(ctx)
	if err != nil {
		return fmt.Errorf("resolve quality profile: %w", err)
	}

	rootFolder, err := c.resolveRootFolder(ctx)
	if err != nil {
		return fmt.Errorf("resolve root folder: %w", err)
	}

	movie := radarrMovie{
		Title:            item.Title,
		Year:             item.Year,
		TmdbID:           tmdbID,
		QualityProfileID: qualityProfileID,
		RootFolderPath:   rootFolder,
		Monitored:        true,
		AddOptions:       &radarrAddOpts{SearchForMovie: true},
	}

	if err := c.post(ctx, "/api/v3/movie", movie, nil); err != nil {
		return fmt.Errorf("radarr add movie: %w", err)
	}
	return nil
}

// GetStatus gets the status of a movie by its Radarr ID.
func (c *Client) GetStatus(ctx context.Context, itemID string) (*core.MediaStatus, error) {
	id, err := strconv.Atoi(itemID)
	if err != nil {
		return nil, fmt.Errorf("invalid radarr itemID %q: %w", itemID, err)
	}

	var movie radarrMovie
	path := fmt.Sprintf("/api/v3/movie/%d", id)
	if err := c.get(ctx, path, nil, &movie); err != nil {
		return nil, fmt.Errorf("radarr get status: %w", err)
	}
	return toMediaStatus(movie), nil
}

// ListItems lists all movies in the Radarr library.
func (c *Client) ListItems(ctx context.Context) ([]core.MediaItem, error) {
	var movies []radarrMovie
	if err := c.get(ctx, "/api/v3/movie", nil, &movies); err != nil {
		return nil, fmt.Errorf("radarr list: %w", err)
	}

	items := make([]core.MediaItem, 0, len(movies))
	for _, m := range movies {
		items = append(items, toMediaItem(m))
	}
	return items, nil
}

// Type returns "radarr".
func (c *Client) Type() string { return "radarr" }

// resolveQualityProfileID finds the quality profile ID by name, or defaults to the first available.
func (c *Client) resolveQualityProfileID(ctx context.Context) (int, error) {
	var profiles []QualityProfile
	if err := c.get(ctx, "/api/v3/qualityprofile", nil, &profiles); err != nil {
		return 0, err
	}
	if len(profiles) == 0 {
		return 0, fmt.Errorf("no quality profiles found")
	}

	// If a profile name is configured, find it
	if c.qualityProfile != "" {
		for _, p := range profiles {
			if strings.EqualFold(p.Name, c.qualityProfile) {
				return p.ID, nil
			}
		}
		return 0, fmt.Errorf("quality profile %q not found", c.qualityProfile)
	}

	// Default to first profile
	return profiles[0].ID, nil
}

// resolveRootFolder returns the configured root folder or defaults to the first available.
func (c *Client) resolveRootFolder(ctx context.Context) (string, error) {
	if c.rootFolder != "" {
		return c.rootFolder, nil
	}

	var folders []RootFolder
	if err := c.get(ctx, "/api/v3/rootfolder", nil, &folders); err != nil {
		return "", err
	}
	if len(folders) == 0 {
		return "", fmt.Errorf("no root folders found")
	}
	return folders[0].Path, nil
}

// ListQualityProfiles returns all quality profiles configured in Radarr.
func (c *Client) ListQualityProfiles(ctx context.Context) ([]QualityProfile, error) {
	var profiles []QualityProfile
	if err := c.get(ctx, "/api/v3/qualityprofile", nil, &profiles); err != nil {
		return nil, fmt.Errorf("radarr list quality profiles: %w", err)
	}
	return profiles, nil
}

// ListRootFolders returns all root folders configured in Radarr.
func (c *Client) ListRootFolders(ctx context.Context) ([]RootFolder, error) {
	var folders []RootFolder
	if err := c.get(ctx, "/api/v3/rootfolder", nil, &folders); err != nil {
		return nil, fmt.Errorf("radarr list root folders: %w", err)
	}
	return folders, nil
}

// CreateRootFolder creates a new root folder in Radarr.
func (c *Client) CreateRootFolder(ctx context.Context, path string) (*RootFolder, error) {
	body := map[string]string{"path": path}
	var folder RootFolder
	if err := c.post(ctx, "/api/v3/rootfolder", body, &folder); err != nil {
		return nil, fmt.Errorf("radarr create root folder: %w", err)
	}
	return &folder, nil
}

// AddDownloadClient adds a download client configuration to Radarr.
func (c *Client) AddDownloadClient(ctx context.Context, cfg DownloadClientConfig) error {
	if err := c.post(ctx, "/api/v3/downloadclient", cfg, nil); err != nil {
		return fmt.Errorf("radarr add download client: %w", err)
	}
	return nil
}

// ListDownloadClients returns all download clients configured in Radarr.
func (c *Client) ListDownloadClients(ctx context.Context) ([]DownloadClientConfig, error) {
	var clients []DownloadClientConfig
	if err := c.get(ctx, "/api/v3/downloadclient", nil, &clients); err != nil {
		return nil, fmt.Errorf("radarr list download clients: %w", err)
	}
	return clients, nil
}

// ListNotifications returns all notification/webhook configurations in Radarr.
func (c *Client) ListNotifications(ctx context.Context) ([]NotificationConfig, error) {
	var notifications []NotificationConfig
	if err := c.get(ctx, "/api/v3/notification", nil, &notifications); err != nil {
		return nil, fmt.Errorf("radarr list notifications: %w", err)
	}
	return notifications, nil
}

// AddNotification adds a notification/webhook configuration to Radarr.
func (c *Client) AddNotification(ctx context.Context, cfg NotificationConfig) error {
	if err := c.post(ctx, "/api/v3/notification", cfg, nil); err != nil {
		return fmt.Errorf("radarr add notification: %w", err)
	}
	return nil
}

// get performs an authenticated GET request to the Radarr API and decodes the JSON response.
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
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBodySize))
		return fmt.Errorf("radarr API error %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// post performs an authenticated POST request to the Radarr API with a JSON body.
func (c *Client) post(ctx context.Context, path string, body, result any) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	u := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(jsonBody)), nil
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBodySize))
		return fmt.Errorf("radarr API error %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// toMediaItem converts a Radarr movie to a core.MediaItem.
func toMediaItem(m radarrMovie) core.MediaItem {
	return core.MediaItem{
		ID:          strconv.Itoa(m.ID),
		Title:       m.Title,
		Year:        m.Year,
		Type:        "movie",
		Description: m.Overview,
		PosterURL:   m.RemotePoster,
		Rating:      m.Ratings.Tmdb.Value,
		Metadata: map[string]string{
			"tmdbId": strconv.Itoa(m.TmdbID),
		},
	}
}

// toMediaStatus converts a Radarr movie to a core.MediaStatus.
func toMediaStatus(m radarrMovie) *core.MediaStatus {
	status := "wanted"
	if m.HasFile {
		status = "downloaded"
	}

	return &core.MediaStatus{
		ItemID: strconv.Itoa(m.ID),
		Status: status,
	}
}
