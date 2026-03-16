# tmux-scout-golang

A tmux plugin for monitoring and navigating Claude Code and Codex sessions. Provides a real-time fzf picker, status bar widget, and crash detection.

## Tech Stack

- **Language:** Go 1.21+, stdlib only — no external dependencies
- **Shell:** Bash (tmux plugin entry point, picker launcher, status widget)

## Build

```bash
make build      # compile to bin/tmux-scout
make test       # go test ./...
make release    # cross-compile for linux/darwin amd64/arm64
```

## Architecture

**Go binary subcommands:**

| Subcommand | Purpose |
|---|---|
| `hook claude` | Process Claude Code hook events (reads env vars, updates session JSON) |
| `hook codex [json]` | Process Codex notify hook events |
| `setup install\|uninstall\|status [--claude\|--codex]` | Install/remove hooks in Claude Code and Codex configs |
| `picker <status-file> <current-pane>` | Render fzf picker lines |
| `picker sync` | Sync session state (poll Codex JSONL, detect crashes) |
| `status-bar` | Emit status bar widget string |

**Data storage:** `~/.tmux-scout/`
- `status.json` — aggregated session index
- `sessions/{id}.json` — per-session state files

**Go file layout (flat `package main`):**
- `main.go` — CLI dispatch
- `store.go` — types + session read/write
- `hook_claude.go`, `hook_codex.go` — hook handlers
- `setup.go`, `setup_claude.go`, `setup_codex.go` — installer
- `picker.go`, `picker_sync.go`, `picker_render.go` — picker logic
- `status_bar.go` — status widget

## Key Constraints

- No external Go dependencies — stdlib only
- Shell scripts (`tmux-scout-golang.tmux`, `scripts/picker/picker.sh`, `scripts/status-widget.sh`) call the Go binary
