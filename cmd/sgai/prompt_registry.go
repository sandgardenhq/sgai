package main

import (
	"strings"

	"github.com/sandgardenhq/sgai/pkg/state"
)

const flowSectionPreamble = `<UserInstructions>
YOU MUST LOAD THE SKILL "set-work-state" - CALL skills({"name":"set-workflow-state"}) TO GET THE SKILL CONTENT.
REMEMBER: file references like @FILENAME.md mean you must read the file $currentWorkingDirectory/FILENAME.md in the workspace.

RIGHT NOW, you must read @GOAL.md and @.sgai/PROJECT_MANAGEMENT.md, then work to achieve @GOAL.md;`

const flowSectionHumanCommDirect = `if you want to tell me something, use ask_user_question to present structured questions;`

const flowSectionHumanCommNonCoordinator = `if you want to tell me something, make sure you must call sgai_update_workflow_state (set blocked and a blocked message);`

const flowSectionMessaging = `You can send messages to other agents using sgai_send_message() (make sure you call sgai_check_outbox() to see if you haven't sent the message you want to send, avoid duplicated messages) and read messages using sgai_check_inbox(). You can also read messages from other agents and send messages to other agents by writing them into @PROJECT_MANAGEMENT.md`

const flowSectionPeekMessageBus = `You can use peek_message_bus() to monitor ALL inter-agent communication (both pending and read messages) in reverse chronological order.`

const flowSectionWorkFocus = `Critically, you must strictly do the work that you are an expert in, and leave other work to other agents.`

const flowSectionNavigation = `## Message-Driven Navigation
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

ABSOLUTELY CRITICAL: always USE SKILLS WHEN ONE SKILL IS AVAILABLE, DIG THE SKILL CONTENT TO BE SURE IT IS APPLICABLE. Use skills({"name":"skill-name"}) to get the skill content, or use skills({"name":"keywords"}) to find skills by tags.`

const flowSectionSelfDriveMode = `# SELF-DRIVE MODE ACTIVE
You are running in Self-Drive mode. This means:
- NO human interaction is allowed at any point
- The ask_user_question and ask_user_work_gate tools DO NOT EXIST
- Skip the BRAINSTORMING step entirely - go directly to work
- Skip the WORK-GATE step entirely - it is implicitly approved
- If your instructions say to brainstorm or ask for approval, SKIP those steps completely
`

const flowSectionSelfDriveModeCoordinator = `- Your master plan starts at reading GOAL.md, then immediately delegate work to specialized agents
- Proceed directly: read GOAL.md → create PROJECT_MANAGEMENT.md → delegate to agents → verify → complete
`

const flowSectionBuildingMode = `# BUILDING MODE ACTIVE
You are running in Building mode. The brainstorming and work-gate phases are complete.
- The human partner has approved the definition — proceed directly to work
- Skip the BRAINSTORMING step entirely - it is already done
- Skip the WORK-GATE step entirely - it is already approved
- Do NOT use ask_user_question or ask_user_work_gate during the building phase
- The retrospective phase is STILL ACTIVE — you must run it when all work is complete
- During the retrospective, the system will re-enable human interaction tools automatically
`

const flowSectionBuildingModeCoordinator = `- Your master plan: read GOAL.md → delegate to agents → verify → run retrospective → complete
- When delegated work is done, send a message to the retrospective agent to start analysis
`

const flowSectionPostSkillsCoordinator = `IMPORTANT: YOU COMMUNICATE WITH THE HUMAN ONLY VIA ask_user_question (structured multi-choice questions).`

const flowSectionPostSkillsNonCoordinator = `IMPORTANT: If you need human clarification, send a message to coordinator: sgai_send_message({toAgent: "coordinator", body: "QUESTION: <your question>"}). The coordinator will handle human communication.`

const flowSectionGuidelines = `# PRODUCTIVE WORK GUIDELINES
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
GOOD PATTERN: Do your work -> Call sgai_update_workflow_state({status:"agent-done"}) once -> STOP`

const flowSectionTailCoordinator = `IMPORTANT: You are the SOLE owner of GOAL.md checkboxes. When delegated work is confirmed complete, you MUST mark the corresponding checkbox by changing '- [ ]' to '- [x]'. Use skills({"name":"project-completion-verification"}) to check status and mark items. Look for 'GOAL COMPLETE:' messages from agents as triggers.`

const flowSectionTailNonCoordinator = `IMPORTANT: When you complete a task listed in GOAL.md, you MUST notify the coordinator: sgai_send_message({toAgent: "coordinator", body: "GOAL COMPLETE: [exact checkbox text from GOAL.md]"}). Do NOT attempt to edit GOAL.md yourself - only the coordinator can mark checkboxes.`

const flowSectionCommonTail = `IMPORTANT: use CALL sgai_send_message({ toAgent: "name-of-the-agent", body: "your message here"}) to communicate with other agents
IMPORTANT: use CALL sgai_send_message({ toAgent: "coordinator", body: "here you write a status update of the progress of your job"}) to communicate with other agents
IMPORTANT: You must to search for known skills with skills({"name":""}) (for all skills), skills({"name":"skill-name"}) (for specific skills) before doing any work and skills({"name":"keywords"}) (for skills by keywords) to get the skill content and use skills when available.
IMPORTANT: You must to search for language specific code snippets with sgai_find_snippets()`

const flowSectionBrainstormingMode = "CRITICAL: think hard and ASK ME QUESTIONS BEFORE BUILDING\n"

const flowSectionContinuousMode = `# CONTINUOUS MODE ACTIVE
You are running in Continuous Mode. This means:
- NO human interaction is allowed at any point
- The ask_user_question and ask_user_work_gate tools DO NOT EXIST
- Skip the BRAINSTORMING step entirely - go directly to work
- Skip the WORK-GATE step entirely - it is implicitly approved
- Retrospectives are NEVER run in this mode
- If your instructions say to brainstorm, ask for approval, or run a retrospective, SKIP those steps completely
`

const flowSectionContinuousModeCoordinator = `- Your master plan starts at reading GOAL.md, then immediately delegate work to specialized agents
- Proceed directly: read GOAL.md → create PROJECT_MANAGEMENT.md → delegate to agents → verify → complete
- Do NOT send work to the retrospective agent
`

type promptOptions struct {
	agent           string
	modeSection     string
	coordinatorPlan string
}

func composePrompt(opts promptOptions) string {
	base := composeFlowTemplate(opts.agent)

	if opts.modeSection == "" {
		return base
	}

	var sb strings.Builder
	sb.WriteString(base)
	sb.WriteString("\n\n")
	sb.WriteString(opts.modeSection)
	if opts.agent == "coordinator" && opts.coordinatorPlan != "" {
		sb.WriteString(opts.coordinatorPlan)
	}

	return sb.String()
}

func modeSectionForMode(interactionMode string) (modeSection, coordinatorPlan string) {
	switch interactionMode {
	case state.ModeSelfDrive:
		return flowSectionSelfDriveMode, flowSectionSelfDriveModeCoordinator
	case state.ModeContinuous:
		return flowSectionContinuousMode, flowSectionContinuousModeCoordinator
	case state.ModeBuilding:
		return flowSectionBuildingMode, flowSectionBuildingModeCoordinator
	default:
		return flowSectionBrainstormingMode, ""
	}
}
