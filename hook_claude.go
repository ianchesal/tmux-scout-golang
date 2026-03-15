package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ClaudeHookPayload struct {
	HookEventName string          `json:"hook_event_name"`
	SessionID     string          `json:"session_id"`
	CWD           string          `json:"cwd"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input"`
	Prompt        string          `json:"prompt"`
	Source        string          `json:"source"`
	Reason        string          `json:"reason"`
}

var (
	reSystemTag    = regexp.MustCompile(`(?is)<system[-_]?(?:instruction|reminder)[^>]*>.*?</system[-_]?(?:instruction|reminder)>`)
	reLeadingXML   = regexp.MustCompile(`(?s)^\s*<[^>]+>.*?</[^>]+>\s*`)
	reMultiNewline = regexp.MustCompile(`\n{2,}`)
)

func cleanPrompt(prompt string) string {
	s := reSystemTag.ReplaceAllString(prompt, "")
	s = reMultiNewline.ReplaceAllString(s, "\n")
	s = reLeadingXML.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func getToolDetails(toolName string, toolInput json.RawMessage) string {
	if toolInput == nil {
		return toolName
	}
	var m map[string]interface{}
	if err := json.Unmarshal(toolInput, &m); err != nil {
		return toolName
	}

	str := func(key string) (string, bool) {
		v, ok := m[key]
		if !ok {
			return "", false
		}
		s, ok := v.(string)
		return s, ok && s != ""
	}

	trunc := func(s string, n int) string {
		if len(s) <= n {
			return s
		}
		return s[:n]
	}

	if cmd, ok := str("command"); ok {
		return trunc(toolName+": "+cmd, 50)
	}
	if fp, ok := str("file_path"); ok {
		return toolName + ": " + filepath.Base(fp)
	}
	if pat, ok := str("pattern"); ok {
		if path, ok2 := str("path"); ok2 {
			return toolName + ": " + trunc(pat, 30) + " in " + filepath.Base(path)
		}
		return toolName + ": " + trunc(pat, 30)
	}
	if u, ok := str("url"); ok {
		return trunc(toolName+": "+u, 50)
	}
	if q, ok := str("query"); ok {
		return trunc(toolName+": "+q, 50)
	}
	if p, ok := str("prompt"); ok {
		return trunc(toolName+": "+p, 50)
	}
	if d, ok := str("description"); ok {
		return trunc(toolName+": "+d, 50)
	}
	if np, ok := str("notebook_path"); ok {
		return toolName + ": " + filepath.Base(np)
	}
	if sk, ok := str("skill"); ok {
		return toolName + ": " + sk
	}

	// Fallback: first string-valued key with len > 0
	for _, v := range m {
		if s, ok := v.(string); ok && s != "" {
			if len(s) > 40 {
				return toolName + ": " + s[:40] + "..."
			}
			return toolName + ": " + s
		}
	}
	return toolName
}

func runHookClaude() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil || len(strings.TrimSpace(string(data))) == 0 {
		return
	}

	var payload ClaudeHookPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return
	}
	if payload.SessionID == "" {
		return
	}

	scoutDir := defaultScoutDir()
	sf, err := ReadStatusFile(scoutDir)
	if err != nil {
		return
	}

	session, exists := sf.Sessions[payload.SessionID]
	if !exists {
		session = Session{
			SessionID: payload.SessionID,
			AgentType: "claude",
			Status:    "idle",
			StartedAt: NowMs(),
		}
	}

	now := NowMs()

	switch payload.HookEventName {
	case "SessionStart":
		session.Status = "idle"
		session.WorkingDirectory = payload.CWD
		pane := os.Getenv("TMUX_PANE")
		if pane != "" {
			session.TmuxPane = &pane
		}
		pid := os.Getppid()
		session.PID = &pid
		session.NeedsAttention = nil
		session.PendingToolUse = nil
		details := payload.Source
		session.LastEvent = &LastEvent{Type: "session_start", Timestamp: now, Details: details}

	case "UserPromptSubmit":
		session.Status = "working"
		cleaned := cleanPrompt(payload.Prompt)
		if cleaned != "" {
			// Take first line, first 100 chars
			line := cleaned
			if idx := strings.Index(line, "\n"); idx >= 0 {
				line = line[:idx]
			}
			if len(line) > 100 {
				line = line[:100]
			}
			line = strings.TrimSpace(line)
			if line != "" {
				session.SessionTitle = line
				session.LastEvent = &LastEvent{Type: "prompt_submit", Timestamp: now, Details: line}
			} else {
				session.LastEvent = &LastEvent{Type: "prompt_submit", Timestamp: now}
			}
		} else {
			session.LastEvent = &LastEvent{Type: "prompt_submit", Timestamp: now}
		}

	case "PreToolUse":
		session.Status = "working"
		details := getToolDetails(payload.ToolName, payload.ToolInput)
		session.PendingToolUse = &PendingToolUse{
			Tool:      payload.ToolName,
			Details:   details,
			Timestamp: now,
		}
		// Set needsAttention for blocking tools
		switch payload.ToolName {
		case "ExitPlanMode", "AskUserQuestion", "mcp__conductor__AskUserQuestion":
			reason := payload.ToolName
			session.NeedsAttention = &reason
		}
		session.LastEvent = &LastEvent{Type: "tool_use", Timestamp: now}

	case "PostToolUse":
		session.Status = "working"
		session.PendingToolUse = nil

	case "Stop":
		session.Status = "completed"
		session.PendingToolUse = nil
		session.NeedsAttention = nil
		session.LastEvent = &LastEvent{Type: "stop", Timestamp: now}

	case "SessionEnd":
		session.Status = "idle"
		session.EndedAt = &now
		session.PendingToolUse = nil
		session.NeedsAttention = nil
		details := payload.Reason
		session.LastEvent = &LastEvent{Type: "session_end", Timestamp: now, Details: details}
	}

	session.LastUpdated = now
	sf.Sessions[payload.SessionID] = session
	PurgeOldSessions(&sf)
	sf.LastUpdated = now

	if err := WriteStatusFile(scoutDir, sf); err != nil {
		fmt.Fprintf(os.Stderr, "tmux-scout: write status: %v\n", err)
	}
	if err := WriteSession(scoutDir, session); err != nil {
		fmt.Fprintf(os.Stderr, "tmux-scout: write session: %v\n", err)
	}
}
