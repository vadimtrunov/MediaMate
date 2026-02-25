package stack

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTestCompose writes a docker-compose.yml to the given directory.
func writeTestCompose(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// writeTestEnv writes a .env file to the given directory.
func writeTestEnv(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// ---------------------------------------------------------------------------
// 1. TestParseComposeServices
// ---------------------------------------------------------------------------

func TestParseComposeServices_ExtractsKnown(t *testing.T) {
	dir := t.TempDir()
	path := writeTestCompose(t, dir, `version: "3.8"
services:
  radarr:
    image: lscr.io/linuxserver/radarr:latest
  sonarr:
    image: lscr.io/linuxserver/sonarr:latest
  qbittorrent:
    image: lscr.io/linuxserver/qbittorrent:latest
  mediamate:
    image: ghcr.io/vadimtrunov/mediamate:latest
networks:
  default:
`)

	components, err := parseComposeServices(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// mediamate should be excluded
	want := []string{"radarr", "sonarr", "qbittorrent"}
	if len(components) != len(want) {
		t.Fatalf("got %d components %v, want %d %v", len(components), components, len(want), want)
	}
	for i, c := range components {
		if c != want[i] {
			t.Errorf("component[%d] = %q, want %q", i, c, want[i])
		}
	}
}

func TestParseComposeServices_SkipsUnknown(t *testing.T) {
	dir := t.TempDir()
	path := writeTestCompose(t, dir, `services:
  radarr:
    image: foo
  my-custom-service:
    image: bar
  nginx:
    image: nginx
  prowlarr:
    image: baz
`)

	components, err := parseComposeServices(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"radarr", "prowlarr"}
	if len(components) != len(want) {
		t.Fatalf("got %v, want %v", components, want)
	}
	for i, c := range components {
		if c != want[i] {
			t.Errorf("component[%d] = %q, want %q", i, c, want[i])
		}
	}
}

func TestParseComposeServices_Errors(t *testing.T) {
	t.Run("error on no known services", func(t *testing.T) {
		dir := t.TempDir()
		writeTestCompose(t, dir, "services:\n  nginx:\n    image: nginx\n")

		_, err := parseComposeServices(filepath.Join(dir, "docker-compose.yml"))
		if err == nil {
			t.Fatal("expected error for compose with no known services")
		}
	})

	t.Run("error on missing file", func(t *testing.T) {
		_, err := parseComposeServices("/nonexistent/docker-compose.yml")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("empty services block", func(t *testing.T) {
		dir := t.TempDir()
		writeTestCompose(t, dir, "services:\nvolumes:\n  data:\n")

		_, err := parseComposeServices(filepath.Join(dir, "docker-compose.yml"))
		if err == nil {
			t.Fatal("expected error for empty services block")
		}
	})
}

func TestParseComposeServices_StopsAtTopLevelKey(t *testing.T) {
	dir := t.TempDir()
	// radarr is under services:, jellyfin appears after networks: — should not be parsed
	path := writeTestCompose(t, dir, `services:
  radarr:
    image: foo
networks:
  jellyfin:
    driver: bridge
`)

	components, err := parseComposeServices(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(components) != 1 || components[0] != "radarr" {
		t.Errorf("got %v, want [radarr]", components)
	}
}

func TestParseComposeServices_SkipsColumnZeroComments(t *testing.T) {
	dir := t.TempDir()
	path := writeTestCompose(t, dir, "services:\n  radarr:\n    image: foo\n# a comment at column 0\n  sonarr:\n    image: bar\n")

	components, err := parseComposeServices(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"radarr", "sonarr"}
	if len(components) != len(want) {
		t.Fatalf("got %v, want %v", components, want)
	}
	for i, c := range components {
		if c != want[i] {
			t.Errorf("component[%d] = %q, want %q", i, c, want[i])
		}
	}
}

// ---------------------------------------------------------------------------
// 2. TestParseEnvFile
// ---------------------------------------------------------------------------

func TestParseEnvFile_ParsesKeyValues(t *testing.T) {
	dir := t.TempDir()
	path := writeTestEnv(t, dir, "CONFIG_DIR=/my/config\nMOVIES_DIR=/my/movies\nDOWNLOADS_DIR=/my/downloads\n")

	vars, err := parseEnvFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vars["CONFIG_DIR"] != "/my/config" {
		t.Errorf("CONFIG_DIR = %q, want /my/config", vars["CONFIG_DIR"])
	}
	if vars["MOVIES_DIR"] != "/my/movies" {
		t.Errorf("MOVIES_DIR = %q, want /my/movies", vars["MOVIES_DIR"])
	}
	if vars["DOWNLOADS_DIR"] != "/my/downloads" {
		t.Errorf("DOWNLOADS_DIR = %q, want /my/downloads", vars["DOWNLOADS_DIR"])
	}
}

func TestParseEnvFile_EdgeCases(t *testing.T) {
	t.Run("skips comments and empty lines", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTestEnv(t, dir, "# This is a comment\nFOO=bar\n\n# Another comment\nBAZ=qux\n")
		vars, err := parseEnvFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(vars) != 2 {
			t.Fatalf("got %d vars, want 2", len(vars))
		}
		if vars["FOO"] != "bar" {
			t.Errorf("FOO = %q, want bar", vars["FOO"])
		}
	})

	t.Run("trims whitespace around key and value", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTestEnv(t, dir, "  KEY  =  value\n")
		vars, err := parseEnvFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if vars["KEY"] != "value" {
			t.Errorf("KEY = %q, want value", vars["KEY"])
		}
	})

	t.Run("skips lines without equals sign", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTestEnv(t, dir, "VALID=yes\nno-equals-sign\nALSO_VALID=true\n")
		vars, err := parseEnvFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(vars) != 2 {
			t.Fatalf("got %d vars, want 2", len(vars))
		}
	})

	t.Run("value with equals sign", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTestEnv(t, dir, "DATABASE_URL=postgres://user:pass@host/db?sslmode=require\n")
		vars, err := parseEnvFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "postgres://user:pass@host/db?sslmode=require"
		if vars["DATABASE_URL"] != want {
			t.Errorf("DATABASE_URL = %q, want %q", vars["DATABASE_URL"], want)
		}
	})

	t.Run("error on missing file", func(t *testing.T) {
		_, err := parseEnvFile("/nonexistent/.env")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})
}

// ---------------------------------------------------------------------------
// 3. TestApplyEnvDir
// ---------------------------------------------------------------------------

func TestApplyEnvDir(t *testing.T) {
	t.Run("overwrites when key exists", func(t *testing.T) {
		vars := map[string]string{"CONFIG_DIR": "/custom/config"}
		dst := "/default/config"
		applyEnvDir(vars, "CONFIG_DIR", &dst)
		if dst != "/custom/config" {
			t.Errorf("dst = %q, want /custom/config", dst)
		}
	})

	t.Run("no-op when key missing", func(t *testing.T) {
		vars := map[string]string{}
		dst := "/default/config"
		applyEnvDir(vars, "CONFIG_DIR", &dst)
		if dst != "/default/config" {
			t.Errorf("dst = %q, want /default/config (unchanged)", dst)
		}
	})

	t.Run("no-op when value empty", func(t *testing.T) {
		vars := map[string]string{"CONFIG_DIR": ""}
		dst := "/default/config"
		applyEnvDir(vars, "CONFIG_DIR", &dst)
		if dst != "/default/config" {
			t.Errorf("dst = %q, want /default/config (unchanged)", dst)
		}
	})
}

// ---------------------------------------------------------------------------
// 4. TestLoadConfigFromCompose
// ---------------------------------------------------------------------------

func TestLoadConfigFromCompose_HappyPath(t *testing.T) {
	dir := t.TempDir()
	compose := "services:\n  radarr:\n    image: r\n  qbittorrent:\n    image: q\n" +
		"  jellyfin:\n    image: j\n  prowlarr:\n    image: p\n"
	env := "CONFIG_DIR=/opt/config\nMOVIES_DIR=/opt/movies\n" +
		"DOWNLOADS_DIR=/opt/downloads\nTV_DIR=/opt/tv\nBOOKS_DIR=/opt/books\nMEDIA_DIR=/opt/media\n"
	writeTestCompose(t, dir, compose)
	writeTestEnv(t, dir, env)

	cfg, err := LoadConfigFromCompose(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantComponents := []string{"radarr", "qbittorrent", "jellyfin", "prowlarr", "mediamate"}
	if len(cfg.Components) != len(wantComponents) {
		t.Fatalf("Components = %v, want %v", cfg.Components, wantComponents)
	}
	for i, c := range cfg.Components {
		if c != wantComponents[i] {
			t.Errorf("Components[%d] = %q, want %q", i, c, wantComponents[i])
		}
	}
	if !cfg.HasComponent(ComponentMediaMate) {
		t.Error("HasComponent(mediamate) = false, want true")
	}
	if cfg.TorrentClient != ComponentQBittorrent {
		t.Errorf("TorrentClient = %q, want %q", cfg.TorrentClient, ComponentQBittorrent)
	}
	if cfg.MediaServer != ComponentJellyfin {
		t.Errorf("MediaServer = %q, want %q", cfg.MediaServer, ComponentJellyfin)
	}
	assertConfigPaths(t, cfg)
}

// assertConfigPaths checks that all directory paths in cfg match expected values.
func assertConfigPaths(t *testing.T, cfg Config) {
	t.Helper()
	wantPaths := map[string]string{
		"ConfigDir": "/opt/config", "MoviesDir": "/opt/movies",
		"DownloadsDir": "/opt/downloads", "TVDir": "/opt/tv",
		"BooksDir": "/opt/books", "MediaDir": "/opt/media",
	}
	gotPaths := map[string]string{
		"ConfigDir": cfg.ConfigDir, "MoviesDir": cfg.MoviesDir,
		"DownloadsDir": cfg.DownloadsDir, "TVDir": cfg.TVDir,
		"BooksDir": cfg.BooksDir, "MediaDir": cfg.MediaDir,
	}
	for name, want := range wantPaths {
		if gotPaths[name] != want {
			t.Errorf("%s = %q, want %q", name, gotPaths[name], want)
		}
	}
}

func TestLoadConfigFromCompose_MissingEnv(t *testing.T) {
	dir := t.TempDir()
	writeTestCompose(t, dir, `services:
  radarr:
    image: lscr.io/linuxserver/radarr:latest
  transmission:
    image: lscr.io/linuxserver/transmission:latest
`)

	cfg, err := LoadConfigFromCompose(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defaults := DefaultConfig()
	if cfg.ConfigDir != defaults.ConfigDir {
		t.Errorf("ConfigDir = %q, want default %q", cfg.ConfigDir, defaults.ConfigDir)
	}
	if cfg.MoviesDir != defaults.MoviesDir {
		t.Errorf("MoviesDir = %q, want default %q", cfg.MoviesDir, defaults.MoviesDir)
	}
	if cfg.TorrentClient != ComponentTransmission {
		t.Errorf("TorrentClient = %q, want %q", cfg.TorrentClient, ComponentTransmission)
	}
	if cfg.MediaServer != "" {
		t.Errorf("MediaServer = %q, want empty", cfg.MediaServer)
	}
}

func TestLoadConfigFromCompose_MissingCompose(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadConfigFromCompose(dir)
	if err == nil {
		t.Fatal("expected error for missing docker-compose.yml")
	}
}

func TestLoadConfigFromCompose_PartialEnv(t *testing.T) {
	dir := t.TempDir()
	writeTestCompose(t, dir, "services:\n  radarr:\n    image: foo\n  deluge:\n    image: bar\n  plex:\n    image: baz\n")
	writeTestEnv(t, dir, "CONFIG_DIR=/custom/config\n")

	cfg, err := LoadConfigFromCompose(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ConfigDir != "/custom/config" {
		t.Errorf("ConfigDir = %q, want /custom/config", cfg.ConfigDir)
	}

	defaults := DefaultConfig()
	if cfg.MoviesDir != defaults.MoviesDir {
		t.Errorf("MoviesDir = %q, want default %q", cfg.MoviesDir, defaults.MoviesDir)
	}
	if cfg.TorrentClient != ComponentDeluge {
		t.Errorf("TorrentClient = %q, want %q", cfg.TorrentClient, ComponentDeluge)
	}
	if cfg.MediaServer != ComponentPlex {
		t.Errorf("MediaServer = %q, want %q", cfg.MediaServer, ComponentPlex)
	}
}

func TestLoadConfigFromCompose_NoTorrentOrMedia(t *testing.T) {
	dir := t.TempDir()
	writeTestCompose(t, dir, "services:\n  radarr:\n    image: foo\n  prowlarr:\n    image: bar\n")

	cfg, err := LoadConfigFromCompose(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.TorrentClient != "" {
		t.Errorf("TorrentClient = %q, want empty", cfg.TorrentClient)
	}
	if cfg.MediaServer != "" {
		t.Errorf("MediaServer = %q, want empty", cfg.MediaServer)
	}
}

// TestHasComponent and TestDockerImage are in generator_test.go.
