package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestIsChromeLine_BorderLine(t *testing.T) {
	if !isChromeLine("  ─────────────────────────────  ") {
		t.Error("line of dashes should be chrome")
	}
}

func TestIsChromeLine_MixedBorderHeavy(t *testing.T) {
	if !isChromeLine("│ some text ──────────────────────────────────────────") {
		t.Error("line mostly box-drawing chars should be chrome")
	}
}

func TestIsChromeLine_NormalText(t *testing.T) {
	if isChromeLine("The authentication middleware needs refactoring.") {
		t.Error("normal text line should not be chrome")
	}
}

func TestIsChromeLine_Empty(t *testing.T) {
	if isChromeLine("") {
		t.Error("empty line should not be chrome")
	}
}

func TestIsChromeLine_WhitespaceOnly(t *testing.T) {
	if isChromeLine("   ") {
		t.Error("whitespace-only line should not be chrome")
	}
}

func TestFilterChromeLines_RemovesBorders(t *testing.T) {
	input := []string{
		"Normal output line",
		"─────────────────────",
		"Another real line",
		"│ tool: bash ──────────────────────────────────────",
		"Result of command",
	}
	got := filterChromeLines(input)
	for _, line := range got {
		if isChromeLine(line) {
			t.Errorf("chrome line leaked through: %q", line)
		}
	}
	if len(got) != 3 {
		t.Errorf("expected 3 non-chrome lines, got %d: %v", len(got), got)
	}
}

func TestFilterChromeLines_CollapsesBlankRuns(t *testing.T) {
	input := []string{
		"Line one",
		"",
		"",
		"",
		"Line two",
	}
	got := filterChromeLines(input)
	if len(got) != 3 {
		t.Errorf("expected 3 lines (line one, blank, line two), got %d: %v", len(got), got)
	}
	blanks := 0
	for _, l := range got {
		if strings.TrimSpace(l) == "" {
			blanks++
		}
	}
	if blanks > 1 {
		t.Errorf("blank runs should collapse to one blank, got %d blanks in %v", blanks, got)
	}
}

func TestPrintPreviewHeader_BasicFields(t *testing.T) {
	pane := "%1"
	s := Session{
		SessionID:        "abc-123",
		AgentType:        "claude",
		Status:           "working",
		WorkingDirectory: "/home/user/project",
		TmuxPane:         &pane,
	}
	var buf strings.Builder
	// Temporarily redirect stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	printPreviewHeader(s)
	w.Close()
	io.Copy(&buf, r)
	os.Stdout = old

	out := buf.String()
	if !strings.Contains(out, "claude") {
		t.Errorf("expected agent 'claude' in output: %q", out)
	}
	if !strings.Contains(out, "working") {
		t.Errorf("expected status 'working' in output: %q", out)
	}
	if !strings.Contains(out, "/home/user/project") {
		t.Errorf("expected working dir in output: %q", out)
	}
	if !strings.Contains(out, "abc-123") {
		t.Errorf("expected session ID in output: %q", out)
	}
}

func TestPrintPreviewHeader_NeedsAttention(t *testing.T) {
	reason := "ExitPlanMode"
	s := Session{
		SessionID:        "xyz-456",
		AgentType:        "claude",
		Status:           "working",
		WorkingDirectory: "/home/user/proj",
		NeedsAttention:   &reason,
	}
	var buf strings.Builder
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	printPreviewHeader(s)
	w.Close()
	io.Copy(&buf, r)
	os.Stdout = old

	out := buf.String()
	if !strings.Contains(out, "working") {
		t.Errorf("expected underlying status 'working' in output: %q", out)
	}
	if !strings.Contains(out, "ExitPlanMode") {
		t.Errorf("expected NeedsAttention reason in output: %q", out)
	}
}

func TestPrintPreviewHeader_EmptyWorkingDir(t *testing.T) {
	s := Session{
		SessionID: "no-dir",
		AgentType: "codex",
		Status:    "idle",
	}
	var buf strings.Builder
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	printPreviewHeader(s)
	w.Close()
	io.Copy(&buf, r)
	os.Stdout = old

	out := buf.String()
	if !strings.Contains(out, "?") {
		t.Errorf("expected '?' fallback for empty working dir: %q", out)
	}
}

func TestLastN_FewerThanN(t *testing.T) {
	input := []string{"a", "b", "c"}
	got := lastN(input, 25)
	if len(got) != 3 {
		t.Errorf("expected 3, got %d", len(got))
	}
}

func TestLastN_MoreThanN(t *testing.T) {
	input := make([]string, 30)
	for i := range input {
		input[i] = "line"
	}
	got := lastN(input, 25)
	if len(got) != 25 {
		t.Errorf("expected 25, got %d", len(got))
	}
}
