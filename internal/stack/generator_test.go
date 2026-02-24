package stack

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// 1. TestDefaultConfig
// ---------------------------------------------------------------------------

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Expected default components.
	wantComponents := []string{
		ComponentRadarr,
		ComponentSonarr,
		ComponentProwlarr,
		ComponentQBittorrent,
		ComponentJellyfin,
		ComponentMediaMate,
	}

	if len(cfg.Components) != len(wantComponents) {
		t.Fatalf("expected %d components, got %d", len(wantComponents), len(cfg.Components))
	}
	for i, want := range wantComponents {
		if cfg.Components[i] != want {
			t.Errorf("component[%d]: expected %q, got %q", i, want, cfg.Components[i])
		}
	}

	// Directory paths.
	if cfg.MediaDir != "/srv/media" {
		t.Errorf("MediaDir: expected /srv/media, got %s", cfg.MediaDir)
	}
	if cfg.MoviesDir != "/srv/media/movies" {
		t.Errorf("MoviesDir: expected /srv/media/movies, got %s", cfg.MoviesDir)
	}
	if cfg.TVDir != "/srv/media/tv" {
		t.Errorf("TVDir: expected /srv/media/tv, got %s", cfg.TVDir)
	}
	if cfg.BooksDir != "/srv/media/books" {
		t.Errorf("BooksDir: expected /srv/media/books, got %s", cfg.BooksDir)
	}
	if cfg.DownloadsDir != "/srv/media/downloads" {
		t.Errorf("DownloadsDir: expected /srv/media/downloads, got %s", cfg.DownloadsDir)
	}
	if cfg.ConfigDir != "/srv/mediamate/config" {
		t.Errorf("ConfigDir: expected /srv/mediamate/config, got %s", cfg.ConfigDir)
	}
	if cfg.OutputDir != "." {
		t.Errorf("OutputDir: expected '.', got %s", cfg.OutputDir)
	}

	// Client/server selections.
	if cfg.TorrentClient != ComponentQBittorrent {
		t.Errorf("TorrentClient: expected %s, got %s", ComponentQBittorrent, cfg.TorrentClient)
	}
	if cfg.MediaServer != ComponentJellyfin {
		t.Errorf("MediaServer: expected %s, got %s", ComponentJellyfin, cfg.MediaServer)
	}
}

// ---------------------------------------------------------------------------
// 2. TestHasComponent
// ---------------------------------------------------------------------------

