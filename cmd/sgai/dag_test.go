package main

import (
	"slices"
	"testing"
)

func TestParseFlowInline(t *testing.T) {
	dotContent := `digraph workflow {
		planner -> coder
		planner -> researcher
		coder -> reviewer
		researcher -> reviewer
		reviewer -> finalizer
	}`

	dag, err := parseFlow(dotContent, "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	if len(dag.EntryNodes) != 1 || dag.EntryNodes[0] != "coordinator" {
		t.Errorf("expected entry node 'coordinator', got %v", dag.EntryNodes)
	}

	if len(dag.Nodes) != 6 {
		t.Errorf("expected 6 nodes (5 original + coordinator), got %d", len(dag.Nodes))
	}

	coordSuccessors := dag.getSuccessors("coordinator")
	if !slices.Contains(coordSuccessors, "planner") {
		t.Errorf("coordinator should have 'planner' as successor, got %v", coordSuccessors)
	}

	plannerSuccessors := dag.getSuccessors("planner")
	if len(plannerSuccessors) != 2 {
		t.Errorf("planner should have 2 successors, got %v", plannerSuccessors)
	}

	if !dag.isTerminal("finalizer") {
		t.Error("finalizer should be terminal")
	}

	if dag.isTerminal("planner") {
		t.Error("planner should not be terminal")
	}
}

func TestCycleDetection(t *testing.T) {
	cyclicDot := `digraph cycle {
		a -> b
		b -> c
		c -> a
	}`

	_, err := parseFlow(cyclicDot, "")
	if err == nil {
		t.Error("expected cycle detection error, got nil")
	}
}

