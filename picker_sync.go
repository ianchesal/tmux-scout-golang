package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

type paneInfo struct {
	PaneID         string
	PanePID        int
	CurrentCommand string
	PaneDead       bool
}

type syncResult struct {
	Status StatusFile
	Panes  map[string]paneInfo
}

var shellCommands = map[string]bool{
	"bash": true, "zsh": true, "sh": true, "fish": true,
	"dash": true, "ksh": true, "tcsh": true, "csh": true, "nu": true,
}

func isShellCommand(cmd string) bool { return shellCommands[cmd] }

func canUseShellFallback(s Session) bool {
	if s.AgentType == "codex" {
		return true
	}
	if s.Status == "working" {
		return true
	}
	if s.PendingToolUse != nil {
		return true
	}
	if s.LastEvent != nil {
		t := s.LastEvent.Type
		if t == "prompt_submit" || t == "tool_use" {
			return true
		}
	}
	return false
}

func getPaneSnapshot() map[string]paneInfo {
	panes := make(map[string]paneInfo)
	out, err := exec.Command("tmux", "list-panes", "-a", "-F",
		"#{pane_id}\t#{pane_pid}\t#{pane_current_command}\t#{pane_dead}\t#{session_name}:#{window_name}").Output()
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

func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	if err == syscall.EPERM {
		return true
	}
	return false
}

func reconcileSessions(scoutDir string, sf *StatusFile, panes map[string]paneInfo) bool {
	now := NowMs()
	changed := false
	for id, s := range sf.Sessions {
		if s.EndedAt != nil || s.TmuxPane == nil {
			continue
		}
		pane, ok := panes[*s.TmuxPane]
		if !ok || pane.PaneDead {
			continue
		}
		var reason string
		if s.PID != nil && *s.PID > 0 && !pidAlive(*s.PID) {
			// If the pane is still running a non-shell command (e.g. "claude"), the stored PID
			// was likely a short-lived wrapper process.  Update to the pane's current PID
			// instead of marking the session as crashed.
			if !isShellCommand(pane.CurrentCommand) && pane.PanePID > 0 {
				s.PID = &pane.PanePID
				s.LastUpdated = now
				sf.Sessions[id] = s
				_ = WriteSession(scoutDir, s)
				changed = true
				continue
			}
			reason = fmt.Sprintf("pid %d exited while pane %s remained open", *s.PID, *s.TmuxPane)
		} else if s.PID == nil && canUseShellFallback(s) && isShellCommand(pane.CurrentCommand) {
			reason = fmt.Sprintf("pane %s returned to shell %s", *s.TmuxPane, pane.CurrentCommand)
		}
		if reason == "" {
			continue
		}
		endedAt := now
		s.Status = "crashed"
		s.EndedAt = &endedAt
		s.NeedsAttention = nil
		s.PendingToolUse = nil
		s.CrashReason = reason
		s.LastEvent = &LastEvent{Type: "process_exit_detected", Timestamp: now, Details: reason}
		s.LastUpdated = now
		sf.Sessions[id] = s
		_ = WriteSession(scoutDir, s)
		changed = true
	}
	return changed
}

type jsonlResult struct {
	Status string
	Title  string
	CWD    string
}

func readCodexJsonl(path string) *jsonlResult {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	codexMarker := "## My request for Codex:"
	processText := func(text string) string {
		if idx := strings.Index(text, codexMarker); idx >= 0 {
			text = text[idx+len(codexMarker):]
		}
		text = strings.TrimSpace(text)
		if nl := strings.Index(text, "\n"); nl >= 0 {
			text = text[:nl]
		}
		if len(text) > 100 {
			text = text[:100]
		}
		return strings.TrimSpace(text)
	}

	var lastUserTitle string
	var lastCompletedTs, lastUserTs int64
	var cwd string
	var waitingForPlan bool
	pendingCalls := make(map[string]bool)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		var ev map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}

		getStr := func(key string) string {
			raw, ok := ev[key]
			if !ok {
				return ""
			}
			var s string
			json.Unmarshal(raw, &s)
			return s
		}
		tsMs := func() int64 {
			ts := getStr("timestamp")
			if ts == "" {
				return 0
			}
			t, err := time.Parse(time.RFC3339, ts)
			if err != nil {
				return 0
			}
			return t.UnixMilli()
		}

		evType := getStr("type")
		var payload map[string]json.RawMessage
		if raw, ok := ev["payload"]; ok {
			json.Unmarshal(raw, &payload)
		}
		payloadStr := func(key string) string {
			raw, ok := payload[key]
			if !ok {
				return ""
			}
			var s string
			json.Unmarshal(raw, &s)
			return s
		}

		switch evType {
		case "event_msg":
			pt := payloadStr("type")
			switch pt {
			case "user_message":
				msg := payloadStr("message")
				if t := processText(msg); t != "" {
					lastUserTitle = t
				}
				lastUserTs = tsMs()
				waitingForPlan = false
			case "task_complete":
				lastCompletedTs = tsMs()
			case "item_completed":
				var item map[string]string
				if raw, ok := payload["item"]; ok {
					json.Unmarshal(raw, &item)
				}
				if item["type"] == "Plan" {
					waitingForPlan = true
				}
			}
		case "turn_context":
			if raw, ok := payload["cwd"]; ok {
				json.Unmarshal(raw, &cwd)
			}
		case "response_item":
			pt := payloadStr("type")
			callID := payloadStr("call_id")
			switch pt {
			case "custom_tool_call", "function_call":
				if callID != "" {
					pendingCalls[callID] = true
				}
			case "custom_tool_call_output", "function_call_output":
				delete(pendingCalls, callID)
			}
			if payloadStr("role") == "user" {
				var content []map[string]json.RawMessage
				if raw, ok := payload["content"]; ok {
					json.Unmarshal(raw, &content)
				}
				for _, part := range content {
					var ptype, ptext string
					if r, ok := part["type"]; ok {
						json.Unmarshal(r, &ptype)
					}
					if r, ok := part["text"]; ok {
						json.Unmarshal(r, &ptext)
					}
					if (ptype == "text" || ptype == "input_text") && ptext != "" {
						if t := processText(ptext); t != "" {
							lastUserTitle = t
							lastUserTs = tsMs()
							waitingForPlan = false
						}
					}
				}
			}
		}
	}

	now := time.Now().UnixMilli()
	status := "completed"
	if waitingForPlan && lastCompletedTs > 0 {
		status = "needsAttention"
	} else if len(pendingCalls) > 0 {
		if now-lastCompletedTs > 30000 && now-lastUserTs > 30000 {
			status = "needsAttention"
		} else {
			status = "working"
		}
	} else if lastUserTs > lastCompletedTs {
		status = "working"
	}

	return &jsonlResult{Status: status, Title: lastUserTitle, CWD: cwd}
}

