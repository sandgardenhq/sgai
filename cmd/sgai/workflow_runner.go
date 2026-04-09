package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sandgardenhq/sgai/pkg/state"
)

type workflowRunner struct {
	dir              string
	goalPath         string
	coord            *state.Coordinator
	metadata         GoalMetadata
	flowDag          *dag
	wfState          state.Workflow
	retroDir         string
	paddedsgai       string
	longestNameLen   int
	mcpURL           string
	logWriter        io.Writer
	retroLogs        retroLogWriters
	iterationCounter int
	previousAgent    string
	backend          Backend
}

type retroLogWriters struct {
	stdout io.WriteCloser
	stderr io.WriteCloser
}

type runResult int

const (
	resultContinue runResult = iota
	resultComplete
	resultInterrupt
)

func (r *workflowRunner) run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			fmt.Println("["+r.paddedsgai+"]", "interrupted, stopping workflow...")
			return
		}

		currentAgent := r.resolveCurrentAgent()
		result := r.runAgent(ctx, currentAgent)

		switch result {
		case resultInterrupt:
			return
		case resultComplete:
			return
		case resultContinue:
		}
	}
}

func (r *workflowRunner) resolveCurrentAgent() string {
	if r.wfState.CurrentAgent == "" {
		return "coordinator"
	}
	return r.wfState.CurrentAgent
}

func (r *workflowRunner) runAgent(ctx context.Context, currentAgent string) runResult {
	r.prepareAgent(currentAgent)

	var errReload error
	r.metadata, errReload = tryReloadGoalMetadata(r.goalPath, r.metadata, r.flowDag)
	if errReload != nil {
		log.Println("failed to reload GOAL.md frontmatter:", errReload)
		return resultInterrupt
	}

	unlockInteractiveForRetrospective(&r.wfState, currentAgent, r.coord, r.paddedsgai)
	r.wfState = r.executeAgent(ctx, currentAgent)

	if ctx.Err() != nil {
		return resultInterrupt
	}

	if r.wfState.Status == state.StatusComplete {
		if redirectToPendingMessageAgent(&r.wfState, r.coord, r.paddedsgai) {
			return resultContinue
		}
		fmt.Println("["+r.paddedsgai+"]", "complete:", r.wfState.Task)
		return resultComplete
	}

	nextAgent := r.resolveNextAgent(currentAgent)
	r.wfState.CurrentAgent = nextAgent
	return resultContinue
}

func (r *workflowRunner) resolveNextAgent(currentAgent string) string {
	pendingAgent := findFirstPendingMessageAgent(r.wfState)
	if pendingAgent != "" {
		fmt.Println("["+r.paddedsgai+"]", "pending messages for", pendingAgent, "- redirecting")
		return pendingAgent
	}

	if r.flowDag.isTerminal(currentAgent) {
		fmt.Println("["+r.paddedsgai+"]", "reached terminal node", currentAgent)
		return "coordinator"
	}

	if currentAgent == "coordinator" && len(r.flowDag.EntryNodes) > 0 {
		return r.flowDag.EntryNodes[0]
	}

	return determineNextAgent(r.flowDag, currentAgent)
}

func (r *workflowRunner) prepareAgent(currentAgent string) {
	if r.previousAgent != "" && r.previousAgent != currentAgent {
		fmt.Println("["+r.paddedsgai+"]", r.previousAgent, "->", currentAgent)
		r.wfState.Todos = []state.TodoItem{}
		if errOverlay := applyLayerFolderOverlay(r.dir); errOverlay != nil {
			log.Println("failed to apply overlay on agent transition:", errOverlay)
		}
	}
	r.previousAgent = currentAgent

	r.wfState.CurrentAgent = currentAgent
	r.wfState.VisitCounts[currentAgent]++
	addAgentHandoffProgress(&r.wfState, currentAgent)
	markCurrentAgentInSequence(&r.wfState, currentAgent)

	snapshot := r.wfState
	if errUpdate := r.coord.UpdateState(func(wf *state.Workflow) {
		*wf = snapshot
	}); errUpdate != nil {
		log.Println("failed to save state:", errUpdate)
	}
}

func (r *workflowRunner) executeAgent(ctx context.Context, currentAgent string) state.Workflow {
	cfg := multiModelConfig{
		dir:              r.dir,
		goalPath:         r.goalPath,
		agent:            currentAgent,
		flowDag:          r.flowDag,
		statePath:        filepath.Join(r.dir, ".sgai", "state.json"),
		coord:            r.coord,
		retrospectiveDir: r.retroDir,
		longestNameLen:   r.longestNameLen,
		paddedsgai:       r.paddedsgai,
		mcpURL:           r.mcpURL,
		logWriter:        r.logWriter,
		stdoutLog:        r.retroLogs.stdout,
		stderrLog:        r.retroLogs.stderr,
		backend:          r.backend,
	}
	return runMultiModelAgent(ctx, cfg, r.wfState, r.metadata, &r.iterationCounter)
}

