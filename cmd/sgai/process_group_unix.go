//go:build linux || darwin

package main

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

const gracefulShutdownTimeout = 5 * time.Second

func commandProcessGroupAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

func terminateProcessGroupOnCancel(ctx context.Context, cmd *exec.Cmd, processExited <-chan struct{}) {
	select {
	case <-ctx.Done():
	case <-processExited:
		return
	}
	pgid := -cmd.Process.Pid
	_ = syscall.Kill(pgid, syscall.SIGTERM)
	select {
	case <-time.After(gracefulShutdownTimeout):
		_ = syscall.Kill(pgid, syscall.SIGKILL)
	case <-processExited:
	}
}

func stopProcessGroup(cmd *exec.Cmd, processExited <-chan struct{}) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pgid := -cmd.Process.Pid
	_ = syscall.Kill(pgid, syscall.SIGTERM)

	select {
	case <-processExited:
	case <-time.After(gracefulShutdownTimeout):
		_ = syscall.Kill(pgid, syscall.SIGKILL)
	}
}
