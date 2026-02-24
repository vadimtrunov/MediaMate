package stack

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/vadimtrunov/MediaMate/internal/backend/prowlarr"
	"github.com/vadimtrunov/MediaMate/internal/backend/radarr"
	"github.com/vadimtrunov/MediaMate/internal/torrent/qbittorrent"
)

// Setup retry configuration.
const (
	setupHealthRetries  = 3
	setupHealthInterval = 10 * time.Second
)

// Default credentials for the qBittorrent LinuxServer container.
const (
	qbitDefaultUser     = "admin"
	qbitDefaultPassword = "adminadmin"
)

// SetupResult records the outcome of a single setup action.
type SetupResult struct {
	Service string // e.g. "radarr"
	Action  string // e.g. "create root folder"
	OK      bool
	Error   string // empty if OK
}

// SetupRunner orchestrates post-stack-init configuration for all services.
type SetupRunner struct {
	logger *slog.Logger
	cfg    *Config
	result *GenerateResult
}

// NewSetupRunner creates a SetupRunner instance.
func NewSetupRunner(cfg *Config, result *GenerateResult, logger *slog.Logger) *SetupRunner {
	if logger == nil {
		logger = slog.Default()
	}
	return &SetupRunner{
		logger: logger,
		cfg:    cfg,
		result: result,
	}
}

// Run performs the full post-init setup sequence:
//  1. Wait for services to become healthy
//  2. Read API keys from config.xml files
//  3. Update .env and mediamate.yaml with real API keys
//  4. Configure Radarr (root folder + download client)
//  5. Configure Prowlarr (application + download client + indexer proxy)
//  6. Configure qBittorrent (save path)
//
// Errors in individual service configuration are recorded but do not halt
// the overall process.
func (sr *SetupRunner) Run(ctx context.Context) []SetupResult {
	var results []SetupResult

	// Step 1: Wait for services to become healthy.
	results = append(results, sr.waitForHealth(ctx)...)

	// Step 2: Read API keys.
	sr.logger.Info("reading API keys from config files")
	keys := ReadAPIKeys(sr.cfg.ConfigDir, sr.cfg.Components, sr.logger)

	// Step 3: Update .env and mediamate.yaml.
	results = append(results, sr.updateConfigs(keys)...)

	// Step 4: Configure Radarr.
	if sr.cfg.HasComponent(ComponentRadarr) {
		if apiKey, ok := keys[ComponentRadarr]; ok {
			results = append(results, sr.setupRadarr(ctx, apiKey)...)
		} else {
			sr.logger.Warn("skipping radarr setup: no API key available")
			results = append(results, SetupResult{
				Service: ComponentRadarr,
				Action:  "setup",
				Error:   "no API key available",
			})
		}
	}

	// Step 5: Configure Prowlarr.
	if sr.cfg.HasComponent(ComponentProwlarr) {
		if apiKey, ok := keys[ComponentProwlarr]; ok {
			results = append(results, sr.setupProwlarr(ctx, apiKey, keys)...)
		} else {
			sr.logger.Warn("skipping prowlarr setup: no API key available")
			results = append(results, SetupResult{
				Service: ComponentProwlarr,
				Action:  "setup",
				Error:   "no API key available",
			})
		}
	}

	// Step 6: Configure qBittorrent.
	if sr.cfg.HasComponent(ComponentQBittorrent) {
		results = append(results, sr.setupQBittorrent(ctx)...)
	}

	return results
}

// waitForHealth retries health checks until all services are healthy or
// retries are exhausted.
func (sr *SetupRunner) waitForHealth(ctx context.Context) []SetupResult {
	hc := NewHealthChecker("", sr.logger)

	var lastResults []ServiceHealth
	for attempt := 1; attempt <= setupHealthRetries; attempt++ {
		sr.logger.Info("health check attempt",
			slog.Int("attempt", attempt),
			slog.Int("max", setupHealthRetries),
		)

		lastResults = hc.CheckAll(ctx, sr.cfg.Components)

		allHealthy := true
		for _, r := range lastResults {
			if !r.Healthy {
				allHealthy = false
				break
			}
		}

		if allHealthy {
			sr.logger.Info("all services healthy")
			break
		}

		if attempt < setupHealthRetries {
			sr.logger.Info("some services unhealthy, retrying",
				slog.Duration("wait", setupHealthInterval),
			)
			select {
			case <-ctx.Done():
				break
			case <-time.After(setupHealthInterval):
			}
		}
	}

	// Convert health results to SetupResults.
	var results []SetupResult
	for _, h := range lastResults {
		r := SetupResult{
			Service: h.Name,
			Action:  "health check",
			OK:      h.Healthy,
		}
		if !h.Healthy {
			r.Error = h.Error
		}
		results = append(results, r)
	}
	return results
}

