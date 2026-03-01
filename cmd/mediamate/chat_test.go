package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vadimtrunov/MediaMate/internal/agent"
	"github.com/vadimtrunov/MediaMate/internal/core"
)

// mockLLM implements core.LLMProvider for testing.
type mockLLM struct {
	responses []*core.Response
	calls     int
}

func (m *mockLLM) Chat(_ context.Context, _ []core.Message, _ []core.Tool) (*core.Response, error) {
	if m.calls >= len(m.responses) {
		return nil, fmt.Errorf("no more responses")
	}
	resp := m.responses[m.calls]
	m.calls++
	return resp, nil
}

func (m *mockLLM) Name() string { return "mock" }

func (m *mockLLM) Close() error { return nil }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestAgent(responses ...*core.Response) *agent.Agent {
	llm := &mockLLM{responses: responses}
	return agent.New(llm, nil, nil, nil, nil, testLogger())
}

func TestChatModel_Init(t *testing.T) {
	a := newTestAgent()
	m := newChatModel(context.Background(), a)

	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return a command (batch of blink + spinner tick)")
	}
}

func TestChatModel_InitialState(t *testing.T) {
	a := newTestAgent()
	m := newChatModel(context.Background(), a)

	if m.waiting {
		t.Error("should not be waiting initially")
	}
	if m.ready {
		t.Error("should not be ready before WindowSizeMsg")
	}
	if len(m.messages) != 1 {
		t.Errorf("expected 1 system message, got %d", len(m.messages))
	}
	if m.messages[0].role != "system" {
		t.Errorf("expected system role, got %q", m.messages[0].role)
	}
	if m.histIdx != -1 {
		t.Errorf("expected histIdx = -1, got %d", m.histIdx)
	}
}

func TestChatModel_WindowSize(t *testing.T) {
	a := newTestAgent()
	m := newChatModel(context.Background(), a)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	cm := updated.(chatModel)

	if !cm.ready {
		t.Error("should be ready after WindowSizeMsg")
	}
	if cm.width != 80 {
		t.Errorf("width = %d, want 80", cm.width)
	}
	if cm.height != 24 {
		t.Errorf("height = %d, want 24", cm.height)
	}
}

func TestChatModel_CtrlC(t *testing.T) {
	a := newTestAgent()
	m := newChatModel(context.Background(), a)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("ctrl+c should return a quit command")
	}
}

func TestChatModel_QuitCommand(t *testing.T) {
	a := newTestAgent()
	m := newChatModel(context.Background(), a)
	// Need to be ready for enter to work
	m.ready = true

	m.textinput.SetValue("/quit")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("/quit should return a quit command")
	}
}

func TestChatModel_ExitCommand(t *testing.T) {
	a := newTestAgent()
	m := newChatModel(context.Background(), a)
	m.ready = true

	m.textinput.SetValue("/exit")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("/exit should return a quit command")
	}
}

func TestChatModel_ResetCommand(t *testing.T) {
	a := newTestAgent(&core.Response{Content: "hi", Done: true})
	m := newChatModel(context.Background(), a)

	// Initialize viewport
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(chatModel)

	// Add a user message to history
	m.messages = append(m.messages, chatEntry{role: "user", content: "test"})

	m.textinput.SetValue("/reset")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	cm := updated.(chatModel)

	if len(cm.messages) != 1 {
		t.Errorf("expected 1 message after reset, got %d", len(cm.messages))
	}
	if cm.messages[0].content != "Conversation reset." {
		t.Errorf("expected reset message, got %q", cm.messages[0].content)
	}
}

func TestChatModel_EmptyInput(t *testing.T) {
	a := newTestAgent()
	m := newChatModel(context.Background(), a)
	m.ready = true

	m.textinput.SetValue("")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	cm := updated.(chatModel)

	if cmd != nil {
		t.Error("empty input should not produce a command")
	}
	if len(cm.messages) != 1 {
		t.Errorf("empty input should not add messages, got %d", len(cm.messages))
	}
}

func TestChatModel_SendMessage(t *testing.T) {
	a := newTestAgent(&core.Response{Content: "hello back", Done: true})
	m := newChatModel(context.Background(), a)

	// Initialize viewport
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(chatModel)

	m.textinput.SetValue("hello")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	cm := updated.(chatModel)

	if !cm.waiting {
		t.Error("should be waiting after sending a message")
	}
	if cmd == nil {
		t.Error("should return a command to send the message")
	}
	// Should have system + user message
	if len(cm.messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(cm.messages))
	}
	if cm.messages[1].role != "user" {
		t.Errorf("expected user role, got %q", cm.messages[1].role)
	}
	if cm.messages[1].content != "hello" {
		t.Errorf("expected 'hello', got %q", cm.messages[1].content)
	}
}

