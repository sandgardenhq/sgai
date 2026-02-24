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
	"time"

	"github.com/adhocore/gronx"
	"github.com/sandgardenhq/sgai/pkg/state"
)

type triggerKind string

const (
	triggerNone     triggerKind = ""
	triggerGoal     triggerKind = "goal-changed"
	triggerSteering triggerKind = "steering-message"
	triggerAuto     triggerKind = "auto-timer"
	triggerCron     triggerKind = "cron-schedule"
)

const (
	continuousModeMaxRetries   = 3
	continuousModePollInterval = 2 * time.Second
)

func runContinuousWorkflow(ctx context.Context, args []string, continuousPrompt string, mcpURL string, logWriter io.Writer) {
	if len(args) < 1 {
		log.Fatalln("usage: sgai <target_directory>")
	}

	dir, errAbs := filepath.Abs(args[0])
	if errAbs != nil {
		log.Fatalln(errAbs)
	}

	goalPath := filepath.Join(dir, "GOAL.md")
	stateJSONPath := filepath.Join(dir, ".sgai", "state.json")

	for {
		if ctx.Err() != nil {
			return
		}

		runWorkflow(ctx, args, mcpURL, logWriter)

		if ctx.Err() != nil {
			return
		}

		runContinuousModePrompt(ctx, dir, continuousPrompt, mcpURL)

		if ctx.Err() != nil {
			return
		}

		checksum, errChecksum := computeGoalChecksum(goalPath)
		if errChecksum != nil {
			log.Println("failed to compute GOAL.md checksum:", errChecksum)
			return
		}

		autoDuration, cronExpr := readContinuousModeAutoCron(dir)

		trigger := watchForTrigger(ctx, dir, stateJSONPath, checksum, autoDuration, cronExpr)
		if trigger == triggerNone {
			return
		}

		if trigger == triggerSteering {
			wfState, errLoad := state.Load(stateJSONPath)
			if errLoad != nil {
				log.Println("failed to load state for steering message:", errLoad)
				return
			}
			found, msg := hasHumanPartnerMessage(wfState.Messages)
			if found && msg != nil {
				if errPrepend := prependSteeringMessage(goalPath, msg.Body); errPrepend != nil {
					log.Println("failed to prepend steering message:", errPrepend)
				}
				markMessageAsRead(stateJSONPath, msg.ID)
			}
		}

		resetWorkflowForNextCycle(stateJSONPath)
	}
}

func runContinuousModePrompt(ctx context.Context, dir string, prompt string, mcpURL string) {
	stateJSONPath := filepath.Join(dir, ".sgai", "state.json")

	updateContinuousModeState(stateJSONPath, "Running continuous mode prompt...", "continuous-mode", "continuous mode prompt started")

	for attempt := range continuousModeMaxRetries {
		if ctx.Err() != nil {
			return
		}

		cmd := exec.CommandContext(ctx, "opencode", "run", "--title", "continuous-mode-prompt")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"OPENCODE_CONFIG_DIR="+filepath.Join(dir, ".sgai"),
			"SGAI_MCP_URL="+mcpURL,
			"SGAI_MCP_INTERACTIVE=auto")
		cmd.Stdin = strings.NewReader(prompt)

		if errRun := cmd.Run(); errRun != nil {
			progressMsg := fmt.Sprintf("continuous mode prompt attempt %d/%d failed: %v", attempt+1, continuousModeMaxRetries, errRun)
			updateContinuousModeProgress(stateJSONPath, progressMsg)
			continue
		}

		updateContinuousModeProgress(stateJSONPath, "continuous mode prompt completed successfully")
		return
	}

	updateContinuousModeProgress(stateJSONPath, "continuous mode prompt failed after all retries, proceeding to watch loop")
}

