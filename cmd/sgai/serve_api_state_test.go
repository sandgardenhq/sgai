package main

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func setupStateTestWorkspace(t *testing.T) (string, *Server) {
	t.Helper()
	installFakeJJWithWorkspaceList(t, 1)
	rootDir := t.TempDir()
	workspace := filepath.Join(rootDir, "root-workspace")
	if err := os.MkdirAll(filepath.Join(workspace, ".jj", "repo"), 0755); err != nil {
		t.Fatal(err)
	}
	createsgaiDir(t, workspace)
	srv := NewServer(rootDir)
	return workspace, srv
}

func TestHandleAPIState(t *testing.T) {
	t.Run("returnsJSONWithWorkspaces", func(t *testing.T) {
		_, srv := setupStateTestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusOK, resp.Body.String())
		}

		var result apiFactoryState
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(result.Workspaces) == 0 {
			t.Error("workspaces should not be empty")
		}
	})

	t.Run("workspaceHasExpectedFields", func(t *testing.T) {
		_, srv := setupStateTestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiFactoryState
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(result.Workspaces) == 0 {
			t.Fatal("workspaces is empty")
		}

		ws := result.Workspaces[0]
		if ws.Name == "" {
			t.Error("workspace Name should not be empty")
		}
		if ws.Dir == "" {
			t.Error("workspace Dir should not be empty")
		}
	})

	t.Run("contentTypeIsJSON", func(t *testing.T) {
		_, srv := setupStateTestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		ct := resp.Header().Get("Content-Type")
		if !strings.Contains(ct, "application/json") {
			t.Errorf("Content-Type = %q; want application/json", ct)
		}
	})

	t.Run("emptyRootDir", func(t *testing.T) {
		rootDir := t.TempDir()
		srv := NewServer(rootDir)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiFactoryState
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if result.Workspaces == nil {
			t.Error("workspaces should not be nil")
		}
	})
}

func TestHandleSignalStream(t *testing.T) {
	t.Run("returnsSSEHeaders", func(t *testing.T) {
		_, srv := setupStateTestWorkspace(t)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mux := http.NewServeMux()
			srv.registerAPIRoutes(mux)
			mux.ServeHTTP(w, r)
		}))
		t.Cleanup(server.Close)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/v1/signal", nil)
		if errReq != nil {
			t.Fatal(errReq)
		}

		type result struct {
			resp *http.Response
			err  error
		}
		ch := make(chan result, 1)
		go func() {
			resp, err := server.Client().Do(req)
			ch <- result{resp, err}
		}()

		select {
		case r := <-ch:
			if r.err != nil {
				t.Fatal(r.err)
			}
			t.Cleanup(func() {
				if errClose := r.resp.Body.Close(); errClose != nil {
					t.Logf("failed to close body: %v", errClose)
				}
			})
			if r.resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d; want %d", r.resp.StatusCode, http.StatusOK)
			}
			ct := r.resp.Header.Get("Content-Type")
			if ct != "text/event-stream" {
				t.Errorf("Content-Type = %q; want text/event-stream", ct)
			}
			cc := r.resp.Header.Get("Cache-Control")
			if cc != "no-cache" {
				t.Errorf("Cache-Control = %q; want no-cache", cc)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for response headers")
		}
	})

	t.Run("receivesReloadEventOnNotify", func(t *testing.T) {
		_, srv := setupStateTestWorkspace(t)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mux := http.NewServeMux()
			srv.registerAPIRoutes(mux)
			mux.ServeHTTP(w, r)
		}))
		t.Cleanup(server.Close)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/v1/signal", nil)
		if errReq != nil {
			t.Fatal(errReq)
		}

		type result struct {
			resp *http.Response
			err  error
		}
		connCh := make(chan result, 1)
		go func() {
			resp, err := server.Client().Do(req)
			connCh <- result{resp, err}
		}()

		var resp *http.Response
		select {
		case r := <-connCh:
			if r.err != nil {
				t.Fatal(r.err)
			}
			resp = r.resp
			t.Cleanup(func() {
				if errClose := resp.Body.Close(); errClose != nil {
					t.Logf("failed to close body: %v", errClose)
				}
			})
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for SSE connection")
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.StatusCode, http.StatusOK)
		}

		eventCh := make(chan string, 1)
		go func() {
			scanner := bufio.NewScanner(resp.Body)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "event:") {
					eventCh <- line
					return
				}
			}
		}()

		time.Sleep(20 * time.Millisecond)
		srv.notifyStateChange()

		select {
		case line := <-eventCh:
			if !strings.Contains(line, "reload") {
				t.Errorf("event line = %q; want containing 'reload'", line)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for reload event")
		}
	})
}

