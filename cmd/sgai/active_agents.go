package main

import (
	"encoding/json"
	"slices"
	"strings"
	"sync"
)

type activeAgent struct {
	ID        string
	Agent     string
	Title     string
	SessionID string
	Model     string
	Status    string
}

type activeAgentTaskInput struct {
	Description  string `json:"description"`
	Prompt       string `json:"prompt"`
	SubagentType string `json:"subagent_type"`
	TaskID       string `json:"task_id"`
	Command      string `json:"command"`
}

type activeAgentTaskMetadata struct {
	SessionID string               `json:"sessionId"`
	Model     activeAgentTaskModel `json:"model"`
}

type activeAgentTaskModel struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
}

type activeAgentTracker struct {
	mu     sync.RWMutex
	agents map[string]activeAgent
}

func newActiveAgentTracker() *activeAgentTracker {
	return &activeAgentTracker{agents: make(map[string]activeAgent)}
}

func (t *activeAgentTracker) applyEvent(event streamEvent) bool {
	if t == nil {
		return false
	}
	switch event.Type {
	case "tool", "tool_use":
		return t.applyToolPart(event.Part)
	default:
		return false
	}
}

func (t *activeAgentTracker) applyToolPart(p part) bool {
	if t == nil || !isTaskToolName(p.Tool) || p.State == nil {
		return false
	}
	switch p.State.Status {
	case "pending", "running":
		return t.upsert(p)
	case "completed", "error", "cancelled", "canceled":
		return t.remove(p)
	default:
		return false
	}
}

func (t *activeAgentTracker) upsert(p part) bool {
	next, ok := activeAgentFromPart(p)
	if !ok {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	current, exists := t.agents[next.ID]
	if exists && current == next {
		return false
	}
	t.agents[next.ID] = next
	return true
}

func (t *activeAgentTracker) remove(p part) bool {
	ids := activeAgentIDsFromPart(p)
	sessionIDs := activeAgentSessionIDs(p)
	t.mu.Lock()
	defer t.mu.Unlock()
	changed := false
	for _, id := range ids {
		if _, ok := t.agents[id]; ok {
			delete(t.agents, id)
			changed = true
		}
	}
	for key, agent := range t.agents {
		if slices.Contains(sessionIDs, agent.SessionID) {
			delete(t.agents, key)
			changed = true
		}
	}
	return changed
}

func (t *activeAgentTracker) snapshot() []activeAgent {
	if t == nil {
		return nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	if len(t.agents) == 0 {
		return nil
	}
	result := make([]activeAgent, 0, len(t.agents))
	for _, agent := range t.agents {
		result = append(result, agent)
	}
	slices.SortFunc(result, func(a, b activeAgent) int {
		return strings.Compare(a.ID, b.ID)
	})
	return result
}

func (t *activeAgentTracker) clear() bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.agents) == 0 {
		return false
	}
	t.agents = make(map[string]activeAgent)
	return true
}

func activeAgentFromPart(p part) (activeAgent, bool) {
	if p.State == nil {
		return activeAgent{}, false
	}
	agent := activeAgentName(p)
	if !isDelegatableActiveAgentName(agent) {
		return activeAgent{}, false
	}
	id := activeAgentIDFromPart(p)
	if id == "" {
		return activeAgent{}, false
	}
	return activeAgent{
		ID:        id,
		Agent:     agent,
		Title:     activeAgentTitle(p),
		SessionID: activeAgentSessionID(p),
		Model:     activeAgentModel(p),
		Status:    p.State.Status,
	}, true
}

func activeAgentIDFromPart(p part) string {
	ids := activeAgentIDsFromPart(p)
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}

func activeAgentIDsFromPart(p part) []string {
	var ids []string
	addStringValue(&ids, p.CallID)
	addStringValue(&ids, p.ID)
	if p.State != nil {
		addStringValue(&ids, activeAgentInput(p).TaskID)
	}
	for _, sessionID := range activeAgentSessionIDs(p) {
		addStringValue(&ids, sessionID)
	}
	agent := activeAgentName(p)
	title := activeAgentTitle(p)
	if agent != "" && title != "" {
		addStringValue(&ids, agent+":"+title)
	}
	return ids
}

func activeAgentName(p part) string {
	if p.State == nil {
		return ""
	}
	if agent := activeAgentInput(p).SubagentType; agent != "" {
		return strings.TrimSpace(agent)
	}
	return ""
}

func isDelegatableActiveAgentName(agent string) bool {
	if agent == "" {
		return false
	}
	switch normalizedActiveAgentName(agent) {
	case "coordinator", "stpa-analyst":
		return false
	default:
		return true
	}
}

