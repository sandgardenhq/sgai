package state

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWorkflow_UnmarshalJSON_NewFormat(t *testing.T) {
	input := `{
		"status": "working",
		"agentSequence": [
			{"agent": "coordinator", "startTime": "2025-12-21T18:26:00Z", "isCurrent": false},
			{"agent": "backend-go-developer", "startTime": "2025-12-21T18:27:00Z", "isCurrent": true}
		]
	}`

	var workflow Workflow
	if err := json.Unmarshal([]byte(input), &workflow); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if workflow.Status != "working" {
		t.Errorf("Status = %q; want %q", workflow.Status, "working")
	}

	want := []AgentSequenceEntry{
		{Agent: "coordinator", StartTime: "2025-12-21T18:26:00Z", IsCurrent: false},
		{Agent: "backend-go-developer", StartTime: "2025-12-21T18:27:00Z", IsCurrent: true},
	}

	if len(workflow.AgentSequence) != len(want) {
		t.Fatalf("len(AgentSequence) = %d; want %d", len(workflow.AgentSequence), len(want))
	}

	for i, got := range workflow.AgentSequence {
		if got != want[i] {
			t.Errorf("AgentSequence[%d] = %+v; want %+v", i, got, want[i])
		}
	}
}

func TestWorkflow_UnmarshalJSON_EmptySequence(t *testing.T) {
	input := `{
		"status": "working"
	}`

	var workflow Workflow
	if err := json.Unmarshal([]byte(input), &workflow); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if workflow.Status != "working" {
		t.Errorf("Status = %q; want %q", workflow.Status, "working")
	}

	if workflow.AgentSequence != nil {
		t.Errorf("AgentSequence = %+v; want nil", workflow.AgentSequence)
	}
}

func TestWorkflow_RoundTrip(t *testing.T) {
	original := Workflow{
		Status: "working",
		Task:   "test task",
		AgentSequence: []AgentSequenceEntry{
			{Agent: "coordinator", StartTime: "2025-12-21T18:26:00Z", IsCurrent: false},
			{Agent: "backend-go-developer", StartTime: "2025-12-21T18:27:00Z", IsCurrent: true},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Workflow
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Status != original.Status {
		t.Errorf("Status = %q; want %q", decoded.Status, original.Status)
	}

	if decoded.Task != original.Task {
		t.Errorf("Task = %q; want %q", decoded.Task, original.Task)
	}

	if len(decoded.AgentSequence) != len(original.AgentSequence) {
		t.Fatalf("len(AgentSequence) = %d; want %d", len(decoded.AgentSequence), len(original.AgentSequence))
	}

	for i, got := range decoded.AgentSequence {
		if got != original.AgentSequence[i] {
			t.Errorf("AgentSequence[%d] = %+v; want %+v", i, got, original.AgentSequence[i])
		}
	}
}

func TestSave_CreatesParentDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, ".sgai", "state.json")

	workflow := Workflow{
		Status: "working",
		Task:   "test task",
	}

	if err := save(nestedPath, workflow); err != nil {
		t.Fatalf("save() failed: %v", err)
	}

	loaded, err := load(nestedPath)
	if err != nil {
		t.Fatalf("load() after save() failed: %v", err)
	}

	if loaded.Status != workflow.Status {
		t.Errorf("Status = %q; want %q", loaded.Status, workflow.Status)
	}

	if loaded.Task != workflow.Task {
		t.Errorf("Task = %q; want %q", loaded.Task, workflow.Task)
	}
}

