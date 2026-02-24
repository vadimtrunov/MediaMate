package stack

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// wizardStep enumerates the stages of the setup wizard.
type wizardStep int

const (
	stepComponents wizardStep = iota
	stepPaths
	stepConfirm
)

// Key constants for bubbletea key handling.
const keyEnter = "enter"

// Path indices for step 2.
const (
	pathMedia     = 0
	pathDownloads = 1
	pathConfig    = 2
	pathOutput    = 3
	pathCount     = 4
)

// pathLabels maps path indices to their display labels.
var pathLabels = [pathCount]string{
	"Media directory",
	"Downloads directory",
	"Config directory",
	"Output directory",
}

// Lipgloss styles used by the wizard.
var (
	wizTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("5"))

	wizSubtitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12"))

	wizSelected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	wizUnselected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	wizLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")).
			Bold(true)

	wizValue = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	wizHelp = lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	wizHighlight = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11"))

	wizCursor = lipgloss.NewStyle().
			Foreground(lipgloss.Color("5")).
			Bold(true)
)

// WizardModel is the Bubble Tea model for the interactive stack setup wizard.
// It walks the user through component selection, path configuration, and a
// final confirmation before producing a Config.
type WizardModel struct {
	// State
	step       wizardStep          // current wizard step
	categories []ComponentCategory // categories from DefaultCategories()
	catIndex   int                 // current category in step 1
	cursor     int                 // cursor position within current list

	// Selections for step 1: [catIndex][optionIndex] = selected
	selections map[int]map[int]bool

	// Text input for step 2
	textinput textinput.Model
	pathIndex int               // which path is being edited (0..3)
	paths     [pathCount]string // media, downloads, config, output

	// Result
	config  Config
	done    bool
	aborted bool

	// UI dimensions
	width  int
	height int
}

// NewWizardModel creates a WizardModel pre-populated with defaults from
// DefaultCategories() and DefaultConfig().
func NewWizardModel() WizardModel {
	cats := DefaultCategories()
	defaults := DefaultConfig()

	// Build initial selections from category defaults.
	selections := make(map[int]map[int]bool, len(cats))
	for ci, cat := range cats {
		selections[ci] = make(map[int]bool, len(cat.Options))
		for oi, opt := range cat.Options {
			if opt == cat.Default {
				selections[ci][oi] = true
			}
		}
	}

	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 256

	paths := [pathCount]string{
		defaults.MediaDir,
		defaults.DownloadsDir,
		defaults.ConfigDir,
		defaults.OutputDir,
	}
	ti.SetValue(paths[0])
	ti.CursorEnd()

	return WizardModel{
		step:       stepComponents,
		categories: cats,
		catIndex:   0,
		cursor:     0,
		selections: selections,
		textinput:  ti,
		pathIndex:  0,
		paths:      paths,
		config:     defaults,
	}
}

