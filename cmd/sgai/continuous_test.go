package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func TestPrependSteeringMessage(t *testing.T) {
	t.Run("withFrontmatter", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		original := "---\nflow: |\n  a -> b\n---\n\nOriginal content here.\n"
		if err := os.WriteFile(goalPath, []byte(original), 0644); err != nil {
			t.Fatal(err)
		}

		if err := prependSteeringMessage(goalPath, "STEERING: do this next"); err != nil {
			t.Fatal(err)
		}

		result, err := os.ReadFile(goalPath)
		if err != nil {
			t.Fatal(err)
		}

		content := string(result)
		if !strings.HasPrefix(content, "---\nflow: |\n  a -> b\n---\n") {
			t.Error("frontmatter was corrupted")
		}
		if !strings.Contains(content, "STEERING: do this next") {
			t.Error("steering message not found")
		}
		if !strings.Contains(content, "Original content here.") {
			t.Error("original content was lost")
		}

		steeringIdx := strings.Index(content, "STEERING: do this next")
		originalIdx := strings.Index(content, "Original content here.")
		if steeringIdx >= originalIdx {
			t.Error("steering message should appear before original content")
		}
	})

	t.Run("withoutFrontmatter", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		original := "Just some content without frontmatter.\n"
		if err := os.WriteFile(goalPath, []byte(original), 0644); err != nil {
			t.Fatal(err)
		}

		if err := prependSteeringMessage(goalPath, "STEERING: new instruction"); err != nil {
			t.Fatal(err)
		}

		result, err := os.ReadFile(goalPath)
		if err != nil {
			t.Fatal(err)
		}

		content := string(result)
		if !strings.HasPrefix(content, "STEERING: new instruction") {
			t.Error("steering message should be at the beginning")
		}
		if !strings.Contains(content, "Just some content without frontmatter.") {
			t.Error("original content was lost")
		}
	})

	t.Run("multipleCalls", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		original := "---\nflow: |\n  x -> y\n---\n\nBase content.\n"
		if err := os.WriteFile(goalPath, []byte(original), 0644); err != nil {
			t.Fatal(err)
		}

		if err := prependSteeringMessage(goalPath, "First steering"); err != nil {
			t.Fatal(err)
		}
		if err := prependSteeringMessage(goalPath, "Second steering"); err != nil {
			t.Fatal(err)
		}

		result, err := os.ReadFile(goalPath)
		if err != nil {
			t.Fatal(err)
		}

		content := string(result)
		if !strings.Contains(content, "First steering") {
			t.Error("first steering message was lost")
		}
		if !strings.Contains(content, "Second steering") {
			t.Error("second steering message was lost")
		}
		if !strings.Contains(content, "Base content.") {
			t.Error("original content was lost")
		}

		metadata, errParse := parseYAMLFrontmatter(result)
		if errParse != nil {
			t.Fatal("frontmatter corrupted after multiple prepends:", errParse)
		}
		if metadata.Flow == "" {
			t.Error("flow metadata was lost after multiple prepends")
		}
	})

	t.Run("unclosedFrontmatter", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		original := "---\nflow: |\n  a -> b\nUnclosed frontmatter content.\n"
		if err := os.WriteFile(goalPath, []byte(original), 0644); err != nil {
			t.Fatal(err)
		}

		if err := prependSteeringMessage(goalPath, "STEERING: test"); err != nil {
			t.Fatal(err)
		}

		result, err := os.ReadFile(goalPath)
		if err != nil {
			t.Fatal(err)
		}

		content := string(result)
		if !strings.HasPrefix(content, "STEERING: test") {
			t.Error("steering message should be at the beginning for unclosed frontmatter")
		}
	})
}

