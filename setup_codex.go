package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// codexHookIdentifierNew is a comment marker written into the TOML block.
	// Using a comment ensures reliable detection regardless of binary path.
	codexHookIdentifierNew = "# managed-by: tmux-scout"
	codexHookIdentifierOld = "tmux-scout/scripts/hooks/codex.js"
)

var notifyRegex = regexp.MustCompile(`(?m)^notify\s*=\s*\[([^\]]*)\]`)

func codexBuildNotifyBlock(binPath string) string {
	return fmt.Sprintf("notify = [\n  %s\n  %q,\n  \"hook\",\n  \"codex\"\n]", codexHookIdentifierNew, binPath)
}

func codexParseNotifyArray(content string) []string {
	m := notifyRegex.FindStringSubmatch(content)
	if m == nil {
		return nil
	}
	strRegex := regexp.MustCompile(`"([^"\\]*(?:\\.[^"\\]*)*)"`)
	matches := strRegex.FindAllStringSubmatch(m[1], -1)
	var result []string
	for _, sm := range matches {
		result = append(result, sm[1])
	}
	return result
}

func codexInstall(configPath, scoutDir, binPath string) (string, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "skipped", nil
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}
	s := string(content)

	if strings.Contains(s, codexHookIdentifierNew) {
		return "ok", nil
	}
	if strings.Contains(s, codexHookIdentifierOld) {
		// Update path in place
		newContent := notifyRegex.ReplaceAllString(s, codexBuildNotifyBlock(binPath))
		return "updated", writeAtomic(configPath, []byte(newContent))
	}

	// Fresh install — backup original notify
	backupPath := filepath.Join(scoutDir, "codex-original-notify.json")
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		existing := codexParseNotifyArray(s)
		backupData, _ := json.MarshalIndent(map[string]interface{}{"notify": existing}, "", "  ")
		_ = os.MkdirAll(scoutDir, 0755)
		_ = writeAtomic(backupPath, backupData)
	}

	newBlock := codexBuildNotifyBlock(binPath)
	var newContent string
	if notifyRegex.MatchString(s) {
		newContent = notifyRegex.ReplaceAllString(s, newBlock)
	} else {
		// Insert after leading comment lines
		lines := strings.Split(s, "\n")
		insertIdx := 0
		for insertIdx < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[insertIdx]), "#") {
			insertIdx++
		}
		lines = append(lines[:insertIdx], append([]string{newBlock}, lines[insertIdx:]...)...)
		newContent = strings.Join(lines, "\n")
	}
	return "installed", writeAtomic(configPath, []byte(newContent))
}

func codexUninstall(configPath, scoutDir string) (string, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "skipped", nil
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}
	s := string(content)
	if !strings.Contains(s, codexHookIdentifierNew) && !strings.Contains(s, codexHookIdentifierOld) {
		return "not_found", nil
	}

	backupPath := filepath.Join(scoutDir, "codex-original-notify.json")
	var newContent string

	if data, err := os.ReadFile(backupPath); err == nil {
		var backup struct {
			Notify []string `json:"notify"`
		}
		if err := json.Unmarshal(data, &backup); err == nil && len(backup.Notify) > 0 {
			// Restore original array
			parts := make([]string, len(backup.Notify))
			for i, v := range backup.Notify {
				parts[i] = fmt.Sprintf("  %q", v)
			}
			restored := "notify = [\n" + strings.Join(parts, ",\n") + "\n]"
			newContent = notifyRegex.ReplaceAllString(s, restored)
		} else {
			// Original had no notify — remove block
			newContent = notifyRegex.ReplaceAllString(s, "")
			newContent = regexp.MustCompile(`\n{3,}`).ReplaceAllString(newContent, "\n\n")
		}
		_ = os.Remove(backupPath)
	} else {
		newContent = notifyRegex.ReplaceAllString(s, "")
		newContent = regexp.MustCompile(`\n{3,}`).ReplaceAllString(newContent, "\n\n")
	}

	return "removed", writeAtomic(configPath, []byte(newContent))
}

func codexStatus(configPath string) (bool, bool) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false, false // not installed, not available
	}
	content, _ := os.ReadFile(configPath)
	s := string(content)
	return strings.Contains(s, codexHookIdentifierNew) || strings.Contains(s, codexHookIdentifierOld), true
}
