package stack

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// arrConfig represents the minimal XML structure of an *arr service config.
type arrConfig struct {
	XMLName xml.Name `xml:"Config"`
	APIKey  string   `xml:"ApiKey"` //nolint:gosec // XML field name matches *arr config format
}

// ServiceAPIKeys maps component names to their extracted API keys.
type ServiceAPIKeys map[string]string

// arrComponents is the set of components that have API keys in config.xml.
var arrComponents = map[string]bool{
	ComponentRadarr:   true,
	ComponentSonarr:   true,
	ComponentReadarr:  true,
	ComponentProwlarr: true,
}

// envKeyMapping maps component names to their .env variable names.
var envKeyMapping = map[string]string{
	ComponentRadarr:   "MEDIAMATE_RADARR_API_KEY",
	ComponentSonarr:   "MEDIAMATE_SONARR_API_KEY",
	ComponentReadarr:  "MEDIAMATE_READARR_API_KEY",
	ComponentProwlarr: "MEDIAMATE_PROWLARR_API_KEY",
}

// ReadAPIKeys reads API keys from the config.xml files of *arr services.
// configDir is the base config directory (e.g., /srv/mediamate/config).
// components is the list of components to read keys for.
// Returns a map of component name -> API key.
// Non-critical errors (missing files) are logged but don't fail the operation.
func ReadAPIKeys(configDir string, components []string, logger *slog.Logger) ServiceAPIKeys {
	if logger == nil {
		logger = slog.Default()
	}

	keys := make(ServiceAPIKeys)

	for _, comp := range components {
		if !arrComponents[comp] {
			continue
		}

		path := filepath.Join(configDir, comp, "config.xml")
		key, err := readAPIKeyFromXML(path)
		if err != nil {
			logger.Warn("could not read API key",
				slog.String("component", comp),
				slog.String("path", path),
				slog.String("error", err.Error()),
			)
			continue
		}

		keys[comp] = key
		logger.Info("read API key",
			slog.String("component", comp),
			slog.String("path", path),
		)
	}

	return keys
}

// readAPIKeyFromXML reads the <ApiKey> element from a config.xml file.
func readAPIKeyFromXML(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}

	var cfg arrConfig
	if err := xml.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("parse %s: %w", path, err)
	}

	if cfg.APIKey == "" {
		return "", fmt.Errorf("empty ApiKey in %s", path)
	}

	return cfg.APIKey, nil
}

// UpdateEnvFile reads an existing .env file and replaces API key placeholders
// with actual values from the extracted keys.
// It replaces lines like "MEDIAMATE_RADARR_API_KEY=your-radarr-api-key-here"
// with "MEDIAMATE_RADARR_API_KEY=<actual-key>".
func UpdateEnvFile(envPath string, keys ServiceAPIKeys) error {
	data, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", envPath, err)
	}

	// Build a reverse lookup: env var name -> actual key value.
	envVarToKey := make(map[string]string)
	for comp, key := range keys {
		if envVar, ok := envKeyMapping[comp]; ok {
			envVarToKey[envVar] = key
		}
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		for envVar, key := range envVarToKey {
			prefix := envVar + "="
			if strings.HasPrefix(line, prefix) {
				lines[i] = prefix + key
				break
			}
		}
	}

	output := strings.Join(lines, "\n")
	if err := os.WriteFile(envPath, []byte(output), permSecret); err != nil {
		return fmt.Errorf("write %s: %w", envPath, err)
	}

	return nil
}

// UpdateMediaMateConfig reads an existing mediamate.yaml and replaces
// API key placeholders with actual values.
// It replaces "${MEDIAMATE_RADARR_API_KEY}" with the actual key value.
func UpdateMediaMateConfig(configPath string, keys ServiceAPIKeys) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", configPath, err)
	}

	content := string(data)
	for comp, key := range keys {
		envVar, ok := envKeyMapping[comp]
		if !ok {
			continue
		}
		placeholder := "${" + envVar + "}"
		content = strings.ReplaceAll(content, placeholder, key)
	}

	if err := os.WriteFile(configPath, []byte(content), permSecret); err != nil {
		return fmt.Errorf("write %s: %w", configPath, err)
	}

	return nil
}