func (r *workflowRunner) runContinuous(ctx context.Context, continuousPrompt string) {
	goalPath := filepath.Join(r.dir, "GOAL.md")
	stateJSONPath := filepath.Join(r.dir, ".sgai", "state.json")

	for {
		if ctx.Err() != nil {
			return
		}

		runWorkflow(ctx, r.dir, r.mcpURL, r.logWriter, r.coord)

		freshCoord, errCoord := state.NewCoordinator(stateJSONPath)
		if errCoord != nil {
			freshCoord = state.NewCoordinatorEmpty(stateJSONPath)
		}
		r.coord = freshCoord

		if ctx.Err() != nil {
			return
		}

		runContinuousModePrompt(ctx, r.dir, continuousPrompt, r.mcpURL, r.coord, r.backend)

		if ctx.Err() != nil {
			return
		}

		checksum, errChecksum := computeGoalChecksum(goalPath)
		if errChecksum != nil {
			log.Println("failed to compute GOAL.md checksum:", errChecksum)
			return
		}

		autoDuration, cronExpr := readContinuousModeAutoCron(r.dir)

		trigger := watchForTrigger(ctx, r.dir, r.coord, checksum, autoDuration, cronExpr)
		if trigger == triggerNone {
			return
		}

		reloadedCoord, errFresh := state.NewCoordinator(stateJSONPath)
		if errFresh == nil {
			r.coord = reloadedCoord
		}

		r.handleTrigger(trigger, goalPath)
		resetWorkflowForNextCycle(r.coord)
	}
}

func (r *workflowRunner) handleTrigger(trigger triggerKind, goalPath string) {
	if trigger != triggerSteering {
		return
	}
	wfState := r.coord.State()
	found, msg := hasHumanPartnerMessage(wfState.Messages)
	if !found || msg == nil {
		return
	}
	if errPrepend := prependSteeringMessage(goalPath, msg.Body); errPrepend != nil {
		log.Println("failed to prepend steering message:", errPrepend)
	}
	markMessageAsRead(r.coord, msg.ID)
}

func buildWorkflowRunner(dir string, mcpURL string, logWriter io.Writer, sessionCoord *state.Coordinator) (*workflowRunner, func(), bool) {
	goalPath := filepath.Join(dir, "GOAL.md")
	goalContent, errRead := os.ReadFile(goalPath)
	if errRead != nil {
		if os.IsNotExist(errRead) {
			log.Fatalln("GOAL.md not found in", dir)
		}
		log.Fatalln(errRead)
	}

	metadata, errParse := parseYAMLFrontmatter(goalContent)
	if errParse != nil {
		log.Fatalln("failed to parse GOAL.md frontmatter:", errParse)
	}

	projectConfig, errConfig := loadProjectConfig(dir)
	if errConfig != nil {
		log.Fatalln("failed to load sgai.json:", errConfig)
	}

	backend, errBackend := resolveBackendStrict(projectConfig)
	if errBackend != nil {
		log.Fatalln(errBackend)
	}

	if _, err := exec.LookPath(backend.BinaryName()); err != nil {
		log.Fatalf("%s is required but not found in PATH", backend.BinaryName())
	}

	if errValidate := validateProjectConfig(projectConfig); errValidate != nil {
		log.Fatalln(errValidate)
	}

	applyConfigDefaults(projectConfig, &metadata)

	if errInit := initializeWorkspaceDir(dir); errInit != nil {
		log.Fatalln("failed to initialize workspace directory:", errInit)
	}

	if errMCP := applyCustomMCPs(dir, projectConfig); errMCP != nil {
		log.Fatalln("failed to apply custom MCPs:", errMCP)
	}

	flowDag, errFlow := parseFlow(metadata.Flow, dir)
	if errFlow != nil {
		log.Fatalln("failed to parse flow:", errFlow)
	}

	if retrospectiveEnabled(metadata) {
		flowDag.injectRetrospectiveEdge()
	}

	ensureImplicitAgentModel(flowDag, &metadata, "project-critic-council")
	ensureImplicitAgentModel(flowDag, &metadata, "retrospective")

	if errModels := backend.ValidateModels(metadata.Models); errModels != nil {
		log.Fatalln(errModels)
	}

	stateJSONPath := filepath.Join(dir, ".sgai", "state.json")
	coord := sessionCoord
	if coord == nil {
		var errCoord error
		coord, errCoord = state.NewCoordinator(stateJSONPath)
		if errCoord != nil && !os.IsNotExist(errCoord) {
			log.Fatalln("failed to read state.json:", errCoord)
		}
		if errCoord != nil {
			coord = state.NewCoordinatorEmpty(stateJSONPath)
		}
	}

	wfState := coord.State()
	newChecksum, errChecksum := computeGoalChecksum(goalPath)
	if errChecksum != nil {
		log.Fatalln("failed to compute GOAL.md checksum:", errChecksum)
	}

	dagAgents := flowDag.allAgents()
	allAgents := buildAllAgents(dagAgents)

	workspaceName := filepath.Base(dir)
	longestNameLen := computeLongestNameLen(allAgents)
	paddedsgai := workspaceName + "][" + "sgai" + strings.Repeat(" ", max(0, longestNameLen-len("sgai")))

	pmPath := filepath.Join(dir, ".sgai", "PROJECT_MANAGEMENT.md")
	retrospectivesBaseDir := filepath.Join(dir, ".sgai", "retrospectives")

	resuming := canResumeWorkflow(wfState, newChecksum)

	retroDir := resolveRetrospectiveDir(resuming, dir, retrospectivesBaseDir, pmPath, stateJSONPath, goalPath)

	retroStdoutLog, retroStderrLog, errRetroLogs := openRetrospectiveLogs(retroDir)
	if errRetroLogs != nil {
		log.Fatalln("failed to open retrospective logs:", errRetroLogs)
	}

	cleanup := func() {
		if errClose := retroStdoutLog.Close(); errClose != nil {
			log.Println("failed to close stdout log:", errClose)
		}
		if errClose := retroStderrLog.Close(); errClose != nil {
			log.Println("failed to close stderr log:", errClose)
		}
		if retroDir != "" {
			if errCopy := copyFinalStateToRetrospective(dir, retroDir); errCopy != nil {
				log.Println("[sgai] warning: failed to copy final state:", errCopy)
			}
		}
	}

	if !resuming {
		preservedMode := wfState.InteractionMode
		freshState := state.Workflow{
			Status:          state.StatusWorking,
			Messages:        []state.Message{},
			GoalChecksum:    newChecksum,
			VisitCounts:     initVisitCounts(allAgents),
			InteractionMode: preservedMode,
		}
		if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
			*wf = freshState
		}); errUpdate != nil {
			log.Println("failed to initialize state.json:", errUpdate)
			cleanup()
			return nil, func() {}, false
		}
		wfState = coord.State()
	}

	retroLogs := retroLogWriters{stdout: retroStdoutLog, stderr: retroStderrLog}
	runner := &workflowRunner{
		dir:            dir,
		goalPath:       goalPath,
		coord:          coord,
		metadata:       metadata,
		flowDag:        flowDag,
		wfState:        wfState,
		retroDir:       retroDir,
		paddedsgai:     paddedsgai,
		longestNameLen: longestNameLen,
		mcpURL:         mcpURL,
		logWriter:      logWriter,
		retroLogs:      retroLogs,
		backend:        backend,
	}
	return runner, cleanup, true
}