func TestSignalBroker(t *testing.T) {
	t.Run("notifiesSubscribers", func(t *testing.T) {
		b := newSignalBroker()
		sub := b.subscribe()
		defer b.unsubscribe(sub)

		b.notify()

		select {
		case <-sub.ch:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("expected notification on channel")
		}
	})

	t.Run("doesNotBlockOnFullChannel", func(_ *testing.T) {
		b := newSignalBroker()
		sub := b.subscribe()
		defer b.unsubscribe(sub)

		b.notify()
		b.notify()
		b.notify()
	})

	t.Run("unsubscribeClosesDoneChannel", func(t *testing.T) {
		b := newSignalBroker()
		sub := b.subscribe()
		b.unsubscribe(sub)

		select {
		case <-sub.done:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("expected done channel to be closed after unsubscribe")
		}
	})

	t.Run("multipleSubscribersAllNotified", func(t *testing.T) {
		b := newSignalBroker()
		sub1 := b.subscribe()
		sub2 := b.subscribe()
		defer b.unsubscribe(sub1)
		defer b.unsubscribe(sub2)

		b.notify()

		select {
		case <-sub1.ch:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("sub1 expected notification")
		}
		select {
		case <-sub2.ch:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("sub2 expected notification")
		}
	})
}

func TestNotifyStateChangeTriggersSignal(t *testing.T) {
	t.Run("serverNotifyStateChangeTriggersSubscribers", func(t *testing.T) {
		_, srv := setupStateTestWorkspace(t)

		sub := srv.signals.subscribe()
		defer srv.signals.unsubscribe(sub)

		srv.notifyStateChange()

		select {
		case <-sub.ch:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("expected signal after notifyStateChange")
		}
	})
}

