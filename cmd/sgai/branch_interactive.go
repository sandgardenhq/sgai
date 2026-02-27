package main

import "context"

type interactiveBranch struct{}

func (b *interactiveBranch) run(ctx context.Context, cfg branchConfig) {
	runWorkflow(ctx, []string{cfg.workspacePath}, cfg.mcpURL, cfg.logWriter, cfg.coord)
}

func (b *interactiveBranch) promptSection() string {
	return flowSectionBrainstormingMode
}

func (b *interactiveBranch) coordinatorPlan() string {
	return ""
}

func (b *interactiveBranch) toolsAllowed() bool {
	return true
}
