package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	claudeHookIdentifierNew = "tmux-scout hook claude"
	claudeHookIdentifierOld = "tmux-scout/scripts/hooks/claude.js"
)

var claudeHookEvents = []string{
	"SessionStart", "UserPromptSubmit", "PreToolUse",
	"PostToolUse", "Stop", "SessionEnd",
}

type claudeHookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

type claudeMatcherGroup struct {
	Matcher string            `json:"matcher"`
	Hooks   []claudeHookEntry `json:"hooks"`
}

func claudeReadSettings(path string) (map[string]interface{}, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("settings.json not found")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s map[string]interface{}
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return s, nil
}

func claudeWriteSettings(path string, s map[string]interface{}) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeAtomic(path, data)
}

func isScoutHook(cmd string) bool {
	return strings.Contains(cmd, claudeHookIdentifierNew) || strings.Contains(cmd, claudeHookIdentifierOld)
}

func claudeInstall(settingsPath, binPath string) (string, error) {
	s, err := claudeReadSettings(settingsPath)
	if err != nil {
		return "skipped", nil
	}

	if s["hooks"] == nil {
		s["hooks"] = map[string]interface{}{}
	}
	hooksMap, ok := s["hooks"].(map[string]interface{})
	if !ok {
		return "skipped", nil
	}

	newCmd := fmt.Sprintf("%s hook claude", binPath)
	changed := false

	for _, event := range claudeHookEvents {
		var groups []claudeMatcherGroup
		if raw, ok := hooksMap[event]; ok {
			b, _ := json.Marshal(raw)
			_ = json.Unmarshal(b, &groups)
		}

		// Find or create catch-all group; always ensure it is at index 0 (prepended)
		catchAllIdx := -1
		for i, g := range groups {
			if g.Matcher == "" {
				catchAllIdx = i
				break
			}
		}
		if catchAllIdx < 0 {
			groups = append([]claudeMatcherGroup{{Matcher: "", Hooks: []claudeHookEntry{}}}, groups...)
			catchAllIdx = 0
			changed = true
		} else if catchAllIdx > 0 {
			// Move existing catch-all to front
			catchAll := groups[catchAllIdx]
			groups = append([]claudeMatcherGroup{catchAll}, append(groups[:catchAllIdx], groups[catchAllIdx+1:]...)...)
			catchAllIdx = 0
			changed = true
		}

		// Find existing scout hook in catch-all
		existingIdx := -1
		for i, h := range groups[catchAllIdx].Hooks {
			if isScoutHook(h.Command) {
				existingIdx = i
				break
			}
		}

		entry := claudeHookEntry{Type: "command", Command: newCmd, Timeout: 5}
		if existingIdx >= 0 {
			if groups[catchAllIdx].Hooks[existingIdx].Command != newCmd {
				groups[catchAllIdx].Hooks[existingIdx] = entry
				changed = true
			}
		} else {
			groups[catchAllIdx].Hooks = append(groups[catchAllIdx].Hooks, entry)
			changed = true
		}

		hooksMap[event] = groups
	}

	if changed {
		s["hooks"] = hooksMap
		if err := claudeWriteSettings(settingsPath, s); err != nil {
			return "", err
		}
		return "installed", nil
	}
	return "ok", nil
}

func claudeUninstall(settingsPath string) (string, error) {
	s, err := claudeReadSettings(settingsPath)
	if err != nil {
		return "skipped", nil
	}
	hooksMap, ok := s["hooks"].(map[string]interface{})
	if !ok {
		return "not_found", nil
	}

	changed := false
	for _, event := range claudeHookEvents {
		raw, ok := hooksMap[event]
		if !ok {
			continue
		}
		var groups []claudeMatcherGroup
		b, _ := json.Marshal(raw)
		_ = json.Unmarshal(b, &groups)

		for gi := len(groups) - 1; gi >= 0; gi-- {
			before := len(groups[gi].Hooks)
			filtered := groups[gi].Hooks[:0]
			for _, h := range groups[gi].Hooks {
				if !isScoutHook(h.Command) {
					filtered = append(filtered, h)
				}
			}
			groups[gi].Hooks = filtered
			if len(groups[gi].Hooks) < before {
				changed = true
			}
			if len(groups[gi].Hooks) == 0 {
				groups = append(groups[:gi], groups[gi+1:]...)
			}
		}

		if len(groups) == 0 {
			delete(hooksMap, event)
		} else {
			hooksMap[event] = groups
		}
	}

	if len(hooksMap) == 0 {
		delete(s, "hooks")
	}

	if changed {
		return "removed", claudeWriteSettings(settingsPath, s)
	}
	return "not_found", nil
}

func claudeStatus(settingsPath string) (int, error) {
	s, err := claudeReadSettings(settingsPath)
	if err != nil {
		return 0, nil
	}
	hooksMap, ok := s["hooks"].(map[string]interface{})
	if !ok {
		return 0, nil
	}
	count := 0
	for _, event := range claudeHookEvents {
		raw, ok := hooksMap[event]
		if !ok {
			continue
		}
		var groups []claudeMatcherGroup
		b, _ := json.Marshal(raw)
		_ = json.Unmarshal(b, &groups)
		for _, g := range groups {
			for _, h := range g.Hooks {
				if isScoutHook(h.Command) {
					count++
				}
			}
		}
	}
	return count, nil
}
