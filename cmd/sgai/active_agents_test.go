package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActiveSubagentTrackerTracksTaskLifecycle(t *testing.T) {
	tracker := newActiveAgentTracker()

	changed := tracker.applyEvent(activeTaskEvent("part-1", "call-1", "running", "go", "child-session-1", "openai/gpt-5.5", "Implement backend"))

	assert.True(t, changed)
	snapshot := tracker.snapshot()
	require.Len(t, snapshot, 1)
	assert.Equal(t, "call-1", snapshot[0].ID)
	assert.Equal(t, "go", snapshot[0].Agent)
	assert.Equal(t, "Implement backend", snapshot[0].Title)
	assert.Equal(t, "child-session-1", snapshot[0].SessionID)
	assert.Equal(t, "openai/gpt-5.5", snapshot[0].Model)
	assert.Equal(t, "running", snapshot[0].Status)

	changed = tracker.applyEvent(activeTaskEvent("part-1", "call-1", "completed", "go", "child-session-1", "openai/gpt-5.5", "Implement backend"))

	assert.True(t, changed)
	assert.Empty(t, tracker.snapshot())
}

func TestActiveSubagentTrackerCleansUpErrorsAndClear(t *testing.T) {
	tracker := newActiveAgentTracker()

	tracker.applyEvent(activeTaskEvent("part-1", "call-1", "pending", "go", "child-session-1", "", "Implement backend"))
	tracker.applyEvent(activeTaskEvent("part-2", "call-2", "running", "reviewer", "child-session-2", "", "Review backend"))

	require.Len(t, tracker.snapshot(), 2)
	assert.True(t, tracker.applyEvent(activeTaskEvent("part-1", "call-1", "error", "go", "child-session-1", "", "Implement backend")))
	remaining := tracker.snapshot()
	require.Len(t, remaining, 1)
	assert.Equal(t, "reviewer", remaining[0].Agent)
	assert.True(t, tracker.clear())
	assert.Empty(t, tracker.snapshot())
	assert.False(t, tracker.clear())
}

func TestActiveSubagentTrackerAllowsDuplicateSameAgentCalls(t *testing.T) {
	tracker := newActiveAgentTracker()

	tracker.applyEvent(activeTaskEvent("part-1", "call-1", "running", "go", "child-session-1", "", "First backend task"))
	tracker.applyEvent(activeTaskEvent("part-2", "call-2", "running", "go", "child-session-2", "", "Second backend task"))

	snapshot := tracker.snapshot()
	require.Len(t, snapshot, 2)
	assert.Equal(t, []string{"call-1", "call-2"}, []string{snapshot[0].ID, snapshot[1].ID})

	tracker.applyEvent(activeTaskEvent("part-1", "call-1", "completed", "go", "child-session-1", "", "First backend task"))

	snapshot = tracker.snapshot()
	require.Len(t, snapshot, 1)
	assert.Equal(t, "call-2", snapshot[0].ID)
}

func TestActiveSubagentTrackerRejectsNonDelegatableAgents(t *testing.T) {
	tests := []struct {
		name  string
		agent string
	}{
		{name: "coordinator", agent: "coordinator"},
		{name: "stpa analyst", agent: "stpa-analyst"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := newActiveAgentTracker()

			changed := tracker.applyEvent(activeTaskEvent("part-1", "call-1", "running", tt.agent, "child-session-1", "", "Implement task"))

			assert.False(t, changed)
			assert.Empty(t, tracker.snapshot())
		})
	}
}

func TestActiveSubagentTrackerRemovesFallbackTaskWithStructuredOutputSession(t *testing.T) {
	tracker := newActiveAgentTracker()
	start := decodeStreamEvent(t, `{"type":"tool","part":{"tool":"task","state":{"status":"running","input":{"subagent_type":"go","description":"Implement backend"},"title":"Implement backend"}}}`)
	completed := decodeStreamEvent(t, `{"type":"tool","part":{"tool":"task","state":{"status":"completed","input":{"subagent_type":"go","description":"Implement backend"},"title":"Implement backend","output":{"sessionID":"child-session-1"}}}}`)

	assert.True(t, tracker.applyEvent(start))
	require.Len(t, tracker.snapshot(), 1)

	assert.True(t, tracker.applyEvent(completed))
	assert.Empty(t, tracker.snapshot())
}