func buildAllAgents(dagAgents []string) []string {
	if slices.Contains(dagAgents, "coordinator") {
		return dagAgents
	}
	return append([]string{"coordinator"}, dagAgents...)
}

func computeLongestNameLen(agents []string) int {
	longest := len("sgai")
	for _, agent := range agents {
		longest = max(longest, len(agent))
	}
	return longest
}

func resolveRetrospectiveDir(resuming bool, dir, retrospectivesBaseDir, pmPath, stateJSONPath, goalPath string) string {
	if resuming {
		retroDirRel := extractRetrospectiveDirFromProjectManagement(pmPath)
		if retroDirRel == "" {
			log.Fatalln("failed to read retrospective directory from PROJECT_MANAGEMENT.md during resume")
		}
		retroDir := filepath.Join(dir, retroDirRel)
		if _, errStat := os.Stat(retroDir); os.IsNotExist(errStat) {
			log.Fatalln("retrospective directory from PROJECT_MANAGEMENT.md does not exist:", retroDir)
		}
		return retroDir
	}

	retroDir := filepath.Join(retrospectivesBaseDir, generateRetrospectiveDirName())
	if errMkdir := os.MkdirAll(retroDir, 0755); errMkdir != nil {
		log.Fatalln("failed to create retrospective directory:", errMkdir)
	}

	retroDirRel, errRel := filepath.Rel(dir, retroDir)
	if errRel != nil {
		log.Fatalln("failed to compute relative retrospective directory path:", errRel)
	}

	if errRemove := os.Remove(stateJSONPath); errRemove != nil && !os.IsNotExist(errRemove) {
		log.Fatalln("failed to truncate state.json on startup:", errRemove)
	}
	if errRemove := os.Remove(pmPath); errRemove != nil && !os.IsNotExist(errRemove) {
		log.Fatalln("failed to truncate PROJECT_MANAGEMENT.md on startup:", errRemove)
	}

	if errUpdate := updateProjectManagementWithRetrospectiveDir(pmPath, retroDirRel); errUpdate != nil {
		log.Fatalln("failed to update PROJECT_MANAGEMENT.md with retrospective directory:", errUpdate)
	}

	goalRetrospectivePath := filepath.Join(retroDir, "GOAL.md")
	if errCopy := copyFileAtomic(goalPath, goalRetrospectivePath); errCopy != nil {
		log.Fatalln("failed to copy GOAL.md to retrospective:", errCopy)
	}

	return retroDir
}
