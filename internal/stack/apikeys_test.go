package stack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile is a test helper that writes content to a file inside a temp directory.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// ---------------------------------------------------------------------------
// 1. TestReadAPIKeyFromXML
// ---------------------------------------------------------------------------

func TestReadAPIKeyFromXML_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.xml", `<Config>
  <ApiKey>abc123def456</ApiKey>
  <Port>7878</Port>
</Config>`)

	key, err := readAPIKeyFromXML(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "abc123def456" {
		t.Errorf("key = %q, want abc123def456", key)
	}
}

func TestReadAPIKeyFromXML_Errors(t *testing.T) {
	t.Run("empty ApiKey", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "config.xml", `<Config><ApiKey></ApiKey></Config>`)

		_, err := readAPIKeyFromXML(path)
		if err == nil {
			t.Fatal("expected error for empty ApiKey")
		}
		if !strings.Contains(err.Error(), "empty ApiKey") {
			t.Errorf("error = %q, want to contain 'empty ApiKey'", err.Error())
		}
	})

	t.Run("missing ApiKey element", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "config.xml", `<Config><Port>7878</Port></Config>`)

		_, err := readAPIKeyFromXML(path)
		if err == nil {
			t.Fatal("expected error for missing ApiKey element")
		}
	})

	t.Run("invalid XML", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "config.xml", "not xml at all")

		_, err := readAPIKeyFromXML(path)
		if err == nil {
			t.Fatal("expected error for invalid XML")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := readAPIKeyFromXML("/nonexistent/config.xml")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})
}

// ---------------------------------------------------------------------------
// 2. TestReadAPIKeys
// ---------------------------------------------------------------------------

// writeArrConfig creates a component config.xml in the given base directory.
func writeArrConfig(t *testing.T, baseDir, component, apiKey string) {
	t.Helper()
	compDir := filepath.Join(baseDir, component)
	if err := os.MkdirAll(compDir, 0o755); err != nil {
		t.Fatal(err)
	}
	xml := `<Config><ApiKey>` + apiKey + `</ApiKey></Config>`
	writeFile(t, compDir, "config.xml", xml)
}

func TestReadAPIKeys_MultipleComponents(t *testing.T) {
	dir := t.TempDir()
	writeArrConfig(t, dir, "radarr", "radarr-key-123")
	writeArrConfig(t, dir, "prowlarr", "prowlarr-key-456")

	components := []string{ComponentRadarr, ComponentProwlarr, ComponentQBittorrent, ComponentJellyfin}
	keys := ReadAPIKeys(dir, components, discardLogger())

	if keys[ComponentRadarr] != "radarr-key-123" {
		t.Errorf("radarr key = %q, want radarr-key-123", keys[ComponentRadarr])
	}
	if keys[ComponentProwlarr] != "prowlarr-key-456" {
		t.Errorf("prowlarr key = %q, want prowlarr-key-456", keys[ComponentProwlarr])
	}

	// Non-arr components should not have keys
	if _, ok := keys[ComponentQBittorrent]; ok {
		t.Error("qbittorrent should not have an API key")
	}
	if _, ok := keys[ComponentJellyfin]; ok {
		t.Error("jellyfin should not have an API key")
	}
}

func TestReadAPIKeys_EdgeCases(t *testing.T) {
	t.Run("skips missing config files", func(t *testing.T) {
		dir := t.TempDir()
		writeArrConfig(t, dir, "radarr", "radarr-key")

		components := []string{ComponentRadarr, ComponentProwlarr}
		keys := ReadAPIKeys(dir, components, discardLogger())

		if keys[ComponentRadarr] != "radarr-key" {
			t.Errorf("radarr key = %q, want radarr-key", keys[ComponentRadarr])
		}
		if _, ok := keys[ComponentProwlarr]; ok {
			t.Error("prowlarr key should not be present (config missing)")
		}
	})

	t.Run("nil logger defaults to slog.Default", func(t *testing.T) {
		dir := t.TempDir()
		keys := ReadAPIKeys(dir, []string{ComponentRadarr}, nil)
		if len(keys) != 0 {
			t.Errorf("expected no keys, got %v", keys)
		}
	})

	t.Run("empty components list", func(t *testing.T) {
		keys := ReadAPIKeys("/any/dir", nil, discardLogger())
		if len(keys) != 0 {
			t.Errorf("expected no keys, got %v", keys)
		}
	})
}

// ---------------------------------------------------------------------------
// 3. TestUpdateEnvFile
// ---------------------------------------------------------------------------

func TestUpdateEnvFile_ReplacesPlaceholders(t *testing.T) {
	dir := t.TempDir()
	envPath := writeFile(t, dir, ".env", `# MediaMate environment
