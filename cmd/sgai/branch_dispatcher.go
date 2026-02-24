package main

import "github.com/sandgardenhq/sgai/pkg/state"

func dispatchBranch(mode string) workflowBranch {
	switch mode {
	case state.ModeContinuous:
		return &continuousBranch{}
	case state.ModeSelfDrive:
		return &selfDriveBranch{}
	default:
		return &interactiveBranch{}
	}
}
