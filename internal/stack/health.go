package stack

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// serviceEndpoints maps component names to their default health check
// endpoints (port + path). A successful probe (HTTP status < 500) means the
// service is running — even a 401 Unauthorized is considered healthy because
// the application is responding.
var serviceEndpoints = map[string]string{
	ComponentRadarr:       ":7878/api/v3/health",
	ComponentSonarr:       ":8989/api/v3/health",
	ComponentReadarr:      ":8787/api/v1/health",
	ComponentProwlarr:     ":9696/api/v1/health",
	ComponentQBittorrent:  ":8080/api/v2/app/version",
	ComponentTransmission: ":9091/transmission/web/",
	ComponentDeluge:       ":8112/",
	ComponentJellyfin:     ":8096/health",
	ComponentPlex:         ":32400/identity",
	ComponentGluetun:      ":8000/v1/publicip/ip",
}

// ServiceHealth holds the result of a single service health probe.
type ServiceHealth struct {
	Name     string        // component name (e.g. "radarr")
	Endpoint string        // URL probed
	Healthy  bool          // true if the service responded with status < 500
	Status   int           // HTTP status code, 0 if unreachable
	Error    string        // error message if unhealthy
	Latency  time.Duration // round-trip time of the probe
}

// HealthChecker probes stack services via HTTP to determine whether they are
// running and responsive.
type HealthChecker struct {
	logger  *slog.Logger
	client  *http.Client
	baseURL string // base URL prefix, default "http://localhost"
}

// NewHealthChecker creates a HealthChecker that sends HTTP probes against
// service endpoints. If baseURL is empty it defaults to "http://localhost".
// If logger is nil, slog.Default() is used.
func NewHealthChecker(baseURL string, logger *slog.Logger) *HealthChecker {
	if baseURL == "" {
		baseURL = "http://localhost"
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &HealthChecker{
		logger:  logger,
		client:  &http.Client{Timeout: 5 * time.Second},
		baseURL: baseURL,
	}
}

// CheckService probes a single service by name and returns its health status.
// The component name must match one of the Component* constants defined in
// stack.go. Unknown names produce a result with Healthy=false and an
// appropriate error message.
func (hc *HealthChecker) CheckService(ctx context.Context, name string) ServiceHealth {
	endpoint, ok := serviceEndpoints[name]
	if !ok {
		return ServiceHealth{
			Name:  name,
			Error: "unknown service",
		}
	}

	url := hc.baseURL + endpoint
	result := ServiceHealth{
		Name:     name,
		Endpoint: url,
	}

	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		result.Error = fmt.Errorf("create request: %w", err).Error()
		result.Latency = time.Since(start)
		return result
	}

	resp, err := hc.client.Do(req) //nolint:gosec // URL is built from internal serviceEndpoints map, not user input
	result.Latency = time.Since(start)

	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.Status = resp.StatusCode
	result.Healthy = resp.StatusCode < 500

	if !result.Healthy {
		result.Error = fmt.Sprintf("unhealthy status: %d", resp.StatusCode)
	}

	return result
}

// CheckAll probes every service in the given list concurrently and returns
// the results in the same order as the input slice. Each probe is logged at
// Info level on completion.
func (hc *HealthChecker) CheckAll(ctx context.Context, services []string) []ServiceHealth {
	results := make([]ServiceHealth, len(services))

	var wg sync.WaitGroup
	wg.Add(len(services))

	for i, name := range services {
		go func(idx int, svc string) {
			defer wg.Done()
			results[idx] = hc.CheckService(ctx, svc)

			r := results[idx]
			hc.logger.Info("health probe",
				slog.String("service", r.Name),
				slog.Bool("healthy", r.Healthy),
				slog.Int("status", r.Status),
				slog.Duration("latency", r.Latency),
				slog.String("error", r.Error),
			)
		}(i, name)
	}

	wg.Wait()
	return results
}
