package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/diegoleme/gh-wt/internal/config"
	"github.com/diegoleme/gh-wt/internal/listutil"
)

type appState int

const (
	stateLoading       appState = iota
	stateList
	stateInput
	stateConfirmDelete
	stateRunning
	stateOutput        // showing command output in dialog
	stateOutputDone    // command finished, press key to close
)

type Model struct {
	state   appState
	entries []listutil.Entry
	cursor  int
	scroll  int // viewport scroll offset
	spinner spinner.Model
	width   int
	height  int

	// Input state
	inputPrompt string
	textInput   textinput.Model
	pendingCmd  *config.Keybinding

	// Output dialog state
	outputLines []string
	outputLabel string
	outputDone  bool
	outputErr   error

	// Status bar
	statusMsg string

	// Program reference (for sending messages from goroutines)
	program **tea.Program

	// Config
	cfg         *config.Config
	keybindings []config.Keybinding

	// Help
	help help.Model
}

func NewModel(cfg *config.Config) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#8957e5"))

	ti := textinput.New()
	ti.CharLimit = 200
	ti.Prompt = "> "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#58a6ff"))
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#c9d1d9"))
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#58a6ff"))

	pp := new(*tea.Program)
	return Model{
		state:       stateLoading,
		cfg:         cfg,
		keybindings: cfg.TUI.Keybindings,
		spinner:     s,
		textInput:   ti,
		help:        help.New(),
		program:     pp,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadEntries)
}

func (m *Model) selectedEntry() *listutil.Entry {
	if m.cursor >= 0 && m.cursor < len(m.entries) {
		return &m.entries[m.cursor]
	}
	return nil
}

func (m *Model) clampCursor() {
	if m.cursor >= len(m.entries) {
		m.cursor = max(0, len(m.entries)-1)
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *Model) viewportHeight() int {
	// top margin (1) + bottom padding (1) + status bar blank line (1) + status bar (1) + footer (1) + buffer (1)
	overhead := 6
	if m.height <= overhead {
		return 10
	}
	return m.height - overhead
}

// scrollToCursor adjusts scroll (in lines) to keep the cursor's card visible.
// Each card ≈ 5 lines (3 content + border/margin), section headers ≈ 3 lines.
func (m *Model) scrollToCursor() {
	linePos := 0
	headerLinePos := 0 // line where the section header for this entry starts
	lastSection := ""
	isFirstInSection := false

	for i, e := range m.entries {
		if e.Section != lastSection {
			headerLinePos = linePos
			linePos += 3 // section header
			lastSection = e.Section
			isFirstInSection = true
		} else {
			isFirstInSection = false
		}
		if i == m.cursor {
			break
		}
		linePos += 5 // card height estimate
		isFirstInSection = false
	}

	cardHeight := 5
	vpHeight := m.viewportHeight()

	// When scrolling up: if first in section, show the header too
	scrollTarget := linePos
	if isFirstInSection {
		scrollTarget = headerLinePos
	}

	if scrollTarget < m.scroll {
		m.scroll = scrollTarget
	}
	if linePos+cardHeight > m.scroll+vpHeight {
		m.scroll = linePos + cardHeight - vpHeight
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
}

// visibleRows returns estimated number of visible entry cards (for page up/down).
func (m *Model) visibleRows() int {
	return m.viewportHeight() / 5
}
