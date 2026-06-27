package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

type workflowRunner struct {
	dir              string
	goalPath         string
	coord            *state.Coordinator
	metadata         GoalMetadata
	wfState          state.Workflow
	retroDir         string
	paddedsgai       string
	mcpURL           string
	logWriter        io.Writer
	retroLogs        retroLogWriters
	runtime          sessionRuntime
	iterationCounter int
}

func addCoordinatorHandoffProgress(wfState *state.Workflow) {
	progressEntry := state.ProgressEntry{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Agent:       "sgai",
		Description: "Handing off to coordinator",
	}
	wfState.Progress = append(wfState.Progress, progressEntry)
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

func runWorkflow(ctx context.Context, dir string, mcpURL string, logWriter io.Writer, sessionCoord *state.Coordinator, runtime sessionRuntime) {
	runner, cleanup, ok := buildWorkflowRunner(dir, mcpURL, logWriter, sessionCoord, runtime)
	if !ok {
		return
	}
	defer cleanup()
	runner.run(ctx)
}

func (r *workflowRunner) run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			fmt.Println("["+r.paddedsgai+"]", "interrupted, stopping workflow...")
			return
		}

		result := r.runAgent(ctx)

		switch result {
		case resultInterrupt:
			return
		case resultComplete:
			return
		case resultContinue:
		}
	}
}

func (r *workflowRunner) runAgent(ctx context.Context) runResult {
	r.prepareCoordinator()

	var errReload error
	r.metadata, errReload = parseYAMLFrontmatterFromFile(r.goalPath)
	if errReload != nil {
		log.Println("failed to reload GOAL.md frontmatter:", errReload)
		return resultInterrupt
	}

	r.wfState = r.executeCoordinator(ctx)

	if ctx.Err() != nil {
		return resultInterrupt
	}

	if r.wfState.Status == state.StatusComplete {
		fmt.Println("["+r.paddedsgai+"]", "complete:", r.wfState.Task)
		return resultComplete
	}

	return resultContinue
}

func (r *workflowRunner) prepareCoordinator() {
	addCoordinatorHandoffProgress(&r.wfState)

	snapshot := r.wfState
	if errUpdate := r.coord.UpdateState(func(wf *state.Workflow) {
		*wf = snapshot
	}); errUpdate != nil {
		log.Println("failed to save state:", errUpdate)
	}
}

func (r *workflowRunner) executeCoordinator(ctx context.Context) state.Workflow {
	cfg := agentRunConfig{
		dir:                   r.dir,
		goalPath:              r.goalPath,
		agent:                 "coordinator",
		statePath:             filepath.Join(r.dir, ".sgai", "state.json"),
		coord:                 r.coord,
		retrospectiveDir:      r.retroDir,
		goalAgents:            r.metadata.Agents,
		paddedsgai:            r.paddedsgai,
		mcpURL:                r.mcpURL,
		logWriter:             r.logWriter,
		stdoutLog:             r.retroLogs.stdout,
		stderrLog:             r.retroLogs.stderr,
		activeAgents:          r.runtime.activeAgents,
		onActiveAgentsChanged: r.runtime.onActiveAgentsChanged,
	}
	return runSingleModelIteration(ctx, cfg, r.wfState, r.metadata, &r.iterationCounter)
}