func TestIsHumanPending(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"waitingForHuman", StatusWaitingForHuman, true},
		{"working", StatusWorking, false},
		{"agentDone", StatusAgentDone, false},
		{"complete", StatusComplete, false},
		{"empty", "", false},
		{"whitespace", " ", false},
		{"tab", "\t", false},
		{"similarWithPlural", "waiting-for-humans", false},
		{"uppercaseVariant", "WAITING-FOR-HUMAN", false},
		{"trailingSpace", "waiting-for-human ", false},
		{"leadingSpace", " waiting-for-human", false},
		{"arbitraryString", "some-random-status", false},
		{"removedHumanCommunication", "human-communication", false},
		{"removedAutoSession", "auto-session", false},
		{"partialMatch", "waiting", false},
		{"partialMatchDash", "waiting-for", false},
		{"newline", "waiting-for-human\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsHumanPending(tt.status); got != tt.want {
				t.Errorf("IsHumanPending(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestNeedsHumanInput(t *testing.T) {
	t.Run("waitingWithMessage", func(t *testing.T) {
		w := Workflow{Status: StatusWaitingForHuman, HumanMessage: "please respond"}
		if !w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = true")
		}
	})

	t.Run("waitingWithMultiChoice", func(t *testing.T) {
		w := Workflow{
			Status: StatusWaitingForHuman,
			MultiChoiceQuestion: &MultiChoiceQuestion{
				Questions: []QuestionItem{{Question: "pick one", Choices: []string{"a", "b"}}},
			},
		}
		if !w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = true")
		}
	})

	t.Run("waitingWithBoth", func(t *testing.T) {
		w := Workflow{
			Status:       StatusWaitingForHuman,
			HumanMessage: "choose",
			MultiChoiceQuestion: &MultiChoiceQuestion{
				Questions: []QuestionItem{{Question: "pick one", Choices: []string{"a", "b"}}},
			},
		}
		if !w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = true")
		}
	})

	t.Run("waitingWithoutContent", func(t *testing.T) {
		w := Workflow{Status: StatusWaitingForHuman}
		if w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = false")
		}
	})

	t.Run("workingWithMessage", func(t *testing.T) {
		w := Workflow{Status: StatusWorking, HumanMessage: "please respond"}
		if w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = false")
		}
	})

	t.Run("emptyWorkflow", func(t *testing.T) {
		w := Workflow{}
		if w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = false")
		}
	})

	t.Run("workingWithBothFieldsSet", func(t *testing.T) {
		w := Workflow{
			Status:       StatusWorking,
			HumanMessage: "choose",
			MultiChoiceQuestion: &MultiChoiceQuestion{
				Questions: []QuestionItem{{Question: "q", Choices: []string{"a"}}},
			},
		}
		if w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = false when status is working")
		}
	})

	t.Run("agentDoneWithBothFieldsSet", func(t *testing.T) {
		w := Workflow{
			Status:       StatusAgentDone,
			HumanMessage: "choose",
			MultiChoiceQuestion: &MultiChoiceQuestion{
				Questions: []QuestionItem{{Question: "q", Choices: []string{"a"}}},
			},
		}
		if w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = false when status is agent-done")
		}
	})

	t.Run("completeWithBothFieldsSet", func(t *testing.T) {
		w := Workflow{
			Status:       StatusComplete,
			HumanMessage: "choose",
			MultiChoiceQuestion: &MultiChoiceQuestion{
				Questions: []QuestionItem{{Question: "q", Choices: []string{"a"}}},
			},
		}
		if w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = false when status is complete")
		}
	})

	t.Run("emptyStatusWithBothFieldsSet", func(t *testing.T) {
		w := Workflow{
			Status:       "",
			HumanMessage: "choose",
			MultiChoiceQuestion: &MultiChoiceQuestion{
				Questions: []QuestionItem{{Question: "q", Choices: []string{"a"}}},
			},
		}
		if w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = false when status is empty")
		}
	})

	t.Run("waitingWithMultiChoiceOnly", func(t *testing.T) {
		w := Workflow{
			Status: StatusWaitingForHuman,
			MultiChoiceQuestion: &MultiChoiceQuestion{
				Questions: []QuestionItem{{Question: "q", Choices: []string{"a"}}},
			},
		}
		if !w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = true with only MultiChoiceQuestion")
		}
	})

	t.Run("waitingWithMessageOnly", func(t *testing.T) {
		w := Workflow{Status: StatusWaitingForHuman, HumanMessage: "question"}
		if !w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = true with only HumanMessage")
		}
	})

	t.Run("waitingWithEmptyMessageAndNilQuestion", func(t *testing.T) {
		w := Workflow{Status: StatusWaitingForHuman, HumanMessage: ""}
		if w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = false with empty message and nil question")
		}
	})

	t.Run("waitingWithWorkGateQuestion", func(t *testing.T) {
		w := Workflow{
			Status: StatusWaitingForHuman,
			MultiChoiceQuestion: &MultiChoiceQuestion{
				Questions:  []QuestionItem{{Question: "q", Choices: []string{"yes", "no"}}},
				IsWorkGate: true,
			},
		}
		if !w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = true for work gate question")
		}
	})
}

