package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureGitExclude(t *testing.T) {
	t.Run("addsEntryToNewExcludeFile", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		gitInfoDir := filepath.Join(gitDir, "info")
		if err := os.MkdirAll(gitInfoDir, 0755); err != nil {
			t.Fatal(err)
		}

		ensureGitExclude(dir)

		excludePath := filepath.Join(gitInfoDir, "exclude")
		content, err := os.ReadFile(excludePath)
		if err != nil {
			t.Fatalf("expected exclude file to exist: %v", err)
		}
		if !strings.Contains(string(content), "/.sgai\n") {
			t.Errorf("expected /.sgai entry in exclude file, got: %q", string(content))
		}
	})

	t.Run("addsEntryToExistingExcludeFile", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		gitInfoDir := filepath.Join(gitDir, "info")
		if err := os.MkdirAll(gitInfoDir, 0755); err != nil {
			t.Fatal(err)
		}
		excludePath := filepath.Join(gitInfoDir, "exclude")
		existingContent := "*.log\n*.tmp\n"
		if err := os.WriteFile(excludePath, []byte(existingContent), 0644); err != nil {
			t.Fatal(err)
		}

		ensureGitExclude(dir)

		content, err := os.ReadFile(excludePath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), existingContent) {
			t.Errorf("expected existing content preserved, got: %q", string(content))
		}
		if !strings.Contains(string(content), "/.sgai\n") {
			t.Errorf("expected /.sgai entry appended, got: %q", string(content))
		}
	})

	t.Run("idempotentWhenEntryExists", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		gitInfoDir := filepath.Join(gitDir, "info")
		if err := os.MkdirAll(gitInfoDir, 0755); err != nil {
			t.Fatal(err)
		}
		excludePath := filepath.Join(gitInfoDir, "exclude")
		existingContent := "*.log\n/.sgai\n*.tmp\n"
		if err := os.WriteFile(excludePath, []byte(existingContent), 0644); err != nil {
			t.Fatal(err)
		}

		ensureGitExclude(dir)

		content, err := os.ReadFile(excludePath)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != existingContent {
			t.Errorf("expected content unchanged, got: %q, want: %q", string(content), existingContent)
		}
	})

	t.Run("skipsNonGitRepository", func(t *testing.T) {
		dir := t.TempDir()

		ensureGitExclude(dir)

		excludePath := filepath.Join(dir, ".git", "info", "exclude")
		if _, err := os.Stat(excludePath); !os.IsNotExist(err) {
			t.Errorf("expected exclude file to not exist for non-git repo")
		}
	})

	t.Run("createsInfoDirectoryIfMissing", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, ".git")
		if err := os.MkdirAll(gitDir, 0755); err != nil {
			t.Fatal(err)
		}

		ensureGitExclude(dir)

		excludePath := filepath.Join(gitDir, "info", "exclude")
		content, err := os.ReadFile(excludePath)
		if err != nil {
			t.Fatalf("expected exclude file to exist: %v", err)
		}
		if !strings.Contains(string(content), "/.sgai\n") {
			t.Errorf("expected /.sgai entry in exclude file, got: %q", string(content))
		}
	})
}
