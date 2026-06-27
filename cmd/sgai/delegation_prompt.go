package main

import (
	"os"
	"strings"
)

func composeCoordinatorPromptTemplate() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(promptSectionPreamble)
	sb.WriteString("\n\n")

	sb.WriteString(promptSectionHumanCommDirect)
	sb.WriteString("\n\n")

	sb.WriteString(promptSectionMessaging)
	sb.WriteString("\n\n")

	sb.WriteString(promptSectionProjectManagementMonitor)
	sb.WriteString("\n\n")

	sb.WriteString(promptSectionWorkFocus)
	sb.WriteString("\n\n")
	sb.WriteString(promptSectionDelegation)
	sb.WriteString("\n")

	sb.WriteString(promptSectionPostSkillsCoordinator)
	sb.WriteString("\n\n")

	sb.WriteString(promptSectionGuidelines)
	sb.WriteString("\n\n")

	sb.WriteString(promptSectionTailCoordinator)
	sb.WriteString("\n")

	sb.WriteString(promptSectionCommonTail)
	sb.WriteString("\n")

	sb.WriteString(promptSectionCoordinatorMessagingTail)
	sb.WriteString("\n")

	return sb.String()
}

func buildCoordinatorDelegationMessage(agents []string, dir string, interactionMode string) string {
	var agentLines []string
	for _, agent := range delegatableAgents(agents) {
		agentPath := dir + "/.sgai/agent/" + agent + ".md"
		content, errRead := os.ReadFile(agentPath)
		var line string
		if errRead != nil {
			line = agent
		} else if desc := extractFrontmatterDescription(string(content)); desc != "" {
			line = agent + ": " + desc
		} else {
			line = agent
		}
		agentLines = append(agentLines, line)
	}
	if len(agentLines) == 0 {
		agentLines = append(agentLines, "(none configured)")
	}
	agentsListStr := strings.Join(agentLines, "\n")

	modeSection, coordPlan := modeSectionForMode(interactionMode)
	msg := composePrompt(promptOptions{
		agent:           "coordinator",
		modeSection:     modeSection,
		coordinatorPlan: coordPlan,
	})

	msg = strings.ReplaceAll(msg, "%AGENTS_LIST%", agentsListStr)

	return msg
}