func TestInteractionMode(t *testing.T) {
	t.Run("toolsAllowed", func(t *testing.T) {
		cases := []struct {
			name string
			mode string
			want bool
		}{
			{"selfDrive", ModeSelfDrive, false},
			{"brainstorming", ModeBrainstorming, true},
			{"building", ModeBuilding, false},
			{"retrospective", ModeRetrospective, true},
			{"empty", "", false},
			{"unknown", "unknown-mode", false},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				w := Workflow{InteractionMode: tc.mode}
				if got := w.ToolsAllowed(); got != tc.want {
					t.Errorf("ToolsAllowed() = %v; want %v", got, tc.want)
				}
			})
		}
	})

	t.Run("explicitModeChecks", func(t *testing.T) {
		cases := []struct {
			name        string
			mode        string
			isSelfDrive bool
			isBuilding  bool
		}{
			{"selfDrive", ModeSelfDrive, true, false},
			{"brainstorming", ModeBrainstorming, false, false},
			{"building", ModeBuilding, false, true},
			{"retrospective", ModeRetrospective, false, false},
			{"empty", "", false, false},
			{"unknown", "unknown-mode", false, false},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				w := Workflow{InteractionMode: tc.mode}
				if got := (w.InteractionMode == ModeSelfDrive); got != tc.isSelfDrive {
					t.Errorf("(mode == ModeSelfDrive) = %v; want %v", got, tc.isSelfDrive)
				}
				if got := (w.InteractionMode == ModeBuilding); got != tc.isBuilding {
					t.Errorf("(mode == ModeBuilding) = %v; want %v", got, tc.isBuilding)
				}
			})
		}
	})

	t.Run("modeTransitions", func(t *testing.T) {
		t.Run("brainstormingToBuilding", func(t *testing.T) {
			w := Workflow{InteractionMode: ModeBrainstorming}
			if w.InteractionMode == ModeSelfDrive {
				t.Error("brainstorming should not be self-drive")
			}
			w.InteractionMode = ModeBuilding
			if w.InteractionMode != ModeBuilding {
				t.Error("mode should be building")
			}
			if w.ToolsAllowed() {
				t.Error("building should not allow tools")
			}
		})

		t.Run("buildingToRetrospective", func(t *testing.T) {
			w := Workflow{InteractionMode: ModeBuilding}
			if w.ToolsAllowed() {
				t.Error("building should not allow tools")
			}
			w.InteractionMode = ModeRetrospective
			if !w.ToolsAllowed() {
				t.Error("retrospective should allow tools")
			}
			if w.InteractionMode == ModeSelfDrive {
				t.Error("retrospective should not be self-drive")
			}
		})

		t.Run("selfDriveNeverAllowsTools", func(t *testing.T) {
			w := Workflow{InteractionMode: ModeSelfDrive}
			if w.ToolsAllowed() {
				t.Error("self-drive should never allow tools")
			}
			if w.InteractionMode != ModeSelfDrive {
				t.Error("self-drive mode should be self-drive")
			}
		})
	})

	t.Run("roundTrip", func(t *testing.T) {
		tmpDir := t.TempDir()
		stPath := filepath.Join(tmpDir, "state.json")

		for _, mode := range []string{ModeSelfDrive, ModeBrainstorming, ModeBuilding, ModeRetrospective, ""} {
			t.Run(mode, func(t *testing.T) {
				wf := Workflow{InteractionMode: mode, Status: StatusWorking}
				if errSave := save(stPath, wf); errSave != nil {
					t.Fatal(errSave)
				}
				loaded, errLoad := load(stPath)
				if errLoad != nil {
					t.Fatal(errLoad)
				}
				if loaded.InteractionMode != mode {
					t.Errorf("InteractionMode = %q after round-trip; want %q", loaded.InteractionMode, mode)
				}
			})
		}
	})

	t.Run("emptyModeOmittedFromJSON", func(t *testing.T) {
		wf := Workflow{Status: StatusWorking}
		data, errMarshal := json.Marshal(wf)
		if errMarshal != nil {
			t.Fatal(errMarshal)
		}
		var raw map[string]json.RawMessage
		if errUnmarshal := json.Unmarshal(data, &raw); errUnmarshal != nil {
			t.Fatal(errUnmarshal)
		}
		if _, exists := raw["interactionMode"]; exists {
			t.Error("interactionMode should be omitted from JSON when empty")
		}
	})
}

