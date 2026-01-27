package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestEnsureJJ(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not found in PATH, skipping integration tests")
	}

	t.Run("initializesJJInGitRepo", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		ensureJJ(dir)

		verifyJJWorks(t, dir)
	})

	t.Run("idempotentWhenAlreadyInitialized", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)
		initJJRepo(t, dir)

		ensureJJ(dir)

		verifyJJWorks(t, dir)
	})

	t.Run("succeedsWithExistingJJRepo", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)
		initJJRepo(t, dir)
		createJJCommit(t, dir)

		ensureJJ(dir)

		verifyJJWorks(t, dir)
	})

	t.Run("skipsForForkWorkspace", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)
		initJJRepo(t, dir)
		forkDir := filepath.Join(dir, "fork-workspace")
		createForkWorkspace(t, dir, forkDir)

		ensureJJ(forkDir)

		verifyJJWorks(t, forkDir)
	})
}

func TestIsExecNotFound(t *testing.T) {
	t.Run("returnsTrueForMissingBinary", func(t *testing.T) {
		cmd := exec.Command("nonexistent-binary-xyz-12345")
		err := cmd.Run()
		if !isExecNotFound(err) {
			t.Errorf("expected isExecNotFound to return true for missing binary, got false")
		}
	})

	t.Run("returnsFalseForExitError", func(t *testing.T) {
		cmd := exec.Command("sh", "-c", "exit 1")
		err := cmd.Run()
		if isExecNotFound(err) {
			t.Errorf("expected isExecNotFound to return false for exit error, got true")
		}
	})

	t.Run("returnsFalseForNilError", func(t *testing.T) {
		if isExecNotFound(nil) {
			t.Errorf("expected isExecNotFound to return false for nil error, got true")
		}
	})

	t.Run("returnsFalseForGenericError", func(t *testing.T) {
		if isExecNotFound(errors.New("some error")) {
			t.Errorf("expected isExecNotFound to return false for generic error, got true")
		}
	})
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to init git repo: %v\n%s", err, output)
	}
}

func initJJRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("jj", "git", "init", "--colocate")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to init jj repo: %v\n%s", err, output)
	}
}

func createJJCommit(t *testing.T, dir string) {
	t.Helper()
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("jj", "commit", "-m", "test commit")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create jj commit: %v\n%s", err, output)
	}
}

func createForkWorkspace(t *testing.T, rootDir, forkDir string) {
	t.Helper()
	cmd := exec.Command("jj", "workspace", "add", forkDir)
	cmd.Dir = rootDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create fork workspace: %v\n%s", err, output)
	}
}

func verifyJJWorks(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("jj", "status")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("jj status failed after ensureJJ: %v\n%s", err, output)
	}
}
