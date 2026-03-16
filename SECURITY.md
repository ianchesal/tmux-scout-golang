# Security Policy

## Supported Versions

This project has not yet made a formal release. Security fixes are applied to the `main` branch. Once versioned releases exist, only the latest release will receive security fixes.

## Reporting a Vulnerability

Please **do not** open a public GitHub issue for security vulnerabilities.

Use [GitHub's private vulnerability reporting](https://github.com/ianchesal/tmux-scout-golang/security/advisories/new) to report issues confidentially. I'll acknowledge within a few days and aim to resolve confirmed vulnerabilities within two weeks.

## Scope

This plugin runs entirely on your local machine with your own user permissions. It reads and writes files under directories you control (`~/.tmux-scout/`, `~/.claude/`, `~/.codex/`). Access to those directories is **by design**, not a vulnerability.

Things that are in scope:

- The plugin executing unintended commands or code
- The plugin exfiltrating data outside your local machine
- A malicious tmux session or hook payload causing unintended behavior beyond the current user's permissions

Things that are **not** in scope:

- The plugin reading files in `~/.claude/` or `~/.codex/` (it is supposed to do this)
- Vulnerabilities in tmux, Claude Code, or Codex themselves
- Issues only exploitable by someone who already has shell access as your user
