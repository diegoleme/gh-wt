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
gh wt open <issue>     # Open worktree in a new terminal session
gh wt status           # Show current worktree state
```

## Configuration

Two-level config: user preferences + repo settings.

### User config (`~/.config/gh-wt/config.yml`)

```yaml
open:
  command: "zellij action new-pane --stacked --cwd {{.WorktreePath}} -- claude"
  on_start: true

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
| `requires` | []string | Show only when conditions are met: `pr`, `issue`, `worktree`, `open_command` |
| `output` | bool | Show command output in a floating dialog |
| `interactive` | bool | Hand over terminal to the command |

## Template variables

| Variable | Description |
|---|---|
| `{{.Branch}}` | Branch name |
| `{{.Path}}` | Worktree absolute path |
| `{{.IssueNumber}}` | Issue number |
| `{{.IssueTitle}}` | Issue title |
| `{{.PRNumber}}` | PR number |
| `{{.Input}}` | User input from `input` prompt |
| `{{.OpenCommand}}` | Resolved open.command from config |

## Tech stack

Go, Cobra, Viper, Bubble Tea, Lipgloss, go-gh
