package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeID(t *testing.T) {
	cases := []struct{ in, want string }{
		{"abc", "abc"},
		{"a/b/c", "a_b_c"},
		{"a\\b", "a_b"},
		{"a:b", "a_b"},
		{"sess:window/pane", "sess_window_pane"},
	}
	for _, c := range cases {
		if got := SanitizeID(c.in); got != c.want {
			t.Errorf("SanitizeID(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestReadStatusFile_NotExist(t *testing.T) {
	dir := t.TempDir()
	sf, err := ReadStatusFile(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sf.Version != 1 {
		t.Errorf("Version = %d, want 1", sf.Version)
	}
	if sf.Sessions == nil {
		t.Error("Sessions should be non-nil map")
	}
}

func TestWriteReadStatusFile_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	pane := "%1"
	sf := StatusFile{
		Version:     1,
		LastUpdated: 1000,
		Sessions: map[string]Session{
			"s1": {
				SessionID: "s1",
				AgentType: "claude",
				Status:    "working",
				TmuxPane:  &pane,
			},
		},
	}
	if err := WriteStatusFile(dir, sf); err != nil {
		t.Fatalf("write error: %v", err)
	}
	// Verify temp file does not linger
	tmpPattern := filepath.Join(dir, "status.json.tmp.*")
	matches, _ := filepath.Glob(tmpPattern)
	if len(matches) > 0 {
		t.Errorf("temp file not cleaned up: %v", matches)
	}

	got, err := ReadStatusFile(dir)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if len(got.Sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(got.Sessions))
	}
	if got.Sessions["s1"].Status != "working" {
		t.Errorf("status = %q, want working", got.Sessions["s1"].Status)
	}
}

func TestWriteReadSession_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sessions"), 0755); err != nil {
		t.Fatal(err)
	}
	pid := 42
	s := Session{SessionID: "abc:123", AgentType: "claude", Status: "idle", PID: &pid}
	if err := WriteSession(dir, s); err != nil {
		t.Fatalf("write error: %v", err)
	}
	got, err := ReadSession(dir, "abc:123")
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if got.SessionID != "abc:123" {
		t.Errorf("SessionID = %q, want abc:123", got.SessionID)
	}
	if got.PID == nil || *got.PID != 42 {
		t.Error("PID not preserved")
	}
	// Confirm file is stored with sanitized name
	sanitized := filepath.Join(dir, "sessions", "abc_123.json")
	if _, err := os.Stat(sanitized); err != nil {
		t.Errorf("expected file %s to exist", sanitized)
	}
}

func TestPurgeOldSessions(t *testing.T) {
	old := int64(1000) // very old endedAt
	recent := NowMs() - 1000
	sf := &StatusFile{
		Sessions: map[string]Session{
			"old":    {SessionID: "old", EndedAt: &old},
			"recent": {SessionID: "recent", EndedAt: &recent},
			"live":   {SessionID: "live"},
		},
	}
	PurgeOldSessions(sf)
	if _, ok := sf.Sessions["old"]; ok {
		t.Error("old session should have been purged")
	}
	if _, ok := sf.Sessions["recent"]; !ok {
		t.Error("recent session should be kept")
	}
	if _, ok := sf.Sessions["live"]; !ok {
		t.Error("live session should be kept")
	}
}
