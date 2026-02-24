package main

import (
	"context"
	"fmt"
	"os/signal"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/vadimtrunov/MediaMate/internal/agent"
	"github.com/vadimtrunov/MediaMate/internal/config"
)

// newQueryCmd returns the "query" subcommand for one-shot queries.
func newQueryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "query [message]",
		Short: "Send a one-shot query to MediaMate",
		Long:  "Send a single message and get a response without entering interactive mode.",
		Example: `  mediamate query "download Dune"
  mediamate query "what movies are downloading?"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			message := strings.Join(args, " ")
			return runQuery(message)
		},
	}
}

// runQuery sends a single message to the agent and prints the response.
func runQuery(message string) error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	logger := config.SetupLogger(cfg.App.LogLevel)
	a, err := initServices(cfg, logger)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	p := tea.NewProgram(newQueryModel(ctx, a, message))
	m, err := p.Run()
	if err != nil {
		return fmt.Errorf("run query: %w", err)
	}

	qm, ok := m.(queryModel)
	if !ok {
		return fmt.Errorf("unexpected model type from tea program")
	}
	if qm.err != nil {
		return qm.err
	}
	return nil
}

// queryResponseMsg carries the agent response back to the TUI.
type queryResponseMsg struct {
	response string
	err      error
}

// queryModel is the Bubble Tea model for one-shot queries with a spinner.
type queryModel struct {
	ctx      context.Context
	agent    *agent.Agent
	message  string
	spinner  spinner.Model
	response string
	err      error
	done     bool
}

// newQueryModel creates a queryModel with the given message to send.
func newQueryModel(ctx context.Context, a *agent.Agent, message string) queryModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styleInfo
	return queryModel{
		ctx:     ctx,
		agent:   a,
		message: message,
		spinner: s,
	}
}

// Init starts the spinner and sends the query concurrently.
func (m queryModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.sendQuery())
}

// Update handles spinner ticks, key presses, and query responses.
func (m queryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case queryResponseMsg:
		m.response = msg.response
		m.err = msg.err
		m.done = true
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View renders the spinner while loading or the response when done.
func (m queryModel) View() string {
	if m.done {
		if m.err != nil {
			return styleError.Render("Error: "+m.err.Error()) + "\n"
		}
		return styleAssistant.Render(m.response) + "\n"
	}
	return m.spinner.View() + styleDim.Render(" Thinking...") + "\n"
}

// sendQuery returns a Bubble Tea command that sends the query to the agent asynchronously.
func (m queryModel) sendQuery() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.agent.HandleMessage(m.ctx, m.message)
		return queryResponseMsg{response: resp, err: err}
	}
}
