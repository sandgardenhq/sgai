package main

import "context"

type interactiveBranch struct{}

func (b *interactiveBranch) run(ctx context.Context, cfg branchConfig) {
	runWorkflow(ctx, cfg.workspacePath, cfg.mcpURL, cfg.logWriter, cfg.coord)
}

func (b *interactiveBranch) toolsAllowed() bool {
	return true
}
