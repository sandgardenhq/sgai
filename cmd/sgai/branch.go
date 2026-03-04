package main

import (
	"context"
	"io"

	"github.com/sandgardenhq/sgai/pkg/state"
)

type workflowBranch interface {
	run(ctx context.Context, cfg branchConfig)
	toolsAllowed() bool
}

type branchConfig struct {
	workspacePath string
	mcpURL        string
	logWriter     io.Writer
	coord         *state.Coordinator
}
