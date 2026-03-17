package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type GeminiHookPayload struct {
	HookEventName    string          `json:"hook_event_name"`
	SessionID        string          `json:"session_id"`
	CWD              string          `json:"cwd"`
	Timestamp        string          `json:"timestamp"`
	Source           string          `json:"source"`
	Reason           string          `json:"reason"`
	Prompt           string          `json:"prompt"`
	PromptResponse   string          `json:"prompt_response"`
	StopHookActive   bool            `json:"stop_hook_active"`
	ToolName         string          `json:"tool_name"`
	ToolInput        json.RawMessage `json:"tool_input"`
	ToolResponse     json.RawMessage `json:"tool_response"`
	NotificationType string          `json:"notification_type"`
	Message          string          `json:"message"`
	Details          json.RawMessage `json:"details"`
}

func applyGeminiHook(session Session, payload GeminiHookPayload, pane string, pid int, now int64) Session {
	// Any non-SessionEnd event proves session is alive — clear stale crash/end state.
	if payload.HookEventName != "SessionEnd" && session.EndedAt != nil {
		session.EndedAt = nil
		session.CrashReason = ""
		session.Status = "idle"
	}

	switch payload.HookEventName {
	case "SessionStart":
		session.Status = "idle"
		session.WorkingDirectory = payload.CWD
		if pane != "" {
			session.TmuxPane = &pane
		}
		session.PID = &pid
		session.NeedsAttention = nil
		session.PendingToolUse = nil
		session.LastEvent = &LastEvent{Type: "session_start", Timestamp: now, Details: payload.Source}

	case "BeforeAgent":
		session.Status = "working"
		cleaned := cleanPrompt(payload.Prompt)
		if cleaned != "" {
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

	case "BeforeTool":
		session.Status = "working"
		details := getToolDetails(payload.ToolName, payload.ToolInput)
		session.PendingToolUse = &PendingToolUse{
			Tool:      payload.ToolName,
			Details:   details,
			Timestamp: now,
		}
		session.LastEvent = &LastEvent{Type: "tool_use", Timestamp: now}

	case "AfterTool":
		session.Status = "working"
		session.PendingToolUse = nil

	case "AfterAgent":
		session.Status = "completed"
		session.PendingToolUse = nil
		session.NeedsAttention = nil
		session.LastEvent = &LastEvent{Type: "stop", Timestamp: now}

	case "SessionEnd":
		session.Status = "idle"
		session.PendingToolUse = nil
		session.NeedsAttention = nil
		if payload.Reason == "clear" {
			session.EndedAt = nil
			session.SessionTitle = ""
		} else {
			session.EndedAt = &now
		}
		session.LastEvent = &LastEvent{Type: "session_end", Timestamp: now, Details: payload.Reason}

	case "Notification":
		if payload.NotificationType == "ToolPermission" {
			session.NeedsAttention = &payload.Message
		}
	}

	session.LastUpdated = now
	return session
}

func runHookGemini() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil || len(strings.TrimSpace(string(data))) == 0 {
		return
	}

	var payload GeminiHookPayload
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
			AgentType: "gemini",
			Status:    "idle",
			StartedAt: NowMs(),
		}
	}

	pane := os.Getenv("TMUX_PANE")
	pid := os.Getppid()
	now := NowMs()

	session = applyGeminiHook(session, payload, pane, pid, now)

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
