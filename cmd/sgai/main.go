package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"maps"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sandgardenhq/sgai/pkg/notify"
	"github.com/sandgardenhq/sgai/pkg/state"
	"golang.org/x/term"
	"sigs.k8s.io/yaml"
)

const workGateApprovalText = "DEFINITION IS COMPLETE, BUILD MAY BEGIN"

var modelVariantPattern = regexp.MustCompile(`^(.+?)\s*\(([^)]+)\)$`)

func parseModelAndVariant(modelSpec string) (model, variant string) {
	matches := modelVariantPattern.FindStringSubmatch(modelSpec)
	if len(matches) == 3 {
		return matches[1], matches[2]
	}
	return modelSpec, ""
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	if os.Args[1] != "help" && os.Args[1] != "-h" && os.Args[1] != "--help" {
		if _, err := exec.LookPath("opencode"); err != nil {
			log.Fatalln("opencode is required but not found in PATH")
		}
	}

	switch os.Args[1] {
	case "serve":
		cmdServe(os.Args[2:])
		return
	case "sessions":
		cmdSessions(os.Args[2:])
		return
	case "retrospective":
		cmdRetrospective(os.Args[2:])
		return
	case "list-agents":
		cmdListAgents(os.Args[2:])
		return
	case "status":
		cmdStatus(os.Args[2:])
		return
	case "mcp":
		cmdMCP(os.Args[2:])
		return
	case "help", "-h", "--help":
		printUsage()
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runWorkflow(ctx, os.Args[1:])
}

func printUsage() {
	fmt.Println(`sgai - AI-powered software factory

Usage:
  sgai [--fresh] <target_directory>   Run workflow for GOAL.md
  sgai serve [--listen-addr addr]           Start web server for session management
  sgai sessions                             List all sessions in .sgai/retrospectives
  sgai status [target_directory]            Show workflow status summary
  sgai retrospective analyze [session-id]   Analyze a session (default: most recent)
  sgai retrospective apply <session-id>     Apply improvements from a session
  sgai list-agents [target_directory]       List available agents

Options:
  --fresh         Force a fresh start (don't resume existing workflow)

Serve Options:
  --listen-addr   HTTP server listen address (default: 127.0.0.1:8080)

Examples:
  sgai .
      Run workflow in current directory
  sgai --fresh .
      Start fresh, don't resume existing workflow
  sgai serve
      Start web UI on localhost:8080
  sgai serve --listen-addr 0.0.0.0:8080
      Start web UI accessible externally
  sgai sessions
      List all sessions with GOAL summary
  sgai status
      Show workflow status for current directory
  sgai status ./my-project
      Show workflow status for specific directory
  sgai retrospective analyze
      Analyze the most recent session
  sgai retrospective analyze 2025-12-30-09-33.3db5
      Analyze specific session
  sgai retrospective apply 2025-12-30-09-33.3db5
      Apply improvements from session
  sgai list-agents
      List all available agents`)
}

// runWorkflow executes the main workflow loop for a target directory.
// It handles flow mode workflows, agent iteration, and human interaction.
func runWorkflow(ctx context.Context, args []string) {
	flagSet := flag.NewFlagSet("sgai", flag.ExitOnError)
	freshFlag := flagSet.Bool("fresh", false, "force a fresh start (delete state.json and PROJECT_MANAGEMENT.md)")
	flagSet.Parse(args) //nolint:errcheck // ExitOnError FlagSet exits on error, never returns non-nil

	if flagSet.NArg() < 1 {
		log.Fatalln("usage: sgai [--fresh] <target_directory>")
	}

	dir, err := filepath.Abs(flagSet.Arg(0))
	if err != nil {
		log.Fatalln(err)
	}

	goalPath := filepath.Join(dir, "GOAL.md")
	goalContent, err := os.ReadFile(goalPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalln("GOAL.md not found in", dir)
		}
		log.Fatalln(err)
	}

	metadata, err := parseYAMLFrontmatter(goalContent)
	if err != nil {
		log.Fatalln("failed to parse GOAL.md frontmatter:", err)
	}

	projectConfig, err := loadProjectConfig(dir)
	if err != nil {
		log.Fatalln("failed to load sgai.json:", err)
	}

	if err := validateProjectConfig(projectConfig); err != nil {
		log.Fatalln(err)
	}

	applyConfigDefaults(projectConfig, &metadata)

	if err := initializeWorkspaceDir(dir); err != nil {
		log.Fatalln("failed to initialize workspace directory:", err)
	}

	if err := applyCustomMCPs(dir, projectConfig); err != nil {
		log.Fatalln("failed to apply custom MCPs:", err)
	}

	if err := os.Chdir(dir); err != nil {
		log.Fatalln("failed to change directory to", dir, err)
	}

	executablePath, err := os.Executable()
	if err != nil {
		log.Fatalln("failed to get executable path:", err)
	}

	flowDag, err := parseFlow(metadata.Flow, dir)
	if err != nil {
		log.Fatalln("failed to parse flow:", err)
	}

	ensureImplicitProjectCriticCouncilModel(flowDag, &metadata)

	if err := validateModels(metadata.Models); err != nil {
		log.Fatalln(err)
	}

	stateJSONPath := filepath.Join(dir, ".sgai", "state.json")
	wfState, err := state.Load(stateJSONPath)
	if err != nil && !os.IsNotExist(err) {
		log.Fatalln("failed to read state.json:", err)
	}
	newChecksum, err := computeGoalChecksum(goalPath)
	if err != nil {
		log.Fatalln("failed to compute GOAL.md checksum:", err)
	}

	dagAgents := flowDag.allAgents()
	var allAgents []string
	if slices.Contains(dagAgents, "coordinator") {
		allAgents = dagAgents
	} else {
		allAgents = append([]string{"coordinator"}, dagAgents...)
	}
	longestNameLen := len("sgai")
	for _, agent := range allAgents {
		longestNameLen = max(longestNameLen, len(agent))
	}
	paddedsgai := "sgai" + strings.Repeat(" ", max(0, longestNameLen-len("sgai")))

	pmPath := filepath.Join(dir, ".sgai", "PROJECT_MANAGEMENT.md")
	retrospectivesBaseDir := filepath.Join(dir, ".sgai", "retrospectives")

	resuming := canResumeWorkflow(wfState, *freshFlag, newChecksum)

	disableRetrospective := projectConfig != nil && projectConfig.DisableRetrospective

	var retrospectiveDir string
	var retrospectiveDirRel string

	switch {
	case disableRetrospective:
		if !resuming {
			if err := os.Remove(stateJSONPath); err != nil && !os.IsNotExist(err) {
				log.Fatalln("failed to truncate state.json on startup:", err)
			}
			if err := os.Remove(pmPath); err != nil && !os.IsNotExist(err) {
				log.Fatalln("failed to truncate PROJECT_MANAGEMENT.md on startup:", err)
			}
		}
	case resuming:
		fmt.Println("["+paddedsgai+"]", "resuming existing workflow...")
		retrospectiveDirRel = extractRetrospectiveDirFromProjectManagement(pmPath)
		if retrospectiveDirRel == "" {
			log.Fatalln("failed to read retrospective directory from PROJECT_MANAGEMENT.md during resume")
		}
		retrospectiveDir = filepath.Join(dir, retrospectiveDirRel)
		if _, err := os.Stat(retrospectiveDir); os.IsNotExist(err) {
			log.Fatalln("retrospective directory from PROJECT_MANAGEMENT.md does not exist:", retrospectiveDir)
		}

		if err := setupOutputCapture(retrospectiveDir); err != nil {
			log.Fatalln("failed to setup output capture:", err)
		}
	default:
		retrospectiveDir = filepath.Join(retrospectivesBaseDir, generateRetrospectiveDirName())
		if err := os.MkdirAll(retrospectiveDir, 0755); err != nil {
			log.Fatalln("failed to create retrospective directory:", err)
		}

		retrospectiveDirRel, err = filepath.Rel(dir, retrospectiveDir)
		if err != nil {
			log.Fatalln("failed to compute relative retrospective directory path:", err)
		}

		if err := os.Remove(stateJSONPath); err != nil && !os.IsNotExist(err) {
			log.Fatalln("failed to truncate state.json on startup:", err)
		}
		if err := os.Remove(pmPath); err != nil && !os.IsNotExist(err) {
			log.Fatalln("failed to truncate PROJECT_MANAGEMENT.md on startup:", err)
		}

		if err := updateProjectManagementWithRetrospectiveDir(pmPath, retrospectiveDirRel); err != nil {
			log.Fatalln("failed to update PROJECT_MANAGEMENT.md with retrospective directory:", err)
		}

		goalRetrospectivePath := filepath.Join(retrospectiveDir, "GOAL.md")
		if err := copyToRetrospective(goalPath, goalRetrospectivePath); err != nil {
			log.Fatalln("failed to copy GOAL.md to retrospective:", err)
		}

		if err := setupOutputCapture(retrospectiveDir); err != nil {
			log.Fatalln("failed to setup output capture:", err)
		}
	}

	defer func() {
		if retrospectiveDir != "" {
			if err := copyFinalStateToRetrospective(dir, retrospectiveDir); err != nil {
				log.Printf("[sgai] warning: failed to copy final state: %v", err)
			}
		}
	}()

	if !resuming {
		wfState = state.Workflow{
			Status:       state.StatusWorking,
			Messages:     []state.Message{},
			GoalChecksum: newChecksum,
			VisitCounts:  initVisitCounts(allAgents),
		}
		if err := state.Save(stateJSONPath, wfState); err != nil {
			log.Println("failed to initialize state.json:", err)
			return
		}
	}

	var iterationCounter int
	var previousAgent string

	defer func() {
		notifyMsg := fmt.Sprintf("JOB COMPLETE - %s", filepath.Base(dir))
		notify.Send("sgai", notifyMsg)
	}()
	for {
		if ctx.Err() != nil {
			fmt.Println("["+paddedsgai+"]", "interrupted, stopping workflow...")
			return
		}

		currentAgent := "coordinator"
		if wfState.CurrentAgent != "" && wfState.CurrentAgent != "coordinator" {
			currentAgent = wfState.CurrentAgent
		}
		wfState.CurrentAgent = currentAgent
		wfState.VisitCounts[currentAgent]++
		addAgentHandoffProgress(&wfState, currentAgent)
		markCurrentAgentInSequence(&wfState, currentAgent)
		if err := state.Save(stateJSONPath, wfState); err != nil {
			log.Println("failed to save state:", err)
			return
		}

		if previousAgent != "" && previousAgent != currentAgent {
			fmt.Println("["+paddedsgai+"]", previousAgent, "->", currentAgent)
			wfState.Todos = []state.TodoItem{}
		}
		previousAgent = currentAgent

		var errReloadGoalMetadata error
		metadata, errReloadGoalMetadata = tryReloadGoalMetadata(goalPath, metadata)
		if errReloadGoalMetadata != nil {
			log.Println("failed to reload GOAL.md frontmatter:", errReloadGoalMetadata)
			return
		}
		wfState = runFlowAgent(ctx, dir, goalPath, currentAgent, flowDag, wfState, stateJSONPath, metadata, retrospectiveDir, longestNameLen, paddedsgai, &iterationCounter, executablePath)
		if applyWorkGateApproval(&wfState, stateJSONPath, paddedsgai) {
			return
		}
		if wfState.Status == state.StatusComplete {
			if redirectToPendingMessageAgent(&wfState, stateJSONPath, paddedsgai) {
				continue
			}
			fmt.Println("["+paddedsgai+"]", "complete:", wfState.Task)
			return
		}

		if hasPendingMessages(&wfState, stateJSONPath, paddedsgai) {
			continue
		}

		if len(flowDag.EntryNodes) > 0 {
			currentAgent = flowDag.EntryNodes[0]
		} else {
			log.Println("no entry nodes in flow DAG")
			return
		}

		for currentAgent != "" {
			if ctx.Err() != nil {
				fmt.Println("["+paddedsgai+"]", "interrupted, stopping workflow...")
				return
			}

			if previousAgent != "" && previousAgent != currentAgent {
				fmt.Println("["+paddedsgai+"]", previousAgent, "->", currentAgent)
				wfState.Todos = []state.TodoItem{}
			}
			previousAgent = currentAgent

			wfState.CurrentAgent = currentAgent
			wfState.VisitCounts[currentAgent]++
			addAgentHandoffProgress(&wfState, currentAgent)
			markCurrentAgentInSequence(&wfState, currentAgent)
			if err := state.Save(stateJSONPath, wfState); err != nil {
				log.Println("failed to save state:", err)
				return
			}

			var errReloadGoalMetadata error
			metadata, errReloadGoalMetadata = tryReloadGoalMetadata(goalPath, metadata)
			if errReloadGoalMetadata != nil {
				log.Println("failed to reload GOAL.md frontmatter:", errReloadGoalMetadata)
				return
			}
			wfState = runFlowAgent(ctx, dir, goalPath, currentAgent, flowDag, wfState, stateJSONPath, metadata, retrospectiveDir, longestNameLen, paddedsgai, &iterationCounter, executablePath)
			if applyWorkGateApproval(&wfState, stateJSONPath, paddedsgai) {
				return
			}

			if wfState.Status == state.StatusComplete {
				if redirectToPendingMessageAgent(&wfState, stateJSONPath, paddedsgai) {
					currentAgent = wfState.CurrentAgent
					continue
				}
				fmt.Println("["+paddedsgai+"]", "complete:", wfState.Task)
				return
			}

			if hasPendingMessages(&wfState, stateJSONPath, paddedsgai) {
				currentAgent = wfState.CurrentAgent
				continue
			}

			if flowDag.isTerminal(currentAgent) {
				break
			}

			currentAgent = determineNextAgent(flowDag, currentAgent)
		}

		if flowDag.isTerminal(currentAgent) {
			fmt.Println("["+paddedsgai+"]", "reached terminal node", currentAgent)
			redirectToCoordinator(&wfState, stateJSONPath, paddedsgai)
			continue
		}
	}
}

