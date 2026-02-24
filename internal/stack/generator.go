package stack

import (
	"bytes"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"text/template"
)

// File permission constants.
const (
	permFile      os.FileMode = 0o644 // standard file permissions
	permSecret    os.FileMode = 0o600 // restricted permissions for files containing secrets
	permDirectory os.FileMode = 0o755 // directory permissions
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// templateData is the data passed to Go templates for rendering Docker Compose
// and .env files. It wraps Config with convenience accessors.
type templateData struct {
	*Config
	Images  imageMap
	Secrets map[string]string
}

// imageMap provides named accessors for Docker images used in templates.
type imageMap struct {
	Radarr       string
	Sonarr       string
	Readarr      string
	Prowlarr     string
	QBittorrent  string
	Transmission string
	Deluge       string
	Jellyfin     string
	Plex         string
	Gluetun      string
	MediaMate    string
}

func newImageMap() imageMap {
	return imageMap{
		Radarr:       DockerImage(ComponentRadarr),
		Sonarr:       DockerImage(ComponentSonarr),
		Readarr:      DockerImage(ComponentReadarr),
		Prowlarr:     DockerImage(ComponentProwlarr),
		QBittorrent:  DockerImage(ComponentQBittorrent),
		Transmission: DockerImage(ComponentTransmission),
		Deluge:       DockerImage(ComponentDeluge),
		Jellyfin:     DockerImage(ComponentJellyfin),
		Plex:         DockerImage(ComponentPlex),
		Gluetun:      DockerImage(ComponentGluetun),
		MediaMate:    DockerImage(ComponentMediaMate),
	}
}

// HasRadarr through HasGluetun are convenience methods for templates.
func (d templateData) HasRadarr() bool       { return d.HasComponent(ComponentRadarr) }
func (d templateData) HasSonarr() bool       { return d.HasComponent(ComponentSonarr) }
func (d templateData) HasReadarr() bool      { return d.HasComponent(ComponentReadarr) }
func (d templateData) HasProwlarr() bool     { return d.HasComponent(ComponentProwlarr) }
func (d templateData) HasQBittorrent() bool  { return d.HasComponent(ComponentQBittorrent) }
func (d templateData) HasTransmission() bool { return d.HasComponent(ComponentTransmission) }
func (d templateData) HasDeluge() bool       { return d.HasComponent(ComponentDeluge) }
func (d templateData) HasJellyfin() bool     { return d.HasComponent(ComponentJellyfin) }
func (d templateData) HasPlex() bool         { return d.HasComponent(ComponentPlex) }
func (d templateData) HasGluetun() bool      { return d.HasComponent(ComponentGluetun) }

// Generator produces Docker Compose, .env, and mediamate.yaml files from a
// Config.
type Generator struct {
	logger *slog.Logger
}

// NewGenerator creates a Generator instance.
func NewGenerator(logger *slog.Logger) *Generator {
	if logger == nil {
		logger = slog.Default()
	}
	return &Generator{logger: logger}
}

// GenerateResult contains the paths of all generated files.
type GenerateResult struct {
	ComposePath string
	EnvPath     string
	ConfigPath  string
}

// Generate produces all stack files in the configured output directory.
// It returns an error if any file already exists and overwrite is false.
func (g *Generator) Generate(cfg *Config, overwrite bool) (*GenerateResult, error) {
	secrets, err := GenerateSecrets(cfg)
	if err != nil {
		return nil, fmt.Errorf("generate secrets: %w", err)
	}

	data := templateData{
		Config:  cfg,
		Images:  newImageMap(),
		Secrets: secrets,
	}

	composePath := filepath.Join(cfg.OutputDir, "docker-compose.yml")
	envPath := filepath.Join(cfg.OutputDir, ".env")
	configPath := filepath.Join(cfg.OutputDir, "mediamate.yaml")

	paths := []string{composePath, envPath, configPath}
	if !overwrite {
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return nil, fmt.Errorf("file already exists: %s (use --overwrite to replace)", p)
			}
		}
	}

	compose, err := g.renderTemplate("templates/docker-compose.yml.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("render docker-compose.yml: %w", err)
	}

	env, err := g.renderTemplate("templates/env.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("render .env: %w", err)
	}

	mmConfig := RenderMediaMateConfig(cfg)

	if err := g.writeFile(composePath, compose, permFile); err != nil {
		return nil, err
	}
	g.logger.Info("generated docker-compose.yml", slog.String("path", composePath))

	if err := g.writeFile(envPath, env, permSecret); err != nil {
		return nil, err
	}
	g.logger.Info("generated .env", slog.String("path", envPath))

	if err := g.writeFile(configPath, []byte(mmConfig), permFile); err != nil {
		return nil, err
	}
	g.logger.Info("generated mediamate.yaml", slog.String("path", configPath))

	return &GenerateResult{
		ComposePath: composePath,
		EnvPath:     envPath,
		ConfigPath:  configPath,
	}, nil
}

// renderTemplate parses and executes a named template from the embedded FS.
func (g *Generator) renderTemplate(name string, data any) ([]byte, error) {
	tmplContent, err := templateFS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", name, err)
	}

	tmpl, err := template.New(filepath.Base(name)).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template %s: %w", name, err)
	}

	return buf.Bytes(), nil
}

// writeFile writes data to a file with the given permissions, creating parent
// directories as needed. Use 0o600 for files containing secrets.
func (g *Generator) writeFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, permDirectory); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}
	return nil
}
