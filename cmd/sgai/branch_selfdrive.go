package main

import "context"

type selfDriveBranch struct{}

func (b *selfDriveBranch) run(ctx context.Context, cfg branchConfig) {
	runWorkflow(ctx, []string{cfg.workspacePath}, cfg.mcpURL, cfg.logWriter)
}

func (b *selfDriveBranch) promptSection() string {
	return flowSectionSelfDriveMode
}

func (b *selfDriveBranch) coordinatorPlan() string {
	return flowSectionSelfDriveModeCoordinator
}

func (b *selfDriveBranch) toolsAllowed() bool {
	return false
}