func applyWorkGateApproval(wfState *state.Workflow, stateJSONPath, paddedsgai string) bool {
	if !wfState.WorkGateApproved {
		return false
	}
	wfState.InteractiveAutoLock = true
	wfState.WorkGateApproved = false
	if err := state.Save(stateJSONPath, *wfState); err != nil {
		log.Println("failed to save state:", err)
		return true
	}
	fmt.Println("["+paddedsgai+"]", "work gate approved, switching to auto mode")
	return false
}

func ensureImplicitProjectCriticCouncilModel(flowDag *dag, metadata *GoalMetadata) {
	if metadata.Models == nil {
		metadata.Models = make(map[string]any)
	}
	_, existsInDag := flowDag.Nodes["project-critic-council"]
	if !existsInDag {
		return
	}
	_, existsInModels := metadata.Models["project-critic-council"]
	if existsInModels {
		return
	}
	coordinatorModel, hasCoordinator := metadata.Models["coordinator"]
	if !hasCoordinator {
		return
	}
	metadata.Models["project-critic-council"] = coordinatorModel
}

func selectModelForAgent(metadataModels map[string]any, agent string) string {
	models := getModelsForAgent(metadataModels, agent)
	if len(models) > 0 {
		return models[0]
	}
	return ""
}

func getModelsForAgent(models map[string]any, agent string) []string {
	val, exists := models[agent]
	if !exists {
		return nil
	}

	switch v := val.(type) {
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

func formatModelID(agent, modelSpec string) string {
	return agent + ":" + modelSpec
}

func extractAgentFromModelID(modelID string) string {
	agent, _, found := strings.Cut(modelID, ":")
	if found {
		return agent
	}
	return modelID
}

func allModelsDone(modelStatuses map[string]string) bool {
	if len(modelStatuses) == 0 {
		return true
	}
	for _, status := range modelStatuses {
		if status != "model-done" && status != "model-error" {
			return false
		}
	}
	return true
}

func hasMessagesForModel(messages []state.Message, modelID string) bool {
	agentName := extractAgentFromModelID(modelID)
	for _, msg := range messages {
		if msg.Read {
			continue
		}
		if msg.ToAgent == modelID || msg.ToAgent == agentName {
			return true
		}
	}
	return false
}

func hasPendingMessagesForAnyModel(messages []state.Message, models []string, agent string) bool {
	for _, modelSpec := range models {
		modelID := formatModelID(agent, modelSpec)
		if hasMessagesForModel(messages, modelID) {
			return true
		}
	}
	return false
}

func syncModelStatuses(modelStatuses map[string]string, models []string, agent string) map[string]string {
	if modelStatuses == nil {
		modelStatuses = make(map[string]string)
	}

	currentModelSet := make(map[string]bool)
	for _, modelSpec := range models {
		modelID := formatModelID(agent, modelSpec)
		currentModelSet[modelID] = true
		if _, exists := modelStatuses[modelID]; !exists {
			modelStatuses[modelID] = "model-working"
		}
	}

	for modelID := range modelStatuses {
		if extractAgentFromModelID(modelID) != agent {
			continue
		}
		if !currentModelSet[modelID] {
			delete(modelStatuses, modelID)
		}
	}

	return modelStatuses
}

func cleanupModelStatuses(wfState *state.Workflow) {
	wfState.ModelStatuses = nil
	wfState.CurrentModel = ""
}

type multiModelConfig struct {
	dir              string
	goalPath         string
	agent            string
	flowDag          *dag
	statePath        string
	retrospectiveDir string
	longestNameLen   int
	paddedsgai       string
	executablePath   string
}

func runMultiModelAgent(ctx context.Context, cfg multiModelConfig, wfState state.Workflow, metadata GoalMetadata, iterationCounter *int) state.Workflow {
	models := getModelsForAgent(metadata.Models, cfg.agent)
	if len(models) <= 1 {
		return runSingleModelIteration(ctx, cfg, wfState, metadata, iterationCounter, models)
	}

	wfState.ModelStatuses = syncModelStatuses(wfState.ModelStatuses, models, cfg.agent)
	if err := state.Save(cfg.statePath, wfState); err != nil {
		log.Fatalln("failed to save state before multi-model loop:", err)
	}

	for {
		if ctx.Err() != nil {
			fmt.Println("["+cfg.paddedsgai+"]", "interrupted, stopping multi-model agent...")
			return wfState
		}

		var errReloadGoalMetadata error
		metadata, errReloadGoalMetadata = tryReloadGoalMetadata(cfg.goalPath, metadata)
		if errReloadGoalMetadata != nil {
			log.Fatalln("failed to reload GOAL.md frontmatter:", errReloadGoalMetadata)
		}
		newModels := getModelsForAgent(metadata.Models, cfg.agent)

		if len(newModels) <= 1 {
			fmt.Println("["+cfg.paddedsgai+"]", "switching to single-model mode for", cfg.agent)
			cleanupModelStatuses(&wfState)
			return runSingleModelIteration(ctx, cfg, wfState, metadata, iterationCounter, newModels)
		}

		wfState.ModelStatuses = syncModelStatuses(wfState.ModelStatuses, newModels, cfg.agent)
		models = newModels

		for _, modelSpec := range models {
			if ctx.Err() != nil {
				return wfState
			}

			modelID := formatModelID(cfg.agent, modelSpec)

			currentStatus := wfState.ModelStatuses[modelID]
			hasMessages := hasMessagesForModel(wfState.Messages, modelID)

			if currentStatus == "model-done" && hasMessages {
				wfState.ModelStatuses[modelID] = "model-working"
				currentStatus = "model-working"
				fmt.Println("["+cfg.paddedsgai+"]", "reverting", modelID, "to model-working due to pending messages")
			}

			if currentStatus == "model-done" || currentStatus == "model-error" {
				continue
			}

			wfState.CurrentModel = modelID
			if err := state.Save(cfg.statePath, wfState); err != nil {
				log.Fatalln("failed to save state before model iteration:", err)
			}

			fmt.Println("["+cfg.paddedsgai+"]", "running model:", modelID)
			wfState = runSingleModelIteration(ctx, cfg, wfState, metadata, iterationCounter, []string{modelSpec})

			newState, err := state.Load(cfg.statePath)
			if err != nil {
				log.Fatalln("failed to load state after model iteration:", err)
			}

			switch newState.Status {
			case state.StatusAgentDone:
				wfState.ModelStatuses[modelID] = "model-done"
				wfState.Status = state.StatusWorking
				if err := state.Save(cfg.statePath, wfState); err != nil {
					log.Fatalln("failed to save state after model done:", err)
				}
			case state.StatusComplete, state.StatusWaitingForHuman:
				return newState
			}
		}

		if allModelsDone(wfState.ModelStatuses) && !hasPendingMessagesForAnyModel(wfState.Messages, models, cfg.agent) {
			fmt.Println("["+cfg.paddedsgai+"]", "multi-model consensus reached for", cfg.agent)
			cleanupModelStatuses(&wfState)
			wfState.Status = state.StatusAgentDone
			if err := state.Save(cfg.statePath, wfState); err != nil {
				log.Fatalln("failed to save state after consensus:", err)
			}
			return wfState
		}
	}
}

func runSingleModelIteration(ctx context.Context, cfg multiModelConfig, wfState state.Workflow, metadata GoalMetadata, iterationCounter *int, models []string) state.Workflow {
	modelSpec := ""
	if len(models) > 0 {
		modelSpec = models[0]
	}
	return runFlowAgentWithModel(ctx, cfg, wfState, metadata, iterationCounter, modelSpec)
}

func runFlowAgentWithModel(ctx context.Context, cfg multiModelConfig, wfState state.Workflow, metadata GoalMetadata, iterationCounter *int, modelSpec string) state.Workflow {
	paddedAgentName := cfg.agent + strings.Repeat(" ", max(0, cfg.longestNameLen-len(cfg.agent)))
	var humanResponse string
	var capturedSessionID string
	outputCapture := newRingWriter()

	for {
		if ctx.Err() != nil {
			fmt.Println("["+cfg.paddedsgai+"]", "interrupted, stopping agent...")
			return wfState
		}

		*iterationCounter++

		prefix := fmt.Sprintf("[%s:%04d]", paddedAgentName, *iterationCounter)

		if err := state.Save(cfg.statePath, wfState); err != nil {
			log.Fatalln("failed to save state before running agent:", err)
		}

		if cfg.retrospectiveDir != "" {
			pmPath := filepath.Join(cfg.dir, ".sgai", "PROJECT_MANAGEMENT.md")
			if _, errStat := os.Stat(pmPath); errStat == nil {
				pmRetrospectivePath := filepath.Join(cfg.retrospectiveDir, "PROJECT_MANAGEMENT.md")
				if err := copyToRetrospective(pmPath, pmRetrospectivePath); err != nil {
					log.Fatalln("failed to copy PROJECT_MANAGEMENT.md to retrospective:", err)
				}
			}
		}

		args := []string{"run", "--format=json", "--agent", cfg.agent}
		if modelSpec != "" {
			model, variant := parseModelAndVariant(modelSpec)
			args = append(args, "--model", model)
			if variant != "" {
				args = append(args, "--variant", variant)
			}
		}
		if capturedSessionID != "" {
			args = append(args, "--session", capturedSessionID)
		}
		title := cfg.agent
		if modelSpec != "" {
			title = cfg.agent + " [" + modelSpec + "]"
		}
		args = append(args, "--title", title)

		var msg string
		if humanResponse != "" {
			msg = humanResponse
			humanResponse = ""
		} else {
			msg = buildFlowMessage(cfg.flowDag, cfg.agent, wfState.VisitCounts, cfg.dir, wfState.InteractiveAutoLock)

			multiModelSection := buildMultiModelSection(wfState.CurrentModel, metadata.Models, cfg.agent)
			if multiModelSection != "" {
				msg += multiModelSection
			}

			pendingCount := 0
			for _, m := range wfState.Messages {
				if m.ToAgent == cfg.agent {
					pendingCount++
				}
			}
			if pendingCount > 0 {
				msgNotification := fmt.Sprintf("\nYOU HAVE %d PENDING MESSAGE(S). YOU MUST CALL `sgai_check_inbox()` TO READ THEM.\n", pendingCount)
				msg = msgNotification + msg
			}

			pendingTodosCount := countPendingTodos(wfState, cfg.agent)
			if pendingTodosCount > 0 {
				todoNudge := fmt.Sprintf("\nYou have %d pending TODO items. Please complete them before marking agent-done.\n", pendingTodosCount)
				msg += todoNudge
			}

			if cfg.agent != "coordinator" {
				outboxPending := 0
				for _, m := range wfState.Messages {
					if m.FromAgent == cfg.agent && !m.Read {
						outboxPending++
					}
				}
				if outboxPending > 0 {
					msg += "\nYou have sent messages that haven't been read yet. For the recipient agents to process them, you MUST yield control by calling sgai_update_workflow_state({status: 'agent-done'}). They cannot run while you hold control.\n"
				}
			}
		}

		interactiveEnv := "yes"
		if wfState.InteractiveAutoLock {
			interactiveEnv = "auto"
		}
		cmd := exec.CommandContext(ctx, "opencode", args...)
		cmd.Env = append(os.Environ(),
			"OPENCODE_CONFIG_DIR=.sgai",
			"SGAI_MCP_EXECUTABLE="+cfg.executablePath,
			"SGAI_MCP_INTERACTIVE="+interactiveEnv)
		cmd.Stdin = strings.NewReader(msg)

		stderrWriter := &prefixWriter{prefix: prefix + " ", w: os.Stderr}
		cmd.Stderr = io.MultiWriter(stderrWriter, outputCapture)

		jsonWriter := &jsonPrettyWriter{prefix: prefix + " ", w: os.Stdout, statePath: cfg.statePath, currentAgent: cfg.agent}
		cmd.Stdout = io.MultiWriter(jsonWriter, outputCapture)

		if err := cmd.Run(); err != nil {
			if ctx.Err() != nil {
				fmt.Println("["+cfg.paddedsgai+"]", "interrupted during agent execution")
				return wfState
			}
			fmt.Fprintln(os.Stderr, "\n=== RAW AGENT OUTPUT (last 1000 lines) ===")
			outputCapture.dump(os.Stderr)
			fmt.Fprintln(os.Stderr, "=== END RAW AGENT OUTPUT ===")
			newState, err := state.Load(cfg.statePath)
			if err != nil {
				log.Fatalln("failed to read state.json:", err)
			}
			newState.Status = state.StatusAgentDone
			if err := state.Save(cfg.statePath, newState); err != nil {
				log.Fatalln("failed to save state:", err)
			}
			fmt.Fprintln(os.Stderr, "agent", cfg.agent, "marked as agent-done due to error")
			return newState
		}
		jsonWriter.Flush()
		capturedSessionID = jsonWriter.sessionID

		newState, err := state.Load(cfg.statePath)
		if err != nil {
			log.Fatalln("failed to read state.json:", err)
		}
		if newState.VisitCounts == nil {
			newState.VisitCounts = make(map[string]int)
		}

		switch newState.Status {
		case "complete":
			if cfg.agent == "coordinator" {
				count := 0
				for _, todo := range wfState.Todos {
					if todo.Status != "completed" && todo.Status != "cancelled" {
						count++
					}
				}
				if count > 0 {
					fmt.Println("["+cfg.paddedsgai+"]", "coordinator cannot complete workflow, there are pending TODO items")
					newState.Status = state.StatusWorking
					addEnvironmentMessage(&newState, cfg.agent, fmt.Sprintf("# Pending TODO items.\nYou have %d pending TODO items. Please complete them before marking workflow complete.\n", count))
					if err := state.Save(cfg.statePath, newState); err != nil {
						log.Fatalln("failed to save state:", err)
					}
					return newState
				}

				if metadata.CompletionGateScript != "" {
					fmt.Println("["+cfg.paddedsgai+"]", "running completionGateScript:", metadata.CompletionGateScript)
					output, errScript := runCompletionGateScript(metadata.CompletionGateScript)
					if errScript != nil {
						fmt.Println("["+cfg.paddedsgai+"]", "completionGateScript failed, blocking completion")
						newState.Status = state.StatusWorking
						addEnvironmentMessage(&newState, cfg.agent, formatCompletionGateScriptFailureMessage(metadata.CompletionGateScript, output))
						if err := state.Save(cfg.statePath, newState); err != nil {
							log.Fatalln("failed to save state:", err)
						}
						return newState
					}
				}
			}

			if cfg.retrospectiveDir != "" {
				goalRetrospectivePath := filepath.Join(cfg.retrospectiveDir, "GOAL.md")
				if err := copyToRetrospective(cfg.goalPath, goalRetrospectivePath); err != nil {
					log.Fatalln("failed to copy GOAL.md to retrospective:", err)
				}

				pmPath := filepath.Join(cfg.dir, ".sgai", "PROJECT_MANAGEMENT.md")
				if _, errStat := os.Stat(pmPath); errStat == nil {
					pmRetrospectivePath := filepath.Join(cfg.retrospectiveDir, "PROJECT_MANAGEMENT.md")
					if err := copyToRetrospective(pmPath, pmRetrospectivePath); err != nil {
						log.Fatalln("failed to copy PROJECT_MANAGEMENT.md to retrospective:", err)
					}
				}

				if capturedSessionID != "" && shouldLogAgent(cfg.dir, cfg.agent) {
					timestamp := time.Now().Format("20060102150405")
					sessionFile := filepath.Join(cfg.retrospectiveDir, fmt.Sprintf("%04d-%s-%s.json", *iterationCounter, cfg.agent, timestamp))
					if err := exportSession(capturedSessionID, sessionFile); err != nil {
						log.Fatalln("failed to export session:", err)
					}
				}
			}
			return newState

		case state.StatusWaitingForHuman:
			if err := state.Save(cfg.statePath, newState); err != nil {
				log.Fatalln("failed to save state:", err)
			}

			notifyMsg := fmt.Sprintf("QUESTION - %s", filepath.Base(cfg.dir))
			notify.Send("sgai", notifyMsg)

			if !newState.InteractiveAutoLock {
				hasMultiChoice := newState.MultiChoiceQuestion != nil && len(newState.MultiChoiceQuestion.Questions) > 0
				isTTY := term.IsTerminal(int(os.Stdin.Fd()))
				switch {
				case hasMultiChoice && isTTY:
					fmt.Println("["+cfg.paddedsgai+"]", "multi-choice question requested...")
					handleMultiChoiceQuestion(cfg.dir, cfg.statePath, newState.MultiChoiceQuestion)
				case isTTY:
					fmt.Println("["+cfg.paddedsgai+"]", "human input requested, opening editor...")
					launchEditorForResponse(cfg.dir, newState.HumanMessage, cfg.statePath)
				default:
					fmt.Println("["+cfg.paddedsgai+"]", "waiting for response...")
				}
			}

			humanResponse = waitForStateTransition(cfg.dir, cfg.statePath)
			if newState.MultiChoiceQuestion != nil && newState.MultiChoiceQuestion.IsWorkGate && strings.Contains(humanResponse, workGateApprovalText) {
				loadedState, errLoad := state.Load(cfg.statePath)
				if errLoad != nil {
					log.Println("failed to load state for work gate approval:", errLoad)
				} else {
					loadedState.WorkGateApproved = true
					if errSave := state.Save(cfg.statePath, loadedState); errSave != nil {
						log.Fatalln("failed to save work gate approval:", errSave)
					}
				}
			}
			wfState = newState
			wfState.Status = state.StatusWorking
			continue

		case state.StatusAgentDone:
			if cfg.retrospectiveDir != "" && capturedSessionID != "" && shouldLogAgent(cfg.dir, cfg.agent) {
				timestamp := time.Now().Format("20060102150405")
				sessionFile := filepath.Join(cfg.retrospectiveDir, fmt.Sprintf("%04d-%s-%s.json", *iterationCounter, cfg.agent, timestamp))
				if err := exportSession(capturedSessionID, sessionFile); err != nil {
					log.Fatalln("failed to export session:", err)
				}
			}
			if err := state.Save(cfg.statePath, newState); err != nil {
				log.Fatalln("failed to save state:", err)
			}
			fmt.Println("["+cfg.paddedsgai+"]", "agent", cfg.agent, "done:", newState.Task)
			return newState

		case state.StatusWorking:
			if err := state.Save(cfg.statePath, newState); err != nil {
				log.Fatalln("failed to save state:", err)
			}

			if agentHasUnreadOutgoingMessages(newState, cfg.agent) {
				fmt.Println("["+cfg.paddedsgai+"]", "agent", cfg.agent, "sent message(s), yielding control...")
				return newState
			}

			fmt.Println("["+cfg.paddedsgai+"]", "agent", cfg.agent, "still working, re-running...")
			wfState = newState
			continue

		default:
			log.Fatalln("["+cfg.paddedsgai+"]", "unexpected status:", newState.Status)
		}
	}
}

func runFlowAgent(ctx context.Context, dir, goalPath, agent string, flowDag *dag, wfState state.Workflow, statePath string, metadata GoalMetadata, retrospectiveDir string, longestNameLen int, paddedsgai string, iterationCounter *int, executablePath string) state.Workflow {
	cfg := multiModelConfig{
		dir:              dir,
		goalPath:         goalPath,
		agent:            agent,
		flowDag:          flowDag,
		statePath:        statePath,
		retrospectiveDir: retrospectiveDir,
		longestNameLen:   longestNameLen,
		paddedsgai:       paddedsgai,
		executablePath:   executablePath,
	}
	return runMultiModelAgent(ctx, cfg, wfState, metadata, iterationCounter)
}

func agentHasUnreadOutgoingMessages(s state.Workflow, agentName string) bool {
	for _, msg := range s.Messages {
		if msg.FromAgent == agentName && !msg.Read {
			return true
		}
	}
	return false
}

func nextMessageID(messages []state.Message) int {
	nextID := 1
	for _, msg := range messages {
		if msg.ID >= nextID {
			nextID = msg.ID + 1
		}
	}
	return nextID
}

func addEnvironmentMessage(wfState *state.Workflow, toAgent, body string) {
	wfState.Messages = append(wfState.Messages, state.Message{
		ID:        nextMessageID(wfState.Messages),
		FromAgent: "environment",
		ToAgent:   toAgent,
		Body:      body,
		Read:      false,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

func addAgentHandoffProgress(wfState *state.Workflow, targetAgent string) {
	progressEntry := state.ProgressEntry{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Agent:       "sgai",
		Description: fmt.Sprintf("Handing off to %s", targetAgent),
	}
	wfState.Progress = append(wfState.Progress, progressEntry)
}

// markCurrentAgentInSequence updates the agent sequence to track the current agent
// with a timestamp. If the current agent is already the last entry, it just marks
// it as current; otherwise, it appends a new entry.
func markCurrentAgentInSequence(s *state.Workflow, currentAgent string) {
	now := time.Now().UTC().Format(time.RFC3339)
	lastIdx := len(s.AgentSequence) - 1
	if lastIdx >= 0 && s.AgentSequence[lastIdx].Agent == currentAgent {
		s.AgentSequence[lastIdx].IsCurrent = true
		return
	}
	for i := range s.AgentSequence {
		s.AgentSequence[i].IsCurrent = false
	}
	s.AgentSequence = append(s.AgentSequence, state.AgentSequenceEntry{
		Agent:     currentAgent,
		StartTime: now,
		IsCurrent: true,
	})
}

// countPendingTodos returns the count of non-completed, non-cancelled TODO items.
// For coordinator, it checks ProjectTodos; for other agents, it checks Todos.
func countPendingTodos(wfState state.Workflow, agent string) int {
	if agent == "coordinator" {
		return 0
	}
	count := 0
	for _, todo := range wfState.Todos {
		if todo.Status != "completed" && todo.Status != "cancelled" {
			count++
		}
	}
	return count
}

// GoalMetadata represents the YAML frontmatter in GOAL.md files.
// It configures workflow flow, per-agent models, and completion gate command.
// Models can be either a single string or an array of strings per agent
// (for multi-model support).
type GoalMetadata struct {
	Flow                 string         `json:"flow,omitempty" yaml:"flow,omitempty"`
	Models               map[string]any `json:"models,omitempty" yaml:"models,omitempty"`
	CompletionGateScript string         `json:"completionGateScript,omitempty" yaml:"completionGateScript,omitempty"`
}

type agentMetadata struct {
	Log      bool     `json:"log" yaml:"log"`
	Snippets []string `json:"snippets" yaml:"snippets"`
}

func shouldLogAgent(dir, agentName string) bool {
	agentPath := filepath.Join(dir, ".sgai", "agent", agentName+".md")
	content, err := os.ReadFile(agentPath)
	if err != nil {
		return true
	}

	delimiter := []byte("---")
	if !bytes.HasPrefix(content, delimiter) {
		return true
	}

	rest := content[len(delimiter):]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	before, _, ok := bytes.Cut(rest, delimiter)
	if !ok {
		return true
	}

	yamlContent := before
	var metadata agentMetadata
	metadata.Log = true
	if err := yaml.Unmarshal(yamlContent, &metadata); err != nil {
		return true
	}

	return metadata.Log
}

func parseAgentSnippets(dir, agentName string) []string {
	agentPath := filepath.Join(dir, ".sgai", "agent", agentName+".md")
	content, err := os.ReadFile(agentPath)
	if err != nil {
		return nil
	}

	delimiter := []byte("---")
	if !bytes.HasPrefix(content, delimiter) {
		return nil
	}

	rest := content[len(delimiter):]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	before, _, ok := bytes.Cut(rest, delimiter)
	if !ok {
		return nil
	}

	yamlContent := before
	var metadata agentMetadata
	if err := yaml.Unmarshal(yamlContent, &metadata); err != nil {
		return nil
	}

	return metadata.Snippets
}

func parseFrontmatterMap(content []byte) map[string]string {
	result := make(map[string]string)
	delimiter := []byte("---")

	if !bytes.HasPrefix(content, delimiter) {
		return result
	}

	rest := content[len(delimiter):]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	before, _, ok := bytes.Cut(rest, delimiter)
	if !ok {
		return result
	}

	yamlContent := before

	for line := range bytes.SplitSeq(yamlContent, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if colonIdx := bytes.IndexByte(trimmed, ':'); colonIdx > 0 {
			key := string(bytes.TrimSpace(trimmed[:colonIdx]))
			value := string(bytes.TrimSpace(trimmed[colonIdx+1:]))
			result[key] = value
		}
	}

	return result
}

// validateModels checks that all agent models in the map are valid according to `opencode models`.
// Returns an error listing invalid models and agents if any are found.
// When model specs include variants (e.g., "model (variant)"), only the base model is validated.
// Supports both single string models and arrays of model strings.
func validateModels(models map[string]any) error {
	if len(models) == 0 {
		return nil
	}

	validModels, err := fetchValidModels()
	if err != nil {
		return fmt.Errorf("failed to fetch valid models: %w", err)
	}

	var invalidAgents []string
	var invalidModelNames []string
	seen := make(map[string]bool)

	for agent := range models {
		modelSpecs := getModelsForAgent(models, agent)
		for _, modelSpec := range modelSpecs {
			if modelSpec == "" {
				continue
			}
			baseModel, _ := parseModelAndVariant(modelSpec)
			if !validModels[baseModel] {
				invalidAgents = append(invalidAgents, agent)
				if !seen[baseModel] {
					invalidModelNames = append(invalidModelNames, baseModel)
					seen[baseModel] = true
				}
			}
		}
	}

	if len(invalidAgents) > 0 {
		slices.Sort(invalidAgents)
		slices.Sort(invalidModelNames)

		validModelList := slices.Sorted(maps.Keys(validModels))

		return fmt.Errorf("invalid model(s) specified:\n  agents: %s\n  invalid models: %s\n  valid models: %s",
			strings.Join(invalidAgents, ", "),
			strings.Join(invalidModelNames, ", "),
			strings.Join(validModelList, ", "))
	}

	return nil
}

func fetchValidModels() (map[string]bool, error) {
	cmd := exec.Command("opencode", "models")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("opencode models command failed: %w", err)
	}

	validModels := make(map[string]bool)
	for line := range strings.SplitSeq(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			validModels[trimmed] = true
		}
	}

	return validModels, nil
}

// tryReloadGoalMetadata attempts to reload GOAL.md frontmatter from disk.
// If the file is unavailable, it preserves current metadata.
func tryReloadGoalMetadata(goalPath string, current GoalMetadata) (GoalMetadata, error) {
	content, errRead := os.ReadFile(goalPath)
	if errRead != nil {
		if os.IsNotExist(errRead) {
			return current, nil
		}
		return current, fmt.Errorf("failed to read GOAL.md: %w", errRead)
	}

	newMetadata, errParse := parseYAMLFrontmatter(content)
	if errParse != nil {
		return current, errParse
	}

	return newMetadata, nil
}

// parseYAMLFrontmatter extracts YAML frontmatter from content delimited by "---".
// If no frontmatter is found, returns default metadata.
func parseYAMLFrontmatter(content []byte) (GoalMetadata, error) {
	delimiter := []byte("---")
	defaultMetadata := GoalMetadata{}

	if !bytes.HasPrefix(content, delimiter) {
		return defaultMetadata, nil
	}

	rest := content[len(delimiter):]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	before, _, ok := bytes.Cut(rest, delimiter)
	if !ok {
		return GoalMetadata{}, fmt.Errorf("no closing '---' found for frontmatter")
	}

	yamlContent := before

	var metadata GoalMetadata
	if err := yaml.Unmarshal(yamlContent, &metadata); err != nil {
		return GoalMetadata{}, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	return metadata, nil
}

//go:embed skel/**
var skelFS embed.FS

func findFirstPendingMessageAgent(s state.Workflow) string {
	if len(s.Messages) == 0 {
		return ""
	}
	for _, msg := range s.Messages {
		if !msg.Read {
			return extractAgentFromModelID(msg.ToAgent)
		}
	}
	return ""
}

func redirectToPendingMessageAgent(s *state.Workflow, statePath, paddedsgai string) bool {
	pendingAgent := findFirstPendingMessageAgent(*s)
	if pendingAgent == "" {
		return false
	}
	fmt.Println("["+paddedsgai+"]", "pending messages for", pendingAgent, "- redirecting before completion")
	s.Status = state.StatusWorking
	s.CurrentAgent = pendingAgent
	s.VisitCounts[pendingAgent]++
	if err := state.Save(statePath, *s); err != nil {
		log.Fatalln("failed to save state:", err)
	}
	return true
}

func redirectToCoordinator(s *state.Workflow, statePath, paddedsgai string) {
	fmt.Println("["+paddedsgai+"]", "redirecting to coordinator before completion")
	s.Status = state.StatusWorking
	s.CurrentAgent = "coordinator"
	s.VisitCounts["coordinator"]++
	if err := state.Save(statePath, *s); err != nil {
		log.Fatalln("failed to save state:", err)
	}
}

func hasPendingMessages(s *state.Workflow, statePath, paddedsgai string) bool {
	for _, msg := range s.Messages {
		if !msg.Read {
			return redirectToPendingMessageAgent(s, statePath, paddedsgai)
		}
	}
	return false
}

func runCompletionGateScript(script string) (string, error) {
	cmd := exec.Command("sh", "-c", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}

func formatCompletionGateScriptFailureMessage(script, output string) string {
	return fmt.Sprintf(`From: environment
To: coordinator
Subject: computable definition of success has failed

The script %s has failed with this output:
<pre>
%s
</pre>
`, script, output)
}

// initVisitCounts initializes a visit counts map with all agents set to 0.
// This ensures send_message validation works before agents are visited.
func initVisitCounts(agents []string) map[string]int {
	counts := make(map[string]int)
	for _, agent := range agents {
		counts[agent] = 0
	}
	return counts
}

func ensureGitExclude(dir string) {
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		fmt.Println("[sgai]", "not a git repository, skipping .git/info/exclude")
		return
	}

	gitInfoDir := filepath.Join(gitDir, "info")
	if err := os.MkdirAll(gitInfoDir, 0755); err != nil {
		log.Println("[sgai]", "failed to create .git/info directory:", err)
		return
	}

	excludePath := filepath.Join(gitInfoDir, "exclude")
	existingContent, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		log.Println("[sgai]", "failed to read .git/info/exclude:", err)
		return
	}

	if dotSGAILinePresent(existingContent) {
		return
	}

	existingContent = append(existingContent, []byte("/.sgai\n")...)
	if err := os.WriteFile(excludePath, existingContent, 0644); err != nil {
		log.Println("[sgai]", "failed to write .git/info/exclude:", err)
		return
	}
}

func dotSGAILinePresent(content []byte) bool {
	for line := range bytes.SplitSeq(content, []byte("\n")) {
		if bytes.Equal(bytes.TrimSpace(line), []byte("/.sgai")) {
			return true
		}
	}
	return false
}

func ensureJJ(dir string) {
	if classifyWorkspace(dir) == workspaceFork {
		return
	}
	cmd := exec.Command("jj", "status")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		if isExecNotFound(err) {
			log.Fatalln("jj is required but not found in PATH")
		}
		initCmd := exec.Command("jj", "git", "init", "--colocate")
		initCmd.Dir = dir
		if errInit := initCmd.Run(); errInit != nil {
			log.Fatalln("failed to initialize jj:", errInit)
		}
	}
}

func isExecNotFound(err error) bool {
	var errExec *exec.Error
	if errors.As(err, &errExec) {
		return errors.Is(errExec.Err, exec.ErrNotFound)
	}
	return false
}

func initializeWorkspaceDir(dir string) error {
	subFS, err := fs.Sub(skelFS, "skel")
	if err != nil {
		return fmt.Errorf("failed to access skeleton filesystem: %w", err)
	}

	if err := fs.WalkDir(subFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		outPath := filepath.Join(dir, path)
		if d.IsDir() {
			return os.MkdirAll(outPath, 0755)
		}
		data, err := fs.ReadFile(subFS, path)
		if err != nil {
			return err
		}
		return os.WriteFile(outPath, data, 0644)
	}); err != nil {
		return fmt.Errorf("failed to unpack skeleton: %w", err)
	}

	if err := applyLayerFolderOverlay(dir); err != nil {
		return fmt.Errorf("failed to apply layer overlay: %w", err)
	}

	if err := initializeJJ(dir); err != nil {
		return fmt.Errorf("failed to initialize jj: %w", err)
	}

	ensureGitExclude(dir)

	return nil
}

func initializeJJ(dir string) error {
	if classifyWorkspace(dir) == workspaceFork {
		return nil
	}
	cmd := exec.Command("jj", "status")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		if isExecNotFound(err) {
			return fmt.Errorf("jj is required but not found in PATH")
		}
		initCmd := exec.Command("jj", "git", "init", "--colocate")
		initCmd.Dir = dir
		if errInit := initCmd.Run(); errInit != nil {
			return fmt.Errorf("failed to run jj git init: %w", errInit)
		}
	}
	return nil
}