MEDIAMATE_RADARR_API_KEY=your-radarr-api-key-here
MEDIAMATE_PROWLARR_API_KEY=your-prowlarr-api-key-here
SOME_OTHER_VAR=keep-this
`)

	keys := ServiceAPIKeys{
		ComponentRadarr:   "real-radarr-key",
		ComponentProwlarr: "real-prowlarr-key",
	}

	if err := UpdateEnvFile(envPath, keys); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "MEDIAMATE_RADARR_API_KEY=real-radarr-key") {
		t.Errorf("radarr key not replaced in:\n%s", content)
	}
	if !strings.Contains(content, "MEDIAMATE_PROWLARR_API_KEY=real-prowlarr-key") {
		t.Errorf("prowlarr key not replaced in:\n%s", content)
	}
	if !strings.Contains(content, "SOME_OTHER_VAR=keep-this") {
		t.Errorf("unrelated var was modified in:\n%s", content)
	}
}

func TestUpdateEnvFile_Permissions(t *testing.T) {
	dir := t.TempDir()
	envPath := writeFile(t, dir, ".env", "MEDIAMATE_RADARR_API_KEY=placeholder\n")

	keys := ServiceAPIKeys{ComponentRadarr: "secret"}
	if err := UpdateEnvFile(envPath, keys); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func TestUpdateEnvFile_Errors(t *testing.T) {
	t.Run("error on missing file", func(t *testing.T) {
		err := UpdateEnvFile("/nonexistent/.env", ServiceAPIKeys{})
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("no matching keys leaves file unchanged", func(t *testing.T) {
		dir := t.TempDir()
		original := "FOO=bar\nBAZ=qux\n"
		envPath := writeFile(t, dir, ".env", original)

		keys := ServiceAPIKeys{ComponentRadarr: "some-key"}
		if err := UpdateEnvFile(envPath, keys); err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(envPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != original {
			t.Errorf("file was modified:\ngot:  %q\nwant: %q", string(data), original)
		}
	})
}

// ---------------------------------------------------------------------------
// 4. TestUpdateMediaMateConfig
// ---------------------------------------------------------------------------

func TestUpdateMediaMateConfig_ReplacesPlaceholders(t *testing.T) {
	dir := t.TempDir()
	configPath := writeFile(t, dir, "mediamate.yaml", `radarr:
  api_key: "${MEDIAMATE_RADARR_API_KEY}"
  url: "http://localhost:7878"
prowlarr:
  api_key: "${MEDIAMATE_PROWLARR_API_KEY}"
  url: "http://localhost:9696"
`)

	keys := ServiceAPIKeys{
		ComponentRadarr:   "radarr-secret",
		ComponentProwlarr: "prowlarr-secret",
	}

	if err := UpdateMediaMateConfig(configPath, keys); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if strings.Contains(content, "${MEDIAMATE_RADARR_API_KEY}") {
		t.Errorf("radarr placeholder not replaced in:\n%s", content)
	}
	if strings.Contains(content, "${MEDIAMATE_PROWLARR_API_KEY}") {
		t.Errorf("prowlarr placeholder not replaced in:\n%s", content)
	}
	if !strings.Contains(content, "radarr-secret") {
		t.Errorf("radarr key not found in:\n%s", content)
	}
	if !strings.Contains(content, "prowlarr-secret") {
		t.Errorf("prowlarr key not found in:\n%s", content)
	}
}

func TestUpdateMediaMateConfig_Permissions(t *testing.T) {
	dir := t.TempDir()
	configPath := writeFile(t, dir, "mediamate.yaml", "api_key: ${MEDIAMATE_RADARR_API_KEY}\n")

	keys := ServiceAPIKeys{ComponentRadarr: "secret"}
	if err := UpdateMediaMateConfig(configPath, keys); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func TestUpdateMediaMateConfig_Errors(t *testing.T) {
	t.Run("error on missing file", func(t *testing.T) {
		err := UpdateMediaMateConfig("/nonexistent/mediamate.yaml", ServiceAPIKeys{})
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("unknown component keys are ignored", func(t *testing.T) {
		dir := t.TempDir()
		original := "some: config\n"
		configPath := writeFile(t, dir, "mediamate.yaml", original)

		// qbittorrent has no envKeyMapping entry
		keys := ServiceAPIKeys{ComponentQBittorrent: "some-key"}
		if err := UpdateMediaMateConfig(configPath, keys); err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != original {
			t.Errorf("file was modified:\ngot:  %q\nwant: %q", string(data), original)
		}
	})
}