func TestHasHumanPartnerMessage(t *testing.T) {
	t.Run("findsHumanPartnerMessage", func(t *testing.T) {
		messages := []state.Message{
			{ID: 1, FromAgent: "coordinator", ToAgent: "backend", Body: "do stuff", Read: false},
			{ID: 2, FromAgent: "Human Partner", ToAgent: "coordinator", Body: "please fix this", Read: false},
			{ID: 3, FromAgent: "backend", ToAgent: "coordinator", Body: "done", Read: false},
		}

		found, msg := hasHumanPartnerMessage(messages)
		if !found {
			t.Fatal("expected to find a Human Partner message")
		}
		if msg.ID != 2 {
			t.Errorf("expected message ID 2, got %d", msg.ID)
		}
		if msg.Body != "please fix this" {
			t.Errorf("unexpected message body: %s", msg.Body)
		}
	})

	t.Run("ignoresReadMessages", func(t *testing.T) {
		messages := []state.Message{
			{ID: 1, FromAgent: "Human Partner", ToAgent: "coordinator", Body: "old message", Read: true},
			{ID: 2, FromAgent: "coordinator", ToAgent: "backend", Body: "do stuff", Read: false},
		}

		found, msg := hasHumanPartnerMessage(messages)
		if found {
			t.Error("should not find read Human Partner messages")
		}
		if msg != nil {
			t.Error("message should be nil when not found")
		}
	})

	t.Run("ignoresAgentMessages", func(t *testing.T) {
		messages := []state.Message{
			{ID: 1, FromAgent: "coordinator", ToAgent: "backend", Body: "build feature", Read: false},
			{ID: 2, FromAgent: "backend", ToAgent: "coordinator", Body: "done", Read: false},
		}

		found, _ := hasHumanPartnerMessage(messages)
		if found {
			t.Error("should not find agent-to-agent messages")
		}
	})

	t.Run("emptyMessages", func(t *testing.T) {
		found, msg := hasHumanPartnerMessage(nil)
		if found {
			t.Error("should return false for nil messages")
		}
		if msg != nil {
			t.Error("msg should be nil for nil messages")
		}

		found, msg = hasHumanPartnerMessage([]state.Message{})
		if found {
			t.Error("should return false for empty messages")
		}
		if msg != nil {
			t.Error("msg should be nil for empty messages")
		}
	})

	t.Run("returnsFirstUnread", func(t *testing.T) {
		messages := []state.Message{
			{ID: 1, FromAgent: "Human Partner", ToAgent: "coordinator", Body: "first", Read: true},
			{ID: 2, FromAgent: "Human Partner", ToAgent: "coordinator", Body: "second", Read: false},
			{ID: 3, FromAgent: "Human Partner", ToAgent: "coordinator", Body: "third", Read: false},
		}

		found, msg := hasHumanPartnerMessage(messages)
		if !found {
			t.Fatal("expected to find message")
		}
		if msg.ID != 2 {
			t.Errorf("expected first unread message (ID 2), got ID %d", msg.ID)
		}
	})
}

