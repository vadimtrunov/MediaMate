// Package stack defines types, constants, and defaults for the Docker Compose
// stack initialization feature. It describes the available media-stack
// components, their categories, and the resulting configuration produced by the
// setup wizard.
package stack

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
