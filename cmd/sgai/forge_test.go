package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDetectForge(t *testing.T) {
	cases := []struct {
		name      string
		remoteURL string
		wantForge string
	}{
		{"github https", "https://github.com/user/repo.git", forgeGitHub},
		{"github ssh", "git@github.com:user/repo.git", forgeGitHub},
		{"gitlab https", "https://gitlab.com/user/repo.git", forgeGitLab},
		{"gitlab ssh", "git@gitlab.com:user/repo.git", forgeGitLab},
		{"bitbucket", "https://bitbucket.org/user/repo.git", forgeUnknown},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := setupGitRepoWithRemote(t, tc.remoteURL)
			fc := detectForge(dir)
			if fc.forgeType != tc.wantForge {
				t.Errorf("detectForge() forgeType = %q; want %q", fc.forgeType, tc.wantForge)
			}
		})
	}

	t.Run("no remote", func(t *testing.T) {
		dir := t.TempDir()
		fc := detectForge(dir)
		if fc.forgeType != forgeUnknown {
			t.Errorf("detectForge() forgeType = %q; want %q", fc.forgeType, forgeUnknown)
		}
	})
}

func TestContainsGitHubHost(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		expect bool
	}{
		{"https", "origin\thttps://github.com/user/repo.git (fetch)", true},
		{"ssh", "origin\tgit@github.com:user/repo.git (fetch)", true},
		{"no match", "origin\thttps://gitlab.com/user/repo.git (fetch)", false},
		{"empty", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsGitHubHost(tc.input)
			if got != tc.expect {
				t.Errorf("containsGitHubHost(%q) = %v; want %v", tc.input, got, tc.expect)
			}
		})
	}
}

func TestContainsGitLabHost(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		expect bool
	}{
		{"https", "origin\thttps://gitlab.com/user/repo.git (fetch)", true},
		{"ssh", "origin\tgit@gitlab.com:user/repo.git (fetch)", true},
		{"no match", "origin\thttps://github.com/user/repo.git (fetch)", false},
		{"empty", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsGitLabHost(tc.input)
			if got != tc.expect {
				t.Errorf("containsGitLabHost(%q) = %v; want %v", tc.input, got, tc.expect)
			}
		})
	}
}

func TestBuildCompletionGateChoices(t *testing.T) {
	cases := []struct {
		name        string
		fc          forgeCapability
		wantChoices []string
	}{
		{
			"github with cli",
			forgeCapability{forgeType: forgeGitHub, cliAvailable: true},
			[]string{choiceCreatePR, choiceContinueWork, choiceDone},
		},
		{
			"gitlab with cli",
			forgeCapability{forgeType: forgeGitLab, cliAvailable: true},
			[]string{choiceCreateMR, choiceContinueWork, choiceDone},
		},
		{
			"github without cli",
			forgeCapability{forgeType: forgeGitHub, cliAvailable: false},
			[]string{choiceContinueWork, choiceDone},
		},
		{
			"no forge",
			forgeCapability{forgeType: forgeUnknown},
			[]string{choiceContinueWork, choiceDone},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildCompletionGateChoices(tc.fc)
			if len(got) != len(tc.wantChoices) {
				t.Fatalf("got %d choices %v; want %d choices %v", len(got), got, len(tc.wantChoices), tc.wantChoices)
			}
			for i, want := range tc.wantChoices {
				if got[i] != want {
					t.Errorf("choice[%d] = %q; want %q", i, got[i], want)
				}
			}
		})
	}
}

func TestReadPRTemplate(t *testing.T) {
	cases := []struct {
		name     string
		setup    func(dir string) error
		wantBody string
	}{
		{
			"uppercase template in .github",
			func(dir string) error {
				githubDir := filepath.Join(dir, ".github")
				if err := os.MkdirAll(githubDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(githubDir, "PULL_REQUEST_TEMPLATE.md"),
					[]byte("## Description\n\nPlease describe your changes."), 0644)
			},
			"## Description\n\nPlease describe your changes.",
		},
		{
			"lowercase template in .github",
			func(dir string) error {
				githubDir := filepath.Join(dir, ".github")
				if err := os.MkdirAll(githubDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(githubDir, "pull_request_template.md"),
					[]byte("lowercase template"), 0644)
			},
			"lowercase template",
		},
		{
			"no template",
			func(_ string) error { return nil },
			"",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := tc.setup(dir); err != nil {
				t.Fatal(err)
			}
			got := readPRTemplate(dir)
			if got != tc.wantBody {
				t.Errorf("readPRTemplate() = %q; want %q", got, tc.wantBody)
			}
		})
	}
}

func TestExtractContributingGuidelines(t *testing.T) {
	cases := []struct {
		name     string
		setup    func(dir string) error
		wantBody string
	}{
		{
			"with contributing file",
			func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "CONTRIBUTING.md"),
					[]byte("# Contributing\n\nPlease follow these guidelines."), 0644)
			},
			"# Contributing\n\nPlease follow these guidelines.",
		},
		{
			"no contributing file",
			func(_ string) error { return nil },
			"",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := tc.setup(dir); err != nil {
				t.Fatal(err)
			}
			got := extractContributingGuidelines(dir)
			if got != tc.wantBody {
				t.Errorf("extractContributingGuidelines() = %q; want %q", got, tc.wantBody)
			}
		})
	}
}