func TestBuildFullFactoryState(t *testing.T) {
	t.Run("returnsEmptyWorkspacesForEmptyRoot", func(t *testing.T) {
		rootDir := t.TempDir()
		srv := NewServer(rootDir)

		result := srv.buildFullFactoryState()

		if result.Workspaces == nil {
			t.Error("workspaces should not be nil")
		}
	})

	t.Run("includesWorkspaceFromRootDir", func(t *testing.T) {
		workspace, srv := setupStateTestWorkspace(t)

		result := srv.buildFullFactoryState()

		found := false
		for _, ws := range result.Workspaces {
			if ws.Dir == workspace {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("workspace %q not found in state", workspace)
		}
	})
}

func setupStateWorkspaceWithData(t *testing.T) (string, *Server) {
	t.Helper()
	workspace, srv := setupStateTestWorkspace(t)

	wfState := state.Workflow{
		Status:       state.StatusAgentDone,
		CurrentAgent: "test-agent",
		Task:         "doing something",
		HumanMessage: "",
		Todos: []state.TodoItem{
			{ID: "t1", Content: "agent task 1", Status: "pending", Priority: "high"},
		},
		ProjectTodos: []state.TodoItem{
			{ID: "p1", Content: "project task 1", Status: "in_progress", Priority: "medium"},
		},
		Messages: []state.Message{
			{ID: 1, FromAgent: "agent-a", ToAgent: "agent-b", Body: "hello", Read: false, CreatedAt: "2026-01-01T00:00:00Z"},
		},
		Progress: []state.ProgressEntry{
			{Timestamp: "2026-01-01T00:00:01Z", Agent: "test-agent", Description: "started work"},
		},
	}
	if _, errSave := state.NewCoordinatorWith(statePath(workspace), wfState); errSave != nil {
		t.Fatal(errSave)
	}

	return workspace, srv
}

func TestHandleAPIStateWorkspaceFields(t *testing.T) {
	t.Run("includesTodos", func(t *testing.T) {
		workspace, srv := setupStateWorkspaceWithData(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiFactoryState
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}

		ws := findWorkspace(t, result, workspace)
		if len(ws.AgentTodos) == 0 {
			t.Error("agentTodos should not be empty")
		}
		if ws.AgentTodos[0].Content != "agent task 1" {
			t.Errorf("agentTodos[0].Content = %q; want %q", ws.AgentTodos[0].Content, "agent task 1")
		}
		if len(ws.ProjectTodos) == 0 {
			t.Error("projectTodos should not be empty")
		}
		if ws.ProjectTodos[0].Content != "project task 1" {
			t.Errorf("projectTodos[0].Content = %q; want %q", ws.ProjectTodos[0].Content, "project task 1")
		}
	})

	t.Run("includesMessages", func(t *testing.T) {
		workspace, srv := setupStateWorkspaceWithData(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiFactoryState
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}

		ws := findWorkspace(t, result, workspace)
		if len(ws.Messages) == 0 {
			t.Error("messages should not be empty")
		}
		if ws.Messages[0].FromAgent != "agent-a" {
			t.Errorf("messages[0].FromAgent = %q; want %q", ws.Messages[0].FromAgent, "agent-a")
		}
	})

	t.Run("includesEvents", func(t *testing.T) {
		workspace, srv := setupStateWorkspaceWithData(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiFactoryState
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}

		ws := findWorkspace(t, result, workspace)
		if len(ws.Events) == 0 {
			t.Error("events should not be empty")
		}
	})

	t.Run("includesCurrentAgent", func(t *testing.T) {
		workspace, srv := setupStateWorkspaceWithData(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiFactoryState
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}

		ws := findWorkspace(t, result, workspace)
		if ws.CurrentAgent != "test-agent" {
			t.Errorf("currentAgent = %q; want %q", ws.CurrentAgent, "test-agent")
		}
	})

	t.Run("includesTaskField", func(t *testing.T) {
		workspace, srv := setupStateWorkspaceWithData(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiFactoryState
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}

		ws := findWorkspace(t, result, workspace)
		if ws.Task != "doing something" {
			t.Errorf("task = %q; want %q", ws.Task, "doing something")
		}
	})

	t.Run("includesCostField", func(t *testing.T) {
		workspace, srv := setupStateTestWorkspace(t)

		wfState := state.Workflow{
			Cost: state.SessionCost{TotalCost: 1.23},
		}
		if _, errSave := state.NewCoordinatorWith(statePath(workspace), wfState); errSave != nil {
			t.Fatal(errSave)
		}

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiFactoryState
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}

		ws := findWorkspace(t, result, workspace)
		if ws.Cost.TotalCost != 1.23 {
			t.Errorf("cost.totalCost = %v; want %v", ws.Cost.TotalCost, 1.23)
		}
	})

	t.Run("includesPendingQuestion", func(t *testing.T) {
		workspace, srv := setupStateTestWorkspace(t)

		wfState := state.Workflow{
			Status:       state.StatusWaitingForHuman,
			CurrentAgent: "asking-agent",
			HumanMessage: "what should I do?",
		}
		if _, errSave := state.NewCoordinatorWith(statePath(workspace), wfState); errSave != nil {
			t.Fatal(errSave)
		}

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiFactoryState
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}

		ws := findWorkspace(t, result, workspace)
		if !ws.NeedsInput {
			t.Error("needsInput should be true when workspace is waiting for human input")
		}
		if ws.PendingQuestion == nil {
			t.Fatal("pendingQuestion should not be nil when waiting for human input")
		}
		if ws.PendingQuestion.Message != "what should I do?" {
			t.Errorf("pendingQuestion.message = %q; want %q", ws.PendingQuestion.Message, "what should I do?")
		}
		if ws.PendingQuestion.AgentName != "asking-agent" {
			t.Errorf("pendingQuestion.agentName = %q; want %q", ws.PendingQuestion.AgentName, "asking-agent")
		}
	})

	t.Run("pendingQuestionNilWhenNoInput", func(t *testing.T) {
		workspace, srv := setupStateTestWorkspace(t)

		wfState := state.Workflow{
			Status: state.StatusWorking,
		}
		if _, errSave := state.NewCoordinatorWith(statePath(workspace), wfState); errSave != nil {
			t.Fatal(errSave)
		}

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiFactoryState
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}

		ws := findWorkspace(t, result, workspace)
		if ws.PendingQuestion != nil {
			t.Error("pendingQuestion should be nil when not waiting for human input")
		}
	})

	t.Run("includesLogLines", func(t *testing.T) {
		workspace, srv := setupStateTestWorkspace(t)

		outputLog := newCircularLogBuffer()
		outputLog.add(logLine{prefix: "prefix", text: "log line 1"})
		outputLog.add(logLine{prefix: "prefix", text: "log line 2"})
		srv.mu.Lock()
		srv.sessions[workspace] = &session{outputLog: outputLog}
		srv.mu.Unlock()

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiFactoryState
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}

		ws := findWorkspace(t, result, workspace)
		if len(ws.Log) == 0 {
			t.Error("log should not be empty when session has output")
		}
	})

	t.Run("includesBadgeStatus", func(t *testing.T) {
		workspace, srv := setupStateTestWorkspace(t)

		wfState := state.Workflow{
			Status: state.StatusWorking,
		}
		if _, errSave := state.NewCoordinatorWith(statePath(workspace), wfState); errSave != nil {
			t.Fatal(errSave)
		}

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiFactoryState
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}

		ws := findWorkspace(t, result, workspace)
		if ws.BadgeText == "" {
			t.Error("badgeText should not be empty")
		}
	})

	t.Run("includesGoalContent", func(t *testing.T) {
		workspace, srv := setupStateTestWorkspace(t)

		goalPath := filepath.Join(workspace, "GOAL.md")
		if errWrite := os.WriteFile(goalPath, []byte("# My Goal\n\n- [ ] Task 1\n"), 0644); errWrite != nil {
			t.Fatal(errWrite)
		}

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiFactoryState
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}

		ws := findWorkspace(t, result, workspace)
		if ws.RawGoalContent == "" {
			t.Error("rawGoalContent should not be empty when GOAL.md exists")
		}
		if !strings.Contains(ws.RawGoalContent, "My Goal") {
			t.Errorf("rawGoalContent should contain 'My Goal', got: %q", ws.RawGoalContent)
		}
		if ws.HasEditedGoal != true {
			t.Error("hasEditedGoal should be true when GOAL.md has body content")
		}
	})

	t.Run("includesMultiChoiceQuestion", func(t *testing.T) {
		workspace, srv := setupStateTestWorkspace(t)

		wfState := state.Workflow{
			Status:       state.StatusWaitingForHuman,
			CurrentAgent: "coordinator",
			HumanMessage: "Choose an option",
			MultiChoiceQuestion: &state.MultiChoiceQuestion{
				Questions: []state.QuestionItem{
					{Question: "Pick one", Choices: []string{"A", "B", "C"}, MultiSelect: false},
				},
			},
		}
		if _, errSave := state.NewCoordinatorWith(statePath(workspace), wfState); errSave != nil {
			t.Fatal(errSave)
		}

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiFactoryState
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}

		ws := findWorkspace(t, result, workspace)
		if ws.PendingQuestion == nil {
			t.Fatal("pendingQuestion should not be nil")
		}
		if ws.PendingQuestion.Type != "multi-choice" {
			t.Errorf("pendingQuestion.type = %q; want %q", ws.PendingQuestion.Type, "multi-choice")
		}
		if len(ws.PendingQuestion.Questions) == 0 {
			t.Error("pendingQuestion.questions should not be empty")
		}
		if ws.PendingQuestion.Questions[0].Question != "Pick one" {
			t.Errorf("questions[0].question = %q; want %q", ws.PendingQuestion.Questions[0].Question, "Pick one")
		}
	})
}

