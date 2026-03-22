package main

import (
	"fmt"
	"os"
	"path/filepath"
)

var version = "dev" // overridden at build time via -ldflags "-X main.version=..."

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "hook":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: tmux-scout hook claude|codex|gemini")
			os.Exit(1)
		}
		switch os.Args[2] {
		case "claude":
			runHookClaude()
		case "codex":
			jsonArg := ""
			if len(os.Args) > 3 {
				jsonArg = os.Args[3]
			}
			runHookCodex(jsonArg)
		case "gemini":
			runHookGemini()
		default:
			fmt.Fprintf(os.Stderr, "unknown hook type: %s\n", os.Args[2])
			os.Exit(1)
		}
	case "setup":
		runSetup(os.Args[2:])
	case "picker":
		if len(os.Args) >= 3 && os.Args[2] == "preview" {
			if len(os.Args) < 5 {
				fmt.Fprintln(os.Stderr, "usage: tmux-scout picker preview <pane-id> <status-file>")
				os.Exit(1)
			}
			runPickerPreview(os.Args[3], os.Args[4])
		} else {
			if len(os.Args) < 4 {
				fmt.Fprintln(os.Stderr, "usage: tmux-scout picker <status-file> <current-pane>")
				os.Exit(1)
			}
			runPicker(os.Args[2], os.Args[3])
		}
	case "status-bar":
		runStatusBar()
	case "migrate":
		runMigrate()
	case "--version", "-version":
		fmt.Println(version)
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`usage: tmux-scout <command>

Commands:
  hook claude              Receive Claude Code hook event from stdin
  hook codex [json]        Receive Codex agent-turn-complete event
  hook gemini              Receive Gemini CLI hook event from stdin
  setup install|uninstall|status [--claude] [--codex] [--gemini] [--quiet]
                           Manage tmux-scout hooks
  picker <status-file> <current-pane>
                           Output fzf-ready session list
  picker preview <pane-id> <status-file>
                           Output structured pane preview
  status-bar               Output tmux status-right widget
  migrate                  Migrate data from ~/.tmux-scout to XDG path
`)
}

// binaryPath returns the absolute resolved path to this executable.
func binaryPath() string {
	exe, err := os.Executable()
	if err != nil {
		return os.Args[0]
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe
	}
	return resolved
}
