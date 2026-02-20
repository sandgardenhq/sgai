package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sandgardenhq/sgai/pkg/state"
)

const (
	forgeGitHub  = "github"
	forgeGitLab  = "gitlab"
	forgeUnknown = "none"

	choiceCreatePR       = "Create Pull Request"
	choiceCreateMR       = "Create Merge Request"
	choiceContinueWork   = "Continue working"
	choiceDone           = "Done"
	completionGatePrompt = "The workflow is complete. What would you like to do next?"
)

type forgeCapability struct {
	forgeType    string
	cliAvailable bool
}

func detectForge(dir string) forgeCapability {
	remoteURL := readGitRemoteURL(dir)
	if remoteURL == "" {
		return forgeCapability{forgeType: forgeUnknown}
	}
	if containsGitHubHost(remoteURL) {
		return forgeCapability{
			forgeType:    forgeGitHub,
			cliAvailable: isCommandAvailable("gh"),
		}
	}
	if containsGitLabHost(remoteURL) {
		return forgeCapability{
			forgeType:    forgeGitLab,
			cliAvailable: isCommandAvailable("glab"),
		}
	}
	return forgeCapability{forgeType: forgeUnknown}
}

func readGitRemoteURL(dir string) string {
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = dir
	output, errCmd := cmd.Output()
	if errCmd != nil {
		return ""
	}
	return string(output)
}

func containsGitHubHost(remoteOutput string) bool {
	return strings.Contains(remoteOutput, "github.com")
}

func containsGitLabHost(remoteOutput string) bool {
	return strings.Contains(remoteOutput, "gitlab.com")
}

func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func buildCompletionGateChoices(fc forgeCapability) []string {
	var choices []string
	if fc.forgeType == forgeGitHub && fc.cliAvailable {
		choices = append(choices, choiceCreatePR)
	}
	if fc.forgeType == forgeGitLab && fc.cliAvailable {
		choices = append(choices, choiceCreateMR)
	}
	choices = append(choices, choiceContinueWork, choiceDone)
	return choices
}

func pushBookmark(dir, workspaceName string) error {
	bookmarkName := "sgai/" + workspaceName
	setCmd := exec.Command("jj", "bookmark", "set", bookmarkName, "--allow-backwards")
	setCmd.Dir = dir
	if errSet := setCmd.Run(); errSet != nil {
		return fmt.Errorf("setting bookmark %s: %w", bookmarkName, errSet)
	}
	pushCmd := exec.Command("jj", "git", "push", "--bookmark", bookmarkName)
	pushCmd.Dir = dir
	if errPush := pushCmd.Run(); errPush != nil {
		return fmt.Errorf("pushing bookmark %s: %w", bookmarkName, errPush)
	}
	return nil
}

func generatePRBody(dir string) string {
	if body := readPRTemplate(dir); body != "" {
		return body
	}
	if body := extractContributingGuidelines(dir); body != "" {
		return body
	}
	return generateFallbackPRBody(dir)
}

func readPRTemplate(dir string) string {
	candidates := []string{
		filepath.Join(dir, ".github", "PULL_REQUEST_TEMPLATE.md"),
		filepath.Join(dir, ".github", "pull_request_template.md"),
		filepath.Join(dir, "PULL_REQUEST_TEMPLATE.md"),
		filepath.Join(dir, "pull_request_template.md"),
	}
	for _, path := range candidates {
		content, errRead := os.ReadFile(path)
		if errRead == nil && len(content) > 0 {
			return string(content)
		}
	}
	return ""
}

func extractContributingGuidelines(dir string) string {
	candidates := []string{
		filepath.Join(dir, "CONTRIBUTING.md"),
		filepath.Join(dir, "contributing.md"),
	}
	for _, path := range candidates {
		content, errRead := os.ReadFile(path)
		if errRead == nil && len(content) > 0 {
			return string(content)
		}
	}
	return ""
}

func generateFallbackPRBody(dir string) string {
	var sections []string
	if title := readGoalTitle(dir); title != "" {
		sections = append(sections, "## Goal\n\n"+title)
	}
	if logOutput := runJJLog(dir); logOutput != "" {
		sections = append(sections, "## Changes\n\n```\n"+logOutput+"\n```")
	}
	if diffStat := runJJDiffStat(dir); diffStat != "" {
		sections = append(sections, "## Diff Summary\n\n```\n"+diffStat+"\n```")
	}
	if len(sections) == 0 {
		return "Pull request created by sgai."
	}
	return strings.Join(sections, "\n\n")
}

