package claudecode

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"os/exec"
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
	client := newTestClient(claudeResponse{})

	_, err := client.Chat(context.Background(), []core.Message{
		{Role: "system", Content: "system only"},
	}, nil)

	if err == nil {
		t.Fatal("expected error for no user messages")
	}
}

func TestChat_CLIFailure(t *testing.T) {
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

func TestChat_ConversationHistory(t *testing.T) {
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

	// First arg should be -p, second should be the prompt
	if len(capturedArgs) < 2 || capturedArgs[0] != "-p" {
		t.Fatalf("expected -p flag, got %v", capturedArgs)
	}
	prompt := capturedArgs[1]

	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	// Prompt should contain conversation history
	if !contains(prompt, "Find inception") {
		t.Error("prompt missing first user message")
	}
	if !contains(prompt, "I found Inception") {
		t.Error("prompt missing assistant response")
	}
	if !contains(prompt, "Download it") {
		t.Error("prompt missing second user message")
	}
	// System message should NOT be in the prompt
	if contains(prompt, "You are MediaMate") {
		t.Error("system message should not be in prompt (sent via --append-system-prompt)")
	}

	// Check system prompt is passed via --append-system-prompt
	foundSystem := false
	for i, arg := range capturedArgs {
		if arg == "--append-system-prompt" && i+1 < len(capturedArgs) {
			if contains(capturedArgs[i+1], "You are MediaMate") {
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
	c := &Client{}
	if c.Name() != "claudecode" {
		t.Errorf("expected claudecode, got %s", c.Name())
	}
}

func TestBuildPrompt(t *testing.T) {
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
			got := buildPrompt(tt.messages)
			if got != tt.want {
				t.Errorf("buildPrompt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractSystemPrompt(t *testing.T) {
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
			got := extractSystemPrompt(tt.messages)
			if got != tt.want {
				t.Errorf("extractSystemPrompt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteMCPConfig(t *testing.T) {
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

func contains(s, substr string) bool {
	for i := range s {
		if i+len(substr) > len(s) {
			break
		}
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