func TestSave_ExistingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	sgaiDir := filepath.Join(tmpDir, ".sgai")
	if err := os.MkdirAll(sgaiDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	statePath := filepath.Join(sgaiDir, "state.json")

	workflow := Workflow{Status: "complete"}

	if err := save(statePath, workflow); err != nil {
		t.Fatalf("save() failed: %v", err)
	}

	loaded, err := load(statePath)
	if err != nil {
		t.Fatalf("load() after save() failed: %v", err)
	}

	if loaded.Status != "complete" {
		t.Errorf("Status = %q; want %q", loaded.Status, "complete")
	}
}

func TestCoordinatorOnUpdate(t *testing.T) {
	t.Run("callbackFiredOnUpdateState", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, ".sgai", "state.json")

		coord := NewCoordinatorEmpty(path)

		var callCount int
		coord.OnUpdate(func() { callCount++ })

		if err := coord.UpdateState(func(wf *Workflow) { wf.Task = "work" }); err != nil {
			t.Fatal(err)
		}
		if callCount != 1 {
			t.Errorf("callCount = %d; want 1", callCount)
		}

		if err := coord.UpdateState(func(wf *Workflow) { wf.Task = "more work" }); err != nil {
			t.Fatal(err)
		}
		if callCount != 2 {
			t.Errorf("callCount = %d; want 2", callCount)
		}
	})

	t.Run("noCallbackNoError", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, ".sgai", "state.json")

		coord := NewCoordinatorEmpty(path)
		if err := coord.UpdateState(func(wf *Workflow) { wf.Task = "work" }); err != nil {
			t.Fatal(err)
		}
	})
}

func TestNewCoordinatorWith(t *testing.T) {
	t.Run("persistsStateToDisk", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, ".sgai", "state.json")

		wf := Workflow{
			Status:       StatusWorking,
			CurrentAgent: "test-agent",
			Task:         "doing things",
		}
		coord, err := NewCoordinatorWith(path, wf)
		if err != nil {
			t.Fatalf("NewCoordinatorWith() error: %v", err)
		}

		if coord.State().CurrentAgent != "test-agent" {
			t.Errorf("CurrentAgent = %q; want %q", coord.State().CurrentAgent, "test-agent")
		}

		loaded, errLoad := load(path)
		if errLoad != nil {
			t.Fatalf("load() after NewCoordinatorWith() failed: %v", errLoad)
		}
		if loaded.CurrentAgent != "test-agent" {
			t.Errorf("persisted CurrentAgent = %q; want %q", loaded.CurrentAgent, "test-agent")
		}
		if loaded.Status != StatusWorking {
			t.Errorf("persisted Status = %q; want %q", loaded.Status, StatusWorking)
		}
	})

	t.Run("createsParentDirectories", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "nested", ".sgai", "state.json")

		wf := Workflow{Status: StatusComplete}
		if _, err := NewCoordinatorWith(path, wf); err != nil {
			t.Fatalf("NewCoordinatorWith() should create parent dirs, got error: %v", err)
		}

		loaded, errLoad := load(path)
		if errLoad != nil {
			t.Fatalf("load() failed: %v", errLoad)
		}
		if loaded.Status != StatusComplete {
			t.Errorf("Status = %q; want %q", loaded.Status, StatusComplete)
		}
	})
}

