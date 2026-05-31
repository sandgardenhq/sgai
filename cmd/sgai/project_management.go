package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func appendProjectManagementSection(dir, title, body string) error {
	pmPath := filepath.Join(dir, ".sgai", "PROJECT_MANAGEMENT.md")
	if errMkdir := os.MkdirAll(filepath.Dir(pmPath), 0755); errMkdir != nil {
		return errMkdir
	}
	file, errOpen := os.OpenFile(pmPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if errOpen != nil {
		return errOpen
	}
	defer func() {
		if errClose := file.Close(); errClose != nil {
			log.Println("failed to close PROJECT_MANAGEMENT.md:", errClose)
		}
	}()
	timestamp := time.Now().UTC().Format(time.RFC3339)
	_, errWrite := fmt.Fprintf(file, "\n## %s (%s)\n%s\n", title, timestamp, strings.TrimSpace(body))
	return errWrite
}

func readPendingSteeringMessage(dir string) string {
	pmPath := filepath.Join(dir, ".sgai", "PROJECT_MANAGEMENT.md")
	data, errRead := os.ReadFile(pmPath)
	if errRead != nil {
		return ""
	}
	content := string(data)
	idx := strings.LastIndex(content, "## Human Steering")
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(content[idx:])
}

func copyProjectManagementToRetrospective(dir, retrospectiveDir string) {
	if retrospectiveDir == "" {
		return
	}
	pmPath := filepath.Join(dir, ".sgai", "PROJECT_MANAGEMENT.md")
	if _, errStat := os.Stat(pmPath); errStat != nil {
		return
	}
	pmRetrospectivePath := filepath.Join(retrospectiveDir, "PROJECT_MANAGEMENT.md")
	if errCopy := copyFileAtomic(pmPath, pmRetrospectivePath); errCopy != nil {
		log.Fatalln("failed to copy PROJECT_MANAGEMENT.md to retrospective:", errCopy)
	}
}
