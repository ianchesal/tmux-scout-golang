#!/usr/bin/env bash
# Short wrapper for use in tmux status-right:
#   set -g status-right '#(/path/to/tmux-scout/scripts/status-widget.sh)'
#
# Note: this script is invoked via #(...) format string — tmux does not inject
# its environment vars in that context, so we query SCOUT_BINARY explicitly.
SCOUT_BINARY=$(tmux show-environment SCOUT_BINARY 2>/dev/null | sed 's/^SCOUT_BINARY=//')
if [ -z "$SCOUT_BINARY" ] || [ ! -x "$SCOUT_BINARY" ]; then
    SCOUT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
    SCOUT_BINARY="$SCOUT_DIR/bin/tmux-scout"
fi
exec "$SCOUT_BINARY" status-bar