func TestWatchForTrigger(t *testing.T) {
	t.Run("detectsGoalChecksumChange", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		sgaiDir := filepath.Join(dir, ".sgai")
		if err := os.MkdirAll(sgaiDir, 0755); err != nil {
			t.Fatal(err)
		}

		originalContent := "---\nflow: |\n  a -> b\n---\n\nOriginal goal.\n"
		if err := os.WriteFile(goalPath, []byte(originalContent), 0644); err != nil {
			t.Fatal(err)
		}

		originalChecksum, err := computeGoalChecksum(goalPath)
		if err != nil {
			t.Fatal(err)
		}

		stateJSONPath := filepath.Join(sgaiDir, "state.json")
		wfState := state.Workflow{
			Status:   state.StatusComplete,
			Messages: []state.Message{},
		}
		if err := state.Save(stateJSONPath, wfState); err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		go func() {
			time.Sleep(500 * time.Millisecond)
			modifiedContent := "---\nflow: |\n  a -> b\n---\n\nModified goal content.\n"
			if err := os.WriteFile(goalPath, []byte(modifiedContent), 0644); err != nil {
				t.Error(err)
			}
		}()

		trigger := watchForTrigger(ctx, dir, stateJSONPath, originalChecksum, 0, "")
		if trigger != triggerGoal {
			t.Errorf("expected trigger %q, got %q", triggerGoal, trigger)
		}
	})

	t.Run("detectsHumanPartnerMessage", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		sgaiDir := filepath.Join(dir, ".sgai")
		if err := os.MkdirAll(sgaiDir, 0755); err != nil {
			t.Fatal(err)
		}

		goalContent := "---\nflow: |\n  a -> b\n---\n\nGoal content.\n"
		if err := os.WriteFile(goalPath, []byte(goalContent), 0644); err != nil {
			t.Fatal(err)
		}

		checksum, err := computeGoalChecksum(goalPath)
		if err != nil {
			t.Fatal(err)
		}

		stateJSONPath := filepath.Join(sgaiDir, "state.json")
		wfState := state.Workflow{
			Status:   state.StatusComplete,
			Messages: []state.Message{},
		}
		if err := state.Save(stateJSONPath, wfState); err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		go func() {
			time.Sleep(500 * time.Millisecond)
			wfState.Messages = append(wfState.Messages, state.Message{
				ID:        1,
				FromAgent: "Human Partner",
				ToAgent:   "coordinator",
				Body:      "new steering instruction",
				Read:      false,
			})
			if err := state.Save(stateJSONPath, wfState); err != nil {
				t.Error(err)
			}
		}()

		trigger := watchForTrigger(ctx, dir, stateJSONPath, checksum, 0, "")
		if trigger != triggerSteering {
			t.Errorf("expected trigger %q, got %q", triggerSteering, trigger)
		}
	})

	t.Run("exitsOnContextCancel", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		sgaiDir := filepath.Join(dir, ".sgai")
		if err := os.MkdirAll(sgaiDir, 0755); err != nil {
			t.Fatal(err)
		}

		goalContent := "---\nflow: |\n  a -> b\n---\n\nGoal content.\n"
		if err := os.WriteFile(goalPath, []byte(goalContent), 0644); err != nil {
			t.Fatal(err)
		}

		checksum, err := computeGoalChecksum(goalPath)
		if err != nil {
			t.Fatal(err)
		}

		stateJSONPath := filepath.Join(sgaiDir, "state.json")
		wfState := state.Workflow{
			Status:   state.StatusComplete,
			Messages: []state.Message{},
		}
		if err := state.Save(stateJSONPath, wfState); err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(500 * time.Millisecond)
			cancel()
		}()

		trigger := watchForTrigger(ctx, dir, stateJSONPath, checksum, 0, "")
		if trigger != triggerNone {
			t.Errorf("expected trigger %q on cancel, got %q", triggerNone, trigger)
		}
	})
}

func TestRunContinuousModePromptObservability(t *testing.T) {
	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	if err := os.MkdirAll(sgaiDir, 0755); err != nil {
		t.Fatal(err)
	}

	stateJSONPath := filepath.Join(sgaiDir, "state.json")
	wfState := state.Workflow{
		Status:   state.StatusWorking,
		Messages: []state.Message{},
		Progress: []state.ProgressEntry{},
	}
	if err := state.Save(stateJSONPath, wfState); err != nil {
		t.Fatal(err)
	}

	updateContinuousModeState(stateJSONPath, "Running continuous mode prompt...", "continuous-mode", "continuous mode prompt started")

	loaded, errLoad := state.Load(stateJSONPath)
	if errLoad != nil {
		t.Fatal(errLoad)
	}

	if loaded.Task != "Running continuous mode prompt..." {
		t.Errorf("expected task 'Running continuous mode prompt...', got %q", loaded.Task)
	}
	if loaded.CurrentAgent != "continuous-mode" {
		t.Errorf("expected currentAgent 'continuous-mode', got %q", loaded.CurrentAgent)
	}
	if len(loaded.Progress) == 0 {
		t.Fatal("expected at least one progress entry")
	}

	lastProgress := loaded.Progress[len(loaded.Progress)-1]
	if lastProgress.Agent != "continuous-mode" {
		t.Errorf("expected progress agent 'continuous-mode', got %q", lastProgress.Agent)
	}
	if lastProgress.Description != "continuous mode prompt started" {
		t.Errorf("expected progress description 'continuous mode prompt started', got %q", lastProgress.Description)
	}
}

