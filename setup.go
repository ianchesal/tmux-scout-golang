package main

import (
	"fmt"
	"os"
)

func runSetup(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: tmux-scout setup install|uninstall|status [--claude] [--codex] [--gemini] [--quiet]")
		os.Exit(1)
	}

	cmd := args[0]
	doClaude, doCodex, doGemini, quiet := true, true, true, false
	explicitTarget := false

	for _, a := range args[1:] {
		switch a {
		case "--claude":
			if !explicitTarget {
				doClaude, doCodex, doGemini = true, false, false
				explicitTarget = true
			} else {
				doClaude = true
			}
		case "--codex":
			if !explicitTarget {
				doClaude, doCodex, doGemini = false, true, false
				explicitTarget = true
			} else {
				doCodex = true
			}
		case "--gemini":
			if !explicitTarget {
				doClaude, doCodex, doGemini = false, false, true
				explicitTarget = true
			} else {
				doGemini = true
			}
		case "--quiet":
			quiet = true
		}
	}

	home, _ := os.UserHomeDir()
	scoutDir := defaultScoutDir()
	claudeSettings := home + "/.claude/settings.json"
	codexConfig := home + "/.codex/config.toml"
	geminiSettings := home + "/.gemini/settings.json"
	binPath := binaryPath()

	switch cmd {
	case "install":
		if doClaude {
			result, err := claudeInstall(claudeSettings, binPath)
			if err != nil && !quiet {
				fmt.Fprintf(os.Stderr, "Claude Code: %v\n", err)
			} else if !quiet {
				fmt.Println("Claude Code:", result)
			}
		}
		if doCodex {
			result, err := codexInstall(codexConfig, scoutDir, binPath)
			if err != nil && !quiet {
				fmt.Fprintf(os.Stderr, "Codex: %v\n", err)
			} else if !quiet {
				fmt.Println("Codex:", result)
			}
		}
		if doGemini {
			result, err := geminiInstall(geminiSettings, binPath)
			if err != nil && !quiet {
				fmt.Fprintf(os.Stderr, "Gemini: %v\n", err)
			} else if !quiet {
				fmt.Println("Gemini:", result)
			}
		}

	case "uninstall":
		if doClaude {
			result, err := claudeUninstall(claudeSettings)
			if err != nil && !quiet {
				fmt.Fprintf(os.Stderr, "Claude Code: %v\n", err)
			} else if !quiet {
				fmt.Println("Claude Code:", result)
			}
		}
		if doCodex {
			result, err := codexUninstall(codexConfig, scoutDir)
			if err != nil && !quiet {
				fmt.Fprintf(os.Stderr, "Codex: %v\n", err)
			} else if !quiet {
				fmt.Println("Codex:", result)
			}
		}
		if doGemini {
			result, err := geminiUninstall(geminiSettings)
			if err != nil && !quiet {
				fmt.Fprintf(os.Stderr, "Gemini: %v\n", err)
			} else if !quiet {
				fmt.Println("Gemini:", result)
			}
		}

	case "status":
		if quiet {
			// Used by tmux-scout.tmux at startup — exit 1 if no tools are installed
			anyOK := false
			if doClaude {
				n, _ := claudeStatus(claudeSettings)
				if n >= len(claudeHookEvents) {
					anyOK = true
				}
			}
			if doCodex {
				installed, _ := codexStatus(codexConfig)
				if installed {
					anyOK = true
				}
			}
			if doGemini {
				n, _ := geminiStatus(geminiSettings)
				if n >= len(geminiHookEvents) {
					anyOK = true
				}
			}
			if !anyOK {
				os.Exit(1)
			}
			return
		}
		if doClaude {
			n, _ := claudeStatus(claudeSettings)
			fmt.Printf("Claude Code: %d/%d hooks installed\n", n, len(claudeHookEvents))
		}
		if doCodex {
			installed, available := codexStatus(codexConfig)
			if !available {
				fmt.Println("Codex:       not installed")
			} else if installed {
				fmt.Println("Codex:       hook installed")
			} else {
				fmt.Println("Codex:       hook not installed")
			}
		}
		if doGemini {
			n, _ := geminiStatus(geminiSettings)
			fmt.Printf("Gemini:      %d/%d hooks installed\n", n, len(geminiHookEvents))
		}
		fmt.Println("Binary:     ", binPath)

	default:
		fmt.Fprintf(os.Stderr, "unknown setup command: %s\n", cmd)
		os.Exit(1)
	}
}