func TestHandleAPIStateMultipleWorkspaces(t *testing.T) {
	t.Run("returnsAllWorkspaces", func(t *testing.T) {
		installFakeJJWithWorkspaceList(t, 2)
		rootDir := t.TempDir()

		ws1 := filepath.Join(rootDir, "workspace-one")
		if errMkdir := os.MkdirAll(filepath.Join(ws1, ".jj", "repo"), 0755); errMkdir != nil {
			t.Fatal(errMkdir)
		}
		createsgaiDir(t, ws1)

		ws2 := filepath.Join(rootDir, "workspace-two")
		if errMkdir := os.MkdirAll(filepath.Join(ws2, ".jj", "repo"), 0755); errMkdir != nil {
			t.Fatal(errMkdir)
		}
		createsgaiDir(t, ws2)

		srv := NewServer(rootDir)
		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiFactoryState
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}

		if len(result.Workspaces) < 2 {
			t.Errorf("expected at least 2 workspaces; got %d", len(result.Workspaces))
		}
	})
}

func TestHandleAPIStateConcurrentAccess(t *testing.T) {
	t.Run("singleflightDeduplicatesConcurrentRequests", func(t *testing.T) {
		_, srv := setupStateTestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		const goroutines = 10
		results := make([]int, goroutines)
		var wg sync.WaitGroup
		for i := range goroutines {
			wg.Go(func() {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
				resp := httptest.NewRecorder()
				mux.ServeHTTP(resp, req)
				results[i] = resp.Code
			})
		}
		wg.Wait()

		for i, code := range results {
			if code != http.StatusOK {
				t.Errorf("goroutine %d: status = %d; want %d", i, code, http.StatusOK)
			}
		}
	})

	t.Run("concurrentRequestsAllReturnValidJSON", func(t *testing.T) {
		_, srv := setupStateTestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		const goroutines = 5
		errs := make([]error, goroutines)
		var wg sync.WaitGroup
		for i := range goroutines {
			wg.Go(func() {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
				resp := httptest.NewRecorder()
				mux.ServeHTTP(resp, req)
				var result apiFactoryState
				errs[i] = json.NewDecoder(resp.Body).Decode(&result)
			})
		}
		wg.Wait()

		for i, err := range errs {
			if err != nil {
				t.Errorf("goroutine %d: failed to decode JSON: %v", i, err)
			}
		}
	})
}