func TestReadContinuousModePrompt(t *testing.T) {
	t.Run("withPrompt", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		content := "---\ncontinuousModePrompt: Review and update project status\n---\n\nGoal content.\n"
		if err := os.WriteFile(goalPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		prompt := readContinuousModePrompt(dir)
		if prompt != "Review and update project status" {
			t.Errorf("expected prompt 'Review and update project status', got %q", prompt)
		}
	})

	t.Run("withoutPrompt", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		content := "---\nflow: |\n  a -> b\n---\n\nGoal content.\n"
		if err := os.WriteFile(goalPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		prompt := readContinuousModePrompt(dir)
		if prompt != "" {
			t.Errorf("expected empty prompt, got %q", prompt)
		}
	})

	t.Run("missingGoalFile", func(t *testing.T) {
		dir := t.TempDir()
		prompt := readContinuousModePrompt(dir)
		if prompt != "" {
			t.Errorf("expected empty prompt for missing file, got %q", prompt)
		}
	})
}

func TestMarkMessageAsRead(t *testing.T) {
	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	if err := os.MkdirAll(sgaiDir, 0755); err != nil {
		t.Fatal(err)
	}

	stateJSONPath := filepath.Join(sgaiDir, "state.json")
	wfState := state.Workflow{
		Status: state.StatusWorking,
		Messages: []state.Message{
			{ID: 1, FromAgent: "Human Partner", ToAgent: "coordinator", Body: "fix this", Read: false},
			{ID: 2, FromAgent: "coordinator", ToAgent: "backend", Body: "build", Read: false},
		},
	}
	if err := state.Save(stateJSONPath, wfState); err != nil {
		t.Fatal(err)
	}

	markMessageAsRead(stateJSONPath, 1)

	loaded, err := state.Load(stateJSONPath)
	if err != nil {
		t.Fatal(err)
	}

	if !loaded.Messages[0].Read {
		t.Error("message 1 should be marked as read")
	}
	if loaded.Messages[0].ReadBy != "continuous-mode" {
		t.Errorf("expected readBy 'continuous-mode', got %q", loaded.Messages[0].ReadBy)
	}
	if loaded.Messages[0].ReadAt == "" {
		t.Error("readAt should be set")
	}
	if loaded.Messages[1].Read {
		t.Error("message 2 should remain unread")
	}
}

func TestResetWorkflowForNextCycle(t *testing.T) {
	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	if err := os.MkdirAll(sgaiDir, 0755); err != nil {
		t.Fatal(err)
	}

	stateJSONPath := filepath.Join(sgaiDir, "state.json")
	wfState := state.Workflow{
		Status:              state.StatusComplete,
		InteractiveAutoLock: false,
	}
	if err := state.Save(stateJSONPath, wfState); err != nil {
		t.Fatal(err)
	}

	resetWorkflowForNextCycle(stateJSONPath)

	loaded, err := state.Load(stateJSONPath)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Status != state.StatusWorking {
		t.Errorf("expected status 'working', got %q", loaded.Status)
	}
	if !loaded.InteractiveAutoLock {
		t.Error("expected interactiveAutoLock to be true")
	}
}