func computeGoalChecksum(goalPath string) (string, error) {
	data, err := os.ReadFile(goalPath)
	if err != nil {
		return "", err
	}

	body := extractBody(data)
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:]), nil
}

func extractBody(content []byte) []byte {
	delimiter := []byte("---")

	if !bytes.HasPrefix(content, delimiter) {
		return content
	}

	rest := content[len(delimiter):]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	closingIdx := bytes.Index(rest, delimiter)
	if closingIdx == -1 {
		return content
	}

	bodyStart := len(delimiter) + 1 + closingIdx + len(delimiter)
	if bodyStart < len(content) && content[bodyStart] == '\n' {
		bodyStart++
	}
	if bodyStart >= len(content) {
		return []byte{}
	}
	return content[bodyStart:]
}

// generateRetrospectiveDirName generates a timestamp-based folder name in format YYYY-MM-DD-HH-II.XXXX
// where XXXX is 4 random lowercase alphanumeric characters [a-z0-9]
func generateRetrospectiveDirName() string {
	timestamp := time.Now().Format("2006-01-02-15-04")
	suffix := make([]byte, 4)
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	if _, err := rand.Read(suffix); err != nil {
		log.Fatalln("failed to generate random suffix:", err)
	}
	for i := range suffix {
		suffix[i] = chars[int(suffix[i])%len(chars)]
	}
	return timestamp + "." + string(suffix)
}

