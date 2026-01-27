package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyLayerFolderOverlay_NosgaiDir(t *testing.T) {
	tmpDir := t.TempDir()
	factoraDir := filepath.Join(tmpDir, ".sgai")
	if err := os.MkdirAll(factoraDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := applyLayerFolderOverlay(tmpDir); err != nil {
		t.Fatalf("expected no error when sgai/ does not exist, got: %v", err)
	}
}

func TestApplyLayerFolderOverlay_sgaiIsFile(t *testing.T) {
	tmpDir := t.TempDir()
	factoraDir := filepath.Join(tmpDir, ".sgai")
	if err := os.MkdirAll(factoraDir, 0755); err != nil {
		t.Fatal(err)
	}

	layerPath := filepath.Join(tmpDir, "sgai")
	if err := os.WriteFile(layerPath, []byte("this is a file, not a directory"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := applyLayerFolderOverlay(tmpDir); err != nil {
		t.Fatalf("expected no error when sgai is a file, got: %v", err)
	}
}

func TestApplyLayerFolderOverlay_CopiesAgentFiles(t *testing.T) {
	tmpDir := t.TempDir()
	factoraDir := filepath.Join(tmpDir, ".sgai", "agent")
	layerDir := filepath.Join(tmpDir, "sgai", "agent")

	if err := os.MkdirAll(factoraDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(layerDir, 0755); err != nil {
		t.Fatal(err)
	}

	skelContent := []byte("skel agent content")
	if err := os.WriteFile(filepath.Join(factoraDir, "custom.md"), skelContent, 0644); err != nil {
		t.Fatal(err)
	}

	layerContent := []byte("layer agent content")
	if err := os.WriteFile(filepath.Join(layerDir, "custom.md"), layerContent, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(layerDir, "newagent.md"), []byte("new agent"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := applyLayerFolderOverlay(tmpDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(factoraDir, "custom.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "layer agent content" {
		t.Errorf("expected layer content to overwrite skel, got: %s", content)
	}

	content, err = os.ReadFile(filepath.Join(factoraDir, "newagent.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new agent" {
		t.Errorf("expected new agent file to be copied, got: %s", content)
	}
}

func TestApplyLayerFolderOverlay_ProtectsCoordinatorMD(t *testing.T) {
	tmpDir := t.TempDir()
	factoraDir := filepath.Join(tmpDir, ".sgai", "agent")
	layerDir := filepath.Join(tmpDir, "sgai", "agent")

	if err := os.MkdirAll(factoraDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(layerDir, 0755); err != nil {
		t.Fatal(err)
	}

	skelContent := []byte("skel coordinator - PROTECTED")
	if err := os.WriteFile(filepath.Join(factoraDir, "coordinator.md"), skelContent, 0644); err != nil {
		t.Fatal(err)
	}

	layerContent := []byte("evil coordinator replacement")
	if err := os.WriteFile(filepath.Join(layerDir, "coordinator.md"), layerContent, 0644); err != nil {
		t.Fatal(err)
	}

	if err := applyLayerFolderOverlay(tmpDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(factoraDir, "coordinator.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "skel coordinator - PROTECTED" {
		t.Errorf("coordinator.md should be protected, got: %s", content)
	}
}

func TestApplyLayerFolderOverlay_CopiesSubfolderFiles(t *testing.T) {
	cases := []struct {
		name       string
		subfolder  string
		srcFile    string
		dstContent string
	}{
		{
			name:       "skillsSubfolder",
			subfolder:  "skills",
			srcFile:    "custom-skill.md",
			dstContent: "skill content for custom-skill",
		},
		{
			name:       "snippetsSubfolder",
			subfolder:  "snippets",
			srcFile:    "go.md",
			dstContent: "go snippet content here",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			dstDir := filepath.Join(tmpDir, ".sgai", tc.subfolder)
			layerDir := filepath.Join(tmpDir, "sgai", tc.subfolder)

			if err := os.MkdirAll(dstDir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(layerDir, 0755); err != nil {
				t.Fatal(err)
			}

			if err := os.WriteFile(filepath.Join(layerDir, tc.srcFile), []byte(tc.dstContent), 0644); err != nil {
				t.Fatal(err)
			}

			if err := applyLayerFolderOverlay(tmpDir); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			content, err := os.ReadFile(filepath.Join(dstDir, tc.srcFile))
			if err != nil {
				t.Fatal(err)
			}
			if string(content) != tc.dstContent {
				t.Errorf("expected content %q, got: %s", tc.dstContent, content)
			}
		})
	}
}

func TestApplyLayerFolderOverlay_IgnoresOtherFolders(t *testing.T) {
	tmpDir := t.TempDir()
	factoraDir := filepath.Join(tmpDir, ".sgai")
	layerDir := filepath.Join(tmpDir, "sgai", "other")

	if err := os.MkdirAll(factoraDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(layerDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(layerDir, "file.txt"), []byte("ignored"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := applyLayerFolderOverlay(tmpDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	otherPath := filepath.Join(factoraDir, "other", "file.txt")
	if _, err := os.Stat(otherPath); !os.IsNotExist(err) {
		t.Errorf("files in other/ should be ignored, but %s exists", otherPath)
	}
}

func TestApplyLayerFolderOverlay_CopiesNestedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".sgai", "skills")
	layerDir := filepath.Join(tmpDir, "sgai", "skills", "coding-practices")

	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(layerDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(layerDir, "go-review.md"), []byte("go review skill"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := applyLayerFolderOverlay(tmpDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nestedPath := filepath.Join(skillsDir, "coding-practices", "go-review.md")
	content, err := os.ReadFile(nestedPath)
	if err != nil {
		t.Fatalf("nested file should exist at %s: %v", nestedPath, err)
	}
	if string(content) != "go review skill" {
		t.Errorf("expected nested skill content, got: %s", content)
	}
}

func TestIsExistingDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	cases := []struct {
		name     string
		setup    func(path string) error
		expected bool
	}{
		{
			name: "existingDirectory",
			setup: func(path string) error {
				return os.MkdirAll(path, 0755)
			},
			expected: true,
		},
		{
			name: "existingFile",
			setup: func(path string) error {
				return os.WriteFile(path, []byte("file content"), 0644)
			},
			expected: false,
		},
		{
			name: "nonExistent",
			setup: func(_ string) error {
				return nil
			},
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testPath := filepath.Join(tmpDir, tc.name)
			if err := tc.setup(testPath); err != nil {
				t.Fatal(err)
			}
			got := isExistingDirectory(testPath)
			if got != tc.expected {
				t.Errorf("isExistingDirectory(%q) = %v; want %v", testPath, got, tc.expected)
			}
		})
	}
}

func TestApplyLayerFolderOverlay_SubfolderIsFile(t *testing.T) {
	tmpDir := t.TempDir()
	factoraDir := filepath.Join(tmpDir, ".sgai", "agent")
	layerDir := filepath.Join(tmpDir, "sgai")

	if err := os.MkdirAll(factoraDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(layerDir, 0755); err != nil {
		t.Fatal(err)
	}

	agentPath := filepath.Join(layerDir, "agent")
	if err := os.WriteFile(agentPath, []byte("agent as file"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := applyLayerFolderOverlay(tmpDir); err != nil {
		t.Fatalf("expected no error when subfolder agent is a file, got: %v", err)
	}
}

func TestIsProtectedFile(t *testing.T) {
	cases := []struct {
		name      string
		subfolder string
		relPath   string
		want      bool
	}{
		{
			name:      "coordinatorMDIsProtected",
			subfolder: "agent",
			relPath:   "coordinator.md",
			want:      true,
		},
		{
			name:      "otherAgentFilesNotProtected",
			subfolder: "agent",
			relPath:   "developer.md",
			want:      false,
		},
		{
			name:      "coordinatorInSkillsNotProtected",
			subfolder: "skills",
			relPath:   "coordinator.md",
			want:      false,
		},
		{
			name:      "coordinatorInSnippetsNotProtected",
			subfolder: "snippets",
			relPath:   "coordinator.md",
			want:      false,
		},
		{
			name:      "nestedCoordinatorNotProtected",
			subfolder: "agent",
			relPath:   "subdir/coordinator.md",
			want:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isProtectedFile(tc.subfolder, tc.relPath)
			if got != tc.want {
				t.Errorf("isProtectedFile(%q, %q) = %v; want %v", tc.subfolder, tc.relPath, got, tc.want)
			}
		})
	}
}
