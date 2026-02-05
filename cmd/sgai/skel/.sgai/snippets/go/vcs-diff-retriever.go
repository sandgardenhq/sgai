---
name: VCS Diff Retriever
description: Retrieves version control diffs, trying jj first then falling back to git; When you need to get diff output from a version control system with jj/git fallback
---

// Package main provides a utility to retrieve version control diffs with jj and git fallback.
package main

import (
	"fmt"
	"os/exec"
)

func vcsDiff(dir string) (diff, vcs string, ok bool) {
	diffFromJJ, errJJ := tryJJDiff(dir)
	if errJJ == nil && len(diffFromJJ) > 0 {
		return diffFromJJ, "jj", true
	}

	diffFromGitHead, errGitHead := tryGitDiffHead(dir)
	if errGitHead == nil && len(diffFromGitHead) > 0 {
		return diffFromGitHead, "git", true
	}

	diffFromGitUncommitted, errGitUncommitted := tryGitDiffUncommitted(dir)
	if errGitUncommitted == nil && len(diffFromGitUncommitted) > 0 {
		return diffFromGitUncommitted, "git", true
	}

	return "", "", false
}

func tryJJDiff(dir string) (string, error) {
	cmd := exec.Command("jj", "diff", "--git")
	cmd.Dir = dir
	output, err := cmd.Output()
	return string(output), err
}

func tryGitDiffHead(dir string) (string, error) {
	cmd := exec.Command("git", "diff", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	return string(output), err
}

func tryGitDiffUncommitted(dir string) (string, error) {
	cmd := exec.Command("git", "diff")
	cmd.Dir = dir
	output, err := cmd.Output()
	return string(output), err
}

func main() {
	targetDir := "."

	diff, vcs, ok := vcsDiff(targetDir)
	if !ok {
		fmt.Println("No diff found or no VCS system detected")
		return
	}

	fmt.Printf("Diff from %s:\n", vcs)
	fmt.Println(diff)
}
