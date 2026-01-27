//go:build darwin

// Package notify provides desktop notification support.
// On macOS, notifications are displayed using osascript.
// On other platforms, Send is a no-op.
package notify

import (
	"os/exec"
	"strings"
)

func sendLocal(title, message string) error {
	title = escapeAppleScript(title)
	message = escapeAppleScript(message)

	script := `display notification "` + message + `" with title "` + title + `"`
	return exec.Command("osascript", "-e", script).Run()
}

func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
