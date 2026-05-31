package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func copyCompletionArtifactsToRetrospective(cfg agentRunConfig) {
	if cfg.retrospectiveDir == "" {
		return
	}
	goalRetrospectivePath := filepath.Join(cfg.retrospectiveDir, "GOAL.md")
	if errCopy := copyFileAtomic(cfg.goalPath, goalRetrospectivePath); errCopy != nil {
		log.Fatalln("failed to copy GOAL.md to retrospective:", errCopy)
	}
	pmPath := filepath.Join(cfg.dir, ".sgai", "PROJECT_MANAGEMENT.md")
	if _, errStat := os.Stat(pmPath); errStat == nil {
		pmRetrospectivePath := filepath.Join(cfg.retrospectiveDir, "PROJECT_MANAGEMENT.md")
		if errCopy := copyFileAtomic(pmPath, pmRetrospectivePath); errCopy != nil {
			log.Fatalln("failed to copy PROJECT_MANAGEMENT.md to retrospective:", errCopy)
		}
	}
}

func generateRetrospectiveDirName() string {
	timestamp := time.Now().Format("2006-01-02-15-04")
	suffix := make([]byte, 4)
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	if _, errRead := rand.Read(suffix); errRead != nil {
		log.Fatalln("failed to generate random suffix:", errRead)
	}
	for i := range suffix {
		suffix[i] = chars[int(suffix[i])%len(chars)]
	}
	return timestamp + "." + string(suffix)
}

func openRetrospectiveLogs(retrospectiveDir string) (io.WriteCloser, io.WriteCloser, error) {
	if retrospectiveDir == "" {
		return nil, nil, nil
	}

	stdoutLogPath := filepath.Join(retrospectiveDir, "stdout.log")
	stderrLogPath := filepath.Join(retrospectiveDir, "stderr.log")

	stdoutLog, errStdout := prepareLogFile(stdoutLogPath)
	if errStdout != nil {
		return nil, nil, fmt.Errorf("preparing stdout.log: %w", errStdout)
	}

	stderrLog, errStderr := prepareLogFile(stderrLogPath)
	if errStderr != nil {
		if errClose := stdoutLog.Close(); errClose != nil {
			log.Println("failed to close stdout log during error cleanup:", errClose)
		}
		return nil, nil, fmt.Errorf("preparing stderr.log: %w", errStderr)
	}

	return stdoutLog, stderrLog, nil
}

func updateProjectManagementWithRetrospectiveDir(pmPath, retrospectiveDirRel string) error {
	const headerDelimiter = "---"
	const headerPrefix = "Retrospective Session: "

	var existingContent []byte
	existingContent, errRead := os.ReadFile(pmPath)
	if errRead != nil && !os.IsNotExist(errRead) {
		return fmt.Errorf("failed to read PROJECT_MANAGEMENT.md: %w", errRead)
	}

	newHeader := fmt.Sprintf("%s\n%s%s\n%s\n", headerDelimiter, headerPrefix, retrospectiveDirRel, headerDelimiter)
	content := string(existingContent)
	lines := linesWithTrailingEmpty(content)

	if len(lines) >= 3 && strings.HasPrefix(lines[0], headerDelimiter) {
		endIdx := -1
		for i := 1; i < len(lines); i++ {
			if strings.HasPrefix(lines[i], headerDelimiter) {
				endIdx = i
				break
			}
		}

		if endIdx > 0 {
			for i := 1; i < endIdx; i++ {
				if strings.HasPrefix(lines[i], headerPrefix) {
					remainingLines := lines[endIdx+1:]
					if len(remainingLines) > 0 && remainingLines[0] == "" {
						remainingLines = remainingLines[1:]
					}
					content = strings.Join(remainingLines, "\n")
					break
				}
			}
		}
	}

	if content != "" && !strings.HasPrefix(content, "\n") {
		newHeader += "\n"
	}

	if errMkdir := os.MkdirAll(filepath.Dir(pmPath), 0755); errMkdir != nil {
		return fmt.Errorf("failed to create .sgai directory: %w", errMkdir)
	}

	if errWrite := os.WriteFile(pmPath, []byte(newHeader+content), 0644); errWrite != nil {
		return fmt.Errorf("failed to write PROJECT_MANAGEMENT.md: %w", errWrite)
	}

	return nil
}

func extractRetrospectiveDirFromProjectManagement(pmPath string) string {
	const headerPrefix = "Retrospective Session: "

	content, errRead := os.ReadFile(pmPath)
	if errRead != nil {
		return ""
	}

	lines := linesWithTrailingEmpty(string(content))
	if len(lines) < 3 {
		return ""
	}

	if !strings.HasPrefix(lines[0], "---") {
		return ""
	}

	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "---") {
			break
		}
		if after, ok := strings.CutPrefix(line, headerPrefix); ok {
			return after
		}
	}

	return ""
}

func copyFinalStateToRetrospective(dir, retrospectiveDir string) error {
	statePath := filepath.Join(dir, ".sgai", "state.json")
	pmPath := filepath.Join(dir, ".sgai", "PROJECT_MANAGEMENT.md")

	if _, errStat := os.Stat(statePath); errStat == nil {
		stateDst := filepath.Join(retrospectiveDir, "state.json")
		if errCopy := copyFileAtomic(statePath, stateDst); errCopy != nil {
			return fmt.Errorf("failed to copy state.json: %w", errCopy)
		}
	}

	if _, errStat := os.Stat(pmPath); errStat == nil {
		pmDst := filepath.Join(retrospectiveDir, "PROJECT_MANAGEMENT.md")
		if errCopy := copyFileAtomic(pmPath, pmDst); errCopy != nil {
			return fmt.Errorf("failed to copy PROJECT_MANAGEMENT.md: %w", errCopy)
		}
	}

	return nil
}
