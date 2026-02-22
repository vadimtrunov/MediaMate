package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/vadimtrunov/MediaMate/internal/core"
	"github.com/vadimtrunov/MediaMate/internal/httpclient"
)

const (
	defaultBaseURL   = "https://api.anthropic.com"
	defaultModel     = "claude-sonnet-4-20250514"
	defaultMaxTokens = 4096
	apiVersion       = "2023-06-01"
)

// Client implements core.LLMProvider for the Claude Messages API.
type Client struct {
	baseURL   string
	apiKey    string
	model     string
	maxTokens int
	http      *httpclient.Client
	logger    *slog.Logger
}

var _ core.LLMProvider = (*Client)(nil)

// New creates a new Claude client.
func New(apiKey, model, baseURL string, logger *slog.Logger) *Client {
	if model == "" {
		model = defaultModel
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &Client{
		baseURL:   baseURL,
		apiKey:    apiKey,
		model:     model,
		maxTokens: defaultMaxTokens,
		http:      httpclient.New(httpclient.DefaultConfig(), logger),
		logger:    logger,
	}
}

// Chat sends messages to Claude and returns the response.
func (c *Client) Chat(ctx context.Context, messages []core.Message, tools []core.Tool) (*core.Response, error) {
	apiReq := c.buildRequest(messages, tools)

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", apiVersion)
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("claude API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var apiResp response
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	c.logger.Debug("claude response",
		slog.String("model", apiResp.Model),
		slog.String("stop_reason", apiResp.StopReason),
		slog.Int("input_tokens", apiResp.Usage.InputTokens),
		slog.Int("output_tokens", apiResp.Usage.OutputTokens),
	)

	return parseResponse(&apiResp), nil
}

// Name returns "claude".
func (c *Client) Name() string { return "claude" }

func (c *Client) buildRequest(messages []core.Message, tools []core.Tool) *request {
	system, apiMsgs := convertMessages(messages)

	req := &request{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		System:    system,
		Messages:  apiMsgs,
	}

	if len(tools) > 0 {
		req.Tools = convertTools(tools)
	}

	return req
}

func (c *Client) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var errResp apiErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		return fmt.Errorf("claude API error %d: %s: %s", resp.StatusCode, errResp.Error.Type, errResp.Error.Message)
	}

	return fmt.Errorf("claude API error %d: %s", resp.StatusCode, string(body))
}

// convertMessages converts core.Messages to Claude API format.
// Extracts system messages, groups tool results, and builds content blocks.
func convertMessages(messages []core.Message) (string, []apiMessage) {
	var system string
	var apiMsgs []apiMessage
	var pendingToolResults []contentBlock

	flushToolResults := func() {
		if len(pendingToolResults) > 0 {
			apiMsgs = append(apiMsgs, apiMessage{
				Role:    "user",
				Content: pendingToolResults,
			})
			pendingToolResults = nil
		}
	}

	for _, msg := range messages {
		if msg.Role == "system" {
			if system != "" {
				system += "\n\n"
			}
			system += msg.Content
			continue
		}

		if msg.ToolResultID != "" {
			pendingToolResults = append(pendingToolResults, contentBlock{
				Type:      "tool_result",
				ToolUseID: msg.ToolResultID,
				Content:   msg.Content,
				IsError:   msg.IsError,
			})
			continue
		}

		flushToolResults()

		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			apiMsgs = append(apiMsgs, buildToolUseMessage(msg))
			continue
		}

		apiMsgs = append(apiMsgs, apiMessage{Role: msg.Role, Content: msg.Content})
	}

	flushToolResults()
	return system, apiMsgs
}

func buildToolUseMessage(msg core.Message) apiMessage {
	var blocks []contentBlock
	if msg.Content != "" {
		blocks = append(blocks, contentBlock{Type: "text", Text: msg.Content})
	}
	for _, tc := range msg.ToolCalls {
		blocks = append(blocks, contentBlock{
			Type:  "tool_use",
			ID:    tc.ID,
			Name:  tc.Name,
			Input: tc.Arguments,
		})
	}
	return apiMessage{Role: "assistant", Content: blocks}
}

// convertTools converts core.Tools to Claude API tools.
func convertTools(tools []core.Tool) []apiTool {
	apiTools := make([]apiTool, 0, len(tools))
	for _, t := range tools {
		schema := apiSchema{Type: "object"}
		if props, ok := t.Parameters["properties"]; ok {
			if p, ok := props.(map[string]any); ok {
				schema.Properties = p
			}
		}
		if req, ok := t.Parameters["required"]; ok {
			switch r := req.(type) {
			case []string:
				schema.Required = r
			case []any:
				for _, v := range r {
					if s, ok := v.(string); ok {
						schema.Required = append(schema.Required, s)
					}
				}
			}
		}

		apiTools = append(apiTools, apiTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		})
	}
	return apiTools
}

// parseResponse converts the Claude API response to core.Response.
func parseResponse(resp *response) *core.Response {
	result := &core.Response{
		Done: resp.StopReason == "end_turn",
	}

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			if result.Content != "" {
				result.Content += "\n"
			}
			result.Content += block.Text
		case "tool_use":
			args := make(map[string]any)
			if v, ok := block.Input.(map[string]any); ok {
				args = v
			}
			result.ToolCalls = append(result.ToolCalls, core.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: args,
			})
		}
	}

	return result
}