func TestGoalMetadataContinuousModePromptParsing(t *testing.T) {
	t.Run("parsesFromFrontmatter", func(t *testing.T) {
		content := []byte("---\ncontinuousModePrompt: Verify all tasks are complete\nflow: |\n  a -> b\n---\n\nGoal.\n")
		metadata, err := parseYAMLFrontmatter(content)
		if err != nil {
			t.Fatal(err)
		}
		if metadata.ContinuousModePrompt != "Verify all tasks are complete" {
			t.Errorf("expected prompt 'Verify all tasks are complete', got %q", metadata.ContinuousModePrompt)
		}
		if metadata.Flow == "" {
			t.Error("flow should still be parsed")
		}
	})

	t.Run("emptyWhenAbsent", func(t *testing.T) {
		content := []byte("---\nflow: |\n  a -> b\n---\n\nGoal.\n")
		metadata, err := parseYAMLFrontmatter(content)
		if err != nil {
			t.Fatal(err)
		}
		if metadata.ContinuousModePrompt != "" {
			t.Errorf("expected empty prompt, got %q", metadata.ContinuousModePrompt)
		}
	})
}

func TestGoalMetadataAutoCronParsing(t *testing.T) {
	t.Run("parsesContinuousModeAuto", func(t *testing.T) {
		content := []byte("---\ncontinuousModeAuto: 1h30m\nflow: |\n  a -> b\n---\n\nGoal.\n")
		metadata, err := parseYAMLFrontmatter(content)
		if err != nil {
			t.Fatal(err)
		}
		if metadata.ContinuousModeAuto != "1h30m" {
			t.Errorf("expected auto '1h30m', got %q", metadata.ContinuousModeAuto)
		}
	})

	t.Run("parsesContinuousModeCron", func(t *testing.T) {
		content := []byte("---\ncontinuousModeCron: '0 * * * *'\nflow: |\n  a -> b\n---\n\nGoal.\n")
		metadata, err := parseYAMLFrontmatter(content)
		if err != nil {
			t.Fatal(err)
		}
		if metadata.ContinuousModeCron != "0 * * * *" {
			t.Errorf("expected cron '0 * * * *', got %q", metadata.ContinuousModeCron)
		}
	})

	t.Run("parsesBothAutoAndCron", func(t *testing.T) {
		content := []byte("---\ncontinuousModeAuto: 30s\ncontinuousModeCron: '*/5 * * * *'\n---\n\nGoal.\n")
		metadata, err := parseYAMLFrontmatter(content)
		if err != nil {
			t.Fatal(err)
		}
		if metadata.ContinuousModeAuto != "30s" {
			t.Errorf("expected auto '30s', got %q", metadata.ContinuousModeAuto)
		}
		if metadata.ContinuousModeCron != "*/5 * * * *" {
			t.Errorf("expected cron '*/5 * * * *', got %q", metadata.ContinuousModeCron)
		}
	})
}

