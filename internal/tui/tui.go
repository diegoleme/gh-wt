package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/diegoleme/gh-wt/internal/config"
)

// Run starts the TUI.
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	m := NewModel(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	*m.program = p

	_, err = p.Run()
	return err
}
