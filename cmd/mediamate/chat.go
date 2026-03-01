package main

import (
	"context"
	"fmt"
	"os/signal"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/vadimtrunov/MediaMate/internal/agent"
	"github.com/vadimtrunov/MediaMate/internal/config"
)

// newChatCmd returns the "chat" subcommand for interactive conversation.
func newChatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "chat",
		Short: "Start an interactive chat session",
		Long: "Start an interactive conversation with MediaMate.\n" +
			"Use /reset to clear history, /quit or Ctrl+C to exit.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runChat()
		},
	}
}

// runChat initializes services and starts the Bubble Tea chat TUI.
func runChat() error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	logger := config.SetupLogger(cfg.App.LogLevel)
	a, err := initServices(cfg, logger)
	if err != nil {
		return err
	}
	defer func() { _ = a.Close() }()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	p := tea.NewProgram(newChatModel(ctx, a), tea.WithAltScreen())

	// Bridge OS signal cancellation into the Bubble Tea event loop.
	go func() {
		<-ctx.Done()
		p.Send(tea.Quit())
	}()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run chat: %w", err)
	}
	return nil
}

// chatResponseMsg carries the agent response back to the TUI.
type chatResponseMsg struct {
	response string
	err      error
}

// Chat role constants.
const (
	roleUser      = "user"
	roleAssistant = "assistant"
	roleSystem    = "system"
)

// chatEntry represents a single message in the chat history.
type chatEntry struct {
	role    string
	content string
}

// chatModel is the Bubble Tea model for interactive chat.
type chatModel struct {
	ctx       context.Context
	agent     *agent.Agent
	viewport  viewport.Model
	textinput textinput.Model
	spinner   spinner.Model
	messages  []chatEntry
	history   []string // input history
	histIdx   int      // current position in input history (-1 = not browsing)
	waiting   bool
	width     int
	height    int
	ready     bool
}

// newChatModel creates a chatModel with text input, spinner, and welcome message.
func newChatModel(ctx context.Context, a *agent.Agent) chatModel {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()
	ti.CharLimit = 2000

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styleInfo

	return chatModel{
		ctx:       ctx,
		agent:     a,
		textinput: ti,
		spinner:   s,
		messages: []chatEntry{
			{role: roleSystem, content: "MediaMate ready. Type a message to start, /reset to clear history, /quit to exit."},
		},
		histIdx: -1,
	}
}

// Init starts the text input blink cursor.
func (m chatModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles incoming messages and user input.
func (m chatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleResize(msg)

	case tea.KeyMsg:
		model, cmd, handled := m.handleKey(msg)
		if handled {
			return model, cmd
		}

	case chatResponseMsg:
		m.handleResponse(msg)
		return m, nil

	case spinner.TickMsg:
		if m.waiting {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	if !m.waiting {
		var tiCmd tea.Cmd
		m.textinput, tiCmd = m.textinput.Update(msg)
		cmds = append(cmds, tiCmd)
	}

	if m.ready {
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		cmds = append(cmds, vpCmd)
	}

	return m, tea.Batch(cmds...)
}

// handleResize adjusts viewport and text input dimensions on terminal resize.
func (m *chatModel) handleResize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
	headerHeight := 1
	inputHeight := 3
	spinnerHeight := 0
	if m.waiting {
		spinnerHeight = 1
	}
	vpHeight := m.height - headerHeight - inputHeight - spinnerHeight
	if vpHeight < 1 {
		vpHeight = 1
	}
	if !m.ready {
		m.viewport = viewport.New(m.width, vpHeight)
		m.viewport.SetContent(m.renderMessages())
		m.ready = true
	} else {
		m.viewport.Width = m.width
		m.viewport.Height = vpHeight
	}
	m.textinput.Width = m.width - 4
}

// handleKey dispatches key events to the appropriate handler.
func (m *chatModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "ctrl+c":
		return *m, tea.Quit, true
	case "up":
		return m.handleHistoryUp()
	case "down":
		return m.handleHistoryDown()
	case "enter":
		return m.handleEnter()
	}
	return *m, nil, false
}