func TestReadContinuousModeAutoCron(t *testing.T) {
	t.Run("parsesAutoDuration", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		content := "---\ncontinuousModeAuto: 2h\n---\n\nGoal.\n"
		if err := os.WriteFile(goalPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		autoDuration, cronExpr := readContinuousModeAutoCron(dir)
		if autoDuration != 2*time.Hour {
			t.Errorf("expected 2h duration, got %v", autoDuration)
		}
		if cronExpr != "" {
			t.Errorf("expected empty cron, got %q", cronExpr)
		}
	})

	t.Run("parsesCronExpression", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		content := "---\ncontinuousModeCron: '0 0 * * *'\n---\n\nGoal.\n"
		if err := os.WriteFile(goalPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		autoDuration, cronExpr := readContinuousModeAutoCron(dir)
		if autoDuration != 0 {
			t.Errorf("expected 0 duration, got %v", autoDuration)
		}
		if cronExpr != "0 0 * * *" {
			t.Errorf("expected cron expression, got %q", cronExpr)
		}
	})

	t.Run("parsesBothAutoAndCron", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		content := "---\ncontinuousModeAuto: 5m\ncontinuousModeCron: '*/10 * * * *'\n---\n\nGoal.\n"
		if err := os.WriteFile(goalPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		autoDuration, cronExpr := readContinuousModeAutoCron(dir)
		if autoDuration != 5*time.Minute {
			t.Errorf("expected 5m duration, got %v", autoDuration)
		}
		if cronExpr != "*/10 * * * *" {
			t.Errorf("expected cron expression, got %q", cronExpr)
		}
	})

	t.Run("invalidDurationReturnsZero", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		content := "---\ncontinuousModeAuto: invalid\n---\n\nGoal.\n"
		if err := os.WriteFile(goalPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		autoDuration, _ := readContinuousModeAutoCron(dir)
		if autoDuration != 0 {
			t.Errorf("expected 0 duration for invalid input, got %v", autoDuration)
		}
	})

	t.Run("missingFileReturnsZero", func(t *testing.T) {
		dir := t.TempDir()

		autoDuration, cronExpr := readContinuousModeAutoCron(dir)
		if autoDuration != 0 {
			t.Errorf("expected 0 duration for missing file, got %v", autoDuration)
		}
		if cronExpr != "" {
			t.Errorf("expected empty cron for missing file, got %q", cronExpr)
		}
	})
}

func TestWatchForTriggerAutoDuration(t *testing.T) {
	t.Run("triggersAfterAutoDuration", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		sgaiDir := filepath.Join(dir, ".sgai")
		if err := os.MkdirAll(sgaiDir, 0755); err != nil {
			t.Fatal(err)
		}

		goalContent := "---\nflow: |\n  a -> b\n---\n\nGoal content.\n"
		if err := os.WriteFile(goalPath, []byte(goalContent), 0644); err != nil {
			t.Fatal(err)
		}

		checksum, err := computeGoalChecksum(goalPath)
		if err != nil {
			t.Fatal(err)
		}

		stateJSONPath := filepath.Join(sgaiDir, "state.json")
		wfState := state.Workflow{
			Status:   state.StatusComplete,
			Messages: []state.Message{},
		}
		if err := state.Save(stateJSONPath, wfState); err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		autoDuration := 100 * time.Millisecond
		trigger := watchForTrigger(ctx, dir, stateJSONPath, checksum, autoDuration, "")
		if trigger != triggerAuto {
			t.Errorf("expected trigger %q, got %q", triggerAuto, trigger)
		}
	})

	t.Run("goalChangeOverridesAutoDuration", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		sgaiDir := filepath.Join(dir, ".sgai")
		if err := os.MkdirAll(sgaiDir, 0755); err != nil {
			t.Fatal(err)
		}

		originalContent := "---\nflow: |\n  a -> b\n---\n\nOriginal goal.\n"
		if err := os.WriteFile(goalPath, []byte(originalContent), 0644); err != nil {
			t.Fatal(err)
		}

		checksum, err := computeGoalChecksum(goalPath)
		if err != nil {
			t.Fatal(err)
		}

		stateJSONPath := filepath.Join(sgaiDir, "state.json")
		wfState := state.Workflow{
			Status:   state.StatusComplete,
			Messages: []state.Message{},
		}
		if err := state.Save(stateJSONPath, wfState); err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		go func() {
			time.Sleep(50 * time.Millisecond)
			modifiedContent := "---\nflow: |\n  a -> b\n---\n\nModified goal content.\n"
			if err := os.WriteFile(goalPath, []byte(modifiedContent), 0644); err != nil {
				t.Error(err)
			}
		}()

		autoDuration := 1 * time.Second
		trigger := watchForTrigger(ctx, dir, stateJSONPath, checksum, autoDuration, "")
		if trigger != triggerGoal {
			t.Errorf("expected trigger %q, got %q", triggerGoal, trigger)
		}
	})
}
