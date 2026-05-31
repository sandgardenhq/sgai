//go:build !linux && !darwin

package main

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

const gracefulShutdownTimeout = 5 * time.Second

func commandProcessGroupAttr() *syscall.SysProcAttr {
	return nil
}

func terminateProcessGroupOnCancel(ctx context.Context, cmd *exec.Cmd, processExited <-chan struct{}) {
	select {
	case <-ctx.Done():
	case <-processExited:
		return
	}
	if cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	select {
	case <-time.After(gracefulShutdownTimeout):
	case <-processExited:
	}
}

func stopProcessGroup(cmd *exec.Cmd, processExited <-chan struct{}) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	select {
	case <-processExited:
	case <-time.After(gracefulShutdownTimeout):
	}
}
