#!/usr/bin/env bash
# Opens a stacked pane with Claude, or focuses it if one with the same cwd already exists.
# Usage: open-or-focus.sh <name> <cwd>
set -euo pipefail

NAME="${1:?Usage: $0 <name> <cwd>}"
CWD="${2:?Usage: $0 <name> <cwd>}"

PANES_JSON=$(zellij action list-panes --all --json 2>/dev/null)

EXISTING_ID=$(echo "$PANES_JSON" | jq -r --arg cwd "$CWD" '
  .[] | select(.terminal_command != null and .pane_cwd == $cwd) | "terminal_\(.id)"
' 2>/dev/null | head -1 || true)

if [ -n "$EXISTING_ID" ]; then
    zellij action focus-pane-id "$EXISTING_ID"
    exit 0
fi

PANE_COUNT=$(echo "$PANES_JSON" | jq '[.[] | select(.is_selectable == true and .is_plugin != true)] | length' 2>/dev/null || echo "0")

if [ "$PANE_COUNT" -gt 1 ]; then
    zellij action move-focus right
    zellij action new-pane --stacked --close-on-exit --name "" --cwd "$CWD" -- claude --name "$NAME"
else
    zellij action new-pane --direction right --close-on-exit --name "" --cwd "$CWD" -- claude --name "$NAME"
fi