func TestActiveSubagentTrackerSupportsConcurrentSnapshots(t *testing.T) {
	tracker := newActiveAgentTracker()
	var wg sync.WaitGroup

	for i := range 25 {
		i := i
		wg.Go(func() {
			id := strconv.Itoa(i)
			tracker.applyEvent(activeTaskEvent("part-"+id, "call-"+id, "running", "go", "child-session-"+id, "", "Backend task "+id))
			_ = tracker.snapshot()
		})
	}
	wg.Wait()

	assert.Len(t, tracker.snapshot(), 25)
}

func TestActiveSubagentTrackerIgnoresMalformedUnknownEvents(t *testing.T) {
	tracker := newActiveAgentTracker()

	events := []streamEvent{
		{Type: "text", Part: part{Text: "hello"}},
		{Type: "tool", Part: part{Tool: "read", State: &toolState{Status: "running", Input: map[string]any{"filePath": "README.md"}}}},
		{Type: "tool", Part: part{Tool: "task", State: &toolState{Status: "running", Input: map[string]any{"description": "missing subagent"}}}},
		{Type: "tool", Part: part{Tool: "task", State: &toolState{Status: "unknown", Input: map[string]any{"subagent_type": "go"}}}},
	}

	for _, event := range events {
		assert.False(t, tracker.applyEvent(event))
	}
	assert.Empty(t, tracker.snapshot())
}

func TestJSONPrettyWriterUpdatesActiveSubagentsFromTaskEvents(t *testing.T) {
	tracker := newActiveAgentTracker()
	notifyCount := 0
	writer := &jsonPrettyWriter{
		w:                     &bytes.Buffer{},
		activeAgents:          tracker,
		onActiveAgentsChanged: func() { notifyCount++ },
	}

	writer.processEvent(activeTaskEvent("part-1", "call-1", "running", "go", "child-session-1", "openai/gpt-5.5", "Implement backend"))

	require.Len(t, tracker.snapshot(), 1)
	assert.Equal(t, 1, notifyCount)

	writer.processEvent(activeTaskEvent("part-1", "call-1", "running", "go", "child-session-1", "openai/gpt-5.5", "Implement backend"))

	assert.Equal(t, 1, notifyCount)

	writer.processEvent(activeTaskEvent("part-1", "call-1", "completed", "go", "child-session-1", "openai/gpt-5.5", "Implement backend"))

	assert.Empty(t, tracker.snapshot())
	assert.Equal(t, 2, notifyCount)
}

func TestJSONPrettyWriterUpdatesActiveSubagentsFromMessagePartUpdatedEvents(t *testing.T) {
	tracker := newActiveAgentTracker()
	notifyCount := 0
	writer := &jsonPrettyWriter{
		w:                     &bytes.Buffer{},
		activeAgents:          tracker,
		onActiveAgentsChanged: func() { notifyCount++ },
	}

	_, errWrite := writer.Write([]byte(
		`{"type":"message.part.updated","properties":{"sessionID":"parent-session","time":1760000000000,"part":{"id":"part-1","sessionID":"child-session-1","type":"tool","callID":"call-1","tool":"task","state":{"status":"running","input":{"subagent_type":"react","description":"Build UI"},"title":"Build UI"},"metadata":{"model":{"providerID":"openai","modelID":"gpt-5.5"}}}}}` + "\n",
	))

	require.NoError(t, errWrite)
	snapshot := tracker.snapshot()
	require.Len(t, snapshot, 1)
	assert.Equal(t, "react", snapshot[0].Agent)
	assert.Equal(t, "Build UI", snapshot[0].Title)
	assert.Equal(t, "child-session-1", snapshot[0].SessionID)
	assert.Equal(t, "openai/gpt-5.5", snapshot[0].Model)
	assert.Equal(t, 1, notifyCount)
}

func TestJSONPrettyWriterClearsActiveSubagentOnStructuredTaskOutput(t *testing.T) {
	tracker := newActiveAgentTracker()
	notifyCount := 0
	var buf bytes.Buffer
	writer := &jsonPrettyWriter{
		w:                     &buf,
		activeAgents:          tracker,
		onActiveAgentsChanged: func() { notifyCount++ },
	}

	_, errWrite := writer.Write([]byte(
		`{"type":"tool","part":{"callID":"call-1","tool":"task","state":{"status":"running","input":{"subagent_type":"go","description":"Implement backend"},"title":"Implement backend"}}}` + "\n" +
			`{"type":"tool","part":{"callID":"call-1","tool":"task","state":{"status":"completed","input":{"subagent_type":"go","description":"Implement backend"},"title":"Implement backend","output":{"sessionID":"child-session-1"}}}}` + "\n",
	))

	require.NoError(t, errWrite)
	assert.Empty(t, tracker.snapshot())
	assert.Equal(t, 2, notifyCount)
	assert.Contains(t, buf.String(), `→ {"sessionID":"child-session-1"}`)
}

