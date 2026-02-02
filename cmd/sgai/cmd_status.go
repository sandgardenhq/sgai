package main

import (
	"fmt"
	"io/fs"
	"log"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func cmdStatus(args []string) {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		log.Fatalln(err)
	}

	statePath := filepath.Join(absDir, ".sgai", "state.json")
	wfState, err := state.Load(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No workflow state found in", absDir)
			return
		}
		log.Fatalln("failed to load state:", err)
	}

	printStatusSection(wfState)
	printModelStatusesSection(wfState)
	printSequenceSection(wfState)
	printProjectTodosSection(wfState)
	printMessagesSection(wfState)
	printProgressSection(wfState)
}

func printStatusSection(wfState state.Workflow) {
	status := wfState.Status
	if status == "" {
		status = "-"
	}
	currentAgent := wfState.CurrentAgent
	if currentAgent == "" {
		currentAgent = "-"
	}
	task := wfState.Task
	if task == "" {
		task = "-"
	}

	fmt.Printf("Status:        %s\n", status)
	fmt.Printf("Current Agent: %s\n", currentAgent)
	if wfState.CurrentModel != "" {
		fmt.Printf("Current Model: %s\n", wfState.CurrentModel)
	}
	fmt.Printf("Task:          %s\n", task)
}

func printModelStatusesSection(wfState state.Workflow) {
	if len(wfState.ModelStatuses) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("Model Statuses:")
	for modelID, status := range wfState.ModelStatuses {
		symbol := modelStatusSymbol(status)
		shortID := extractModelShortName(modelID)
		fmt.Printf("  %s %-35s %s\n", symbol, shortID, status)
	}
}

func modelStatusSymbol(status string) string {
	switch status {
	case "model-working":
		return "◐"
	case "model-done":
		return "●"
	case "model-error":
		return "✕"
	default:
		return "○"
	}
}

func extractModelShortName(modelID string) string {
	_, modelSpec, found := strings.Cut(modelID, ":")
	if found {
		return modelSpec
	}
	return modelID
}

func printSequenceSection(wfState state.Workflow) {
	if len(wfState.AgentSequence) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("Sequence:")

	now := time.Now().UTC()
	for i, entry := range wfState.AgentSequence {
		startTime, err := time.Parse(time.RFC3339, entry.StartTime)
		if err != nil {
			continue
		}

		var elapsed time.Duration
		switch {
		case entry.IsCurrent:
			elapsed = now.Sub(startTime)
		case i+1 < len(wfState.AgentSequence):
			nextStartTime, err := time.Parse(time.RFC3339, wfState.AgentSequence[i+1].StartTime)
			if err == nil {
				elapsed = nextStartTime.Sub(startTime)
			} else {
				elapsed = now.Sub(startTime)
			}
		default:
			elapsed = now.Sub(startTime)
		}

		elapsedStr := formatDuration(elapsed)
		marker := ""
		if entry.IsCurrent {
			marker = " *"
		}
		fmt.Printf("  %-20s %s%s\n", entry.Agent, elapsedStr, marker)
	}
}

func formatDuration(d time.Duration) string {
	totalSeconds := int(d.Seconds())
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func printProjectTodosSection(wfState state.Workflow) {
	if len(wfState.ProjectTodos) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("Project TODOs:")

	for _, todo := range wfState.ProjectTodos {
		if todo.Status == "completed" || todo.Status == "cancelled" {
			continue
		}
		symbol := todoStatusSymbol(todo.Status)
		fmt.Printf("  %s %s (%s)\n", symbol, todo.Content, todo.Priority)
	}
}

func printMessagesSection(wfState state.Workflow) {
	if len(wfState.Messages) == 0 {
		return
	}

	unreadCount := 0
	for _, msg := range wfState.Messages {
		if !msg.Read {
			unreadCount++
		}
	}

	fmt.Println()
	if unreadCount > 0 {
		fmt.Printf("Messages: %d unread\n", unreadCount)
	} else {
		fmt.Println("Messages:")
	}

	for _, msg := range wfState.Messages {
		if !msg.Read {
			subject := extractMessageSubject(msg.Body)
			fmt.Printf("  %s → %s: %s\n", msg.FromAgent, msg.ToAgent, subject)
		}
	}
}

func extractMessageSubject(body string) string {
	lines := strings.SplitSeq(body, "\n")
	for line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			if len(trimmed) > 50 {
				return trimmed[:47] + "..."
			}
			return trimmed
		}
	}
	return ""
}

func printProgressSection(wfState state.Workflow) {
	if len(wfState.Progress) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("Progress:")

	maxEntries := 10
	startIdx := 0
	if len(wfState.Progress) > maxEntries {
		startIdx = len(wfState.Progress) - maxEntries
	}

	for i := len(wfState.Progress) - 1; i >= startIdx; i-- {
		entry := wfState.Progress[i]
		timeStr := formatProgressTimestamp(entry.Timestamp)
		fmt.Printf("  %s %s: %s\n", timeStr, entry.Agent, entry.Description)
	}
}

func formatProgressTimestamp(timestamp string) string {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return timestamp
	}
	return t.Local().Format("15:04:05")
}

func cmdListAgents(args []string) {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		log.Fatalln(err)
	}

	skelAgentsFS, err := fs.Sub(skelFS, "skel/.sgai/agent")
	if err != nil {
		log.Fatalln("failed to access skeleton agents:", err)
	}

	skelAgents := make(map[string]string)
	err = fs.WalkDir(skelAgentsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		name := strings.TrimSuffix(path, ".md")
		content, err := fs.ReadFile(skelAgentsFS, path)
		if err != nil {
			return nil
		}
		desc := extractFrontmatterDescription(string(content))
		skelAgents[name] = desc
		return nil
	})
	if err != nil {
		log.Fatalln("failed to list skeleton agents:", err)
	}

	dirAgents := make(map[string]string)
	dirAgentsPath := filepath.Join(absDir, ".sgai", "agent")
	if _, err := os.Stat(dirAgentsPath); err == nil {
		entries, err := os.ReadDir(dirAgentsPath)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
					continue
				}
				name := strings.TrimSuffix(entry.Name(), ".md")
				content, err := os.ReadFile(filepath.Join(dirAgentsPath, entry.Name()))
				if err != nil {
					continue
				}
				desc := extractFrontmatterDescription(string(content))
				dirAgents[name] = desc
			}
		}
	}

	fmt.Println("Skeleton agents:")
	skelNames := slices.Sorted(maps.Keys(skelAgents))
	for _, name := range skelNames {
		desc := skelAgents[name]
		if desc != "" {
			fmt.Printf("  %s: %s\n", name, desc)
		} else {
			fmt.Printf("  %s\n", name)
		}
	}

	if len(dirAgents) > 0 {
		fmt.Println("\nDirectory agents (.sgai/agent/):")
		dirNames := slices.Sorted(maps.Keys(dirAgents))
		for _, name := range dirNames {
			desc := dirAgents[name]
			if desc != "" {
				fmt.Printf("  %s: %s\n", name, desc)
			} else {
				fmt.Printf("  %s\n", name)
			}
		}
	}
}