// updateConfigs writes real API keys into .env and mediamate.yaml.
func (sr *SetupRunner) updateConfigs(keys ServiceAPIKeys) []SetupResult {
	var results []SetupResult

	if err := UpdateEnvFile(sr.result.EnvPath, keys); err != nil {
		sr.logger.Error("failed to update .env", slog.String("error", err.Error()))
		results = append(results, SetupResult{
			Service: "env",
			Action:  "update .env with API keys",
			Error:   err.Error(),
		})
	} else {
		sr.logger.Info("updated .env with API keys")
		results = append(results, SetupResult{
			Service: "env",
			Action:  "update .env with API keys",
			OK:      true,
		})
	}

	if err := UpdateMediaMateConfig(sr.result.ConfigPath, keys); err != nil {
		sr.logger.Error("failed to update mediamate.yaml", slog.String("error", err.Error()))
		results = append(results, SetupResult{
			Service: "config",
			Action:  "update mediamate.yaml with API keys",
			Error:   err.Error(),
		})
	} else {
		sr.logger.Info("updated mediamate.yaml with API keys")
		results = append(results, SetupResult{
			Service: "config",
			Action:  "update mediamate.yaml with API keys",
			OK:      true,
		})
	}

	return results
}

// servicePort extracts the port number from a serviceEndpoints entry.
// For example, ":7878/api/v3/health" returns "7878".
func servicePort(component string) string {
	endpoint, ok := serviceEndpoints[component]
	if !ok {
		return ""
	}
	// endpoint format is ":<port>/path..."
	endpoint = strings.TrimPrefix(endpoint, ":")
	if idx := strings.Index(endpoint, "/"); idx > 0 {
		return endpoint[:idx]
	}
	return endpoint
}

// serviceURL returns the localhost base URL for a given component.
func serviceURL(component string) string {
	port := servicePort(component)
	if port == "" {
		return ""
	}
	return "http://localhost:" + port
}

// setupRadarr creates the root folder and adds a qBittorrent download client
// to Radarr, skipping each if it already exists.
func (sr *SetupRunner) setupRadarr(ctx context.Context, apiKey string) []SetupResult {
	var results []SetupResult
	url := serviceURL(ComponentRadarr)
	client := radarr.New(url, apiKey, "", "", sr.logger)

	// Create root folder.
	results = append(results, sr.radarrCreateRootFolder(ctx, client)...)

	// Add qBittorrent download client.
	if sr.cfg.HasComponent(ComponentQBittorrent) {
		results = append(results, sr.radarrAddDownloadClient(ctx, client)...)
	}

	return results
}

// radarrCreateRootFolder creates the movies root folder in Radarr if it does
// not already exist.
func (sr *SetupRunner) radarrCreateRootFolder(ctx context.Context, client *radarr.Client) []SetupResult {
	const action = "create root folder"

	folders, err := client.ListRootFolders(ctx)
	if err != nil {
		sr.logger.Error("radarr: failed to list root folders", slog.String("error", err.Error()))
		return []SetupResult{{Service: ComponentRadarr, Action: action, Error: err.Error()}}
	}

	for _, f := range folders {
		if f.Path == sr.cfg.MoviesDir {
			sr.logger.Info("radarr: root folder already exists", slog.String("path", sr.cfg.MoviesDir))
			return []SetupResult{{Service: ComponentRadarr, Action: action, OK: true}}
		}
	}

	if _, err := client.CreateRootFolder(ctx, sr.cfg.MoviesDir); err != nil {
		sr.logger.Error("radarr: failed to create root folder", slog.String("error", err.Error()))
		return []SetupResult{{Service: ComponentRadarr, Action: action, Error: err.Error()}}
	}

	sr.logger.Info("radarr: created root folder", slog.String("path", sr.cfg.MoviesDir))
	return []SetupResult{{Service: ComponentRadarr, Action: action, OK: true}}
}

