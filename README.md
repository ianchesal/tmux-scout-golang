# tmux-scout-golang

This started out as a Golang rewrite of [tmux-scout](https://github.com/qeesung/tmux-scout). All credit for the genesis of this belongs to [qeesung](https://github.com/qeesung). 

A tmux plugin for monitoring and navigating [Claude Code](https://docs.anthropic.com/en/docs/claude-code) and [Codex](https://github.com/openai/codex) sessions. Provides a real-time fzf picker to jump between agent panes, a status bar widget showing session counts, and crash detection for dead sessions.

## WARNING

This is still a work in progress. As of now, only the Claude Code paths have been tested. The Codex stuff probably works, but is completely untested.

## Features

- **Session picker** — `prefix + O` opens an fzf popup listing all active agent sessions with status tags (`WAIT` / `BUSY` / `DONE` / `IDLE`), project names, prompt titles, and live tool details
- **Pane preview** — right-side preview panel shows the last 40 lines of each session's tmux pane
- **Status bar widget** — displays session counts by status (e.g. `0|1|2`) in tmux's status-right, refreshed every 2 seconds
- **Auto-refresh** — `Ctrl-T` toggles automatic picker reload every 2 seconds
- **Crash detection** — dead processes and stale Codex JSONL files are automatically detected and cleaned up

## Requirements

- [tmux](https://github.com/tmux/tmux) >= 3.2
- [fzf](https://github.com/junegunn/fzf) >= 0.51 (with `--listen` and `--tmux` support)

## Installation

### With [TPM](https://github.com/tmux-plugins/tpm)

Requires Go 1.21+ on your `$PATH` — the plugin compiles itself on first load.

Add to `~/.tmux.conf`:

```bash
set -g @plugin 'ianchesal/tmux-scout-golang'
```

Press `prefix + I` to install. On the next tmux reload, the binary is compiled automatically. A build failure will show as a tmux message with a log path.

### Manual

```bash
git clone https://github.com/ianchesal/tmux-scout-golang.git ~/.tmux/plugins/tmux-scout-golang
```

Add to `~/.tmux.conf`:

```bash
run-shell ~/.tmux/plugins/tmux-scout-golang/tmux-scout-golang.tmux
```

Reload tmux: `tmux source ~/.tmux.conf`

## Building from Source

Requires Go 1.21+. CI tests against Go 1.21, 1.22, 1.23, and the current stable release.

```bash
git clone https://github.com/ianchesal/tmux-scout-golang.git
cd tmux-scout-golang
make build   # outputs bin/tmux-scout
```

To run tests:

```bash
make test
```

## Hook Setup

tmux-scout needs hooks installed in Claude Code and/or Codex to track sessions. Run the setup command after installation:

```bash
# SCOUT_DIR is set automatically when the plugin loads — these commands can be copy-pasted directly
eval "$(tmux show-env -g SCOUT_DIR)" && "$SCOUT_DIR/scripts/setup.sh" install

# Other operations
eval "$(tmux show-env -g SCOUT_DIR)" && "$SCOUT_DIR/scripts/setup.sh" install --claude   # Claude Code only
eval "$(tmux show-env -g SCOUT_DIR)" && "$SCOUT_DIR/scripts/setup.sh" install --codex    # Codex only
eval "$(tmux show-env -g SCOUT_DIR)" && "$SCOUT_DIR/scripts/setup.sh" uninstall          # Remove all hooks
eval "$(tmux show-env -g SCOUT_DIR)" && "$SCOUT_DIR/scripts/setup.sh" status             # Check installation status
```

The installer is **idempotent** — running it multiple times is safe. If you move the repository, re-running install will automatically update hook paths.

### What gets modified

- **Claude Code**: Adds a hook entry to each of the 6 event types in `~/.claude/settings.json`
- **Codex**: Sets the `notify` field in `~/.codex/config.toml` (original notify command is backed up and chained)

## Usage

### Picker

Press `prefix + O` (default) to open the session picker.

| Key | Action |
|---|---|
| `Enter` | Jump to selected session's pane |
| `Ctrl-R` | Refresh session list |
| `Ctrl-T` | Toggle auto-refresh (every 2s) |
| `Esc` | Close picker |

Each line shows:

```
* [ BUSY ] claude  my-project                "implement the login page"  Bash: npm test
```

- `*` — current pane indicator
- `[ WAIT ]` / `[ BUSY ]` / `[ DONE ]` / `[ IDLE ]` — session status
- Agent type (claude / codex)
- Project directory name
- Session title (first prompt)
- Current tool details (for working sessions)

### Status Bar

The status widget is not automatically injected — you need to add it manually. The plugin sets a `SCOUT_DIR` environment variable at load time, so you can use `$SCOUT_DIR` to reference the widget script regardless of install location.

**Without a theme plugin**, add to `~/.tmux.conf`:

```bash
set -g status-right '#($SCOUT_DIR/scripts/status-widget.sh) #S'
set -g status-interval 2
```

**With a theme plugin** (e.g. `minimal-tmux-status`), directly setting `status-right` won't work because the theme overrides it. Use the theme's own option instead:

```bash
# minimal-tmux-status
set -g @minimal-tmux-status-right '#($SCOUT_DIR/scripts/status-widget.sh) #S'
```

The widget shows:

```
W|B|D
```

Where `W` = waiting for attention (red), `B` = busy/working (yellow), `D` = done/completed (green). An optional `I` = idle (blue) appears when idle sessions exist.

## Configuration

### Keybinding

```bash
set -g @scout-key "O"    # default: O (prefix + O)
```

### Status Bar

```bash
set -g @scout-status-format '{W}/{B}/{D}'         # custom separators
set -g @scout-status-format '{W} wait {B} busy'   # with labels
```

Placeholders: `{W}` wait, `{B}` busy, `{D}` done, `{I}` idle.

## Data Storage

Session data is stored in `~/.tmux-scout/`:

```
~/.tmux-scout/
├── status.json                      # Aggregated session index
├── sessions/                        # Per-session JSON files
│   ├── {session-id}.json
│   └── ...
└── codex-original-notify.json       # Backup of original Codex notify command
```

Sessions older than 24 hours are automatically cleaned up.

## Known Issues

* None of the Codex paths have been tested


## See Also

* [qeesung/tmux-scout](https://github.com/qeesung/tmux-scout) -- the genesis for this project came about after they posted this to the r/tmux sub-reddit. I wanted a binary approach to doing what they were doing and took on rewriting it all in Golang. All credit belongs to qeesung for the original idea and implementation here.


## License

[MIT](LICENSE)
