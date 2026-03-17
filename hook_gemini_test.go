package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func newGeminiSession() Session {
	return Session{
		SessionID: "test-session-id",
		AgentType: "gemini",
		Status:    "idle",
		StartedAt: 1000,
	}
}

func TestGeminiSessionStart_Startup(t *testing.T) {
	s := newGeminiSession()
	p := GeminiHookPayload{
		HookEventName: "SessionStart",
		SessionID:     "test-session-id",
		CWD:           "/home/user/project",
		Source:        "startup",
	}
	got := applyGeminiHook(s, p, "%1", 42, 2000)
	if got.Status != "idle" {
		t.Errorf("status: got %q, want idle", got.Status)
	}
	if got.WorkingDirectory != "/home/user/project" {
		t.Errorf("WorkingDirectory: got %q", got.WorkingDirectory)
	}
	if got.TmuxPane == nil || *got.TmuxPane != "%1" {
		t.Errorf("TmuxPane: got %v", got.TmuxPane)
	}
	if got.PID == nil || *got.PID != 42 {
		t.Errorf("PID: got %v", got.PID)
	}
	if got.NeedsAttention != nil {
		t.Errorf("NeedsAttention should be nil")
	}
	if got.PendingToolUse != nil {
		t.Errorf("PendingToolUse should be nil")
	}
	if got.LastEvent == nil || got.LastEvent.Type != "session_start" || got.LastEvent.Details != "startup" {
		t.Errorf("LastEvent: got %v", got.LastEvent)
	}
	if got.LastUpdated != 2000 {
		t.Errorf("LastUpdated: got %d, want 2000", got.LastUpdated)
	}
	// StartedAt set by caller (runHookGemini new-session init) — applyGeminiHook must not clear it
	if got.StartedAt != 1000 {
		t.Errorf("StartedAt: got %d, want 1000 (must not be reset by applyGeminiHook)", got.StartedAt)
	}
}

func TestGeminiSessionStart_Resume(t *testing.T) {
	s := newGeminiSession()
	p := GeminiHookPayload{
		HookEventName: "SessionStart",
		SessionID:     "test-session-id",
		CWD:           "/work",
		Source:        "resume",
	}
	got := applyGeminiHook(s, p, "%2", 99, 3000)
	if got.Status != "idle" {
		t.Errorf("status: got %q", got.Status)
	}
	if got.WorkingDirectory != "/work" {
		t.Errorf("WorkingDirectory: got %q", got.WorkingDirectory)
	}
	if got.LastEvent == nil || got.LastEvent.Details != "resume" {
		t.Errorf("LastEvent.Details: got %v", got.LastEvent)
	}
}

func TestGeminiSessionStart_ClearSource(t *testing.T) {
	s := newGeminiSession()
	p := GeminiHookPayload{
		HookEventName: "SessionStart",
		SessionID:     "test-session-id",
		CWD:           "/work",
		Source:        "clear",
	}
	got := applyGeminiHook(s, p, "%3", 77, 4000)
	if got.Status != "idle" {
		t.Errorf("status: got %q", got.Status)
	}
	if got.LastEvent == nil || got.LastEvent.Details != "clear" {
		t.Errorf("LastEvent.Details for clear source: got %v", got.LastEvent)
	}
}

func TestGeminiSessionStart_EmptyPane(t *testing.T) {
	s := newGeminiSession()
	p := GeminiHookPayload{
		HookEventName: "SessionStart",
		SessionID:     "test-session-id",
		CWD:           "/work",
	}
	got := applyGeminiHook(s, p, "", 99, 3000)
	if got.TmuxPane != nil {
		t.Errorf("TmuxPane should be nil when pane is empty, got %v", got.TmuxPane)
	}
}

func TestGeminiBeforeAgent_SetsWorkingAndTitle(t *testing.T) {
	s := newGeminiSession()
	p := GeminiHookPayload{
		HookEventName: "BeforeAgent",
		SessionID:     "test-session-id",
		Prompt:        "Fix the login bug\nsome more context",
	}
	got := applyGeminiHook(s, p, "", 0, 5000)
	if got.Status != "working" {
		t.Errorf("status: got %q, want working", got.Status)
	}
	if got.SessionTitle != "Fix the login bug" {
		t.Errorf("SessionTitle: got %q", got.SessionTitle)
	}
	if got.LastEvent == nil || got.LastEvent.Type != "prompt_submit" || got.LastEvent.Details != "Fix the login bug" {
		t.Errorf("LastEvent: %v", got.LastEvent)
	}
}

func TestGeminiBeforeAgent_TruncatesAt100(t *testing.T) {
	s := newGeminiSession()
	long := strings.Repeat("a", 150)
	p := GeminiHookPayload{
		HookEventName: "BeforeAgent",
		SessionID:     "test-session-id",
		Prompt:        long,
	}
	got := applyGeminiHook(s, p, "", 0, 5000)
	if len(got.SessionTitle) > 100 {
		t.Errorf("SessionTitle too long: %d chars", len(got.SessionTitle))
	}
	if len(got.SessionTitle) == 0 {
		t.Errorf("SessionTitle should be non-empty")
	}
}

