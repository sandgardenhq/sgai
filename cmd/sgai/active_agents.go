package main

import (
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
		addStringValue(&ids, firstStringValue(p.State.Input, "task_id", "taskId", "id"))
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
	if agent := firstStringValue(p.State.Input, "subagent_type", "subagentType", "subagent", "subagent_name", "agent", "agent_type", "agentType"); agent != "" {
		return strings.TrimSpace(agent)
	}
	return strings.TrimSpace(firstStringValue(p.State.Metadata, "subagent_type", "subagentType", "subagent", "subagent_name", "agent", "agent_type", "agentType"))
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
	if title := firstStringValue(p.State.Input, "description", "title", "command"); title != "" {
		return title
	}
	return firstStringValue(p.State.Metadata, "title", "description")
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
	if sessionID := firstStringValue(p.State.Metadata, "sessionId", "sessionID", "session_id", "session"); sessionID != "" {
		addStringValue(&sessionIDs, sessionID)
	}
	if sessionID := firstStringValue(p.State.Input, "sessionId", "sessionID", "session_id", "session"); sessionID != "" {
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
	if model := modelValue(p.State.Metadata["model"]); model != "" {
		return model
	}
	return firstStringValue(p.State.Metadata, "modelID", "modelId", "model")
}

func modelValue(value any) string {
	if text := stringValue(value); text != "" {
		return text
	}
	model, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	providerID := firstStringValue(model, "providerID", "providerId", "provider_id", "provider")
	modelID := firstStringValue(model, "modelID", "modelId", "model_id", "id", "model")
	if providerID != "" && modelID != "" {
		return providerID + "/" + modelID
	}
	return modelID
}

func firstStringValue(values map[string]any, keys ...string) string {
	if len(values) == 0 {
		return ""
	}
	for _, key := range keys {
		if text := stringValue(values[key]); text != "" {
			return text
		}
	}
	for _, key := range keys {
		for existingKey, value := range values {
			if strings.EqualFold(existingKey, key) {
				if text := stringValue(value); text != "" {
					return text
				}
			}
		}
	}
	return ""
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
