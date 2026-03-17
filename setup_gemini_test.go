package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeGeminiSettings(t *testing.T, dir string, settings map[string]interface{}) string {
	t.Helper()
	p := filepath.Join(dir, "settings.json")
	data, _ := json.MarshalIndent(settings, "", "  ")
	_ = os.WriteFile(p, data, 0644)
	return p
}

func TestGeminiInstall_MissingFile_Skipped(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json") // does not exist
	result, err := geminiInstall(settingsPath, "/usr/bin/tmux-scout")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "skipped" {
		t.Errorf("expected skipped, got %q", result)
	}
}

func TestGeminiInstall_FreshInstall(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeGeminiSettings(t, dir, map[string]interface{}{})
	binPath := "/usr/local/bin/tmux-scout"

	result, err := geminiInstall(settingsPath, binPath)
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
	for _, event := range geminiHookEvents {
		if _, ok := hooks[event]; !ok {
			t.Errorf("hook for event %s not installed", event)
		}
	}

	raw := string(data)
	if strings.Count(raw, `"name": "tmux-scout"`) != len(geminiHookEvents) {
		t.Errorf("expected %d hook entries with name=tmux-scout", len(geminiHookEvents))
	}
	if !strings.Contains(raw, `"timeout": 5000`) {
		t.Error("timeout 5000 not found")
	}
	if !strings.Contains(raw, `"matcher": "*"`) {
		t.Error(`matcher "*" not found`)
	}
	expectedCmd := binPath + " hook gemini"
	if !strings.Contains(raw, expectedCmd) {
		t.Errorf("expected command %q not found", expectedCmd)
	}
}

func TestGeminiInstall_Idempotent(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeGeminiSettings(t, dir, map[string]interface{}{})
	binPath := "/usr/local/bin/tmux-scout"

	_, _ = geminiInstall(settingsPath, binPath)
	_, err := geminiInstall(settingsPath, binPath)
	if err != nil {
		t.Fatalf("second install error: %v", err)
	}

	data, _ := os.ReadFile(settingsPath)
	count := strings.Count(string(data), geminiHookIdentifier)
	if count != len(geminiHookEvents) {
		t.Errorf("idempotent install: expected %d entries, got %d", len(geminiHookEvents), count)
	}
}

func TestGeminiInstall_UpdatesStaleCommand(t *testing.T) {
	dir := t.TempDir()
	staleHooks := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"matcher": "*",
					"hooks": []interface{}{
						map[string]interface{}{
							"name":    "tmux-scout",
							"type":    "command",
							"command": "/old/path/tmux-scout hook gemini",
							"timeout": 5000,
						},
					},
				},
			},
		},
	}
	settingsPath := writeGeminiSettings(t, dir, staleHooks)
	binPath := "/new/path/tmux-scout"

	_, err := geminiInstall(settingsPath, binPath)
	if err != nil {
		t.Fatalf("install error: %v", err)
	}

	data, _ := os.ReadFile(settingsPath)
	raw := string(data)
	if strings.Contains(raw, "/old/path") {
		t.Error("stale path still present after update")
	}
	if !strings.Contains(raw, "/new/path") {
		t.Error("new path not written")
	}
}

func TestGeminiInstall_PreservesUnrelatedKeys(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeGeminiSettings(t, dir, map[string]interface{}{
		"theme":            "dark",
		"someOtherSetting": true,
	})

	_, err := geminiInstall(settingsPath, "/usr/bin/tmux-scout")
	if err != nil {
		t.Fatalf("install error: %v", err)
	}

	data, _ := os.ReadFile(settingsPath)
	raw := string(data)
	if !strings.Contains(raw, `"theme"`) {
		t.Error("unrelated key 'theme' was removed")
	}
	if !strings.Contains(raw, `"someOtherSetting"`) {
		t.Error("unrelated key 'someOtherSetting' was removed")
	}
}

func TestGeminiUninstall(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeGeminiSettings(t, dir, map[string]interface{}{})
	binPath := "/usr/local/bin/tmux-scout"

	_, _ = geminiInstall(settingsPath, binPath)
	result, err := geminiUninstall(settingsPath)
	if err != nil {
		t.Fatalf("uninstall error: %v", err)
	}
	if result != "removed" {
		t.Errorf("expected removed, got %q", result)
	}

	data, _ := os.ReadFile(settingsPath)
	if strings.Contains(string(data), geminiHookIdentifier) {
		t.Error("hook still present after uninstall")
	}
}

func TestGeminiUninstall_NotInstalled_ReturnsNotFound(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeGeminiSettings(t, dir, map[string]interface{}{}) // no hooks key

	result, err := geminiUninstall(settingsPath)
	if err != nil {
		t.Fatalf("uninstall error: %v", err)
	}
	if result != "not_found" {
		t.Errorf("expected not_found, got %q", result)
	}
}

func TestGeminiStatus_BeforeAndAfterInstall(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeGeminiSettings(t, dir, map[string]interface{}{})
	binPath := "/usr/local/bin/tmux-scout"

	n, _ := geminiStatus(settingsPath)
	if n != 0 {
		t.Errorf("before install: got %d, want 0", n)
	}

	_, _ = geminiInstall(settingsPath, binPath)
	n, _ = geminiStatus(settingsPath)
	if n != len(geminiHookEvents) {
		t.Errorf("after install: got %d, want %d", n, len(geminiHookEvents))
	}
}

func TestGeminiStatus_MissingFile(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json") // does not exist

	n, err := geminiStatus(settingsPath)
	if err != nil {
		t.Errorf("expected nil error for missing file, got %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}
