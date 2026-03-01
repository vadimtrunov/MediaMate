package claudecode

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/vadimtrunov/MediaMate/internal/core"
)

var testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func newTestClient(response claudeResponse) *Client {
	responseJSON, _ := json.Marshal(response)

	return &Client{
		cliPath:   "claude",
		mcpConfig: "/tmp/test-mcp.json",
		logger:    testLogger,
		newCmd: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "echo", string(responseJSON)) //nolint:gosec // test helper
		},
	}
}

func TestChat_SimpleResponse(t *testing.T) {
	t.Parallel()
	client := newTestClient(claudeResponse{
		Type:    "result",
		Subtype: "success",
		Result:  "I found Inception (2010) - a great sci-fi thriller by Christopher Nolan.",
	})

	resp, err := client.Chat(context.Background(), []core.Message{
		{Role: "system", Content: "You are MediaMate."},
		{Role: "user", Content: "Find inception"},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "I found Inception (2010) - a great sci-fi thriller by Christopher Nolan." {
		t.Errorf("unexpected content: %s", resp.Content)
	}
	if !resp.Done {
		t.Error("expected Done=true")
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("expected no tool calls, got %d", len(resp.ToolCalls))
	}
}

func TestChat_ErrorResponse(t *testing.T) {
	t.Parallel()
	client := newTestClient(claudeResponse{
		Type:    "result",
		Subtype: "error",
		Result:  "something went wrong",
		IsError: true,
	})

	_, err := client.Chat(context.Background(), []core.Message{
		{Role: "user", Content: "test"},
	}, nil)

	if err == nil {
		t.Fatal("expected error for is_error=true response")
	}
}

func TestChat_NoMessages(t *testing.T) {
	t.Parallel()
	client := newTestClient(claudeResponse{})

	_, err := client.Chat(context.Background(), []core.Message{
		{Role: "system", Content: "system only"},
	}, nil)

	if err == nil {
		t.Fatal("expected error for no user messages")
	}
}

func TestChat_CLIFailure(t *testing.T) {
	t.Parallel()
	client := &Client{
		cliPath:   "false", // always exits with code 1
		mcpConfig: "/tmp/test.json",
		logger:    testLogger,
		newCmd: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "false")
		},
	}

	_, err := client.Chat(context.Background(), []core.Message{
		{Role: "user", Content: "test"},
	}, nil)

	if err == nil {
		t.Fatal("expected error for CLI failure")
	}
}

