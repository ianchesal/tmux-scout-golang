#!/usr/bin/env bash
# tmux-scout picker — fzf popup to browse and jump to agent sessions
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCOUT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# XDG data directory — must precede --generate guard so both code paths get correct value
SCOUT_DATA_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/tmux-scout"
STATUS_FILE="$SCOUT_DATA_DIR/status.json"

# Resolve binary — tmux env is not inherited by fzf's bash $0 --generate self-invocations
_sb=$(tmux show-environment SCOUT_BINARY 2>/dev/null | sed 's/^SCOUT_BINARY=//' || true)
[ -n "$_sb" ] && SCOUT_BINARY="$_sb"
: "${SCOUT_BINARY:=$SCOUT_DIR/bin/tmux-scout}"

# --generate: called by fzf ctrl-r reload
if [ "${1:-}" = "--generate" ]; then
  "$SCOUT_BINARY" picker "$STATUS_FILE" "${2:-}"
  exit 0
fi

if [ ! -f "$STATUS_FILE" ]; then
  tmux display-message "No Claude sessions found. Start a Claude instance first."
  exit 0
fi

CURRENT_PANE=$(tmux display-message -p '#{pane_id}' 2>/dev/null || true)
PICKER_WIDTH=$(tmux show-option -gqv @scout_picker_width 2>/dev/null)
PICKER_HEIGHT=$(tmux show-option -gqv @scout_picker_height 2>/dev/null)
: "${PICKER_WIDTH:=85}"
: "${PICKER_HEIGHT:=75}"
RELOAD_CMD="bash $(printf '%q' "$0") --generate $(printf '%q' "$CURRENT_PANE")"
AUTO_FLAG="/tmp/tmux-scout-auto-$$"
LISTEN_PORT=$((10000 + RANDOM % 50000))

# Auto-refresh on by default
touch "$AUTO_FLAG"

# Background auto-refresh daemon: polls flag every 2s, sends reload via fzf --listen
(
  trap 'exit 0' TERM
  while true; do
    sleep 2 &
    wait $! || exit 0
    [ -f "$AUTO_FLAG" ] || continue
    T=$(date +%H:%M:%S)
    curl -sS -XPOST "localhost:$LISTEN_PORT" -d "reload($RELOAD_CMD)+change-border-label( tmux-scout · auto-refresh $T )" 2>/dev/null || break
  done
) &
AUTO_PID=$!

selected=$("$SCOUT_BINARY" picker "$STATUS_FILE" "$CURRENT_PANE" | fzf \
  --listen=$LISTEN_PORT \
  --tmux "center,${PICKER_WIDTH}%,${PICKER_HEIGHT}%,border-native" \
  --ansi \
  --no-mouse \
  --prompt='> ' \
  --color='border:bright-cyan,label:bright-white' \
  --delimiter='\t' \
  --with-nth=2 \
  --header=$'\nEnter: jump | Ctrl-R: refresh | Ctrl-T: auto-refresh | Esc: cancel' \
  --header-lines=1 \
  --bind="ctrl-r:reload($RELOAD_CMD)" \
  --bind="ctrl-t:execute-silent(if [ -f $AUTO_FLAG ]; then rm -f $AUTO_FLAG; else touch $AUTO_FLAG; fi)+reload($RELOAD_CMD)+transform:if [ -f $AUTO_FLAG ]; then printf \"change-border-label( tmux-scout · auto-refresh \$(date +%H:%M:%S) )\"; else printf 'change-border-label( tmux-scout )'; fi" \
  --preview="$SCOUT_BINARY picker preview {1} $(printf '%q' "$STATUS_FILE") 2>/dev/null" \
  --preview-window=right:50%:wrap:border-left \
  --preview-label=" pane preview " \
  --layout=reverse-list \
  --border=rounded \
  --border-label=" tmux-scout · auto-refresh " \
  --border-label-pos=3 \
  --highlight-line \
  --info=inline-right \
  --separator="─" \
  --pointer="▶" \
  --no-sort \
  --cycle \
  || true)

kill $AUTO_PID 2>/dev/null; wait $AUTO_PID 2>/dev/null
rm -f "$AUTO_FLAG"
[ -z "$selected" ] && exit 0

pane_id=$(echo "$selected" | cut -f1)

if [ "$pane_id" = "UNBOUND" ]; then
  tmux display-popup -w 64 -h 16 -T " tmux-scout " -E bash -c '
printf "\n"
printf "   ⚠  Cannot jump to this session\n"
printf "\n"
printf "   Codex'\''s hook mechanism only fires after the first\n"
printf "   turn completes (agent-turn-complete), so before that\n"
printf "   we have no way to know which pane it'\''s running in.\n"
printf "\n"
printf "   This session was discovered from Codex'\''s log files,\n"
printf "   but the pane link is not yet established.\n"
printf "\n"
printf "   \033[1mWait for Codex to finish its first response,\n"
printf "   then refresh the picker.\033[0m\n"
printf "\n"
printf "   \033[2mPress any key to close\033[0m\n"
read -rsn1
'
  exit 0
fi

# Jump to the pane
target=$(tmux display-message -p -t "$pane_id" '#{session_name}:#{window_index}' 2>/dev/null) || exit 0
tmux switch-client -t "$target" 2>/dev/null || tmux select-window -t "$target"
tmux select-pane -t "$pane_id"
