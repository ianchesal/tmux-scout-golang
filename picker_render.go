package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func getPaneSnapshotRender() map[string]paneInfo {
	panes := make(map[string]paneInfo)
	out, err := exec.Command("tmux", "list-panes", "-a", "-F",
		"#{pane_id}\t#{pane_pid}\t#{pane_current_command}\t#{pane_dead}").Output()
	if err != nil {
		return panes
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			continue
		}
		var pid int
		fmt.Sscanf(parts[1], "%d", &pid)
		panes[parts[0]] = paneInfo{
			PaneID:         parts[0],
			PanePID:        pid,
			CurrentCommand: parts[2],
			PaneDead:       parts[3] == "1",
		}
	}
	return panes
}

func isNeedsAttention(s Session) bool {
	if s.NeedsAttention != nil {
		return true
	}
	if s.PendingToolUse != nil && s.PendingToolUse.Timestamp > 0 {
		return NowMs()-s.PendingToolUse.Timestamp > 5000
	}
	return false
}

func isActiveSession(s Session, panes map[string]paneInfo) bool {
	if s.EndedAt != nil || s.Status == "crashed" {
		return false
	}
	if s.TmuxPane == nil {
		return true // unbound, discovered from JSONL
	}
	pane, ok := panes[*s.TmuxPane]
	if !ok || pane.PaneDead {
		return false
	}
	if s.PID == nil {
		if canUseShellFallback(s) && isShellCommand(pane.CurrentCommand) {
			return false
		}
		return true
	}
	return pidAlive(*s.PID)
}

func getActiveSessions(sf StatusFile, panes map[string]paneInfo) []Session {
	byPane := make(map[string]Session)
	var unbound []Session
	for _, s := range sf.Sessions {
		if !isActiveSession(s, panes) {
			continue
		}
		if s.TmuxPane == nil {
			unbound = append(unbound, s)
			continue
		}
		if existing, ok := byPane[*s.TmuxPane]; !ok || s.LastUpdated > existing.LastUpdated {
			byPane[*s.TmuxPane] = s
		}
	}
	result := make([]Session, 0, len(byPane)+len(unbound))
	for _, s := range byPane {
		result = append(result, s)
	}
	return append(result, unbound...)
}

func groupOrder(s Session) int {
	if isNeedsAttention(s) {
		return 0
	}
	switch s.Status {
	case "working":
		return 1
	case "completed":
		return 2
	}
	return 3
}

func formatLine(s Session, currentPane string) string {
	unbound := s.TmuxPane == nil
	paneID := "UNBOUND"
	if !unbound {
		paneID = *s.TmuxPane
	}

	tag := "\x1b[34m[ IDLE ]\x1b[0m"
	if isNeedsAttention(s) {
		tag = "\x1b[31m[ WAIT ]\x1b[0m"
	} else if s.Status == "working" {
		tag = "\x1b[33m[ BUSY ]\x1b[0m"
	} else if s.Status == "completed" {
		tag = "\x1b[32m[ DONE ]\x1b[0m"
	}

	cur := " "
	if !unbound && currentPane != "" && *s.TmuxPane == currentPane {
		cur = "\x1b[33m*\x1b[0m"
	}

	agent := "\x1b[38;5;209mclaude\x1b[0m"
	if s.AgentType == "codex" {
		agent = "\x1b[38;5;114mcodex \x1b[0m"
	} else if s.AgentType == "gemini" {
		agent = "\x1b[38;5;33mgemini\x1b[0m"
	}

	projectName := filepath.Base(s.WorkingDirectory)
	if projectName == "" || projectName == "." {
		projectName = "?"
	}
	if len(projectName) > 25 {
		projectName = projectName[:24] + "~"
	}
	project := fmt.Sprintf("%-25s", projectName)

	title := ""
	if s.SessionTitle != "" {
		t := strings.ReplaceAll(s.SessionTitle, "\r", " ")
		t = strings.ReplaceAll(t, "\n", " ")
		if len(t) > 50 {
			t = t[:50]
		}
		title = fmt.Sprintf("\x1b[2m\"%s\"\x1b[0m", t)
	}

	detail := ""
	if unbound {
		detail = "  \x1b[2m(pane not yet linked — waiting for first response)\x1b[0m"
	} else if s.PendingToolUse != nil && s.PendingToolUse.Details != "" {
		d := strings.ReplaceAll(s.PendingToolUse.Details, "\r", " ")
		d = strings.ReplaceAll(d, "\n", " ")
		if len(d) > 40 {
			d = d[:40]
		}
		detail = fmt.Sprintf("  \x1b[36m%s\x1b[0m", d)
	}

	return fmt.Sprintf("%s\t%s %s %s %s %s%s", paneID, cur, tag, agent, project, title, detail)
}

func Render(sf StatusFile, currentPane string, panes map[string]paneInfo) {
	if panes == nil {
		panes = getPaneSnapshotRender()
	}

	active := getActiveSessions(sf, panes)
	sort.Slice(active, func(i, j int) bool {
		gi, gj := groupOrder(active[i]), groupOrder(active[j])
		if gi != gj {
			return gi < gj
		}
		return active[i].LastUpdated > active[j].LastUpdated
	})

	hStatus := fmt.Sprintf("%-8s", "STATUS ")
	hProject := fmt.Sprintf("%-25s", "PROJECT")
	fmt.Printf("_\t  %s AGENT  %s TITLE\n", hStatus, hProject)

	if len(active) == 0 {
		fmt.Println("NONE\tNo active sessions found.")
		return
	}
	for _, s := range active {
		fmt.Println(formatLine(s, currentPane))
	}
}
