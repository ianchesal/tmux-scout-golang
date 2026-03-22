package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

func runMigrate() {
	home, _ := os.UserHomeDir()
	oldDir := filepath.Join(home, ".tmux-scout")
	newDir := defaultScoutDir()

	// Step 1: data directory
	msg, err := migrateData(oldDir, newDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "data: error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("data:", msg)

	// Step 2: binary
	localBin := filepath.Join(home, ".local", "bin")
	binaryPlaced, msg2, err := migrateBinary(localBin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "binary: error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("binary:", msg2)

	// Step 3: hook reinstall
	if !binaryPlaced {
		return
	}
	newBin := filepath.Join(localBin, "tmux-scout")
	claudeSettings := filepath.Join(home, ".claude", "settings.json")
	codexConfig := filepath.Join(home, ".codex", "config.toml")
	geminiSettings := filepath.Join(home, ".gemini", "settings.json")

	_, _ = claudeInstall(claudeSettings, newBin)
	_, _ = geminiInstall(geminiSettings, newBin)
	_, _ = codexUninstall(codexConfig, newDir)
	_, _ = codexInstall(codexConfig, newDir, newBin)
	fmt.Println("hooks: reinstalled with new binary path")
}

// migrateData moves src to dst, with cross-device fallback.
// Returns a human-readable message and an error if the move was attempted and failed.
func migrateData(src, dst string) (string, error) {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return "nothing to migrate", nil
	}
	if _, err := os.Stat(dst); err == nil {
		return "destination already exists, skipping", nil
	}

	if err := os.Rename(src, dst); err != nil {
		// Cross-device: fall back to recursive copy then delete
		var linkErr *os.LinkError
		if errors.As(err, &linkErr) && errors.Is(linkErr.Err, syscall.EXDEV) {
			if err2 := copyDir(src, dst); err2 != nil {
				return "", fmt.Errorf("copy fallback failed: %w", err2)
			}
			if err2 := os.RemoveAll(src); err2 != nil {
				return "", fmt.Errorf("remove source after copy failed: %w", err2)
			}
		} else {
			return "", err
		}
	}
	return fmt.Sprintf("migrated %s → %s", src, dst), nil
}

// migrateBinary creates a symlink at localBin/tmux-scout → current executable.
// Returns (placed bool, message string, error).
func migrateBinary(localBin string) (bool, string, error) {
	if _, err := os.Stat(localBin); os.IsNotExist(err) {
		return false, "~/.local/bin not found, skipping", nil
	}

	dst := filepath.Join(localBin, "tmux-scout")
	src := binaryPath()

	// Remove any existing file/symlink at dst before creating the new symlink
	_ = os.Remove(dst)

	if err := os.Symlink(src, dst); err != nil {
		return false, "", fmt.Errorf("symlink %s: %w", dst, err)
	}
	return true, fmt.Sprintf("symlinked %s → %s", dst, src), nil
}

// copyDir recursively copies src directory tree to dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target)
	})
}

// copyFile copies a single file from src to dst, creating parent dirs as needed.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
