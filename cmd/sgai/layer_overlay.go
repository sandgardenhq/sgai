package main

import (
	"io/fs"
	"os"
	"path/filepath"
)

func applyLayerFolderOverlay(dir string) error {
	layerDir := filepath.Join(dir, "sgai")
	if !isExistingDirectory(layerDir) {
		return nil
	}

	allowedSubfolders := []string{"agent", "skills", "snippets"}
	for _, subfolder := range allowedSubfolders {
		srcDir := filepath.Join(layerDir, subfolder)
		dstDir := filepath.Join(dir, ".sgai", subfolder)
		if err := copyLayerSubfolder(srcDir, dstDir, subfolder); err != nil {
			return err
		}
	}

	return nil
}

func copyLayerSubfolder(srcDir, dstDir, subfolder string) error {
	if !isExistingDirectory(srcDir) {
		return nil
	}

	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dstDir, relPath), 0755)
		}

		if isProtectedFile(subfolder, relPath) {
			return nil
		}

		return copyFileAtomic(path, filepath.Join(dstDir, relPath))
	})
}

func isExistingDirectory(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.IsDir()
}

func isProtectedFile(subfolder, relPath string) bool {
	return subfolder == "agent" && relPath == "coordinator.md"
}