func readGoalTitle(dir string) string {
	goalPath := filepath.Join(dir, "GOAL.md")
	content, errRead := os.ReadFile(goalPath)
	if errRead != nil {
		return ""
	}
	body := extractBody(content)
	for line := range strings.SplitSeq(strings.TrimSpace(string(body)), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func runJJLog(dir string) string {
	cmd := exec.Command("jj", "log", "--no-graph", "-r", "..@", "-T",
		`change_id.short(8) ++ " " ++ coalesce(description.first_line(), "(no description)") ++ "\n"`)
	cmd.Dir = dir
	output, errCmd := cmd.Output()
	if errCmd != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func runJJDiffStat(dir string) string {
	cmd := exec.Command("jj", "diff", "--stat")
	cmd.Dir = dir
	output, errCmd := cmd.Output()
	if errCmd != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func extractPRTitle(dir string) string {
	title := readGoalTitle(dir)
	if title == "" {
		return "sgai: automated changes"
	}
	return title
}

func createGitHubPR(dir, workspaceName, baseBranch string) error {
	title := extractPRTitle(dir)
	body := generatePRBody(dir)
	head := "sgai/" + workspaceName
	cmd := exec.Command("gh", "pr", "create",
		"--title", title,
		"--body", body,
		"--head", head,
		"--base", baseBranch,
	)
	cmd.Dir = dir
	output, errCmd := cmd.CombinedOutput()
	if errCmd != nil {
		return fmt.Errorf("gh pr create: %s: %w", strings.TrimSpace(string(output)), errCmd)
	}
	return nil
}

func createGitLabMR(dir, workspaceName, baseBranch string) error {
	title := extractPRTitle(dir)
	body := generatePRBody(dir)
	source := "sgai/" + workspaceName
	cmd := exec.Command("glab", "mr", "create",
		"--title", title,
		"--description", body,
		"--source-branch", source,
		"--target-branch", baseBranch,
	)
	cmd.Dir = dir
	output, errCmd := cmd.CombinedOutput()
	if errCmd != nil {
		return fmt.Errorf("glab mr create: %s: %w", strings.TrimSpace(string(output)), errCmd)
	}
	return nil
}

type completionGateResult struct {
	continueWorking bool
}

func handleCompletionGate(ctx context.Context, dir, stateJSONPath string, wfState state.Workflow, paddedsgai string) completionGateResult {
	fc := detectForge(dir)
	choices := buildCompletionGateChoices(fc)

	wfState.MultiChoiceQuestion = &state.MultiChoiceQuestion{
		Questions: []state.QuestionItem{
			{
				Question:    completionGatePrompt,
				Choices:     choices,
				MultiSelect: false,
			},
		},
	}
	wfState.HumanMessage = completionGatePrompt
	wfState.Status = state.StatusWaitingForHuman

	if errSave := state.Save(stateJSONPath, wfState); errSave != nil {
		log.Println("failed to save completion gate state:", errSave)
		return completionGateResult{}
	}

	fmt.Println("["+paddedsgai+"]", "waiting for completion gate response...")

	humanResponse, cancelled := waitForStateTransition(ctx, dir, stateJSONPath)
	if cancelled {
		return completionGateResult{}
	}

	return dispatchCompletionGateResponse(dir, humanResponse, paddedsgai)
}

func dispatchCompletionGateResponse(dir, response, paddedsgai string) completionGateResult {
	switch {
	case strings.Contains(response, choiceCreatePR):
		executeForgeCreation(dir, forgeGitHub, paddedsgai)
		return completionGateResult{}
	case strings.Contains(response, choiceCreateMR):
		executeForgeCreation(dir, forgeGitLab, paddedsgai)
		return completionGateResult{}
	case strings.Contains(response, choiceContinueWork):
		fmt.Println("["+paddedsgai+"]", "user chose to continue working")
		return completionGateResult{continueWorking: true}
	default:
		fmt.Println("["+paddedsgai+"]", "workflow complete")
		return completionGateResult{}
	}
}

func executeForgeCreation(dir, targetForge, paddedsgai string) {
	workspaceName := filepath.Base(dir)
	rootDir := resolveRootDir(dir)
	baseBranch := resolveBaseBookmark(rootDir)

	fmt.Println("["+paddedsgai+"]", "pushing bookmark sgai/"+workspaceName+"...")
	if errPush := pushBookmark(dir, workspaceName); errPush != nil {
		fmt.Println("["+paddedsgai+"]", "failed to push bookmark:", errPush)
		fmt.Println("["+paddedsgai+"]", "you can push manually and create the PR/MR yourself")
		return
	}

	switch targetForge {
	case forgeGitHub:
		fmt.Println("["+paddedsgai+"]", "creating GitHub pull request...")
		if errPR := createGitHubPR(dir, workspaceName, baseBranch); errPR != nil {
			fmt.Println("["+paddedsgai+"]", "failed to create pull request:", errPR)
			fmt.Println("["+paddedsgai+"]", "bookmark was pushed - you can create the PR manually")
			return
		}
		fmt.Println("["+paddedsgai+"]", "pull request created successfully")
	case forgeGitLab:
		fmt.Println("["+paddedsgai+"]", "creating GitLab merge request...")
		if errMR := createGitLabMR(dir, workspaceName, baseBranch); errMR != nil {
			fmt.Println("["+paddedsgai+"]", "failed to create merge request:", errMR)
			fmt.Println("["+paddedsgai+"]", "bookmark was pushed - you can create the MR manually")
			return
		}
		fmt.Println("["+paddedsgai+"]", "merge request created successfully")
	}
}

func resolveRootDir(dir string) string {
	if classifyWorkspace(dir) == workspaceFork {
		if root := getRootWorkspacePath(dir); root != "" {
			return root
		}
	}
	return dir
}