func (r *workflowRunner) runContinuous(ctx context.Context, continuousPrompt string) {
	goalPath := filepath.Join(r.dir, "GOAL.md")
	stateJSONPath := filepath.Join(r.dir, ".sgai", "state.json")

	for {
		if ctx.Err() != nil {
			return
		}

		runWorkflow(ctx, r.dir, r.mcpURL, r.logWriter, r.coord, r.runtime)

		freshCoord, errCoord := state.NewCoordinator(stateJSONPath)
		if errCoord != nil {
			freshCoord = state.NewCoordinatorEmpty(stateJSONPath)
		}
		r.coord = freshCoord

		if ctx.Err() != nil {
			return
		}

		runContinuousModePrompt(ctx, r.dir, continuousPrompt, r.mcpURL, r.coord)

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
	steeringMessage := readPendingSteeringMessage(r.dir)
	if steeringMessage == "" {
		return
	}
	if errPrepend := prependSteeringMessage(goalPath, steeringMessage); errPrepend != nil {
		log.Println("failed to prepend steering message:", errPrepend)
	}
}

func buildWorkflowRunner(dir string, mcpURL string, logWriter io.Writer, sessionCoord *state.Coordinator, runtime sessionRuntime) (*workflowRunner, func(), bool) {
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

	stateJSONPath := filepath.Join(dir, ".sgai", "state.json")
	coord := sessionCoord
	if coord == nil {
		var errCoord error
		coord, errCoord = state.NewCoordinator(stateJSONPath)
		if errCoord != nil && !errors.Is(errCoord, os.ErrNotExist) {
			log.Fatalln("failed to read state.json:", errCoord)
		}
		if errCoord != nil {
			coord = state.NewCoordinatorEmpty(stateJSONPath)
		}
	}

	wfState := coord.State()
	workspaceName := filepath.Base(dir)
	paddedsgai := workspaceName + "][" + "sgai"

	pmPath := filepath.Join(dir, ".sgai", "PROJECT_MANAGEMENT.md")
	retrospectivesBaseDir := filepath.Join(dir, ".sgai", "retrospectives")

	_, errStateStat := os.Stat(stateJSONPath)
	resuming := errStateStat == nil && wfState.Status != ""

	retrospectiveOn := retrospectiveEnabled(metadata)
	retroDir := ""
	if retrospectiveOn {
		retroDir = resolveRetrospectiveDir(resuming, dir, retrospectivesBaseDir, pmPath, stateJSONPath, goalPath)
	}

	retroStdoutLog, retroStderrLog, errRetroLogs := openRetrospectiveLogs(retroDir)
	if errRetroLogs != nil {
		log.Fatalln("failed to open retrospective logs:", errRetroLogs)
	}

	cleanup := func() {
		if retroStdoutLog != nil {
			if errClose := retroStdoutLog.Close(); errClose != nil {
				log.Println("failed to close stdout log:", errClose)
			}
		}
		if retroStderrLog != nil {
			if errClose := retroStderrLog.Close(); errClose != nil {
				log.Println("failed to close stderr log:", errClose)
			}
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
		dir:        dir,
		goalPath:   goalPath,
		coord:      coord,
		metadata:   metadata,
		wfState:    wfState,
		retroDir:   retroDir,
		paddedsgai: paddedsgai,
		mcpURL:     mcpURL,
		logWriter:  logWriter,
		retroLogs:  retroLogs,
		runtime:    runtime,
	}
	return runner, cleanup, true
}

func runtimeGoalAgents(goalAgents []string) []string {
	agents := make([]string, 0, len(goalAgents))
	for _, agent := range goalAgents {
		if isRetiredWorkflowAgent(agent) {
			continue
		}
		agents = append(agents, agent)
	}
	return agents
}

func delegatableAgents(agents []string) []string {
	delegatable := make([]string, 0, len(agents))
	for _, agent := range agents {
		if !isDelegatableAgent(agent) {
			continue
		}
		delegatable = append(delegatable, agent)
	}
	return delegatable
}

func isDelegatableAgent(agent string) bool {
	return agent != "" && agent != "coordinator" && !isRetiredWorkflowAgent(agent)
}

func isRetiredWorkflowAgent(agent string) bool {
	return agent == "stpa-analyst"
}

func resolveRetrospectiveDir(resuming bool, dir, retrospectivesBaseDir, pmPath string, _ string, goalPath string) string {
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

	if errUpdate := updateProjectManagementWithRetrospectiveDir(pmPath, retroDirRel); errUpdate != nil {
		log.Fatalln("failed to update PROJECT_MANAGEMENT.md with retrospective directory:", errUpdate)
	}

	goalRetrospectivePath := filepath.Join(retroDir, "GOAL.md")
	if errCopy := copyFileAtomic(goalPath, goalRetrospectivePath); errCopy != nil {
		log.Fatalln("failed to copy GOAL.md to retrospective:", errCopy)
	}

	return retroDir
}
