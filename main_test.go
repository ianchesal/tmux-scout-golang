package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

const testBinary = "/tmp/tmux-scout-test"

// TestMain builds the versioned test binary once for all version tests.
func TestMain(m *testing.M) {
	build := exec.Command("go", "build", "-ldflags", "-X main.version=v0.2.0", "-o", testBinary, ".")
	if out, err := build.CombinedOutput(); err != nil {
		panic("build failed: " + string(out))
	}
	code := m.Run()
	os.Remove(testBinary)
	os.Exit(code)
}

func TestVersionFlags(t *testing.T) {
	for _, flag := range []string{"--version", "-version"} {
		t.Run(flag, func(t *testing.T) {
			out, err := exec.Command(testBinary, flag).Output()
			if err != nil {
				t.Fatalf("%s exited non-zero: %v", flag, err)
			}
			got := strings.TrimSpace(string(out))
			if got != "v0.2.0" {
				t.Errorf("%s = %q, want %q", flag, got, "v0.2.0")
			}
		})
	}
}