func setupOutputCapture(retrospectiveDir string) error {
	stdoutLogPath := filepath.Join(retrospectiveDir, "stdout.log")
	stderrLogPath := filepath.Join(retrospectiveDir, "stderr.log")

	stdoutLog, err := prepareLogFile(stdoutLogPath)
	if err != nil {
		return fmt.Errorf("preparing stdout.log: %w", err)
	}

	stderrLog, err := prepareLogFile(stderrLogPath)
	if err != nil {
		return fmt.Errorf("preparing stderr.log: %w", err)
	}

	originalStdout := os.Stdout
	originalStderr := os.Stderr

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}

	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}

	os.Stdout = stdoutW
	os.Stderr = stderrW

	go func() {
		if _, err := io.Copy(io.MultiWriter(originalStdout, stdoutLog), stdoutR); err != nil {
			log.Println("write failed:", err)
		}
	}()
	go func() {
		if _, err := io.Copy(io.MultiWriter(originalStderr, stderrLog), stderrR); err != nil {
			log.Println("write failed:", err)
		}
	}()

	return nil
}

func waitForStateTransition(dir, statePath string) string {
	responsePath := filepath.Join(dir, ".sgai", "response.txt")
	for {
		st, err := state.Load(statePath)
		if err == nil && st.Status == state.StatusWorking {
			data, err := os.ReadFile(responsePath)
			if err != nil {
				return ""
			}
			if err := os.Remove(responsePath); err != nil {
				log.Println("cleanup failed:", err)
			}
			return string(data)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func writeResponseAndTransition(dir, statePath, response string) {
	responsePath := filepath.Join(dir, ".sgai", "response.txt")
	if err := os.WriteFile(responsePath, []byte(response), 0644); err != nil {
		log.Fatalln("failed to write response file:", err)
	}
	st, err := state.Load(statePath)
	if err != nil {
		log.Fatalln("failed to load state:", err)
	}
	st.Status = state.StatusWorking
	if err := state.Save(statePath, st); err != nil {
		log.Fatalln("failed to save state:", err)
	}
}

func launchEditorForResponse(dir, humanMessage, statePath string) {
	response, err := openEditorForResponse(humanMessage)
	if err != nil {
		log.Fatalln("failed to get human response:", err)
	}
	writeResponseAndTransition(dir, statePath, response)
}

func handleMultiChoiceQuestion(dir, statePath string, mcq *state.MultiChoiceQuestion) {
	response, err := collectMultiChoiceResponse(mcq)
	if err != nil {
		log.Fatalln("failed to collect multi-choice response:", err)
	}

	wfState, err := state.Load(statePath)
	if err != nil {
		log.Fatalln("failed to load state:", err)
	}
	wfState.MultiChoiceQuestion = nil
	if err := state.Save(statePath, wfState); err != nil {
		log.Fatalln("failed to clear multi-choice question:", err)
	}

	writeResponseAndTransition(dir, statePath, response)
}

func collectMultiChoiceResponse(mcq *state.MultiChoiceQuestion) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	var allResponses []string

	for qIdx, q := range mcq.Questions {
		fmt.Println()
		fmt.Printf("# Question %d of %d\n", qIdx+1, len(mcq.Questions))
		fmt.Println(q.Question)
		fmt.Println()

		if q.MultiSelect {
			fmt.Println("(Select one or more options by entering numbers separated by commas, e.g., 1,3)")
		} else {
			fmt.Println("(Select one option by entering its number)")
		}
		fmt.Println()

		for i, choice := range q.Choices {
			fmt.Printf("  [%d] %s\n", i+1, choice)
		}
		fmt.Println()
		fmt.Println("  [O] Other (provide custom input)")
		fmt.Println()

		var selectedChoices []string
		for {
			fmt.Print("Your selection: ")
			input, err := reader.ReadString('\n')
			if err != nil {
				return "", fmt.Errorf("failed to read input: %w", err)
			}
			input = strings.TrimSpace(input)

			if input == "" {
				fmt.Println("Please enter a selection.")
				continue
			}

			selectedChoices, err = parseChoiceSelection(input, q.Choices, q.MultiSelect)
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}
			break
		}

		fmt.Println()
		fmt.Print("Other (optional, press Enter to skip): ")
		otherInput, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read other input: %w", err)
		}
		otherInput = strings.TrimSpace(otherInput)

		if !q.MultiSelect && len(selectedChoices) == 0 && otherInput == "" {
			return "", fmt.Errorf("must select at least one option or provide custom input")
		}

		response := formatMultiChoiceResponse(selectedChoices, otherInput)
		if len(mcq.Questions) > 1 {
			allResponses = append(allResponses, fmt.Sprintf("Q%d: %s\n%s", qIdx+1, q.Question, response))
		} else {
			allResponses = append(allResponses, response)
		}
	}

	return strings.Join(allResponses, "\n\n"), nil
}

