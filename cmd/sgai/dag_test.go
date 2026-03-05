package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComposeFlowTemplate(t *testing.T) {
	tests := []struct {
		name         string
		currentAgent string
		validate     func(*testing.T, string)
	}{
		{
			name:         "coordinatorAgent",
			currentAgent: "coordinator",
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, flowSectionPreamble)
				assert.Contains(t, result, flowSectionHumanCommDirect)
				assert.Contains(t, result, flowSectionMessaging)
				assert.Contains(t, result, flowSectionPeekMessageBus)
				assert.Contains(t, result, flowSectionWorkFocus)
				assert.Contains(t, result, flowSectionNavigation)
				assert.Contains(t, result, flowSectionPostSkillsCoordinator)
				assert.Contains(t, result, flowSectionGuidelines)
				assert.Contains(t, result, flowSectionTailCoordinator)
				assert.Contains(t, result, flowSectionCommonTail)
			},
		},
		{
			name:         "nonCoordinatorAgent",
			currentAgent: "backend-developer",
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, flowSectionPreamble)
				assert.Contains(t, result, flowSectionHumanCommNonCoordinator)
				assert.Contains(t, result, flowSectionMessaging)
				assert.NotContains(t, result, flowSectionPeekMessageBus)
				assert.Contains(t, result, flowSectionWorkFocus)
				assert.Contains(t, result, flowSectionNavigation)
				assert.Contains(t, result, flowSectionPostSkillsNonCoordinator)
				assert.Contains(t, result, flowSectionGuidelines)
				assert.Contains(t, result, flowSectionTailNonCoordinator)
				assert.Contains(t, result, flowSectionCommonTail)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := composeFlowTemplate(tt.currentAgent)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestParseFlow(t *testing.T) {
	tests := []struct {
		name        string
		flowSpec    string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *dag)
	}{
		{
			name:     "emptyFlow",
			flowSpec: "",
			wantErr:  false,
			validate: func(t *testing.T, d *dag) {
				assert.Contains(t, d.Nodes, "coordinator")
				assert.Contains(t, d.Nodes, "general-purpose")
			},
		},
		{
			name:     "simpleFlow",
			flowSpec: `"agent1" -> "agent2"`,
			wantErr:  false,
			validate: func(t *testing.T, d *dag) {
				assert.Contains(t, d.Nodes, "coordinator")
				assert.Contains(t, d.Nodes, "agent1")
				assert.Contains(t, d.Nodes, "agent2")
				assert.Contains(t, d.Nodes, "project-critic-council")
			},
		},
		{
			name:     "digraphFlow",
			flowSpec: `digraph G { "a" -> "b" }`,
			wantErr:  false,
			validate: func(t *testing.T, d *dag) {
				assert.Contains(t, d.Nodes, "coordinator")
				assert.Contains(t, d.Nodes, "a")
				assert.Contains(t, d.Nodes, "b")
			},
		},
		{
			name:        "invalidDot",
			flowSpec:    `digraph G { "a" -> }`,
			wantErr:     true,
			errContains: "failed to",
		},
	}

	t.Run("fileFlowSpec", func(t *testing.T) {
		dir := t.TempDir()
		flowContent := `digraph G { "x" -> "y" }`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "flow.dot"), []byte(flowContent), 0644))
		result, err := parseFlow("@flow.dot", dir)
		require.NoError(t, err)
		assert.Contains(t, result.Nodes, "x")
		assert.Contains(t, result.Nodes, "y")
	})

	t.Run("fileFlowSpecMissing", func(t *testing.T) {
		_, err := parseFlow("@missing.dot", t.TempDir())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read flow file")
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFlow(tt.flowSpec, t.TempDir())

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestDagEnsureNode(t *testing.T) {
	d := &dag{Nodes: make(map[string]*dagNode)}

	node1 := d.ensureNode("agent1")
	assert.NotNil(t, node1)
	assert.Equal(t, "agent1", node1.Name)

	node1Again := d.ensureNode("agent1")
	assert.Equal(t, node1, node1Again, "should return same node for same name")

	node2 := d.ensureNode("agent2")
	assert.NotNil(t, node2)
	assert.Equal(t, "agent2", node2.Name)
	assert.NotEqual(t, node1, node2)
}

func TestDagGetSuccessors(t *testing.T) {
	d := &dag{
		Nodes: map[string]*dagNode{
			"agent1": {Name: "agent1", Successors: []string{"agent2", "agent3"}},
			"agent2": {Name: "agent2", Successors: []string{}},
		},
	}

	successors1 := d.getSuccessors("agent1")
	assert.ElementsMatch(t, []string{"agent2", "agent3"}, successors1)

	successors2 := d.getSuccessors("agent2")
	assert.Empty(t, successors2)

	successors3 := d.getSuccessors("nonexistent")
	assert.Nil(t, successors3)
}

func TestDagGetPredecessors(t *testing.T) {
	d := &dag{
		Nodes: map[string]*dagNode{
			"agent1": {Name: "agent1", Predecessors: []string{}},
			"agent2": {Name: "agent2", Predecessors: []string{"agent1"}},
		},
	}

	pred1 := d.getPredecessors("agent1")
	assert.Empty(t, pred1)

	pred2 := d.getPredecessors("agent2")
	assert.ElementsMatch(t, []string{"agent1"}, pred2)

	pred3 := d.getPredecessors("nonexistent")
	assert.Nil(t, pred3)
}

func TestDagIsTerminal(t *testing.T) {
	d := &dag{
		Nodes: map[string]*dagNode{
			"agent1": {Name: "agent1", Successors: []string{"agent2"}},
			"agent2": {Name: "agent2", Successors: []string{}},
		},
	}

	assert.False(t, d.isTerminal("agent1"))
	assert.True(t, d.isTerminal("agent2"))
	assert.False(t, d.isTerminal("nonexistent"))
}

func TestDagAllAgents(t *testing.T) {
	d := &dag{
		Nodes: map[string]*dagNode{
			"coordinator": {Name: "coordinator"},
			"agent1":      {Name: "agent1"},
			"agent2":      {Name: "agent2"},
		},
	}

	agents := d.allAgents()
	assert.ElementsMatch(t, []string{"agent1", "agent2", "coordinator"}, agents)
	assert.True(t, isSorted(agents), "agents should be sorted")
}

func TestDetermineNextAgent(t *testing.T) {
	d := &dag{
		Nodes: map[string]*dagNode{
			"agent1": {Name: "agent1", Successors: []string{"agent2"}},
			"agent2": {Name: "agent2", Successors: []string{}},
		},
	}

	next := determineNextAgent(d, "agent1")
	assert.Equal(t, "coordinator", next)

	nextTerminal := determineNextAgent(d, "agent2")
	assert.Empty(t, nextTerminal)
}

func TestDagToDOT(t *testing.T) {
	d := &dag{
		Nodes: map[string]*dagNode{
			"agent1": {Name: "agent1", Successors: []string{"agent2"}},
			"agent2": {Name: "agent2", Successors: []string{}},
		},
	}

	dot := d.toDOT()
	assert.Contains(t, dot, "strict digraph G")
	assert.Contains(t, dot, `"agent1" -> "agent2"`)
	assert.Contains(t, dot, "rankdir=LR")
}

func TestDagInjectCoordinatorEdges(t *testing.T) {
	tests := []struct {
		name     string
		setupDag func() *dag
		validate func(*testing.T, *dag)
	}{
		{
			name: "alreadyHasCoordinator",
			setupDag: func() *dag {
				return &dag{
					Nodes: map[string]*dagNode{
						"coordinator": {Name: "coordinator", Successors: []string{"agent1"}},
						"agent1":      {Name: "agent1", Predecessors: []string{"coordinator"}},
					},
					EntryNodes: []string{"coordinator"},
				}
			},
			validate: func(t *testing.T, d *dag) {
				assert.Equal(t, []string{"coordinator"}, d.EntryNodes)
			},
		},
		{
			name: "needsInjection",
			setupDag: func() *dag {
				return &dag{
					Nodes: map[string]*dagNode{
						"agent1": {Name: "agent1", Predecessors: []string{}},
						"agent2": {Name: "agent2", Predecessors: []string{"agent1"}},
					},
					EntryNodes: []string{"agent1"},
				}
			},
			validate: func(t *testing.T, d *dag) {
				assert.Equal(t, []string{"coordinator"}, d.EntryNodes)
				assert.Contains(t, d.Nodes, "coordinator")
				assert.Contains(t, d.Nodes["coordinator"].Successors, "agent1")
				assert.Contains(t, d.Nodes["agent1"].Predecessors, "coordinator")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.setupDag()
			d.injectCoordinatorEdges()
			tt.validate(t, d)
		})
	}
}

func TestDagDetectCycles(t *testing.T) {
	tests := []struct {
		name        string
		setupDag    func() *dag
		wantErr     bool
		errContains string
	}{
		{
			name: "noCycle",
			setupDag: func() *dag {
				return &dag{
					Nodes: map[string]*dagNode{
						"agent1": {Name: "agent1", Successors: []string{"agent2"}},
						"agent2": {Name: "agent2", Successors: []string{}},
					},
				}
			},
			wantErr: false,
		},
		{
			name: "hasCycle",
			setupDag: func() *dag {
				return &dag{
					Nodes: map[string]*dagNode{
						"agent1": {Name: "agent1", Successors: []string{"agent2"}},
						"agent2": {Name: "agent2", Successors: []string{"agent1"}},
					},
				}
			},
			wantErr:     true,
			errContains: "cycle detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.setupDag()
			err := d.detectCycles()

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestBuildMultiModelSection(t *testing.T) {
	tests := []struct {
		name         string
		currentModel string
		models       map[string]any
		currentAgent string
		expected     string
	}{
		{
			name:         "emptyModel",
			currentModel: "",
			models:       map[string]any{"agent1": "model1"},
			currentAgent: "agent1",
			expected:     "",
		},
		{
			name:         "singleModel",
			currentModel: "model1",
			models:       map[string]any{"agent1": "model1"},
			currentAgent: "agent1",
			expected:     "",
		},
		{
			name:         "multiModel",
			currentModel: "agent1/model1",
			models: map[string]any{
				"agent1": []any{"model1", "model2"},
			},
			currentAgent: "agent1",
			expected:     "Multi-Model Agent Context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildMultiModelSection(tt.currentModel, tt.models, tt.currentAgent)
			if tt.expected == "" {
				assert.Empty(t, result)
			} else {
				assert.Contains(t, result, "Multi-Model Agent Context")
				assert.Contains(t, result, tt.currentModel)
			}
		})
	}
}

func TestBuildMultiModelSectionYouMarker(t *testing.T) {
	result := buildMultiModelSection("agent1:model1", map[string]any{
		"agent1": []any{"model1", "model2"},
	}, "agent1")
	assert.Contains(t, result, "<-- YOU")
	assert.Contains(t, result, "agent1:model1")
	assert.Contains(t, result, "agent1:model2")
}

func TestBuildFlowMessageWithAgentDescription(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, ".sgai", "agent")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "coordinator.md"), []byte("---\ndescription: Orchestrates the workflow\n---\n# Coordinator"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "worker.md"), []byte("---\ndescription: Does the work\nsnippets:\n  - go\n  - python\n---\n# Worker"), 0644))

	d := &dag{
		Nodes: map[string]*dagNode{
			"coordinator": {Name: "coordinator", Successors: []string{"worker"}, Predecessors: []string{}},
			"worker":      {Name: "worker", Successors: []string{}, Predecessors: []string{"coordinator"}},
		},
	}
	msg := buildFlowMessage(d, "worker", map[string]int{"coordinator": 1, "worker": 1}, dir, "brainstorming", map[string]string{})
	assert.Contains(t, msg, "Orchestrates the workflow")
	assert.Contains(t, msg, "Does the work")
	assert.Contains(t, msg, "sgai_find_snippets()")
	assert.Contains(t, msg, "go")
	assert.Contains(t, msg, "python")
}

