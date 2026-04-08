package tui

import "github.com/diegoleme/gh-wt/internal/listutil"

type entriesLoadedMsg struct {
	entries []listutil.Entry
	err     error
}

type commandFinishedMsg struct {
	label   string
	err     error
	refresh bool
}

type commandOutputMsg struct {
	line string
}

type statusMsg string
