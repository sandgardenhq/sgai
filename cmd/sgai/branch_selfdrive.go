package main

import "context"

type selfDriveBranch struct{}

func (b *selfDriveBranch) run(ctx context.Context, cfg branchConfig) {
	runWorkflow(ctx, cfg.workspacePath, cfg.mcpURL, cfg.logWriter, cfg.coord)
}

func (b *selfDriveBranch) toolsAllowed() bool {
	return false
}
