# gh-wt

A GitHub CLI extension for worktree-driven development. One issue = one branch = one worktree = one PR.

## Install

```bash
gh extension install diegoleme/gh-wt
```

## Usage

### TUI

```bash
gh wt
```

Opens an interactive dashboard with all your issues organized in sections:

- **Done** — closed issues with worktrees still active
- **In Progress** — open issues with worktrees
- **Todo** — open issues without worktrees

### CLI

```bash
gh wt start <issue>    # Create branch + worktree + link issue + run hooks
gh wt list             # List worktrees with PR/CI/review status
gh wt prune            # Remove worktrees for merged/closed branches
gh wt prune <number>   # Remove worktree for a specific issue or PR
gh wt status           # Show current worktree state
```

## Configuration

Two-level config: user preferences + repo settings.

### User config (`~/.config/gh-wt/config.yml`)

```yaml
tui:
  keybindings:
    - key: s
      label: "start"
      command: "gh wt start {{.IssueNumber}}"
      output: true

    - key: d
      label: "delete"
      confirm: true
      requires:
        - worktree
      command: "git worktree remove '{{.Path}}' --force && git branch -D '{{.Branch}}'"
      output: true

    - key: w
      label: "open web"
      requires:
        - issue
      command: "gh issue view {{.IssueNumber}} --web"
```

### Repo config (`.gh-wt.yml`)

```yaml
worktree:
  path: "../{{.RepoName}}-wt/{{.Branch}}"
  copy-ignored:
    enabled: true
    exclude: ["node_modules/", ".turbo/"]

hooks:
  post-start:
    - run: "npm install"
```

## Keybinding options

| Field | Type | Description |
|---|---|---|
| `key` | string | Keyboard shortcut |
| `label` | string | Label shown in footer |
| `command` | string | Shell command with template variables |
| `input` | string | Prompt for user input before executing |
| `confirm` | bool | Ask for confirmation (y/N) |
| `requires` | []string | Show only when conditions are met: `pr`, `issue`, `worktree` |
| `output` | bool | Show command output in a floating dialog |
| `interactive` | bool | Hand over terminal to the command |

## Template variables

| Variable | Type | Description |
|---|---|---|
| `{{.Branch}}` | string | Branch name |
| `{{.Path}}` | string | Worktree absolute path |
| `{{.IssueTitle}}` | string | Issue title |
| `{{.Input}}` | string | User input from `input` prompt |
| `{{.IssueNumber}}` | int | Issue number |
| `{{.PRNumber}}` | int | PR number |

Commands are executed via `sh -c`, so values would normally need shell quoting.
gh-wt does this for you: **string variables are auto-escaped** and rendered
already wrapped in single quotes, so `{{.IssueTitle}}` is always one shell
argument, even if the title contains spaces, apostrophes, `$`, backticks, etc.
Integer variables are interpolated raw so they work in arithmetic comparisons.

```yaml
# Always safe — no manual quoting needed:
command: "open-pane.sh '#'{{.IssueNumber}}-{{.IssueTitle}} {{.Path}}"
command: "git worktree remove {{.Path}} --force && git branch -D {{.Branch}}"
command: "if [ {{.PRNumber}} -gt 0 ]; then gh pr view {{.PRNumber}}; fi"

# Don't add your own quotes around string variables — they're already quoted:
command: "echo '{{.Branch}}'"   # renders as: echo ''main''  (works, but ugly)

# A leading literal '#' must be quoted (in YAML or shell) so sh doesn't read it
# as the start of a comment — this is a shell rule, unrelated to gh-wt.
```

## Tech stack

Go, Cobra, Viper, Bubble Tea, Lipgloss, go-gh
