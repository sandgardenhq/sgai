package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

func dotSGAILinePresent(content []byte) bool {
	for line := range bytes.SplitSeq(content, []byte("\n")) {
		if bytes.Equal(bytes.TrimSpace(line), []byte("/.sgai")) {
			return true
		}
	}
	return false
}

func initializeWorkspaceDir(dir string) error {
	if errUnpack := unpackSkeleton(dir); errUnpack != nil {
		return fmt.Errorf("failed to unpack skeleton: %w", errUnpack)
	}

	if errOverlay := applyLayerFolderOverlay(dir); errOverlay != nil {
		return fmt.Errorf("failed to apply layer overlay: %w", errOverlay)
	}

	if errInit := initializeJJ(dir); errInit != nil {
		return fmt.Errorf("failed to initialize jj: %w", errInit)
	}

	if errExclude := addGitExclude(dir); errExclude != nil {
		return fmt.Errorf("failed to add git exclude: %w", errExclude)
	}

	return nil
}

func initializeJJ(dir string) error {
	if classifyWorkspace(dir) == workspaceFork {
		return nil
	}
	cmd := exec.Command("jj", "status")
	cmd.Dir = dir
	if errRun := cmd.Run(); errRun != nil {
		if isExecNotFound(errRun) {
			return fmt.Errorf("jj is required but not found in PATH")
		}
		initCmd := exec.Command("jj", "git", "init", "--colocate")
		initCmd.Dir = dir
		if errInit := initCmd.Run(); errInit != nil {
			return fmt.Errorf("failed to run jj git init: %w", errInit)
		}
	}
	return nil
}

func isExecNotFound(err error) bool {
	var errExec *exec.Error
	if errors.As(err, &errExec) {
		return errors.Is(errExec.Err, exec.ErrNotFound)
	}
	return false
}

func applyLayerFolderOverlay(dir string) error {
	layerDir := filepath.Join(dir, "sgai")
	if !isExistingDirectory(layerDir) {
		return nil
	}

	allowedSubfolders := []string{"agent", "skills", "snippets"}
	for _, subfolder := range allowedSubfolders {
		srcDir := filepath.Join(layerDir, subfolder)
		dstDir := filepath.Join(dir, ".sgai", subfolder)
		if errCopy := copyLayerSubfolder(srcDir, dstDir); errCopy != nil {
			return errCopy
		}
	}

	return nil
}

func copyLayerSubfolder(srcDir, dstDir string) error {
	if !isExistingDirectory(srcDir) {
		return nil
	}

	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, errRel := filepath.Rel(srcDir, path)
		if errRel != nil {
			return errRel
		}

		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dstDir, relPath), 0755)
		}

		return copyFileAtomic(path, filepath.Join(dstDir, relPath))
	})
}

func isExistingDirectory(path string) bool {
	fi, errStat := os.Stat(path)
	if errStat != nil {
		return false
	}
	return fi.IsDir()
}
