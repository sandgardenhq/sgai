package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func buildAgentArgs(agent, modelSpec, sessionID string) []string {
	args := []string{"run", "--format=json", "--agent", agent}
	if modelSpec != "" {
		model, variant := parseModelAndVariant(modelSpec)
		args = append(args, "--model", model)
		if variant != "" {
			args = append(args, "--variant", variant)
		}
	}
	if sessionID != "" {
		args = append(args, "--session", sessionID)
	}
	title := agent
	if modelSpec != "" {
		title = agent + " [" + modelSpec + "]"
	}
	args = append(args, "--title", title)
	args = append(args, "--thinking")
	return args
}

func buildAgentEnv(cfg agentRunConfig, wfState state.Workflow, modelSpec string) []string {
	interactiveEnv := "yes"
	if wfState.InteractionMode == state.ModeSelfDrive {
		interactiveEnv = "auto"
	}

	agentIdentity := cfg.agent
	if modelSpec != "" {
		model, variant := parseModelAndVariant(modelSpec)
		agentIdentity = cfg.agent + "|" + model + "|" + variant
	}

	return buildManagedOpenCodeEnv(cfg.dir, cfg.mcpURL, agentIdentity, interactiveEnv)
}

func executeAgentProcess(ctx context.Context, cfg agentRunConfig, agentArgs []string, agentMsg, prefix string, outputCapture *ringWriter, wfState state.Workflow, modelSpec string) (state.Workflow, string, *state.Workflow) {
	stderrOut := buildAgentOutputWriter(os.Stderr, cfg.logWriter, cfg.stderrLog)
	stdoutOut := buildAgentOutputWriter(os.Stdout, cfg.logWriter, cfg.stdoutLog)
	stderrWriter := &prefixWriter{prefix: prefix + " ", w: stderrOut}
	jsonWriter := &jsonPrettyWriter{prefix: prefix + " ", w: stdoutOut, coord: cfg.coord, currentAgent: cfg.agent, activeAgents: cfg.activeAgents, onActiveAgentsChanged: cfg.onActiveAgentsChanged}

	cfg.coord.ResetAgentDoneWatchdog()
	agentCtx, agentCancel := context.WithCancel(ctx)
	cfg.coord.SetAgentCancel(agentCancel)

	cmd := exec.CommandContext(agentCtx, "opencode", agentArgs...)
	cmd.Dir = cfg.dir
	cmd.SysProcAttr = commandProcessGroupAttr()
	cmd.Env = buildAgentEnv(cfg, wfState, modelSpec)
	cmd.Stdin = strings.NewReader(agentMsg)
	cmd.Stderr = io.MultiWriter(stderrWriter, outputCapture)
	cmd.Stdout = io.MultiWriter(jsonWriter, outputCapture)

	if errStart := cmd.Start(); errStart != nil {
		agentCancel()
		clearActiveAgentsForRun(cfg)
		fmt.Fprintln(os.Stderr, "failed to start opencode:", errStart)
		if errUpdate := cfg.coord.UpdateState(func(wf *state.Workflow) {
			wf.Status = state.StatusAgentDone
		}); errUpdate != nil {
			log.Fatalln("failed to save state:", errUpdate)
		}
		fmt.Fprintln(os.Stderr, "agent", cfg.agent, "marked as agent-done due to start failure")
		result := cfg.coord.State()
		return state.Workflow{}, "", &result
	}
	cfg.coord.SetLogFunc(func(message string) {
		if _, errWrite := fmt.Fprintln(stdoutOut, formatLogTimestamp(time.Now())+prefix+"  → "+message); errWrite != nil {
			log.Println("write failed:", errWrite)
		}
	})

	processExited := make(chan struct{})
	go terminateProcessGroupOnCancel(agentCtx, cmd, processExited)

	errWait := cmd.Wait()
	cfg.coord.SetLogFunc(nil)
	close(processExited)
	cfg.coord.Stop()
	agentCancel()

	if errWait != nil {
		clearActiveAgentsForRun(cfg)
		if ctx.Err() != nil {
			fmt.Println("["+cfg.paddedsgai+"]", "interrupted during agent execution")
			return state.Workflow{}, "", &wfState
		}
		fmt.Fprintln(os.Stderr, "\n=== RAW AGENT OUTPUT (last 1000 lines) ===")
		outputCapture.dump(os.Stderr)
		fmt.Fprintln(os.Stderr, "=== END RAW AGENT OUTPUT ===")
		if errUpdate := cfg.coord.UpdateState(func(wf *state.Workflow) {
			wf.Status = state.StatusAgentDone
		}); errUpdate != nil {
			log.Fatalln("failed to save state:", errUpdate)
		}
		fmt.Fprintln(os.Stderr, "agent", cfg.agent, "marked as agent-done due to error:", errWait)
		result := cfg.coord.State()
		return state.Workflow{}, "", &result
	}

	jsonWriter.Flush()
	clearActiveAgentsForRun(cfg)
	return cfg.coord.State(), jsonWriter.sessionID, nil
}

func clearActiveAgentsForRun(cfg agentRunConfig) {
	if cfg.activeAgents != nil && cfg.activeAgents.clear() && cfg.onActiveAgentsChanged != nil {
		cfg.onActiveAgentsChanged()
	}
}

func exportAgentSession(cfg agentRunConfig, sessionID string, iteration int) {
	timestamp := time.Now().Format("20060102150405")
	sessionFile := filepath.Join(cfg.retrospectiveDir, fmt.Sprintf("%04d-%s-%s.json", iteration, cfg.agent, timestamp))
	output, errExport := exportSessionBytes(cfg.dir, sessionID)
	if errExport != nil {
		log.Fatalln("failed to export session:", errExport)
	}
	if errMkdir := os.MkdirAll(filepath.Dir(sessionFile), 0755); errMkdir != nil {
		log.Fatalln("failed to export session:", errMkdir)
	}
	if errWrite := os.WriteFile(sessionFile, output, 0644); errWrite != nil {
		log.Fatalln("failed to export session:", errWrite)
	}
}
