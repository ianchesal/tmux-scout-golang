package main

import (
	"strings"
	"testing"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
func i64Ptr(i int64) *int64   { return &i }

func TestIsActiveSession_EndedAt(t *testing.T) {
	s := Session{SessionID: "s1", EndedAt: i64Ptr(1000)}
	if isActiveSession(s, nil) {
		t.Error("session with endedAt should not be active")
	}
}

func TestIsActiveSession_Crashed(t *testing.T) {
	s := Session{SessionID: "s1", Status: "crashed"}
	if isActiveSession(s, nil) {
		t.Error("crashed session should not be active")
	}
}

func TestIsActiveSession_NoPaneIncluded(t *testing.T) {
	s := Session{SessionID: "s1", AgentType: "codex", Status: "working"}
	if !isActiveSession(s, map[string]paneInfo{}) {
		t.Error("unbound session should be active")
	}
}

func TestIsActiveSession_DeadPane(t *testing.T) {
	pane := strPtr("%1")
	s := Session{SessionID: "s1", TmuxPane: pane, Status: "working"}
	panes := map[string]paneInfo{"%1": {PaneID: "%1", PaneDead: true}}
	if isActiveSession(s, panes) {
		t.Error("session with dead pane should not be active")
	}
}

func TestIsActiveSession_PaneGone(t *testing.T) {
	pane := strPtr("%99")
	s := Session{SessionID: "s1", TmuxPane: pane, Status: "working"}
	panes := map[string]paneInfo{} // %99 not present
	if isActiveSession(s, panes) {
		t.Error("session with missing pane should not be active")
	}
}

func TestIsNeedsAttention_NeedsAttentionField(t *testing.T) {
	s := Session{NeedsAttention: strPtr("waiting")}
	if !isNeedsAttention(s) {
		t.Error("session with needsAttention set should need attention")
	}
}

func TestIsNeedsAttention_StaleTool(t *testing.T) {
	old := NowMs() - 10000 // 10s ago
	s := Session{PendingToolUse: &PendingToolUse{Tool: "Bash", Timestamp: old}}
	if !isNeedsAttention(s) {
		t.Error("session with stale pendingToolUse should need attention")
	}
}

func TestIsNeedsAttention_RecentTool(t *testing.T) {
	recent := NowMs() - 1000 // 1s ago
	s := Session{PendingToolUse: &PendingToolUse{Tool: "Bash", Timestamp: recent}}
	if isNeedsAttention(s) {
		t.Error("session with recent pendingToolUse should NOT need attention yet")
	}
}

func TestFormatLine_WaitTag(t *testing.T) {
	s := Session{
		SessionID:        "s1",
		AgentType:        "claude",
		Status:           "working",
		NeedsAttention:   strPtr("ExitPlanMode"),
		WorkingDirectory: "/home/user/myproject",
		TmuxPane:         strPtr("%5"),
	}
	line := formatLine(s, "%7")
	if !strings.HasPrefix(line, "%5\t") {
		t.Errorf("line should start with pane ID: %q", line)
	}
	if !strings.Contains(line, "WAIT") {
		t.Errorf("WAIT tag expected in: %q", line)
	}
	if strings.Contains(line, "[ WAIT ]") {
		t.Errorf("brackets should be removed from tag: %q", line)
	}
}

func TestFormatLine_Unbound(t *testing.T) {
	s := Session{
		SessionID:        "s1",
		AgentType:        "codex",
		Status:           "working",
		WorkingDirectory: "/home/user/project",
	}
	line := formatLine(s, "")
	if !strings.HasPrefix(line, "UNBOUND\t") {
		t.Errorf("unbound line should start with UNBOUND: %q", line)
	}
	if !strings.Contains(line, "not yet linked") {
		t.Errorf("unbound detail expected: %q", line)
	}
}

func TestFormatLine_CurrentPane(t *testing.T) {
	pane := "%3"
	s := Session{
		SessionID:        "s1",
		AgentType:        "claude",
		Status:           "completed",
		WorkingDirectory: "/home/user/proj",
		TmuxPane:         &pane,
	}
	line := formatLine(s, "%3")
	if !strings.Contains(line, "*") {
		t.Errorf("current pane marker * expected: %q", line)
	}
}

func TestGetActiveSessions_DeduplicateByPane(t *testing.T) {
	pane := "%1"
	panes := map[string]paneInfo{"%1": {PaneID: "%1"}}
	sf := StatusFile{Sessions: map[string]Session{
		"old": {SessionID: "old", TmuxPane: &pane, Status: "working", LastUpdated: 100},
		"new": {SessionID: "new", TmuxPane: &pane, Status: "working", LastUpdated: 200},
	}}
	active := getActiveSessions(sf, panes)
	if len(active) != 1 {
		t.Fatalf("expected 1 session after dedup, got %d", len(active))
	}
	if active[0].SessionID != "new" {
		t.Errorf("expected newest session, got %q", active[0].SessionID)
	}
}

func TestFormatLine_GeminiAgent(t *testing.T) {
	s := Session{
		SessionID:        "s1",
		AgentType:        "gemini",
		Status:           "idle",
		WorkingDirectory: "/home/user/project",
		TmuxPane:         strPtr("%9"),
	}
	line := formatLine(s, "")
	if !strings.Contains(line, "gemini") {
		t.Errorf("gemini agent label expected in line: %q", line)
	}
}

func TestFormatLine_BusyTagNoBrackets(t *testing.T) {
	s := Session{
		SessionID:        "s1",
		AgentType:        "claude",
		Status:           "working",
		WorkingDirectory: "/home/user/proj",
		TmuxPane:         strPtr("%1"),
	}
	line := formatLine(s, "")
	if !strings.Contains(line, "BUSY") {
		t.Errorf("BUSY tag expected: %q", line)
	}
	if strings.Contains(line, "[ BUSY ]") {
		t.Errorf("brackets must be gone: %q", line)
	}
}

func TestFormatLine_DoneTagNoBrackets(t *testing.T) {
	s := Session{
		SessionID:        "s1",
		AgentType:        "claude",
		Status:           "completed",
		WorkingDirectory: "/home/user/proj",
		TmuxPane:         strPtr("%1"),
	}
	line := formatLine(s, "")
	if !strings.Contains(line, "DONE") {
		t.Errorf("DONE tag expected: %q", line)
	}
	if strings.Contains(line, "[ DONE ]") {
		t.Errorf("brackets must be gone: %q", line)
	}
}

func TestFormatLine_IdleTagNoBrackets(t *testing.T) {
	s := Session{
		SessionID:        "s1",
		AgentType:        "claude",
		Status:           "idle",
		WorkingDirectory: "/home/user/proj",
		TmuxPane:         strPtr("%1"),
	}
	line := formatLine(s, "")
	if !strings.Contains(line, "IDLE") {
		t.Errorf("IDLE tag expected: %q", line)
	}
	if strings.Contains(line, "[ IDLE ]") {
		t.Errorf("brackets must be gone: %q", line)
	}
}

func TestFormatLine_TitleWithSpaceShown(t *testing.T) {
	s := Session{
		SessionID:        "s1",
		AgentType:        "claude",
		Status:           "working",
		WorkingDirectory: "/home/user/proj",
		SessionTitle:     "Fix the auth bug",
		TmuxPane:         strPtr("%1"),
	}
	line := formatLine(s, "")
	if !strings.Contains(line, "Fix the auth bug") {
		t.Errorf("human-readable title should appear: %q", line)
	}
}

func TestFormatLine_TitleWithoutSpaceSuppressed(t *testing.T) {
	s := Session{
		SessionID:        "s1",
		AgentType:        "claude",
		Status:           "working",
		WorkingDirectory: "/home/user/proj",
		SessionTitle:     "xK9mP2abc",
		TmuxPane:         strPtr("%1"),
	}
	line := formatLine(s, "")
	if strings.Contains(line, "xK9mP2abc") {
		t.Errorf("identifier-like title should be suppressed: %q", line)
	}
}

func TestFormatLine_TitleWhitespaceOnlySuppressed(t *testing.T) {
	s := Session{
		SessionID:        "s1",
		AgentType:        "claude",
		Status:           "working",
		WorkingDirectory: "/home/user/proj",
		SessionTitle:     "   ",
		TmuxPane:         strPtr("%1"),
	}
	line := formatLine(s, "")
	if strings.Contains(line, `""`) {
		t.Errorf("whitespace-only title should not produce empty quoted string: %q", line)
	}
}
