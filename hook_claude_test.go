package main

import (
	"encoding/json"
	"testing"
)

func TestGetToolDetails_Command(t *testing.T) {
	input, _ := json.Marshal(map[string]string{"command": "echo hello world this is a very long command that exceeds fifty characters limit"})
	got := getToolDetails("Bash", json.RawMessage(input))
	want := "Bash: echo hello world this is a very long command"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGetToolDetails_FilePath(t *testing.T) {
	input, _ := json.Marshal(map[string]string{"file_path": "/home/user/project/src/main.go"})
	got := getToolDetails("Read", json.RawMessage(input))
	if got != "Read: main.go" {
		t.Errorf("got %q, want %q", got, "Read: main.go")
	}
}

func TestGetToolDetails_PatternAndPath(t *testing.T) {
	input, _ := json.Marshal(map[string]string{"pattern": "func main", "path": "/home/user/project"})
	got := getToolDetails("Grep", json.RawMessage(input))
	if got != "Grep: func main in project" {
		t.Errorf("got %q", got)
	}
}

func TestGetToolDetails_PatternOnly(t *testing.T) {
	input, _ := json.Marshal(map[string]string{"pattern": "func main"})
	got := getToolDetails("Grep", json.RawMessage(input))
	if got != "Grep: func main" {
		t.Errorf("got %q", got)
	}
}

func TestGetToolDetails_URL(t *testing.T) {
	input, _ := json.Marshal(map[string]string{"url": "https://example.com/api"})
	got := getToolDetails("WebFetch", json.RawMessage(input))
	if got != "WebFetch: https://example.com/api" {
		t.Errorf("got %q", got)
	}
}

func TestGetToolDetails_Fallback(t *testing.T) {
	input, _ := json.Marshal(map[string]string{"something_else": "some value here"})
	got := getToolDetails("MyTool", json.RawMessage(input))
	if got != "MyTool: some value here" {
		t.Errorf("got %q", got)
	}
}

func TestGetToolDetails_NoInput(t *testing.T) {
	got := getToolDetails("Bash", nil)
	if got != "Bash" {
		t.Errorf("got %q, want %q", got, "Bash")
	}
}

func TestCleanPrompt_StripSystemInstruction(t *testing.T) {
	input := "<system-instruction>do this</system-instruction>\nActual user prompt"
	got := cleanPrompt(input)
	if got != "Actual user prompt" {
		t.Errorf("got %q", got)
	}
}

func TestCleanPrompt_StripSystemReminder(t *testing.T) {
	input := "Some text\n<system-reminder>reminder content</system-reminder>\nMore text"
	got := cleanPrompt(input)
	if got != "Some text\nMore text" {
		t.Errorf("got %q", got)
	}
}

func TestCleanPrompt_StripLeadingXMLBlock(t *testing.T) {
	input := "<context>some context data</context>\nReal question"
	got := cleanPrompt(input)
	if got != "Real question" {
		t.Errorf("got %q", got)
	}
}

func TestCleanPrompt_NoXML(t *testing.T) {
	input := "plain text prompt"
	got := cleanPrompt(input)
	if got != "plain text prompt" {
		t.Errorf("got %q", got)
	}
}
