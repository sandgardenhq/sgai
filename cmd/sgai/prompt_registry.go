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

const flowSectionMessaging = `.sgai/PROJECT_MANAGEMENT.md is the shared ledger for inter-agent state, handoffs, blockers, questions, and completion evidence. Read it before acting and append concise sections when handing work off or reporting completion.`

const flowSectionPeekMessageBus = `Use .sgai/PROJECT_MANAGEMENT.md to monitor inter-agent communication and workflow history.`

const flowSectionWorkFocus = `Critically, you must strictly do the work that you are an expert in, and leave other work to other agents.`

const flowSectionNavigation = `## Navigation
Navigation between agents is explicit:
- Append the handoff, blocker, question, or completion evidence to .sgai/PROJECT_MANAGEMENT.md
- Route to a DAG agent with sgai_update_workflow_state({status:"agent-done", navigate:{to:"agent-name", reason:"short reason"}})
- If no navigate target is provided, the system follows the workflow DAG and terminal agents return to coordinator

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
- If GOAL.md enables retrospective with retrospective: true, the system will route to the retrospective agent before completion
`

const flowSectionBuildingModeCoordinator = `- Your master plan: read GOAL.md → delegate to agents → verify with project-critic → complete
- When delegated work is done, invoke the project-critic wrapper subagent to verify all GOAL.md checkboxes are genuinely complete
- If GOAL.md enables retrospective with retrospective: true, wait for the system's retrospective routing before final completion
`

const flowSectionRetrospectiveMode = `# RETROSPECTIVE MODE ACTIVE
You are running in Retrospective mode. The building phase is complete.
- Human interaction tools (ask_user_question) are re-enabled
- If you are the retrospective agent: analyze session artifacts and append RETRO_QUESTION sections to PROJECT_MANAGEMENT.md, then navigate to coordinator
- If you are the coordinator: you MUST relay RETRO_QUESTION sections to the human using ask_user_question, then append the human's answer to PROJECT_MANAGEMENT.md and navigate back to retrospective
- This phase runs only when GOAL.md explicitly opts in with retrospective: true
`

const flowSectionRetrospectiveModeCoordinator = `- You are in the RETRO_QUESTION relay phase
- Check PROJECT_MANAGEMENT.md for RETRO_QUESTION sections from the retrospective agent
- When you see RETRO_QUESTION sections, relay them to the human using ask_user_question
- Append the human's actual response to PROJECT_MANAGEMENT.md and navigate back to retrospective
- When you see RETRO_COMPLETE sections, proceed to mark the workflow as complete
- Do NOT mark complete until RETRO_COMPLETE is received
`

const flowSectionPostSkillsCoordinator = `IMPORTANT: YOU COMMUNICATE WITH THE HUMAN ONLY VIA ask_user_question (structured multi-choice questions).`

const flowSectionPostSkillsNonCoordinator = `IMPORTANT: If you need human clarification, append QUESTION: <your question> to PROJECT_MANAGEMENT.md, then navigate to coordinator with sgai_update_workflow_state({status:"agent-done", navigate:{to:"coordinator", reason:"human clarification needed"}}). The coordinator will handle human communication.`

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
EXCEPTION: After calling sgai_update_workflow_state({status:"agent-done"}), you MUST NOT make any more tool calls. A text-only response after agent-done is correct behavior — the system handles the transition.

# CRITICAL: WHAT HAPPENS AFTER "agent-done"
When you set status: "agent-done":
1. If you included navigate, the system routes to that DAG agent
2. If no navigate target is provided, the system follows the workflow DAG and terminal agents return to coordinator
3. You should STOP making tool calls - your turn is over
4. Do NOT call sgai_update_workflow_state multiple times with the same status

ANTI-PATTERN: Setting "agent-done" then continuing to make calls (the system handles the transition!)
GOOD PATTERN: Do your work -> Call sgai_update_workflow_state({status:"agent-done"}) once -> STOP`

const flowSectionTailCoordinator = `IMPORTANT: You are the SOLE owner of GOAL.md checkboxes. When delegated work is confirmed complete, you MUST mark the corresponding checkbox by changing '- [ ]' to '- [x]'. Use skills({"name":"project-completion-verification"}) to check status and mark items. Look for 'GOAL COMPLETE:' PROJECT_MANAGEMENT.md entries from agents as triggers.`

const flowSectionTailNonCoordinator = `IMPORTANT: When you complete a task listed in GOAL.md, you MUST append a PROJECT_MANAGEMENT.md entry starting with GOAL COMPLETE: [exact checkbox text from GOAL.md], then navigate to coordinator. Do NOT attempt to edit GOAL.md yourself - only the coordinator can mark checkboxes.`

const flowSectionCommonTail = `IMPORTANT: use .sgai/PROJECT_MANAGEMENT.md to communicate with other agents and use navigate in sgai_update_workflow_state to route control
IMPORTANT: You must to search for known skills with skills({"name":""}) (for all skills), skills({"name":"skill-name"}) (for specific skills) before doing any work and skills({"name":"keywords"}) (for skills by keywords) to get the skill content and use skills when available.
IMPORTANT: You must to search for language specific code snippets with sgai_find_snippets()`

const flowSectionCoordinatorMessagingTail = `IMPORTANT: As coordinator, record your own status with sgai_update_workflow_state or .sgai/PROJECT_MANAGEMENT.md instead of routing to yourself.`

const flowSectionNonCoordinatorMessagingTail = `IMPORTANT: append status updates to .sgai/PROJECT_MANAGEMENT.md. When coordinator action is needed, navigate to coordinator with sgai_update_workflow_state({status:"agent-done", navigate:{to:"coordinator", reason:"status update ready"}}).`

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
	case state.ModeRetrospective:
		return flowSectionRetrospectiveMode, flowSectionRetrospectiveModeCoordinator
	default:
		return flowSectionBrainstormingMode, ""
	}
}
