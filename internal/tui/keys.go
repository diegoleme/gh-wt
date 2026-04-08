package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/diegoleme/gh-wt/internal/config"
	"github.com/diegoleme/gh-wt/internal/listutil"
)

// Built-in navigation keys (always available, not configurable)
type builtinKeys struct {
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Top      key.Binding
	Bottom   key.Binding
	Refresh  key.Binding
	Quit     key.Binding
}

var builtins = builtinKeys{
	Up:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
	Down:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
	PageUp:   key.NewBinding(key.WithKeys("ctrl+u", "pgup"), key.WithHelp("ctrl+u", "page up")),
	PageDown: key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("ctrl+d", "page down")),
	Top:      key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
	Bottom:   key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
	Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// checkRequires returns true if all of the keybinding's requires conditions are met.
func checkRequires(kb config.Keybinding, entry *listutil.Entry, cfg *config.Config) bool {
	if len(kb.Requires) == 0 {
		return true
	}
	if entry == nil {
		return false
	}
	for _, req := range kb.Requires {
		if !checkRequire(req, entry, cfg) {
			return false
		}
	}
	return true
}

func checkRequire(req string, entry *listutil.Entry, cfg *config.Config) bool {
	switch req {
	case "pr":
		return entry.PRNumber > 0
	case "issue":
		return entry.IssueNumber > 0
	case "open_command":
		return cfg.Open.Command != ""
	case "branch":
		return entry.HasWorktree && entry.Branch != ""
	case "worktree":
		return entry.HasWorktree
	default:
		return true
	}
}

// visibleBindings returns keybindings that are visible for the current entry.
func visibleBindings(bindings []config.Keybinding, entry *listutil.Entry, cfg *config.Config) []config.Keybinding {
	var visible []config.Keybinding
	for _, kb := range bindings {
		if checkRequires(kb, entry, cfg) {
			visible = append(visible, kb)
		}
	}
	return visible
}
