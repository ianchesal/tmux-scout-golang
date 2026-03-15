#!/usr/bin/env bash
# Short wrapper for use in tmux status-right:
#   set -g status-right '#(/path/to/tmux-scout/scripts/status-widget.sh)'
SCOUT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
exec "$SCOUT_DIR/bin/tmux-scout" status-bar