func normalizedActiveAgentName(agent string) string {
	replacer := strings.NewReplacer("_", "-", " ", "-")
	return replacer.Replace(strings.ToLower(strings.TrimSpace(agent)))
}

func activeAgentTitle(p part) string {
	if p.State == nil {
		return ""
	}
	if p.State.Title != "" {
		return p.State.Title
	}
	if title := activeAgentInput(p).Description; title != "" {
		return title
	}
	return ""
}

func activeAgentSessionID(p part) string {
	sessionIDs := activeAgentSessionIDs(p)
	if len(sessionIDs) == 0 {
		return ""
	}
	return sessionIDs[0]
}

func activeAgentSessionIDs(p part) []string {
	if p.State == nil {
		return nil
	}
	var sessionIDs []string
	addStringValue(&sessionIDs, p.SessionID)
	if sessionID := activeAgentPartMetadata(p).SessionID; sessionID != "" {
		addStringValue(&sessionIDs, sessionID)
	}
	if sessionID := activeAgentStateMetadata(p).SessionID; sessionID != "" {
		addStringValue(&sessionIDs, sessionID)
	}
	if sessionID := activeAgentInput(p).TaskID; sessionID != "" {
		addStringValue(&sessionIDs, sessionID)
	}
	for _, sessionID := range p.State.Output.sessionIDs() {
		addStringValue(&sessionIDs, sessionID)
	}
	return sessionIDs
}

func addStringValue(values *[]string, value string) {
	value = strings.TrimSpace(value)
	if value == "" || slices.Contains(*values, value) {
		return
	}
	*values = append(*values, value)
}

func activeAgentModel(p part) string {
	if p.State == nil {
		return ""
	}
	if model := activeAgentPartMetadata(p).Model.String(); model != "" {
		return model
	}
	if model := activeAgentStateMetadata(p).Model.String(); model != "" {
		return model
	}
	return ""
}

func (m activeAgentTaskModel) String() string {
	if m.ProviderID != "" && m.ModelID != "" {
		return m.ProviderID + "/" + m.ModelID
	}
	return m.ModelID
}

func activeAgentInput(p part) activeAgentTaskInput {
	if p.State == nil {
		return activeAgentTaskInput{}
	}
	return decodeActiveAgentMap[activeAgentTaskInput](p.State.Input)
}

func activeAgentPartMetadata(p part) activeAgentTaskMetadata {
	return decodeActiveAgentMap[activeAgentTaskMetadata](p.Metadata)
}

func activeAgentStateMetadata(p part) activeAgentTaskMetadata {
	if p.State == nil {
		return activeAgentTaskMetadata{}
	}
	return decodeActiveAgentMap[activeAgentTaskMetadata](p.State.Metadata)
}

func decodeActiveAgentMap[T any](values map[string]any) T {
	var result T
	if len(values) == 0 {
		return result
	}
	data, errMarshal := json.Marshal(values)
	if errMarshal != nil {
		return result
	}
	if errUnmarshal := json.Unmarshal(data, &result); errUnmarshal != nil {
		return result
	}
	return result
}

func isTaskToolName(tool string) bool {
	switch strings.ToLower(strings.TrimSpace(tool)) {
	case "task", "tasks":
		return true
	default:
		return false
	}
}

type apiActiveAgent struct {
	ID        string `json:"id"`
	Agent     string `json:"agent"`
	Title     string `json:"title,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	Model     string `json:"model,omitempty"`
	Status    string `json:"status"`
}

func activeAgentAPIEntries(snapshot []activeAgent) []apiActiveAgent {
	if len(snapshot) == 0 {
		return []apiActiveAgent{}
	}
	entries := make([]apiActiveAgent, 0, len(snapshot))
	for _, agent := range snapshot {
		entries = append(entries, apiActiveAgent(agent))
	}
	return entries
}

func (s *session) activeAgentSnapshot(running bool) []activeAgent {
	if s == nil || !running {
		return nil
	}
	s.mu.Lock()
	tracker := s.activeAgents
	s.mu.Unlock()
	if tracker == nil {
		return nil
	}
	return tracker.snapshot()
}

func (s *session) clearActiveAgents() {
	if s == nil {
		return
	}
	s.mu.Lock()
	tracker := s.activeAgents
	s.mu.Unlock()
	if tracker == nil {
		return
	}
	tracker.clear()
}

type sessionRuntime struct {
	activeAgents          *activeAgentTracker
	onActiveAgentsChanged func()
}
