package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mactaggart/gographviz"
)

func composeFlowTemplate(currentAgent string) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(flowSectionPreamble)
	sb.WriteString("\n\n")

	switch currentAgent {
	case "coordinator":
		sb.WriteString(flowSectionHumanCommDirect)
	default:
		sb.WriteString(flowSectionHumanCommNonCoordinator)
	}
	sb.WriteString("\n\n")

	sb.WriteString(flowSectionMessaging)
	sb.WriteString("\n\n")

	if currentAgent == "coordinator" {
		sb.WriteString(flowSectionPeekMessageBus)
		sb.WriteString("\n\n")
	}

	sb.WriteString(flowSectionWorkFocus)
	sb.WriteString("\n\n")
	sb.WriteString(flowSectionNavigation)
	sb.WriteString("\n")

	switch currentAgent {
	case "coordinator":
		sb.WriteString(flowSectionPostSkillsCoordinator)
	default:
		sb.WriteString(flowSectionPostSkillsNonCoordinator)
	}
	sb.WriteString("\n\n")

	sb.WriteString(flowSectionGuidelines)
	sb.WriteString("\n\n")

	switch currentAgent {
	case "coordinator":
		sb.WriteString(flowSectionTailCoordinator)
		sb.WriteString("\n")
	default:
		sb.WriteString(flowSectionTailNonCoordinator)
		sb.WriteString("\n")
	}

	sb.WriteString(flowSectionCommonTail)
	sb.WriteString("\n")

	return sb.String()
}

type dagNode struct {
	Name         string
	Predecessors []string
	Successors   []string
}

type dag struct {
	Nodes      map[string]*dagNode
	EntryNodes []string
}

func (d *dag) ensureNode(name string) *dagNode {
	node, exists := d.Nodes[name]
	if exists {
		return node
	}
	node = &dagNode{Name: name}
	d.Nodes[name] = node
	return node
}

func parseFlow(flowSpec string, dir string) (*dag, error) {
	var dotContent string

	switch {
	case isAutoFlowSpec(flowSpec):
		dotContent = "digraph G {\n\"coordinator\" -> \"general-purpose\"\n}"
	case strings.HasPrefix(flowSpec, "digraph"):
		dotContent = flowSpec
	case strings.HasPrefix(flowSpec, "@"):
		content, err := os.ReadFile(filepath.Join(dir, flowSpec[1:]))
		if err != nil {
			return nil, fmt.Errorf("failed to read flow file %s: %w", flowSpec[1:], err)
		}
		dotContent = string(content)
	default:
		dotContent = "digraph G {\n" + flowSpec + "\n}"
	}

	graphAst, err := gographviz.ParseString(dotContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DOT: %w", err)
	}

	graph := gographviz.NewGraph()
	if err := gographviz.Analyse(graphAst, graph); err != nil {
		return nil, fmt.Errorf("failed to analyze DOT: %w", err)
	}

	d := &dag{
		Nodes: make(map[string]*dagNode),
	}

	for _, node := range graph.Nodes.Sorted() {
		name := strings.Trim(node.Name, "\"")
		d.ensureNode(name)
	}

	for _, edge := range graph.Edges.Edges {
		src := strings.Trim(edge.Src, "\"")
		dst := strings.Trim(edge.Dst, "\"")

		srcNode := d.ensureNode(src)
		dstNode := d.ensureNode(dst)

		srcNode.Successors = append(srcNode.Successors, dst)
		dstNode.Predecessors = append(dstNode.Predecessors, src)
	}

	for name, node := range d.Nodes {
		if len(node.Predecessors) == 0 {
			d.EntryNodes = append(d.EntryNodes, name)
		}
	}
	slices.Sort(d.EntryNodes)

	if len(d.EntryNodes) == 0 {
		return nil, fmt.Errorf("no entry nodes found in DAG (all nodes have predecessors, which implies a cycle)")
	}

	d.injectCoordinatorEdges()
	d.injectProjectCriticCouncilEdge()

	if err := d.detectCycles(); err != nil {
		return nil, err
	}

	return d, nil
}

