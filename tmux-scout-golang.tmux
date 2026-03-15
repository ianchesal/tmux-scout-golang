#!/usr/bin/env bash
CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Build binary if missing (e.g. after TPM install)
if [ ! -f "$CURRENT_DIR/bin/tmux-scout" ]; then
  if command -v go >/dev/null 2>&1; then
    tmux display-message "tmux-scout: building binary (requires Go)..."
    if ! (cd "$CURRENT_DIR" && go build -o bin/tmux-scout . 2>/tmp/tmux-scout-build.log); then
      tmux display-message "tmux-scout: build failed — see /tmp/tmux-scout-build.log"
      return 1 2>/dev/null || exit 1
    fi
  else
    tmux display-message "tmux-scout: binary missing and Go not found. Run: cd $CURRENT_DIR && make build"
    return 1 2>/dev/null || exit 1
  fi
fi

key=$(tmux show-option -gqv "@scout-key")
[ -z "$key" ] && key="O"
tmux set-environment -g SCOUT_DIR "$CURRENT_DIR"
tmux bind-key "$key" run-shell -b "$CURRENT_DIR/scripts/picker/picker.sh"
tmux run-shell -b "\"$CURRENT_DIR/bin/tmux-scout\" setup status --quiet 2>/dev/null || tmux display-message 'tmux-scout: hooks not installed. Run: $CURRENT_DIR/scripts/setup.sh install'"

# Status bar widget — users add #($SCOUT_DIR/scripts/status-widget.sh) to their status-right config
