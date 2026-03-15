package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeToml(t *testing.T, dir, content string) string {
	t.Helper()
	p := filepath.Join(dir, "config.toml")
	_ = os.WriteFile(p, []byte(content), 0644)
	return p
}

func TestCodexInstall_FreshNoNotify(t *testing.T) {
	dir := t.TempDir()
	scoutDir := t.TempDir()
	toml := writeToml(t, dir, "# Codex config\n[model]\nname = \"o4-mini\"\n")
	binPath := "/usr/local/bin/tmux-scout"

	result, err := codexInstall(toml, scoutDir, binPath)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "installed" {
		t.Errorf("got %q, want installed", result)
	}
	data, _ := os.ReadFile(toml)
	if !strings.Contains(string(data), codexHookIdentifierNew) {
		t.Error("new identifier not in config")
	}
}

func TestCodexInstall_ExistingNotify(t *testing.T) {
	dir := t.TempDir()
	scoutDir := t.TempDir()
	toml := writeToml(t, dir, `notify = [
  "slack-notify",
  "--channel",
  "dev"
]
`)
	binPath := "/usr/local/bin/tmux-scout"

	_, err := codexInstall(toml, scoutDir, binPath)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Original should be backed up
	backupPath := filepath.Join(scoutDir, "codex-original-notify.json")
	if _, err := os.Stat(backupPath); err != nil {
		t.Error("backup not created")
	}
	data, _ := os.ReadFile(toml)
	if !strings.Contains(string(data), codexHookIdentifierNew) {
		t.Error("new identifier not in config")
	}
}

func TestCodexUninstall_RestoresOriginal(t *testing.T) {
	dir := t.TempDir()
	scoutDir := t.TempDir()
	original := `notify = [
  "slack-notify",
  "--channel",
  "dev"
]
`
	toml := writeToml(t, dir, original)
	binPath := "/usr/local/bin/tmux-scout"

	_, _ = codexInstall(toml, scoutDir, binPath)
	_, err := codexUninstall(toml, scoutDir)
	if err != nil {
		t.Fatalf("uninstall error: %v", err)
	}
	data, _ := os.ReadFile(toml)
	if strings.Contains(string(data), codexHookIdentifierNew) {
		t.Error("new hook still present after uninstall")
	}
	if !strings.Contains(string(data), "slack-notify") {
		t.Error("original notify not restored")
	}
}

func TestCodexUninstall_NoBackup(t *testing.T) {
	dir := t.TempDir()
	scoutDir := t.TempDir()
	toml := writeToml(t, dir, "# config\n")
	binPath := "/usr/local/bin/tmux-scout"

	// Install first (no original notify)
	_, _ = codexInstall(toml, scoutDir, binPath)
	// Delete backup if created
	_ = os.Remove(filepath.Join(scoutDir, "codex-original-notify.json"))

	_, err := codexUninstall(toml, scoutDir)
	if err != nil {
		t.Fatalf("uninstall error: %v", err)
	}
	data, _ := os.ReadFile(toml)
	if strings.Contains(string(data), codexHookIdentifierNew) {
		t.Error("hook still present after uninstall")
	}
}