var uuidRegex = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

func syncCodexSessions(scoutDir string, sf *StatusFile) bool {
	home, _ := os.UserHomeDir()
	sessionsBase := filepath.Join(home, ".codex", "sessions")
	if _, err := os.Stat(sessionsBase); err != nil {
		return false
	}

	now := time.Now()
	yesterday := time.Now().Add(-24 * time.Hour)
	changed := false
	knownThreadIDs := make(map[string]bool)
	for _, s := range sf.Sessions {
		if s.AgentType == "codex" && s.ThreadID != nil {
			knownThreadIDs[*s.ThreadID] = true
		}
	}

	jsonlPathCache := make(map[string]string)
	jsonlResultCache := make(map[string]*jsonlResult)

	for _, day := range []time.Time{now, yesterday} {
		dir := filepath.Join(sessionsBase, day.Format("2006"), day.Format("01"), day.Format("02"))
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".jsonl") {
				continue
			}
			m := uuidRegex.FindString(entry.Name())
			if m == "" {
				continue
			}
			fp := filepath.Join(dir, entry.Name())
			if _, exists := jsonlPathCache[m]; !exists {
				jsonlPathCache[m] = fp
			}
			if knownThreadIDs[m] {
				continue
			}
			fi, err := os.Stat(fp)
			if err != nil || time.Since(fi.ModTime()) > 5*time.Minute {
				continue
			}
			result := readCodexJsonl(fp)
			if result == nil {
				continue
			}
			jsonlResultCache[fp] = result

			threadID := m
			nowMs := NowMs()
			startedAt := nowMs
			if fi.Mode().IsRegular() {
				startedAt = fi.ModTime().UnixMilli()
			}
			var needsAttn *string
			if result.Status == "needsAttention" {
				w := "waiting"
				needsAttn = &w
			}
			sessionStatus := result.Status
			if result.Status == "needsAttention" {
				sessionStatus = "working"
			}
			sf.Sessions[m] = Session{
				SessionID:        m,
				AgentType:        "codex",
				StartedAt:        startedAt,
				Status:           sessionStatus,
				NeedsAttention:   needsAttn,
				WorkingDirectory: result.CWD,
				SessionTitle:     result.Title,
				ThreadID:         &threadID,
				LastEvent:        &LastEvent{Type: "discovered", Timestamp: nowMs},
				LastUpdated:      nowMs,
			}
			knownThreadIDs[m] = true
			changed = true
		}
	}

	// Phase 2: enrich existing
	for id, s := range sf.Sessions {
		if s.AgentType != "codex" || s.EndedAt != nil {
			continue
		}
		tid := id
		if s.ThreadID != nil {
			tid = *s.ThreadID
		}
		jsonlPath := jsonlPathCache[tid]
		if jsonlPath == "" {
			continue
		}
		// Stale crash detection for unbound sessions
		if s.TmuxPane == nil && s.PID == nil {
			fi, err := os.Stat(jsonlPath)
			if err == nil && time.Since(fi.ModTime()) > 5*time.Minute {
				nowMs := NowMs()
				endedAt := nowMs
				s.Status = "crashed"
				s.EndedAt = &endedAt
				s.NeedsAttention = nil
				s.PendingToolUse = nil
				s.CrashReason = "JSONL file inactive"
				s.LastEvent = &LastEvent{Type: "process_exit_detected", Timestamp: nowMs}
				s.LastUpdated = nowMs
				sf.Sessions[id] = s
				_ = WriteSession(scoutDir, s)
				changed = true
				continue
			}
		}
		result := jsonlResultCache[jsonlPath]
		if result == nil {
			result = readCodexJsonl(jsonlPath)
		}
		if result == nil {
			continue
		}
		sessionChanged := false
		if result.Title != "" && result.Title != s.SessionTitle {
			s.SessionTitle = result.Title
			sessionChanged = true
		}
		var newStatus string
		var newNeedsAttn *string
		switch result.Status {
		case "needsAttention":
			newStatus = "working"
			w := "waiting for approval"
			newNeedsAttn = &w
		case "working":
			newStatus = "working"
		default:
			newStatus = "completed"
		}
		if newStatus != s.Status || ptrStr(newNeedsAttn) != ptrStr(s.NeedsAttention) {
			s.Status = newStatus
			s.NeedsAttention = newNeedsAttn
			s.LastUpdated = NowMs()
			sessionChanged = true
		}
		if sessionChanged {
			sf.Sessions[id] = s
			_ = WriteSession(scoutDir, s)
			changed = true
		}
	}
	return changed
}

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