func TestHasComponent(t *testing.T) {
	cfg := Config{
		Components: []string{ComponentRadarr, ComponentSonarr, ComponentQBittorrent},
	}

	tests := []struct {
		name      string
		component string
		want      bool
	}{
		{"present radarr", ComponentRadarr, true},
		{"present sonarr", ComponentSonarr, true},
		{"present qbittorrent", ComponentQBittorrent, true},
		{"absent jellyfin", ComponentJellyfin, false},
		{"absent plex", ComponentPlex, false},
		{"absent empty string", "", false},
		{"absent unknown", "unknown-component", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := cfg.HasComponent(tc.component)
			if got != tc.want {
				t.Errorf("HasComponent(%q) = %v, want %v", tc.component, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 3. TestDockerImage
// ---------------------------------------------------------------------------

func TestDockerImage(t *testing.T) {
	knownComponents := []string{
		ComponentRadarr,
		ComponentSonarr,
		ComponentReadarr,
		ComponentProwlarr,
		ComponentQBittorrent,
		ComponentTransmission,
		ComponentDeluge,
		ComponentJellyfin,
		ComponentPlex,
		ComponentGluetun,
		ComponentMediaMate,
	}

	for _, comp := range knownComponents {
		t.Run(comp, func(t *testing.T) {
			img := DockerImage(comp)
			if img == "" {
				t.Errorf("DockerImage(%q) returned empty string", comp)
			}
		})
	}

	t.Run("unknown component", func(t *testing.T) {
		img := DockerImage("nonexistent")
		if img != "" {
			t.Errorf("DockerImage(\"nonexistent\") = %q, want empty string", img)
		}
	})
}

// ---------------------------------------------------------------------------
// 4. TestDefaultCategories
// ---------------------------------------------------------------------------

func TestDefaultCategories(t *testing.T) {
	cats := DefaultCategories()

	if len(cats) == 0 {
		t.Fatal("DefaultCategories returned empty slice")
	}

	// Every category must have at least one option.
	for _, cat := range cats {
		if len(cat.Options) == 0 {
			t.Errorf("category %q has no options", cat.Name)
		}
	}

	// Verify specific well-known categories and their defaults.
	expected := map[string]string{
		"Movies":    ComponentRadarr,
		"TV Shows":  ComponentSonarr,
		"Indexers":  ComponentProwlarr,
		"Torrents":  ComponentQBittorrent,
		"Streaming": ComponentJellyfin,
		"Books":     "",
		"VPN":       "",
	}

	for _, cat := range cats {
		want, ok := expected[cat.Name]
		if !ok {
			continue
		}
		if cat.Default != want {
			t.Errorf("category %q: default = %q, want %q", cat.Name, cat.Default, want)
		}
	}

	// Verify Torrents category is required.
	for _, cat := range cats {
		if cat.Name == "Torrents" {
			if !cat.Required {
				t.Error("Torrents category should be required")
			}
		}
	}
}

// ---------------------------------------------------------------------------
// 5. TestGeneratePassword
// ---------------------------------------------------------------------------

func TestGeneratePassword(t *testing.T) {
	t.Run("correct length", func(t *testing.T) {
		p, err := GeneratePassword(16)
		if err != nil {
			t.Fatalf("GeneratePassword(16) error: %v", err)
		}
		// 16 bytes -> 32 hex characters.
		if len(p) != 32 {
			t.Errorf("expected length 32, got %d", len(p))
		}
	})

	t.Run("different lengths", func(t *testing.T) {
		for _, n := range []int{1, 4, 8, 32} {
			p, err := GeneratePassword(n)
			if err != nil {
				t.Fatalf("GeneratePassword(%d) error: %v", n, err)
			}
			if len(p) != 2*n {
				t.Errorf("GeneratePassword(%d): expected length %d, got %d", n, 2*n, len(p))
			}
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		seen := make(map[string]bool)
		for i := 0; i < 50; i++ {
			p, err := GeneratePassword(16)
			if err != nil {
				t.Fatalf("GeneratePassword(16) error on iteration %d: %v", i, err)
			}
			if seen[p] {
				t.Fatalf("duplicate password generated: %s", p)
			}
			seen[p] = true
		}
	})
}

// ---------------------------------------------------------------------------
// 6. TestGenerateSecrets
// ---------------------------------------------------------------------------

func TestGenerateSecrets(t *testing.T) {
	t.Run("qbittorrent present", func(t *testing.T) {
		cfg := &Config{
			Components: []string{ComponentQBittorrent, ComponentMediaMate},
		}
		secrets, err := GenerateSecrets(cfg)
		if err != nil {
			t.Fatalf("GenerateSecrets error: %v", err)
		}
		if _, ok := secrets["QBITTORRENT_PASSWORD"]; !ok {
			t.Error("expected QBITTORRENT_PASSWORD in secrets")
		}
		if len(secrets["QBITTORRENT_PASSWORD"]) != 32 {
			t.Errorf("QBITTORRENT_PASSWORD length: expected 32, got %d", len(secrets["QBITTORRENT_PASSWORD"]))
		}
	})

	t.Run("transmission present", func(t *testing.T) {
		cfg := &Config{
			Components: []string{ComponentTransmission},
		}
		secrets, err := GenerateSecrets(cfg)
		if err != nil {
			t.Fatalf("GenerateSecrets error: %v", err)
		}
		if _, ok := secrets["TRANSMISSION_PASSWORD"]; !ok {
			t.Error("expected TRANSMISSION_PASSWORD in secrets")
		}
	})

	t.Run("deluge present", func(t *testing.T) {
		cfg := &Config{
			Components: []string{ComponentDeluge},
		}
		secrets, err := GenerateSecrets(cfg)
		if err != nil {
			t.Fatalf("GenerateSecrets error: %v", err)
		}
		if _, ok := secrets["DELUGE_PASSWORD"]; !ok {
			t.Error("expected DELUGE_PASSWORD in secrets")
		}
	})

	t.Run("no torrent client", func(t *testing.T) {
		cfg := &Config{
			Components: []string{ComponentRadarr, ComponentJellyfin},
		}
		secrets, err := GenerateSecrets(cfg)
		if err != nil {
			t.Fatalf("GenerateSecrets error: %v", err)
		}
		if len(secrets) != 0 {
			t.Errorf("expected 0 secrets for no-torrent config, got %d", len(secrets))
		}
	})
}

// ---------------------------------------------------------------------------
// helpers for generator tests
// ---------------------------------------------------------------------------

func newTestGenerator(t *testing.T) *Generator {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewGenerator(logger)
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

func assertContains(t *testing.T, content, substr, context string) {
	t.Helper()
	if !strings.Contains(content, substr) {
		t.Errorf("%s: expected to contain %q", context, substr)
	}
}

func assertNotContains(t *testing.T, content, substr, context string) {
	t.Helper()
	if strings.Contains(content, substr) {
		t.Errorf("%s: expected NOT to contain %q", context, substr)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("expected file to exist: %s", path)
	}
}

// ---------------------------------------------------------------------------
// 7. TestGenerateCompose
// ---------------------------------------------------------------------------

func TestGenerateCompose(t *testing.T) {
	gen := newTestGenerator(t)
	cfg := DefaultConfig()
	cfg.OutputDir = t.TempDir()

	result, err := gen.Generate(&cfg, false)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	// Verify all files exist.
	assertFileExists(t, result.ComposePath)
	assertFileExists(t, result.EnvPath)
	assertFileExists(t, result.ConfigPath)

	// --- docker-compose.yml ---
	compose := readFile(t, result.ComposePath)

	// Must contain default components.
	for _, want := range []string{"radarr", "sonarr", "qbittorrent", "jellyfin", "mediamate"} {
		assertContains(t, compose, want, "docker-compose.yml")
	}

	// Must NOT contain components not in defaults.
	for _, absent := range []string{
		"readarr",
		"container_name: plex",
		"container_name: gluetun",
	} {
		assertNotContains(t, compose, absent, "docker-compose.yml")
	}

	// --- .env ---
	envContent := readFile(t, result.EnvPath)
	for _, want := range []string{
		"CONFIG_DIR=",
		"MEDIA_DIR=",
		"MOVIES_DIR=",
		"TV_DIR=",
		"DOWNLOADS_DIR=",
		"MEDIAMATE_LLM_API_KEY=",
		"MEDIAMATE_TMDB_API_KEY=",
		"MEDIAMATE_RADARR_API_KEY=",
		"MEDIAMATE_SONARR_API_KEY=",
		"MEDIAMATE_JELLYFIN_API_KEY=",
		"QBITTORRENT_PASSWORD=",
	} {
		assertContains(t, envContent, want, ".env")
	}

	// .env should NOT contain readarr key (not in defaults).
	assertNotContains(t, envContent, "MEDIAMATE_READARR_API_KEY", ".env")

	// --- mediamate.yaml ---
	mmConfig := readFile(t, result.ConfigPath)
	assertContains(t, mmConfig, "radarr:", "mediamate.yaml")
	assertContains(t, mmConfig, "jellyfin:", "mediamate.yaml")
	assertContains(t, mmConfig, "sonarr:", "mediamate.yaml")
	assertContains(t, mmConfig, "qbittorrent:", "mediamate.yaml")
}

// ---------------------------------------------------------------------------
// 8. TestGenerateMinimalConfig
// ---------------------------------------------------------------------------

func TestGenerateMinimalConfig(t *testing.T) {
	gen := newTestGenerator(t)

	cfg := Config{
		Components:    []string{ComponentQBittorrent, ComponentMediaMate},
		MediaDir:      "/tmp/media",
		MoviesDir:     "/tmp/media/movies",
		TVDir:         "/tmp/media/tv",
		BooksDir:      "/tmp/media/books",
		DownloadsDir:  "/tmp/media/downloads",
		ConfigDir:     "/tmp/config",
		OutputDir:     t.TempDir(),
		TorrentClient: ComponentQBittorrent,
		MediaServer:   "",
	}

	result, err := gen.Generate(&cfg, false)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	compose := readFile(t, result.ComposePath)

	// Must contain qbittorrent and mediamate.
	assertContains(t, compose, "qbittorrent", "minimal compose")
	assertContains(t, compose, "mediamate", "minimal compose")

	// Must NOT contain components that were not selected.
	assertNotContains(t, compose, "radarr", "minimal compose")
	assertNotContains(t, compose, "sonarr", "minimal compose")
	assertNotContains(t, compose, "jellyfin", "minimal compose")
	assertNotContains(t, compose, "prowlarr", "minimal compose")
	assertNotContains(t, compose, "readarr", "minimal compose")
	assertNotContains(t, compose, "plex", "minimal compose")
	assertNotContains(t, compose, "gluetun", "minimal compose")
}

// ---------------------------------------------------------------------------
// 9. TestGenerateOverwriteProtection
// ---------------------------------------------------------------------------

func TestGenerateOverwriteProtection(t *testing.T) {
	gen := newTestGenerator(t)
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.OutputDir = dir

	// Create an existing file that Generate would try to write.
	existingPath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(existingPath, []byte("existing content"), 0o600); err != nil {
		t.Fatalf("creating existing file: %v", err)
	}

	_, err := gen.Generate(&cfg, false)
	if err == nil {
		t.Fatal("expected error when file already exists and overwrite=false")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists', got: %v", err)
	}

	// Verify the existing file was not modified.
	content := readFile(t, existingPath)
	if content != "existing content" {
		t.Error("existing file was modified despite overwrite=false")
	}
}

// ---------------------------------------------------------------------------
// 10. TestGenerateOverwriteAllowed
// ---------------------------------------------------------------------------

func TestGenerateOverwriteAllowed(t *testing.T) {
	gen := newTestGenerator(t)
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.OutputDir = dir

	// Create existing files that Generate would overwrite.
	for _, name := range []string{"docker-compose.yml", ".env", "mediamate.yaml"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("old content"), 0o600); err != nil {
			t.Fatalf("creating %s: %v", name, err)
		}
	}

	result, err := gen.Generate(&cfg, true)
	if err != nil {
		t.Fatalf("Generate with overwrite=true error: %v", err)
	}

	// Verify files were actually overwritten (content is different from "old content").
	compose := readFile(t, result.ComposePath)
	if compose == "old content" {
		t.Error("docker-compose.yml was not overwritten")
	}
	assertContains(t, compose, "services:", "overwritten docker-compose.yml")

	envContent := readFile(t, result.EnvPath)
	if envContent == "old content" {
		t.Error(".env was not overwritten")
	}

	mmConfig := readFile(t, result.ConfigPath)
	if mmConfig == "old content" {
		t.Error("mediamate.yaml was not overwritten")
	}
}

// ---------------------------------------------------------------------------
// 11. TestRenderMediaMateConfigDefault
// ---------------------------------------------------------------------------

func TestRenderMediaMateConfigDefault(t *testing.T) {
	cfg := DefaultConfig()
	rendered := RenderMediaMateConfig(&cfg)

	// Radarr section with Docker hostname.
	assertContains(t, rendered, "radarr:", "mediamate config")
	assertContains(t, rendered, "http://radarr:7878", "mediamate config radarr url")

	// Sonarr section with Docker hostname.
	assertContains(t, rendered, "sonarr:", "mediamate config")
	assertContains(t, rendered, "http://sonarr:8989", "mediamate config sonarr url")

	// qBittorrent section with Docker hostname.
	assertContains(t, rendered, "qbittorrent:", "mediamate config")
	assertContains(t, rendered, "http://qbittorrent:8080", "mediamate config qbit url")

	// Jellyfin section with Docker hostname.
	assertContains(t, rendered, "jellyfin:", "mediamate config")
	assertContains(t, rendered, "http://jellyfin:8096", "mediamate config jellyfin url")

	// Should NOT contain readarr (not in defaults).
	assertNotContains(t, rendered, "http://readarr:8787", "mediamate config no readarr")

	// Common sections.
	assertContains(t, rendered, "llm:", "mediamate config llm")
	assertContains(t, rendered, "tmdb:", "mediamate config tmdb")
	assertContains(t, rendered, "telegram:", "mediamate config telegram")
	assertContains(t, rendered, "app:", "mediamate config app")
}

// ---------------------------------------------------------------------------
// 12. TestRenderMediaMateConfigVariants
// ---------------------------------------------------------------------------

func TestRenderMediaMateConfigVariants(t *testing.T) {
	t.Run("with readarr", func(t *testing.T) {
		cfg := Config{
			Components:    []string{ComponentReadarr, ComponentQBittorrent, ComponentMediaMate},
			TorrentClient: ComponentQBittorrent,
			MediaServer:   "",
		}
		rendered := RenderMediaMateConfig(&cfg)

		assertContains(t, rendered, "readarr:", "mediamate config with readarr")
		assertContains(t, rendered, "http://readarr:8787", "mediamate config readarr url")
		assertNotContains(t, rendered, "http://radarr:7878", "mediamate config no radarr")
		assertNotContains(t, rendered, "http://sonarr:8989", "mediamate config no sonarr")
	})

	t.Run("qbittorrent behind gluetun", func(t *testing.T) {
		cfg := Config{
			Components:    []string{ComponentQBittorrent, ComponentGluetun, ComponentMediaMate},
			TorrentClient: ComponentQBittorrent,
			MediaServer:   "",
		}
		rendered := RenderMediaMateConfig(&cfg)

		// When gluetun is present, qbittorrent URL should use gluetun as host.
		assertContains(t, rendered, "http://gluetun:8080", "mediamate config qbit behind gluetun")
		assertNotContains(t, rendered, "http://qbittorrent:8080", "mediamate config no direct qbit url")
	})

	t.Run("transmission client", func(t *testing.T) {
		cfg := Config{
			Components:    []string{ComponentTransmission, ComponentMediaMate},
			TorrentClient: ComponentTransmission,
			MediaServer:   "",
		}
		rendered := RenderMediaMateConfig(&cfg)

		// Transmission section is commented out but should reference the host.
		assertContains(t, rendered, "transmission", "mediamate config transmission")
	})

	t.Run("plex media server", func(t *testing.T) {
		cfg := Config{
			Components:    []string{ComponentPlex, ComponentQBittorrent, ComponentMediaMate},
			TorrentClient: ComponentQBittorrent,
			MediaServer:   ComponentPlex,
		}
		rendered := RenderMediaMateConfig(&cfg)

		// Plex section is commented out but should be present.
		assertContains(t, rendered, "plex", "mediamate config plex")
		assertNotContains(t, rendered, "jellyfin:", "mediamate config no jellyfin")
	})
}