// injectCoordinatorEdges ensures coordinator is the entry point for all chains.
// Any entry node that is not "coordinator" will have a coordinator -> node edge added.
// After injection, coordinator becomes the only entry node.
func (d *dag) injectCoordinatorEdges() {
	if len(d.EntryNodes) == 1 && d.EntryNodes[0] == "coordinator" {
		return
	}

	coordNode := d.ensureNode("coordinator")
	for _, entryNode := range d.EntryNodes {
		if entryNode == "coordinator" {
			continue
		}
		if !slices.Contains(coordNode.Successors, entryNode) {
			coordNode.Successors = append(coordNode.Successors, entryNode)
		}
		targetNode := d.Nodes[entryNode]
		if !slices.Contains(targetNode.Predecessors, "coordinator") {
			targetNode.Predecessors = append(targetNode.Predecessors, "coordinator")
		}
	}
	slices.Sort(coordNode.Successors)

	d.EntryNodes = []string{"coordinator"}
}

func (d *dag) injectProjectCriticCouncilEdge() {
	coordNode := d.ensureNode("coordinator")
	pccNode := d.ensureNode("project-critic-council")

	if !slices.Contains(coordNode.Successors, "project-critic-council") {
		coordNode.Successors = append(coordNode.Successors, "project-critic-council")
	}
	if !slices.Contains(pccNode.Predecessors, "coordinator") {
		pccNode.Predecessors = append(pccNode.Predecessors, "coordinator")
	}
	slices.Sort(coordNode.Successors)
}

func (d *dag) injectRetrospectiveEdge() {
	coordNode := d.ensureNode("coordinator")
	retroNode := d.ensureNode("retrospective")

	if !slices.Contains(coordNode.Successors, "retrospective") {
		coordNode.Successors = append(coordNode.Successors, "retrospective")
	}
	if !slices.Contains(retroNode.Predecessors, "coordinator") {
		retroNode.Predecessors = append(retroNode.Predecessors, "coordinator")
	}
	slices.Sort(coordNode.Successors)
}

