package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateData_NothingToMigrate(t *testing.T) {
	src := filepath.Join(t.TempDir(), "old-tmux-scout")
	dst := filepath.Join(t.TempDir(), "new-tmux-scout")
	msg, err := migrateData(src, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(msg, "nothing to migrate") {
		t.Errorf("got %q, want 'nothing to migrate'", msg)
	}
}

func TestMigrateData_DestinationExists(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	// Write a file to src so it exists
	os.WriteFile(filepath.Join(src, "status.json"), []byte("{}"), 0644)
	msg, err := migrateData(src, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(msg, "already exists") {
		t.Errorf("got %q, want 'already exists'", msg)
	}
}

func TestMigrateBinary_LocalBinNotFound(t *testing.T) {
	placed, msg, err := migrateBinary(filepath.Join(t.TempDir(), "nonexistent"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if placed {
		t.Error("placed should be false when localBin does not exist")
	}
	if !strings.Contains(msg, "not found") {
		t.Errorf("got %q, want message containing 'not found'", msg)
	}
}

func TestMigrateBinary_SymlinksBinary(t *testing.T) {
	localBin := t.TempDir()
	placed, msg, err := migrateBinary(localBin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !placed {
		t.Error("placed should be true when localBin exists")
	}
	if !strings.Contains(msg, "symlinked") {
		t.Errorf("got %q, want message containing 'symlinked'", msg)
	}
	dst := filepath.Join(localBin, "tmux-scout")
	// Verify it's a symlink, not a copy
	fi, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("symlink not found at destination: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("destination should be a symlink")
	}
	// Verify the symlink target is executable
	target, err := os.Readlink(dst)
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("symlink target not found: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("symlink target is not executable")
	}
}

func TestMigrateData_MovesTree(t *testing.T) {
	src := t.TempDir()
	dstParent := t.TempDir()
	dst := filepath.Join(dstParent, "tmux-scout")

	// Create a realistic tree
	os.MkdirAll(filepath.Join(src, "sessions"), 0755)
	os.WriteFile(filepath.Join(src, "status.json"), []byte(`{"version":1}`), 0644)
	os.WriteFile(filepath.Join(src, "sessions", "abc.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(src, "codex-original-notify.json"), []byte(`{}`), 0644)

	msg, err := migrateData(src, dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(msg, "migrated") {
		t.Errorf("got %q, want 'migrated'", msg)
	}

	// Source should be gone
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("source should have been removed")
	}
	// Destination files should exist
	if _, err := os.Stat(filepath.Join(dst, "status.json")); err != nil {
		t.Error("status.json missing in destination")
	}
	if _, err := os.Stat(filepath.Join(dst, "sessions", "abc.json")); err != nil {
		t.Error("sessions/abc.json missing in destination")
	}
	if _, err := os.Stat(filepath.Join(dst, "codex-original-notify.json")); err != nil {
		t.Error("codex-original-notify.json missing in destination")
	}
}