func TestBuildFlowMessageAgentNoDescription(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, ".sgai", "agent")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "coordinator.md"), []byte("---\ndescription: Orchestrates the workflow\n---\n# Coordinator"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "worker.md"), []byte("---\ntitle: Worker Agent\n---\n# Worker with no description field"), 0644))

	d := &dag{
		Nodes: map[string]*dagNode{
			"coordinator": {Name: "coordinator", Successors: []string{"worker"}, Predecessors: []string{}},
			"worker":      {Name: "worker", Successors: []string{}, Predecessors: []string{"coordinator"}},
		},
	}
	msg := buildFlowMessage(d, "worker", map[string]int{"coordinator": 1, "worker": 0}, dir, "brainstorming", map[string]string{})
	assert.Contains(t, msg, "Orchestrates the workflow")
	assert.NotContains(t, msg, "description:")
}

func TestInjectCoordinatorEdgesWithCoordinatorAsEntry(t *testing.T) {
	d := &dag{
		Nodes: map[string]*dagNode{
			"coordinator": {Name: "coordinator", Successors: []string{}, Predecessors: []string{}},
			"agent1":      {Name: "agent1", Successors: []string{}, Predecessors: []string{}},
		},
		EntryNodes: []string{"agent1", "coordinator"},
	}
	d.injectCoordinatorEdges()
	assert.Equal(t, []string{"coordinator"}, d.EntryNodes)
	assert.Contains(t, d.Nodes["coordinator"].Successors, "agent1")
	assert.Contains(t, d.Nodes["agent1"].Predecessors, "coordinator")
}

