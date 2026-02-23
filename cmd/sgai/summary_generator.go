package main

import (
	"bytes"
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

type summaryGenerator struct {
	mu          sync.Mutex
	debounceMap map[string]*summaryDebouncer
	shutdownCtx context.Context
	server      *Server
}

type summaryDebouncer struct {
	mu       sync.Mutex
	timer    *time.Timer
	cancelFn context.CancelFunc
}

func newSummaryGenerator(shutdownCtx context.Context, srv *Server) *summaryGenerator {
	return &summaryGenerator{
		debounceMap: make(map[string]*summaryDebouncer),
		shutdownCtx: shutdownCtx,
		server:      srv,
	}
}

func (g *summaryGenerator) trigger(workspacePath string) {
	g.mu.Lock()
	d, ok := g.debounceMap[workspacePath]
	if !ok {
		d = &summaryDebouncer{}
		g.debounceMap[workspacePath] = d
	}
	g.mu.Unlock()

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.cancelFn != nil {
		d.cancelFn()
		d.cancelFn = nil
	}

	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(2*time.Second, func() {
		g.runGeneration(workspacePath)
	})
}

func (g *summaryGenerator) runGeneration(workspacePath string) {
	g.mu.Lock()
	d := g.debounceMap[workspacePath]
	g.mu.Unlock()

	if d == nil {
		return
	}

	ctx, cancel := context.WithCancel(g.shutdownCtx)
	d.mu.Lock()
	d.cancelFn = cancel
	d.mu.Unlock()

	defer cancel()

	goalBody := readGoalBody(workspacePath)
	if goalBody == "" {
		return
	}

	summary := generateSummaryViaOpenCode(ctx, workspacePath, goalBody)
	if summary == "" {
		return
	}

	saveSummaryIfNotManual(workspacePath, summary)

	g.server.publishGlobalAndWorkspace(filepath.Base(workspacePath), workspacePath, sseEvent{
		Type: "session:update",
		Data: map[string]string{"workspace": filepath.Base(workspacePath)},
	})
}

func (g *summaryGenerator) stop() {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, d := range g.debounceMap {
		d.mu.Lock()
		if d.timer != nil {
			d.timer.Stop()
		}
		if d.cancelFn != nil {
			d.cancelFn()
		}
		d.mu.Unlock()
	}
}

func readGoalBody(workspacePath string) string {
	data, errRead := os.ReadFile(filepath.Join(workspacePath, "GOAL.md"))
	if errRead != nil {
		return ""
	}
	body := extractBody(data)
	return strings.TrimSpace(string(body))
}

func generateSummaryViaOpenCode(ctx context.Context, workspacePath, goalBody string) string {
	modelSpecs := modelsForAgentFromGoal(workspacePath, "coordinator")
	if len(modelSpecs) == 0 {
		return ""
	}

	modelID, variant := parseModelAndVariant(modelSpecs[0])

	args := []string{"run", "-m", modelID}
	if variant != "" {
		args = append(args, "--variant", variant)
	}
	args = append(args, "--agent", "build", "--title", "summary")

	prompt := "Read this GOAL.md and produce a single English sentence summarizing the project goal. Output ONLY the shortest summary sentence you can, nothing else.\n\n" + goalBody

	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Dir = workspacePath
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = append(os.Environ(), "OPENCODE_CONFIG_DIR="+filepath.Join(workspacePath, ".sgai"))
	cmd.Stdin = strings.NewReader(prompt)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if errRun := cmd.Run(); errRun != nil {
		log.Println("summary generation failed:", errRun)
		return ""
	}

	return cleanSummaryOutput(stdout.String())
}

func cleanSummaryOutput(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.Trim(trimmed, "\"'`")
	trimmed = strings.TrimSpace(trimmed)
	return trimmed
}

func saveSummaryIfNotManual(workspacePath, summary string) {
	wfState, errLoad := state.Load(statePath(workspacePath))
	if errLoad != nil {
		return
	}
	if wfState.SummaryManual {
		return
	}
	wfState.Summary = summary
	if errSave := state.Save(statePath(workspacePath), wfState); errSave != nil {
		log.Println("failed to save summary:", errSave)
	}
}
