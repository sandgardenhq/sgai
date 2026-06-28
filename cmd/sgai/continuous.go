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

	"github.com/adhocore/gronx"
	"github.com/sandgardenhq/sgai/pkg/state"
)

type triggerKind string

const (
	triggerNone triggerKind = ""
	triggerGoal triggerKind = "goal-changed"
	triggerAuto triggerKind = "auto-timer"
	triggerCron triggerKind = "cron-schedule"
)

const (
	continuousModeMaxRetries   = 3
	continuousModePollInterval = 2 * time.Second
)

func runContinuousWorkflow(ctx context.Context, dir string, continuousPrompt string, mcpURL string, logWriter io.Writer, sessionCoord *state.Coordinator) {
	runner := &workflowRunner{
		dir:       dir,
		mcpURL:    mcpURL,
		logWriter: logWriter,
		coord:     sessionCoord,
	}
	runner.runContinuous(ctx, continuousPrompt)
}

func runContinuousModePrompt(ctx context.Context, dir string, prompt string, mcpURL string, coord *state.Coordinator) {
	updateContinuousModeState(coord, "Running continuous mode prompt...", "continuous-mode", "continuous mode prompt started")

	for attempt := range continuousModeMaxRetries {
		if ctx.Err() != nil {
			return
		}

		cmd := exec.CommandContext(ctx, "opencode", "run", "--title", "continuous-mode-prompt")
		cmd.Dir = dir
		cmd.Env = buildManagedOpenCodeEnv(dir, mcpURL, "continuous-mode", "auto")
		cmd.Stdin = strings.NewReader(prompt)

		if errRun := cmd.Run(); errRun != nil {
			progressMsg := fmt.Sprintf("continuous mode prompt attempt %d/%d failed: %v", attempt+1, continuousModeMaxRetries, errRun)
			updateContinuousModeProgress(coord, progressMsg)
			continue
		}

		updateContinuousModeProgress(coord, "continuous mode prompt completed successfully")
		return
	}

	updateContinuousModeProgress(coord, "continuous mode prompt failed after all retries, proceeding to watch loop")
}

func watchForTrigger(ctx context.Context, dir string, lastChecksum string, autoDuration time.Duration, cronExpr string) triggerKind {
	goalPath := filepath.Join(dir, "GOAL.md")

	var deadline time.Time
	var deadlineTrigger triggerKind

	if cronExpr != "" {
		nextTick, errNext := gronx.NextTick(cronExpr, false)
		if errNext != nil {
			log.Println("failed to parse cron expression:", errNext)
		} else {
			deadline = nextTick
			deadlineTrigger = triggerCron
		}
	}

	now := time.Now()
	if autoDuration > 0 && (deadline.IsZero() || now.Add(autoDuration).Before(deadline)) {
		deadline = now.Add(autoDuration)
		deadlineTrigger = triggerAuto
	}

	for {
		select {
		case <-ctx.Done():
			return triggerNone
		case <-time.After(continuousModePollInterval):
		}

		currentChecksum, errChecksum := computeGoalChecksum(goalPath)
		if errChecksum != nil {
			continue
		}
		if currentChecksum != lastChecksum {
			return triggerGoal
		}

		if !deadline.IsZero() && time.Now().After(deadline) {
			return deadlineTrigger
		}
	}
}

func updateContinuousModeState(coord *state.Coordinator, task, agent, progressMsg string) {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
		wf.Task = task
		wf.Progress = append(wf.Progress, state.ProgressEntry{
			Timestamp:   timestamp,
			Agent:       agent,
			Description: progressMsg,
		})
	}); errUpdate != nil {
		log.Println("failed to update continuous mode state:", errUpdate)
	}
}

func updateContinuousModeProgress(coord *state.Coordinator, progressMsg string) {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
		wf.Progress = append(wf.Progress, state.ProgressEntry{
			Timestamp:   timestamp,
			Agent:       "continuous-mode",
			Description: progressMsg,
		})
	}); errUpdate != nil {
		log.Println("failed to update continuous mode progress:", errUpdate)
	}
}

func resetWorkflowForNextCycle(coord *state.Coordinator) {
	if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
		wf.Status = state.StatusWorking
		wf.InteractionMode = state.ModeContinuous
	}); errUpdate != nil {
		log.Println("failed to reset workflow for next cycle:", errUpdate)
	}
}

func readContinuousModePrompt(workspacePath string) string {
	goalPath := filepath.Join(workspacePath, "GOAL.md")
	data, errRead := os.ReadFile(goalPath)
	if errRead != nil {
		return ""
	}
	metadata, errParse := parseYAMLFrontmatter(data)
	if errParse != nil {
		return ""
	}
	return metadata.ContinuousModePrompt
}

func readContinuousModeAutoCron(workspacePath string) (time.Duration, string) {
	goalPath := filepath.Join(workspacePath, "GOAL.md")
	data, errRead := os.ReadFile(goalPath)
	if errRead != nil {
		return 0, ""
	}
	metadata, errParse := parseYAMLFrontmatter(data)
	if errParse != nil {
		return 0, ""
	}

	var autoDuration time.Duration
	if metadata.ContinuousModeAuto != "" {
		parsed, errParseDuration := time.ParseDuration(metadata.ContinuousModeAuto)
		if errParseDuration != nil {
			log.Println("failed to parse continuousModeAuto duration:", errParseDuration)
		} else {
			autoDuration = parsed
		}
	}

	return autoDuration, metadata.ContinuousModeCron
}
