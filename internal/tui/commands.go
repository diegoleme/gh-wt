package tui

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/diegoleme/gh-wt/internal/listutil"
)

func loadEntries() tea.Msg {
	entries, err := listutil.BuildIssueEntries()
	return entriesLoadedMsg{entries: entries, err: err}
}

// entryKey returns a stable identifier for an entry, used to track per-card
// processing state. It must survive a reload of the entries slice, so it is
// keyed by issue number (or branch/path for worktrees without an issue)
// rather than by slice index.
func entryKey(e *listutil.Entry) string {
	if e == nil {
		return ""
	}
	switch {
	case e.IssueNumber > 0:
		return fmt.Sprintf("issue:%d", e.IssueNumber)
	case e.Branch != "":
		return "branch:" + e.Branch
	default:
		return "path:" + e.Path
	}
}

// shellQuote wraps s in single quotes safe for POSIX shells, escaping any
// embedded single quotes via the standard '\” trick. Used internally by
// shellSafe so template authors don't have to think about it.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// shellSafe is a string that renders itself shell-quoted when formatted by
// text/template (which uses fmt's %v / %s verbs and honors fmt.Stringer).
// String values exposed to user templates are wrapped in this type so that
// {{.X}} substitutions are always safe to feed to `sh -c`, regardless of
// whether the value contains spaces, quotes, or other shell metacharacters.
type shellSafe string

func (s shellSafe) String() string { return shellQuote(string(s)) }

// resolveCommand resolves template variables in a command string. String
// values are auto-quoted; numeric values are interpolated raw.
func resolveCommand(cmdTemplate string, entry *listutil.Entry, input string) (string, error) {
	tmpl, err := template.New("cmd").Parse(cmdTemplate)
	if err != nil {
		return "", err
	}

	data := map[string]interface{}{
		"Branch":      shellSafe(""),
		"Path":        shellSafe(""),
		"IssueNumber": 0,
		"IssueTitle":  shellSafe(""),
		"PRNumber":    0,
		"Input":       shellSafe(input),
	}

	if entry != nil {
		absPath, _ := filepath.Abs(entry.Path)
		data["Branch"] = shellSafe(entry.Branch)
		data["Path"] = shellSafe(absPath)
		data["IssueNumber"] = entry.IssueNumber
		data["IssueTitle"] = shellSafe(entry.IssueTitle)
		data["PRNumber"] = entry.PRNumber
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// execCommand runs a shell command in the background and returns a message.
func execCommand(label, command string, refresh bool) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("sh", "-c", command)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return commandFinishedMsg{
				label:   label,
				err:     fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output))),
				refresh: refresh,
			}
		}
		return commandFinishedMsg{label: label, refresh: refresh}
	}
}

// execFireAndForget runs a command that inherits the process environment
// but discards output. Used for commands like `zellij action` that need
// access to the terminal session context.
func execFireAndForget(label, entryKey, command string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("sh", "-c", command)
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = nil
		err := cmd.Run()
		return commandFinishedMsg{label: label, entryKey: entryKey, err: err, refresh: false}
	}
}

// execInteractive runs a command interactively, giving it the terminal.
// Always refreshes after completion since interactive commands typically change state.
func execInteractive(label, command string) tea.Cmd {
	c := exec.Command("sh", "-c", command)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return commandFinishedMsg{label: label, err: err, refresh: true}
	})
}

// execWithOutput runs a command and streams its output line by line via messages.
func execWithOutput(p *tea.Program, label, command string, refresh bool) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("sh", "-c", command)

		// Combine stdout and stderr into one pipe
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return commandFinishedMsg{label: label, err: err, refresh: refresh}
		}
		cmd.Stderr = cmd.Stdout

		if err := cmd.Start(); err != nil {
			return commandFinishedMsg{label: label, err: err, refresh: refresh}
		}

		// Stream output line by line
		scanner := bufio.NewScanner(io.Reader(stdout))
		for scanner.Scan() {
			line := scanner.Text()
			p.Send(commandOutputMsg{line: line})
		}

		err = cmd.Wait()
		return commandFinishedMsg{label: label, err: err, refresh: refresh}
	}
}
