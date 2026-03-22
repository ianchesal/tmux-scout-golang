package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

// boxDrawingChars is the set of characters treated as UI chrome.
var boxDrawingChars = map[rune]bool{
	'─': true, '│': true, '╭': true, '╰': true, '╯': true, '╮': true,
	'▶': true, '◀': true, '▲': true, '▼': true, '═': true,
	'┌': true, '┐': true, '└': true, '┘': true,
	'├': true, '┤': true, '┬': true, '┴': true, '┼': true,
}

// isChromeLine returns true if ≥50% of non-space runes are box-drawing characters.
func isChromeLine(line string) bool {
	var total, boxCount int
	for _, r := range line {
		if unicode.IsSpace(r) {
			continue
		}
		total++
		if boxDrawingChars[r] {
			boxCount++
		}
	}
	if total == 0 {
		return false
	}
	return boxCount*2 >= total
}

// filterChromeLines removes chrome lines and collapses runs of blank lines.
func filterChromeLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	lastBlank := false
	for _, l := range lines {
		if isChromeLine(l) {
			continue
		}
		blank := strings.TrimSpace(l) == ""
		if blank && lastBlank {
			continue // collapse blank run
		}
		out = append(out, l)
		lastBlank = blank
	}
	return out
}

// lastN returns the last n elements of s, or all of s if len(s) <= n.
func lastN(s []string, n int) []string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

// PreviewPane prints a structured preview for the given tmux pane.
// statusFile is the full path to status.json; filepath.Dir is used to get the
// scout directory that ReadStatusFile expects.
func PreviewPane(paneID, statusFile string) {
	sf, err := ReadStatusFile(filepath.Dir(statusFile))
	if err == nil {
		// Find the session for this pane
		for _, s := range sf.Sessions {
			if s.TmuxPane != nil && *s.TmuxPane == paneID {
				printPreviewHeader(s)
				break
			}
		}
	}
	printPreviewCapture(paneID)
}

func printPreviewHeader(s Session) {
	agentType := s.AgentType
	if agentType == "" {
		agentType = "unknown"
	}
	status := s.Status
	if s.NeedsAttention != nil && *s.NeedsAttention != "" {
		status = fmt.Sprintf("%s (waiting: %s)", s.Status, *s.NeedsAttention)
	}
	dir := s.WorkingDirectory
	if dir == "" {
		dir = "?"
	}
	fmt.Printf("Agent:   %s\n", agentType)
	fmt.Printf("Status:  %s\n", status)
	fmt.Printf("Dir:     %s\n", dir)
	fmt.Printf("Session: %s\n", s.SessionID)
	fmt.Println("─────────────────────────────────────")
}

func printPreviewCapture(paneID string) {
	out, err := exec.Command("tmux", "capture-pane", "-pJ", "-t", paneID).Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		fmt.Println("(session ended)")
		return
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	filtered := filterChromeLines(lines)
	tail := lastN(filtered, 25)
	for _, l := range tail {
		fmt.Println(l)
	}
}
