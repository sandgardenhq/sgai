package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mactaggart/gographviz"
)

const flowContinueMessageCoordinator = `
<UserInstructions>
YOU MUST LOAD THE SKILL "set-work-state" - CALL skills({"name":"set-workflow-state"}) TO GET THE SKILL CONTENT.
REMEMBER: file references like @FILENAME.md mean you must read the file $currentWorkingDirectory/FILENAME.md in the workspace.

RIGHT NOW, you must read @GOAL.md and @.sgai/PROJECT_MANAGEMENT.md, then work to achieve @GOAL.md;

if you want to tell me something, use ask_user_question to present structured questions;

You can send messages to other agents using sgai_send_message() (make sure you call sgai_check_outbox() to see if you haven't sent the message you want to send, avoid duplicated messages) and read messages using sgai_check_inbox(). You can also read messages from other agents and send messages to other agents by writing them into @PROJECT_MANAGEMENT.md

You can use peek_message_bus() to monitor ALL inter-agent communication (both pending and read messages) in reverse chronological order.

Critically, you must strictly do the work that you are an expert in, and leave other work to other agents.

## Message-Driven Navigation
Navigation between agents is driven by inter-agent messages:
- Send a message to an agent using sgai_send_message() to route work to them
- When you set status "agent-done", the system checks for pending messages and routes to the agent with the oldest unread message
- When no messages are pending, control returns to coordinator

## Your Position in the Workflow
Current agent: %CURRENT_AGENT%
Predecessors (can receive work from): %PREDECESSORS%
Successors (can pass work to): %SUCCESSORS%

## Visit Counts
%VISIT_COUNTS%

## All Agents
%AGENTS_LIST%

</UserInstructions>.

ABSOLUTELY CRITICAL: always USE SKILLS WHEN ONE SKILL IS AVAILABLE, DIG THE SKILL CONTENT TO BE SURE IT IS APPLICABLE. Use skills({"name":"skill-name"}) to get the skill content, or use skills({"name":"keywords"}) to find skills by tags.
IMPORTANT: YOU COMMUNICATE WITH THE HUMAN ONLY VIA ask_user_question (structured multi-choice questions).

# PRODUCTIVE WORK GUIDELINES
BEFORE calling sgai_update_workflow_state, ask yourself:
1. Have I actually done productive work this turn? (read files, wrote code, ran commands, analyzed results)
2. If I only called sgai_update_workflow_state with status "working", I'm wasting a turn - DO SOMETHING PRODUCTIVE FIRST.
3. Status "working" should be used ONLY after doing substantial work that needs continuation.
4. If my work is complete, use status "agent-done" so the system can move forward.

ANTI-PATTERN: Repeatedly calling sgai_update_workflow_state({status:"working"}) without doing real work creates infinite loops.
GOOD PATTERN: Read files -> Write code -> Run tests -> THEN sgai_update_workflow_state with appropriate status.

# COMPLETION GATE
CRITICALLY IMPORTANT: IF YOUR LAST MESSAGE IS NOT A TOOL CALL, THE HUMAN PARTNER WILL NOT SEE IT.
DID YOU DO PRODUCTIVE WORK before updating state? If not, go do something useful first.

# CRITICAL: WHAT HAPPENS AFTER "agent-done"
When you set status: "agent-done":
1. The system checks for pending messages and routes to the agent with the oldest unread message
2. If no messages are pending, control returns to coordinator
3. You should STOP making tool calls - your turn is over
4. Do NOT call sgai_update_workflow_state multiple times with the same status

ANTI-PATTERN: Setting "agent-done" then continuing to make calls (the system handles the transition!)
GOOD PATTERN: Do your work -> Call sgai_update_workflow_state({status:"agent-done"}) once -> STOP

IMPORTANT: You are the SOLE owner of GOAL.md checkboxes. When delegated work is confirmed complete, you MUST mark the corresponding checkbox by changing '- [ ]' to '- [x]'. Use skills({"name":"project-completion-verification"}) to check status and mark items. Look for 'GOAL COMPLETE:' messages from agents as triggers.
IMPORTANT: use CALL sgai_send_message({ toAgent: "name-of-the-agent", body: "your message here"}) to communicate with other agents
IMPORTANT: use CALL sgai_send_message({ toAgent: "coordinator", body: "here you write a status update of the progress of your job"}) to communicate with other agents
IMPORTANT: You must to search for known skills with skills({"name":""}) (for all skills), skills({"name":"skill-name"}) (for specific skills) before doing any work and skills({"name":"keywords"}) (for skills by keywords) to get the skill content and use skills when available.
IMPORTANT: You must to search for language specific code snippets with sgai_find_snippets()
`