var (
	claudeBusyRE        = regexp.MustCompile(`[↓↑] [\d.,]+[kKmM]? tokens\)|✻ Thinking|∴ Thinking`)
	claudeDoneRE        = regexp.MustCompile(`✻ (Baked|Brewed|Churned|Cogitated|Cooked|Crunched|Sautéed|Worked) for `)
	claudeIdleRE        = regexp.MustCompile(`✻ Idle`)
	claudeInterruptedRE = regexp.MustCompile(`Interrupted . What should Claude do instead`)
	claudeWaitStrings   = []string{
		"Do you want to proceed?", "Would you like to proceed?",
		"Enter plan mode?", "Exit plan mode?", "Do you want to allow",
	}
	codexWaitFooter = []string{"enter to submit answer", "enter to submit all"}
	codexWaitDialog = []string{
		"enter to submit", "Implement this plan?", "Approve Once",
		"approve network access", "Submit with unanswered",
		"Install MCP servers?", "Enable full access?", "Enable multi-agent?",
	}
)

func detectPaneState(paneID, agentType string) string {
	out, err := exec.Command("tmux", "capture-pane", "-t", paneID, "-p", "-S", "-15").Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	start := len(lines) - 8
	if start < 0 {
		start = 0
	}
	tail := strings.Join(lines[start:], "\n")

	if agentType == "codex" {
		for _, s := range codexWaitFooter {
			if strings.Contains(tail, s) {
				return "needsAttention"
			}
		}
		if strings.Contains(tail, "esc to interrupt") {
			return "working"
		}
		for _, s := range codexWaitDialog {
			if strings.Contains(tail, s) {
				return "needsAttention"
			}
		}
		return "completed"
	}

	if claudeBusyRE.MatchString(tail) {
		return "working"
	}
	if claudeDoneRE.MatchString(tail) {
		return "completed"
	}
	for _, s := range claudeWaitStrings {
		if strings.Contains(tail, s) {
			return "needsAttention"
		}
	}
	if claudeIdleRE.MatchString(tail) {
		return "completed"
	}
	if claudeInterruptedRE.MatchString(tail) {
		return "completed"
	}
	return ""
}

func applyPaneGroundTruth(scoutDir string, sf *StatusFile) bool {
	changed := false
	now := NowMs()
	for id, s := range sf.Sessions {
		if s.TmuxPane == nil || s.EndedAt != nil {
			continue
		}
		state := detectPaneState(*s.TmuxPane, s.AgentType)
		if state == "" {
			continue
		}
		sessionChanged := false
		switch state {
		case "needsAttention":
			w := "waiting for approval"
			if ptrStr(s.NeedsAttention) != w {
				s.NeedsAttention = &w
				sessionChanged = true
			}
		case "working":
			if s.NeedsAttention != nil || s.PendingToolUse != nil || s.Status != "working" {
				s.NeedsAttention = nil
				s.PendingToolUse = nil
				s.Status = "working"
				sessionChanged = true
			}
		case "completed":
			if s.NeedsAttention != nil || s.PendingToolUse != nil || s.Status != "completed" {
				s.NeedsAttention = nil
				s.PendingToolUse = nil
				s.Status = "completed"
				sessionChanged = true
			}
		}
		if sessionChanged {
			s.LastUpdated = now
			sf.Sessions[id] = s
			_ = WriteSession(scoutDir, s)
			changed = true
		}
	}
	return changed
}

func Sync(statusFilePath, scoutDir string) syncResult {
	sf, _ := ReadStatusFile(scoutDir)
	panes := getPaneSnapshot()
	changed := reconcileSessions(scoutDir, &sf, panes)
	if syncCodexSessions(scoutDir, &sf) {
		changed = true
	}
	if applyPaneGroundTruth(scoutDir, &sf) {
		changed = true
	}
	if changed {
		sf.LastUpdated = NowMs()
		_ = WriteStatusFile(scoutDir, sf)
	}
	return syncResult{Status: sf, Panes: panes}
}
