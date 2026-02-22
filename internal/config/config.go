package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the main application configuration
type Config struct {
	// LLM configuration
	LLM LLMConfig `yaml:"llm"`

	// Media backends
	Radarr  *RadarrConfig  `yaml:"radarr,omitempty"`
	Sonarr  *SonarrConfig  `yaml:"sonarr,omitempty"`
	Readarr *ReadarrConfig `yaml:"readarr,omitempty"`

	// Torrent clients
	QBittorrent *QBittorrentConfig `yaml:"qbittorrent,omitempty"`

	// Media servers
	Jellyfin *JellyfinConfig `yaml:"jellyfin,omitempty"`

	// Frontends
	Telegram *TelegramConfig `yaml:"telegram,omitempty"`

	// Metadata providers
	TMDb TMDbConfig `yaml:"tmdb"`

	// Application settings
	App AppConfig `yaml:"app"`
}

// LLMConfig holds LLM provider configuration
type LLMConfig struct {
	Provider string `yaml:"provider"` // "claude", "openai", "ollama"
	APIKey   string `yaml:"api_key"`
	Model    string `yaml:"model,omitempty"`
	BaseURL  string `yaml:"base_url,omitempty"` // For Ollama
}

// RadarrConfig holds Radarr configuration
type RadarrConfig struct {
	URL            string `yaml:"url"`
	APIKey         string `yaml:"api_key"`
	QualityProfile string `yaml:"quality_profile,omitempty"`
	RootFolder     string `yaml:"root_folder,omitempty"`
}

// SonarrConfig holds Sonarr configuration
type SonarrConfig struct {
	URL            string `yaml:"url"`
	APIKey         string `yaml:"api_key"`
	QualityProfile string `yaml:"quality_profile,omitempty"`
	RootFolder     string `yaml:"root_folder,omitempty"`
}

// ReadarrConfig holds Readarr configuration
type ReadarrConfig struct {
	URL            string `yaml:"url"`
	APIKey         string `yaml:"api_key"`
	QualityProfile string `yaml:"quality_profile,omitempty"`
	RootFolder     string `yaml:"root_folder,omitempty"`
}

// QBittorrentConfig holds qBittorrent configuration
type QBittorrentConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// JellyfinConfig holds Jellyfin configuration
type JellyfinConfig struct {
	URL    string `yaml:"url"`
	APIKey string `yaml:"api_key"`
}

// TelegramConfig holds Telegram bot configuration
type TelegramConfig struct {
	BotToken       string   `yaml:"bot_token"`
	AllowedUserIDs []int64  `yaml:"allowed_user_ids,omitempty"`
}

// TMDbConfig holds TMDb API configuration
type TMDbConfig struct {
	APIKey string `yaml:"api_key"`
}

// AppConfig holds application-level settings
type AppConfig struct {
	LogLevel string `yaml:"log_level"` // "debug", "info", "warn", "error"
	DataDir  string `yaml:"data_dir"`  // Directory for database and cache
}

// Load loads configuration from a YAML file with environment variable overrides
func Load(path string) (*Config, error) {
	// Read YAML file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Override with environment variables
	cfg.applyEnvOverrides()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// applyEnvOverrides overrides config values with environment variables
func (c *Config) applyEnvOverrides() {
	// LLM
	if v := os.Getenv("MEDIAMATE_LLM_PROVIDER"); v != "" {
		c.LLM.Provider = v
	}
	if v := os.Getenv("MEDIAMATE_LLM_API_KEY"); v != "" {
		c.LLM.APIKey = v
	}
	if v := os.Getenv("MEDIAMATE_LLM_MODEL"); v != "" {
		c.LLM.Model = v
	}

	// TMDb
	if v := os.Getenv("MEDIAMATE_TMDB_API_KEY"); v != "" {
		c.TMDb.APIKey = v
	}

	// Radarr
	if c.Radarr != nil {
		if v := os.Getenv("MEDIAMATE_RADARR_URL"); v != "" {
			c.Radarr.URL = v
		}
		if v := os.Getenv("MEDIAMATE_RADARR_API_KEY"); v != "" {
			c.Radarr.APIKey = v
		}
	}

	// qBittorrent
	if c.QBittorrent != nil {
		if v := os.Getenv("MEDIAMATE_QBITTORRENT_URL"); v != "" {
			c.QBittorrent.URL = v
		}
		if v := os.Getenv("MEDIAMATE_QBITTORRENT_USERNAME"); v != "" {
			c.QBittorrent.Username = v
		}
		if v := os.Getenv("MEDIAMATE_QBITTORRENT_PASSWORD"); v != "" {
			c.QBittorrent.Password = v
		}
	}

	// Jellyfin
	if c.Jellyfin != nil {
		if v := os.Getenv("MEDIAMATE_JELLYFIN_URL"); v != "" {
			c.Jellyfin.URL = v
		}
		if v := os.Getenv("MEDIAMATE_JELLYFIN_API_KEY"); v != "" {
			c.Jellyfin.APIKey = v
		}
	}

	// Telegram
	if c.Telegram != nil {
		if v := os.Getenv("MEDIAMATE_TELEGRAM_BOT_TOKEN"); v != "" {
			c.Telegram.BotToken = v
		}
	}

	// App
	if v := os.Getenv("MEDIAMATE_LOG_LEVEL"); v != "" {
		c.App.LogLevel = v
	}
	if v := os.Getenv("MEDIAMATE_DATA_DIR"); v != "" {
		c.App.DataDir = v
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate LLM provider
	if c.LLM.Provider == "" {
		return fmt.Errorf("llm.provider is required")
	}
	if c.LLM.Provider != "claude" && c.LLM.Provider != "openai" && c.LLM.Provider != "ollama" {
		return fmt.Errorf("llm.provider must be 'claude', 'openai', or 'ollama'")
	}
	if c.LLM.Provider != "ollama" && c.LLM.APIKey == "" {
		return fmt.Errorf("llm.api_key is required for provider '%s'", c.LLM.Provider)
	}

	// Validate TMDb
	if c.TMDb.APIKey == "" {
		return fmt.Errorf("tmdb.api_key is required")
	}

	// Validate at least one media backend is configured
	if c.Radarr == nil && c.Sonarr == nil && c.Readarr == nil {
		return fmt.Errorf("at least one media backend (radarr, sonarr, readarr) must be configured")
	}

	// Validate Radarr if configured
	if c.Radarr != nil {
		if c.Radarr.URL == "" {
			return fmt.Errorf("radarr.url is required")
		}
		if c.Radarr.APIKey == "" {
			return fmt.Errorf("radarr.api_key is required")
		}
	}

	// Set defaults
	if c.App.LogLevel == "" {
		c.App.LogLevel = "info"
	}
	if c.App.DataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		c.App.DataDir = filepath.Join(homeDir, ".mediamate")
	}

	return nil
}