func waitForWaitingForHuman(t *testing.T, coord *Coordinator) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		if coord.State().Status == StatusWaitingForHuman {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for waiting-for-human status, got %q", coord.State().Status)
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func TestAskAndWaitPreservesStateOnContextCancel(t *testing.T) {
	t.Run("contextCancelKeepsWaitingForHumanState", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".sgai", "state.json")

		coord, err := NewCoordinatorWith(path, Workflow{
			Status:          StatusWorking,
			InteractionMode: ModeBrainstorming,
		})
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		question := &MultiChoiceQuestion{
			Questions: []QuestionItem{
				{Question: "Pick one", Choices: []string{"A", "B"}},
			},
		}

		errCh := make(chan error, 1)
		go func() {
			_, err := coord.AskAndWait(ctx, question, "Pick one")
			errCh <- err
		}()

		waitForWaitingForHuman(t, coord)

		cancel()

		err = <-errCh
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}

		afterState := coord.State()
		if afterState.Status != StatusWaitingForHuman {
			t.Errorf("status should remain waiting-for-human after context cancel, got %q", afterState.Status)
		}
		if afterState.MultiChoiceQuestion == nil {
			t.Error("question should persist after context cancel")
		}
		if afterState.HumanMessage != "Pick one" {
			t.Errorf("humanMessage should persist after context cancel, got %q", afterState.HumanMessage)
		}
	})

	t.Run("respondClearsStateNormally", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".sgai", "state.json")

		coord, err := NewCoordinatorWith(path, Workflow{
			Status:          StatusWorking,
			InteractionMode: ModeBrainstorming,
		})
		if err != nil {
			t.Fatal(err)
		}

		question := &MultiChoiceQuestion{
			Questions: []QuestionItem{
				{Question: "Pick one", Choices: []string{"A", "B"}},
			},
		}

		type result struct {
			answer string
			err    error
		}
		done := make(chan result, 1)
		go func() {
			a, e := coord.AskAndWait(context.Background(), question, "Pick one")
			done <- result{a, e}
		}()

		waitForWaitingForHuman(t, coord)
		coord.Respond("A")

		r := <-done
		if r.err != nil {
			t.Fatalf("unexpected error: %v", r.err)
		}
		if r.answer != "A" {
			t.Errorf("expected answer 'A', got %q", r.answer)
		}

		afterState := coord.State()
		if afterState.Status == StatusWaitingForHuman {
			t.Error("status should not be waiting-for-human after successful response")
		}
		if afterState.MultiChoiceQuestion != nil {
			t.Error("question should be cleared after successful response")
		}
	})

	t.Run("lateResponseAvailableOnNextAskAndWait", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".sgai", "state.json")

		coord, err := NewCoordinatorWith(path, Workflow{
			Status:          StatusWorking,
			InteractionMode: ModeBrainstorming,
		})
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		question := &MultiChoiceQuestion{
			Questions: []QuestionItem{
				{Question: "Pick one", Choices: []string{"A", "B"}},
			},
		}

		errCh := make(chan error, 1)
		go func() {
			_, err := coord.AskAndWait(ctx, question, "Pick one")
			errCh <- err
		}()

		waitForWaitingForHuman(t, coord)
		cancel()
		<-errCh

		coord.Respond("A")

		type result struct {
			answer string
			err    error
		}
		done := make(chan result, 1)
		go func() {
			a, e := coord.AskAndWait(context.Background(), question, "Pick one")
			done <- result{a, e}
		}()

		r := <-done
		if r.err != nil {
			t.Fatalf("unexpected error: %v", r.err)
		}
		if r.answer != "A" {
			t.Errorf("expected late answer 'A', got %q", r.answer)
		}
	})

	t.Run("firstRespondWinsAfterTimeout", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".sgai", "state.json")

		coord, err := NewCoordinatorWith(path, Workflow{
			Status:          StatusWorking,
			InteractionMode: ModeBrainstorming,
		})
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		question := &MultiChoiceQuestion{
			Questions: []QuestionItem{
				{Question: "Pick one", Choices: []string{"A", "B"}},
			},
		}

		errCh := make(chan error, 1)
		go func() {
			_, err := coord.AskAndWait(ctx, question, "Pick one")
			errCh <- err
		}()

		waitForWaitingForHuman(t, coord)
		cancel()
		<-errCh

		coord.Respond("A")
		coord.Respond("B")

		type result struct {
			answer string
			err    error
		}
		done := make(chan result, 1)
		go func() {
			a, e := coord.AskAndWait(context.Background(), question, "Pick one")
			done <- result{a, e}
		}()

		r := <-done
		if r.err != nil {
			t.Fatalf("unexpected error: %v", r.err)
		}
		if r.answer != "A" {
			t.Errorf("expected first answer 'A' to win (channel-in-channel buffering), got %q", r.answer)
		}
	})
}

