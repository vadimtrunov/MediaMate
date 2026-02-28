package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type validateCase struct {
	name    string
	modify  func(*Config)
	wantErr string
}

// validConfig returns a minimal Config that passes Validate().
func validConfig() Config {
	return Config{
		LLM:  LLMConfig{Provider: "claude", APIKey: "test-key"},
		TMDb: TMDbConfig{APIKey: "tmdb-key"},
		Radarr: &ArrConfig{
			URL:    "http://localhost:7878",
			APIKey: "radarr-key",
		},
		App: AppConfig{LogLevel: "info", DataDir: "/tmp/test"},
	}
}

func TestValidate_CoreFields(t *testing.T) {
	t.Parallel()

	tests := []validateCase{
		{"valid_claude", nil, ""},
		{"valid_openai", func(c *Config) { c.LLM.Provider = "openai" }, ""},
		{"valid_ollama_no_apikey", func(c *Config) {
			c.LLM.Provider = "ollama"
			c.LLM.APIKey = ""
		}, ""},
		{"missing_provider", func(c *Config) { c.LLM.Provider = "" }, "llm.provider is required"},
		{"invalid_provider", func(c *Config) { c.LLM.Provider = "gemini" }, "llm.provider must be"},
		{"claude_no_apikey", func(c *Config) { c.LLM.APIKey = "" }, "llm.api_key is required"},
		{"openai_no_apikey", func(c *Config) {
			c.LLM.Provider = "openai"
			c.LLM.APIKey = ""
		}, "llm.api_key is required"},
		{"missing_tmdb_key", func(c *Config) { c.TMDb.APIKey = "" }, "tmdb.api_key is required"},
		{"no_backends", func(c *Config) { c.Radarr = nil }, "at least one media backend"},
		{"radarr_missing_url", func(c *Config) { c.Radarr.URL = "" }, "radarr.url is required"},
		{"radarr_missing_apikey", func(c *Config) { c.Radarr.APIKey = "" }, "radarr.api_key is required"},
		{"radarr_invalid_scheme", func(c *Config) {
			c.Radarr.URL = "ftp://localhost:7878"
		}, "must use http or https"},
		{"radarr_url_no_host", func(c *Config) { c.Radarr.URL = "http://" }, "missing host"},
		{"invalid_log_level", func(c *Config) { c.App.LogLevel = "trace" }, "app.log_level must be one of"},
		{"warning_accepted", func(c *Config) { c.App.LogLevel = "warning" }, ""},
	}

	runValidateTests(t, tests)
}

func TestValidate_OptionalServices(t *testing.T) {
	t.Parallel()

	tests := []validateCase{
		{"sonarr_only_valid", func(c *Config) {
			c.Radarr = nil
			c.Sonarr = &ArrConfig{URL: "http://localhost:8989", APIKey: "key"}
		}, ""},
		{"readarr_only_valid", func(c *Config) {
			c.Radarr = nil
			c.Readarr = &ArrConfig{URL: "http://localhost:8787", APIKey: "key"}
		}, ""},
		{"qbit_missing_url", func(c *Config) {
			c.QBittorrent = &QBittorrentConfig{Username: "u", Password: "p"}
		}, "qbittorrent.url is required"},
		{"qbit_missing_user", func(c *Config) {
			c.QBittorrent = &QBittorrentConfig{URL: "http://localhost:8080", Password: "p"}
		}, "qbittorrent.username is required"},
		{"qbit_missing_pass", func(c *Config) {
			c.QBittorrent = &QBittorrentConfig{URL: "http://localhost:8080", Username: "u"}
		}, "qbittorrent.password is required"},
		{"jellyfin_missing_url", func(c *Config) {
			c.Jellyfin = &JellyfinConfig{APIKey: "key"}
		}, "jellyfin.url is required"},
		{"jellyfin_missing_apikey", func(c *Config) {
			c.Jellyfin = &JellyfinConfig{URL: "http://localhost:8096"}
		}, "jellyfin.api_key is required"},
		{"telegram_missing_token", func(c *Config) {
			c.Telegram = &TelegramConfig{}
		}, "telegram.bot_token is required"},
		{"webhook_port_negative", func(c *Config) {
			c.Webhook = &WebhookConfig{Port: -1}
		}, "webhook.port must be between 1 and 65535"},
		{"webhook_port_too_high", func(c *Config) {
			c.Webhook = &WebhookConfig{Port: 65536}
		}, "webhook.port must be between 1 and 65535"},
		{"webhook_port_max_valid", func(c *Config) {
			c.Webhook = &WebhookConfig{Port: 65535, Secret: "s"}
		}, ""},
		{"webhook_port_zero_gets_default", func(c *Config) {
			c.Webhook = &WebhookConfig{Port: 0, Secret: "s"}
		}, ""},
		{"webhook_missing_secret", func(c *Config) {
			c.Webhook = &WebhookConfig{Port: 8080}
		}, "webhook.secret is required"},
	}

	runValidateTests(t, tests)
}

