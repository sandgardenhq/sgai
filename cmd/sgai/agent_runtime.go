package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sandgardenhq/sgai/pkg/state"
)

type agentRunConfig struct {
	dir                   string
	goalPath              string
	agent                 string
	statePath             string
	coord                 *state.Coordinator
	retrospectiveDir      string
	goalAgents            []string
	paddedsgai            string
	mcpURL                string
	logWriter             io.Writer
	stdoutLog             io.Writer
	stderrLog             io.Writer
	activeAgents          *activeAgentTracker
	onActiveAgentsChanged func()
}

func runSingleModelIteration(ctx context.Context, cfg agentRunConfig, wfState state.Workflow, metadata GoalMetadata, iterationCounter *int) state.Workflow {
	var capturedSessionID string
	var consecutiveWorkingIterations int
	outputCapture := newRingWriter()

	for {
		if ctx.Err() != nil {
			fmt.Println("["+cfg.paddedsgai+"]", "interrupted, stopping agent...")
			return wfState
		}

		*iterationCounter++
		prefix := buildAgentPrefix(cfg.dir, cfg.agent, *iterationCounter)

		saveState(cfg.coord, wfState)
		copyProjectManagementToRetrospective(cfg.dir, cfg.retrospectiveDir)

		agentArgs := buildAgentArgs(cfg.agent, metadata.Model, capturedSessionID)
		agentMsg := buildAgentMessage(cfg, wfState, metadata)

		newState, capturedSessionID, errExec := executeAgentProcess(ctx, cfg, agentArgs, agentMsg, prefix, outputCapture, wfState, metadata.Model)
		if errExec != nil {
			return *errExec
		}

		if cfg.retrospectiveDir != "" && capturedSessionID != "" && shouldLogAgent(cfg.dir, cfg.agent) {
			exportAgentSession(cfg, capturedSessionID, *iterationCounter)
		}
		if capturedSessionID != "" {
			if errReconcile := reconcileAgentUsage(cfg.dir, cfg.coord, cfg.agent, capturedSessionID, metadata.Model); errReconcile != nil {
				log.Println("failed to reconcile opencode usage:", errReconcile)
			}
			newState = cfg.coord.State()
		}

		switch newState.Status {
		case state.StatusComplete:
			return handleCompleteStatus(ctx, cfg, newState, wfState, metadata)

		case state.StatusWaitingForHuman:
			wfState = handleWaitingForHumanStatus(cfg, newState)
			continue

		case state.StatusAgentDone:
			saveState(cfg.coord, newState)
			fmt.Println("["+cfg.paddedsgai+"]", "agent", cfg.agent, "done:", newState.Task)
			return newState

		case state.StatusWorking:
			saveState(cfg.coord, newState)
			consecutiveWorkingIterations = handleWorkingLoop(cfg, &capturedSessionID, consecutiveWorkingIterations)
			wfState = newState
			continue

		default:
			log.Fatalln("["+cfg.paddedsgai+"]", "unexpected status:", newState.Status)
		}
	}
}

func buildAgentPrefix(dir, paddedAgentName string, iteration int) string {
	workspaceName := filepath.Base(dir)
	return fmt.Sprintf("[%s][%s:%04d]", workspaceName, paddedAgentName, iteration)
}

func buildAgentMessage(cfg agentRunConfig, wfState state.Workflow, metadata GoalMetadata) string {
	msg := buildCoordinatorDelegationMessage(metadata.Agents, cfg.dir, wfState.InteractionMode)

	pendingTodosCount := countPendingTodos(wfState, cfg.agent)
	if pendingTodosCount > 0 {
		msg += fmt.Sprintf("\nYou have %d pending TODO items. Please complete them before marking agent-done.\n", pendingTodosCount)
	}

	snippets := parseAgentSnippets(cfg.dir, cfg.agent)
	if len(snippets) > 0 {
		snippetsStr := strings.Join(snippets, ", ")
		snippetNudge := fmt.Sprintf("\nIMPORTANT: This agent specializes in %s. YOU MUST call sgai_find_snippets() for these languages BEFORE writing code: %s\n", snippetsStr, snippetsStr)
		msg += snippetNudge
	}

	return msg
}

