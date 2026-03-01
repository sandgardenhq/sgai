package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
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

func TestBuildFlowMessageSelfDriveMode(t *testing.T) {
	dotContent := `digraph workflow {
		coordinator -> planner
		planner -> coder
	}`

	d, err := parseFlow(dotContent, "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	visits := map[string]int{"coordinator": 1, "planner": 0, "coder": 0, "project-critic-council": 0}
	dir := t.TempDir()

	t.Run("selfDriveIncludesSelfDriveInstructions", func(t *testing.T) {
		msg := buildFlowMessage(d, "coordinator", visits, dir, state.ModeSelfDrive, "some-flow")
		if !strings.Contains(msg, "SELF-DRIVE MODE ACTIVE") {
			t.Error("self-drive message should contain SELF-DRIVE MODE ACTIVE")
		}
		if !strings.Contains(msg, "ask_user_question and ask_user_work_gate tools DO NOT EXIST") {
			t.Error("self-drive message should state tools do not exist")
		}
		if !strings.Contains(msg, "Skip the BRAINSTORMING step entirely") {
			t.Error("self-drive message should instruct to skip brainstorming")
		}
		if !strings.Contains(msg, "Skip the WORK-GATE step entirely") {
			t.Error("self-drive message should instruct to skip work-gate")
		}
		if strings.Contains(msg, "ASK ME QUESTIONS BEFORE BUILDING") {
			t.Error("self-drive message should not contain interactive prompt")
		}
	})

	t.Run("interactiveModeIncludesAskQuestions", func(t *testing.T) {
		msg := buildFlowMessage(d, "coordinator", visits, dir, state.ModeBrainstorming, "some-flow")
		if !strings.Contains(msg, "ASK ME QUESTIONS BEFORE BUILDING") {
			t.Error("interactive message should contain ASK ME QUESTIONS prompt")
		}
		if strings.Contains(msg, "SELF-DRIVE MODE ACTIVE") {
			t.Error("interactive message should not contain self-drive instructions")
		}
	})

	t.Run("selfDriveNonCoordinator", func(t *testing.T) {
		msg := buildFlowMessage(d, "planner", visits, dir, state.ModeSelfDrive, "some-flow")
		if !strings.Contains(msg, "SELF-DRIVE MODE ACTIVE") {
			t.Error("self-drive message for non-coordinator should contain SELF-DRIVE MODE ACTIVE")
		}
		if strings.Contains(msg, "ASK ME QUESTIONS BEFORE BUILDING") {
			t.Error("self-drive message for non-coordinator should not contain interactive prompt")
		}
		if strings.Contains(msg, "delegate work to specialized agents") {
			t.Error("self-drive message for non-coordinator should not contain coordinator delegation instructions")
		}
		if strings.Contains(msg, "create PROJECT_MANAGEMENT.md") {
			t.Error("self-drive message for non-coordinator should not contain coordinator delegation flow")
		}
	})

	t.Run("selfDriveCoordinatorIncludesDelegation", func(t *testing.T) {
		msg := buildFlowMessage(d, "coordinator", visits, dir, state.ModeSelfDrive, "some-flow")
		if !strings.Contains(msg, "delegate work to specialized agents") {
			t.Error("self-drive coordinator message should contain delegation instructions")
		}
		if !strings.Contains(msg, "create PROJECT_MANAGEMENT.md") {
			t.Error("self-drive coordinator message should contain delegation flow")
		}
	})

	t.Run("buildingModeIncludesBuildingInstructions", func(t *testing.T) {
		msg := buildFlowMessage(d, "coordinator", visits, dir, state.ModeBuilding, "some-flow")
		if !strings.Contains(msg, "BUILDING MODE ACTIVE") {
			t.Error("building mode message should contain BUILDING MODE ACTIVE")
		}
		if strings.Contains(msg, "SELF-DRIVE MODE ACTIVE") {
			t.Error("building mode message should not contain SELF-DRIVE MODE ACTIVE")
		}
		if strings.Contains(msg, "ASK ME QUESTIONS BEFORE BUILDING") {
			t.Error("building mode message should not contain interactive prompt")
		}
		if !strings.Contains(msg, "retrospective phase is STILL ACTIVE") {
			t.Error("building mode message should mention retrospective is still active")
		}
	})

	t.Run("buildingModeCoordinatorIncludesRetrospective", func(t *testing.T) {
		msg := buildFlowMessage(d, "coordinator", visits, dir, state.ModeBuilding, "some-flow")
		if !strings.Contains(msg, "run retrospective") {
			t.Error("building mode coordinator message should mention running retrospective")
		}
		if !strings.Contains(msg, "send a message to the retrospective agent") {
			t.Error("building mode coordinator message should instruct to message retrospective agent")
		}
	})

	t.Run("buildingModeNonCoordinator", func(t *testing.T) {
		msg := buildFlowMessage(d, "planner", visits, dir, state.ModeBuilding, "some-flow")
		if !strings.Contains(msg, "BUILDING MODE ACTIVE") {
			t.Error("building mode for non-coordinator should contain BUILDING MODE ACTIVE")
		}
		if strings.Contains(msg, "send a message to the retrospective agent") {
			t.Error("building mode for non-coordinator should not contain coordinator retrospective instructions")
		}
	})

	t.Run("emptyModeDefaultsToInteractive", func(t *testing.T) {
		msg := buildFlowMessage(d, "coordinator", visits, dir, "", "some-flow")
		if !strings.Contains(msg, "ASK ME QUESTIONS BEFORE BUILDING") {
			t.Error("empty mode message should contain interactive prompt")
		}
		if strings.Contains(msg, "SELF-DRIVE MODE ACTIVE") {
			t.Error("empty mode message should not contain self-drive instructions")
		}
		if strings.Contains(msg, "BUILDING MODE ACTIVE") {
			t.Error("empty mode message should not contain building instructions")
		}
	})
}

func TestParseFlowAuto(t *testing.T) {
	d, err := parseFlow("auto", "")
	if err != nil {
		t.Fatalf("parseFlow with auto spec failed: %v", err)
	}

	if len(d.Nodes) != 3 {
		t.Errorf("expected 3 nodes (coordinator, general-purpose, project-critic-council), got %d", len(d.Nodes))
	}

	if len(d.EntryNodes) != 1 || d.EntryNodes[0] != "coordinator" {
		t.Errorf("expected coordinator to be the only entry node, got %v", d.EntryNodes)
	}

	coordNode := d.Nodes["coordinator"]
	if !slices.Contains(coordNode.Successors, "general-purpose") {
		t.Errorf("coordinator should have general-purpose as successor, got: %v", coordNode.Successors)
	}

	if !d.isTerminal("general-purpose") {
		t.Error("general-purpose should be a terminal node")
	}
}

func TestParseFlowAutoMatchesEmpty(t *testing.T) {
	dagAuto, errAuto := parseFlow("auto", "")
	if errAuto != nil {
		t.Fatalf("parseFlow(auto) failed: %v", errAuto)
	}

	dagEmpty, errEmpty := parseFlow("", "")
	if errEmpty != nil {
		t.Fatalf("parseFlow(empty) failed: %v", errEmpty)
	}

	autoAgents := dagAuto.allAgents()
	emptyAgents := dagEmpty.allAgents()
	if !slices.Equal(autoAgents, emptyAgents) {
		t.Errorf("auto and empty should produce the same agents, got auto=%v, empty=%v", autoAgents, emptyAgents)
	}
}

func TestIsAutoFlowSpec(t *testing.T) {
	cases := []struct {
		name string
		spec string
		want bool
	}{
		{"empty", "", true},
		{"auto", "auto", true},
		{"explicitFlow", "coordinator -> planner", false},
		{"digraph", "digraph G {}", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAutoFlowSpec(tc.spec); got != tc.want {
				t.Errorf("isAutoFlowSpec(%q) = %v; want %v", tc.spec, got, tc.want)
			}
		})
	}
}