func parseChoiceSelection(input string, choices []string, multiSelect bool) ([]string, error) {
	input = strings.ToUpper(strings.TrimSpace(input))

	if input == "O" {
		return nil, nil
	}

	parts := strings.Split(input, ",")
	var selectedIndices []int

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.ToUpper(part) == "O" {
			continue
		}

		idx, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid selection '%s': must be a number or 'O'", part)
		}
		if idx < 1 || idx > len(choices) {
			return nil, fmt.Errorf("invalid selection %d: must be between 1 and %d", idx, len(choices))
		}
		selectedIndices = append(selectedIndices, idx-1)
	}

	if !multiSelect && len(selectedIndices) > 1 {
		return nil, fmt.Errorf("single-select mode: please select only one option")
	}

	var selected []string
	for _, idx := range selectedIndices {
		selected = append(selected, choices[idx])
	}

	return selected, nil
}

func formatMultiChoiceResponse(selectedChoices []string, otherInput string) string {
	var parts []string

	if len(selectedChoices) > 0 {
		parts = append(parts, "Selected: "+strings.Join(selectedChoices, ", "))
	}

	if otherInput != "" {
		parts = append(parts, "Other: "+otherInput)
	}

	return strings.Join(parts, "\n")
}

func openEditorForResponse(humanMessage string) (string, error) {
	editor := os.Getenv("EDITOR")

	if editor == "" {
		fmt.Println("# Agent Message")
		fmt.Println()
		fmt.Println(humanMessage)
		fmt.Println()

		fd := int(os.Stdin.Fd())
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			fmt.Println("# Your Response (end with Ctrl+D):")
			fmt.Println()
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
		defer func() {
			if err := term.Restore(fd, oldState); err != nil {
				log.Println("close failed:", err)
			}
		}()

		t := term.NewTerminal(os.Stdin, "> ")
		if _, err := fmt.Fprintln(t, "# Your Response (empty line to finish):"); err != nil {
			log.Println("write failed:", err)
		}
		var lines []string
		for {
			line, err := t.ReadLine()
			if err != nil {
				break
			}
			if line == "" {
				break
			}
			lines = append(lines, line)
		}
		return strings.Join(lines, "\n"), nil
	}

	tmpFile, err := os.CreateTemp("", "sgai-*.md")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	defer func() {
		if err := os.Remove(tmpPath); err != nil {
			log.Println("cleanup failed:", err)
		}
	}()

	content := "# Agent Message\n\n" + humanMessage + "\n\n# Your Response\n\n"
	if _, err := tmpFile.WriteString(content); err != nil {
		if errClose := tmpFile.Close(); errClose != nil {
			log.Println("close failed:", errClose)
		}
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		log.Println("close failed:", err)
	}

	editorParts := strings.Fields(editor)
	cmd := exec.Command(editorParts[0], append(editorParts[1:], tmpPath)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type prefixWriter struct {
	prefix string
	w      io.Writer
}

func (p *prefixWriter) Write(data []byte) (int, error) {
	lines := linesWithTrailingEmpty(string(data))
	for i, line := range lines {
		if i < len(lines)-1 || line != "" {
			if _, err := p.w.Write([]byte(p.prefix + line + "\n")); err != nil {
				return 0, err
			}
		}
	}
	return len(data), nil
}

type streamEvent struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
	SessionID string `json:"sessionID"`
	Part      part   `json:"part"`
}

type part struct {
	Type   string     `json:"type"`
	Text   string     `json:"text,omitempty"`
	Tool   string     `json:"tool,omitempty"`
	State  *toolState `json:"state,omitempty"`
	Cost   float64    `json:"cost,omitempty"`
	Tokens partTokens `json:"tokens"`
}

type partTokens struct {
	Input     int        `json:"input"`
	Output    int        `json:"output"`
	Reasoning int        `json:"reasoning"`
	Cache     cacheStats `json:"cache"`
}

type cacheStats struct {
	Read  int `json:"read"`
	Write int `json:"write"`
}

type toolState struct {
	Status string         `json:"status"`
	Input  map[string]any `json:"input"`
	Title  string         `json:"title,omitempty"`
	Output string         `json:"output,omitempty"`
	Error  string         `json:"error,omitempty"`
}

type jsonPrettyWriter struct {
	prefix       string
	w            io.Writer
	buf          []byte
	currentText  strings.Builder
	sessionID    string
	statePath    string
	currentAgent string
	stepCounter  int
}

func (j *jsonPrettyWriter) Write(data []byte) (int, error) {
	j.buf = append(j.buf, data...)
	j.processBuffer()
	return len(data), nil
}

func (j *jsonPrettyWriter) processBuffer() {
	for {
		idx := strings.Index(string(j.buf), "\n")
		if idx == -1 {
			return
		}

		line := j.buf[:idx]
		j.buf = j.buf[idx+1:]

		if len(line) == 0 {
			continue
		}

		var event streamEvent
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}

		j.processEvent(event)
	}
}