// Init returns the initial command (text input blink).
func (m WizardModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles incoming Bubble Tea messages.
func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textinput.Width = msg.Width - 4
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward to textinput when on the paths step.
	if m.step == stepPaths {
		var cmd tea.Cmd
		m.textinput, cmd = m.textinput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleKey dispatches key events based on the current wizard step.
func (m WizardModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global: ctrl+c always quits.
	if key == "ctrl+c" {
		m.aborted = true
		return m, tea.Quit
	}

	switch m.step {
	case stepComponents:
		return m.updateComponents(key)
	case stepPaths:
		return m.updatePaths(msg)
	case stepConfirm:
		return m.updateConfirm(key)
	}

	return m, nil
}

// ---------- Step 1: Component Selection ----------

func (m WizardModel) updateComponents(key string) (tea.Model, tea.Cmd) {
	cat := m.categories[m.catIndex]
	optCount := len(cat.Options)

	switch key {
	case "q":
		m.aborted = true
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < optCount-1 {
			m.cursor++
		}

	case " ":
		m.toggleOption(cat)

	case keyEnter:
		// Validate required categories before advancing.
		if cat.Required && !m.hasSelection(m.catIndex) {
			return m, nil // block — user must select at least one option
		}
		// Move to next category or to step 2.
		if m.catIndex < len(m.categories)-1 {
			m.catIndex++
			m.cursor = 0
		} else {
			m.step = stepPaths
			m.pathIndex = 0
			m.textinput.SetValue(m.paths[0])
			m.textinput.CursorEnd()
		}
	}

	return m, nil
}

// hasSelection reports whether at least one option is selected in the given category.
func (m *WizardModel) hasSelection(catIndex int) bool {
	for _, selected := range m.selections[catIndex] {
		if selected {
			return true
		}
	}
	return false
}

// toggleOption toggles an option at the current cursor position, respecting
// MultiSelect and radio-button semantics.
func (m *WizardModel) toggleOption(cat ComponentCategory) {
	sel := m.selections[m.catIndex]
	if cat.MultiSelect {
		sel[m.cursor] = !sel[m.cursor]
	} else {
		// Radio-button: clear others and toggle current.
		current := sel[m.cursor]
		for k := range sel {
			delete(sel, k)
		}
		if !current {
			sel[m.cursor] = true
		}
	}
}

// ---------- Step 2: Path Configuration ----------

func (m WizardModel) updatePaths(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "q":
		// Only quit if the text input is empty or if it is the default value.
		// Actually, since text input captures 'q', we do not intercept here.
		// 'q' gets typed into the field instead. We rely on ctrl+c to quit.

	case keyEnter:
		// Confirm current path and move to next.
		m.paths[m.pathIndex] = m.textinput.Value()
		if m.pathIndex < pathCount-1 {
			m.pathIndex++
			m.textinput.SetValue(m.paths[m.pathIndex])
			m.textinput.CursorEnd()
		} else {
			// All paths confirmed; build config and go to step 3.
			m.buildConfig()
			m.step = stepConfirm
		}
		return m, nil

	case "tab", "shift+tab":
		if key == "tab" {
			// Save current value and advance to next path field.
			m.paths[m.pathIndex] = m.textinput.Value()
			m.pathIndex = (m.pathIndex + 1) % pathCount
		} else {
			m.paths[m.pathIndex] = m.textinput.Value()
			m.pathIndex = (m.pathIndex + pathCount - 1) % pathCount
		}
		m.textinput.SetValue(m.paths[m.pathIndex])
		m.textinput.CursorEnd()
		return m, nil

	case "esc":
		// Go back to step 1 (last category).
		m.paths[m.pathIndex] = m.textinput.Value()
		m.step = stepComponents
		m.catIndex = len(m.categories) - 1
		m.cursor = 0
		return m, nil
	}

	// Forward all other keys to the textinput.
	var cmd tea.Cmd
	m.textinput, cmd = m.textinput.Update(msg)
	return m, cmd
}

// ---------- Step 3: Confirmation ----------

func (m WizardModel) updateConfirm(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		m.aborted = true
		return m, tea.Quit
	case "b":
		// Go back to paths step.
		m.step = stepPaths
		m.pathIndex = 0
		m.textinput.SetValue(m.paths[0])
		m.textinput.CursorEnd()
		return m, nil
	case keyEnter:
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

// ---------- Config builder ----------

// buildConfig constructs the final Config from wizard selections.
func (m *WizardModel) buildConfig() {
	var components []string
	var torrentClient string
	var mediaServer string

	for ci, cat := range m.categories {
		sel := m.selections[ci]
		for oi, opt := range cat.Options {
			if sel[oi] {
				components = append(components, opt)

				// Track torrent client.
				if cat.Name == "Torrents" {
					torrentClient = opt
				}
				// Track media server.
				if cat.Name == "Streaming" {
					mediaServer = opt
				}
			}
		}
	}

	// Always include mediamate itself.
	components = append(components, ComponentMediaMate)

	mediaDir := m.paths[pathMedia]

	m.config = Config{
		Components:    components,
		MediaDir:      mediaDir,
		MoviesDir:     filepath.Join(mediaDir, "movies"),
		TVDir:         filepath.Join(mediaDir, "tv"),
		BooksDir:      filepath.Join(mediaDir, "books"),
		DownloadsDir:  m.paths[pathDownloads],
		ConfigDir:     m.paths[pathConfig],
		OutputDir:     m.paths[pathOutput],
		TorrentClient: torrentClient,
		MediaServer:   mediaServer,
	}
}

// ---------- View ----------

// View renders the wizard UI for the current step.
func (m WizardModel) View() string {
	var b strings.Builder

	b.WriteString(wizTitle.Render("MediaMate Stack Setup"))
	b.WriteString("\n\n")

	switch m.step {
	case stepComponents:
		m.viewComponents(&b)
	case stepPaths:
		m.viewPaths(&b)
	case stepConfirm:
		m.viewConfirm(&b)
	}

	return b.String()
}

// renderOptionIndicator returns the styled string for a checkbox/radio option.
func renderOptionIndicator(selected, multiSelect bool, opt string) string {
	if selected {
		if multiSelect {
			return wizSelected.Render("[x] " + opt)
		}
		return wizSelected.Render("(*) " + opt)
	}
	if multiSelect {
		return wizUnselected.Render("[ ] " + opt)
	}
	return wizUnselected.Render("( ) " + opt)
}

func (m WizardModel) viewComponents(b *strings.Builder) {
	b.WriteString(wizSubtitle.Render(fmt.Sprintf("Step 1/3: Select Components (%d/%d)", m.catIndex+1, len(m.categories))))
	b.WriteString("\n\n")

	cat := m.categories[m.catIndex]
	b.WriteString(wizLabel.Render(fmt.Sprintf("Category: %s", cat.Name)))
	b.WriteString(wizHelp.Render(fmt.Sprintf(" -- %s", cat.Description)))
	if cat.Required {
		b.WriteString(wizHighlight.Render(" (required)"))
	}
	b.WriteString("\n\n")

	sel := m.selections[m.catIndex]
	for oi, opt := range cat.Options {
		// Cursor indicator.
		if oi == m.cursor {
			b.WriteString(wizCursor.Render("> "))
		} else {
			b.WriteString("  ")
		}

		// Checkbox or radio indicator.
		b.WriteString(renderOptionIndicator(sel[oi], cat.MultiSelect, opt))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if cat.Required && !m.hasSelection(m.catIndex) {
		b.WriteString(wizHighlight.Render("  ⚠ At least one option must be selected"))
		b.WriteString("\n")
	}
	b.WriteString(wizHelp.Render("  Space: toggle, Enter: next, q: quit"))
}

func (m WizardModel) viewPaths(b *strings.Builder) {
	b.WriteString(wizSubtitle.Render("Step 2/3: Configure Paths"))
	b.WriteString("\n\n")

	// Find the longest label for alignment.
	maxLen := 0
	for _, lbl := range pathLabels {
		if len(lbl) > maxLen {
			maxLen = len(lbl)
		}
	}

	for i := 0; i < pathCount; i++ {
		label := pathLabels[i]
		padding := strings.Repeat(" ", maxLen-len(label))

		if i == m.pathIndex {
			b.WriteString(wizLabel.Render(fmt.Sprintf("%s:%s ", label, padding)))
			b.WriteString(m.textinput.View())
		} else {
			b.WriteString(wizHelp.Render(fmt.Sprintf("%s:%s ", label, padding)))
			b.WriteString(wizValue.Render(m.paths[i]))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(wizHelp.Render("  Enter: confirm, Tab: next field, Esc: back, Ctrl+C: quit"))
}

func (m WizardModel) viewConfirm(b *strings.Builder) {
	b.WriteString(wizSubtitle.Render("Step 3/3: Confirm"))
	b.WriteString("\n\n")

	// Component summary.
	b.WriteString(wizLabel.Render("Components: "))
	b.WriteString(wizValue.Render(strings.Join(m.config.Components, ", ")))
	b.WriteString("\n")

	// Path summary.
	b.WriteString(wizLabel.Render("Media:      "))
	b.WriteString(wizValue.Render(m.config.MediaDir))
	b.WriteString("\n")

	b.WriteString(wizLabel.Render("Downloads:  "))
	b.WriteString(wizValue.Render(m.config.DownloadsDir))
	b.WriteString("\n")

	b.WriteString(wizLabel.Render("Config:     "))
	b.WriteString(wizValue.Render(m.config.ConfigDir))
	b.WriteString("\n")

	b.WriteString(wizLabel.Render("Output:     "))
	b.WriteString(wizValue.Render(m.config.OutputDir))
	b.WriteString("\n")

	b.WriteString("\n")
	b.WriteString(wizHelp.Render("  Enter: generate files, b: go back, q: quit"))
}

// ---------- Public accessors ----------

// Done reports whether the wizard completed successfully.
func (m WizardModel) Done() bool { return m.done }

// Aborted reports whether the user quit the wizard.
func (m WizardModel) Aborted() bool { return m.aborted }

// Config returns the final Config assembled by the wizard. The returned
// value is only meaningful when Done() returns true.
func (m WizardModel) Config() Config { return m.config }
