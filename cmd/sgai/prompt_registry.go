package main

import (
	"strings"

	"github.com/sandgardenhq/sgai/pkg/state"
)

const promptSectionPreamble = `<UserInstructions>
YOU MUST LOAD THE SKILL "set-workflow-state" - CALL find_skills({"name":"set-workflow-state"}) TO DISCOVER IT, THEN CALL skill({"name":"set-workflow-state"}) TO GET THE SKILL CONTENT.
REMEMBER: file references like @FILENAME.md mean you must read the file $currentWorkingDirectory/FILENAME.md in the workspace.

RIGHT NOW, you must read @GOAL.md and @.sgai/PROJECT_MANAGEMENT.md, then work to achieve @GOAL.md;`

const promptSectionHumanCommDirect = `if you want to tell me something, use ask_user_question to present structured questions;`

const promptSectionHumanCommNonCoordinator = `if you want to tell me something, make sure you must call sgai_update_workflow_state (set blocked and a blocked message);`

const promptSectionMessaging = `.sgai/PROJECT_MANAGEMENT.md is the shared ledger for inter-agent state, handoffs, blockers, questions, and completion evidence. Read it before acting and append concise sections when handing work off or reporting completion.`

const promptSectionProjectManagementMonitor = `Use .sgai/PROJECT_MANAGEMENT.md to monitor inter-agent communication and workflow history.`

const promptSectionWorkFocus = `Critically, you must strictly do the work that you are an expert in, and leave other work to other agents.`

const promptSectionDelegation = `## Delegation
SGAI runs this top-level OpenCode session as the coordinator. Use available OpenCode subagents for delegation instead of routing the SGAI runtime between agents:
- Append handoffs, blockers, questions, or completion evidence to .sgai/PROJECT_MANAGEMENT.md
- Delegate to available subagents through OpenCode's subagent/delegation mechanisms
- Use sgai_update_workflow_state for durable status only; do not use navigate to cycle SGAI through the GOAL agents

## Available OpenCode Subagents for Delegation
%AGENTS_LIST%

## Visit Counts
%VISIT_COUNTS%

</UserInstructions>.

ABSOLUTELY CRITICAL: always USE SKILLS WHEN ONE SKILL IS AVAILABLE, DIG THE SKILL CONTENT TO BE SURE IT IS APPLICABLE. Use find_skills({"name":"keywords"}) to discover skills by name, tag, or keyword, then use skill({"name":"skill-name"}) to load the skill content.`

const promptSectionSelfDriveMode = `# SELF-DRIVE MODE ACTIVE
You are running in Self-Drive mode. This means:
- NO human interaction is allowed at any point
- The ask_user_question and ask_user_work_gate tools DO NOT EXIST
- Skip the BRAINSTORMING step entirely - go directly to work
- Skip the WORK-GATE step entirely - it is implicitly approved
- If your instructions say to brainstorm or ask for approval, SKIP those steps completely
`

const promptSectionSelfDriveModeCoordinator = `- Your master plan starts at reading GOAL.md, then immediately delegate work to specialized agents
- Proceed directly: read GOAL.md → create PROJECT_MANAGEMENT.md → delegate to agents → verify → complete
`

const promptSectionBuildingMode = `# BUILDING MODE ACTIVE
You are running in Building mode. The brainstorming and work-gate phases are complete.
- The human partner has approved the definition — proceed directly to work
- Skip the BRAINSTORMING step entirely - it is already done
- Skip the WORK-GATE step entirely - it is already approved
- Do NOT use ask_user_question or ask_user_work_gate during the building phase
- If GOAL.md enables retrospective with retrospective: true, handle retrospective work as coordinator-owned analysis before completion
`

const promptSectionBuildingModeCoordinator = `- Your master plan: read GOAL.md → delegate to agents → verify with project-critic → complete
- When delegated work is done, invoke the project-critic wrapper subagent to verify all GOAL.md checkboxes are genuinely complete
- If GOAL.md enables retrospective with retrospective: true, perform coordinator-owned retrospective analysis before final completion
`

const promptSectionRetrospectiveMode = `# RETROSPECTIVE MODE ACTIVE
You are running in Retrospective mode. The building phase is complete.
- Human interaction tools (ask_user_question) are re-enabled
- The coordinator owns retrospective analysis, human questions, and final completion decisions
- Analyze session artifacts, ask any needed human questions directly, and record outcomes in PROJECT_MANAGEMENT.md
- This phase runs only when GOAL.md explicitly opts in with retrospective: true
`

const promptSectionRetrospectiveModeCoordinator = `- Perform coordinator-owned retrospective analysis before final completion
- Ask the human direct retrospective questions when needed, then record answers in PROJECT_MANAGEMENT.md
- Complete when the coordinator-owned retrospective is done and project completion criteria are satisfied
`

