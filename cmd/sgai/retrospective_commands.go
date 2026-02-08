package main

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"time"
)

const retrospectiveDirPattern = `^\d{4}-\d{2}-\d{2}-\d{2}-\d{2}\.[a-z0-9]{4}$`

var retrospectiveDirPatternRE = regexp.MustCompile(retrospectiveDirPattern)

func cmdSessions(_ []string) {
	sessions := listRetrospectiveSessions()
	if len(sessions) == 0 {
		fmt.Println("No sessions found")
		return
	}

	slices.SortFunc(sessions, cmp.Compare)

	retrospectivesDir := filepath.Join(".sgai", "retrospectives")
	for _, session := range sessions {
		goalPath := filepath.Join(retrospectivesDir, session, "GOAL.md")
		summary := extractGoalSummary(goalPath)
		if summary != "" {
			fmt.Printf("%s - %s\n", session, summary)
		} else {
			fmt.Printf("%s - (no GOAL.md)\n", session)
		}
	}
}

func listRetrospectiveSessions() []string {
	retrospectivesDir := filepath.Join(".sgai", "retrospectives")
	entries, err := os.ReadDir(retrospectivesDir)
	if err != nil {
		return nil
	}

	var sessions []string
	for _, entry := range entries {
		if entry.IsDir() && retrospectiveDirPatternRE.MatchString(entry.Name()) {
			sessions = append(sessions, entry.Name())
		}
	}
	return sessions
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

func cmdRetrospective(args []string) {
	if len(args) == 0 {
		printRetrospectiveUsage()
		return
	}

	switch args[0] {
	case "analyze":
		cmdRetrospectiveAnalyze(args[1:])
	case "apply":
		cmdRetrospectiveApply(args[1:])
	default:
		fmt.Printf("Unknown retrospective subcommand: %s\n\n", args[0])
		printRetrospectiveUsage()
		os.Exit(1)
	}
}

func printRetrospectiveUsage() {
	fmt.Println(`Usage: sgai retrospective <subcommand> [args]

Subcommands:
  analyze [session-id]    Analyze a session (default: most recent)
  apply <session-id>      Apply improvements from a session`)
}

func cmdRetrospectiveAnalyze(args []string) {
	var sessionID string
	var externalTempDir string

	for i := 0; i < len(args); i++ {
		if after, ok := strings.CutPrefix(args[i], "--temp-dir="); ok {
			externalTempDir = after
			continue
		}
		if args[i] == "--temp-dir" && i+1 < len(args) {
			externalTempDir = args[i+1]
			i++
			continue
		}
		if sessionID == "" {
			sessionID = args[i]
		}
	}

	if sessionID == "" {
		sessionID = findMostRecentSession()
		if sessionID == "" {
			log.Fatalln("no sessions found")
		}
		fmt.Println("[analyze] Using most recent session:", sessionID)
	}

	retrospectivesDir := filepath.Join(".sgai", "retrospectives")
	sessionPath := filepath.Join(retrospectivesDir, sessionID)

	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		log.Fatalln("session not found:", sessionID)
	}

	absSessionPath, err := filepath.Abs(sessionPath)
	if err != nil {
		log.Fatalln("failed to get absolute session path:", err)
	}

	tempDir := externalTempDir
	if tempDir == "" {
		var errCreate error
		tempDir, errCreate = os.MkdirTemp("", "sgai-retrospective-*")
		if errCreate != nil {
			log.Fatalln("failed to create temp directory:", errCreate)
		}
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				log.Println("cleanup failed:", err)
			}
		}()
	}

	fmt.Println("[analyze] Created temp directory:", tempDir)

	if err := copyDirectoryIfExists(sessionPath, tempDir); err != nil {
		log.Println("failed to copy session directory:", err)
		return
	}

	coordinatorModel := coordinatorModelFromCurrentDir()
	goalContent := buildRetrospectiveGoalContent(absSessionPath, coordinatorModel)
	goalPath := filepath.Join(tempDir, "GOAL.md")
	if err := os.WriteFile(goalPath, []byte(goalContent), 0644); err != nil {
		log.Println("failed to write GOAL.md:", err)
		return
	}

	fmt.Println("[analyze] Starting retrospective workflow...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runWorkflow(ctx, []string{tempDir})

	improvementsSrc := filepath.Join(tempDir, "IMPROVEMENTS.md")
	improvementsDst := filepath.Join(absSessionPath, "IMPROVEMENTS.md")

	if _, err := os.Stat(improvementsSrc); err == nil {
		if err := copyFile(improvementsSrc, improvementsDst); err != nil {
			log.Println("[analyze] warning: failed to copy IMPROVEMENTS.md:", err)
		} else {
			fmt.Println("[analyze] Created:", improvementsDst)
		}
	} else {
		fmt.Println("[analyze] No IMPROVEMENTS.md generated")
	}

	fmt.Println("[analyze] Analysis complete for session:", sessionID)
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
%sinteractive: auto
---
Analyze session: %s
`, modelsSection, absSessionPath)
}

func coordinatorModelFromCurrentDir() string {
	goalData, errRead := os.ReadFile("GOAL.md")
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

func findMostRecentSession() string {
	sessions := listRetrospectiveSessions()
	if len(sessions) == 0 {
		return ""
	}
	return slices.Max(sessions)
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

func cmdRetrospectiveApply(args []string) {
	selectedMode := false
	var sessionID string

	for i := range args {
		if args[i] == "--selected" {
			selectedMode = true
			continue
		}
		if sessionID == "" {
			sessionID = args[i]
		}
	}

	if sessionID == "" {
		log.Fatalln("usage: sgai retrospective apply [--selected] <session-id>")
	}

	retrospectivesDir := filepath.Join(".sgai", "retrospectives")
	sessionPath := filepath.Join(retrospectivesDir, sessionID)

	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		log.Fatalln("session not found:", sessionID)
	}

	improvementsPath := filepath.Join(sessionPath, "IMPROVEMENTS.md")

	if _, err := os.Stat(improvementsPath); os.IsNotExist(err) {
		log.Fatalln("no improvements.md in session", sessionID, "run 'sgai retrospective analyze", sessionID, "first.'")
	}

	const prefix = "[apply   ]"
	var content []byte
	var err error

	if selectedMode {
		fmt.Println(prefix, "reading selected improvements from stdin...")
		content, err = io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalln("failed to read selected improvements from stdin:", err)
		}
	} else {
		fmt.Println(prefix, "reading improvements from:", improvementsPath)
		content, err = os.ReadFile(improvementsPath)
		if err != nil {
			log.Fatalln("failed to read improvements file:", err)
		}
	}

	fmt.Println(prefix, "delegating to retrospective-applier agent...")
	runAgent(prefix, "retrospective-applier", fmt.Sprintf(`Read this IMPROVEMENTS.md file and apply all approved improvements.

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
Report what was created.`, improvementsPath, sessionPath, string(content)))

	reportPath := filepath.Join(sessionPath, "RETROSPECTIVE_REPORT.md")
	report := fmt.Sprintf("# Retrospective Report\n\nGenerated: %s\n\n## Applied Improvements\n\nSee %s for details.\n", time.Now().Format(time.RFC3339), improvementsPath)
	if err := os.WriteFile(reportPath, []byte(report), 0644); err != nil {
		log.Println(prefix, "warning: failed to write retrospective report:", err)
	}

	fmt.Println()
	fmt.Println(prefix, "application complete!")
	fmt.Printf("%s Created %s\n", prefix, reportPath)
}

func runAgent(prefix, agentName, message string) {
	cmd := exec.Command("opencode", "run", "--agent", agentName, "--title", agentName)
	cmd.Env = append(os.Environ(), "OPENCODE_CONFIG_DIR=.sgai")
	cmd.Stdin = strings.NewReader(message)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Println(prefix, "warning: agent", agentName, "failed:", err)
	}
}
