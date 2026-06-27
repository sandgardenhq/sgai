package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	Running    bool
	Output     string
	Message    string
	BadRequest bool
	Error      error
}

func (s *Server) adhocStartService(workspacePath, prompt, model string) adhocStartResult {
	if strings.TrimSpace(prompt) == "" || strings.TrimSpace(model) == "" {
		return adhocStartResult{BadRequest: true, Error: fmt.Errorf("prompt and model are required")}
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
	cmd.SysProcAttr = commandProcessGroupAttr()
	cmd.Env = buildBaseOpenCodeEnv(workspacePath)
	cmd.Stdin = strings.NewReader(st.promptText)
	writer := &lockedWriter{mu: &st.mu, buf: &st.output}
	prefix := fmt.Sprintf("[%s:%04d]", filepath.Base(workspacePath), 0)
	stdoutPW := &prefixWriter{prefix: prefix + " ", w: os.Stdout}
	stderrPW := &prefixWriter{prefix: prefix + " ", w: os.Stderr}
	stdoutPassthrough := io.MultiWriter(stdoutPW, writer)
	sessionIDCapture := &sessionIDCaptureWriter{detectedWriter: stdoutPassthrough, passthrough: stdoutPassthrough}
	cmd.Stdout = sessionIDCapture
	cmd.Stderr = io.MultiWriter(stderrPW, writer)
	commandLine := "$ opencode " + strings.Join(args, " ")
	promptLine := "prompt: " + st.promptText
	if _, errWriteCommand := fmt.Fprintln(stderrPW, commandLine); errWriteCommand != nil {
		st.running = false
		st.mu.Unlock()
		return adhocStartResult{Error: fmt.Errorf("writing command line: %w", errWriteCommand)}
	}
	if _, errWritePrompt := fmt.Fprintln(stderrPW, promptLine); errWritePrompt != nil {
		st.running = false
		st.mu.Unlock()
		return adhocStartResult{Error: fmt.Errorf("writing prompt line: %w", errWritePrompt)}
	}
	st.output.WriteString(commandLine + "\n")
	st.output.WriteString(promptLine + "\n")

	if errStart := cmd.Start(); errStart != nil {
		st.running = false
		st.mu.Unlock()
		return adhocStartResult{Error: fmt.Errorf("failed to start command: %w", errStart)}
	}

	done := make(chan struct{})
	st.cmd = cmd
	st.done = done
	st.mu.Unlock()

	go func() {
		defer close(done)
		errWait := cmd.Wait()
		sessionIDCapture.Flush()
		st.mu.Lock()
		if errWait != nil {
			st.output.WriteString("\n[command exited with error: " + errWait.Error() + "]\n")
		}
		capturedSessionID := sessionIDCapture.sessionID
		selectedModel := st.selectedModel
		st.running = false
		st.cmd = nil
		st.done = nil
		st.mu.Unlock()
		if errWait == nil {
			s.reconcileAdhocUsage(workspacePath, capturedSessionID, selectedModel)
		}
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
