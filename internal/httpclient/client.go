package httpclient

import (
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// Config holds retry and timeout configuration.
type Config struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Timeout    time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
		MaxDelay:   10 * time.Second,
		Timeout:    30 * time.Second,
	}
}

// Client wraps http.Client with retry logic.
type Client struct {
	http   *http.Client
	config Config
	logger *slog.Logger
}

// New creates a new Client with a default http.Client.
func New(cfg Config, logger *slog.Logger) *Client {
	return &Client{
		http: &http.Client{
			Timeout: cfg.Timeout,
		},
		config: cfg,
		logger: logger,
	}
}

// NewWithHTTPClient creates a Client with a custom http.Client (e.g. for cookie jars).
func NewWithHTTPClient(cfg Config, httpClient *http.Client, logger *slog.Logger) *Client {
	return &Client{
		http:   httpClient,
		config: cfg,
		logger: logger,
	}
}

// Do executes an HTTP request with retry logic.
// Retries on 429, 500, 502, 503, 504 and transient network errors.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response

	for attempt := range c.config.MaxRetries {
		if attempt > 0 {
			c.waitBeforeRetry(attempt, lastResp, req.URL.String())
			if err := replayBody(req); err != nil {
				return nil, err
			}
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			lastResp = nil
			continue
		}

		if !shouldRetry(resp.StatusCode) {
			return resp, nil
		}

		lastErr = fmt.Errorf("HTTP %d from %s", resp.StatusCode, req.URL.String())
		lastResp = resp
		resp.Body.Close()
	}

	if lastErr != nil {
		return nil, fmt.Errorf("request failed after %d attempts: %w", c.config.MaxRetries, lastErr)
	}
	return nil, fmt.Errorf("request failed after %d attempts", c.config.MaxRetries)
}

func (c *Client) waitBeforeRetry(attempt int, lastResp *http.Response, url string) {
	delay := c.backoff(attempt)
	if d := retryAfterDelay(lastResp); d > delay {
		delay = d
	}

	c.logger.Debug("retrying request",
		slog.Int("attempt", attempt+1),
		slog.String("delay", delay.String()),
		slog.String("url", url),
	)
	time.Sleep(delay)
}

func retryAfterDelay(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}
	ra := resp.Header.Get("Retry-After")
	if ra == "" {
		return 0
	}
	seconds, err := strconv.Atoi(ra)
	if err != nil {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func replayBody(req *http.Request) error {
	if req.GetBody == nil {
		return nil
	}
	body, err := req.GetBody()
	if err != nil {
		return fmt.Errorf("failed to replay request body: %w", err)
	}
	req.Body = body
	return nil
}

// shouldRetry returns true for status codes that warrant a retry.
func shouldRetry(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

// backoff calculates the delay for a given attempt with jitter.
func (c *Client) backoff(attempt int) time.Duration {
	delay := float64(c.config.BaseDelay) * math.Pow(2, float64(attempt-1))
	if delay > float64(c.config.MaxDelay) {
		delay = float64(c.config.MaxDelay)
	}
	// Add 20% jitter
	jitter := delay * 0.2 * rand.Float64()
	return time.Duration(delay + jitter)
}
