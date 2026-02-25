package stack

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/vadimtrunov/MediaMate/internal/backend/prowlarr"
	"github.com/vadimtrunov/MediaMate/internal/backend/radarr"
	"github.com/vadimtrunov/MediaMate/internal/torrent/qbittorrent"
)

// discardLogger returns a logger that discards all output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// ---------------------------------------------------------------------------
// 1. TestServicePort
// ---------------------------------------------------------------------------

func TestServicePort(t *testing.T) {
	tests := []struct {
		name      string
		component string
		want      string
	}{
		{"radarr", ComponentRadarr, "7878"},
		{"sonarr", ComponentSonarr, "8989"},
		{"prowlarr", ComponentProwlarr, "9696"},
		{"qbittorrent", ComponentQBittorrent, "8080"},
		{"jellyfin", ComponentJellyfin, "8096"},
		{"plex", ComponentPlex, "32400"},
		{"flaresolverr", ComponentFlareSolverr, "8191"},
		{"gluetun", ComponentGluetun, "8000"},
		{"unknown component", "nonexistent", ""},
		{"empty string", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := servicePort(tc.component)
			if got != tc.want {
				t.Errorf("servicePort(%q) = %q, want %q", tc.component, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 2. TestServiceURL
// ---------------------------------------------------------------------------

func TestServiceURL(t *testing.T) {
	tests := []struct {
		name      string
		component string
		want      string
	}{
		{"radarr", ComponentRadarr, "http://localhost:7878"},
		{"sonarr", ComponentSonarr, "http://localhost:8989"},
		{"prowlarr", ComponentProwlarr, "http://localhost:9696"},
		{"qbittorrent", ComponentQBittorrent, "http://localhost:8080"},
		{"jellyfin", ComponentJellyfin, "http://localhost:8096"},
		{"plex", ComponentPlex, "http://localhost:32400"},
		{"unknown component", "nonexistent", ""},
		{"empty string", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := serviceURL(tc.component)
			if got != tc.want {
				t.Errorf("serviceURL(%q) = %q, want %q", tc.component, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 2b. TestDockerServiceURL
// ---------------------------------------------------------------------------

func TestDockerServiceURL(t *testing.T) {
	tests := []struct {
		name      string
		component string
		want      string
	}{
		{"radarr", ComponentRadarr, "http://radarr:7878"},
		{"sonarr", ComponentSonarr, "http://sonarr:8989"},
		{"prowlarr", ComponentProwlarr, "http://prowlarr:9696"},
		{"qbittorrent", ComponentQBittorrent, "http://qbittorrent:8080"},
		{"jellyfin", ComponentJellyfin, "http://jellyfin:8096"},
		{"flaresolverr", ComponentFlareSolverr, "http://flaresolverr:8191"},
		{"unknown component", "nonexistent", ""},
		{"empty string", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dockerServiceURL(tc.component)
			if got != tc.want {
				t.Errorf("dockerServiceURL(%q) = %q, want %q", tc.component, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 3. TestSetupRunnerRun — integration test with mock servers
// ---------------------------------------------------------------------------

// newRadarrTestServer creates an httptest server that mocks the Radarr API
// endpoints used during setup. It returns the server and a pointer to a
// counter tracking POST calls (to verify create vs skip behavior).
func newRadarrTestServer(
	t *testing.T,
	existingFolders []radarr.RootFolder,
	existingClients []radarr.DownloadClientConfig,
) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var postCalls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v3/rootfolder":
			json.NewEncoder(w).Encode(existingFolders)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v3/rootfolder":
			postCalls.Add(1)
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			json.NewEncoder(w).Encode(radarr.RootFolder{ID: 1, Path: body["path"]})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v3/downloadclient":
			json.NewEncoder(w).Encode(existingClients)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v3/downloadclient":
			postCalls.Add(1)
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("radarr mock: unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &postCalls
}

// newProwlarrTestServer creates an httptest server that mocks the Prowlarr API.
func newProwlarrTestServer(
	t *testing.T,
	existingApps []prowlarr.Application,
	existingClients []prowlarr.DownloadClient,
	existingProxies []prowlarr.IndexerProxy,
) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var postCalls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/applications":
			json.NewEncoder(w).Encode(existingApps)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/applications":
			postCalls.Add(1)
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/downloadclient":
			json.NewEncoder(w).Encode(existingClients)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/downloadclient":
			postCalls.Add(1)
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/indexerproxy":
			json.NewEncoder(w).Encode(existingProxies)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/indexerproxy":
			postCalls.Add(1)
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("prowlarr mock: unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &postCalls
}

// newQBittorrentTestServer creates an httptest server that mocks qBittorrent.
func newQBittorrentTestServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var prefCalls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v2/auth/login":
			w.Write([]byte("Ok."))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/app/setPreferences":
			prefCalls.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("qbittorrent mock: unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &prefCalls
}

// setupRunnerTestEnv holds shared state for TestSetupRunnerRun subtests.
type setupRunnerTestEnv struct {
	sr            *SetupRunner
	cfg           *Config
	envPath       string
	configPath    string
	radarrSrv     *httptest.Server
	radarrPosts   *atomic.Int32
	prowlarrSrv   *httptest.Server
	prowlarrPosts *atomic.Int32
	qbitSrv       *httptest.Server
	qbitPrefCalls *atomic.Int32
}

// newSetupRunnerTestEnv creates the mock servers, temp files, and SetupRunner
// needed by the TestSetupRunnerRun subtests.
func newSetupRunnerTestEnv(t *testing.T) *setupRunnerTestEnv {
	t.Helper()

	radarrSrv, radarrPosts := newRadarrTestServer(t, nil, nil)
	prowlarrSrv, prowlarrPosts := newProwlarrTestServer(t, nil, nil, nil)
	qbitSrv, qbitPrefCalls := newQBittorrentTestServer(t)

	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config")

	for _, comp := range []string{ComponentRadarr, ComponentProwlarr} {
		dir := filepath.Join(configDir, comp)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		xmlContent := `<Config><ApiKey>test-api-key-` + comp + `</ApiKey></Config>`
		if err := os.WriteFile(filepath.Join(dir, "config.xml"), []byte(xmlContent), 0o600); err != nil {
			t.Fatalf("write config.xml: %v", err)
		}
	}

	envContent := "MEDIAMATE_RADARR_API_KEY=placeholder\nMEDIAMATE_PROWLARR_API_KEY=placeholder\n"
	envPath := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envPath, []byte(envContent), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	yamlContent := "radarr:\n  api_key: ${MEDIAMATE_RADARR_API_KEY}\nprowlarr:\n  api_key: ${MEDIAMATE_PROWLARR_API_KEY}\n"
	configPath := filepath.Join(tmpDir, "mediamate.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("write mediamate.yaml: %v", err)
	}

	cfg := &Config{
		Components:   []string{ComponentRadarr, ComponentProwlarr, ComponentQBittorrent, ComponentFlareSolverr},
		MoviesDir:    "/movies",
		DownloadsDir: "/downloads",
		ConfigDir:    configDir,
	}

	result := &GenerateResult{
		EnvPath:    envPath,
		ConfigPath: configPath,
	}

	return &setupRunnerTestEnv{
		sr:            NewSetupRunner(cfg, result, discardLogger()),
		cfg:           cfg,
		envPath:       envPath,
		configPath:    configPath,
		radarrSrv:     radarrSrv,
		radarrPosts:   radarrPosts,
		prowlarrSrv:   prowlarrSrv,
		prowlarrPosts: prowlarrPosts,
		qbitSrv:       qbitSrv,
		qbitPrefCalls: qbitPrefCalls,
	}
}

func TestSetupRunnerRadarr(t *testing.T) {
	env := newSetupRunnerTestEnv(t)
	ctx := context.Background()

	client := radarr.New(env.radarrSrv.URL, "test-api-key-radarr", "", "", discardLogger())
	results := env.sr.radarrCreateRootFolder(ctx, client)
	results = append(results, env.sr.radarrAddDownloadClient(ctx, client)...)

	for _, r := range results {
		if !r.OK {
			t.Errorf("radarr setup failed: %s — %s", r.Action, r.Error)
		}
	}
	if env.radarrPosts.Load() != 2 {
		t.Errorf("expected 2 radarr POST calls, got %d", env.radarrPosts.Load())
	}
}

func TestSetupRunnerProwlarr(t *testing.T) {
	env := newSetupRunnerTestEnv(t)
	ctx := context.Background()

	client := prowlarr.New(env.prowlarrSrv.URL, "test-api-key-prowlarr", discardLogger())
	results := env.sr.prowlarrAddRadarr(ctx, client, "test-api-key-radarr")
	results = append(results, env.sr.prowlarrAddDownloadClient(ctx, client)...)
	results = append(results, env.sr.prowlarrAddFlareSolverr(ctx, client)...)

	for _, r := range results {
		if !r.OK {
			t.Errorf("prowlarr setup failed: %s — %s", r.Action, r.Error)
		}
	}
	if env.prowlarrPosts.Load() != 3 {
		t.Errorf("expected 3 prowlarr POST calls, got %d", env.prowlarrPosts.Load())
	}
}

func TestSetupRunnerQBittorrent(t *testing.T) {
	env := newSetupRunnerTestEnv(t)
	ctx := context.Background()

	client, err := qbittorrent.New(env.qbitSrv.URL, qbitDefaultUser, qbitDefaultPassword, discardLogger())
	if err != nil {
		t.Fatalf("create qbittorrent client: %v", err)
	}
	prefs := map[string]any{"save_path": env.cfg.DownloadsDir}
	if err := client.SetPreferences(ctx, prefs); err != nil {
		t.Fatalf("qbittorrent set preferences: %v", err)
	}
	if env.qbitPrefCalls.Load() != 1 {
		t.Errorf("expected 1 qbittorrent setPreferences call, got %d", env.qbitPrefCalls.Load())
	}
}

func TestSetupRunnerUpdateConfigs(t *testing.T) {
	env := newSetupRunnerTestEnv(t)

	keys := ServiceAPIKeys{
		ComponentRadarr:   "test-api-key-radarr",
		ComponentProwlarr: "test-api-key-prowlarr",
	}
	results := env.sr.updateConfigs(keys)
	for _, r := range results {
		if !r.OK {
			t.Errorf("updateConfigs failed: %s — %s", r.Action, r.Error)
		}
	}

	envData, err := os.ReadFile(env.envPath)
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	if !strings.Contains(string(envData), "MEDIAMATE_RADARR_API_KEY=test-api-key-radarr") {
		t.Errorf(".env should contain radarr API key, got: %s", string(envData))
	}
	if !strings.Contains(string(envData), "MEDIAMATE_PROWLARR_API_KEY=test-api-key-prowlarr") {
		t.Errorf(".env should contain prowlarr API key, got: %s", string(envData))
	}

	yamlData, err := os.ReadFile(env.configPath)
	if err != nil {
		t.Fatalf("read mediamate.yaml: %v", err)
	}
	if strings.Contains(string(yamlData), "${MEDIAMATE_RADARR_API_KEY}") {
		t.Error("mediamate.yaml should not contain radarr placeholder")
	}
	if !strings.Contains(string(yamlData), "test-api-key-radarr") {
		t.Errorf("mediamate.yaml should contain radarr API key, got: %s", string(yamlData))
	}
}

// ---------------------------------------------------------------------------
// 4. TestSetupRunnerIdempotent — skip when resources already exist
// ---------------------------------------------------------------------------

// newIdempotentSetupRunner creates a SetupRunner for idempotency tests.
func newIdempotentSetupRunner() *SetupRunner {
	cfg := &Config{
		Components:   []string{ComponentRadarr, ComponentProwlarr, ComponentQBittorrent, ComponentFlareSolverr},
		MoviesDir:    "/movies",
		DownloadsDir: "/downloads",
	}
	return NewSetupRunner(cfg, &GenerateResult{}, discardLogger())
}

// assertIdempotent verifies that a single OK result was returned with zero POST calls.
func assertIdempotent(t *testing.T, results []SetupResult, postCalls *atomic.Int32) {
	t.Helper()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].OK {
		t.Errorf("expected OK=true, got error: %s", results[0].Error)
	}
	if postCalls.Load() != 0 {
		t.Errorf("expected 0 POST calls, got %d", postCalls.Load())
	}
}

func TestIdempotentRadarr(t *testing.T) {
	ctx := context.Background()
	sr := newIdempotentSetupRunner()

	t.Run("root folder already exists", func(t *testing.T) {
		existingFolders := []radarr.RootFolder{{ID: 1, Path: "/movies"}}
		srv, postCalls := newRadarrTestServer(t, existingFolders, nil)
		client := radarr.New(srv.URL, "key", "", "", discardLogger())
		assertIdempotent(t, sr.radarrCreateRootFolder(ctx, client), postCalls)
	})

	t.Run("download client already exists", func(t *testing.T) {
		existingClients := []radarr.DownloadClientConfig{
			{Name: "qBittorrent", Implementation: "QBittorrent"},
		}
		srv, postCalls := newRadarrTestServer(t, nil, existingClients)
		client := radarr.New(srv.URL, "key", "", "", discardLogger())
		assertIdempotent(t, sr.radarrAddDownloadClient(ctx, client), postCalls)
	})
}

func TestIdempotentProwlarr(t *testing.T) {
	ctx := context.Background()
	sr := newIdempotentSetupRunner()

	t.Run("application already exists", func(t *testing.T) {
		existingApps := []prowlarr.Application{
			{Name: "Radarr", Implementation: "Radarr"},
		}
		srv, postCalls := newProwlarrTestServer(t, existingApps, nil, nil)
		client := prowlarr.New(srv.URL, "key", discardLogger())
		assertIdempotent(t, sr.prowlarrAddRadarr(ctx, client, "radarr-key"), postCalls)
	})

	t.Run("download client already exists", func(t *testing.T) {
		existingClients := []prowlarr.DownloadClient{
			{Name: "qBittorrent", Implementation: "QBittorrent"},
		}
		srv, postCalls := newProwlarrTestServer(t, nil, existingClients, nil)
		client := prowlarr.New(srv.URL, "key", discardLogger())
		assertIdempotent(t, sr.prowlarrAddDownloadClient(ctx, client), postCalls)
	})

	t.Run("flaresolverr proxy already exists", func(t *testing.T) {
		existingProxies := []prowlarr.IndexerProxy{
			{Name: "FlareSolverr", Implementation: "FlareSolverr"},
		}
		srv, postCalls := newProwlarrTestServer(t, nil, nil, existingProxies)
		client := prowlarr.New(srv.URL, "key", discardLogger())
		assertIdempotent(t, sr.prowlarrAddFlareSolverr(ctx, client), postCalls)
	})
}

// ---------------------------------------------------------------------------
// 5. TestWaitForHealth — verify health-check loop behavior
// ---------------------------------------------------------------------------

// healthTestServer creates an httptest server and returns the server, base host,
// and port suffix for use in health-check tests.
func healthTestServer(t *testing.T, statusCode int) (baseHost, port string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusCode)
	}))
	t.Cleanup(srv.Close)

	addr := srv.URL[len("http://"):]
	colonIdx := strings.LastIndex(addr, ":")
	return "http://" + addr[:colonIdx], addr[colonIdx:]
}

func TestWaitForHealth(t *testing.T) {
	baseHost, port := healthTestServer(t, http.StatusOK)

	const svc1 = "test-health-svc-a"
	const svc2 = "test-health-svc-b"
	serviceEndpoints[svc1] = port + "/health-a"
	serviceEndpoints[svc2] = port + "/health-b"
	t.Cleanup(func() {
		delete(serviceEndpoints, svc1)
		delete(serviceEndpoints, svc2)
	})

	components := []string{svc1, svc2}

	t.Run("all services healthy", func(t *testing.T) {
		hc := NewHealthChecker(baseHost, discardLogger())
		healthResults := hc.CheckAll(context.Background(), components)

		if len(healthResults) != 2 {
			t.Fatalf("expected 2 health results, got %d", len(healthResults))
		}
		for _, h := range healthResults {
			if !h.Healthy {
				t.Errorf("expected %s to be healthy, got error: %s", h.Name, h.Error)
			}
		}
	})

	t.Run("health to setup result conversion", func(t *testing.T) {
		hc := NewHealthChecker(baseHost, discardLogger())
		healthResults := hc.CheckAll(context.Background(), components)

		var setupResults []SetupResult
		for _, h := range healthResults {
			r := SetupResult{
				Service: h.Name,
				Action:  "health check",
				OK:      h.Healthy,
			}
			if !h.Healthy {
				r.Error = h.Error
			}
			setupResults = append(setupResults, r)
		}

		for i, r := range setupResults {
			if r.Service != components[i] {
				t.Errorf("result[%d].Service = %q, want %q", i, r.Service, components[i])
			}
			if r.Action != "health check" {
				t.Errorf("result[%d].Action = %q, want %q", i, r.Action, "health check")
			}
			if !r.OK {
				t.Errorf("result[%d] should be OK", i)
			}
		}
	})
}

func TestWaitForHealthUnhealthy(t *testing.T) {
	baseHost, port := healthTestServer(t, http.StatusServiceUnavailable)

	const svc = "test-unhealthy-svc"
	serviceEndpoints[svc] = port + "/health"
	t.Cleanup(func() { delete(serviceEndpoints, svc) })

	hc := NewHealthChecker(baseHost, discardLogger())
	results := hc.CheckAll(context.Background(), []string{svc})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Healthy {
		t.Error("expected service to be unhealthy (503)")
	}
	if results[0].Status != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", results[0].Status)
	}
}

// ---------------------------------------------------------------------------
// 6. TestUpdateConfigs — verify .env and yaml file updates
// ---------------------------------------------------------------------------

// setupUpdateConfigsEnv creates temp .env and mediamate.yaml files, runs
// updateConfigs, and returns the SetupRunner along with file paths for
// verification by subtests.
func setupUpdateConfigsEnv(t *testing.T) (envPath, configPath string) {
	t.Helper()
	tmpDir := t.TempDir()

	envPath = filepath.Join(tmpDir, ".env")
	envContent := strings.Join([]string{
		"# MediaMate Environment",
		"MEDIAMATE_RADARR_API_KEY=your-radarr-api-key-here",
		"MEDIAMATE_SONARR_API_KEY=your-sonarr-api-key-here",
		"MEDIAMATE_PROWLARR_API_KEY=your-prowlarr-api-key-here",
		"SOME_OTHER_VAR=untouched",
	}, "\n")
	if err := os.WriteFile(envPath, []byte(envContent), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	configPath = filepath.Join(tmpDir, "mediamate.yaml")
	yamlContent := strings.Join([]string{
		"radarr:",
		"  api_key: ${MEDIAMATE_RADARR_API_KEY}",
		"sonarr:",
		"  api_key: ${MEDIAMATE_SONARR_API_KEY}",
		"prowlarr:",
		"  api_key: ${MEDIAMATE_PROWLARR_API_KEY}",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("write mediamate.yaml: %v", err)
	}

	return envPath, configPath
}

// runUpdateConfigs executes updateConfigs and returns the env/config paths for
// subsequent assertions.
func runUpdateConfigs(t *testing.T) (envPath, configPath string) {
	t.Helper()
	envPath, configPath = setupUpdateConfigsEnv(t)

	sr := NewSetupRunner(
		&Config{},
		&GenerateResult{EnvPath: envPath, ConfigPath: configPath},
		discardLogger(),
	)
	keys := ServiceAPIKeys{
		ComponentRadarr:   "real-radarr-key-abc123",
		ComponentProwlarr: "real-prowlarr-key-def456",
	}
	results := sr.updateConfigs(keys)
	if len(results) != 2 {
		t.Fatalf("expected 2 results (env + config), got %d", len(results))
	}
	for _, r := range results {
		if !r.OK {
			t.Errorf("updateConfigs %s failed: %s", r.Action, r.Error)
		}
	}
	return envPath, configPath
}

func TestUpdateConfigsEnvFile(t *testing.T) {
	envPath, _ := runUpdateConfigs(t)

	envData, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	envStr := string(envData)

	if !strings.Contains(envStr, "MEDIAMATE_RADARR_API_KEY=real-radarr-key-abc123") {
		t.Errorf(".env should contain radarr key, got:\n%s", envStr)
	}
	if !strings.Contains(envStr, "MEDIAMATE_PROWLARR_API_KEY=real-prowlarr-key-def456") {
		t.Errorf(".env should contain prowlarr key, got:\n%s", envStr)
	}
	if !strings.Contains(envStr, "MEDIAMATE_SONARR_API_KEY=your-sonarr-api-key-here") {
		t.Errorf(".env sonarr key should be unchanged, got:\n%s", envStr)
	}
	if !strings.Contains(envStr, "SOME_OTHER_VAR=untouched") {
		t.Errorf(".env SOME_OTHER_VAR should be untouched, got:\n%s", envStr)
	}
}

func TestUpdateConfigsYAMLFile(t *testing.T) {
	_, configPath := runUpdateConfigs(t)

	yamlData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read mediamate.yaml: %v", err)
	}
	yamlStr := string(yamlData)

	if strings.Contains(yamlStr, "${MEDIAMATE_RADARR_API_KEY}") {
		t.Error("mediamate.yaml should not contain radarr placeholder")
	}
	if !strings.Contains(yamlStr, "real-radarr-key-abc123") {
		t.Errorf("mediamate.yaml should contain radarr key, got:\n%s", yamlStr)
	}
	if strings.Contains(yamlStr, "${MEDIAMATE_PROWLARR_API_KEY}") {
		t.Error("mediamate.yaml should not contain prowlarr placeholder")
	}
	if !strings.Contains(yamlStr, "real-prowlarr-key-def456") {
		t.Errorf("mediamate.yaml should contain prowlarr key, got:\n%s", yamlStr)
	}
	if !strings.Contains(yamlStr, "${MEDIAMATE_SONARR_API_KEY}") {
		t.Errorf("mediamate.yaml sonarr placeholder should remain, got:\n%s", yamlStr)
	}
}

func TestUpdateConfigsMissingFiles(t *testing.T) {
	sr := NewSetupRunner(
		&Config{},
		&GenerateResult{
			EnvPath:    "/nonexistent/path/.env",
			ConfigPath: "/nonexistent/path/mediamate.yaml",
		},
		discardLogger(),
	)

	results := sr.updateConfigs(ServiceAPIKeys{ComponentRadarr: "key"})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.OK {
			t.Errorf("expected failure for missing file: %s", r.Action)
		}
		if r.Error == "" {
			t.Errorf("expected non-empty error for missing file: %s", r.Action)
		}
	}
}

// ---------------------------------------------------------------------------
// 7. TestNewSetupRunner — constructor validation
// ---------------------------------------------------------------------------

func TestNewSetupRunner(t *testing.T) {
	t.Run("nil logger uses default", func(t *testing.T) {
		sr := NewSetupRunner(&Config{}, &GenerateResult{}, nil)
		if sr.logger == nil {
			t.Error("logger should not be nil when nil is passed")
		}
	})

	t.Run("provided logger is used", func(t *testing.T) {
		logger := discardLogger()
		sr := NewSetupRunner(&Config{}, &GenerateResult{}, logger)
		if sr.logger != logger {
			t.Error("expected provided logger to be used")
		}
	})

	t.Run("config and result are stored", func(t *testing.T) {
		cfg := &Config{MoviesDir: "/test-movies"}
		result := &GenerateResult{EnvPath: "/test/.env"}
		sr := NewSetupRunner(cfg, result, discardLogger())
		if sr.cfg != cfg {
			t.Error("cfg not stored correctly")
		}
		if sr.result != result {
			t.Error("result not stored correctly")
		}
	})
}

// ---------------------------------------------------------------------------
// 8. TestSetupResult — verify the type
// ---------------------------------------------------------------------------

func TestSetupResultFields(t *testing.T) {
	r := SetupResult{
		Service: "radarr",
		Action:  "create root folder",
		OK:      true,
		Error:   "",
	}

	if r.Service != "radarr" {
		t.Errorf("Service = %q, want %q", r.Service, "radarr")
	}
	if r.Action != "create root folder" {
		t.Errorf("Action = %q, want %q", r.Action, "create root folder")
	}
	if !r.OK {
		t.Error("expected OK=true")
	}
	if r.Error != "" {
		t.Errorf("expected empty Error, got %q", r.Error)
	}

	// Error case.
	r2 := SetupResult{
		Service: "prowlarr",
		Action:  "add download client",
		OK:      false,
		Error:   "connection refused",
	}
	if r2.Service != "prowlarr" {
		t.Errorf("Service = %q, want %q", r2.Service, "prowlarr")
	}
	if r2.Action != "add download client" {
		t.Errorf("Action = %q, want %q", r2.Action, "add download client")
	}
	if r2.OK {
		t.Error("expected OK=false")
	}
	if r2.Error != "connection refused" {
		t.Errorf("Error = %q, want %q", r2.Error, "connection refused")
	}
}

// ---------------------------------------------------------------------------
// 9. TestRadarrSetupAPIError — verify error handling when API returns errors
// ---------------------------------------------------------------------------

// newErrorServer creates an httptest server that returns 500 for all requests.
func newErrorServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// assertSingleFailure verifies that results contain exactly one failed result.
func assertSingleFailure(t *testing.T, results []SetupResult) {
	t.Helper()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].OK {
		t.Error("expected failure")
	}
}

func TestRadarrSetupAPIError(t *testing.T) {
	ctx := context.Background()
	sr := NewSetupRunner(
		&Config{
			Components: []string{ComponentRadarr, ComponentQBittorrent},
			MoviesDir:  "/movies",
		},
		&GenerateResult{},
		discardLogger(),
	)

	t.Run("list root folders error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal"}`))
		}))
		defer srv.Close()

		client := radarr.New(srv.URL, "key", "", "", discardLogger())
		results := sr.radarrCreateRootFolder(ctx, client)
		assertSingleFailure(t, results)
		if results[0].Error == "" {
			t.Error("expected non-empty error")
		}
	})

	t.Run("list download clients error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/v3/downloadclient" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode([]radarr.RootFolder{})
		}))
		defer srv.Close()

		client := radarr.New(srv.URL, "key", "", "", discardLogger())
		results := sr.radarrAddDownloadClient(ctx, client)
		assertSingleFailure(t, results)
	})
}

// ---------------------------------------------------------------------------
// 10. TestProwlarrSetupAPIError — verify error handling for prowlarr
// ---------------------------------------------------------------------------

func TestProwlarrSetupAPIError(t *testing.T) {
	ctx := context.Background()
	sr := NewSetupRunner(
		&Config{
			Components: []string{ComponentProwlarr, ComponentQBittorrent, ComponentFlareSolverr},
		},
		&GenerateResult{},
		discardLogger(),
	)

	t.Run("list applications error", func(t *testing.T) {
		srv := newErrorServer(t)
		client := prowlarr.New(srv.URL, "key", discardLogger())
		results := sr.prowlarrAddRadarr(ctx, client, "radarr-key")
		assertSingleFailure(t, results)
	})

	t.Run("list download clients error", func(t *testing.T) {
		srv := newErrorServer(t)
		client := prowlarr.New(srv.URL, "key", discardLogger())
		results := sr.prowlarrAddDownloadClient(ctx, client)
		assertSingleFailure(t, results)
	})

	t.Run("list indexer proxies error", func(t *testing.T) {
		srv := newErrorServer(t)
		client := prowlarr.New(srv.URL, "key", discardLogger())
		results := sr.prowlarrAddFlareSolverr(ctx, client)
		assertSingleFailure(t, results)
	})
}

// ---------------------------------------------------------------------------
// 11. TestQBittorrentSetupError — verify qBittorrent error paths
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// 12. TestRadarrAddWebhook — webhook registration tests
// ---------------------------------------------------------------------------

func newWebhookRadarrServer(
	t *testing.T,
	existing []radarr.NotificationConfig,
) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var postCalls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v3/notification":
			json.NewEncoder(w).Encode(existing)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v3/notification":
			postCalls.Add(1)
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("webhook mock: unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &postCalls
}

func TestRadarrAddWebhook_New(t *testing.T) {
	ctx := context.Background()
	var receivedBody []byte
	var postCalls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v3/notification":
			json.NewEncoder(w).Encode([]radarr.NotificationConfig{})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v3/notification":
			postCalls.Add(1)
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read request body: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			receivedBody = body
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("webhook mock: unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	sr := NewSetupRunner(
		&Config{Components: []string{ComponentRadarr, ComponentMediaMate}},
		&GenerateResult{},
		discardLogger(),
	)

	client := radarr.New(srv.URL, "key", "", "", discardLogger())
	results := sr.radarrAddWebhook(ctx, client)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].OK {
		t.Errorf("expected OK, got error: %s", results[0].Error)
	}
	if postCalls.Load() != 1 {
		t.Errorf("expected 1 POST, got %d", postCalls.Load())
	}
	if body := string(receivedBody); !strings.Contains(body, ":8080/webhooks/radarr") {
		t.Errorf("expected default port 8080 in webhook URL, got: %s", body)
	}
}

func TestRadarrAddWebhook_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	existing := []radarr.NotificationConfig{
		{ID: 1, Name: "MediaMate", Implementation: "Webhook"},
	}
	srv, postCalls := newWebhookRadarrServer(t, existing)
	sr := NewSetupRunner(
		&Config{Components: []string{ComponentRadarr, ComponentMediaMate}},
		&GenerateResult{},
		discardLogger(),
	)

	client := radarr.New(srv.URL, "key", "", "", discardLogger())
	results := sr.radarrAddWebhook(ctx, client)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].OK {
		t.Errorf("expected OK (skipped), got error: %s", results[0].Error)
	}
	if postCalls.Load() != 0 {
		t.Errorf("expected 0 POST calls (skipped), got %d", postCalls.Load())
	}
}

func TestRadarrAddWebhook_ListError(t *testing.T) {
	ctx := context.Background()
	srv := newErrorServer(t)
	sr := NewSetupRunner(
		&Config{Components: []string{ComponentRadarr, ComponentMediaMate}},
		&GenerateResult{},
		discardLogger(),
	)

	client := radarr.New(srv.URL, "key", "", "", discardLogger())
	results := sr.radarrAddWebhook(ctx, client)

	assertSingleFailure(t, results)
}

func TestRadarrAddWebhook_CustomPortAndSecret(t *testing.T) {
	ctx := context.Background()
	var receivedBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v3/notification":
			json.NewEncoder(w).Encode([]radarr.NotificationConfig{})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v3/notification":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read request body: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			receivedBody = body
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	sr := NewSetupRunner(
		&Config{
			Components:    []string{ComponentRadarr, ComponentMediaMate},
			WebhookPort:   9090,
			WebhookSecret: "my-secret",
		},
		&GenerateResult{},
		discardLogger(),
	)

	client := radarr.New(srv.URL, "key", "", "", discardLogger())
	results := sr.radarrAddWebhook(ctx, client)

	if len(results) != 1 || !results[0].OK {
		t.Fatalf("expected success, got %+v", results)
	}

	body := string(receivedBody)
	if !strings.Contains(body, ":9090/webhooks/radarr") {
		t.Errorf("expected port 9090 in webhook URL, got: %s", body)
	}
	if !strings.Contains(body, "X-Webhook-Secret") {
		t.Errorf("expected X-Webhook-Secret header, got: %s", body)
	}
	if !strings.Contains(body, "my-secret") {
		t.Errorf("expected secret value in body, got: %s", body)
	}
}

func TestQBittorrentSetupError(t *testing.T) {
	// Server that rejects login.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/auth/login" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Fails."))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client, err := qbittorrent.New(srv.URL, "admin", "wrongpass", discardLogger())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	err = client.SetPreferences(context.Background(), map[string]any{"save_path": "/downloads"})
	if err == nil {
		t.Error("expected error when login fails")
	}
}
