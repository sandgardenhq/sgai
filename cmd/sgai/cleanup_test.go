package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFileAtomic(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	content := []byte("test content")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyFileAtomic(srcPath, dstPath); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(result) != string(content) {
		t.Errorf("expected %q, got %q", content, result)
	}

	tmpFile := dstPath + ".tmp"
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("temporary file was not cleaned up")
	}
}

func TestCopyFileAtomicWithSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "subdir", "dest.txt")

	content := []byte("test content")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyFileAtomic(srcPath, dstPath); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(result) != string(content) {
		t.Errorf("expected %q, got %q", content, result)
	}
}