func TestBuildFlowMessageAutoFlowNudge(t *testing.T) {
	d, err := parseFlow("", "")
	if err != nil {
		t.Fatalf("parseFlow failed: %v", err)
	}

	visits := map[string]int{"coordinator": 0, "general-purpose": 0, "project-critic-council": 0}
	dir := t.TempDir()

	t.Run("coordinatorWithEmptyFlowGetsNudge", func(t *testing.T) {
		msg := buildFlowMessage(d, "coordinator", visits, dir, state.ModeBuilding, "")
		if !strings.Contains(msg, "auto-flow-mode") {
			t.Error("coordinator with empty flow should get auto-flow-mode nudge")
		}
		if !strings.Contains(msg, "CRITICAL") {
			t.Error("nudge should contain CRITICAL marker")
		}
	})

	t.Run("coordinatorWithAutoFlowGetsNudge", func(t *testing.T) {
		msg := buildFlowMessage(d, "coordinator", visits, dir, state.ModeBuilding, "auto")
		if !strings.Contains(msg, "auto-flow-mode") {
			t.Error("coordinator with auto flow should get auto-flow-mode nudge")
		}
	})

	t.Run("coordinatorWithExplicitFlowNoNudge", func(t *testing.T) {
		msg := buildFlowMessage(d, "coordinator", visits, dir, state.ModeBuilding, "coordinator -> planner")
		if strings.Contains(msg, "auto-flow-mode") {
			t.Error("coordinator with explicit flow should NOT get auto-flow-mode nudge")
		}
	})

	t.Run("nonCoordinatorWithEmptyFlowNoNudge", func(t *testing.T) {
		msg := buildFlowMessage(d, "general-purpose", visits, dir, state.ModeBuilding, "")
		if strings.Contains(msg, "auto-flow-mode") {
			t.Error("non-coordinator agents should NOT get auto-flow-mode nudge")
		}
	})
}

