package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const geminiHookIdentifier = "tmux-scout hook gemini"

var geminiHookEvents = []string{
	"SessionStart", "SessionEnd",
	"BeforeAgent", "AfterAgent",
	"BeforeTool", "AfterTool",
	"Notification",
}

type geminiHookEntry struct {
	Name    string `json:"name"`    // required by Gemini CLI hook schema; not present in Claude hook entries
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"` // milliseconds (Gemini CLI), unlike Claude which uses seconds
}

type geminiMatcherGroup struct {
	Matcher string            `json:"matcher"`
	Hooks   []geminiHookEntry `json:"hooks"`
}

func geminiReadSettings(path string) (map[string]interface{}, error) {
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

func geminiWriteSettings(path string, s map[string]interface{}) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeAtomic(path, data)
}

func isGeminiScoutHook(cmd string) bool {
	return strings.Contains(cmd, geminiHookIdentifier)
}

func geminiInstall(settingsPath, binPath string) (string, error) {
	s, err := geminiReadSettings(settingsPath)
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

	newCmd := fmt.Sprintf("%s hook gemini", binPath)
	changed := false

	for _, event := range geminiHookEvents {
		var groups []geminiMatcherGroup
		if raw, ok := hooksMap[event]; ok {
			b, _ := json.Marshal(raw)
			_ = json.Unmarshal(b, &groups)
		}

		// Find or create "*" matcher group; always ensure it is at index 0.
		catchAllIdx := -1
		for i, g := range groups {
			if g.Matcher == "*" {
				catchAllIdx = i
				break
			}
		}
		if catchAllIdx < 0 {
			groups = append([]geminiMatcherGroup{{Matcher: "*", Hooks: []geminiHookEntry{}}}, groups...)
			catchAllIdx = 0
			changed = true
		} else if catchAllIdx > 0 {
			catchAll := groups[catchAllIdx]
			groups = append([]geminiMatcherGroup{catchAll}, append(groups[:catchAllIdx], groups[catchAllIdx+1:]...)...)
			catchAllIdx = 0
			changed = true
		}

		// Find existing scout hook in the "*" group (identity: command contains geminiHookIdentifier).
		existingIdx := -1
		for i, h := range groups[catchAllIdx].Hooks {
			if isGeminiScoutHook(h.Command) {
				existingIdx = i
				break
			}
		}

		entry := geminiHookEntry{Name: "tmux-scout", Type: "command", Command: newCmd, Timeout: 5000}
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
		if err := geminiWriteSettings(settingsPath, s); err != nil {
			return "", err
		}
		return "installed", nil
	}
	return "ok", nil
}

func geminiUninstall(settingsPath string) (string, error) {
	s, err := geminiReadSettings(settingsPath)
	if err != nil {
		return "skipped", nil
	}
	hooksMap, ok := s["hooks"].(map[string]interface{})
	if !ok {
		return "not_found", nil
	}

	changed := false
	for _, event := range geminiHookEvents {
		raw, ok := hooksMap[event]
		if !ok {
			continue
		}
		var groups []geminiMatcherGroup
		b, _ := json.Marshal(raw)
		_ = json.Unmarshal(b, &groups)

		for gi := len(groups) - 1; gi >= 0; gi-- {
			before := len(groups[gi].Hooks)
			filtered := groups[gi].Hooks[:0]
			for _, h := range groups[gi].Hooks {
				if !isGeminiScoutHook(h.Command) {
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
		return "removed", geminiWriteSettings(settingsPath, s)
	}
	return "not_found", nil
}

func geminiStatus(settingsPath string) (int, error) {
	s, err := geminiReadSettings(settingsPath)
	if err != nil {
		return 0, nil
	}
	hooksMap, ok := s["hooks"].(map[string]interface{})
	if !ok {
		return 0, nil
	}
	count := 0
	for _, event := range geminiHookEvents {
		raw, ok := hooksMap[event]
		if !ok {
			continue
		}
		var groups []geminiMatcherGroup
		b, _ := json.Marshal(raw)
		_ = json.Unmarshal(b, &groups)
		for _, g := range groups {
			for _, h := range g.Hooks {
				if isGeminiScoutHook(h.Command) {
					count++
				}
			}
		}
	}
	return count, nil
}
