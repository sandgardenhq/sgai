package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func computeGoalChecksum(goalPath string) (string, error) {
	data, err := os.ReadFile(goalPath)
	if err != nil {
		return "", err
	}

	body := extractBody(data)
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:]), nil
}

func extractBody(content []byte) []byte {
	delimiter := []byte("---")

	if !bytes.HasPrefix(content, delimiter) {
		return content
	}

	rest := content[len(delimiter):]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	closingIdx := bytes.Index(rest, delimiter)
	if closingIdx == -1 {
		return content
	}

	bodyStart := len(delimiter) + 1 + closingIdx + len(delimiter)
	if bodyStart < len(content) && content[bodyStart] == '\n' {
		bodyStart++
	}
	if bodyStart >= len(content) {
		return []byte{}
	}
	return content[bodyStart:]
}

// generateRetrospectiveDirName generates a timestamp-based folder name in format YYYY-MM-DD-HH-II.XXXX
// where XXXX is 4 random lowercase alphanumeric characters [a-z0-9]
func generateRetrospectiveDirName() string {
	timestamp := time.Now().Format("2006-01-02-15-04")
	suffix := make([]byte, 4)
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	if _, err := rand.Read(suffix); err != nil {
		log.Fatalln("failed to generate random suffix:", err)
	}
	for i := range suffix {
		suffix[i] = chars[int(suffix[i])%len(chars)]
	}
	return timestamp + "." + string(suffix)
}

func setupOutputCapture(retrospectiveDir string) error {
	stdoutLogPath := filepath.Join(retrospectiveDir, "stdout.log")
	stderrLogPath := filepath.Join(retrospectiveDir, "stderr.log")

	stdoutLog, err := prepareLogFile(stdoutLogPath)
	if err != nil {
		return fmt.Errorf("preparing stdout.log: %w", err)
	}

	stderrLog, err := prepareLogFile(stderrLogPath)
	if err != nil {
		return fmt.Errorf("preparing stderr.log: %w", err)
	}

	originalStdout := os.Stdout
	originalStderr := os.Stderr

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}

	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}

	os.Stdout = stdoutW
	os.Stderr = stderrW

	go func() {
		if _, err := io.Copy(io.MultiWriter(originalStdout, stdoutLog), stdoutR); err != nil {
			log.Println("write failed:", err)
		}
	}()
	go func() {
		if _, err := io.Copy(io.MultiWriter(originalStderr, stderrLog), stderrR); err != nil {
			log.Println("write failed:", err)
		}
	}()

	return nil
}

func updateProjectManagementWithRetrospectiveDir(pmPath, retrospectiveDirRel string) error {
	const headerDelimiter = "---"
	const headerPrefix = "Retrospective Session: "

	var existingContent []byte
	existingContent, err := os.ReadFile(pmPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read PROJECT_MANAGEMENT.md: %w", err)
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

	finalContent := newHeader + content

	if err := os.MkdirAll(filepath.Dir(pmPath), 0755); err != nil {
		return fmt.Errorf("failed to create .sgai directory: %w", err)
	}

	if err := os.WriteFile(pmPath, []byte(finalContent), 0644); err != nil {
		return fmt.Errorf("failed to write PROJECT_MANAGEMENT.md: %w", err)
	}

	return nil
}

// extractRetrospectiveDirFromProjectManagement parses the PROJECT_MANAGEMENT.md
// frontmatter to extract the Retrospective Session path.
func extractRetrospectiveDirFromProjectManagement(pmPath string) string {
	const headerPrefix = "Retrospective Session: "

	content, err := os.ReadFile(pmPath)
	if err != nil {
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

func copyToRetrospective(src, dst string) error {
	return copyFileAtomic(src, dst)
}

func copyFileAtomic(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	tmpDst := dst + ".tmp"

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := srcFile.Close(); err != nil {
			log.Println("close failed:", err)
		}
	}()

	tmpFile, err := os.Create(tmpDst)
	if err != nil {
		return err
	}
	tmpClosed := false
	defer func() {
		if !tmpClosed {
			if errClose := tmpFile.Close(); errClose != nil {
				log.Println("close failed:", errClose)
			}
		}
		if err != nil {
			if errRemove := os.Remove(tmpDst); errRemove != nil {
				log.Println("cleanup failed:", errRemove)
			}
		}
	}()

	if _, err = io.Copy(tmpFile, srcFile); err != nil {
		return err
	}

	if err = tmpFile.Close(); err != nil {
		return err
	}
	tmpClosed = true

	if err = os.Rename(tmpDst, dst); err != nil {
		return err
	}

	return nil
}

func copyFinalStateToRetrospective(dir, retrospectiveDir string) error {
	statePath := filepath.Join(dir, ".sgai", "state.json")
	pmPath := filepath.Join(dir, ".sgai", "PROJECT_MANAGEMENT.md")

	if _, err := os.Stat(statePath); err == nil {
		stateDst := filepath.Join(retrospectiveDir, "state.json")
		if err := copyFileAtomic(statePath, stateDst); err != nil {
			return fmt.Errorf("failed to copy state.json: %w", err)
		}
	}

	if _, err := os.Stat(pmPath); err == nil {
		pmDst := filepath.Join(retrospectiveDir, "PROJECT_MANAGEMENT.md")
		if err := copyFileAtomic(pmPath, pmDst); err != nil {
			return fmt.Errorf("failed to copy PROJECT_MANAGEMENT.md: %w", err)
		}
	}

	return nil
}

func exportSession(sessionID, outputPath string) error {
	cmd := exec.Command("opencode", "export", sessionID)
	cmd.Env = append(os.Environ(), "OPENCODE_CONFIG_DIR=.sgai")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("opencode export failed: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(outputPath, output, 0644)
}
