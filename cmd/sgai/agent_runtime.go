package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sandgardenhq/sgai/pkg/state"
)

type agentRunConfig struct {
	dir              string
	goalPath         string
	agent            string
	statePath        string
	coord            *state.Coordinator
	retrospectiveDir string
	goalAgents       []string
	paddedsgai       string
	mcpURL           string
	logWriter        io.Writer
	stdoutLog        io.Writer
	stderrLog        io.Writer
}

func buildIterationPrefix(dir string, iteration int) string {
	workspaceName := filepath.Base(dir)
	return fmt.Sprintf("[%s:%04d]", workspaceName, iteration)
}

func buildAgentMessage(cfg agentRunConfig, wfState state.Workflow, metadata GoalMetadata) string {
	var agentLines []string
	for _, agent := range delegatableAgents(metadata.Agents) {
		agentPath := cfg.dir + "/.sgai/agent/" + agent + ".md"
		content, errRead := os.ReadFile(agentPath)
		var line string
		if errRead != nil {
			line = agent
		} else if desc := extractFrontmatterDescription(string(content)); desc != "" {
			line = agent + ": " + desc
		} else {
			line = agent
		}
		agentLines = append(agentLines, line)
	}
	if len(agentLines) == 0 {
		agentLines = append(agentLines, "(none configured)")
	}
	modeSection, coordPlan := modeSectionForMode(wfState.InteractionMode)

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(promptSectionPreamble)
	sb.WriteString("\n\n")

	sb.WriteString(promptSectionHumanCommDirect)
	sb.WriteString("\n\n")

	sb.WriteString(promptSectionMessaging)
	sb.WriteString("\n\n")

	sb.WriteString(promptSectionProjectManagementMonitor)
	sb.WriteString("\n\n")

	sb.WriteString(promptSectionWorkFocus)
	sb.WriteString("\n\n")
	sb.WriteString(promptSectionDelegation)
	sb.WriteString("\n")
	sb.WriteString(`## Parallel Task Subagent Usage
CRITICALLY IMPORTANT: use as many Task subagents as possible at all times. Your default action is to split work into independent units and launch every safe Task subagent concurrently.
- Before doing coordinator-direct work, identify which parts can be delegated to Task subagents.
- If two or more Task subagents can run without editing the same files or depending on each other's output, launch them in the same parallel batch.
- Prefer multiple focused Task subagents over one broad Task subagent.
- Only serialize Task subagents when there is a concrete dependency, shared edit target, or required ordering.
- If you do not delegate or you serialize work, record the specific reason in .sgai/PROJECT_MANAGEMENT.md.
`)
	sb.WriteString("\n")

	sb.WriteString(promptSectionPostSkillsCoordinator)
	sb.WriteString("\n\n")

	sb.WriteString(promptSectionGuidelines)
	sb.WriteString("\n\n")

	sb.WriteString(promptSectionTailCoordinator)
	sb.WriteString("\n")

	sb.WriteString(promptSectionCommonTail)
	sb.WriteString("\n")

	sb.WriteString(promptSectionCoordinatorMessagingTail)
	sb.WriteString("\n")
	sb.WriteString("\n")
	sb.WriteString(modeSection)
	sb.WriteString(coordPlan)

	msg := strings.ReplaceAll(sb.String(), "%AGENTS_LIST%", strings.Join(agentLines, "\n"))

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