func runValidateTests(t *testing.T, tests []validateCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := validConfig()
			if tt.modify != nil {
				tt.modify(&cfg)
			}
			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		wantErr string
	}{
		{"valid_http", "http://localhost:7878", ""},
		{"valid_https", "https://radarr.example.com", ""},
		{"valid_with_path", "http://localhost:7878/radarr", ""},
		{"ftp_scheme", "ftp://localhost", "must use http or https"},
		{"no_scheme", "localhost:7878", "must use http or https"},
		{"empty_string", "", "must use http or https"},
		{"missing_host", "http://", "missing host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateURL(tt.url, "test.url")
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestSetDefaults_Webhook(t *testing.T) {
	t.Parallel()

	t.Run("port_default", func(t *testing.T) {
		t.Parallel()
		cfg := Config{Webhook: &WebhookConfig{Port: 0}}
		cfg.setDefaults()
		if cfg.Webhook.Port != 8080 {
			t.Errorf("expected default port 8080, got %d", cfg.Webhook.Port)
		}
	})

	t.Run("port_preserved", func(t *testing.T) {
		t.Parallel()
		cfg := Config{Webhook: &WebhookConfig{Port: 9090}}
		cfg.setDefaults()
		if cfg.Webhook.Port != 9090 {
			t.Errorf("expected port 9090, got %d", cfg.Webhook.Port)
		}
	})

	t.Run("nil_no_panic", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.setDefaults() // must not panic
		if cfg.Webhook != nil {
			t.Error("expected Webhook to remain nil")
		}
	})
}

func TestSetDefaults_ProgressInterval(t *testing.T) {
	t.Parallel()

	t.Run("negative_preserved", func(t *testing.T) {
		t.Parallel()
		cfg := Config{Webhook: &WebhookConfig{Progress: ProgressConfig{Interval: -5}}}
		cfg.setDefaults()
		if cfg.Webhook.Progress.Interval != -5 {
			t.Errorf("expected negative interval preserved as -5, got %d", cfg.Webhook.Progress.Interval)
		}
	})

	t.Run("zero_gets_default", func(t *testing.T) {
		t.Parallel()
		cfg := Config{Webhook: &WebhookConfig{Progress: ProgressConfig{Interval: 0}}}
		cfg.setDefaults()
		if cfg.Webhook.Progress.Interval != 15 {
			t.Errorf("expected default interval 15, got %d", cfg.Webhook.Progress.Interval)
		}
	})

	t.Run("positive_preserved", func(t *testing.T) {
		t.Parallel()
		cfg := Config{Webhook: &WebhookConfig{Progress: ProgressConfig{Interval: 30}}}
		cfg.setDefaults()
		if cfg.Webhook.Progress.Interval != 30 {
			t.Errorf("expected interval 30, got %d", cfg.Webhook.Progress.Interval)
		}
	})
}

func TestValidate_ProgressIntervalNegative(t *testing.T) {
	t.Parallel()

	t.Run("negative_interval_rejected_by_validate", func(t *testing.T) {
		t.Parallel()
		cfg := validConfig()
		cfg.Webhook = &WebhookConfig{Port: 8080, Secret: "s", Progress: ProgressConfig{Enabled: true, Interval: -10}}
		err := cfg.Validate()
		if err == nil || !strings.Contains(err.Error(), "webhook.progress.interval must be positive") {
			t.Fatalf("expected validation error for negative interval, got %v", err)
		}
	})
}

func TestSetDefaults_AppConfig(t *testing.T) {
	t.Parallel()

	t.Run("log_level_default", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.setDefaults()
		if cfg.App.LogLevel != "info" {
			t.Errorf("expected default log level 'info', got %q", cfg.App.LogLevel)
		}
	})

	t.Run("log_level_preserved", func(t *testing.T) {
		t.Parallel()
		cfg := Config{App: AppConfig{LogLevel: "debug"}}
		cfg.setDefaults()
		if cfg.App.LogLevel != "debug" {
			t.Errorf("expected log level 'debug', got %q", cfg.App.LogLevel)
		}
	})

	t.Run("data_dir_default", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.setDefaults()
		if cfg.App.DataDir == "" {
			t.Fatal("expected non-empty default DataDir")
		}
		if !strings.HasSuffix(cfg.App.DataDir, ".mediamate") {
			t.Errorf("expected DataDir ending in .mediamate, got %q", cfg.App.DataDir)
		}
	})

	t.Run("data_dir_preserved", func(t *testing.T) {
		t.Parallel()
		cfg := Config{App: AppConfig{DataDir: "/custom/dir"}}
		cfg.setDefaults()
		if cfg.App.DataDir != "/custom/dir" {
			t.Errorf("expected /custom/dir, got %q", cfg.App.DataDir)
		}
	})
}

