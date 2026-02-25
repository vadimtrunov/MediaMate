// Package stack defines types, constants, and defaults for the Docker Compose
// stack initialization feature. It describes the available media-stack
// components, their categories, and the resulting configuration produced by the
// setup wizard.
package stack

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Component name constants used throughout the stack configuration.
const (
	ComponentRadarr       = "radarr"
	ComponentSonarr       = "sonarr"
	ComponentReadarr      = "readarr"
	ComponentProwlarr     = "prowlarr"
	ComponentQBittorrent  = "qbittorrent"
	ComponentTransmission = "transmission"
	ComponentDeluge       = "deluge"
	ComponentJellyfin     = "jellyfin"
	ComponentPlex         = "plex"
	ComponentGluetun      = "gluetun"
	ComponentFlareSolverr = "flaresolverr"
	ComponentMediaMate    = "mediamate"
)

// dockerImages maps component names to their multi-arch Docker image references.
var dockerImages = map[string]string{
	ComponentRadarr:       "lscr.io/linuxserver/radarr:latest",
	ComponentSonarr:       "lscr.io/linuxserver/sonarr:latest",
	ComponentReadarr:      "lscr.io/linuxserver/readarr:latest",
	ComponentProwlarr:     "lscr.io/linuxserver/prowlarr:latest",
	ComponentQBittorrent:  "lscr.io/linuxserver/qbittorrent:latest",
	ComponentTransmission: "lscr.io/linuxserver/transmission:latest",
	ComponentDeluge:       "lscr.io/linuxserver/deluge:latest",
	ComponentJellyfin:     "lscr.io/linuxserver/jellyfin:latest",
	ComponentPlex:         "lscr.io/linuxserver/plex:latest",
	ComponentGluetun:      "qmcgaw/gluetun:latest",
	ComponentFlareSolverr: "ghcr.io/flaresolverr/flaresolverr:latest",
	ComponentMediaMate:    "ghcr.io/vadimtrunov/mediamate:latest",
}

// DockerImage returns the Docker image reference for the given component.
// It returns an empty string if the component is unknown.
func DockerImage(component string) string {
	return dockerImages[component]
}

// ComponentCategory describes a group of related stack components that a user
// can choose from during the setup wizard. For example, the "Torrents" category
// offers qBittorrent, Transmission, and Deluge.
type ComponentCategory struct {
	Name        string   // human-readable category name, e.g. "Movies"
	Description string   // short description, e.g. "Movie management"
	Options     []string // available component names for this category
	Default     string   // pre-selected option; empty if none
	Required    bool     // when true the user must pick at least one option
	MultiSelect bool     // when true the user may pick more than one option
}

// DefaultCategories returns the standard set of component categories with their
// defaults. The order matches the recommended wizard presentation order.
func DefaultCategories() []ComponentCategory {
	return []ComponentCategory{
		{
			Name:        "Movies",
			Description: "Movie management",
			Options:     []string{ComponentRadarr},
			Default:     ComponentRadarr,
			Required:    false,
		},
		{
			Name:        "TV Shows",
			Description: "TV show management",
			Options:     []string{ComponentSonarr},
			Default:     ComponentSonarr,
			Required:    false,
		},
		{
			Name:        "Books",
			Description: "Book and audiobook management",
			Options:     []string{ComponentReadarr},
			Default:     "",
			Required:    false,
		},
		{
			Name:        "Indexers",
			Description: "Torrent and Usenet indexer management",
			Options:     []string{ComponentProwlarr},
			Default:     ComponentProwlarr,
			Required:    false,
		},
		{
			Name:        "Torrents",
			Description: "Torrent download client",
			Options:     []string{ComponentQBittorrent, ComponentTransmission, ComponentDeluge},
			Default:     ComponentQBittorrent,
			Required:    true,
		},
		{
			Name:        "Streaming",
			Description: "Media streaming server",
			Options:     []string{ComponentJellyfin, ComponentPlex},
			Default:     ComponentJellyfin,
			Required:    false,
		},
		{
			Name:        "VPN",
			Description: "VPN tunnel for torrent traffic",
			Options:     []string{ComponentGluetun},
			Default:     "",
			Required:    false,
		},
	}
}

// Config holds the final configuration produced by the setup wizard. It
// describes which components are enabled, where media files live, and where
// generated files should be written.
type Config struct {
	// Components lists the selected component names (e.g. "radarr", "sonarr").
	Components []string

	// MediaDir is the root media directory.
	MediaDir string
	// MoviesDir is the movies subdirectory.
	MoviesDir string
	// TVDir is the TV shows subdirectory.
	TVDir string
	// BooksDir is the books subdirectory.
	BooksDir string
	// DownloadsDir is the downloads directory.
	DownloadsDir string
	// ConfigDir is the configuration directory for services.
	ConfigDir string

	// OutputDir is the directory where generated files are written.
	OutputDir string

	// TorrentClient is the chosen torrent client ("qbittorrent", "transmission",
	// or "deluge").
	TorrentClient string

	// MediaServer is the chosen media server ("jellyfin" or "plex").
	MediaServer string

	// WebhookPort is the port for the MediaMate webhook server (default 8080).
	WebhookPort int
	// WebhookSecret is the shared secret for webhook authentication.
	WebhookSecret string
}