func TestParseFlowCycleDetection(t *testing.T) {
	_, err := parseFlow("coordinator -> agent1\nagent1 -> coordinator", t.TempDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestParseFlowNoEntryNodes(t *testing.T) {
	_, err := parseFlow("@invalid-nonexistent-file.dot", t.TempDir())
	assert.Error(t, err)
}

func isSorted(s []string) bool {
	for i := 1; i < len(s); i++ {
		if s[i-1] > s[i] {
			return false
		}
	}
	return true
}

func TestBuildFlowMessage(t *testing.T) {
	tests := []struct {
		name            string
		dag             *dag
		currentAgent    string
		visitCounts     map[string]int
		interactionMode string
		alias           map[string]string
		validate        func(*testing.T, string)
	}{
		{
			name: "buildMessageForEntryNode",
			dag: &dag{
				Nodes: map[string]*dagNode{
					"coordinator": {Name: "coordinator", Successors: []string{"agent1"}, Predecessors: []string{}},
					"agent1":      {Name: "agent1", Successors: []string{}, Predecessors: []string{"coordinator"}},
				},
			},
			currentAgent:    "coordinator",
			visitCounts:     map[string]int{"coordinator": 1, "agent1": 0},
			interactionMode: "brainstorming",
			alias:           map[string]string{},
			validate: func(t *testing.T, msg string) {
				assert.Contains(t, msg, "coordinator")
				assert.Contains(t, msg, "(none - entry node)")
				assert.Contains(t, msg, "agent1")
				assert.Contains(t, msg, "coordinator: 1 visits")
			},
		},
		{
			name: "buildMessageForTerminalNode",
			dag: &dag{
				Nodes: map[string]*dagNode{
					"coordinator": {Name: "coordinator", Successors: []string{"agent1"}, Predecessors: []string{}},
					"agent1":      {Name: "agent1", Successors: []string{}, Predecessors: []string{"coordinator"}},
				},
			},
			currentAgent:    "agent1",
			visitCounts:     map[string]int{"coordinator": 1, "agent1": 1},
			interactionMode: "brainstorming",
			alias:           map[string]string{},
			validate: func(t *testing.T, msg string) {
				assert.Contains(t, msg, "agent1")
				assert.Contains(t, msg, "coordinator")
				assert.Contains(t, msg, "(none - terminal node)")
				assert.Contains(t, msg, "agent1: 1 visits")
			},
		},
		{
			name: "buildMessageForMiddleNode",
			dag: &dag{
				Nodes: map[string]*dagNode{
					"coordinator": {Name: "coordinator", Successors: []string{"agent1"}, Predecessors: []string{}},
					"agent1":      {Name: "agent1", Successors: []string{"agent2"}, Predecessors: []string{"coordinator"}},
					"agent2":      {Name: "agent2", Successors: []string{}, Predecessors: []string{"agent1"}},
				},
			},
			currentAgent:    "agent1",
			visitCounts:     map[string]int{"coordinator": 1, "agent1": 1, "agent2": 0},
			interactionMode: "brainstorming",
			alias:           map[string]string{},
			validate: func(t *testing.T, msg string) {
				assert.Contains(t, msg, "agent1")
				assert.Contains(t, msg, "coordinator")
				assert.Contains(t, msg, "agent2")
				assert.Contains(t, msg, "agent1: 1 visits")
			},
		},
		{
			name: "buildMessageWithAlias",
			dag: &dag{
				Nodes: map[string]*dagNode{
					"coordinator":  {Name: "coordinator", Successors: []string{"agent1-alias"}, Predecessors: []string{}},
					"agent1-alias": {Name: "agent1-alias", Successors: []string{}, Predecessors: []string{"coordinator"}},
				},
			},
			currentAgent:    "agent1-alias",
			visitCounts:     map[string]int{"coordinator": 1, "agent1-alias": 1},
			interactionMode: "brainstorming",
			alias:           map[string]string{"agent1-alias": "agent1"},
			validate: func(t *testing.T, msg string) {
				assert.Contains(t, msg, "agent1-alias")
				assert.Contains(t, msg, "coordinator")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			msg := buildFlowMessage(tt.dag, tt.currentAgent, tt.visitCounts, dir, tt.interactionMode, tt.alias)
			if tt.validate != nil {
				tt.validate(t, msg)
			}
		})
	}
}
