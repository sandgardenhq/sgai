package main

import "context"

type continuousBranch struct{}

func (b *continuousBranch) run(ctx context.Context, cfg branchConfig) {
	continuousPrompt := readContinuousModePrompt(cfg.workspacePath)
	runContinuousWorkflow(ctx, []string{cfg.workspacePath}, continuousPrompt, cfg.mcpURL, cfg.logWriter, cfg.coord)
}

func (b *continuousBranch) promptSection() string {
	return flowSectionContinuousMode
}

func (b *continuousBranch) coordinatorPlan() string {
	return flowSectionContinuousModeCoordinator
}

func (b *continuousBranch) toolsAllowed() bool {
	return false
}