func TestBuildWorkspaceFullStateExposesMemoryOnlyActiveSubagents(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "active-agents")
	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{Status: state.StatusWorking})
	require.NoError(t, errCoord)
	tracker := newActiveAgentTracker()
	tracker.applyEvent(activeTaskEvent("part-1", "call-1", "running", "go", "child-session-1", "openai/gpt-5.5", "Implement backend"))
	server.mu.Lock()
	server.sessions[wsDir] = &session{running: true, activeAgents: tracker}
	server.mu.Unlock()

	full := server.buildWorkspaceFullState(workspaceInfo{Directory: wsDir, DirName: "active-agents", Running: true, HasWorkspace: true}, nil)

	require.Len(t, full.ActiveAgents, 1)
	assert.Equal(t, "go", full.ActiveAgents[0].Agent)
	assert.Equal(t, "child-session-1", full.ActiveAgents[0].SessionID)
	data, errRead := os.ReadFile(statePath)
	require.NoError(t, errRead)
	assert.NotContains(t, string(data), "activeAgents")
	assert.NotContains(t, string(data), "currentAgent")
	var persisted state.Workflow
	require.NoError(t, json.Unmarshal(data, &persisted))
}

func TestBuildWorkspaceFullStateExposesEmptyActiveSubagentsForStoppedSessions(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "stopped-agents")
	_, errCoord := state.NewCoordinatorWith(filepath.Join(wsDir, ".sgai", "state.json"), state.Workflow{Status: state.StatusWorking})
	require.NoError(t, errCoord)
	tracker := newActiveAgentTracker()
	tracker.applyEvent(activeTaskEvent("part-1", "call-1", "running", "go", "child-session-1", "", "Implement backend"))
	server.mu.Lock()
	server.sessions[wsDir] = &session{running: false, activeAgents: tracker}
	server.mu.Unlock()

	full := server.buildWorkspaceFullState(workspaceInfo{Directory: wsDir, DirName: "stopped-agents", Running: false, HasWorkspace: true}, nil)

	assert.Empty(t, full.ActiveAgents)

	server.mu.Lock()
	delete(server.sessions, wsDir)
	server.mu.Unlock()

	full = server.buildWorkspaceFullState(workspaceInfo{Directory: wsDir, DirName: "stopped-agents", Running: false, HasWorkspace: true}, nil)

	assert.Empty(t, full.ActiveAgents)
}

func TestHandleAPIStateExposesEmptyActiveSubagentsArray(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "api-empty-active-agents")
	_, errCoord := state.NewCoordinatorWith(filepath.Join(wsDir, ".sgai", "state.json"), state.Workflow{Status: state.StatusComplete})
	require.NoError(t, errCoord)

	w := serveHTTP(server, "GET", "/api/v1/state", "")

	assert.Equal(t, 200, w.Code)
	var resp apiFactoryState
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Workspaces, 1)
	assert.NotNil(t, resp.Workspaces[0].ActiveAgents)
	assert.Empty(t, resp.Workspaces[0].ActiveAgents)
}

func TestStopSessionClearsActiveSubagents(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "stop-active-agents")
	tracker := newActiveAgentTracker()
	tracker.applyEvent(activeTaskEvent("part-1", "call-1", "running", "go", "child-session-1", "", "Implement backend"))
	server.mu.Lock()
	server.sessions[wsDir] = &session{running: true, activeAgents: tracker}
	server.mu.Unlock()

	server.stopSession(wsDir)

	assert.Empty(t, tracker.snapshot())
}

func activeTaskEvent(partID, callID, status, agent, sessionID, model, title string) streamEvent {
	metadata := map[string]any{}
	if sessionID != "" {
		metadata["sessionId"] = sessionID
	}
	if model != "" {
		metadata["model"] = map[string]any{"providerID": "openai", "modelID": "gpt-5.5"}
	}
	input := map[string]any{"description": title}
	if agent != "" {
		input["subagent_type"] = agent
	}
	return streamEvent{
		Type: "tool",
		Part: part{
			ID:     partID,
			CallID: callID,
			Tool:   "task",
			State: &toolState{
				Status:   status,
				Input:    input,
				Title:    title,
				Metadata: metadata,
			},
		},
	}
}

func decodeStreamEvent(t *testing.T, raw string) streamEvent {
	t.Helper()
	var event streamEvent
	require.NoError(t, json.Unmarshal([]byte(raw), &event))
	return event
}