func TestRebuildDAG(t *testing.T) {
	t.Run("returnsCorrectDAG", func(t *testing.T) {
		metadata := GoalMetadata{Flow: "planner -> coder"}
		visitCounts := make(map[string]int)
		d, agents, longest, err := rebuildDAG(&metadata, t.TempDir(), visitCounts)
		if err != nil {
			t.Fatalf("rebuildDAG failed: %v", err)
		}
		if !slices.Contains(agents, "planner") {
			t.Error("agents should contain planner")
		}
		if !slices.Contains(agents, "coder") {
			t.Error("agents should contain coder")
		}
		if !slices.Contains(agents, "coordinator") {
			t.Error("agents should contain coordinator")
		}
		if _, exists := d.Nodes["planner"]; !exists {
			t.Error("DAG should contain planner node")
		}
		if _, exists := d.Nodes["coder"]; !exists {
			t.Error("DAG should contain coder node")
		}
		if longest < len("coordinator") {
			t.Errorf("longestNameLen should be at least %d, got %d", len("coordinator"), longest)
		}
	})

	t.Run("newAgentsGetZeroVisitCounts", func(t *testing.T) {
		metadata := GoalMetadata{Flow: "planner -> coder"}
		visitCounts := map[string]int{"coordinator": 3}
		_, _, _, err := rebuildDAG(&metadata, t.TempDir(), visitCounts)
		if err != nil {
			t.Fatalf("rebuildDAG failed: %v", err)
		}
		if visitCounts["planner"] != 0 {
			t.Errorf("new agent planner should have 0 visits, got %d", visitCounts["planner"])
		}
		if visitCounts["coder"] != 0 {
			t.Errorf("new agent coder should have 0 visits, got %d", visitCounts["coder"])
		}
	})

	t.Run("existingVisitCountsPreserved", func(t *testing.T) {
		metadata := GoalMetadata{Flow: "planner -> coder"}
		visitCounts := map[string]int{"coordinator": 5, "planner": 2}
		_, _, _, err := rebuildDAG(&metadata, t.TempDir(), visitCounts)
		if err != nil {
			t.Fatalf("rebuildDAG failed: %v", err)
		}
		if visitCounts["coordinator"] != 5 {
			t.Errorf("existing coordinator visits should be 5, got %d", visitCounts["coordinator"])
		}
		if visitCounts["planner"] != 2 {
			t.Errorf("existing planner visits should be 2, got %d", visitCounts["planner"])
		}
	})

	t.Run("invalidFlowReturnsError", func(t *testing.T) {
		metadata := GoalMetadata{Flow: "digraph INVALID {{{"}
		visitCounts := make(map[string]int)
		_, _, _, err := rebuildDAG(&metadata, t.TempDir(), visitCounts)
		if err == nil {
			t.Error("rebuildDAG should return error for invalid flow")
		}
	})

	t.Run("retrospectiveInjectedWhenEnabled", func(t *testing.T) {
		metadata := GoalMetadata{Flow: "planner -> coder", Retrospective: "yes"}
		visitCounts := make(map[string]int)
		d, agents, _, err := rebuildDAG(&metadata, t.TempDir(), visitCounts)
		if err != nil {
			t.Fatalf("rebuildDAG failed: %v", err)
		}
		if _, exists := d.Nodes["retrospective"]; !exists {
			t.Error("DAG should contain retrospective node when enabled")
		}
		if !slices.Contains(agents, "retrospective") {
			t.Error("agents should contain retrospective when enabled")
		}
	})

	t.Run("retrospectiveNotInjectedWhenDisabled", func(t *testing.T) {
		metadata := GoalMetadata{Flow: "planner -> coder", Retrospective: "no"}
		visitCounts := make(map[string]int)
		d, _, _, err := rebuildDAG(&metadata, t.TempDir(), visitCounts)
		if err != nil {
			t.Fatalf("rebuildDAG failed: %v", err)
		}
		if _, exists := d.Nodes["retrospective"]; exists {
			t.Error("DAG should not contain retrospective node when disabled")
		}
	})
}

func truncateForTest(s string) string {
	if len(s) > 80 {
		return s[:80] + "..."
	}
	return s
}
