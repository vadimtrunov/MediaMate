package claude

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vadimtrunov/MediaMate/internal/core"
	"github.com/vadimtrunov/MediaMate/internal/httpclient"
)

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &Client{
		baseURL:   server.URL,
		apiKey:    "test-key",
		model:     "test-model",
		maxTokens: 1024,
		http:      httpclient.New(httpclient.DefaultConfig(), slog.New(slog.NewTextHandler(io.Discard, nil))),
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestChat_SimpleMessage(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key=test-key, got %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != apiVersion {
			t.Errorf("expected anthropic-version=%s, got %s", apiVersion, r.Header.Get("anthropic-version"))
		}

		json.NewEncoder(w).Encode(response{
			ID:         "msg_123",
			Type:       "message",
			Role:       "assistant",
			Content:    []contentBlock{{Type: "text", Text: "Hello!"}},
			StopReason: "end_turn",
		})
	}))

	resp, err := client.Chat(context.Background(), []core.Message{
		{Role: "user", Content: "Hi"},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hello!" {
		t.Errorf("expected Hello!, got %s", resp.Content)
	}
	if !resp.Done {
		t.Error("expected Done=true for end_turn")
	}
}

func TestChat_ToolUse(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(response{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []contentBlock{
				{Type: "text", Text: "Let me search for that."},
				{
					Type:  "tool_use",
					ID:    "call_abc",
					Name:  "search_movie",
					Input: map[string]any{"query": "inception"},
				},
			},
			StopReason: "tool_use",
		})
	}))

	resp, err := client.Chat(context.Background(), []core.Message{
		{Role: "user", Content: "Find inception"},
	}, []core.Tool{
		{Name: "search_movie", Description: "Search movies", Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string"},
			},
			"required": []string{"query"},
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Done {
		t.Error("expected Done=false for tool_use")
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "search_movie" {
		t.Errorf("expected search_movie, got %s", resp.ToolCalls[0].Name)
	}
	if resp.ToolCalls[0].Arguments["query"] != "inception" {
		t.Errorf("expected query=inception, got %v", resp.ToolCalls[0].Arguments["query"])
	}
}

func TestChat_SystemMessage(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req request
		json.NewDecoder(r.Body).Decode(&req)

		if req.System != "You are a media assistant." {
			t.Errorf("expected system message, got %q", req.System)
		}
		if len(req.Messages) != 1 {
			t.Errorf("expected 1 message (no system in messages), got %d", len(req.Messages))
		}

		json.NewEncoder(w).Encode(response{
			Content:    []contentBlock{{Type: "text", Text: "ok"}},
			StopReason: "end_turn",
		})
	}))

	_, err := client.Chat(context.Background(), []core.Message{
		{Role: "system", Content: "You are a media assistant."},
		{Role: "user", Content: "Hi"},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChat_ToolResults(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req request
		json.NewDecoder(r.Body).Decode(&req)

		// Should have: assistant with tool_use, user with tool_result
		if len(req.Messages) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(req.Messages))
		}

		// Third message should be user with tool_result content blocks
		msg := req.Messages[2]
		if msg.Role != "user" {
			t.Errorf("expected user role for tool result, got %s", msg.Role)
		}

		// Content should be an array of content blocks
		blocks, ok := msg.Content.([]any)
		if !ok {
			t.Fatalf("expected content to be an array, got %T", msg.Content)
		}
		if len(blocks) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(blocks))
		}

		json.NewEncoder(w).Encode(response{
			Content:    []contentBlock{{Type: "text", Text: "Found it!"}},
			StopReason: "end_turn",
		})
	}))

	_, err := client.Chat(context.Background(), []core.Message{
		{Role: "user", Content: "Find inception"},
		{Role: "assistant", Content: "Searching...", ToolCalls: []core.ToolCall{
			{ID: "call_1", Name: "search_movie", Arguments: map[string]any{"query": "inception"}},
		}},
		{Role: "user", Content: `[{"title": "Inception"}]`, ToolResultID: "call_1"},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChat_APIError(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiErrorResponse{
			Type: "error",
			Error: apiError{
				Type:    "invalid_request_error",
				Message: "Invalid API key",
			},
		})
	}))

	_, err := client.Chat(context.Background(), []core.Message{
		{Role: "user", Content: "Hi"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestConvertMessages_PlainMessages(t *testing.T) {
	system, msgs := convertMessages([]core.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
		{Role: "user", Content: "How are you?"},
	})

	if system != "" {
		t.Errorf("expected no system, got %q", system)
	}
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	if msgs[0].Content != "Hello" {
		t.Errorf("expected Hello, got %v", msgs[0].Content)
	}
}

func TestConvertMessages_SystemExtracted(t *testing.T) {
	system, msgs := convertMessages([]core.Message{
		{Role: "system", Content: "Be helpful"},
		{Role: "user", Content: "Hi"},
	})

	if system != "Be helpful" {
		t.Errorf("expected 'Be helpful', got %q", system)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
}

func TestConvertMessages_ToolCallAndResult(t *testing.T) {
	_, msgs := convertMessages([]core.Message{
		{Role: "user", Content: "Search"},
		{Role: "assistant", Content: "Searching", ToolCalls: []core.ToolCall{
			{ID: "c1", Name: "search", Arguments: map[string]any{"q": "test"}},
		}},
		{Role: "user", Content: "result1", ToolResultID: "c1"},
		{Role: "user", Content: "Final answer", ToolResultID: ""},
	})

	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}

	// Message 1: plain user
	if msgs[0].Role != "user" {
		t.Errorf("msg[0] role: expected user, got %s", msgs[0].Role)
	}

	// Message 2: assistant with tool_use blocks
	if msgs[1].Role != "assistant" {
		t.Errorf("msg[1] role: expected assistant, got %s", msgs[1].Role)
	}
	blocks, ok := msgs[1].Content.([]contentBlock)
	if !ok {
		t.Fatalf("msg[1] content: expected []contentBlock, got %T", msgs[1].Content)
	}
	if len(blocks) != 2 { // text + tool_use
		t.Errorf("expected 2 blocks, got %d", len(blocks))
	}

	// Message 3: tool result grouped
	if msgs[2].Role != "user" {
		t.Errorf("msg[2] role: expected user, got %s", msgs[2].Role)
	}
	resultBlocks, ok := msgs[2].Content.([]contentBlock)
	if !ok {
		t.Fatalf("msg[2] content: expected []contentBlock, got %T", msgs[2].Content)
	}
	if len(resultBlocks) != 1 {
		t.Errorf("expected 1 tool result block, got %d", len(resultBlocks))
	}
	if resultBlocks[0].Type != "tool_result" {
		t.Errorf("expected tool_result type, got %s", resultBlocks[0].Type)
	}
}

func TestConvertMessages_MultipleToolResults(t *testing.T) {
	_, msgs := convertMessages([]core.Message{
		{Role: "assistant", ToolCalls: []core.ToolCall{
			{ID: "c1", Name: "tool1"},
			{ID: "c2", Name: "tool2"},
		}},
		{Role: "user", Content: "r1", ToolResultID: "c1"},
		{Role: "user", Content: "r2", ToolResultID: "c2"},
	})

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (assistant + grouped results), got %d", len(msgs))
	}

	// Second message should have 2 tool_result blocks grouped
	resultBlocks, ok := msgs[1].Content.([]contentBlock)
	if !ok {
		t.Fatalf("expected []contentBlock, got %T", msgs[1].Content)
	}
	if len(resultBlocks) != 2 {
		t.Errorf("expected 2 tool result blocks, got %d", len(resultBlocks))
	}
}

func TestConvertTools(t *testing.T) {
	tools := convertTools([]core.Tool{
		{
			Name:        "search",
			Description: "Search movies",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
				},
				"required": []string{"query"},
			},
		},
	})

	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "search" {
		t.Errorf("expected search, got %s", tools[0].Name)
	}
	if tools[0].InputSchema.Type != "object" {
		t.Errorf("expected object schema type, got %s", tools[0].InputSchema.Type)
	}
	if len(tools[0].InputSchema.Required) != 1 || tools[0].InputSchema.Required[0] != "query" {
		t.Errorf("unexpected required: %v", tools[0].InputSchema.Required)
	}
}