// DefaultConfig returns a Config populated with sensible defaults
// suitable for a typical media server setup.
func DefaultConfig() Config {
	return Config{
		Components: []string{
			ComponentRadarr,
			ComponentSonarr,
			ComponentProwlarr,
			ComponentQBittorrent,
			ComponentJellyfin,
			ComponentMediaMate,
		},
		MediaDir:      "/srv/media",
		MoviesDir:     "/srv/media/movies",
		TVDir:         "/srv/media/tv",
		BooksDir:      "/srv/media/books",
		DownloadsDir:  "/srv/media/downloads",
		ConfigDir:     "/srv/mediamate/config",
		OutputDir:     ".",
		TorrentClient: ComponentQBittorrent,
		MediaServer:   ComponentJellyfin,
		WebhookPort:   8080,
	}
}

// HasComponent reports whether the given component name is present in the
// configuration's selected components list.
func (c *Config) HasComponent(name string) bool {
	for _, comp := range c.Components {
		if comp == name {
			return true
		}
	}
	return false
}

// knownComponents is the set of valid component names. Used by
// LoadConfigFromCompose to filter service names from docker-compose.yml.
var knownComponents = map[string]bool{
	ComponentRadarr:       true,
	ComponentSonarr:       true,
	ComponentReadarr:      true,
	ComponentProwlarr:     true,
	ComponentQBittorrent:  true,
	ComponentTransmission: true,
	ComponentDeluge:       true,
	ComponentJellyfin:     true,
	ComponentPlex:         true,
	ComponentGluetun:      true,
	ComponentFlareSolverr: true,
	ComponentMediaMate:    true,
}

// torrentComponents lists all torrent client component names.
var torrentComponents = map[string]bool{
	ComponentQBittorrent:  true,
	ComponentTransmission: true,
	ComponentDeluge:       true,
}

// mediaServerComponents lists all media server component names.
var mediaServerComponents = map[string]bool{
	ComponentJellyfin: true,
	ComponentPlex:     true,
}

// serviceNameRe matches a top-level service definition in docker-compose.yml.
// It relies on exactly 2-space indentation, which is the format produced by
// our Compose generator. Hand-edited files with different indent may not parse.
var serviceNameRe = regexp.MustCompile(`^ {2}(\w[\w-]*):\s*$`)

// LoadConfigFromCompose reads a docker-compose.yml and .env file from dir and
// returns a Config whose Components list reflects the actual services defined
// in the compose file. Directory paths are read from the .env file; any
// missing values fall back to DefaultConfig defaults.
func LoadConfigFromCompose(dir string) (Config, error) {
	composePath := filepath.Join(dir, "docker-compose.yml")
	envPath := filepath.Join(dir, ".env")

	// Start from defaults so any missing values are populated.
	cfg := DefaultConfig()

	// --- Parse docker-compose.yml for service names ---
	components, err := parseComposeServices(composePath)
	if err != nil {
		return Config{}, fmt.Errorf("load config from compose: %w", err)
	}

	// Always include MediaMate itself so that setup steps guarded by
	// HasComponent(ComponentMediaMate) (e.g. webhook registration) run.
	components = append(components, ComponentMediaMate)
	cfg.Components = components

	// Derive TorrentClient and MediaServer from the component list.
	cfg.TorrentClient = ""
	for _, c := range components {
		if torrentComponents[c] {
			cfg.TorrentClient = c
			break
		}
	}
	cfg.MediaServer = ""
	for _, c := range components {
		if mediaServerComponents[c] {
			cfg.MediaServer = c
			break
		}
	}

	// --- Parse .env for directory paths ---
	envVars, err := parseEnvFile(envPath)
	if err != nil {
		// .env is optional; log and continue with defaults.
		slog.Debug("could not read .env file, using default paths",
			slog.String("path", envPath),
			slog.String("error", err.Error()),
		)
		return cfg, nil
	}

	applyEnvDir(envVars, "CONFIG_DIR", &cfg.ConfigDir)
	applyEnvDir(envVars, "MOVIES_DIR", &cfg.MoviesDir)
	applyEnvDir(envVars, "DOWNLOADS_DIR", &cfg.DownloadsDir)
	applyEnvDir(envVars, "TV_DIR", &cfg.TVDir)
	applyEnvDir(envVars, "BOOKS_DIR", &cfg.BooksDir)
	applyEnvDir(envVars, "MEDIA_DIR", &cfg.MediaDir)

	return cfg, nil
}

// parseComposeServices reads a docker-compose.yml file and extracts the
// top-level service names. It skips the "mediamate" service itself and only
// returns names that are known components.
func parseComposeServices(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var components []string
	inServices := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Detect the start of the services block.
		if strings.TrimSpace(line) == "services:" {
			inServices = true
			continue
		}

		// A top-level key (no indent, ends with colon) after services:
		// means we left the services block (e.g. "networks:").
		// Skip YAML comments at column 0 — they are valid inside any block.
		if inServices && line != "" && line[0] != ' ' && line[0] != '\t' {
			if line[0] == '#' {
				continue
			}
			break
		}

		if !inServices {
			continue
		}

		// Match service names (exactly 2-space indent).
		matches := serviceNameRe.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		name := matches[1]
		if name == ComponentMediaMate {
			continue
		}
		if knownComponents[name] {
			components = append(components, name)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	if len(components) == 0 {
		return nil, fmt.Errorf("no known services found in %s", path)
	}

	return components, nil
}

// parseEnvFile reads a .env file and returns a map of key=value pairs.
// Lines starting with # and empty lines are skipped.
func parseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		vars[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return vars, nil
}

// applyEnvDir sets *dst to the value of envVars[key] when the key is present
// and non-empty.
func applyEnvDir(envVars map[string]string, key string, dst *string) {
	if v := envVars[key]; v != "" {
		*dst = v
	}
}
