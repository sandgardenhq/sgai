package main

import "github.com/sandgardenhq/sgai/pkg/state"

const promptSectionPreamble = `<UserInstructions>
YOU MUST LOAD THE SKILL "set-workflow-state" - CALL find_skills({"name":"set-workflow-state"}) TO DISCOVER IT, THEN CALL skill({"name":"set-workflow-state"}) TO GET THE SKILL CONTENT.
REMEMBER: file references like @FILENAME.md mean you must read the file $currentWorkingDirectory/FILENAME.md in the workspace.

RIGHT NOW, you must read @GOAL.md and @.sgai/PROJECT_MANAGEMENT.md, then work to achieve @GOAL.md;`

const promptSectionHumanCommDirect = `if you want to tell me something, use ask_user_question to present structured questions;`

const promptSectionMessaging = `.sgai/PROJECT_MANAGEMENT.md is the shared ledger for inter-agent state, handoffs, blockers, questions, and completion evidence. Read it before acting and append concise coordinator sections when handing work off or reporting completion. Coordinator state updates must be explicit: current phase, concrete work completed, next action, blocker if any, and the expected owner of the next step.`

const promptSectionProjectManagementMonitor = `Use .sgai/PROJECT_MANAGEMENT.md to monitor inter-agent communication and workflow history.`

const promptSectionWorkFocus = `Critically, you must strictly do the work that you are an expert in, and leave other work to other agents.`

const promptSectionDelegation = `## Delegation
SGAI runs this top-level session as the coordinator. Use the Task tool for subagent delegation instead of routing the SGAI runtime between agents:
- Append handoffs, blockers, questions, or completion evidence to .sgai/PROJECT_MANAGEMENT.md
- Make every coordinator state update explicit about current phase, completed work, next step, blocker status, and handoff target
- Delegate by calling the Task tool with one of the available subagent types listed below
- Do not use bash, shell commands, opencode, or opencode run to delegate work
- Use sgai_update_workflow_state for durable status only; delegate with the Task tool only

## Available Task Subagents for Delegation
%AGENTS_LIST%

</UserInstructions>.

ABSOLUTELY CRITICAL: always USE SKILLS WHEN ONE SKILL IS AVAILABLE, DIG THE SKILL CONTENT TO BE SURE IT IS APPLICABLE. Use find_skills({"name":"keywords"}) to discover skills by name, tag, or keyword, then use skill({"name":"skill-name"}) to load the skill content.`

const promptSectionSelfDriveMode = `# SELF-DRIVE MODE ACTIVE
You are running in Self-Drive mode. This means:
- NO human interaction is allowed at any point
- The ask_user_question and ask_user_work_gate tools DO NOT EXIST
- Skip the BRAINSTORMING step entirely - go directly to work
- Skip the WORK-GATE step entirely - it is implicitly approved
- Do NOT ask workflow-choice questions such as task decomposition, write spec first, direct implementation, or plan mode
- If your instructions say to brainstorm or ask for approval, SKIP those steps completely
`

const promptSectionSelfDriveModeCoordinator = `- Your master plan starts at reading GOAL.md, then immediately delegate work to specialized agents
- Proceed directly: read GOAL.md → create PROJECT_MANAGEMENT.md → delegate to agents → verify → complete
`

const promptSectionInteractiveMode = `# INTERACTIVE MODE ACTIVE
You are running in Interactive mode. This means:
- Human interaction tools may be available to the coordinator only
- Non-coordinator subagents must not ask the human directly
- Check GOAL.md and .sgai/PROJECT_MANAGEMENT.md to determine whether brainstorming, work gate, implementation, verification, or retrospective work is currently needed
- If the work gate is already approved in .sgai/PROJECT_MANAGEMENT.md, Do NOT ask workflow-choice questions such as task decomposition, write spec first, direct implementation, or plan mode
`

const promptSectionInteractiveModeCoordinator = `- Read GOAL.md and .sgai/PROJECT_MANAGEMENT.md, then continue from the durable ledger state
- If the work gate is already approved, delegate implementation with the Task tool and verify completion
- If requirements or approval are genuinely missing, ask the human through the appropriate question tool
`

