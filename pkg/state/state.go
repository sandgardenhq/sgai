// Package state provides types and functions for managing workflow state.
// The state is persisted as JSON in .sgai/state.json within project directories.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Workflow status constants define the possible states of a sgai workflow.
const (
	StatusWorking         = "working"
	StatusAgentDone       = "agent-done"
	StatusComplete        = "complete"
	StatusWaitingForHuman = "waiting-for-human"
)

// InteractionMode constants define the possible interaction modes of a sgai session.
const (
	ModeInteractive = "interactive"
	ModeSelfDrive   = "self-drive"
	ModeContinuous  = "continuous"
)

// IsHumanPending reports whether the given status indicates the workflow
// is waiting for a human response.
func IsHumanPending(status string) bool {
	return status == StatusWaitingForHuman
}

// NeedsHumanInput reports whether this workflow is actively waiting for
// human input, meaning it has a pending question or message for the human.
func (w Workflow) NeedsHumanInput() bool {
	return w.Status == StatusWaitingForHuman && (w.MultiChoiceQuestion != nil || w.HumanMessage != "")
}

// ValidStatuses contains the workflow status values that agents can set
// via the update_workflow_state tool. StatusWaitingForHuman is excluded
// because it is set only by askUserQuestion and askUserWorkGate tools.
var ValidStatuses = []string{
	StatusWorking,
	StatusAgentDone,
	StatusComplete,
}

// TodoItem represents a single item in the agent's TODO list.
// The structure matches the opencode todo.updated event payload.
type TodoItem struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

// ProgressEntry represents a single progress log entry.
type ProgressEntry struct {
	Timestamp   string `json:"timestamp"`
	Agent       string `json:"agent"`
	Description string `json:"description"`
}

// QuestionItem represents a single question in a multi-question batch.
type QuestionItem struct {
	Question    string   `json:"question"`
	Choices     []string `json:"choices"`
	MultiSelect bool     `json:"multiSelect"`
}

// MultiChoiceQuestion stores an active batch of questions for human response.
// Used by the AskUserQuestion tool to present choices to the human partner.
type MultiChoiceQuestion struct {
	Questions  []QuestionItem `json:"questions"`
	IsWorkGate bool           `json:"isWorkGate,omitempty"`
}

// Workflow represents the complete workflow state for a sgai session.
// It tracks progress and workflow status.
type Workflow struct {
	Status              string               `json:"status"`
	Task                string               `json:"task"`
	Progress            []ProgressEntry      `json:"progress"`
	HumanMessage        string               `json:"humanMessage"`
	MultiChoiceQuestion *MultiChoiceQuestion `json:"multiChoiceQuestion,omitempty"`
	Todos               []TodoItem           `json:"todos,omitempty"`
	ProjectTodos        []TodoItem           `json:"projectTodos,omitempty"`
	SessionID           string               `json:"sessionId,omitempty"`

	InteractionMode string `json:"interactionMode,omitempty"`

	// Summary is a single-sentence summary of the project goal.
	// Generated automatically when GOAL.md is saved or workspace starts,
	// unless SummaryManual is true (indicating user has manually edited it).
	Summary string `json:"summary,omitempty"`

	// SummaryManual indicates whether the summary was manually edited by user.
	// When true, automatic summary generation is skipped.
	SummaryManual bool `json:"summaryManual,omitempty"`
}

// ToolsAllowed reports whether the current interaction mode permits
// human-interaction tools (ask_user_question, ask_user_work_gate).
func (w Workflow) ToolsAllowed() bool {
	return w.InteractionMode == ModeInteractive
}

func load(path string) (Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Workflow{}, err
	}
	var wf Workflow
	if err := json.Unmarshal(data, &wf); err != nil {
		return Workflow{}, err
	}
	return wf, nil
}

func save(path string, wf Workflow) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(wf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
