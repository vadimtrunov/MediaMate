package stack

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func newTestHealthChecker(t *testing.T, baseURL string) *HealthChecker {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewHealthChecker(baseURL, logger)
}

// ---------------------------------------------------------------------------
// 1. TestCheckServiceWithServer — end-to-end with httptest
// ---------------------------------------------------------------------------

func TestCheckServiceWithServer(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantHealth bool
	}{
		{"200 OK", http.StatusOK, true},
		{"401 Unauthorized", http.StatusUnauthorized, true},
		{"403 Forbidden", http.StatusForbidden, true},
		{"404 Not Found", http.StatusNotFound, true},
		{"500 Internal Server Error", http.StatusInternalServerError, false},
		{"503 Service Unavailable", http.StatusServiceUnavailable, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer srv.Close()

			// Extract host:port from test server URL (e.g. "http://127.0.0.1:12345").
			// We need baseURL without the port, and set a temporary endpoint with the port.
			// Easier: just use CheckService logic directly by setting baseURL
			// to include everything up to the port.
			//
			// Since serviceEndpoints has entries like ":8096/health", we can't
			// easily match that to our test server. Instead, temporarily add a
			// test entry to serviceEndpoints.
			const testService = "test-service"
			// Parse port from test server.
			addr := srv.URL[len("http://"):]                    // "127.0.0.1:12345"
			colonIdx := strings.LastIndex(addr, ":")            // index of ":"
			port := addr[colonIdx:]                             // ":12345"
			baseHost := "http://" + addr[:colonIdx]             // "http://127.0.0.1"
			serviceEndpoints[testService] = port + "/test-path" // ":12345/test-path"
			defer delete(serviceEndpoints, testService)

			hc := newTestHealthChecker(t, baseHost)
			result := hc.CheckService(context.Background(), testService)

			if result.Healthy != tc.wantHealth {
				t.Errorf("Healthy = %v, want %v (status %d)", result.Healthy, tc.wantHealth, tc.statusCode)
			}
			if result.Status != tc.statusCode {
				t.Errorf("Status = %d, want %d", result.Status, tc.statusCode)
			}
			if result.Latency <= 0 {
				t.Error("Latency should be > 0")
			}
			if result.Name != testService {
				t.Errorf("Name = %q, want %q", result.Name, testService)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 2. TestCheckServiceUnreachable
// ---------------------------------------------------------------------------

func TestCheckServiceUnreachable(t *testing.T) {
	// Use a port that nothing listens on.
	const testService = "unreachable-service"
	serviceEndpoints[testService] = ":19999/health"
	defer delete(serviceEndpoints, testService)

	hc := newTestHealthChecker(t, "http://127.0.0.1")
	result := hc.CheckService(context.Background(), testService)

	if result.Healthy {
		t.Error("expected Healthy=false for unreachable service")
	}
	if result.Status != 0 {
		t.Errorf("expected Status=0 for unreachable service, got %d", result.Status)
	}
	if result.Error == "" {
		t.Error("expected non-empty Error for unreachable service")
	}
}

// ---------------------------------------------------------------------------
// 3. TestCheckServiceUnknown
// ---------------------------------------------------------------------------

func TestCheckServiceUnknown(t *testing.T) {
	hc := newTestHealthChecker(t, "")
	result := hc.CheckService(context.Background(), "nonexistent-component")

	if result.Healthy {
		t.Error("expected Healthy=false for unknown service")
	}
	if result.Error != "unknown service" {
		t.Errorf("expected Error='unknown service', got %q", result.Error)
	}
}

// ---------------------------------------------------------------------------
// 4. TestCheckAllConcurrent
// ---------------------------------------------------------------------------

func TestCheckAllConcurrent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	addr := srv.URL[len("http://"):]
	colonIdx := strings.LastIndex(addr, ":")
	port := addr[colonIdx:]
	baseHost := "http://" + addr[:colonIdx]

	services := []string{"svc-a", "svc-b", "svc-c"}
	for _, svc := range services {
		serviceEndpoints[svc] = port + "/" + svc
	}
	t.Cleanup(func() {
		for _, svc := range services {
			delete(serviceEndpoints, svc)
		}
	})

	hc := newTestHealthChecker(t, baseHost)
	results := hc.CheckAll(context.Background(), services)

	if len(results) != len(services) {
		t.Fatalf("expected %d results, got %d", len(services), len(results))
	}

	// Verify order is preserved.
	for i, svc := range services {
		if results[i].Name != svc {
			t.Errorf("result[%d].Name = %q, want %q", i, results[i].Name, svc)
		}
		if !results[i].Healthy {
			t.Errorf("result[%d] (%s) expected healthy", i, svc)
		}
	}
}

// ---------------------------------------------------------------------------
// 5. TestNewHealthCheckerDefaults
// ---------------------------------------------------------------------------

func TestNewHealthCheckerDefaults(t *testing.T) {
	hc := NewHealthChecker("", nil)

	if hc.baseURL != "http://localhost" {
		t.Errorf("baseURL = %q, want %q", hc.baseURL, "http://localhost")
	}
	if hc.logger == nil {
		t.Error("logger should not be nil")
	}
	if hc.client == nil {
		t.Error("client should not be nil")
	}
}