// handleHistoryUp navigates to the previous entry in the input history.
func (m *chatModel) handleHistoryUp() (tea.Model, tea.Cmd, bool) {
	if m.waiting || len(m.history) == 0 {
		return *m, nil, false
	}
	if m.histIdx == -1 {
		m.histIdx = len(m.history) - 1
	} else if m.histIdx > 0 {
		m.histIdx--
	}
	m.textinput.SetValue(m.history[m.histIdx])
	m.textinput.CursorEnd()
	return *m, nil, true
}

// handleHistoryDown navigates to the next entry in the input history.
func (m *chatModel) handleHistoryDown() (tea.Model, tea.Cmd, bool) {
	if m.waiting || m.histIdx < 0 {
		return *m, nil, false
	}
	if m.histIdx < len(m.history)-1 {
		m.histIdx++
		m.textinput.SetValue(m.history[m.histIdx])
	} else {
		m.histIdx = -1
		m.textinput.SetValue("")
	}
	m.textinput.CursorEnd()
	return *m, nil, true
}

// handleEnter processes the current input: slash commands or sends a message to the agent.
func (m *chatModel) handleEnter() (tea.Model, tea.Cmd, bool) {
	if m.waiting {
		return *m, nil, true
	}
	input := strings.TrimSpace(m.textinput.Value())
	if input == "" {
		return *m, nil, true
	}
	m.textinput.SetValue("")
	m.histIdx = -1

	switch input {
	case "/quit", "/exit":
		return *m, tea.Quit, true
	case "/reset":
		m.agent.Reset()
		m.messages = []chatEntry{
			{role: roleSystem, content: "Conversation reset."},
		}
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return *m, nil, true
	}

	m.history = append(m.history, input)
	m.messages = append(m.messages, chatEntry{role: roleUser, content: input})
	m.waiting = true
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
	return *m, tea.Batch(m.sendMessage(input), m.spinner.Tick), true
}

// handleResponse appends the agent's response or error to the chat history.
func (m *chatModel) handleResponse(msg chatResponseMsg) {
	m.waiting = false
	if msg.err != nil {
		m.messages = append(m.messages, chatEntry{role: roleSystem, content: "Error: " + msg.err.Error()})
	} else {
		m.messages = append(m.messages, chatEntry{role: roleAssistant, content: msg.response})
	}
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
}

// View renders the chat UI with message history, spinner, and input field.
func (m chatModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("5")).
		Render("MediaMate Chat")

	var spinnerLine string
	if m.waiting {
		spinnerLine = "\n" + m.spinner.View() + styleDim.Render(" Thinking...")
	}

	inputBorder := lipgloss.NewStyle().
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8")).
		PaddingTop(0)

	return title + "\n" +
		m.viewport.View() +
		spinnerLine + "\n" +
		inputBorder.Render(m.textinput.View())
}

// renderMessages formats all chat entries into a styled string for the viewport.
func (m chatModel) renderMessages() string {
	if len(m.messages) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, msg := range m.messages {
		switch msg.role {
		case roleUser:
			sb.WriteString(styleUser.Render("You: "))
			sb.WriteString(msg.content)
		case roleAssistant:
			sb.WriteString(styleAssistant.Render(msg.content))
		case roleSystem:
			sb.WriteString(styleDim.Render(msg.content))
		}
		sb.WriteString("\n\n")
	}
	return sb.String()
}

// sendMessage returns a Bubble Tea command that sends the input to the agent asynchronously.
func (m chatModel) sendMessage(input string) tea.Cmd {
	return func() tea.Msg {
		resp, err := m.agent.HandleMessage(m.ctx, input)
		return chatResponseMsg{response: resp, err: err}
	}
}
