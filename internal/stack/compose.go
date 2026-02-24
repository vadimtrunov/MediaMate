package stack

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

// ContainerStatus represents the status of a single container as reported by
// docker compose ps --format json.
type ContainerStatus struct {
	Name    string `json:"Name"`
	Service string `json:"Service"`
	State   string `json:"State"`
	Health  string `json:"Health"`
	Status  string `json:"Status"`
}

// Compose is a wrapper around the docker compose CLI. It shells out to
// docker compose via os/exec rather than using the Docker SDK directly.
type Compose struct {
	logger *slog.Logger
	file   string // path to docker-compose.yml
}

// NewCompose creates a Compose instance for the given docker-compose.yml path.
// If logger is nil, slog.Default() is used.
func NewCompose(file string, logger *slog.Logger) *Compose {
	if logger == nil {
		logger = slog.Default()
	}
	return &Compose{
		logger: logger,
		file:   file,
	}
}

// findDockerCompose checks whether the docker binary is available on the
// system PATH. It returns the resolved path to docker or an error if not found.
func findDockerCompose() (string, error) {
	path, err := exec.LookPath("docker")
	if err != nil {
		return "", fmt.Errorf("docker not found: install Docker and Docker Compose plugin: %w", err)
	}
	return path, nil
}

// run executes a docker compose command with the given arguments and returns the
// combined stdout/stderr output. The compose file is injected automatically via
// the -f flag.
func (c *Compose) run(ctx context.Context, args ...string) ([]byte, error) {
	dockerPath, err := findDockerCompose()
	if err != nil {
		return nil, err
	}

	// Build the full argument list: docker compose -f <file> <args...>
	fullArgs := make([]string, 0, 3+len(args))
	fullArgs = append(fullArgs, "compose", "-f", c.file)
	fullArgs = append(fullArgs, args...)

	c.logger.Debug("running docker compose", slog.String("command", dockerPath), slog.Any("args", fullArgs))

	cmd := exec.CommandContext(ctx, dockerPath, fullArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("docker compose %s: %w: %s", strings.Join(args, " "), err, output)
	}
	return output, nil
}

// Up starts all services defined in the compose file in detached mode.
// If the command fails, the combined output is included in the error.
func (c *Compose) Up(ctx context.Context) error {
	c.logger.Info("starting stack", slog.String("file", c.file))
	_, err := c.run(ctx, "up", "-d")
	if err != nil {
		return fmt.Errorf("compose up: %w", err)
	}
	return nil
}

// Down stops and removes all containers defined in the compose file.
func (c *Compose) Down(ctx context.Context) error {
	c.logger.Info("stopping stack", slog.String("file", c.file))
	_, err := c.run(ctx, "down")
	if err != nil {
		return fmt.Errorf("compose down: %w", err)
	}
	return nil
}

// PS returns the status of all containers in the compose project. It parses
// the JSON output of docker compose ps --format json, which emits one JSON
// object per line (not a JSON array).
func (c *Compose) PS(ctx context.Context) ([]ContainerStatus, error) {
	c.logger.Debug("listing containers", slog.String("file", c.file))
	output, err := c.run(ctx, "ps", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("compose ps: %w", err)
	}

	var containers []ContainerStatus

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var cs ContainerStatus
		if err := json.Unmarshal(line, &cs); err != nil {
			return nil, fmt.Errorf("parse container status: %w: %s", err, line)
		}
		containers = append(containers, cs)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read compose ps output: %w", err)
	}

	return containers, nil
}
