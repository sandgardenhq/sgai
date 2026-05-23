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

	switch currentAgent {
	case "coordinator":
		sb.WriteString(flowSectionCoordinatorMessagingTail)
	default:
		sb.WriteString(flowSectionNonCoordinatorMessagingTail)
	}
	sb.WriteString("\n")

	return sb.String()
}

type dagNode struct {
	Name         string
	Predecessors []string
	Successors   []string
}

type dag struct {
	Nodes            map[string]*dagNode
	EntryNodes       []string
	parsedEntryNodes []string
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
	case flowSpec == "":
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
	d.parsedEntryNodes = slices.Clone(d.EntryNodes)

	if len(d.EntryNodes) == 0 {
		return nil, fmt.Errorf("no entry nodes found in DAG (all nodes have predecessors, which implies a cycle)")
	}

	d.injectCoordinatorEdges()

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

func (d *dag) removeRetrospective() {
	retrospectiveNode, exists := d.Nodes["retrospective"]
	if !exists {
		return
	}
	for _, predecessor := range retrospectiveNode.Predecessors {
		if predecessor == "retrospective" {
			continue
		}
		predecessorNode := d.Nodes[predecessor]
		if predecessorNode == nil {
			continue
		}
		for _, successor := range retrospectiveNode.Successors {
			if successor == "retrospective" || successor == predecessor {
				continue
			}
			successorNode := d.Nodes[successor]
			if successorNode == nil {
				continue
			}
			if !slices.Contains(predecessorNode.Successors, successor) {
				predecessorNode.Successors = append(predecessorNode.Successors, successor)
			}
			if !slices.Contains(successorNode.Predecessors, predecessor) {
				successorNode.Predecessors = append(successorNode.Predecessors, predecessor)
			}
		}
		slices.Sort(predecessorNode.Successors)
	}
	delete(d.Nodes, "retrospective")
	for _, node := range d.Nodes {
		node.Predecessors = slices.DeleteFunc(node.Predecessors, func(agent string) bool {
			return agent == "retrospective"
		})
		node.Successors = slices.DeleteFunc(node.Successors, func(agent string) bool {
			return agent == "retrospective"
		})
	}
	d.recomputeEntryNodes()
	d.injectCoordinatorEdges()
}

func (d *dag) recomputeEntryNodes() {
	d.EntryNodes = d.EntryNodes[:0]
	for name, node := range d.Nodes {
		if len(node.Predecessors) == 0 {
			d.EntryNodes = append(d.EntryNodes, name)
		}
	}
	slices.Sort(d.EntryNodes)
	d.parsedEntryNodes = slices.Clone(d.EntryNodes)
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

func buildFlowMessage(d *dag, currentAgent string, visitCounts map[string]int, dir string, interactionMode string, alias map[string]string) string {
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
		baseAgent := resolveBaseAgent(alias, agent)
		agentPath := dir + "/.sgai/agent/" + baseAgent + ".md"
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

	baseCurrentAgent := resolveBaseAgent(alias, currentAgent)
	snippets := parseAgentSnippets(dir, baseCurrentAgent)
	if len(snippets) > 0 {
		snippetsStr := strings.Join(snippets, ", ")
		snippetNudge := fmt.Sprintf("\nIMPORTANT: This agent specializes in %s. YOU MUST call sgai_find_snippets() for these languages BEFORE writing code: %s\n", snippetsStr, snippetsStr)
		msg += snippetNudge
	}

	return msg
}
