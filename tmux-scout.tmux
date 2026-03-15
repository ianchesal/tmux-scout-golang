#!/usr/bin/env bash
CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
key=$(tmux show-option -gqv "@scout-key")
[ -z "$key" ] && key="O"
tmux set-environment -g SCOUT_DIR "$CURRENT_DIR"
tmux bind-key "$key" run-shell -b "$CURRENT_DIR/scripts/picker/picker.sh"
tmux run-shell -b "\"$CURRENT_DIR/bin/tmux-scout\" setup status --quiet 2>/dev/null || tmux display-message 'tmux-scout: hooks not installed. Run: $CURRENT_DIR/scripts/setup.sh install'"

# Status bar widget — users add #($SCOUT_DIR/scripts/status-widget.sh) to their status-right config