func (j *jsonPrettyWriter) processEvent(event streamEvent) {
	if event.SessionID != "" {
		j.sessionID = event.SessionID
	}
	part := event.Part

	switch event.Type {
	case "text":
		if part.Text != "" {
			j.currentText.WriteString(part.Text)
		}

	case "tool", "tool_use":
		j.flushText()
		if part.State != nil {
			toolCall := formatToolCall(part.Tool, part.State.Input)
			switch part.State.Status {
			case "pending":
				if _, err := fmt.Fprintln(j.w, j.prefix+toolCall); err != nil {
					log.Println("write failed:", err)
				}
			case "running":
				if _, err := fmt.Fprintln(j.w, j.prefix+toolCall+" ..."); err != nil {
					log.Println("write failed:", err)
				}
			case "completed":
				if _, err := fmt.Fprintln(j.w, j.prefix+toolCall); err != nil {
					log.Println("write failed:", err)
				}
				if part.State.Output != "" {
					if isTodoTool(part.Tool) {
						j.formatTodoOutput(part.State.Output)
					} else {
						for line := range strings.SplitSeq(part.State.Output, "\n") {
							if _, err := fmt.Fprintln(j.w, j.prefix+"  → "+line); err != nil {
								log.Println("write failed:", err)
							}
						}
					}
				}
			case "error":
				if _, err := fmt.Fprintln(j.w, j.prefix+toolCall+" ERROR:", part.State.Error); err != nil {
					log.Println("write failed:", err)
				}
			}
		}

	case "step_start":
		j.flushText()
		j.stepCounter++

	case "step_finish":
		j.flushText()
		j.recordStepCost(part, event.Timestamp)

	case "reasoning":
		j.flushText()
		if part.Text != "" {
			if _, err := fmt.Fprintln(j.w, j.prefix+"[thinking] ..."); err != nil {
				log.Println("write failed:", err)
			}
		}

	default:
		if event.Type != "" {
			if _, err := fmt.Fprintln(j.w, j.prefix+"["+event.Type+"]"); err != nil {
				log.Println("write failed:", err)
			}
		}
	}
}