func TestMCPTimeoutSurvival(t *testing.T) {
	t.Run("questionStatePreservedAfterMCPTimeout", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".sgai", "state.json")

		coord, err := NewCoordinatorWith(path, Workflow{
			Status:          StatusWorking,
			InteractionMode: ModeBrainstorming,
		})
		if err != nil {
			t.Fatal(err)
		}

		question := &MultiChoiceQuestion{
			Questions: []QuestionItem{
				{Question: "Which option?", Choices: []string{"A", "B"}},
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)
		go func() {
			_, err := coord.AskAndWait(ctx, question, "Which option?")
			errCh <- err
		}()

		waitForWaitingForHuman(t, coord)
		cancel()
		if err := <-errCh; !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled from MCP timeout simulation, got %v", err)
		}

		afterTimeout := coord.State()
		if afterTimeout.Status != StatusWaitingForHuman {
			t.Errorf("status should remain waiting-for-human after MCP timeout, got %q", afterTimeout.Status)
		}
		if afterTimeout.MultiChoiceQuestion == nil {
			t.Error("question should persist after MCP timeout so UI can display it")
		}
	})

	t.Run("humanAnswerAfterMCPTimeoutDeliveredOnRetry", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".sgai", "state.json")

		coord, err := NewCoordinatorWith(path, Workflow{
			Status:          StatusWorking,
			InteractionMode: ModeBrainstorming,
		})
		if err != nil {
			t.Fatal(err)
		}

		question := &MultiChoiceQuestion{
			Questions: []QuestionItem{
				{Question: "Which option?", Choices: []string{"A", "B"}},
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)
		go func() {
			_, err := coord.AskAndWait(ctx, question, "Which option?")
			errCh <- err
		}()

		waitForWaitingForHuman(t, coord)
		cancel()
		<-errCh

		coord.Respond("A")

		type result struct {
			answer string
			err    error
		}
		done := make(chan result, 1)
		go func() {
			a, e := coord.AskAndWait(context.Background(), question, "Which option?")
			done <- result{a, e}
		}()

		r := <-done
		if r.err != nil {
			t.Fatalf("unexpected error on retry: %v", r.err)
		}
		if r.answer != "A" {
			t.Errorf("expected buffered answer 'A' after timeout, got %q", r.answer)
		}
	})

	t.Run("humanAnswerArrivesAfterRetryStarts", func(t *testing.T) {
		simulateTimeoutThenRetry(t, "B")
	})
}