func TestChatModel_ReceiveResponse(t *testing.T) {
	a := newTestAgent()
	m := newChatModel(context.Background(), a)

	// Initialize viewport
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(chatModel)
	m.waiting = true

	updated, _ = m.Update(chatResponseMsg{response: "hello!", err: nil})
	cm := updated.(chatModel)

	if cm.waiting {
		t.Error("should not be waiting after response")
	}
	lastMsg := cm.messages[len(cm.messages)-1]
	if lastMsg.role != "assistant" {
		t.Errorf("expected assistant role, got %q", lastMsg.role)
	}
	if lastMsg.content != "hello!" {
		t.Errorf("expected 'hello!', got %q", lastMsg.content)
	}
}

func TestChatModel_ReceiveError(t *testing.T) {
	a := newTestAgent()
	m := newChatModel(context.Background(), a)

	// Initialize viewport
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(chatModel)
	m.waiting = true

	updated, _ = m.Update(chatResponseMsg{err: fmt.Errorf("oops")})
	cm := updated.(chatModel)

	if cm.waiting {
		t.Error("should not be waiting after error response")
	}
	lastMsg := cm.messages[len(cm.messages)-1]
	if lastMsg.role != "system" {
		t.Errorf("expected system role for error, got %q", lastMsg.role)
	}
	if !strings.Contains(lastMsg.content, "oops") {
		t.Errorf("expected error message to contain 'oops', got %q", lastMsg.content)
	}
}

func TestChatModel_InputHistory(t *testing.T) {
	a := newTestAgent(
		&core.Response{Content: "r1", Done: true},
		&core.Response{Content: "r2", Done: true},
	)
	m := newChatModel(context.Background(), a)

	// Initialize viewport
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(chatModel)

	// Send two messages
	m.textinput.SetValue("first")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(chatModel)
	// Simulate response
	updated, _ = m.Update(chatResponseMsg{response: "r1"})
	m = updated.(chatModel)

	m.textinput.SetValue("second")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(chatModel)
	updated, _ = m.Update(chatResponseMsg{response: "r2"})
	m = updated.(chatModel)

	if len(m.history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(m.history))
	}

	// Press up — should get "second"
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(chatModel)
	if m.textinput.Value() != "second" {
		t.Errorf("up once: expected 'second', got %q", m.textinput.Value())
	}

	// Press up again — should get "first"
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(chatModel)
	if m.textinput.Value() != "first" {
		t.Errorf("up twice: expected 'first', got %q", m.textinput.Value())
	}

	// Press down — should get "second"
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(chatModel)
	if m.textinput.Value() != "second" {
		t.Errorf("down once: expected 'second', got %q", m.textinput.Value())
	}

	// Press down again — should clear input
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(chatModel)
	if m.textinput.Value() != "" {
		t.Errorf("down past end: expected empty, got %q", m.textinput.Value())
	}
}

func TestChatModel_IgnoreInputWhileWaiting(t *testing.T) {
	a := newTestAgent()
	m := newChatModel(context.Background(), a)
	m.ready = true
	m.waiting = true

	m.textinput.SetValue("should be ignored")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	cm := updated.(chatModel)

	if cmd != nil {
		t.Error("should not send when waiting")
	}
	if len(cm.messages) != 1 {
		t.Error("should not add message when waiting")
	}
}

func TestChatModel_RenderMessages(t *testing.T) {
	a := newTestAgent()
	m := newChatModel(context.Background(), a)
	m.messages = []chatEntry{
		{role: "system", content: "Welcome"},
		{role: "user", content: "Hi"},
		{role: "assistant", content: "Hello!"},
	}

	rendered := m.renderMessages()
	if rendered == "" {
		t.Error("renderMessages returned empty string")
	}
	if !strings.Contains(rendered, "Hi") {
		t.Error("rendered output should contain user message")
	}
	if !strings.Contains(rendered, "Hello!") {
		t.Error("rendered output should contain assistant message")
	}
}

func TestChatModel_ViewNotReady(t *testing.T) {
	a := newTestAgent()
	m := newChatModel(context.Background(), a)

	view := m.View()
	if view != "Initializing..." {
		t.Errorf("expected 'Initializing...', got %q", view)
	}
}
