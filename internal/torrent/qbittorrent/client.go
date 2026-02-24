package qbittorrent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"

	"github.com/vadimtrunov/MediaMate/internal/core"
	"github.com/vadimtrunov/MediaMate/internal/httpclient"
)

// etaInfinity is the qBittorrent sentinel value indicating an unknown/infinite ETA.
const etaInfinity = 8640000

// Client implements core.TorrentClient for qBittorrent.
type Client struct {
	baseURL  string
	username string
	password string
	http     *httpclient.Client
	mu       sync.Mutex
	loggedIn bool
	logger   *slog.Logger
}

var _ core.TorrentClient = (*Client)(nil)

// New creates a new qBittorrent client.
func New(baseURL, username, password string, logger *slog.Logger) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	cfg := httpclient.DefaultConfig()
	httpClient := &http.Client{
		Timeout: cfg.Timeout,
		Jar:     jar,
	}

	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		http:     httpclient.NewWithHTTPClient(cfg, httpClient, logger),
		logger:   logger,
	}, nil
}

// List returns all torrents.
func (c *Client) List(ctx context.Context) ([]core.Torrent, error) {
	var torrents []qbitTorrent
	if err := c.getJSON(ctx, "/api/v2/torrents/info", nil, &torrents); err != nil {
		return nil, fmt.Errorf("list torrents: %w", err)
	}

	result := make([]core.Torrent, 0, len(torrents))
	for _, t := range torrents {
		result = append(result, toTorrent(t))
	}
	return result, nil
}

// GetProgress gets progress for a specific torrent.
func (c *Client) GetProgress(ctx context.Context, hash string) (*core.TorrentProgress, error) {
	params := url.Values{"hashes": {hash}}
	var torrents []qbitTorrent
	if err := c.getJSON(ctx, "/api/v2/torrents/info", params, &torrents); err != nil {
		return nil, fmt.Errorf("get torrent progress: %w", err)
	}
	if len(torrents) == 0 {
		return nil, fmt.Errorf("torrent %s not found", hash)
	}

	t := torrents[0]
	eta := t.ETA
	if eta >= etaInfinity {
		eta = 0
	}

	return &core.TorrentProgress{
		Hash:          t.Hash,
		Progress:      t.Progress * 100,
		Downloaded:    t.Downloaded,
		Total:         t.TotalSize,
		DownloadSpeed: t.DLSpeed,
		ETA:           eta,
	}, nil
}

// Pause pauses a torrent.
func (c *Client) Pause(ctx context.Context, hash string) error {
	return c.postForm(ctx, "/api/v2/torrents/pause", url.Values{"hashes": {hash}})
}

// Resume resumes a torrent.
func (c *Client) Resume(ctx context.Context, hash string) error {
	return c.postForm(ctx, "/api/v2/torrents/resume", url.Values{"hashes": {hash}})
}

// Remove removes a torrent.
func (c *Client) Remove(ctx context.Context, hash string, deleteFiles bool) error {
	data := url.Values{
		"hashes":      {hash},
		"deleteFiles": {fmt.Sprintf("%t", deleteFiles)},
	}
	return c.postForm(ctx, "/api/v2/torrents/delete", data)
}

// Name returns "qbittorrent".
func (c *Client) Name() string { return "qbittorrent" }

// login authenticates with the qBittorrent Web API.
func (c *Client) login(ctx context.Context) error {
	data := url.Values{
		"username": {c.username},
		"password": {c.password},
	}

	u := c.baseURL + "/api/v2/auth/login"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("login request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || strings.TrimSpace(string(body)) != "Ok." {
		return fmt.Errorf("login failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	c.loggedIn = true
	return nil
}

// ensureLoggedIn logs in to qBittorrent if not already authenticated.
func (c *Client) ensureLoggedIn(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.loggedIn {
		return c.login(ctx)
	}
	return nil
}

// doWithAuth executes a request with authentication, re-logging in on 403.
func (c *Client) doWithAuth(ctx context.Context, req *http.Request) (*http.Response, error) {
	if err := c.ensureLoggedIn(ctx); err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusForbidden {
		return resp, nil
	}

	// Session expired — re-login and retry once
	_ = resp.Body.Close()

	c.mu.Lock()
	c.loggedIn = false
	loginErr := c.login(ctx)
	c.mu.Unlock()
	if loginErr != nil {
		return nil, fmt.Errorf("re-login failed: %w", loginErr)
	}

	// Replay the body for POST requests
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, fmt.Errorf("replay body: %w", err)
		}
		req.Body = body
	}

	return c.http.Do(req)
}

// getJSON performs an authenticated GET request and decodes the JSON response.
func (c *Client) getJSON(ctx context.Context, path string, params url.Values, result any) error {
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

	resp, err := c.doWithAuth(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qbittorrent API error %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// postForm performs an authenticated POST request with form-encoded data.
func (c *Client) postForm(ctx context.Context, path string, data url.Values) error {
	body := data.Encode()
	u := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(body)), nil
	}

	resp, err := c.doWithAuth(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qbittorrent API error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// toTorrent converts a qBittorrent API torrent to a core.Torrent.
func toTorrent(t qbitTorrent) core.Torrent {
	eta := t.ETA
	if eta >= etaInfinity {
		eta = 0
	}

	return core.Torrent{
		Hash:          t.Hash,
		Name:          t.Name,
		Size:          t.Size,
		Progress:      t.Progress * 100,
		Status:        mapState(t.State),
		DownloadSpeed: t.DLSpeed,
		UploadSpeed:   t.UPSpeed,
		ETA:           eta,
	}
}

// mapState maps qBittorrent's internal state strings to normalized status names.
func mapState(state string) string {
	switch state {
	case "downloading", "forcedDL", "stalledDL", "metaDL", "allocating", "queuedDL", "checkingDL":
		return "downloading"
	case "uploading", "forcedUP", "stalledUP", "queuedUP", "checkingUP":
		return "seeding"
	case "pausedDL", "pausedUP":
		return "paused"
	case "error", "missingFiles", "unknown":
		return "error"
	default:
		return state
	}
}
