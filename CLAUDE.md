# tmux-scout-golang

A tmux plugin for monitoring and navigating Claude Code, Codex, and Gemini CLI sessions. Provides a real-time fzf picker, status bar widget, and crash detection.

## Tech Stack

- **Language:** Go 1.21+, stdlib only — no external dependencies
- **Shell:** Bash (tmux plugin entry point, picker launcher, status widget)

## Build

```bash
make build      # compile to bin/tmux-scout
make test       # go test ./...
make release    # cross-compile for linux/darwin amd64/arm64
make tag        # verify preconditions, run tests, tag, and push to trigger GitHub release
```

## Releasing

`main` is branch-protected — all changes (including version bumps) require a pull request.

To cut a release, use the project Claude Code skill: say "cut a new release" and it will prompt for patch / minor / major, handle the branch + PR, and run `make tag` after the PR merges.

Manually: create a branch, update `.version` to the new `vMAJOR.MINOR.PATCH`, commit, push, open a PR. After merge, pull `main` and run `make tag`.

## Architecture

**Go binary subcommands:**

| Subcommand | Purpose |
|---|---|
| `hook claude` | Process Claude Code hook events (reads env vars, updates session JSON) |
| `hook codex [json]` | Process Codex notify hook events |
| `hook gemini` | Process Gemini CLI hook events (reads env vars, updates session JSON) |
| `setup install\|uninstall\|status [--claude\|--codex\|--gemini]` | Install/remove hooks in Claude Code, Codex, and Gemini CLI configs |
| `picker <status-file> <current-pane>` | Render fzf picker lines |
| `picker preview <pane-id> <status-file>` | Render structured pane preview (metadata header + filtered terminal output) |
| `picker sync` | Sync session state (poll Codex JSONL, detect crashes) |
| `status-bar` | Emit status bar widget string |

**Data storage:** `~/.cache/tmux-scout/` (XDG; respects `$XDG_CACHE_HOME`)
- `status.json` — aggregated session index
- `sessions/{id}.json` — per-session state files

**Go file layout (flat `package main`):**
- `main.go` — CLI dispatch
- `store.go` — types + session read/write
- `hook_claude.go`, `hook_codex.go`, `hook_gemini.go` — hook handlers
- `setup.go`, `setup_claude.go`, `setup_codex.go`, `setup_gemini.go` — installer
- `picker.go`, `picker_sync.go`, `picker_render.go`, `picker_preview.go` — picker logic
- `status_bar.go` — status widget

## Key Constraints

- No external Go dependencies — stdlib only
- Shell scripts (`tmux-scout-golang.tmux`, `scripts/picker/picker.sh`, `scripts/status-widget.sh`) call the Go binary