func watchForTrigger(ctx context.Context, dir string, stateJSONPath string, lastChecksum string, autoDuration time.Duration, cronExpr string) triggerKind {
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

		wfState, errLoad := state.Load(stateJSONPath)
		if errLoad != nil {
			continue
		}
		found, _ := hasHumanPartnerMessage(wfState.Messages)
		if found {
			return triggerSteering
		}

		if !deadline.IsZero() && time.Now().After(deadline) {
			return deadlineTrigger
		}
	}
}

func prependSteeringMessage(goalPath string, message string) error {
	content, errRead := os.ReadFile(goalPath)
	if errRead != nil {
		return fmt.Errorf("reading GOAL.md: %w", errRead)
	}

	delimiter := []byte("---")

	if !bytes.HasPrefix(content, delimiter) {
		newContent := message + "\n\n" + string(content)
		return os.WriteFile(goalPath, []byte(newContent), 0644)
	}

	rest := content[len(delimiter):]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	closingIdx := bytes.Index(rest, delimiter)
	if closingIdx == -1 {
		newContent := message + "\n\n" + string(content)
		return os.WriteFile(goalPath, []byte(newContent), 0644)
	}

	frontmatterEnd := len(delimiter) + 1 + closingIdx + len(delimiter)
	if frontmatterEnd < len(content) && content[frontmatterEnd] == '\n' {
		frontmatterEnd++
	}

	var buf bytes.Buffer
	buf.Write(content[:frontmatterEnd])
	buf.WriteString("\n")
	buf.WriteString(message)
	buf.WriteString("\n\n")
	if frontmatterEnd < len(content) {
		buf.Write(content[frontmatterEnd:])
	}

	return os.WriteFile(goalPath, buf.Bytes(), 0644)
}

func hasHumanPartnerMessage(messages []state.Message) (bool, *state.Message) {
	for i := range messages {
		if messages[i].Read {
			continue
		}
		if messages[i].FromAgent == "Human Partner" {
			return true, &messages[i]
		}
	}
	return false, nil
}

func updateContinuousModeState(stateJSONPath, task, agent, progressMsg string) {
	wfState, errLoad := state.Load(stateJSONPath)
	if errLoad != nil {
		return
	}
	wfState.Task = task
	wfState.CurrentAgent = agent
	wfState.Progress = append(wfState.Progress, state.ProgressEntry{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Agent:       agent,
		Description: progressMsg,
	})
	if errSave := state.Save(stateJSONPath, wfState); errSave != nil {
		log.Println("failed to update continuous mode state:", errSave)
	}
}

func updateContinuousModeProgress(stateJSONPath, progressMsg string) {
	wfState, errLoad := state.Load(stateJSONPath)
	if errLoad != nil {
		return
	}
	wfState.Progress = append(wfState.Progress, state.ProgressEntry{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Agent:       "continuous-mode",
		Description: progressMsg,
	})
	if errSave := state.Save(stateJSONPath, wfState); errSave != nil {
		log.Println("failed to update continuous mode progress:", errSave)
	}
}

func markMessageAsRead(stateJSONPath string, messageID int) {
	wfState, errLoad := state.Load(stateJSONPath)
	if errLoad != nil {
		return
	}
	for i := range wfState.Messages {
		if wfState.Messages[i].ID == messageID {
			wfState.Messages[i].Read = true
			wfState.Messages[i].ReadAt = time.Now().UTC().Format(time.RFC3339)
			wfState.Messages[i].ReadBy = "continuous-mode"
			break
		}
	}
	if errSave := state.Save(stateJSONPath, wfState); errSave != nil {
		log.Println("failed to mark message as read:", errSave)
	}
}

func resetWorkflowForNextCycle(stateJSONPath string) {
	wfState, errLoad := state.Load(stateJSONPath)
	if errLoad != nil {
		return
	}
	wfState.Status = state.StatusWorking
	wfState.InteractionMode = state.ModeContinuous
	wfState.CurrentAgent = "coordinator"
	if errSave := state.Save(stateJSONPath, wfState); errSave != nil {
		log.Println("failed to reset workflow for next cycle:", errSave)
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
