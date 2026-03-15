package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCanUseShellFallback_Codex(t *testing.T) {
	s := Session{AgentType: "codex", Status: "idle"}
	if !canUseShellFallback(s) {
		t.Error("codex should always use shell fallback")
	}
}

func TestCanUseShellFallback_ClaudeWorking(t *testing.T) {
	s := Session{AgentType: "claude", Status: "working"}
	if !canUseShellFallback(s) {
		t.Error("working claude should use shell fallback")
	}
}

func TestCanUseShellFallback_ClaudeIdle(t *testing.T) {
	s := Session{AgentType: "claude", Status: "idle"}
	if canUseShellFallback(s) {
		t.Error("idle claude without lastEvent should not use shell fallback")
	}
}

func TestCanUseShellFallback_ClaudeWithPendingTool(t *testing.T) {
	s := Session{AgentType: "claude", Status: "completed", PendingToolUse: &PendingToolUse{Tool: "Bash"}}
	if !canUseShellFallback(s) {
		t.Error("session with pendingToolUse should use shell fallback")
	}
}

func TestIsShellCommand(t *testing.T) {
	shells := []string{"bash", "zsh", "sh", "fish", "dash", "ksh", "tcsh", "csh", "nu"}
	for _, s := range shells {
		if !isShellCommand(s) {
			t.Errorf("%s should be a shell command", s)
		}
	}
	nonShells := []string{"node", "python", "claude", "codex", "vim"}
	for _, s := range nonShells {
		if isShellCommand(s) {
			t.Errorf("%s should not be a shell command", s)
		}
	}
}

func TestReadCodexJsonl_Completed(t *testing.T) {
	dir := t.TempDir()
	lines := []string{
		`{"type":"event_msg","payload":{"type":"user_message","message":"Do the thing"},"timestamp":"2024-01-01T00:00:00Z"}`,
		`{"type":"event_msg","payload":{"type":"task_complete"},"timestamp":"2024-01-01T00:01:00Z"}`,
	}
	f := filepath.Join(dir, "test.jsonl")
	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	_ = os.WriteFile(f, []byte(content), 0644)

	result := readCodexJsonl(f)
	if result == nil {
		t.Fatal("nil result")
	}
	if result.Status != "completed" {
		t.Errorf("status = %q, want completed", result.Status)
	}
	if result.Title != "Do the thing" {
		t.Errorf("title = %q", result.Title)
	}
}

func TestReadCodexJsonl_Working(t *testing.T) {
	dir := t.TempDir()
	// User message but no task_complete yet
	content := `{"type":"event_msg","payload":{"type":"user_message","message":"Still working"},"timestamp":"2024-01-01T00:00:00Z"}` + "\n"
	f := filepath.Join(dir, "test.jsonl")
	_ = os.WriteFile(f, []byte(content), 0644)

	result := readCodexJsonl(f)
	if result == nil {
		t.Fatal("nil result")
	}
	if result.Status != "working" {
		t.Errorf("status = %q, want working", result.Status)
	}
}

func TestReadCodexJsonl_PlanWaiting(t *testing.T) {
	dir := t.TempDir()
	content := fmt.Sprintf(
		"%s\n%s\n",
		`{"type":"event_msg","payload":{"type":"user_message","message":"Please plan"},"timestamp":"2024-01-01T00:00:00Z"}`,
		`{"type":"event_msg","payload":{"type":"item_completed","item":{"type":"Plan"}},"timestamp":"2024-01-01T00:00:01Z"}`,
	) + `{"type":"event_msg","payload":{"type":"task_complete"},"timestamp":"2024-01-01T00:00:02Z"}` + "\n"
	f := filepath.Join(dir, "test.jsonl")
	_ = os.WriteFile(f, []byte(content), 0644)

	result := readCodexJsonl(f)
	if result == nil {
		t.Fatal("nil result")
	}
	if result.Status != "needsAttention" {
		t.Errorf("status = %q, want needsAttention", result.Status)
	}
}

func TestReadCodexJsonl_MarkerStrip(t *testing.T) {
	dir := t.TempDir()
	content := `{"type":"event_msg","payload":{"type":"user_message","message":"context here\n## My request for Codex:\nActual task"},"timestamp":"2024-01-01T00:00:00Z"}` + "\n"
	f := filepath.Join(dir, "test.jsonl")
	_ = os.WriteFile(f, []byte(content), 0644)

	result := readCodexJsonl(f)
	if result == nil {
		t.Fatal("nil result")
	}
	if result.Title != "Actual task" {
		t.Errorf("title = %q, want 'Actual task'", result.Title)
	}
}
