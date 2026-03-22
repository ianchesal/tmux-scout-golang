package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LastEvent struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
	Details   string `json:"details,omitempty"`
	TurnID    string `json:"turnId,omitempty"`
}

type PendingToolUse struct {
	Tool      string `json:"tool"`
	Details   string `json:"details"`
	Timestamp int64  `json:"timestamp"`
}

type Session struct {
	SessionID        string          `json:"sessionId"`
	AgentType        string          `json:"agentType"`
	Status           string          `json:"status"`
	NeedsAttention   *string         `json:"needsAttention"`
	TmuxPane         *string         `json:"tmuxPane"`
	WorkingDirectory string          `json:"workingDirectory"`
	SessionTitle     string          `json:"sessionTitle,omitempty"`
	PID              *int            `json:"pid"`
	ThreadID         *string         `json:"threadId,omitempty"`
	StartedAt        int64           `json:"startedAt"`
	EndedAt          *int64          `json:"endedAt"`
	LastUpdated      int64           `json:"lastUpdated"`
	PendingToolUse   *PendingToolUse `json:"pendingToolUse"`
	LastEvent        *LastEvent      `json:"lastEvent,omitempty"`
	CrashReason      string          `json:"crashReason,omitempty"`
}

type StatusFile struct {
	Version     int                `json:"version"`
	Sessions    map[string]Session `json:"sessions"`
	LastUpdated int64              `json:"lastUpdated"`
}

func NowMs() int64 { return time.Now().UnixMilli() }

func SanitizeID(id string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' {
			return '_'
		}
		return r
	}, id)
}

func defaultScoutDir() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "tmux-scout")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "tmux-scout")
}

func statusFilePath(scoutDir string) string {
	return filepath.Join(scoutDir, "status.json")
}

func sessionFilePath(scoutDir, id string) string {
	return filepath.Join(scoutDir, "sessions", SanitizeID(id)+".json")
}

func writeAtomic(path string, data []byte) error {
	tmp := fmt.Sprintf("%s.tmp.%d", path, os.Getpid())
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(tmp, data, 0644); err != nil { //nolint:gosec // #nosec G703 -- tmp path is constructed internally via fmt.Sprintf from a caller-controlled path, not from user input
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func ReadStatusFile(scoutDir string) (StatusFile, error) {
	data, err := os.ReadFile(statusFilePath(scoutDir))
	if err != nil {
		if os.IsNotExist(err) {
			return StatusFile{Version: 1, Sessions: map[string]Session{}, LastUpdated: NowMs()}, nil
		}
		return StatusFile{}, err
	}
	var sf StatusFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return StatusFile{Version: 1, Sessions: map[string]Session{}, LastUpdated: NowMs()}, nil
	}
	if sf.Sessions == nil {
		sf.Sessions = map[string]Session{}
	}
	return sf, nil
}

func WriteStatusFile(scoutDir string, sf StatusFile) error {
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(statusFilePath(scoutDir), data)
}

func ReadSession(scoutDir, id string) (Session, error) {
	data, err := os.ReadFile(sessionFilePath(scoutDir, id))
	if err != nil {
		return Session{}, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return Session{}, err
	}
	return s, nil
}

func WriteSession(scoutDir string, s Session) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(sessionFilePath(scoutDir, s.SessionID), data)
}

func PurgeOldSessions(sf *StatusFile) {
	cutoff := NowMs() - 24*60*60*1000
	for id, s := range sf.Sessions {
		if s.EndedAt != nil && *s.EndedAt < cutoff {
			delete(sf.Sessions, id)
		}
	}
}