func TestGeminiBeforeAgent_EmptyPromptNoTitleChange(t *testing.T) {
	s := newGeminiSession()
	s.SessionTitle = "previous title"
	p := GeminiHookPayload{
		HookEventName: "BeforeAgent",
		SessionID:     "test-session-id",
		Prompt:        "",
	}
	got := applyGeminiHook(s, p, "", 0, 5000)
	if got.Status != "working" {
		t.Errorf("status: got %q", got.Status)
	}
	if got.SessionTitle != "previous title" {
		t.Errorf("SessionTitle should be unchanged, got %q", got.SessionTitle)
	}
	if got.LastEvent == nil || got.LastEvent.Type != "prompt_submit" {
		t.Errorf("LastEvent should be set to prompt_submit even on empty prompt, got %v", got.LastEvent)
	}
}

func TestGeminiBeforeTool(t *testing.T) {
	s := newGeminiSession()
	input, _ := json.Marshal(map[string]string{"file_path": "/home/user/main.go"})
	p := GeminiHookPayload{
		HookEventName: "BeforeTool",
		SessionID:     "test-session-id",
		ToolName:      "Write",
		ToolInput:     json.RawMessage(input),
	}
	got := applyGeminiHook(s, p, "", 0, 6000)
	if got.Status != "working" {
		t.Errorf("status: got %q", got.Status)
	}
	if got.PendingToolUse == nil {
		t.Fatal("PendingToolUse should not be nil")
	}
	if got.PendingToolUse.Tool != "Write" {
		t.Errorf("PendingToolUse.Tool: got %q", got.PendingToolUse.Tool)
	}
	if got.LastEvent == nil || got.LastEvent.Type != "tool_use" {
		t.Errorf("LastEvent: %v", got.LastEvent)
	}
}

func TestGeminiAfterTool_ClearsPendingToolUse(t *testing.T) {
	s := newGeminiSession()
	s.PendingToolUse = &PendingToolUse{Tool: "Write", Details: "Write: main.go", Timestamp: 5000}
	prevEvent := &LastEvent{Type: "tool_use", Timestamp: 5000}
	s.LastEvent = prevEvent
	p := GeminiHookPayload{
		HookEventName: "AfterTool",
		SessionID:     "test-session-id",
		ToolName:      "Write",
	}
	got := applyGeminiHook(s, p, "", 0, 7000)
	if got.Status != "working" {
		t.Errorf("status: got %q", got.Status)
	}
	if got.PendingToolUse != nil {
		t.Errorf("PendingToolUse should be nil after AfterTool")
	}
	// LastEvent must not be assigned a new value
	if got.LastEvent == nil || got.LastEvent.Type != "tool_use" {
		t.Errorf("LastEvent should remain as tool_use after AfterTool, got %v", got.LastEvent)
	}
}

func TestGeminiAfterTool_NoopWhenNilPendingToolUse(t *testing.T) {
	s := newGeminiSession()
	p := GeminiHookPayload{
		HookEventName: "AfterTool",
		SessionID:     "test-session-id",
	}
	got := applyGeminiHook(s, p, "", 0, 7000)
	if got.Status != "working" {
		t.Errorf("status: got %q", got.Status)
	}
	if got.PendingToolUse != nil {
		t.Errorf("PendingToolUse should remain nil")
	}
}

func TestGeminiAfterAgent(t *testing.T) {
	s := newGeminiSession()
	s.Status = "working"
	attn := "needs attention"
	s.NeedsAttention = &attn
	s.PendingToolUse = &PendingToolUse{Tool: "Write"}
	p := GeminiHookPayload{
		HookEventName: "AfterAgent",
		SessionID:     "test-session-id",
	}
	got := applyGeminiHook(s, p, "", 0, 8000)
	if got.Status != "completed" {
		t.Errorf("status: got %q", got.Status)
	}
	if got.PendingToolUse != nil {
		t.Errorf("PendingToolUse should be nil")
	}
	if got.NeedsAttention != nil {
		t.Errorf("NeedsAttention should be nil")
	}
	if got.LastEvent == nil || got.LastEvent.Type != "stop" {
		t.Errorf("LastEvent: %v", got.LastEvent)
	}
}

func TestGeminiSessionEnd_Clear(t *testing.T) {
	s := newGeminiSession()
	s.Status = "working"
	s.SessionTitle = "old title"
	endedAt := int64(5000)
	s.EndedAt = &endedAt
	p := GeminiHookPayload{
		HookEventName: "SessionEnd",
		SessionID:     "test-session-id",
		Reason:        "clear",
	}
	got := applyGeminiHook(s, p, "", 0, 9000)
	if got.Status != "idle" {
		t.Errorf("status: got %q", got.Status)
	}
	if got.EndedAt != nil {
		t.Errorf("EndedAt should be nil for clear reason, got %v", got.EndedAt)
	}
	if got.SessionTitle != "" {
		t.Errorf("SessionTitle should be cleared, got %q", got.SessionTitle)
	}
	if got.LastEvent == nil || got.LastEvent.Details != "clear" {
		t.Errorf("LastEvent: %v", got.LastEvent)
	}
	if got.NeedsAttention != nil {
		t.Errorf("NeedsAttention should be nil after SessionEnd")
	}
	if got.PendingToolUse != nil {
		t.Errorf("PendingToolUse should be nil after SessionEnd")
	}
}