func (d *dag) detectCycles() error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(node string) error
	dfs = func(node string) error {
		visited[node] = true
		recStack[node] = true

		for _, successor := range d.Nodes[node].Successors {
			if !visited[successor] {
				if err := dfs(successor); err != nil {
					return err
				}
			} else if recStack[successor] {
				return fmt.Errorf("cycle detected: %s -> %s", node, successor)
			}
		}

		recStack[node] = false
		return nil
	}

	for name := range d.Nodes {
		if !visited[name] {
			if err := dfs(name); err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *dag) getSuccessors(nodeName string) []string {
	if node, exists := d.Nodes[nodeName]; exists {
		return node.Successors
	}
	return nil
}

func (d *dag) getPredecessors(nodeName string) []string {
	if node, exists := d.Nodes[nodeName]; exists {
		return node.Predecessors
	}
	return nil
}

func (d *dag) isTerminal(nodeName string) bool {
	if node, exists := d.Nodes[nodeName]; exists {
		return len(node.Successors) == 0
	}
	return false
}

func (d *dag) allAgents() []string {
	agents := make([]string, 0, len(d.Nodes))
	for name := range d.Nodes {
		agents = append(agents, name)
	}
	slices.Sort(agents)
	return agents
}

func determineNextAgent(d *dag, currentAgent string) string {
	if d.isTerminal(currentAgent) {
		return ""
	}
	return "coordinator"
}

func (d *dag) toDOT() string {
	var lines []string
	agents := d.allAgents()
	lines = append(lines, "strict digraph G {")
	lines = append(lines, "    rankdir=LR;")

	for _, node := range agents {
		lines = append(lines, fmt.Sprintf(`    "%s"`, node))
	}

	for _, node := range agents {
		for _, succ := range d.Nodes[node].Successors {
			lines = append(lines, fmt.Sprintf(`    "%s" -> "%s"`, node, succ))
		}
	}

	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

// buildMultiModelSection generates a multi-model awareness section for the continuation message.
// Returns empty string if currentModel is empty (not in multi-model mode).
// When in multi-model mode, shows the current model's identity and lists sibling models.
func buildMultiModelSection(currentModel string, models map[string]any, currentAgent string) string {
	if currentModel == "" {
		return ""
	}

	agentModels := getModelsForAgent(models, currentAgent)
	if len(agentModels) <= 1 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n## Multi-Model Agent Context\n\n")
	sb.WriteString("You are running as part of a multi-model agent. Multiple models collaborate within this agent.\n\n")
	sb.WriteString("**Your identity:** ")
	sb.WriteString(currentModel)
	sb.WriteString("\n\n")
	sb.WriteString("**Sibling models in this agent:**\n")

	for _, modelSpec := range agentModels {
		modelID := formatModelID(currentAgent, modelSpec)
		sb.WriteString("  - ")
		sb.WriteString(modelID)
		if modelID == currentModel {
			sb.WriteString("  <-- YOU")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\nUse `sgai_send_message({toAgent: \"<sibling-model-id>\", body: \"...\"})` to message siblings.\n")
	sb.WriteString("Use `sgai_check_inbox()` to read messages from siblings.\n")

	return sb.String()
}

func buildFlowMessage(d *dag, currentAgent string, visitCounts map[string]int, dir string, interactionMode string, flowSpec string) string {
	predecessors := d.getPredecessors(currentAgent)
	predecessorsStr := strings.Join(predecessors, ", ")
	if predecessorsStr == "" {
		predecessorsStr = "(none - entry node)"
	}

	successors := d.getSuccessors(currentAgent)
	successorsStr := strings.Join(successors, ", ")
	if successorsStr == "" {
		successorsStr = "(none - terminal node)"
	}

	var visitLines []string
	agents := d.allAgents()
	for _, agent := range agents {
		count := visitCounts[agent]
		visitLines = append(visitLines, fmt.Sprintf("  %s: %d visits", agent, count))
	}
	visitCountsStr := strings.Join(visitLines, "\n")

	var agentLines []string
	for _, agent := range agents {
		agentPath := dir + "/.sgai/agent/" + agent + ".md"
		content, err := os.ReadFile(agentPath)
		var line string
		if err != nil {
			line = agent
		} else if desc := extractFrontmatterDescription(string(content)); desc != "" {
			line = agent + ": " + desc
		} else {
			line = agent
		}
		if agent == currentAgent {
			line += " <-- YOU ARE HERE"
		}
		agentLines = append(agentLines, line)
	}
	agentsListStr := strings.Join(agentLines, "\n")

	modeSection, coordPlan := modeSectionForMode(interactionMode)
	msg := composePrompt(promptOptions{
		agent:           currentAgent,
		modeSection:     modeSection,
		coordinatorPlan: coordPlan,
	})

	msg = strings.ReplaceAll(msg, "%CURRENT_AGENT%", currentAgent)
	msg = strings.ReplaceAll(msg, "%PREDECESSORS%", predecessorsStr)
	msg = strings.ReplaceAll(msg, "%SUCCESSORS%", successorsStr)
	msg = strings.ReplaceAll(msg, "%VISIT_COUNTS%", visitCountsStr)
	msg = strings.ReplaceAll(msg, "%AGENTS_LIST%", agentsListStr)

	snippets := parseAgentSnippets(dir, currentAgent)
	if len(snippets) > 0 {
		snippetsStr := strings.Join(snippets, ", ")
		snippetNudge := fmt.Sprintf("\nIMPORTANT: This agent specializes in %s. YOU MUST call sgai_find_snippets() for these languages BEFORE writing code: %s\n", snippetsStr, snippetsStr)
		msg += snippetNudge
	}

	if currentAgent == "coordinator" && isAutoFlowSpec(flowSpec) {
		msg += autoFlowNudge
	}

	return msg
}

func isAutoFlowSpec(flowSpec string) bool {
	return flowSpec == "" || flowSpec == "auto"
}

const autoFlowNudge = `
CRITICAL: The GOAL.md has no explicit agent flow configured (flow is empty or "auto").
You MUST use the ` + "`auto-flow-mode`" + ` skill to survey available agents, analyze the workspace,
select and pair appropriate agents, and update GOAL.md with the flow configuration.
Call skills({"name":"auto-flow-mode"}) IMMEDIATELY.
`
