package prowlarr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/vadimtrunov/MediaMate/internal/httpclient"
)

// Client is the Prowlarr API v1 client.
type Client struct {
	baseURL string
	apiKey  string
	http    *httpclient.Client
	logger  *slog.Logger
}

// New creates a new Prowlarr client.
func New(baseURL, apiKey string, logger *slog.Logger) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    httpclient.New(httpclient.DefaultConfig(), logger),
		logger:  logger,
	}
}

// AddApplication adds an application (e.g., Radarr/Sonarr) to Prowlarr.
func (c *Client) AddApplication(ctx context.Context, app Application) error {
	if err := c.post(ctx, "/api/v1/applications", app, nil); err != nil {
		return fmt.Errorf("prowlarr add application: %w", err)
	}
	return nil
}

// ListApplications returns all applications configured in Prowlarr.
func (c *Client) ListApplications(ctx context.Context) ([]Application, error) {
	var apps []Application
	if err := c.get(ctx, "/api/v1/applications", nil, &apps); err != nil {
		return nil, fmt.Errorf("prowlarr list applications: %w", err)
	}
	return apps, nil
}

// AddDownloadClient adds a download client configuration to Prowlarr.
func (c *Client) AddDownloadClient(ctx context.Context, dc DownloadClient) error {
	if err := c.post(ctx, "/api/v1/downloadclient", dc, nil); err != nil {
		return fmt.Errorf("prowlarr add download client: %w", err)
	}
	return nil
}

// ListDownloadClients returns all download clients configured in Prowlarr.
func (c *Client) ListDownloadClients(ctx context.Context) ([]DownloadClient, error) {
	var clients []DownloadClient
	if err := c.get(ctx, "/api/v1/downloadclient", nil, &clients); err != nil {
		return nil, fmt.Errorf("prowlarr list download clients: %w", err)
	}
	return clients, nil
}

// AddIndexerProxy adds an indexer proxy (e.g., FlareSolverr) to Prowlarr.
func (c *Client) AddIndexerProxy(ctx context.Context, proxy IndexerProxy) error {
	if err := c.post(ctx, "/api/v1/indexerproxy", proxy, nil); err != nil {
		return fmt.Errorf("prowlarr add indexer proxy: %w", err)
	}
	return nil
}

// ListIndexerProxies returns all indexer proxies configured in Prowlarr.
func (c *Client) ListIndexerProxies(ctx context.Context) ([]IndexerProxy, error) {
	var proxies []IndexerProxy
	if err := c.get(ctx, "/api/v1/indexerproxy", nil, &proxies); err != nil {
		return nil, fmt.Errorf("prowlarr list indexer proxies: %w", err)
	}
	return proxies, nil
}

// get performs an authenticated GET request to the Prowlarr API and decodes the JSON response.
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("prowlarr API error %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// post performs an authenticated POST request to the Prowlarr API with a JSON body.
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
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("prowlarr API error %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}
