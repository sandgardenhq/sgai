package main

import (
	"slices"
	"strings"
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

	if len(dag.Nodes) != 7 {
		t.Errorf("expected 7 nodes (5 original + coordinator + project-critic-council), got %d", len(dag.Nodes))
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

	if len(dag.Nodes) != 3 {
		t.Errorf("expected 3 nodes (coordinator, general-purpose, project-critic-council), got %d", len(dag.Nodes))
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

func TestInjectProjectCriticCouncilEdgeCreatesNodeAndEdge(t *testing.T) {
	dotContent := `digraph workflow {
		coordinator -> planner
		planner -> coder
	}`

	dag, err := parseFlow(dotContent, "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	pccNode, exists := dag.Nodes["project-critic-council"]
	if !exists {
		t.Fatal("expected project-critic-council node to exist")
	}

	coordNode := dag.Nodes["coordinator"]
	if !slices.Contains(coordNode.Successors, "project-critic-council") {
		t.Errorf("coordinator should have project-critic-council as successor, got: %v", coordNode.Successors)
	}

	if !slices.Contains(pccNode.Predecessors, "coordinator") {
		t.Errorf("project-critic-council should have coordinator as predecessor, got: %v", pccNode.Predecessors)
	}
}

func TestInjectProjectCriticCouncilEdgeExistingNodeNoEdge(t *testing.T) {
	dotContent := `digraph workflow {
		coordinator -> planner
		"project-critic-council"
	}`

	dag, err := parseFlow(dotContent, "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	pccNode := dag.Nodes["project-critic-council"]
	coordNode := dag.Nodes["coordinator"]

	if !slices.Contains(coordNode.Successors, "project-critic-council") {
		t.Errorf("coordinator should have project-critic-council as successor, got: %v", coordNode.Successors)
	}

	if !slices.Contains(pccNode.Predecessors, "coordinator") {
		t.Errorf("project-critic-council should have coordinator as predecessor, got: %v", pccNode.Predecessors)
	}
}

func TestInjectProjectCriticCouncilEdgeIdempotent(t *testing.T) {
	dotContent := `digraph workflow {
		coordinator -> "project-critic-council"
		coordinator -> planner
	}`

	dag, err := parseFlow(dotContent, "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	coordNode := dag.Nodes["coordinator"]
	pccNode := dag.Nodes["project-critic-council"]

	pccCount := 0
	for _, s := range coordNode.Successors {
		if s == "project-critic-council" {
			pccCount++
		}
	}
	if pccCount != 1 {
		t.Errorf("expected exactly 1 project-critic-council in coordinator.Successors, got %d", pccCount)
	}

	coordCount := 0
	for _, p := range pccNode.Predecessors {
		if p == "coordinator" {
			coordCount++
		}
	}
	if coordCount != 1 {
		t.Errorf("expected exactly 1 coordinator in project-critic-council.Predecessors, got %d", coordCount)
	}
}

func TestInjectProjectCriticCouncilEdgeSuccessorsSorted(t *testing.T) {
	dotContent := `digraph workflow {
		coordinator -> zebra
		coordinator -> alpha
	}`

	dag, err := parseFlow(dotContent, "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	coordNode := dag.Nodes["coordinator"]
	if !slices.IsSorted(coordNode.Successors) {
		t.Errorf("coordinator.Successors should be sorted, got: %v", coordNode.Successors)
	}
}

func TestInjectRetrospectiveEdgeCreatesNodeAndEdge(t *testing.T) {
	dotContent := `digraph workflow {
		coordinator -> planner
		planner -> coder
	}`

	dag, err := parseFlow(dotContent, "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	dag.injectRetrospectiveEdge()

	retroNode, exists := dag.Nodes["retrospective"]
	if !exists {
		t.Fatal("expected retrospective node to exist")
	}

	coordNode := dag.Nodes["coordinator"]
	if !slices.Contains(coordNode.Successors, "retrospective") {
		t.Errorf("coordinator should have retrospective as successor, got: %v", coordNode.Successors)
	}

	if !slices.Contains(retroNode.Predecessors, "coordinator") {
		t.Errorf("retrospective should have coordinator as predecessor, got: %v", retroNode.Predecessors)
	}
}

func TestInjectRetrospectiveEdgeIdempotent(t *testing.T) {
	dotContent := `digraph workflow {
		coordinator -> "retrospective"
		coordinator -> planner
	}`

	dag, err := parseFlow(dotContent, "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	dag.injectRetrospectiveEdge()

	coordNode := dag.Nodes["coordinator"]

	retroCount := 0
	for _, s := range coordNode.Successors {
		if s == "retrospective" {
			retroCount++
		}
	}
	if retroCount != 1 {
		t.Errorf("expected exactly 1 retrospective in coordinator.Successors, got %d", retroCount)
	}
}

func TestInjectRetrospectiveEdgeSuccessorsSorted(t *testing.T) {
	dotContent := `digraph workflow {
		coordinator -> zebra
		coordinator -> alpha
	}`

	dag, err := parseFlow(dotContent, "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	dag.injectRetrospectiveEdge()

	coordNode := dag.Nodes["coordinator"]
	if !slices.IsSorted(coordNode.Successors) {
		t.Errorf("coordinator.Successors should be sorted after retrospective injection, got: %v", coordNode.Successors)
	}
}

func TestRetrospectiveAppearsInAllAgents(t *testing.T) {
	dotContent := `digraph workflow {
		planner -> coder
	}`

	dag, err := parseFlow(dotContent, "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	dag.injectRetrospectiveEdge()

	agents := dag.allAgents()
	if !slices.Contains(agents, "retrospective") {
		t.Errorf("allAgents() should contain retrospective, got: %v", agents)
	}
}

func TestParseFlowDoesNotInjectRetrospective(t *testing.T) {
	dotContent := `digraph workflow {
		coordinator -> planner
		planner -> coder
	}`

	dag, err := parseFlow(dotContent, "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	if _, exists := dag.Nodes["retrospective"]; exists {
		t.Error("parseFlow should not inject retrospective node; injection is now conditional")
	}
}

func TestComposeFlowTemplateCoordinatorContent(t *testing.T) {
	msg := composeFlowTemplate("coordinator")

	requiredPhrases := []string{
		"ask_user_question to present structured questions",
		"peek_message_bus()",
		"YOU COMMUNICATE WITH THE HUMAN ONLY VIA ask_user_question",
		"SOLE owner of GOAL.md checkboxes",
		flowSectionPreamble,
		flowSectionMessaging,
		flowSectionWorkFocus,
		flowSectionNavigation,
		flowSectionGuidelines,
		flowSectionCommonTail,
	}
	for _, phrase := range requiredPhrases {
		if !strings.Contains(msg, phrase) {
			t.Errorf("coordinator template missing required phrase: %q", truncateForTest(phrase))
		}
	}

	absentPhrases := []string{
		"send a message to coordinator: sgai_send_message",
		"notify the coordinator",
		"GOAL COMPLETE: [exact checkbox text",
	}
	for _, phrase := range absentPhrases {
		if strings.Contains(msg, phrase) {
			t.Errorf("coordinator template should not contain: %q", phrase)
		}
	}
}

func TestComposeFlowTemplateNonCoordinatorContent(t *testing.T) {
	msg := composeFlowTemplate("backend-go-developer")

	requiredPhrases := []string{
		"sgai_update_workflow_state (set blocked",
		"send a message to coordinator",
		"notify the coordinator",
		"GOAL COMPLETE: [exact checkbox text",
		flowSectionPreamble,
		flowSectionMessaging,
		flowSectionWorkFocus,
		flowSectionNavigation,
		flowSectionGuidelines,
		flowSectionCommonTail,
	}
	for _, phrase := range requiredPhrases {
		if !strings.Contains(msg, phrase) {
			t.Errorf("non-coordinator template missing required phrase: %q", truncateForTest(phrase))
		}
	}

	absentPhrases := []string{
		"peek_message_bus()",
		"SOLE owner of GOAL.md",
		"YOU COMMUNICATE WITH THE HUMAN ONLY VIA ask_user_question",
	}
	for _, phrase := range absentPhrases {
		if strings.Contains(msg, phrase) {
			t.Errorf("non-coordinator template should not contain: %q", phrase)
		}
	}
}

func TestComposeFlowTemplateRetrospectiveContent(t *testing.T) {
	msg := composeFlowTemplate("retrospective")

	requiredPhrases := []string{
		"sgai_update_workflow_state (set blocked",
		"send a message to coordinator",
		"notify the coordinator",
		"GOAL COMPLETE: [exact checkbox text",
		`"QUESTION: <your question>"`,
		"The coordinator will handle human communication",
		flowSectionPreamble,
		flowSectionMessaging,
		flowSectionWorkFocus,
		flowSectionNavigation,
		flowSectionGuidelines,
		flowSectionCommonTail,
	}
	for _, phrase := range requiredPhrases {
		if !strings.Contains(msg, phrase) {
			t.Errorf("retrospective template missing required phrase: %q", truncateForTest(phrase))
		}
	}

	absentPhrases := []string{
		"peek_message_bus()",
		"SOLE owner of GOAL.md",
		"YOU COMMUNICATE WITH THE HUMAN ONLY VIA ask_user_question",
		"RETRO_QUESTION:",
		"RETRO_COMPLETE:",
		"THROUGH the coordinator",
	}
	for _, phrase := range absentPhrases {
		if strings.Contains(msg, phrase) {
			t.Errorf("retrospective template should not contain: %q", phrase)
		}
	}

	nonCoordMsg := composeFlowTemplate("backend-go-developer")
	if msg != nonCoordMsg {
		t.Error("retrospective template should be identical to standard non-coordinator template")
	}
}

func TestComposeFlowTemplateSharedSections(t *testing.T) {
	coordMsg := composeFlowTemplate("coordinator")
	nonCoordMsg := composeFlowTemplate("backend-go-developer")
	retroMsg := composeFlowTemplate("retrospective")

	sharedSections := []string{
		flowSectionPreamble,
		flowSectionMessaging,
		flowSectionWorkFocus,
		flowSectionNavigation,
		flowSectionGuidelines,
		flowSectionCommonTail,
	}

	for _, section := range sharedSections {
		if !strings.Contains(coordMsg, section) {
			t.Errorf("coordinator missing shared section: %q", truncateForTest(section))
		}
		if !strings.Contains(nonCoordMsg, section) {
			t.Errorf("non-coordinator missing shared section: %q", truncateForTest(section))
		}
		if !strings.Contains(retroMsg, section) {
			t.Errorf("retrospective missing shared section: %q", truncateForTest(section))
		}
	}
}

func truncateForTest(s string) string {
	if len(s) > 80 {
		return s[:80] + "..."
	}
	return s
}