func TestLoad_ValidMinimal(t *testing.T) {
	t.Parallel()
	path := writeTempYAML(t, minimalYAML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LLM.Provider != "claude" {
		t.Errorf("expected provider claude, got %q", cfg.LLM.Provider)
	}
	if cfg.LLM.APIKey != "yaml-key" {
		t.Errorf("expected api key yaml-key, got %q", cfg.LLM.APIKey)
	}
	if cfg.Radarr == nil || cfg.Radarr.URL != "http://localhost:7878" {
		t.Error("expected radarr config to be loaded")
	}
	if cfg.App.LogLevel != "info" {
		t.Errorf("expected default log level info, got %q", cfg.App.LogLevel)
	}
}

func TestLoad_Errors(t *testing.T) {
	t.Parallel()

	t.Run("invalid_yaml", func(t *testing.T) {
		t.Parallel()
		path := writeTempYAML(t, "{{invalid yaml}}")
		_, err := Load(path)
		if err == nil {
			t.Fatal("expected error for invalid YAML")
		}
		if !strings.Contains(err.Error(), "failed to parse") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("file_not_found", func(t *testing.T) {
		t.Parallel()
		_, err := Load("/nonexistent/path/config.yaml")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
		if !strings.Contains(err.Error(), "config file not found") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("path_is_directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		_, err := Load(dir)
		if err == nil {
			t.Fatal("expected error for directory path")
		}
		if !strings.Contains(err.Error(), "directory") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestLoad_AllOptionalServices(t *testing.T) {
	fullYAML := `
llm:
  provider: claude
  api_key: test-key
tmdb:
  api_key: tmdb-key
radarr:
  url: http://localhost:7878
  api_key: radarr-key
qbittorrent:
  url: http://localhost:8080
  username: admin
  password: pass
jellyfin:
  url: http://localhost:8096
  api_key: jf-key
telegram:
  bot_token: "123:ABC"
webhook:
  port: 9090
  secret: my-secret
`
	path := writeTempYAML(t, fullYAML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.QBittorrent == nil {
		t.Error("expected qbittorrent config")
	}
	if cfg.Jellyfin == nil {
		t.Error("expected jellyfin config")
	}
	if cfg.Telegram == nil || cfg.Telegram.BotToken != "123:ABC" {
		t.Error("expected telegram config")
	}
	if cfg.Webhook == nil || cfg.Webhook.Port != 9090 {
		t.Errorf("expected webhook port 9090, got %v", cfg.Webhook)
	}
}

func TestEnvOverrides_LLM(t *testing.T) {
	t.Run("api_key_override", func(t *testing.T) {
		path := writeTempYAML(t, minimalYAML)
		t.Setenv("MEDIAMATE_LLM_API_KEY", "env-key")
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.LLM.APIKey != "env-key" {
			t.Errorf("expected env-key, got %q", cfg.LLM.APIKey)
		}
	})

	t.Run("provider_override", func(t *testing.T) {
		path := writeTempYAML(t, minimalYAML)
		t.Setenv("MEDIAMATE_LLM_PROVIDER", "openai")
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.LLM.Provider != "openai" {
			t.Errorf("expected openai, got %q", cfg.LLM.Provider)
		}
	})

	t.Run("radarr_url_override", func(t *testing.T) {
		path := writeTempYAML(t, minimalYAML)
		t.Setenv("MEDIAMATE_RADARR_URL", "http://radarr:7878")
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Radarr.URL != "http://radarr:7878" {
			t.Errorf("expected http://radarr:7878, got %q", cfg.Radarr.URL)
		}
	})

	t.Run("log_level_override", func(t *testing.T) {
		path := writeTempYAML(t, minimalYAML)
		t.Setenv("MEDIAMATE_LOG_LEVEL", "debug")
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.App.LogLevel != "debug" {
			t.Errorf("expected debug, got %q", cfg.App.LogLevel)
		}
	})
}

func TestEnvOverrides_ServicesCreatedFromEnv(t *testing.T) {
	t.Run("sonarr", func(t *testing.T) {
		path := writeTempYAML(t, minimalYAML)
		t.Setenv("MEDIAMATE_SONARR_URL", "http://sonarr:8989")
		t.Setenv("MEDIAMATE_SONARR_API_KEY", "sonarr-key")
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Sonarr == nil {
			t.Fatal("expected sonarr to be created from env")
		}
		if cfg.Sonarr.URL != "http://sonarr:8989" {
			t.Errorf("expected http://sonarr:8989, got %q", cfg.Sonarr.URL)
		}
	})

	t.Run("telegram", func(t *testing.T) {
		path := writeTempYAML(t, minimalYAML)
		t.Setenv("MEDIAMATE_TELEGRAM_BOT_TOKEN", "123:TOKEN")
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Telegram == nil || cfg.Telegram.BotToken != "123:TOKEN" {
			t.Error("expected telegram created from env")
		}
	})

	t.Run("jellyfin", func(t *testing.T) {
		path := writeTempYAML(t, minimalYAML)
		t.Setenv("MEDIAMATE_JELLYFIN_URL", "http://jellyfin:8096")
		t.Setenv("MEDIAMATE_JELLYFIN_API_KEY", "jf-key")
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Jellyfin == nil {
			t.Fatal("expected jellyfin created from env")
		}
		if cfg.Jellyfin.URL != "http://jellyfin:8096" {
			t.Errorf("expected http://jellyfin:8096, got %q", cfg.Jellyfin.URL)
		}
	})

	t.Run("qbittorrent", func(t *testing.T) {
		path := writeTempYAML(t, minimalYAML)
		t.Setenv("MEDIAMATE_QBITTORRENT_URL", "http://qbit:8080")
		t.Setenv("MEDIAMATE_QBITTORRENT_USERNAME", "admin")
		t.Setenv("MEDIAMATE_QBITTORRENT_PASSWORD", "pass")
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.QBittorrent == nil {
			t.Fatal("expected qbittorrent created from env")
		}
		if cfg.QBittorrent.URL != "http://qbit:8080" {
			t.Errorf("expected http://qbit:8080, got %q", cfg.QBittorrent.URL)
		}
	})
}

func TestEnvOverrides_WebhookPort(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		path := writeTempYAML(t, minimalYAML)
		t.Setenv("MEDIAMATE_WEBHOOK_PORT", "not-a-number")
		_, err := Load(path)
		if err == nil {
			t.Fatal("expected validation error for invalid webhook port")
		}
		if !strings.Contains(err.Error(), "webhook.port must be between") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("valid", func(t *testing.T) {
		path := writeTempYAML(t, minimalYAML)
		t.Setenv("MEDIAMATE_WEBHOOK_PORT", "9090")
		t.Setenv("MEDIAMATE_WEBHOOK_SECRET", "test-secret")
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Webhook == nil || cfg.Webhook.Port != 9090 {
			t.Errorf("expected webhook port 9090, got %v", cfg.Webhook)
		}
	})
}

func TestApplyArrEnv(t *testing.T) {
	t.Run("nil_no_env", func(t *testing.T) {
		result := applyArrEnv(nil, "MEDIAMATE_TEST_URL", "MEDIAMATE_TEST_KEY")
		if result != nil {
			t.Error("expected nil when no env vars set")
		}
	})

	t.Run("nil_with_url_env", func(t *testing.T) {
		t.Setenv("MEDIAMATE_TEST2_URL", "http://test:1234")
		result := applyArrEnv(nil, "MEDIAMATE_TEST2_URL", "MEDIAMATE_TEST2_KEY")
		if result == nil {
			t.Fatal("expected non-nil ArrConfig")
		}
		if result.URL != "http://test:1234" {
			t.Errorf("expected http://test:1234, got %q", result.URL)
		}
	})

	t.Run("existing_overridden", func(t *testing.T) {
		t.Setenv("MEDIAMATE_TEST3_URL", "http://new:5678")
		t.Setenv("MEDIAMATE_TEST3_KEY", "new-key")
		existing := &ArrConfig{URL: "http://old:1234", APIKey: "old-key"}
		result := applyArrEnv(existing, "MEDIAMATE_TEST3_URL", "MEDIAMATE_TEST3_KEY")
		if result.URL != "http://new:5678" {
			t.Errorf("expected http://new:5678, got %q", result.URL)
		}
		if result.APIKey != "new-key" {
			t.Errorf("expected new-key, got %q", result.APIKey)
		}
	})

	t.Run("existing_partial_override", func(t *testing.T) {
		t.Setenv("MEDIAMATE_TEST4_KEY", "env-key")
		existing := &ArrConfig{URL: "http://old:1234", APIKey: "old-key"}
		result := applyArrEnv(existing, "MEDIAMATE_TEST4_URL", "MEDIAMATE_TEST4_KEY")
		if result.URL != "http://old:1234" {
			t.Errorf("expected URL preserved, got %q", result.URL)
		}
		if result.APIKey != "env-key" {
			t.Errorf("expected env-key, got %q", result.APIKey)
		}
	})
}

func TestValidateConfigPath(t *testing.T) {
	t.Parallel()

	t.Run("valid_file", func(t *testing.T) {
		t.Parallel()
		path := writeTempYAML(t, "test")
		if err := validateConfigPath(path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()
		err := validateConfigPath("/nonexistent/file.yaml")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "config file not found") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("is_directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		err := validateConfigPath(dir)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "directory") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

const minimalYAML = `
llm:
  provider: claude
  api_key: yaml-key
tmdb:
  api_key: tmdb-key
radarr:
  url: http://localhost:7878
  api_key: radarr-key
`

// writeTempYAML creates a temporary YAML file and returns its path.
func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp yaml: %v", err)
	}
	return path
}
