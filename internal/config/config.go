package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the main application configuration
type Config struct {
	// LLM configuration
	LLM LLMConfig `yaml:"llm"`

	// Media backends
	Radarr  *ArrConfig `yaml:"radarr,omitempty"`
	Sonarr  *ArrConfig `yaml:"sonarr,omitempty"`
	Readarr *ArrConfig `yaml:"readarr,omitempty"`

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

// ArrConfig holds configuration for *arr backends (Radarr, Sonarr, Readarr)
type ArrConfig struct {
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
	BotToken       string  `yaml:"bot_token"`
	AllowedUserIDs []int64 `yaml:"allowed_user_ids,omitempty"`
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

// validateConfigPath validates that the config path points to an existing file
func validateConfigPath(path string) error {
	cleanPath := filepath.Clean(path)

	info, err := os.Stat(cleanPath)
	if err != nil {
		return fmt.Errorf("config file not found: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("config path is a directory, not a file: %s", cleanPath)
	}

	return nil
}

// Load loads configuration from a YAML file with environment variable overrides
func Load(path string) (*Config, error) {
	if err := validateConfigPath(path); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Clean(path)) // #nosec G304 -- path is validated by validateConfigPath
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg.applyEnvOverrides()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// applyEnvOverrides overrides config values with environment variables
func (c *Config) applyEnvOverrides() {
	c.applyLLMEnv()
	c.applyMetadataEnv()
	c.applyBackendsEnv()
	c.applyTorrentEnv()
	c.applyMediaServerEnv()
	c.applyFrontendsEnv()
	c.applyAppEnv()
}

func (c *Config) applyLLMEnv() {
	if v := os.Getenv("MEDIAMATE_LLM_PROVIDER"); v != "" {
		c.LLM.Provider = v
	}
	if v := os.Getenv("MEDIAMATE_LLM_API_KEY"); v != "" {
		c.LLM.APIKey = v
	}
	if v := os.Getenv("MEDIAMATE_LLM_MODEL"); v != "" {
		c.LLM.Model = v
	}
	if v := os.Getenv("MEDIAMATE_LLM_BASE_URL"); v != "" {
		c.LLM.BaseURL = v
	}
}

func (c *Config) applyMetadataEnv() {
	if v := os.Getenv("MEDIAMATE_TMDB_API_KEY"); v != "" {
		c.TMDb.APIKey = v
	}
}

func (c *Config) applyBackendsEnv() {
	c.Radarr = applyArrEnv(c.Radarr, "MEDIAMATE_RADARR_URL", "MEDIAMATE_RADARR_API_KEY")
	c.Sonarr = applyArrEnv(c.Sonarr, "MEDIAMATE_SONARR_URL", "MEDIAMATE_SONARR_API_KEY")
	c.Readarr = applyArrEnv(c.Readarr, "MEDIAMATE_READARR_URL", "MEDIAMATE_READARR_API_KEY")
}

func applyArrEnv(cfg *ArrConfig, urlEnv, keyEnv string) *ArrConfig {
	envURL := os.Getenv(urlEnv)
	envKey := os.Getenv(keyEnv)
	if envURL == "" && envKey == "" {
		return cfg
	}
	if cfg == nil {
		cfg = &ArrConfig{}
	}
	if envURL != "" {
		cfg.URL = envURL
	}
	if envKey != "" {
		cfg.APIKey = envKey
	}
	return cfg
}

func (c *Config) applyTorrentEnv() {
	qbitURL := os.Getenv("MEDIAMATE_QBITTORRENT_URL")
	qbitUser := os.Getenv("MEDIAMATE_QBITTORRENT_USERNAME")
	qbitPass := os.Getenv("MEDIAMATE_QBITTORRENT_PASSWORD")
	if qbitURL != "" || qbitUser != "" || qbitPass != "" {
		if c.QBittorrent == nil {
			c.QBittorrent = &QBittorrentConfig{}
		}
		if qbitURL != "" {
			c.QBittorrent.URL = qbitURL
		}
		if qbitUser != "" {
			c.QBittorrent.Username = qbitUser
		}
		if qbitPass != "" {
			c.QBittorrent.Password = qbitPass
		}
	}
}

func (c *Config) applyMediaServerEnv() {
	jellyfinURL := os.Getenv("MEDIAMATE_JELLYFIN_URL")
	jellyfinKey := os.Getenv("MEDIAMATE_JELLYFIN_API_KEY")
	if jellyfinURL != "" || jellyfinKey != "" {
		if c.Jellyfin == nil {
			c.Jellyfin = &JellyfinConfig{}
		}
		if jellyfinURL != "" {
			c.Jellyfin.URL = jellyfinURL
		}
		if jellyfinKey != "" {
			c.Jellyfin.APIKey = jellyfinKey
		}
	}
}

func (c *Config) applyFrontendsEnv() {
	telegramToken := os.Getenv("MEDIAMATE_TELEGRAM_BOT_TOKEN")
	if telegramToken != "" {
		if c.Telegram == nil {
			c.Telegram = &TelegramConfig{}
		}
		c.Telegram.BotToken = telegramToken
	}
}

func (c *Config) applyAppEnv() {
	if v := os.Getenv("MEDIAMATE_LOG_LEVEL"); v != "" {
		c.App.LogLevel = v
	}
	if v := os.Getenv("MEDIAMATE_DATA_DIR"); v != "" {
		c.App.DataDir = v
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	c.setDefaults()

	if err := c.validateLLM(); err != nil {
		return err
	}

	if err := c.validateMetadata(); err != nil {
		return err
	}

	if err := c.validateBackends(); err != nil {
		return err
	}

	if err := c.validateOptionalServices(); err != nil {
		return err
	}

	return c.validateApp()
}

func (c *Config) validateLLM() error {
	if c.LLM.Provider == "" {
		return fmt.Errorf("llm.provider is required")
	}
	if c.LLM.Provider != "claude" {
		return fmt.Errorf("llm.provider must be 'claude' (only supported provider in this version)")
	}
	if c.LLM.APIKey == "" {
		return fmt.Errorf("llm.api_key is required for provider '%s'", c.LLM.Provider)
	}
	return nil
}

func (c *Config) validateMetadata() error {
	if c.TMDb.APIKey == "" {
		return fmt.Errorf("tmdb.api_key is required")
	}
	return nil
}

func validateArrConfig(cfg *ArrConfig, name string) error {
	if cfg == nil {
		return nil
	}
	if cfg.URL == "" {
		return fmt.Errorf("%s.url is required", name)
	}
	if err := validateURL(cfg.URL, name+".url"); err != nil {
		return err
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("%s.api_key is required", name)
	}
	return nil
}

func (c *Config) validateBackends() error {
	if c.Radarr == nil && c.Sonarr == nil && c.Readarr == nil {
		return fmt.Errorf("at least one media backend (radarr, sonarr, readarr) must be configured")
	}

	if err := validateArrConfig(c.Radarr, "radarr"); err != nil {
		return err
	}
	if err := validateArrConfig(c.Sonarr, "sonarr"); err != nil {
		return err
	}

	return validateArrConfig(c.Readarr, "readarr")
}

func (c *Config) validateOptionalServices() error {
	if c.QBittorrent != nil {
		if c.QBittorrent.URL == "" {
			return fmt.Errorf("qbittorrent.url is required when qbittorrent is configured")
		}
		if err := validateURL(c.QBittorrent.URL, "qbittorrent.url"); err != nil {
			return err
		}
		if c.QBittorrent.Username == "" {
			return fmt.Errorf("qbittorrent.username is required when qbittorrent is configured")
		}
		if c.QBittorrent.Password == "" {
			return fmt.Errorf("qbittorrent.password is required when qbittorrent is configured")
		}
	}

	if c.Jellyfin != nil {
		if c.Jellyfin.URL == "" {
			return fmt.Errorf("jellyfin.url is required when jellyfin is configured")
		}
		if err := validateURL(c.Jellyfin.URL, "jellyfin.url"); err != nil {
			return err
		}
		if c.Jellyfin.APIKey == "" {
			return fmt.Errorf("jellyfin.api_key is required when jellyfin is configured")
		}
	}

	if c.Telegram != nil {
		if c.Telegram.BotToken == "" {
			return fmt.Errorf("telegram.bot_token is required when telegram is configured")
		}
	}

	return nil
}

func validateURL(rawURL, fieldName string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%s is not a valid URL: %w", fieldName, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s must use http or https scheme, got %q", fieldName, u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("%s is missing host", fieldName)
	}
	return nil
}

var validLogLevels = map[string]bool{
	"debug": true, "info": true, "warn": true, "warning": true, "error": true,
}

func (c *Config) validateApp() error {
	if !validLogLevels[c.App.LogLevel] {
		return fmt.Errorf("app.log_level must be one of: debug, info, warn, error; got %q", c.App.LogLevel)
	}
	return nil
}

func (c *Config) setDefaults() {
	if c.App.LogLevel == "" {
		c.App.LogLevel = "info"
	}
	if c.App.DataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil || homeDir == "" {
			homeDir = "."
		}
		c.App.DataDir = filepath.Join(homeDir, ".mediamate")
	}
}
