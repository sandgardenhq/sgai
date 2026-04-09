package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type adhocStatusResult struct {
	Running bool
	Output  string
	Message string
}

func (s *Server) adhocStatusService(workspacePath string) adhocStatusResult {
	st := s.getAdhocState(workspacePath)
	st.mu.Lock()
	running := st.running
	output := st.output.String()
	st.mu.Unlock()

	return adhocStatusResult{Running: running, Output: output, Message: "adhoc status"}
}

type adhocStartResult struct {
	Running bool
	Output  string
	Message string
	Error   error
}

func (s *Server) adhocStartService(workspacePath, prompt, model string) adhocStartResult {
	if strings.TrimSpace(prompt) == "" || strings.TrimSpace(model) == "" {
		return adhocStartResult{Error: fmt.Errorf("prompt and model are required")}
	}

	st := s.getAdhocState(workspacePath)
	st.mu.Lock()
	if st.running {
		output := st.output.String()
		st.mu.Unlock()
		return adhocStartResult{Running: true, Output: output, Message: "ad-hoc prompt already running"}
	}

	st.running = true
	st.output.Reset()
	st.selectedModel = strings.TrimSpace(model)
	st.promptText = strings.TrimSpace(prompt)

	args := buildAdhocArgs(st.selectedModel)
	cmd := exec.Command("opencode", args...)
	cmd.Dir = workspacePath
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = append(os.Environ(), "OPENCODE_CONFIG_DIR="+filepath.Join(workspacePath, ".sgai"))
	cmd.Stdin = strings.NewReader(st.promptText)
	writer := &lockedWriter{mu: &st.mu, buf: &st.output}
	prefix := fmt.Sprintf("[%s][adhoc:0000]", filepath.Base(workspacePath))
	stdoutPW := &prefixWriter{prefix: prefix + " ", w: os.Stdout, startTime: time.Now()}
	stderrPW := &prefixWriter{prefix: prefix + " ", w: os.Stderr, startTime: time.Now()}
	cmd.Stdout = io.MultiWriter(stdoutPW, writer)
	cmd.Stderr = io.MultiWriter(stderrPW, writer)
	commandLine := "$ opencode " + strings.Join(args, " ")
	promptLine := "prompt: " + st.promptText
	_, _ = fmt.Fprintln(stderrPW, commandLine)
	_, _ = fmt.Fprintln(stderrPW, promptLine)
	st.output.WriteString(commandLine + "\n")
	st.output.WriteString(promptLine + "\n")

	if errStart := cmd.Start(); errStart != nil {
		st.running = false
		st.mu.Unlock()
		return adhocStartResult{Error: fmt.Errorf("failed to start command")}
	}

	st.cmd = cmd
	st.mu.Unlock()

	go func() {
		errWait := cmd.Wait()
		st.mu.Lock()
		if errWait != nil {
			st.output.WriteString("\n[command exited with error: " + errWait.Error() + "]\n")
		}
		st.running = false
		st.cmd = nil
		st.mu.Unlock()
	}()

	s.notifyStateChange()

	return adhocStartResult{Running: true, Message: "ad-hoc prompt started"}
}

type adhocStopResult struct {
	Running bool
	Output  string
	Message string
}

func (s *Server) adhocStopService(workspacePath string) adhocStopResult {
	st := s.getAdhocState(workspacePath)
	st.stop()
	s.notifyStateChange()

	st.mu.Lock()
	output := st.output.String()
	st.mu.Unlock()

	return adhocStopResult{Running: false, Output: output, Message: "ad-hoc stopped"}
}