func TestGeneratePRBody(t *testing.T) {
	cases := []struct {
		name      string
		setup     func(dir string) error
		wantExact string
		wantNot   string
		wantNE    bool
	}{
		{
			"with template",
			func(dir string) error {
				githubDir := filepath.Join(dir, ".github")
				if err := os.MkdirAll(githubDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(githubDir, "PULL_REQUEST_TEMPLATE.md"),
					[]byte("## PR Template\nDescribe changes."), 0644)
			},
			"## PR Template\nDescribe changes.",
			"",
			false,
		},
		{
			"with contributing",
			func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "CONTRIBUTING.md"),
					[]byte("# Contributing Guidelines"), 0644)
			},
			"# Contributing Guidelines",
			"",
			false,
		},
		{
			"fallback with goal",
			func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "GOAL.md"),
					[]byte("---\n---\nImplement the feature.\n"), 0644)
			},
			"",
			"Pull request created by sgai.",
			true,
		},
		{
			"generic fallback",
			func(_ string) error { return nil },
			"Pull request created by sgai.",
			"",
			false,
		},
		{
			"template takes priority over contributing",
			func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "CONTRIBUTING.md"),
					[]byte("contributing content"), 0644); err != nil {
					return err
				}
				githubDir := filepath.Join(dir, ".github")
				if err := os.MkdirAll(githubDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(githubDir, "PULL_REQUEST_TEMPLATE.md"),
					[]byte("template content"), 0644)
			},
			"template content",
			"",
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := tc.setup(dir); err != nil {
				t.Fatal(err)
			}
			got := generatePRBody(dir)
			if tc.wantExact != "" && got != tc.wantExact {
				t.Errorf("generatePRBody() = %q; want %q", got, tc.wantExact)
			}
			if tc.wantNE && got == "" {
				t.Error("expected non-empty fallback body")
			}
			if tc.wantNot != "" && got == tc.wantNot {
				t.Errorf("generatePRBody() = %q; want something other than %q", got, tc.wantNot)
			}
		})
	}
}

func TestReadGoalTitle(t *testing.T) {
	cases := []struct {
		name  string
		setup func(dir string) error
		want  string
	}{
		{
			"with goal file",
			func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "GOAL.md"),
					[]byte("---\nflow: |\n  a -> b\n---\nImplement dark mode toggle.\n\n## Acceptance Criteria\n"), 0644)
			},
			"Implement dark mode toggle.",
		},
		{
			"no goal file",
			func(_ string) error { return nil },
			"",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := tc.setup(dir); err != nil {
				t.Fatal(err)
			}
			got := readGoalTitle(dir)
			if got != tc.want {
				t.Errorf("readGoalTitle() = %q; want %q", got, tc.want)
			}
		})
	}
}

func TestExtractPRTitle(t *testing.T) {
	cases := []struct {
		name  string
		setup func(dir string) error
		want  string
	}{
		{
			"with goal file",
			func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "GOAL.md"),
					[]byte("---\n---\nAdd authentication system.\n"), 0644)
			},
			"Add authentication system.",
		},
		{
			"fallback",
			func(_ string) error { return nil },
			"sgai: automated changes",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := tc.setup(dir); err != nil {
				t.Fatal(err)
			}
			got := extractPRTitle(dir)
			if got != tc.want {
				t.Errorf("extractPRTitle() = %q; want %q", got, tc.want)
			}
		})
	}
}

func TestDispatchCompletionGateResponse(t *testing.T) {
	cases := []struct {
		name         string
		response     string
		wantContinue bool
	}{
		{"continue working", "Selected: Continue working", true},
		{"done", "Selected: Done", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			result := dispatchCompletionGateResponse(dir, tc.response, "test")
			if result.continueWorking != tc.wantContinue {
				t.Errorf("dispatchCompletionGateResponse(%q) continueWorking = %v; want %v",
					tc.response, result.continueWorking, tc.wantContinue)
			}
		})
	}
}

func TestResolveRootDirStandalone(t *testing.T) {
	dir := t.TempDir()
	got := resolveRootDir(dir)
	if got != dir {
		t.Errorf("resolveRootDir() = %q; want %q for standalone", got, dir)
	}
}

func setupGitRepoWithRemote(t *testing.T, remoteURL string) string {
	t.Helper()
	dir := t.TempDir()
	initCmd := exec.Command("git", "init")
	initCmd.Dir = dir
	if errInit := initCmd.Run(); errInit != nil {
		t.Fatalf("git init failed: %v", errInit)
	}
	addCmd := exec.Command("git", "remote", "add", "origin", remoteURL)
	addCmd.Dir = dir
	if errAdd := addCmd.Run(); errAdd != nil {
		t.Fatalf("git remote add failed: %v", errAdd)
	}
	return dir
}