func TestChat_WithModel(t *testing.T) {
	t.Parallel()
	var capturedArgs []string
	client := &Client{
		cliPath:   "claude",
		model:     "claude-sonnet-4-20250514",
		mcpConfig: "/tmp/test.json",
		logger:    testLogger,
		newCmd: func(ctx context.Context, _ string, args ...string) *exec.Cmd {
			capturedArgs = args
			resp := claudeResponse{Type: "result", Result: "ok"}
			data, _ := json.Marshal(resp)
			return exec.CommandContext(ctx, "echo", string(data)) //nolint:gosec // test helper
		},
	}

	_, err := client.Chat(context.Background(), []core.Message{
		{Role: "user", Content: "test"},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for i, arg := range capturedArgs {
		if arg == "--model" && i+1 < len(capturedArgs) && capturedArgs[i+1] == "claude-sonnet-4-20250514" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected --model flag in args: %v", capturedArgs)
	}
}

func chatWithHistory(t *testing.T) []string {
	t.Helper()
	var capturedArgs []string
	client := &Client{
		cliPath:   "claude",
		mcpConfig: "/tmp/test.json",
		logger:    testLogger,
		newCmd: func(ctx context.Context, _ string, args ...string) *exec.Cmd {
			capturedArgs = args
			resp := claudeResponse{Type: "result", Result: "ok"}
			data, _ := json.Marshal(resp)
			return exec.CommandContext(ctx, "echo", string(data)) //nolint:gosec // test helper
		},
	}

	_, err := client.Chat(context.Background(), []core.Message{
		{Role: "system", Content: "You are MediaMate."},
		{Role: "user", Content: "Find inception"},
		{Role: "assistant", Content: "I found Inception (2010)."},
		{Role: "user", Content: "Download it"},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return capturedArgs
}

func TestChat_PromptContainsHistory(t *testing.T) {
	t.Parallel()
	capturedArgs := chatWithHistory(t)

	if len(capturedArgs) < 2 || capturedArgs[0] != "-p" {
		t.Fatalf("expected -p flag, got %v", capturedArgs)
	}
	prompt := capturedArgs[1]

	if !strings.Contains(prompt, "Find inception") {
		t.Error("prompt missing first user message")
	}
	if !strings.Contains(prompt, "I found Inception") {
		t.Error("prompt missing assistant response")
	}
	if !strings.Contains(prompt, "Download it") {
		t.Error("prompt missing second user message")
	}
	if strings.Contains(prompt, "You are MediaMate") {
		t.Error("system message should not be in prompt (sent via --append-system-prompt)")
	}
}

func TestChat_SystemPromptFlag(t *testing.T) {
	t.Parallel()
	capturedArgs := chatWithHistory(t)

	foundSystem := false
	for i, arg := range capturedArgs {
		if arg == "--append-system-prompt" && i+1 < len(capturedArgs) {
			if strings.Contains(capturedArgs[i+1], "You are MediaMate") {
				foundSystem = true
			}
			break
		}
	}
	if !foundSystem {
		t.Error("expected system prompt via --append-system-prompt")
	}
}

func TestName(t *testing.T) {
	t.Parallel()
	c := &Client{}
	if c.Name() != "claudecode" {
		t.Errorf("expected claudecode, got %s", c.Name())
	}
}

func TestBuildPrompt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		messages []core.Message
		want     string
	}{
		{
			name: "single_user_message",
			messages: []core.Message{
				{Role: "user", Content: "hello"},
			},
			want: "User: hello",
		},
		{
			name: "conversation",
			messages: []core.Message{
				{Role: "user", Content: "hi"},
				{Role: "assistant", Content: "hello"},
				{Role: "user", Content: "bye"},
			},
			want: "User: hi\n\nAssistant: hello\n\nUser: bye",
		},
		{
			name: "skips_system",
			messages: []core.Message{
				{Role: "system", Content: "be helpful"},
				{Role: "user", Content: "hi"},
			},
			want: "User: hi",
		},
		{
			name: "skips_tool_results",
			messages: []core.Message{
				{Role: "user", Content: "search"},
				{Role: "user", Content: "result", ToolResultID: "call_1"},
				{Role: "user", Content: "next"},
			},
			want: "User: search\n\nUser: next",
		},
		{
			name: "skips_empty_assistant",
			messages: []core.Message{
				{Role: "user", Content: "hi"},
				{Role: "assistant", Content: ""},
				{Role: "user", Content: "bye"},
			},
			want: "User: hi\n\nUser: bye",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildPrompt(tt.messages)
			if got != tt.want {
				t.Errorf("buildPrompt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractSystemPrompt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		messages []core.Message
		want     string
	}{
		{
			name: "single_system",
			messages: []core.Message{
				{Role: "system", Content: "be helpful"},
				{Role: "user", Content: "hi"},
			},
			want: "be helpful",
		},
		{
			name: "multiple_system",
			messages: []core.Message{
				{Role: "system", Content: "part1"},
				{Role: "system", Content: "part2"},
			},
			want: "part1\n\npart2",
		},
		{
			name: "no_system",
			messages: []core.Message{
				{Role: "user", Content: "hi"},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractSystemPrompt(tt.messages)
			if got != tt.want {
				t.Errorf("extractSystemPrompt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteMCPConfig(t *testing.T) {
	t.Parallel()
	path, err := writeMCPConfig("/usr/local/bin/mediamate", "/etc/mediamate/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	servers, ok := cfg["mcpServers"].(map[string]any)
	if !ok {
		t.Fatal("missing mcpServers")
	}
	mediamate, ok := servers["mediamate"].(map[string]any)
	if !ok {
		t.Fatal("missing mediamate server")
	}
	if mediamate["command"] != "/usr/local/bin/mediamate" {
		t.Errorf("unexpected command: %v", mediamate["command"])
	}
	args, ok := mediamate["args"].([]any)
	if !ok || len(args) != 3 {
		t.Fatalf("unexpected args: %v", mediamate["args"])
	}
	if args[0] != "mcp-serve" || args[1] != "-c" || args[2] != "/etc/mediamate/config.yaml" {
		t.Errorf("unexpected args: %v", args)
	}
}
