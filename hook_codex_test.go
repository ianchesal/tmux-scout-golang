package main

import (
	"encoding/json"
	"testing"
)

func TestExtractCodexTitle_FromInputMessages_String(t *testing.T) {
	msgs := []interface{}{
		map[string]interface{}{
			"role":    "user",
			"content": "## My request for Codex:\nFix the login bug",
		},
	}
	data, _ := json.Marshal(msgs)
	got := extractCodexTitle(json.RawMessage(data), "")
	if got != "Fix the login bug" {
		t.Errorf("got %q", got)
	}
}

func TestExtractCodexTitle_FromInputMessages_Array(t *testing.T) {
	msgs := []interface{}{
		map[string]interface{}{
			"role": "user",
			"content": []interface{}{
				map[string]interface{}{
					"type": "input_text",
					"text": "## My request for Codex:\nRefactor the auth module",
				},
			},
		},
	}
	data, _ := json.Marshal(msgs)
	got := extractCodexTitle(json.RawMessage(data), "")
	if got != "Refactor the auth module" {
		t.Errorf("got %q", got)
	}
}

func TestExtractCodexTitle_NoMarker(t *testing.T) {
	msgs := []interface{}{
		map[string]interface{}{
			"role":    "user",
			"content": "Just a plain request without marker",
		},
	}
	data, _ := json.Marshal(msgs)
	got := extractCodexTitle(json.RawMessage(data), "")
	if got != "Just a plain request without marker" {
		t.Errorf("got %q", got)
	}
}

func TestExtractCodexTitle_Fallback(t *testing.T) {
	// empty input messages, fallback to last assistant message
	got := extractCodexTitle(nil, "The assistant response here")
	if got != "The assistant response here" {
		t.Errorf("got %q", got)
	}
}

func TestExtractCodexTitle_TruncatesAt100(t *testing.T) {
	long := "## My request for Codex:\n" + string(make([]byte, 150))
	msgs := []interface{}{
		map[string]interface{}{"role": "user", "content": long},
	}
	data, _ := json.Marshal(msgs)
	got := extractCodexTitle(json.RawMessage(data), "")
	if len(got) > 100 {
		t.Errorf("title too long: %d chars", len(got))
	}
}