func TestDetermineNextAgentReturnsCoordinator(t *testing.T) {
	dotContent := `digraph workflow {
		planner -> coder
		planner -> researcher
	}`

	dag, err := parseFlow(dotContent, "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	next := determineNextAgent(dag, "planner")

	if next != "coordinator" {
		t.Errorf("expected 'coordinator', got '%s'", next)
	}
}

func TestTerminalNodeReturnsEmpty(t *testing.T) {
	dotContent := `digraph workflow {
		a -> b
	}`

	dag, err := parseFlow(dotContent, "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	next := determineNextAgent(dag, "b")

	if next != "" {
		t.Errorf("expected empty string for terminal node, got '%s'", next)
	}
}

func TestCoordinatorAlreadyInGraph(t *testing.T) {
	dotContent := `digraph workflow {
		coordinator -> planner
		planner -> coder
	}`

	dag, err := parseFlow(dotContent, "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	if _, exists := dag.Nodes["coordinator"]; !exists {
		t.Error("expected coordinator node to exist in graph")
	}

	if len(dag.EntryNodes) != 1 || dag.EntryNodes[0] != "coordinator" {
		t.Errorf("expected coordinator to be the only entry node, got %v", dag.EntryNodes)
	}

	agents := dag.allAgents()
	if !slices.Contains(agents, "coordinator") {
		t.Error("expected coordinator in allAgents()")
	}
}

// TestInjectCoordinatorEdges tests that coordinator is prepended to all entry nodes.
// This is the fix for the bug where only EntryNodes[0] was used after coordinator.
func TestInjectCoordinatorEdges(t *testing.T) {
	// This is the problematic flow from the bug report:
	// - backend-go-developer and general-purpose are both entry nodes
	// - Without the fix, only backend-go-developer (alphabetically first) gets executed
	dotContent := `digraph workflow {
		"backend-go-developer" -> "go-readability-reviewer"
		"backend-go-developer" -> "htmx-picocss-frontend-developer"
		"general-purpose" -> "htmx-picocss-frontend-developer"
	}`

	dag, err := parseFlow(dotContent, "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	// After injection, coordinator should be the only entry node
	if len(dag.EntryNodes) != 1 || dag.EntryNodes[0] != "coordinator" {
		t.Errorf("expected coordinator to be the only entry node, got %v", dag.EntryNodes)
	}

	// Coordinator should exist in the graph
	coordNode, exists := dag.Nodes["coordinator"]
	if !exists {
		t.Fatal("expected coordinator node to exist in graph")
	}

	// Coordinator should have edges to both original entry nodes
	expectedSuccessors := []string{"backend-go-developer", "general-purpose"}
	for _, expected := range expectedSuccessors {
		if !slices.Contains(coordNode.Successors, expected) {
			t.Errorf("coordinator should have successor %q, got successors: %v", expected, coordNode.Successors)
		}
	}

	// Original entry nodes should now have coordinator as predecessor
	backendNode := dag.Nodes["backend-go-developer"]
	if !slices.Contains(backendNode.Predecessors, "coordinator") {
		t.Errorf("backend-go-developer should have coordinator as predecessor, got: %v", backendNode.Predecessors)
	}

	generalNode := dag.Nodes["general-purpose"]
	if !slices.Contains(generalNode.Predecessors, "coordinator") {
		t.Errorf("general-purpose should have coordinator as predecessor, got: %v", generalNode.Predecessors)
	}
}

func TestInjectCoordinatorEdgesWithCoordinatorAlreadyPresent(t *testing.T) {
	// When coordinator is explicitly in the flow but not connected to all entry nodes
	dotContent := `digraph workflow {
		coordinator -> "backend-go-developer"
		"backend-go-developer" -> "go-readability-reviewer"
		"general-purpose" -> "htmx-picocss-frontend-developer"
	}`

	dag, err := parseFlow(dotContent, "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	// Coordinator should be the only entry node
	if len(dag.EntryNodes) != 1 || dag.EntryNodes[0] != "coordinator" {
		t.Errorf("expected coordinator to be the only entry node, got %v", dag.EntryNodes)
	}

	coordNode := dag.Nodes["coordinator"]
	// Coordinator should have edges to both backend-go-developer AND general-purpose
	if !slices.Contains(coordNode.Successors, "backend-go-developer") {
		t.Errorf("coordinator should have successor backend-go-developer, got: %v", coordNode.Successors)
	}
	if !slices.Contains(coordNode.Successors, "general-purpose") {
		t.Errorf("coordinator should have successor general-purpose, got: %v", coordNode.Successors)
	}
}

func TestInjectCoordinatorEdgesSingleEntryNode(t *testing.T) {
	// When there's a single entry node (not coordinator), it should get coordinator prepended
	dotContent := `digraph workflow {
		planner -> coder
		coder -> reviewer
	}`

	dag, err := parseFlow(dotContent, "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	if len(dag.EntryNodes) != 1 || dag.EntryNodes[0] != "coordinator" {
		t.Errorf("expected coordinator to be the only entry node, got %v", dag.EntryNodes)
	}

	coordNode := dag.Nodes["coordinator"]
	if !slices.Contains(coordNode.Successors, "planner") {
		t.Errorf("coordinator should have successor planner, got: %v", coordNode.Successors)
	}

	plannerNode := dag.Nodes["planner"]
	if !slices.Contains(plannerNode.Predecessors, "coordinator") {
		t.Errorf("planner should have coordinator as predecessor, got: %v", plannerNode.Predecessors)
	}
}

func TestInjectCoordinatorEdgesCoordinatorIsOnlyEntryNode(t *testing.T) {
	dotContent := `digraph workflow {
		coordinator -> planner
		planner -> coder
	}`

	dag, err := parseFlow(dotContent, "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	if len(dag.EntryNodes) != 1 || dag.EntryNodes[0] != "coordinator" {
		t.Errorf("expected coordinator to be the only entry node, got %v", dag.EntryNodes)
	}

	coordNode := dag.Nodes["coordinator"]
	if !slices.Contains(coordNode.Successors, "planner") {
		t.Errorf("coordinator should have 'planner' as successor, got: %v", coordNode.Successors)
	}
}

func TestDefaultFlowWhenNoFrontmatter(t *testing.T) {
	dag, err := parseFlow("", "")
	if err != nil {
		t.Fatalf("parseFlow with empty spec failed: %v", err)
	}

	if len(dag.Nodes) != 2 {
		t.Errorf("expected 2 nodes (coordinator, general-purpose), got %d", len(dag.Nodes))
	}

	if len(dag.EntryNodes) != 1 || dag.EntryNodes[0] != "coordinator" {
		t.Errorf("expected coordinator to be the only entry node, got %v", dag.EntryNodes)
	}

	coordNode := dag.Nodes["coordinator"]
	if !slices.Contains(coordNode.Successors, "general-purpose") {
		t.Errorf("coordinator should have general-purpose as successor, got: %v", coordNode.Successors)
	}

	if !dag.isTerminal("general-purpose") {
		t.Error("general-purpose should be a terminal node")
	}
}

func TestExtractBody(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "with frontmatter",
			input: `---
flow: "a -> b"
model: claude
---
# My Goal

This is the body.`,
			expected: `# My Goal

This is the body.`,
		},
		{
			name: "no frontmatter",
			input: `# My Goal

This is the body without frontmatter.`,
			expected: `# My Goal

This is the body without frontmatter.`,
		},
		{
			name:     "empty file",
			input:    "",
			expected: "",
		},
		{
			name: "only frontmatter",
			input: `---
flow: "a -> b"
---
`,
			expected: "",
		},
		{
			name: "frontmatter without closing delimiter",
			input: `---
flow: "a -> b"
# Body starts here`,
			expected: `---
flow: "a -> b"
# Body starts here`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(extractBody([]byte(tt.input)))
			if result != tt.expected {
				t.Errorf("extractBody() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHashOnlyBody(t *testing.T) {
	// Test that changing frontmatter doesn't change the hash
	content1 := `---
model: claude-1
---
# Goal
Task description`

	content2 := `---
model: claude-2
---
# Goal
Task description`

	// Both should have the same body hash since only the model changed
	body1 := extractBody([]byte(content1))
	body2 := extractBody([]byte(content2))

	if string(body1) != string(body2) {
		t.Errorf("extractBody() produced different bodies:\n%q\nvs\n%q", body1, body2)
	}

	// Changing the body should change the hash
	content3 := `---
model: claude-1
---
# Goal
Different task description`

	body3 := extractBody([]byte(content3))
	if string(body1) == string(body3) {
		t.Error("extractBody() should produce different bodies when content differs")
	}
}
