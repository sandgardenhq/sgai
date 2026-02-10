//go:build darwin

package main

import "os/exec"

func openInTerminal(scriptPath string) error {
	return exec.Command("open", "-a", "Terminal", scriptPath).Run()
}
