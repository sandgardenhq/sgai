package main

import (
	"fmt"
	"os"
	"strings"
)

func composeCoordinatorPromptTemplate(currentAgent string) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(promptSectionPreamble)
	sb.WriteString("\n\n")

	switch currentAgent {
	case "coordinator":
		sb.WriteString(promptSectionHumanCommDirect)
	default:
		sb.WriteString(promptSectionHumanCommNonCoordinator)
	}
	sb.WriteString("\n\n")

	sb.WriteString(promptSectionMessaging)
	sb.WriteString("\n\n")

	if currentAgent == "coordinator" {
		sb.WriteString(promptSectionProjectManagementMonitor)
		sb.WriteString("\n\n")
	}

	sb.WriteString(promptSectionWorkFocus)
	sb.WriteString("\n\n")
	sb.WriteString(promptSectionDelegation)
	sb.WriteString("\n")

	switch currentAgent {
	case "coordinator":
		sb.WriteString(promptSectionPostSkillsCoordinator)
	default:
		sb.WriteString(promptSectionPostSkillsNonCoordinator)
	}
	sb.WriteString("\n\n")

	sb.WriteString(promptSectionGuidelines)
	sb.WriteString("\n\n")

	switch currentAgent {
	case "coordinator":
		sb.WriteString(promptSectionTailCoordinator)
		sb.WriteString("\n")
	default:
		sb.WriteString(promptSectionTailNonCoordinator)
		sb.WriteString("\n")
	}

	sb.WriteString(promptSectionCommonTail)
	sb.WriteString("\n")

	switch currentAgent {
	case "coordinator":
		sb.WriteString(promptSectionCoordinatorMessagingTail)
	default:
		sb.WriteString(promptSectionNonCoordinatorMessagingTail)
	}
	sb.WriteString("\n")

	return sb.String()
}

func buildCoordinatorDelegationMessage(agents []string, visitCounts map[string]int, dir string, interactionMode string) string {
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

	var visitLines []string
	for _, agent := range agents {
		count := visitCounts[agent]
		visitLines = append(visitLines, fmt.Sprintf("  %s: %d visits", agent, count))
	}
	visitCountsStr := strings.Join(visitLines, "\n")

	modeSection, coordPlan := modeSectionForMode(interactionMode)
	msg := composePrompt(promptOptions{
		agent:           "coordinator",
		modeSection:     modeSection,
		coordinatorPlan: coordPlan,
	})

	msg = strings.ReplaceAll(msg, "%AGENTS_LIST%", agentsListStr)
	msg = strings.ReplaceAll(msg, "%VISIT_COUNTS%", visitCountsStr)

	return msg
}
