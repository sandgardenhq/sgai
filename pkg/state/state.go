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
	StatusWorking            = "working"
	StatusAgentDone          = "agent-done"
	StatusComplete           = "complete"
	StatusHumanCommunication = "human-communication"
	StatusWaitingForHuman    = "waiting-for-human"
)

// ValidStatuses contains all valid workflow status values for validation.
var ValidStatuses = []string{
	StatusWorking,
	StatusAgentDone,
	StatusComplete,
	StatusHumanCommunication,
	StatusWaitingForHuman,
}

// TodoItem represents a single item in the agent's TODO list.
// The structure matches the opencode todo.updated event payload.
type TodoItem struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

// AgentSequenceEntry represents an agent's visit in the workflow sequence.
// It tracks when the agent started and whether it is the current agent.
type AgentSequenceEntry struct {
	Agent     string `json:"agent"`
	StartTime string `json:"startTime"`
	IsCurrent bool   `json:"isCurrent"`
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

// TokenUsage tracks token counts from a step.
type TokenUsage struct {
	Input      int `json:"input"`
	Output     int `json:"output"`
	Reasoning  int `json:"reasoning"`
	CacheRead  int `json:"cacheRead"`
	CacheWrite int `json:"cacheWrite"`
}

// Add accumulates token counts from another TokenUsage into this one.
func (t *TokenUsage) Add(other TokenUsage) {
	t.Input += other.Input
	t.Output += other.Output
	t.Reasoning += other.Reasoning
	t.CacheRead += other.CacheRead
	t.CacheWrite += other.CacheWrite
}

// StepCost tracks cost for a single step.
type StepCost struct {
	StepID    string     `json:"stepId"`
	Agent     string     `json:"agent"`
	Cost      float64    `json:"cost"`
	Tokens    TokenUsage `json:"tokens"`
	Timestamp string     `json:"timestamp"`
}

// AgentCost aggregates costs for an agent.
type AgentCost struct {
	Agent  string     `json:"agent"`
	Cost   float64    `json:"cost"`
	Tokens TokenUsage `json:"tokens"`
	Steps  []StepCost `json:"steps"`
}

// SessionCost tracks all costs for the session.
type SessionCost struct {
	TotalCost   float64     `json:"totalCost"`
	TotalTokens TokenUsage  `json:"totalTokens"`
	ByAgent     []AgentCost `json:"byAgent"`
}

// Workflow represents the complete workflow state for a sgai session.
// It tracks progress, inter-agent messaging, and workflow status.
type Workflow struct {
	Status              string               `json:"status"`
	Task                string               `json:"task"`
	Progress            []ProgressEntry      `json:"progress"`
	HumanMessage        string               `json:"humanMessage"`
	MultiChoiceQuestion *MultiChoiceQuestion `json:"multiChoiceQuestion,omitempty"`
	Messages            []Message            `json:"messages"`
	GoalChecksum        string               `json:"goalChecksum"`
	VisitCounts         map[string]int       `json:"visitCounts,omitempty"`
	CurrentAgent        string               `json:"currentAgent,omitempty"`
	Todos               []TodoItem           `json:"todos,omitempty"`
	ProjectTodos        []TodoItem           `json:"projectTodos,omitempty"`
	AgentSequence       []AgentSequenceEntry `json:"agentSequence,omitempty"`
	SessionID           string               `json:"sessionId,omitempty"`

	Cost SessionCost `json:"cost"`

	WorkGateApproved bool `json:"workGateApproved,omitempty"`

	// ModelStatuses tracks per-model status in multi-model agents.
	// Key is model ID (agent:modelSpec), value is "model-working", "model-done", or "model-error".
	// This field is ephemeral and cleared when the agent transitions.
	ModelStatuses map[string]string `json:"modelStatuses,omitempty"`

	// CurrentModel tracks the currently executing model in multi-model agents.
	// Format is "agentName:modelSpec".
	CurrentModel string `json:"currentModel,omitempty"`
}

// Message represents an inter-agent message in the workflow system.
type Message struct {
	ID        int    `json:"id"`
	FromAgent string `json:"fromAgent"`
	ToAgent   string `json:"toAgent"`
	Body      string `json:"body"`
	Read      bool   `json:"read"`
	ReadAt    string `json:"readAt,omitempty"`
	ReadBy    string `json:"readBy,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"` // ISO 8601 format (e.g., "2025-12-19T14:30:00Z")
}

// Load reads and parses workflow state from the given JSON file path.
func Load(path string) (Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Workflow{}, err
	}
	var state Workflow
	if err := json.Unmarshal(data, &state); err != nil {
		return Workflow{}, err
	}
	return state, nil
}

// Save writes workflow state to the given path as formatted JSON.
// It creates parent directories if they don't exist.
func Save(path string, state Workflow) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
