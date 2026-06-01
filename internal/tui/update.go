package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/diegoleme/gh-wt/internal/config"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case entriesLoadedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %s", msg.err)
			m.state = stateList
			return m, nil
		}
		m.entries = msg.entries
		m.state = stateList
		m.clampCursor()
		m.scrollToCursor()
		m.statusMsg = ""
		return m, nil

	case commandOutputMsg:
		m.outputLines = append(m.outputLines, msg.line)
		return m, nil

	case commandFinishedMsg:
		if msg.entryKey != "" {
			delete(m.processing, msg.entryKey)
		}
		if m.state == stateOutput {
			// Output dialog mode — show done state
			m.outputDone = true
			m.outputErr = msg.err
			m.state = stateOutputDone
			return m, nil
		}
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("✗ %s: %s", msg.label, msg.err)
		} else {
			m.statusMsg = fmt.Sprintf("✓ %s", msg.label)
		}
		if msg.refresh {
			m.state = stateLoading
			return m, tea.Batch(m.spinner.Tick, loadEntries)
		}
		// Fire-and-forget commands run in the background without changing
		// state, and may finish while the list is reloading or another dialog
		// is open. Leave the current state untouched so we don't yank the user
		// out of it; only the status bar and the card's spinner update.
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward messages to textinput when in input state (for cursor blink)
	if m.state == stateInput {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys (always active)
	if key.Matches(msg, builtins.Quit) {
		return m, tea.Quit
	}

	switch m.state {
	case stateList:
		return m.handleListKey(msg)
	case stateInput:
		return m.handleInputKey(msg)
	case stateConfirmDelete:
		return m.handleConfirmKey(msg)
	case stateOutputDone:
		// Any key closes the output dialog and refreshes
		m.state = stateLoading
		m.outputLines = nil
		m.outputDone = false
		m.outputErr = nil
		return m, tea.Batch(m.spinner.Tick, loadEntries)
	}

	return m, nil
}

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, builtins.Up):
		if m.cursor > 0 {
			m.cursor--
			m.scrollToCursor()
		}
		return m, nil

	case key.Matches(msg, builtins.Down):
		if m.cursor < len(m.entries)-1 {
			m.cursor++
			m.scrollToCursor()
		}
		return m, nil

	case key.Matches(msg, builtins.PageUp):
		half := m.visibleRows() / 2
		if half < 1 {
			half = 1
		}
		m.cursor -= half
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.scrollToCursor()
		return m, nil

	case key.Matches(msg, builtins.PageDown):
		half := m.visibleRows() / 2
		if half < 1 {
			half = 1
		}
		m.cursor += half
		if m.cursor >= len(m.entries) {
			m.cursor = len(m.entries) - 1
		}
		m.scrollToCursor()
		return m, nil

	case key.Matches(msg, builtins.Top):
		m.cursor = 0
		m.scrollToCursor()
		return m, nil

	case key.Matches(msg, builtins.Bottom):
		m.cursor = len(m.entries) - 1
		m.scrollToCursor()
		return m, nil

	case key.Matches(msg, builtins.Refresh):
		m.state = stateLoading
		m.statusMsg = "Refreshing..."
		return m, tea.Batch(m.spinner.Tick, loadEntries)
	}

	// Check configurable keybindings
	for i := range m.keybindings {
		kb := &m.keybindings[i]
		if msg.String() == kb.Key {
			return m.executeKeybinding(kb)
		}
	}

	return m, nil
}

func (m Model) executeKeybinding(kb *config.Keybinding) (tea.Model, tea.Cmd) {
	entry := m.selectedEntry()

	// Block any command on a card that already has one in flight, so the user
	// can't, e.g., fire "open" twice on the same issue. Other cards stay free.
	if key := entryKey(entry); key != "" {
		if running, ok := m.processing[key]; ok {
			m.statusMsg = fmt.Sprintf("⏳ %s in progress on this issue", running)
			return m, nil
		}
	}

	// Check requires
	if !checkRequires(*kb, entry) {
		m.statusMsg = fmt.Sprintf("✗ %s: requires %s", kb.Label, strings.Join(kb.Requires, ", "))
		return m, nil
	}

	// If needs input, transition to input state
	if kb.Input != "" {
		m.state = stateInput
		m.inputPrompt = kb.Input
		m.textInput.SetValue("")
		m.textInput.Placeholder = kb.Input
		m.textInput.Focus()
		m.pendingCmd = kb
		return m, m.textInput.Cursor.BlinkCmd()
	}

	// If needs confirmation, transition to confirm state
	if kb.Confirm {
		m.state = stateConfirmDelete
		m.pendingCmd = kb
		return m, nil
	}

	// Execute directly
	return m.runCommand(kb, "")
}

func (m Model) runCommand(kb *config.Keybinding, input string) (tea.Model, tea.Cmd) {
	entry := m.selectedEntry()

	resolved, err := resolveCommand(kb.Command, entry, input)
	if err != nil {
		m.statusMsg = fmt.Sprintf("✗ %s: %s", kb.Label, err)
		return m, nil
	}

	if kb.Output && m.program != nil && *m.program != nil {
		m.state = stateOutput
		m.outputLines = nil
		m.outputLabel = kb.Label
		m.outputDone = false
		m.outputErr = nil
		return m, tea.Batch(
			m.spinner.Tick,
			execWithOutput(*m.program, kb.Label, resolved, true),
		)
	}

	if kb.Interactive {
		m.state = stateRunning
		return m, execInteractive(kb.Label, resolved)
	}

	// Default: fire and forget (no output capture, no terminal handoff).
	// Mark the card as processing so it shows a spinner and rejects further
	// commands until this one finishes; the key lets it run concurrently with
	// open commands on other cards.
	key := entryKey(entry)
	if key != "" {
		m.processing[key] = kb.Label
	}
	return m, execFireAndForget(kb.Label, key, resolved)
}

func (m Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.state = stateList
		m.textInput.Blur()
		m.pendingCmd = nil
		return m, nil

	case tea.KeyEnter:
		value := m.textInput.Value()
		if m.pendingCmd != nil && value != "" {
			kb := m.pendingCmd
			m.pendingCmd = nil
			m.textInput.Blur()
			return m.runCommand(kb, value)
		}
		return m, nil

	default:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.pendingCmd != nil {
			kb := m.pendingCmd
			m.pendingCmd = nil
			return m.runCommand(kb, "")
		}
		return m, nil

	case "n", "N", "esc":
		m.state = stateList
		m.pendingCmd = nil
		return m, nil
	}

	return m, nil
}
