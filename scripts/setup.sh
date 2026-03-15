#!/usr/bin/env bash
# tmux-scout setup wrapper
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCOUT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
exec "$SCOUT_DIR/bin/tmux-scout" setup "$@"