const promptSectionPostSkillsCoordinator = `IMPORTANT: YOU COMMUNICATE WITH THE HUMAN ONLY VIA ask_user_question (structured multi-choice questions).`

const promptSectionGuidelines = `# PRODUCTIVE WORK GUIDELINES
BEFORE calling sgai_update_workflow_state, ask yourself:
1. Have I actually done productive work this turn? (read files, wrote code, ran commands, analyzed results)
2. If I only called sgai_update_workflow_state with status "working", I'm wasting a turn - DO SOMETHING PRODUCTIVE FIRST.
3. Status "working" should be used ONLY after doing substantial work that needs continuation.
4. If my work is complete, use status "agent-done" so the system can move forward.

ANTI-PATTERN: Repeatedly calling sgai_update_workflow_state({status:"working"}) without doing real work creates infinite loops.
GOOD PATTERN: Read files -> Write code -> Run tests -> THEN sgai_update_workflow_state with appropriate status.
GOOD STATUS DETAIL: "coordinator direct verified repository validation, reviewed go test ./cmd/sgai results, no blockers, next owner coordinator for completion review".

# COMPLETION GATE
CRITICALLY IMPORTANT: IF YOUR LAST MESSAGE IS NOT A TOOL CALL, THE HUMAN PARTNER WILL NOT SEE IT.
DID YOU DO PRODUCTIVE WORK before updating state? If not, go do something useful first.
EXCEPTION: After calling sgai_update_workflow_state({status:"agent-done"}), you MUST NOT make any more tool calls. A text-only response after agent-done is correct behavior — the system handles the transition.

# CRITICAL: WHAT HAPPENS AFTER "agent-done"
When you set status: "agent-done":
1. The Task subagent delegation mechanism returns control to the coordinator
2. Use PROJECT_MANAGEMENT.md for any handoff details the coordinator must read
3. You should STOP making tool calls - your turn is over
4. Do NOT call sgai_update_workflow_state multiple times with the same status

ANTI-PATTERN: Setting "agent-done" then continuing to make calls (the system handles the transition!)
GOOD PATTERN: Do your work -> Call sgai_update_workflow_state({status:"agent-done"}) once -> STOP`

const promptSectionTailCoordinator = `IMPORTANT: You are the SOLE owner of GOAL.md checkboxes. When delegated work is confirmed complete, you MUST mark the corresponding checkbox by changing '- [ ]' to '- [x]'. Use find_skills({"name":"project-completion-verification"}) to discover the skill, then skill({"name":"project-completion-verification"}) to check status and mark items. Look for 'GOAL COMPLETE:' PROJECT_MANAGEMENT.md entries from agents as triggers.`

const promptSectionCommonTail = `IMPORTANT: use .sgai/PROJECT_MANAGEMENT.md to communicate with other agents and use the Task subagent delegation handoff to return control to the coordinator
IMPORTANT: Coordinator state updates must be explicit. Include what changed, what evidence exists, what remains, and who owns the next step.
IMPORTANT: You must search for known skills with find_skills({"name":""}) (for all skills), find_skills({"name":"skill-name"}) (for specific skills), or find_skills({"name":"keywords"}) (for skills by keywords) before doing any work, then use skill({"name":"skill-name"}) to get the skill content when available.
IMPORTANT: You must to search for language specific code snippets with sgai_find_snippets()`

const promptSectionCoordinatorMessagingTail = `IMPORTANT: As coordinator, record your own status with sgai_update_workflow_state or .sgai/PROJECT_MANAGEMENT.md instead of routing to yourself. Say "coordinator direct" for your own work and identify Task subagents in the Task prompt you send them.`

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

func modeSectionForMode(interactionMode string) (modeSection, coordinatorPlan string) {
	switch interactionMode {
	case state.ModeSelfDrive:
		return promptSectionSelfDriveMode, promptSectionSelfDriveModeCoordinator
	case state.ModeContinuous:
		return promptSectionContinuousMode, promptSectionContinuousModeCoordinator
	case state.ModeInteractive:
		return promptSectionInteractiveMode, promptSectionInteractiveModeCoordinator
	default:
		return promptSectionInteractiveMode, promptSectionInteractiveModeCoordinator
	}
}
