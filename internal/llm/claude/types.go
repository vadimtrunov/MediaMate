package claude

// request is the Claude Messages API request body.
type request struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system,omitempty"`
	Messages  []apiMessage `json:"messages"`
	Tools     []apiTool    `json:"tools,omitempty"`
}

// apiMessage is a message in Claude API format.
type apiMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []contentBlock
}

// contentBlock represents a content block in the Claude API.
type contentBlock struct {
	Type      string `json:"type"`                  // "text", "tool_use", "tool_result"
	Text      string `json:"text,omitempty"`        // for "text"
	ID        string `json:"id,omitempty"`          // for "tool_use"
	Name      string `json:"name,omitempty"`        // for "tool_use"
	Input     any    `json:"input,omitempty"`       // for "tool_use"
	ToolUseID string `json:"tool_use_id,omitempty"` // for "tool_result"
	Content   string `json:"content,omitempty"`     // for "tool_result"
	IsError   bool   `json:"is_error,omitempty"`    // for "tool_result"
}

// apiTool is a tool definition in Claude API format.
type apiTool struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	InputSchema apiSchema `json:"input_schema"`
}

// apiSchema represents a JSON Schema for tool parameters.
type apiSchema struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties,omitempty"`
	Required   []string       `json:"required,omitempty"`
}

// response is the Claude Messages API response body.
type response struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Role       string         `json:"role"`
	Content    []contentBlock `json:"content"`
	Model      string         `json:"model"`
	StopReason string         `json:"stop_reason"`
	Usage      apiUsage       `json:"usage"`
}

// apiUsage tracks token usage.
type apiUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// apiErrorResponse is the error response from Claude API.
type apiErrorResponse struct {
	Type  string   `json:"type"`
	Error apiError `json:"error"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