func (j *jsonPrettyWriter) flushText() {
	if j.currentText.Len() > 0 {
		text := j.currentText.String()
		for line := range strings.SplitSeq(text, "\n") {
			if _, err := fmt.Fprintln(j.w, j.prefix+line); err != nil {
				log.Println("write failed:", err)
			}
		}
		j.currentText.Reset()
	}
}

func (j *jsonPrettyWriter) Flush() {
	j.processBuffer()
	j.flushText()
}

func (j *jsonPrettyWriter) recordStepCost(p part, timestamp int64) {
	if j.statePath == "" || j.currentAgent == "" {
		return
	}
	if p.Cost == 0 && p.Tokens.Input == 0 && p.Tokens.Output == 0 {
		return
	}

	wfState, err := state.Load(j.statePath)
	if err != nil {
		return
	}

	stepCost := state.StepCost{
		StepID: fmt.Sprintf("%s-step-%d", j.currentAgent, j.stepCounter),
		Agent:  j.currentAgent,
		Cost:   p.Cost,
		Tokens: state.TokenUsage{
			Input:      p.Tokens.Input,
			Output:     p.Tokens.Output,
			Reasoning:  p.Tokens.Reasoning,
			CacheRead:  p.Tokens.Cache.Read,
			CacheWrite: p.Tokens.Cache.Write,
		},
		Timestamp: time.Unix(0, timestamp*int64(time.Millisecond)).UTC().Format(time.RFC3339),
	}

	wfState.Cost.TotalCost += stepCost.Cost
	wfState.Cost.TotalTokens.Add(stepCost.Tokens)

	agentIdx := slices.IndexFunc(wfState.Cost.ByAgent, func(ac state.AgentCost) bool {
		return ac.Agent == j.currentAgent
	})
	if agentIdx == -1 {
		wfState.Cost.ByAgent = append(wfState.Cost.ByAgent, state.AgentCost{
			Agent:  j.currentAgent,
			Cost:   stepCost.Cost,
			Tokens: stepCost.Tokens,
			Steps:  []state.StepCost{stepCost},
		})
	} else {
		wfState.Cost.ByAgent[agentIdx].Cost += stepCost.Cost
		wfState.Cost.ByAgent[agentIdx].Tokens.Add(stepCost.Tokens)
		wfState.Cost.ByAgent[agentIdx].Steps = append(wfState.Cost.ByAgent[agentIdx].Steps, stepCost)
	}

	if err := state.Save(j.statePath, wfState); err != nil {
		log.Println("failed to save state:", err)
	}
}

func isTodoTool(tool string) bool {
	switch tool {
	case "todowrite", "todoread", "sgai_project_todowrite", "sgai_project_todoread":
		return true
	default:
		return false
	}
}