func handleCompleteStatus(ctx context.Context, cfg agentRunConfig, newState, wfState state.Workflow, metadata GoalMetadata) state.Workflow {
	if cfg.agent != "coordinator" {
		fmt.Println("["+cfg.paddedsgai+"]", "agent", cfg.agent, "set status=complete, only coordinator can complete workflow; treating as agent-done")
		newState.Status = state.StatusAgentDone
		saveState(cfg.coord, newState)
		return newState
	}

	if blocked := blockCompletionOnPendingTodos(cfg, newState, wfState); blocked != nil {
		return *blocked
	}

	if blocked := blockCompletionOnGateScript(ctx, cfg, newState, metadata); blocked != nil {
		return *blocked
	}

	copyCompletionArtifactsToRetrospective(cfg)
	return newState
}

func blockCompletionOnPendingTodos(cfg agentRunConfig, newState, wfState state.Workflow) *state.Workflow {
	count := 0
	for _, todo := range wfState.Todos {
		if todo.Status != "completed" && todo.Status != "cancelled" {
			count++
		}
	}
	if count == 0 {
		return nil
	}
	fmt.Println("["+cfg.paddedsgai+"]", "coordinator cannot complete workflow, there are pending TODO items")
	newState.Status = state.StatusWorking
	if errAppend := appendProjectManagementSection(cfg.dir, "Pending TODO Items", fmt.Sprintf("You have %d pending TODO items. Please complete them before marking workflow complete.", count)); errAppend != nil {
		log.Println("failed to append pending TODO blocker to PROJECT_MANAGEMENT.md:", errAppend)
	}
	saveState(cfg.coord, newState)
	return &newState
}

func blockCompletionOnGateScript(ctx context.Context, cfg agentRunConfig, newState state.Workflow, metadata GoalMetadata) *state.Workflow {
	if metadata.CompletionGateScript == "" {
		return nil
	}
	fmt.Println("["+cfg.paddedsgai+"]", "running completionGateScript:", metadata.CompletionGateScript)
	newState.Task = "running completionGateScript: " + metadata.CompletionGateScript
	saveState(cfg.coord, newState)
	output, errScript := runCompletionGateScript(ctx, cfg.dir, metadata.CompletionGateScript)
	if errScript == nil {
		return nil
	}
	fmt.Println("["+cfg.paddedsgai+"]", "completionGateScript failed, blocking completion")
	newState.Status = state.StatusWorking
	if errAppend := appendProjectManagementSection(cfg.dir, "Completion Gate Failure", formatCompletionGateScriptFailureMessage(metadata.CompletionGateScript, output)); errAppend != nil {
		log.Println("failed to append completion gate failure to PROJECT_MANAGEMENT.md:", errAppend)
	}
	saveState(cfg.coord, newState)
	return &newState
}

func runCompletionGateScript(ctx context.Context, dir, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	cmd.Dir = dir
	cmd.SysProcAttr = commandProcessGroupAttr()

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	errStart := cmd.Start()
	if errStart != nil {
		return "", errStart
	}

	processExited := make(chan struct{})
	go terminateProcessGroupOnCancel(ctx, cmd, processExited)

	errWait := cmd.Wait()
	close(processExited)
	return buf.String(), errWait
}

func formatCompletionGateScriptFailureMessage(script, output string) string {
	return fmt.Sprintf(`From: environment
To: coordinator
Subject: computable definition of success has failed

The script %s has failed with this output:
<pre>
%s
</pre>
`, script, output)
}

func handleWaitingForHumanStatus(cfg agentRunConfig, newState state.Workflow) state.Workflow {
	saveState(cfg.coord, newState)
	if newState.MultiChoiceQuestion != nil || newState.HumanMessage != "" {
		log.Println("agent", cfg.agent, "has pending question after timeout, preserving state for notification")
		newState.Status = state.StatusWorking
		return newState
	}
	fmt.Println("["+cfg.paddedsgai+"]", "waiting-for-human status without pending question; re-running...")
	newState.Status = state.StatusWorking
	return newState
}

func handleWorkingLoop(cfg agentRunConfig, capturedSessionID *string, consecutiveWorkingIterations int) int {
	consecutiveWorkingIterations++
	if consecutiveWorkingIterations >= maxConsecutiveWorkingIterations {
		fmt.Println("["+cfg.paddedsgai+"]", "agent", cfg.agent, "stuck in working loop after", consecutiveWorkingIterations, "iterations; discarding session to recover")
		*capturedSessionID = ""
		consecutiveWorkingIterations = 0
	}
	fmt.Println("["+cfg.paddedsgai+"]", "agent", cfg.agent, "still working, re-running...")
	return consecutiveWorkingIterations
}
