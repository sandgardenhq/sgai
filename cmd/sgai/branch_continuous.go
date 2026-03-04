package main

import "context"

type continuousBranch struct{}

func (b *continuousBranch) run(ctx context.Context, cfg branchConfig) {
	continuousPrompt := readContinuousModePrompt(cfg.workspacePath)
	runContinuousWorkflow(ctx, cfg.workspacePath, continuousPrompt, cfg.mcpURL, cfg.logWriter, cfg.coord)
}

func (b *continuousBranch) toolsAllowed() bool {
	return false
}