// radarrAddDownloadClient adds qBittorrent as a download client in Radarr if
// one with the same name does not already exist.
//
//nolint:dupl // radarr and prowlarr use different types, dedup would need a shared interface
func (sr *SetupRunner) radarrAddDownloadClient(ctx context.Context, client *radarr.Client) []SetupResult {
	const action = "add download client"
	const clientName = "qBittorrent"

	existing, err := client.ListDownloadClients(ctx)
	if err != nil {
		sr.logger.Error("radarr: failed to list download clients", slog.String("error", err.Error()))
		return []SetupResult{{Service: ComponentRadarr, Action: action, Error: err.Error()}}
	}

	for _, dc := range existing {
		if dc.Name == clientName {
			sr.logger.Info("radarr: download client already exists", slog.String("name", clientName))
			return []SetupResult{{Service: ComponentRadarr, Action: action, OK: true}}
		}
	}

	qbitPort := servicePort(ComponentQBittorrent)
	cfg := radarr.DownloadClientConfig{
		Name:           clientName,
		Implementation: "QBittorrent",
		ConfigContract: "QBittorrentSettings",
		Enable:         true,
		Protocol:       "torrent",
		Priority:       1,
		Fields: []radarr.DownloadClientField{
			{Name: "host", Value: "localhost"},
			{Name: "port", Value: qbitPort},
			{Name: "username", Value: qbitDefaultUser},
			{Name: "password", Value: qbitDefaultPassword},
			{Name: "movieCategory", Value: "radarr"},
		},
	}

	if err := client.AddDownloadClient(ctx, cfg); err != nil {
		sr.logger.Error("radarr: failed to add download client", slog.String("error", err.Error()))
		return []SetupResult{{Service: ComponentRadarr, Action: action, Error: err.Error()}}
	}

	sr.logger.Info("radarr: added download client", slog.String("name", clientName))
	return []SetupResult{{Service: ComponentRadarr, Action: action, OK: true}}
}

// setupProwlarr links Radarr as an application, adds the torrent download
// client, and adds FlareSolverr as an indexer proxy.
func (sr *SetupRunner) setupProwlarr(ctx context.Context, apiKey string, keys ServiceAPIKeys) []SetupResult {
	var results []SetupResult
	url := serviceURL(ComponentProwlarr)
	client := prowlarr.New(url, apiKey, sr.logger)

	// Add Radarr application.
	if sr.cfg.HasComponent(ComponentRadarr) {
		if radarrKey, ok := keys[ComponentRadarr]; ok {
			results = append(results, sr.prowlarrAddRadarr(ctx, client, radarrKey)...)
		}
	}

	// Add download client.
	if sr.cfg.HasComponent(ComponentQBittorrent) {
		results = append(results, sr.prowlarrAddDownloadClient(ctx, client)...)
	}

	// Add FlareSolverr indexer proxy.
	if sr.cfg.HasComponent(ComponentFlareSolverr) {
		results = append(results, sr.prowlarrAddFlareSolverr(ctx, client)...)
	}

	return results
}

// prowlarrAddRadarr adds Radarr as an application in Prowlarr if it does not
// already exist.
func (sr *SetupRunner) prowlarrAddRadarr(ctx context.Context, client *prowlarr.Client, radarrAPIKey string) []SetupResult {
	const action = "add radarr application"
	const appName = "Radarr"

	existing, err := client.ListApplications(ctx)
	if err != nil {
		sr.logger.Error("prowlarr: failed to list applications", slog.String("error", err.Error()))
		return []SetupResult{{Service: ComponentProwlarr, Action: action, Error: err.Error()}}
	}

	for _, app := range existing {
		if app.Name == appName {
			sr.logger.Info("prowlarr: radarr application already linked", slog.String("name", appName))
			return []SetupResult{{Service: ComponentProwlarr, Action: action, OK: true}}
		}
	}

	radarrPort := servicePort(ComponentRadarr)
	radarrBaseURL := fmt.Sprintf("http://localhost:%s", radarrPort)

	app := prowlarr.Application{
		Name:           appName,
		Implementation: "Radarr",
		ConfigContract: "RadarrSettings",
		SyncLevel:      "fullSync",
		Fields: []prowlarr.Field{
			{Name: "prowlarrUrl", Value: serviceURL(ComponentProwlarr)},
			{Name: "baseUrl", Value: radarrBaseURL},
			{Name: "apiKey", Value: radarrAPIKey},
		},
	}

	if err := client.AddApplication(ctx, app); err != nil {
		sr.logger.Error("prowlarr: failed to add radarr application", slog.String("error", err.Error()))
		return []SetupResult{{Service: ComponentProwlarr, Action: action, Error: err.Error()}}
	}

	sr.logger.Info("prowlarr: added radarr application")
	return []SetupResult{{Service: ComponentProwlarr, Action: action, OK: true}}
}

