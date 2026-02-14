package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const retrospectiveDirPattern = `^\d{4}-\d{2}-\d{2}-\d{2}-\d{2}\.[a-z0-9]{4}$`

var retrospectiveDirPatternRE = regexp.MustCompile(retrospectiveDirPattern)

func runRetrospectiveAnalysis(ctx context.Context, dir, sessionID, tempDir string, logWriter io.Writer) error {
	sessionPath := filepath.Join(dir, ".sgai", "retrospectives", sessionID)

	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if err := copyDirectoryIfExists(sessionPath, tempDir); err != nil {
		return fmt.Errorf("failed to copy session directory: %w", err)
	}

	coordinatorModel := coordinatorModelFromDir(dir)
	goalContent := buildRetrospectiveGoalContent(sessionPath, coordinatorModel)
	goalPath := filepath.Join(tempDir, "GOAL.md")
	if err := os.WriteFile(goalPath, []byte(goalContent), 0644); err != nil {
		return fmt.Errorf("failed to write GOAL.md: %w", err)
	}

	retroMCPURL, retroMCPClose, errMCP := startMCPHTTPServer(tempDir)
	if errMCP != nil {
		return fmt.Errorf("failed to start MCP server: %w", errMCP)
	}
	defer retroMCPClose()

	runWorkflow(ctx, []string{tempDir}, retroMCPURL, logWriter)

	improvementsSrc := filepath.Join(tempDir, "IMPROVEMENTS.md")
	improvementsDst := filepath.Join(sessionPath, "IMPROVEMENTS.md")

	if _, err := os.Stat(improvementsSrc); err == nil {
		if err := copyFile(improvementsSrc, improvementsDst); err != nil {
			return fmt.Errorf("failed to copy IMPROVEMENTS.md: %w", err)
		}
	}

	return nil
}

func runRetrospectiveApply(dir, sessionID, selectedContent string, stdout, stderr io.Writer) error {
	sessionPath := filepath.Join(dir, ".sgai", "retrospectives", sessionID)

	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	improvementsPath := filepath.Join(sessionPath, "IMPROVEMENTS.md")

	if _, err := os.Stat(improvementsPath); os.IsNotExist(err) {
		return fmt.Errorf("no IMPROVEMENTS.md in session %s", sessionID)
	}

	message := fmt.Sprintf(`Read this IMPROVEMENTS.md file and apply all approved improvements.

IMPROVEMENTS FILE PATH: %s
RETROSPECTIVE DIRECTORY: %s

IMPROVEMENTS.MD CONTENT:
---
%s
---

Read the markdown above and apply only the improvements that have been approved by the human reviewer.
For approved skills, create them in sgai/skills/<name>/SKILL.md
For approved snippets, create them in sgai/snippets/<language>/<name>
For approved agent improvements, create/modify them in sgai/agent/<name>.md
Skip any improvements that were not approved or were vetoed.
Report what was created.`, improvementsPath, sessionPath, selectedContent)

	runAgent(dir, "[apply   ]", "retrospective-applier", message, stdout, stderr)

	reportPath := filepath.Join(sessionPath, "RETROSPECTIVE_REPORT.md")
	report := fmt.Sprintf("# Retrospective Report\n\nGenerated: %s\n\n## Applied Improvements\n\nSee %s for details.\n", time.Now().Format(time.RFC3339), improvementsPath)
	if err := os.WriteFile(reportPath, []byte(report), 0644); err != nil {
		log.Println("[apply]", "warning: failed to write retrospective report:", err)
	}

	return nil
}

func buildRetrospectiveGoalContent(absSessionPath, coordinatorModel string) string {
	var modelsSection string
	if coordinatorModel != "" {
		modelsSection = fmt.Sprintf(`models:
  "coordinator": "%s"
  "retrospective-session-analyzer": "%s"
  "retrospective-code-analyzer": "%s"
  "retrospective-refiner": "%s"
`, coordinatorModel, coordinatorModel, coordinatorModel, coordinatorModel)
	}
	return fmt.Sprintf(`---
flow: |
  "coordinator" -> "retrospective-session-analyzer"
  "retrospective-session-analyzer" -> "retrospective-code-analyzer"
  "retrospective-code-analyzer" -> "retrospective-refiner"
%s
---
Analyze session: %s
`, modelsSection, absSessionPath)
}

func coordinatorModelFromDir(dir string) string {
	goalData, errRead := os.ReadFile(filepath.Join(dir, "GOAL.md"))
	if errRead != nil {
		return ""
	}
	metadata, errParse := parseYAMLFrontmatter(goalData)
	if errParse != nil {
		return ""
	}
	models := getModelsForAgent(metadata.Models, "coordinator")
	if len(models) == 0 {
		return ""
	}
	return models[0]
}

func copyDirectoryIfExists(src, dst string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}

		return copyFile(path, dstPath)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, content, 0644)
}

func extractGoalSummary(goalPath string) string {
	content, err := os.ReadFile(goalPath)
	if err != nil {
		return ""
	}

	normalizedContent := normalizeEscapedNewlines(content)
	body := extractBody(normalizedContent)
	lines := bytes.SplitSeq(body, []byte("\n"))
	for line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 {
			return string(trimmed)
		}
	}
	return ""
}

func normalizeEscapedNewlines(content []byte) []byte {
	return bytes.ReplaceAll(content, []byte(`\n`), []byte("\n"))
}

func runAgent(dir, prefix, agentName, message string, stdout, stderr io.Writer) {
	cmd := exec.Command("opencode", "run", "--agent", agentName, "--title", agentName)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "OPENCODE_CONFIG_DIR="+filepath.Join(dir, ".sgai"))
	cmd.Stdin = strings.NewReader(message)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		log.Println(prefix, "warning: agent", agentName, "failed:", err)
	}
}