func TestGeminiSessionEnd_Exit(t *testing.T) {
	s := newGeminiSession()
	p := GeminiHookPayload{
		HookEventName: "SessionEnd",
		SessionID:     "test-session-id",
		Reason:        "exit",
	}
	got := applyGeminiHook(s, p, "", 0, 9000)
	if got.Status != "idle" {
		t.Errorf("status: got %q", got.Status)
	}
	if got.EndedAt == nil {
		t.Fatal("EndedAt should be set for exit reason")
	}
	if *got.EndedAt != 9000 {
		t.Errorf("EndedAt: got %d, want 9000", *got.EndedAt)
	}
	if got.NeedsAttention != nil {
		t.Errorf("NeedsAttention should be nil after SessionEnd")
	}
	if got.PendingToolUse != nil {
		t.Errorf("PendingToolUse should be nil after SessionEnd")
	}
}

func TestGeminiSessionEnd_OtherReasons(t *testing.T) {
	for _, reason := range []string{"logout", "prompt_input_exit", "other"} {
		t.Run(reason, func(t *testing.T) {
			s := newGeminiSession()
			p := GeminiHookPayload{
				HookEventName: "SessionEnd",
				SessionID:     "test-session-id",
				Reason:        reason,
			}
			got := applyGeminiHook(s, p, "", 0, 9000)
			if got.EndedAt == nil {
				t.Errorf("EndedAt should be set for reason %q", reason)
			}
		})
	}
}

func TestGeminiNotification_ToolPermission(t *testing.T) {
	s := newGeminiSession()
	p := GeminiHookPayload{
		HookEventName:    "Notification",
		SessionID:        "test-session-id",
		NotificationType: "ToolPermission",
		Message:          "Gemini wants to use write_file",
	}
	got := applyGeminiHook(s, p, "", 0, 10000)
	if got.NeedsAttention == nil {
		t.Fatal("NeedsAttention should be set")
	}
	if *got.NeedsAttention != "Gemini wants to use write_file" {
		t.Errorf("NeedsAttention: got %q", *got.NeedsAttention)
	}
	if got.LastEvent != nil {
		t.Errorf("Notification should not update LastEvent, got %v", got.LastEvent)
	}
}

func TestGeminiNotification_UnknownType_Ignored(t *testing.T) {
	s := newGeminiSession()
	p := GeminiHookPayload{
		HookEventName:    "Notification",
		SessionID:        "test-session-id",
		NotificationType: "SomethingElse",
		Message:          "whatever",
	}
	got := applyGeminiHook(s, p, "", 0, 10000)
	if got.NeedsAttention != nil {
		t.Errorf("NeedsAttention should remain nil, got %v", got.NeedsAttention)
	}
}

func TestGeminiUnknownEvent_NoStateChange(t *testing.T) {
	s := newGeminiSession()
	s.Status = "working"
	p := GeminiHookPayload{
		HookEventName: "SomeUnknownEvent",
		SessionID:     "test-session-id",
	}
	got := applyGeminiHook(s, p, "", 0, 11000)
	if got.Status != "working" {
		t.Errorf("status should be unchanged, got %q", got.Status)
	}
}

func TestGeminiStaleStateReset(t *testing.T) {
	endedAt := int64(1000)
	s := newGeminiSession()
	s.EndedAt = &endedAt
	s.CrashReason = "crashed"
	s.Status = "idle"
	p := GeminiHookPayload{
		HookEventName: "BeforeAgent",
		SessionID:     "test-session-id",
		Prompt:        "do stuff",
	}
	got := applyGeminiHook(s, p, "", 0, 5000)
	if got.EndedAt != nil {
		t.Errorf("EndedAt should be cleared by pre-switch reset")
	}
	if got.CrashReason != "" {
		t.Errorf("CrashReason should be cleared")
	}
	if got.Status != "working" {
		t.Errorf("Status should be working after BeforeAgent, got %q", got.Status)
	}
}

func TestGeminiSessionStart_StaleStateReset(t *testing.T) {
	endedAt := int64(1000)
	s := newGeminiSession()
	s.EndedAt = &endedAt
	s.CrashReason = "crashed"
	p := GeminiHookPayload{
		HookEventName: "SessionStart",
		SessionID:     "test-session-id",
		CWD:           "/work",
	}
	got := applyGeminiHook(s, p, "", 0, 5000)
	if got.EndedAt != nil {
		t.Errorf("EndedAt should be cleared by pre-switch reset on SessionStart")
	}
	if got.CrashReason != "" {
		t.Errorf("CrashReason should be cleared by pre-switch reset on SessionStart")
	}
	if got.Status != "idle" {
		t.Errorf("Status should be idle after SessionStart, got %q", got.Status)
	}
}
