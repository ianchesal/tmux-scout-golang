package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	statusBarBase = "#[fg=#e06c75]{W}#[default]|#[fg=#e5c07b]{B}#[default]|#[fg=#98c379]{D}#[default]"
	statusBarIdle = "|#[fg=#61afef]{I}#[default]"
)

func runStatusBar() {
	scoutDir := defaultScoutDir()
	statusFile := scoutDir + "/status.json"

	result := Sync(statusFile, scoutDir)
	active := getActiveSessions(result.Status, result.Panes)

	var wait, busy, done, idle int
	for _, s := range active {
		if isNeedsAttention(s) {
			wait++
		} else if s.Status == "working" {
			busy++
		} else if s.Status == "completed" {
			done++
		} else if s.Status == "idle" {
			idle++
		}
	}

	if wait+busy+done+idle == 0 {
		return
	}

	var fmtStr string
	if out, err := exec.Command("tmux", "show-option", "-gqv", "@scout-status-format").Output(); err == nil {
		fmtStr = strings.TrimSpace(string(out))
	}

	replace := func(s string) string {
		s = strings.ReplaceAll(s, "{W}", fmt.Sprintf("%d", wait))
		s = strings.ReplaceAll(s, "{B}", fmt.Sprintf("%d", busy))
		s = strings.ReplaceAll(s, "{D}", fmt.Sprintf("%d", done))
		s = strings.ReplaceAll(s, "{I}", fmt.Sprintf("%d", idle))
		return s
	}

	if fmtStr != "" {
		fmt.Fprint(os.Stdout, replace(fmtStr))
		return
	}

	output := replace(statusBarBase)
	if idle > 0 {
		output += replace(statusBarIdle)
	}
	fmt.Fprint(os.Stdout, output+" ")
}
