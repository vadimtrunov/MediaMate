package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vadimtrunov/MediaMate/internal/core"
)

func TestQueryModel_Init(t *testing.T) {
	a := newTestAgent(&core.Response{Content: "ok", Done: true})
	m := newQueryModel(context.Background(), a, "test")

	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return a batch command (spinner + query)")
	}
}

func TestQueryModel_InitialState(t *testing.T) {
	a := newTestAgent()
	m := newQueryModel(context.Background(), a, "hello")

	if m.done {
		t.Error("should not be done initially")
	}
	if m.message != "hello" {
		t.Errorf("message = %q, want %q", m.message, "hello")
	}
	if m.response != "" {
		t.Error("response should be empty initially")
	}
	if m.err != nil {
		t.Error("err should be nil initially")
	}
}

func TestQueryModel_ReceiveResponse(t *testing.T) {
	a := newTestAgent()
	m := newQueryModel(context.Background(), a, "test")

	updated, cmd := m.Update(queryResponseMsg{response: "got it!", err: nil})
	qm := updated.(queryModel)

	if !qm.done {
		t.Error("should be done after response")
	}
	if qm.response != "got it!" {
		t.Errorf("response = %q, want %q", qm.response, "got it!")
	}
	if qm.err != nil {
		t.Error("should not have error")
	}
	// Should quit after receiving response
	if cmd == nil {
		t.Error("should return quit command")
	}
}

func TestQueryModel_ReceiveError(t *testing.T) {
	a := newTestAgent()
	m := newQueryModel(context.Background(), a, "test")

	updated, _ := m.Update(queryResponseMsg{err: fmt.Errorf("failed")})
	qm := updated.(queryModel)

	if !qm.done {
		t.Error("should be done after error")
	}
	if qm.err == nil {
		t.Error("should have error")
	}
	if qm.err.Error() != "failed" {
		t.Errorf("err = %q, want %q", qm.err.Error(), "failed")
	}
}

func TestQueryModel_CtrlC(t *testing.T) {
	a := newTestAgent()
	m := newQueryModel(context.Background(), a, "test")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("ctrl+c should return quit command")
	}
}

func TestQueryModel_SpinnerUpdate(t *testing.T) {
	a := newTestAgent()
	m := newQueryModel(context.Background(), a, "test")

	// Get a valid tick message from the spinner's own Tick command.
	tickCmd := m.spinner.Tick
	tickMsg := tickCmd()

	updated, cmd := m.Update(tickMsg)
	qm := updated.(queryModel)
	if qm.done {
		t.Error("spinner tick should not mark as done")
	}
	if cmd == nil {
		t.Error("spinner tick should return next tick command")
	}
}

func TestQueryModel_ViewWhileLoading(t *testing.T) {
	a := newTestAgent()
	m := newQueryModel(context.Background(), a, "test")

	view := m.View()
	if !strings.Contains(view, "Thinking") {
		t.Errorf("loading view should contain 'Thinking', got %q", view)
	}
}

func TestQueryModel_ViewWithResponse(t *testing.T) {
	a := newTestAgent()
	m := newQueryModel(context.Background(), a, "test")
	m.done = true
	m.response = "Here is your answer"

	view := m.View()
	if !strings.Contains(view, "Here is your answer") {
		t.Errorf("response view should contain response, got %q", view)
	}
}

func TestQueryModel_ViewWithError(t *testing.T) {
	a := newTestAgent()
	m := newQueryModel(context.Background(), a, "test")
	m.done = true
	m.err = fmt.Errorf("something went wrong")

	view := m.View()
	if !strings.Contains(view, "something went wrong") {
		t.Errorf("error view should contain error message, got %q", view)
	}
}