// prowlarrAddDownloadClient adds qBittorrent as a download client in Prowlarr
// if one with the same name does not already exist.
//
//nolint:dupl // prowlarr and radarr use different types, dedup would need a shared interface
func (sr *SetupRunner) prowlarrAddDownloadClient(ctx context.Context, client *prowlarr.Client) []SetupResult {
	const action = "add download client"
	const clientName = "qBittorrent"

	existing, err := client.ListDownloadClients(ctx)
	if err != nil {
		sr.logger.Error("prowlarr: failed to list download clients", slog.String("error", err.Error()))
		return []SetupResult{{Service: ComponentProwlarr, Action: action, Error: err.Error()}}
	}

	for _, dc := range existing {
		if dc.Name == clientName {
			sr.logger.Info("prowlarr: download client already exists", slog.String("name", clientName))
			return []SetupResult{{Service: ComponentProwlarr, Action: action, OK: true}}
		}
	}

	qbitPort := servicePort(ComponentQBittorrent)
	dc := prowlarr.DownloadClient{
		Name:           clientName,
		Implementation: "QBittorrent",
		ConfigContract: "QBittorrentSettings",
		Enable:         true,
		Protocol:       "torrent",
		Priority:       1,
		Fields: []prowlarr.Field{
			{Name: "host", Value: "localhost"},
			{Name: "port", Value: qbitPort},
			{Name: "username", Value: qbitDefaultUser},
			{Name: "password", Value: qbitDefaultPassword},
			{Name: "category", Value: "prowlarr"},
		},
	}

	if err := client.AddDownloadClient(ctx, dc); err != nil {
		sr.logger.Error("prowlarr: failed to add download client", slog.String("error", err.Error()))
		return []SetupResult{{Service: ComponentProwlarr, Action: action, Error: err.Error()}}
	}

	sr.logger.Info("prowlarr: added download client", slog.String("name", clientName))
	return []SetupResult{{Service: ComponentProwlarr, Action: action, OK: true}}
}

// prowlarrAddFlareSolverr adds FlareSolverr as an indexer proxy in Prowlarr
// if one with the same name does not already exist.
func (sr *SetupRunner) prowlarrAddFlareSolverr(ctx context.Context, client *prowlarr.Client) []SetupResult {
	const action = "add flaresolverr proxy"
	const proxyName = "FlareSolverr"

	existing, err := client.ListIndexerProxies(ctx)
	if err != nil {
		sr.logger.Error("prowlarr: failed to list indexer proxies", slog.String("error", err.Error()))
		return []SetupResult{{Service: ComponentProwlarr, Action: action, Error: err.Error()}}
	}

	for _, p := range existing {
		if p.Name == proxyName {
			sr.logger.Info("prowlarr: flaresolverr proxy already exists", slog.String("name", proxyName))
			return []SetupResult{{Service: ComponentProwlarr, Action: action, OK: true}}
		}
	}

	flareSolverrURL := serviceURL(ComponentFlareSolverr)
	proxy := prowlarr.IndexerProxy{
		Name:           proxyName,
		Implementation: "FlareSolverr",
		ConfigContract: "FlareSolverrSettings",
		Fields: []prowlarr.Field{
			{Name: "host", Value: flareSolverrURL},
			{Name: "requestTimeout", Value: 60},
		},
	}

	if err := client.AddIndexerProxy(ctx, proxy); err != nil {
		sr.logger.Error("prowlarr: failed to add flaresolverr proxy", slog.String("error", err.Error()))
		return []SetupResult{{Service: ComponentProwlarr, Action: action, Error: err.Error()}}
	}

	sr.logger.Info("prowlarr: added flaresolverr proxy")
	return []SetupResult{{Service: ComponentProwlarr, Action: action, OK: true}}
}

// setupQBittorrent sets the default download save path in qBittorrent.
func (sr *SetupRunner) setupQBittorrent(ctx context.Context) []SetupResult {
	const action = "set download path"

	qbitURL := serviceURL(ComponentQBittorrent)
	client, err := qbittorrent.New(qbitURL, qbitDefaultUser, qbitDefaultPassword, sr.logger)
	if err != nil {
		sr.logger.Error("qbittorrent: failed to create client", slog.String("error", err.Error()))
		return []SetupResult{{Service: ComponentQBittorrent, Action: action, Error: err.Error()}}
	}

	prefs := map[string]any{
		"save_path": sr.cfg.DownloadsDir,
	}
	if err := client.SetPreferences(ctx, prefs); err != nil {
		sr.logger.Error("qbittorrent: failed to set preferences", slog.String("error", err.Error()))
		return []SetupResult{{Service: ComponentQBittorrent, Action: action, Error: err.Error()}}
	}

	sr.logger.Info("qbittorrent: set download path", slog.String("path", sr.cfg.DownloadsDir))
	return []SetupResult{{Service: ComponentQBittorrent, Action: action, OK: true}}
}