func TestHandleAPIStateRemovedEndpoints(t *testing.T) {
	removedEndpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/workspaces"},
		{http.MethodGet, "/api/v1/workspaces/root-workspace"},
		{http.MethodGet, "/api/v1/workspaces/root-workspace/session"},
		{http.MethodGet, "/api/v1/workspaces/root-workspace/messages"},
		{http.MethodGet, "/api/v1/workspaces/root-workspace/todos"},
		{http.MethodGet, "/api/v1/workspaces/root-workspace/log"},
		{http.MethodGet, "/api/v1/workspaces/root-workspace/changes"},
		{http.MethodGet, "/api/v1/workspaces/root-workspace/events"},
		{http.MethodGet, "/api/v1/workspaces/root-workspace/forks"},
		{http.MethodGet, "/api/v1/workspaces/root-workspace/pending-question"},
		{http.MethodGet, "/api/v1/workspaces/root-workspace/commits"},
	}

	for _, ep := range removedEndpoints {
		t.Run(ep.method+"_"+ep.path, func(t *testing.T) {
			_, srv := setupStateTestWorkspace(t)

			mux := http.NewServeMux()
			srv.registerAPIRoutes(mux)

			req := httptest.NewRequest(ep.method, ep.path, nil)
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)

			if resp.Code != http.StatusMethodNotAllowed && resp.Code != http.StatusNotFound {
				t.Errorf("%s %s: status = %d; want 404 or 405 (endpoint should be removed)", ep.method, ep.path, resp.Code)
			}
		})
	}
}

func TestHandleAPIStateKeptEndpoints(t *testing.T) {
	keptEndpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/state"},
		{http.MethodGet, "/api/v1/agents"},
		{http.MethodGet, "/api/v1/skills"},
		{http.MethodGet, "/api/v1/snippets"},
		{http.MethodGet, "/api/v1/models"},
		{http.MethodGet, "/api/v1/compose"},
	}

	for _, ep := range keptEndpoints {
		t.Run(ep.method+"_"+ep.path, func(t *testing.T) {
			_, srv := setupStateTestWorkspace(t)

			mux := http.NewServeMux()
			srv.registerAPIRoutes(mux)

			req := httptest.NewRequest(ep.method, ep.path, nil)
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)

			if resp.Code == http.StatusNotFound || resp.Code == http.StatusMethodNotAllowed {
				t.Errorf("%s %s: status = %d; endpoint should still exist", ep.method, ep.path, resp.Code)
			}
		})
	}
}

func TestHandleAPIStateWorkspaceActions(t *testing.T) {
	t.Run("includesActions", func(t *testing.T) {
		workspace, srv := setupStateTestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiFactoryState
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}

		ws := findWorkspace(t, result, workspace)
		if ws.Actions == nil {
			t.Error("actions should not be nil (default actions should be present)")
		}
	})
}