const promptSectionPostSkillsCoordinator = `IMPORTANT: YOU COMMUNICATE WITH THE HUMAN ONLY VIA ask_user_question (structured multi-choice questions).`

const promptSectionPostSkillsNonCoordinator = `IMPORTANT: If you need human clarification, append QUESTION: <your question> to PROJECT_MANAGEMENT.md, then finish with sgai_update_workflow_state({status:"agent-done"}) so control returns through OpenCode's subagent/delegation mechanism. The coordinator will handle human communication.`

const promptSectionGuidelines = `# PRODUCTIVE WORK GUIDELINES
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
1. The OpenCode subagent/delegation mechanism returns control to the coordinator
2. Use PROJECT_MANAGEMENT.md for any handoff details the coordinator must read
3. You should STOP making tool calls - your turn is over
4. Do NOT call sgai_update_workflow_state multiple times with the same status

ANTI-PATTERN: Setting "agent-done" then continuing to make calls (the system handles the transition!)
GOOD PATTERN: Do your work -> Call sgai_update_workflow_state({status:"agent-done"}) once -> STOP`

const promptSectionTailCoordinator = `IMPORTANT: You are the SOLE owner of GOAL.md checkboxes. When delegated work is confirmed complete, you MUST mark the corresponding checkbox by changing '- [ ]' to '- [x]'. Use find_skills({"name":"project-completion-verification"}) to discover the skill, then skill({"name":"project-completion-verification"}) to check status and mark items. Look for 'GOAL COMPLETE:' PROJECT_MANAGEMENT.md entries from agents as triggers.`

const promptSectionTailNonCoordinator = `IMPORTANT: When you complete a task listed in GOAL.md, you MUST append a PROJECT_MANAGEMENT.md entry starting with GOAL COMPLETE: [exact checkbox text from GOAL.md], then finish so control returns to the coordinator through OpenCode's subagent/delegation mechanism. Do NOT attempt to edit GOAL.md yourself - only the coordinator can mark checkboxes.`

const promptSectionCommonTail = `IMPORTANT: use .sgai/PROJECT_MANAGEMENT.md to communicate with other agents and use OpenCode subagent/delegation handoff to return control to the coordinator
IMPORTANT: You must search for known skills with find_skills({"name":""}) (for all skills), find_skills({"name":"skill-name"}) (for specific skills), or find_skills({"name":"keywords"}) (for skills by keywords) before doing any work, then use skill({"name":"skill-name"}) to get the skill content when available.
IMPORTANT: You must to search for language specific code snippets with sgai_find_snippets()`

const promptSectionCoordinatorMessagingTail = `IMPORTANT: As coordinator, record your own status with sgai_update_workflow_state or .sgai/PROJECT_MANAGEMENT.md instead of routing to yourself.`

const promptSectionNonCoordinatorMessagingTail = `IMPORTANT: append status updates to .sgai/PROJECT_MANAGEMENT.md. When coordinator action is needed, finish and return through OpenCode's subagent/delegation mechanism.`

const promptSectionBrainstormingMode = "CRITICAL: think hard and ASK ME QUESTIONS BEFORE BUILDING\n"

const promptSectionContinuousMode = `# CONTINUOUS MODE ACTIVE
You are running in Continuous Mode. This means:
- NO human interaction is allowed at any point
- The ask_user_question and ask_user_work_gate tools DO NOT EXIST
- Skip the BRAINSTORMING step entirely - go directly to work
- Skip the WORK-GATE step entirely - it is implicitly approved
- Retrospectives are NEVER run in this mode
- If your instructions say to brainstorm, ask for approval, or run a retrospective, SKIP those steps completely
`

const promptSectionContinuousModeCoordinator = `- Your master plan starts at reading GOAL.md, then immediately delegate work to specialized agents
- Proceed directly: read GOAL.md → create PROJECT_MANAGEMENT.md → delegate to agents → verify → complete
- Do NOT run retrospective work in Continuous Mode
`

type promptOptions struct {
	agent           string
	modeSection     string
	coordinatorPlan string
}

func composePrompt(opts promptOptions) string {
	base := composeCoordinatorPromptTemplate(opts.agent)

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
		return promptSectionSelfDriveMode, promptSectionSelfDriveModeCoordinator
	case state.ModeContinuous:
		return promptSectionContinuousMode, promptSectionContinuousModeCoordinator
	case state.ModeBuilding:
		return promptSectionBuildingMode, promptSectionBuildingModeCoordinator
	case state.ModeRetrospective:
		return promptSectionRetrospectiveMode, promptSectionRetrospectiveModeCoordinator
	default:
		return promptSectionBrainstormingMode, ""
	}
}
