package claudecode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vadimtrunov/MediaMate/internal/core"
)

// Client implements core.LLMProvider using the Claude Code CLI as a subprocess.
// It delegates tool calling to Claude Code via MCP, so the agent's tool loop
// receives only final text responses (no ToolCalls).
type Client struct {
	cliPath    string
	model      string
	configPath string // absolute path to mediamate config yaml
	mcpConfig  string // path to generated temp MCP config json
	logger     *slog.Logger
	newCmd     func(ctx context.Context, name string, args ...string) *exec.Cmd
}

var _ core.LLMProvider = (*Client)(nil)

// New creates a new Claude Code provider.
// It requires the `claude` CLI to be installed and on PATH.
// configPath is the path to the mediamate YAML config file (used by mcp-serve).
func New(model, configPath string, logger *slog.Logger) (*Client, error) {
	if logger == nil {
		logger = slog.Default()
	}

	cliPath, err := exec.LookPath("claude")
	if err != nil {
		return nil, fmt.Errorf("claude CLI not found in PATH: %w (install from https://claude.ai/code)", err)
	}

	selfPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("cannot determine mediamate binary path: %w", err)
	}

	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}

	mcpConfigPath, err := writeMCPConfig(selfPath, absConfigPath)
	if err != nil {
		return nil, fmt.Errorf("write MCP config: %w", err)
	}

	logger.Info("claudecode provider initialized",
		slog.String("cli", cliPath),
		slog.String("mcp_config", mcpConfigPath),
	)

	return &Client{
		cliPath:    cliPath,
		model:      model,
		configPath: absConfigPath,
		mcpConfig:  mcpConfigPath,
		logger:     logger,
		newCmd:     exec.CommandContext,
	}, nil
}

// Chat sends a prompt to Claude Code CLI and returns the final response.
// Claude Code handles tool calling internally via MCP, so the returned
// Response always has Done=true and no ToolCalls.
func (c *Client) Chat(ctx context.Context, messages []core.Message, _ []core.Tool) (*core.Response, error) {
	prompt := buildPrompt(messages)
	if prompt == "" {
		return nil, fmt.Errorf("no user messages to send")
	}

	args := []string{
		"-p", prompt,
		"--output-format", "json",
		"--mcp-config", c.mcpConfig,
		"--allowedTools", "mcp__mediamate__*",
	}

	systemPrompt := extractSystemPrompt(messages)
	if systemPrompt != "" {
		args = append(args, "--append-system-prompt", systemPrompt)
	}

	if c.model != "" {
		args = append(args, "--model", c.model)
	}

	c.logger.Debug("running claude CLI", slog.String("prompt_length", fmt.Sprintf("%d", len(prompt))))

	cmd := c.newCmd(ctx, c.cliPath, args...)
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("claude CLI failed (exit %d): %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("claude CLI failed: %w", err)
	}

	var resp claudeResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("parse claude response: %w", err)
	}

	if resp.IsError {
		return nil, fmt.Errorf("claude code error: %s", resp.Result)
	}

	return &core.Response{
		Content: resp.Result,
		Done:    true,
	}, nil
}

// Close removes the temporary MCP config file.
// It is safe to call multiple times.
func (c *Client) Close() error {
	if c.mcpConfig == "" {
		return nil
	}
	err := os.Remove(c.mcpConfig)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	c.mcpConfig = ""
	return nil
}

// Name returns the provider name.
func (c *Client) Name() string { return "claudecode" }

// claudeResponse represents the JSON output from `claude -p --output-format json`.
type claudeResponse struct {
	Type      string  `json:"type"`
	Subtype   string  `json:"subtype"`
	Result    string  `json:"result"`
	IsError   bool    `json:"is_error"`
	CostUSD   float64 `json:"cost_usd"`
	SessionID string  `json:"session_id"`
}

// writeMCPConfig creates a temporary JSON config file for Claude Code's --mcp-config flag.
// It configures a "mediamate" MCP server that runs `mediamate mcp-serve`.
func writeMCPConfig(selfPath, configPath string) (string, error) {
	cfg := map[string]any{
		"mcpServers": map[string]any{
			"mediamate": map[string]any{
				"type":    "stdio",
				"command": selfPath,
				"args":    []string{"mcp-serve", "-c", configPath},
			},
		},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal MCP config: %w", err)
	}

	f, err := os.CreateTemp("", "mediamate-mcp-*.json")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		name := f.Name()
		_ = os.Remove(name) //nolint:gosec // path from os.CreateTemp is safe
		return "", fmt.Errorf("write temp file: %w", err)
	}

	return f.Name(), nil
}

// extractSystemPrompt collects all system messages into a single string.
func extractSystemPrompt(messages []core.Message) string {
	var parts []string
	for _, msg := range messages {
		if msg.Role == "system" {
			parts = append(parts, msg.Content)
		}
	}
	return strings.Join(parts, "\n\n")
}

// buildPrompt formats conversation history as a text prompt for Claude Code.
// System messages are excluded (sent via --append-system-prompt).
// Tool results are excluded (Claude Code handles tools internally via MCP).
func buildPrompt(messages []core.Message) string {
	var parts []string
	for _, msg := range messages {
		if msg.Role == "system" || msg.ToolResultID != "" {
			continue
		}
		switch msg.Role {
		case "user":
			parts = append(parts, "User: "+msg.Content)
		case "assistant":
			if msg.Content != "" {
				parts = append(parts, "Assistant: "+msg.Content)
			}
		}
	}
	return strings.Join(parts, "\n\n")
}