func TestBuildWorkspaceFullStateFields(t *testing.T) {
	t.Run("setsDefaultCurrentAgent", func(t *testing.T) {
		rootDir := t.TempDir()
		workspace := filepath.Join(rootDir, "test-ws")
		if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
			t.Fatal(errMkdir)
		}
		createsgaiDir(t, workspace)

		srv := NewServer(rootDir)
		ws := workspaceInfo{Directory: workspace, DirName: "test-ws"}
		result := srv.buildWorkspaceFullState(ws, nil)

		if result.CurrentAgent != "Unknown" {
			t.Errorf("currentAgent = %q; want %q when no agent set", result.CurrentAgent, "Unknown")
		}
	})

	t.Run("setsDefaultStatus", func(t *testing.T) {
		rootDir := t.TempDir()
		workspace := filepath.Join(rootDir, "test-ws")
		if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
			t.Fatal(errMkdir)
		}
		createsgaiDir(t, workspace)

		srv := NewServer(rootDir)
		ws := workspaceInfo{Directory: workspace, DirName: "test-ws"}
		result := srv.buildWorkspaceFullState(ws, nil)

		if result.Status != "-" {
			t.Errorf("status = %q; want %q when no status set", result.Status, "-")
		}
	})

	t.Run("reflectsRunningWorkspace", func(t *testing.T) {
		rootDir := t.TempDir()
		workspace := filepath.Join(rootDir, "test-ws")
		if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
			t.Fatal(errMkdir)
		}
		createsgaiDir(t, workspace)

		srv := NewServer(rootDir)
		ws := workspaceInfo{Directory: workspace, DirName: "test-ws", Running: true}
		result := srv.buildWorkspaceFullState(ws, nil)

		if !result.Running {
			t.Error("running should be true for a running workspace")
		}
	})

	t.Run("includesAgentSequence", func(t *testing.T) {
		rootDir := t.TempDir()
		workspace := filepath.Join(rootDir, "test-ws")
		if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
			t.Fatal(errMkdir)
		}
		createsgaiDir(t, workspace)

		wfState := state.Workflow{
			CurrentAgent: "coord",
			AgentSequence: []state.AgentSequenceEntry{
				{Agent: "coord", StartTime: "2026-01-01T00:00:00Z", IsCurrent: true},
			},
		}
		if _, errSave := state.NewCoordinatorWith(statePath(workspace), wfState); errSave != nil {
			t.Fatal(errSave)
		}

		srv := NewServer(rootDir)
		ws := workspaceInfo{Directory: workspace, DirName: "test-ws"}
		result := srv.buildWorkspaceFullState(ws, nil)

		if len(result.AgentSequence) == 0 {
			t.Error("agentSequence should not be empty")
		}
		if result.AgentSequence[0].Agent != "coord" {
			t.Errorf("agentSequence[0].agent = %q; want %q", result.AgentSequence[0].Agent, "coord")
		}
	})
}

func TestHandleAPIStateSignalBrokerEdgeCases(t *testing.T) {
	t.Run("unsubscribeWhileNotifying", func(t *testing.T) {
		b := newSignalBroker()
		sub1 := b.subscribe()
		sub2 := b.subscribe()

		b.unsubscribe(sub1)
		b.notify()

		select {
		case <-sub2.ch:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("sub2 should still receive notification after sub1 unsubscribed")
		}
		b.unsubscribe(sub2)
	})

	t.Run("notifyWithNoSubscribers", func(_ *testing.T) {
		b := newSignalBroker()
		b.notify()
	})

	t.Run("subscribeAfterNotify", func(t *testing.T) {
		b := newSignalBroker()
		b.notify()

		sub := b.subscribe()
		defer b.unsubscribe(sub)

		select {
		case <-sub.ch:
			t.Fatal("subscriber added after notify should not receive old notification")
		case <-time.After(50 * time.Millisecond):
		}
	})
}

func findWorkspace(t *testing.T, state apiFactoryState, dir string) apiWorkspaceFullState {
	t.Helper()
	for _, ws := range state.Workspaces {
		if ws.Dir == dir {
			return ws
		}
	}
	t.Fatalf("workspace %q not found in state (workspaces: %v)", dir, workspaceDirs(state))
	return apiWorkspaceFullState{}
}

func workspaceDirs(state apiFactoryState) []string {
	dirs := make([]string, len(state.Workspaces))
	for i, ws := range state.Workspaces {
		dirs[i] = ws.Dir
	}
	return dirs
}