func (j *jsonPrettyWriter) formatTodoOutput(output string) {
	type todo struct {
		Content  string `json:"content"`
		Status   string `json:"status"`
		Priority string `json:"priority"`
	}

	jsonOutput := stripMCPTodoPrefix(output)

	var todos []todo
	if err := json.Unmarshal([]byte(jsonOutput), &todos); err != nil {
		for line := range strings.SplitSeq(output, "\n") {
			if _, err := fmt.Fprintln(j.w, j.prefix+"  → "+line); err != nil {
				log.Println("write failed:", err)
			}
		}
		return
	}

	for _, t := range todos {
		symbol := todoStatusSymbol(t.Status)
		if _, err := fmt.Fprintf(j.w, "%s  → %s %s (%s)\n", j.prefix, symbol, t.Content, t.Priority); err != nil {
			log.Println("write failed:", err)
		}
	}
}

func stripMCPTodoPrefix(output string) string {
	idx := strings.Index(output, "\n[")
	if idx == -1 {
		return output
	}
	prefix := strings.TrimSpace(output[:idx])
	if strings.HasSuffix(prefix, "todos") || strings.HasSuffix(prefix, "todo") {
		return output[idx+1:]
	}
	return output
}

func todoStatusSymbol(status string) string {
	switch status {
	case "pending":
		return "○"
	case "in_progress":
		return "◐"
	case "completed":
		return "●"
	case "cancelled":
		return "✕"
	default:
		return "○"
	}
}

func formatToolCall(tool string, input map[string]any) string {
	if len(input) == 0 {
		return tool
	}
	escapeReplacer := strings.NewReplacer(
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
	)
	var parts []string
	for k, v := range input {
		switch val := v.(type) {
		case string:
			val = escapeReplacer.Replace(val)
			if k != "filePath" && len(val) > 50 {
				val = val[:47] + "..."
			}
			parts = append(parts, k+": '"+val+"'")
		case bool:
			parts = append(parts, k+": "+strconv.FormatBool(val))
		case float64:
			parts = append(parts, k+": "+strconv.FormatFloat(val, 'f', -1, 64))
		default:
			parts = append(parts, k+": "+fmt.Sprint(val))
		}
	}
	return tool + "(" + strings.Join(parts, ", ") + ")"
}

func extractFrontmatterDescription(content string) string {
	fm := parseFrontmatterMap([]byte(content))
	return fm["description"]
}

func updateProjectManagementWithRetrospectiveDir(pmPath, retrospectiveDirRel string) error {
	const headerDelimiter = "---"
	const headerPrefix = "Retrospective Session: "

	var existingContent []byte
	existingContent, err := os.ReadFile(pmPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read PROJECT_MANAGEMENT.md: %w", err)
	}

	newHeader := fmt.Sprintf("%s\n%s%s\n%s\n", headerDelimiter, headerPrefix, retrospectiveDirRel, headerDelimiter)

	content := string(existingContent)
	lines := linesWithTrailingEmpty(content)

	if len(lines) >= 3 && strings.HasPrefix(lines[0], headerDelimiter) {
		endIdx := -1
		for i := 1; i < len(lines); i++ {
			if strings.HasPrefix(lines[i], headerDelimiter) {
				endIdx = i
				break
			}
		}

		if endIdx > 0 {
			for i := 1; i < endIdx; i++ {
				if strings.HasPrefix(lines[i], headerPrefix) {
					remainingLines := lines[endIdx+1:]
					if len(remainingLines) > 0 && remainingLines[0] == "" {
						remainingLines = remainingLines[1:]
					}
					content = strings.Join(remainingLines, "\n")
					break
				}
			}
		}
	}

	if content != "" && !strings.HasPrefix(content, "\n") {
		newHeader += "\n"
	}

	finalContent := newHeader + content

	if err := os.MkdirAll(filepath.Dir(pmPath), 0755); err != nil {
		return fmt.Errorf("failed to create .sgai directory: %w", err)
	}

	if err := os.WriteFile(pmPath, []byte(finalContent), 0644); err != nil {
		return fmt.Errorf("failed to write PROJECT_MANAGEMENT.md: %w", err)
	}

	return nil
}

// canResumeWorkflow determines if an existing workflow can be resumed based on
// the current state, whether the --fresh flag was provided, and whether the
// GOAL.md checksum matches the stored checksum.
func canResumeWorkflow(wfState state.Workflow, freshFlag bool, currentGoalChecksum string) bool {
	if freshFlag {
		return false
	}
	if wfState.GoalChecksum != currentGoalChecksum {
		return false
	}
	return wfState.Status == state.StatusWorking ||
		wfState.Status == state.StatusAgentDone ||
		state.IsHumanPending(wfState.Status)
}

// extractRetrospectiveDirFromProjectManagement parses the PROJECT_MANAGEMENT.md
// frontmatter to extract the Retrospective Session path.
func extractRetrospectiveDirFromProjectManagement(pmPath string) string {
	const headerPrefix = "Retrospective Session: "

	content, err := os.ReadFile(pmPath)
	if err != nil {
		return ""
	}

	lines := linesWithTrailingEmpty(string(content))
	if len(lines) < 3 {
		return ""
	}

	if !strings.HasPrefix(lines[0], "---") {
		return ""
	}

	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "---") {
			break
		}
		if after, ok := strings.CutPrefix(line, headerPrefix); ok {
			return after
		}
	}

	return ""
}

func copyToRetrospective(src, dst string) error {
	return copyFileAtomic(src, dst)
}

func copyFileAtomic(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	tmpDst := dst + ".tmp"

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := srcFile.Close(); err != nil {
			log.Println("close failed:", err)
		}
	}()

	tmpFile, err := os.Create(tmpDst)
	if err != nil {
		return err
	}
	tmpClosed := false
	defer func() {
		if !tmpClosed {
			if errClose := tmpFile.Close(); errClose != nil {
				log.Println("close failed:", errClose)
			}
		}
		if err != nil {
			if errRemove := os.Remove(tmpDst); errRemove != nil {
				log.Println("cleanup failed:", errRemove)
			}
		}
	}()

	if _, err = io.Copy(tmpFile, srcFile); err != nil {
		return err
	}

	if err = tmpFile.Close(); err != nil {
		return err
	}
	tmpClosed = true

	if err = os.Rename(tmpDst, dst); err != nil {
		return err
	}

	return nil
}

func copyFinalStateToRetrospective(dir, retrospectiveDir string) error {
	statePath := filepath.Join(dir, ".sgai", "state.json")
	pmPath := filepath.Join(dir, ".sgai", "PROJECT_MANAGEMENT.md")

	if _, err := os.Stat(statePath); err == nil {
		stateDst := filepath.Join(retrospectiveDir, "state.json")
		if err := copyFileAtomic(statePath, stateDst); err != nil {
			return fmt.Errorf("failed to copy state.json: %w", err)
		}
	}

	if _, err := os.Stat(pmPath); err == nil {
		pmDst := filepath.Join(retrospectiveDir, "PROJECT_MANAGEMENT.md")
		if err := copyFileAtomic(pmPath, pmDst); err != nil {
			return fmt.Errorf("failed to copy PROJECT_MANAGEMENT.md: %w", err)
		}
	}

	return nil
}

func exportSession(sessionID, outputPath string) error {
	cmd := exec.Command("opencode", "export", sessionID)
	cmd.Env = append(os.Environ(), "OPENCODE_CONFIG_DIR=.sgai")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("opencode export failed: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(outputPath, output, 0644)
}

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

	skelAgents := make(map[string]string) // name -> description
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
	skelNames := make([]string, 0, len(skelAgents))
	for name := range skelAgents {
		skelNames = append(skelNames, name)
	}
	slices.Sort(skelNames)
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
		dirNames := make([]string, 0, len(dirAgents))
		for name := range dirAgents {
			dirNames = append(dirNames, name)
		}
		slices.Sort(dirNames)
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

func applyLayerFolderOverlay(dir string) error {
	layerDir := filepath.Join(dir, "sgai")
	if !isExistingDirectory(layerDir) {
		return nil
	}

	allowedSubfolders := []string{"agent", "skills", "snippets"}
	for _, subfolder := range allowedSubfolders {
		srcDir := filepath.Join(layerDir, subfolder)
		dstDir := filepath.Join(dir, ".sgai", subfolder)
		if err := copyLayerSubfolder(srcDir, dstDir, subfolder); err != nil {
			return err
		}
	}

	return nil
}

func copyLayerSubfolder(srcDir, dstDir, subfolder string) error {
	if !isExistingDirectory(srcDir) {
		return nil
	}

	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dstDir, relPath), 0755)
		}

		if isProtectedFile(subfolder, relPath) {
			return nil
		}

		return copyFileAtomic(path, filepath.Join(dstDir, relPath))
	})
}

func isExistingDirectory(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.IsDir()
}

func isProtectedFile(subfolder, relPath string) bool {
	return subfolder == "agent" && relPath == "coordinator.md"
}