func simulateTimeoutThenRetry(t *testing.T, expectedAnswer string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".sgai", "state.json")

	coord, err := NewCoordinatorWith(path, Workflow{
		Status:          StatusWorking,
		InteractionMode: ModeBrainstorming,
	})
	if err != nil {
		t.Fatal(err)
	}

	question := &MultiChoiceQuestion{
		Questions: []QuestionItem{
			{Question: "Pick one", Choices: []string{"A", "B"}},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := coord.AskAndWait(ctx, question, "Pick one")
		errCh <- err
	}()

	waitForWaitingForHuman(t, coord)
	cancel()
	<-errCh

	type result struct {
		answer string
		err    error
	}
	done := make(chan result, 1)
	go func() {
		a, e := coord.AskAndWait(context.Background(), question, "Pick one")
		done <- result{a, e}
	}()

	waitForWaitingForHuman(t, coord)
	coord.Respond(expectedAnswer)

	r := <-done
	if r.err != nil {
		t.Fatalf("unexpected error: %v", r.err)
	}
	if r.answer != expectedAnswer {
		t.Errorf("expected answer %q, got %q", expectedAnswer, r.answer)
	}
}

func TestAskAndWaitChannelInChannelIsolation(t *testing.T) {
	t.Run("sequentialCallsDoNotShareState", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".sgai", "state.json")

		coord, err := NewCoordinatorWith(path, Workflow{
			Status:          StatusWorking,
			InteractionMode: ModeBrainstorming,
		})
		if err != nil {
			t.Fatal(err)
		}

		question := &MultiChoiceQuestion{
			Questions: []QuestionItem{
				{Question: "Pick one", Choices: []string{"A", "B"}},
			},
		}

		for i, expected := range []string{"first", "second", "third"} {
			type result struct {
				answer string
				err    error
			}
			done := make(chan result, 1)
			go func() {
				a, e := coord.AskAndWait(context.Background(), question, "Pick one")
				done <- result{a, e}
			}()

			waitForWaitingForHuman(t, coord)
			coord.Respond(expected)

			r := <-done
			if r.err != nil {
				t.Fatalf("call %d: unexpected error: %v", i, r.err)
			}
			if r.answer != expected {
				t.Errorf("call %d: expected answer %q, got %q", i, expected, r.answer)
			}

			afterState := coord.State()
			if afterState.Status == StatusWaitingForHuman {
				t.Errorf("call %d: status should not be waiting-for-human after response", i)
			}
		}
	})

	t.Run("timeoutThenRetryDoesNotMixAnswers", func(t *testing.T) {
		simulateTimeoutThenRetry(t, "correct-answer")
	})
}