const flowContinueMessageNonCoordinator = `
<UserInstructions>
YOU MUST LOAD THE SKILL "set-work-state" - CALL skills({"name":"set-workflow-state"}) TO GET THE SKILL CONTENT.
REMEMBER: file references like @FILENAME.md mean you must read the file $currentWorkingDirectory/FILENAME.md in the workspace.

RIGHT NOW, you must read @GOAL.md and @.sgai/PROJECT_MANAGEMENT.md, then work to achieve @GOAL.md;

if you want to tell me something, make sure you must call sgai_update_workflow_state (set blocked and a blocked message);

You can send messages to other agents using sgai_send_message() (make sure you call sgai_check_outbox() to see if you haven't sent the message you want to send, avoid duplicated messages) and read messages using sgai_check_inbox(). You can also read messages from other agents and send messages to other agents by writing them into @PROJECT_MANAGEMENT.md

Critically, you must strictly do the work that you are an expert in, and leave other work to other agents.

## Message-Driven Navigation
Navigation between agents is driven by inter-agent messages:
- Send a message to an agent using sgai_send_message() to route work to them
- When you set status "agent-done", the system checks for pending messages and routes to the agent with the oldest unread message
- When no messages are pending, control returns to coordinator

## Your Position in the Workflow
Current agent: %CURRENT_AGENT%
Predecessors (can receive work from): %PREDECESSORS%
Successors (can pass work to): %SUCCESSORS%

## Visit Counts
%VISIT_COUNTS%

## All Agents
%AGENTS_LIST%

</UserInstructions>.

ABSOLUTELY CRITICAL: always USE SKILLS WHEN ONE SKILL IS AVAILABLE, DIG THE SKILL CONTENT TO BE SURE IT IS APPLICABLE. Use skills({"name":"skill-name"}) to get the skill content, or use skills({"name":"keywords"}) to find skills by tags.
IMPORTANT: If you need human clarification, send a message to coordinator: sgai_send_message({toAgent: "coordinator", body: "QUESTION: <your question>"}). The coordinator will handle human communication.

# PRODUCTIVE WORK GUIDELINES
BEFORE calling sgai_update_workflow_state, ask yourself:
1. Have I actually done productive work this turn? (read files, wrote code, ran commands, analyzed results)
2. If I only called sgai_update_workflow_state with status "working", I'm wasting a turn - DO SOMETHING PRODUCTIVE FIRST.
3. Status "working" should be used ONLY after doing substantial work that needs continuation.
4. If my work is complete, use status "agent-done" so the system can move forward.

ANTI-PATTERN: Repeatedly calling sgai_update_workflow_state({status:"working"}) without doing real work creates infinite loops.
GOOD PATTERN: Read files -> Write code -> Run tests -> THEN sgai_update_workflow_state with appropriate status.

# COMPLETION GATE
CRITICALLY IMPORTANT: IF YOUR LAST MESSAGE IS NOT A TOOL CALL, THE HUMAN PARTNER WILL NOT SEE IT.
DID YOU DO PRODUCTIVE WORK before updating state? If not, go do something useful first.

# CRITICAL: WHAT HAPPENS AFTER "agent-done"
When you set status: "agent-done":
1. The system checks for pending messages and routes to the agent with the oldest unread message
2. If no messages are pending, control returns to coordinator
3. You should STOP making tool calls - your turn is over
4. Do NOT call sgai_update_workflow_state multiple times with the same status

ANTI-PATTERN: Setting "agent-done" then continuing to make calls (the system handles the transition!)
GOOD PATTERN: Do your work -> Call sgai_update_workflow_state({status:"agent-done"}) once -> STOP

IMPORTANT: When you complete a task listed in GOAL.md, you MUST notify the coordinator: sgai_send_message({toAgent: "coordinator", body: "GOAL COMPLETE: [exact checkbox text from GOAL.md]"}). Do NOT attempt to edit GOAL.md yourself - only the coordinator can mark checkboxes.
IMPORTANT: use CALL sgai_send_message({ toAgent: "name-of-the-agent", body: "your message here"}) to communicate with other agents
IMPORTANT: use CALL sgai_send_message({ toAgent: "coordinator", body: "here you write a status update of the progress of your job"}) to communicate with other agents
IMPORTANT: You must to search for known skills with skills({"name":""}) (for all skills), skills({"name":"skill-name"}) (for specific skills) before doing any work and skills({"name":"keywords"}) (for skills by keywords) to get the skill content and use skills when available.
IMPORTANT: You must to search for language specific code snippets with sgai_find_snippets()
`

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

func buildFlowMessage(d *dag, currentAgent string, visitCounts map[string]int, dir string, interactive string) string {
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

	var msg string
	if currentAgent == "coordinator" {
		msg = flowContinueMessageCoordinator
	} else {
		msg = flowContinueMessageNonCoordinator
	}

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

	if interactive == "yes" {
		msg += "\n\nCRITICAL: think hard and ASK ME QUESTIONS BEFORE BUILDING\n"
	}

	return msg
}
