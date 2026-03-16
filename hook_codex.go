package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type codexOriginalNotify struct {
	Notify []string `json:"notify"`
}

func forwardToOriginalNotify(jsonArg string) {
	scoutDir := defaultScoutDir()
	backupPath := scoutDir + "/codex-original-notify.json"
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return
	}
	var orig codexOriginalNotify
	if err := json.Unmarshal(data, &orig); err != nil || len(orig.Notify) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	args := append(orig.Notify[1:], jsonArg)
	cmd := exec.CommandContext(ctx, orig.Notify[0], args...) // #nosec G204 -- command sourced from user's own backed-up Codex config, not external input
	_ = cmd.Run()
}

func extractCodexTitle(inputMessages json.RawMessage, lastAssistantMsg string) string {
	marker := "## My request for Codex:"

	processText := func(text string) string {
		if idx := strings.Index(text, marker); idx >= 0 {
			text = text[idx+len(marker):]
		}
		// trim leading whitespace left by marker removal before first-line extraction
		text = strings.TrimLeft(text, " \t\r\n")
		// first line, first 100 chars, trimmed
		if nl := strings.Index(text, "\n"); nl >= 0 {
			text = text[:nl]
		}
		text = strings.TrimSpace(text)
		if len(text) > 100 {
			text = text[:100]
		}
		return text
	}

	if inputMessages != nil {
		var msgs []map[string]interface{}
		if err := json.Unmarshal(inputMessages, &msgs); err == nil {
			for _, msg := range msgs {
				role, _ := msg["role"].(string)
				if role != "user" {
					continue
				}
				switch c := msg["content"].(type) {
				case string:
					if t := processText(c); t != "" {
						return t
					}
				case []interface{}:
					for _, item := range c {
						m, ok := item.(map[string]interface{})
						if !ok {
							continue
						}
						typ, _ := m["type"].(string)
						if typ != "text" && typ != "input_text" {
							continue
						}
						text, _ := m["text"].(string)
						if t := processText(text); t != "" {
							return t
						}
					}
				}
			}
		}
	}

	// Fallback: first 100 chars of last-assistant-message, first line
	if lastAssistantMsg != "" {
		text := lastAssistantMsg
		if nl := strings.Index(text, "\n"); nl >= 0 {
			text = text[:nl]
		}
		text = strings.TrimSpace(text)
		if len(text) > 100 {
			text = text[:100]
		}
		return text
	}
	return ""
}

type codexPayload struct {
	Type             string          `json:"type"`
	ThreadID         string          `json:"thread-id"`
	TurnID           string          `json:"turn-id"`
	CWD              string          `json:"cwd"`
	InputMessages    json.RawMessage `json:"input-messages"`
	LastAssistantMsg string          `json:"last-assistant-message"`
}

func runHookCodex(jsonArg string) {
	forwardToOriginalNotify(jsonArg)

	if strings.TrimSpace(jsonArg) == "" {
		return
	}

	var p codexPayload
	if err := json.Unmarshal([]byte(jsonArg), &p); err != nil {
		return
	}
	if p.Type != "agent-turn-complete" {
		return
	}

	title := extractCodexTitle(p.InputMessages, p.LastAssistantMsg)
	now := NowMs()

	scoutDir := defaultScoutDir()
	sf, err := ReadStatusFile(scoutDir)
	if err != nil {
		return
	}

	// Use threadId as session key; fallback to turnId
	sessionKey := p.ThreadID
	if sessionKey == "" {
		sessionKey = p.TurnID
	}
	if sessionKey == "" {
		return
	}

	session, exists := sf.Sessions[sessionKey]
	if !exists {
		session = Session{
			SessionID: sessionKey,
			StartedAt: now,
		}
	}

	session.AgentType = "codex"
	session.Status = "completed"
	session.EndedAt = nil
	session.LastUpdated = now
	session.WorkingDirectory = p.CWD
	if title != "" {
		session.SessionTitle = title
	}

	pane := os.Getenv("TMUX_PANE")
	if pane != "" {
		session.TmuxPane = &pane
	}
	pid := os.Getppid()
	session.PID = &pid

	threadID := p.ThreadID
	if threadID != "" {
		session.ThreadID = &threadID
	}

	session.LastEvent = &LastEvent{
		Type:      "turn_complete",
		Timestamp: now,
		TurnID:    p.TurnID,
	}

	sf.Sessions[sessionKey] = session
	sf.LastUpdated = now

	if err := WriteStatusFile(scoutDir, sf); err != nil {
		fmt.Fprintf(os.Stderr, "tmux-scout: write status: %v\n", err)
	}
	if err := WriteSession(scoutDir, session); err != nil {
		fmt.Fprintf(os.Stderr, "tmux-scout: write session: %v\n", err)
	}
}
