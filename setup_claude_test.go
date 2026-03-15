package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSettings(t *testing.T, dir string, settings map[string]interface{}) string {
	t.Helper()
	p := filepath.Join(dir, "settings.json")
	data, _ := json.MarshalIndent(settings, "", "  ")
	_ = os.WriteFile(p, data, 0644)
	return p
}

func TestClaudeInstall_FreshInstall(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeSettings(t, dir, map[string]interface{}{})
	binPath := "/usr/local/bin/tmux-scout"

	result, err := claudeInstall(settingsPath, binPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "skipped" {
		t.Fatal("should not be skipped")
	}

	data, _ := os.ReadFile(settingsPath)
	var s map[string]interface{}
	_ = json.Unmarshal(data, &s)
	hooks, ok := s["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("hooks not written")
	}
	for _, event := range claudeHookEvents {
		if _, ok := hooks[event]; !ok {
			t.Errorf("hook for %s not installed", event)
		}
	}
}

func TestClaudeInstall_Idempotent(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeSettings(t, dir, map[string]interface{}{})
	binPath := "/usr/local/bin/tmux-scout"

	_, _ = claudeInstall(settingsPath, binPath)
	_, err := claudeInstall(settingsPath, binPath)
	if err != nil {
		t.Fatalf("second install error: %v", err)
	}
	// Verify no duplicates
	data, _ := os.ReadFile(settingsPath)
	var s map[string]interface{}
	_ = json.Unmarshal(data, &s)
	// Count hook entries for SessionStart
	count := strings.Count(string(data), claudeHookIdentifierNew)
	// Should be exactly 6 (one per event)
	if count != 6 {
		t.Errorf("expected 6 hook entries, got %d", count)
	}
}

func TestClaudeUninstall(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeSettings(t, dir, map[string]interface{}{})
	binPath := "/usr/local/bin/tmux-scout"

	_, _ = claudeInstall(settingsPath, binPath)
	_, err := claudeUninstall(settingsPath)
	if err != nil {
		t.Fatalf("uninstall error: %v", err)
	}

	data, _ := os.ReadFile(settingsPath)
	if strings.Contains(string(data), claudeHookIdentifierNew) {
		t.Error("hook still present after uninstall")
	}
}

func TestClaudeStatus(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeSettings(t, dir, map[string]interface{}{})
	binPath := "/usr/local/bin/tmux-scout"

	installed, _ := claudeStatus(settingsPath)
	if installed != 0 {
		t.Errorf("before install: got %d, want 0", installed)
	}

	_, _ = claudeInstall(settingsPath, binPath)
	installed, _ = claudeStatus(settingsPath)
	if installed != len(claudeHookEvents) {
		t.Errorf("after install: got %d, want %d", installed, len(claudeHookEvents))
	}
}