func TestAgentCancelWatchdog(t *testing.T) {
	t.Run("setAndGetAgentCancel", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".sgai", "state.json")
		coord := NewCoordinatorEmpty(path)

		var cancelled bool
		cancel := func() { cancelled = true }
		coord.SetAgentCancel(cancel)

		got := coord.GetAgentCancel()
		if got == nil {
			t.Fatal("expected non-nil cancel func after SetAgentCancel")
		}
		got()
		if !cancelled {
			t.Error("expected cancel func to be callable via GetAgentCancel")
		}
	})

	t.Run("getAgentCancelReturnsNilBeforeSet", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".sgai", "state.json")
		coord := NewCoordinatorEmpty(path)

		if coord.GetAgentCancel() != nil {
			t.Error("expected nil cancel func before SetAgentCancel")
		}
	})

	t.Run("resetClearsAgentCancel", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".sgai", "state.json")
		coord := NewCoordinatorEmpty(path)

		coord.SetAgentCancel(func() {})
		coord.ResetAgentDoneWatchdog()

		if coord.GetAgentCancel() != nil {
			t.Error("expected nil cancel func after ResetAgentDoneWatchdog")
		}
	})

	t.Run("watchdogFiresOnStartAgentDoneWatchdog", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".sgai", "state.json")
		coord := NewCoordinatorEmpty(path)

		cancelled := make(chan struct{})
		cancel := func() { close(cancelled) }

		coord.mu.Lock()
		coord.doneTimer = time.AfterFunc(50*time.Millisecond, cancel)
		coord.mu.Unlock()

		select {
		case <-cancelled:
		case <-time.After(2 * time.Second):
			t.Fatal("watchdog did not fire within deadline")
		}
	})

	t.Run("stopCancelsPendingTimer", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".sgai", "state.json")
		coord := NewCoordinatorEmpty(path)

		fired := make(chan struct{}, 1)
		cancel := func() { fired <- struct{}{} }
		coord.SetAgentCancel(cancel)

		coord.mu.Lock()
		coord.doneTimer = time.AfterFunc(50*time.Millisecond, cancel)
		coord.mu.Unlock()

		coord.Stop()

		select {
		case <-fired:
			t.Error("watchdog should not have fired after Stop()")
		case <-time.After(200 * time.Millisecond):
		}
	})

	t.Run("resetAllowsWatchdogToStartOnSubsequentAgentRun", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".sgai", "state.json")
		coord := NewCoordinatorEmpty(path)

		coord.SetAgentCancel(func() {})
		coord.StartAgentDoneWatchdog(coord.GetAgentCancel())

		if !coord.IsShuttingDown() {
			t.Error("expected IsShuttingDown true after first StartAgentDoneWatchdog")
		}

		coord.ResetAgentDoneWatchdog()

		if coord.IsShuttingDown() {
			t.Error("expected IsShuttingDown false after ResetAgentDoneWatchdog")
		}

		coord.SetAgentCancel(func() {})
		coord.StartAgentDoneWatchdog(coord.GetAgentCancel())

		if !coord.IsShuttingDown() {
			t.Error("expected IsShuttingDown true after second StartAgentDoneWatchdog following Reset")
		}
		coord.Stop()
	})

	t.Run("startWatchdogWithNilCancelIsNoOp", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".sgai", "state.json")
		coord := NewCoordinatorEmpty(path)

		coord.StartAgentDoneWatchdog(nil)

		if coord.IsShuttingDown() {
			t.Error("IsShuttingDown should be false after StartAgentDoneWatchdog(nil)")
		}
	})

	t.Run("setAgentCancelConcurrentAccess", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".sgai", "state.json")
		coord := NewCoordinatorEmpty(path)

		done := make(chan struct{})
		for range 10 {
			go func() {
				coord.SetAgentCancel(func() {})
				_ = coord.GetAgentCancel()
				done <- struct{}{}
			}()
		}
		for range 10 {
			<-done
		}
	})
}

func TestProgressEntry_UnmarshalJSON_NewFormat(t *testing.T) {
	input := `{"timestamp":"2026-01-01T10:43:36-08:00","agent":"coordinator","description":"Started assessing GOAL.md"}`

	var entry ProgressEntry
	if err := json.Unmarshal([]byte(input), &entry); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if entry.Timestamp != "2026-01-01T10:43:36-08:00" {
		t.Errorf("Timestamp = %q; want %q", entry.Timestamp, "2026-01-01T10:43:36-08:00")
	}

	if entry.Agent != "coordinator" {
		t.Errorf("Agent = %q; want %q", entry.Agent, "coordinator")
	}

	if entry.Description != "Started assessing GOAL.md" {
		t.Errorf("Description = %q; want %q", entry.Description, "Started assessing GOAL.md")
	}
}