func TestParseResponse_TextOnly(t *testing.T) {
	resp := parseResponse(&response{
		Content:    []contentBlock{{Type: "text", Text: "Hello"}},
		StopReason: "end_turn",
	})

	if resp.Content != "Hello" {
		t.Errorf("expected Hello, got %s", resp.Content)
	}
	if !resp.Done {
		t.Error("expected Done=true")
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("expected no tool calls, got %d", len(resp.ToolCalls))
	}
}

func TestParseResponse_ToolUse(t *testing.T) {
	resp := parseResponse(&response{
		Content: []contentBlock{
			{Type: "text", Text: "Searching..."},
			{Type: "tool_use", ID: "call_1", Name: "search", Input: map[string]any{"q": "test"}},
		},
		StopReason: "tool_use",
	})

	if resp.Done {
		t.Error("expected Done=false for tool_use")
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].ID != "call_1" {
		t.Errorf("expected call_1, got %s", resp.ToolCalls[0].ID)
	}
}

func TestHeaders(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key test-key, got %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("expected anthropic-version 2023-06-01, got %s", r.Header.Get("anthropic-version"))
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		json.NewEncoder(w).Encode(response{
			Content:    []contentBlock{{Type: "text", Text: "ok"}},
			StopReason: "end_turn",
		})
	}))

	_, err := client.Chat(context.Background(), []core.Message{{Role: "user", Content: "Hi"}}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
