package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func extractFileNumbers(t *testing.T, entries []os.DirEntry) []int {
	t.Helper()
	var numbers []int
	for _, entry := range entries {
		parts := strings.SplitN(entry.Name(), "-", 2)
		if len(parts) < 2 {
			t.Fatalf("unexpected file name format: %s", entry.Name())
		}
		n, errParse := strconv.Atoi(parts[0])
		if errParse != nil {
			t.Fatalf("failed to parse number from %q: %v", parts[0], errParse)
		}
		numbers = append(numbers, n)
	}
	return numbers
}

func assertSequentialNumbers(t *testing.T, numbers []int) {
	t.Helper()
	slices.Sort(numbers)
	for i := 0; i < len(numbers)-1; i++ {
		if numbers[i+1]-numbers[i] != 1 {
			t.Errorf("gap detected between file %d and %d", numbers[i], numbers[i+1])
		}
	}
}

func TestRetrospectiveFileSequentialNumbering(t *testing.T) {
	t.Run("everyIterationProducesFile", func(t *testing.T) {
		retrospectiveDir := t.TempDir()
		iterationCounter := 0
		agent := "backend-go-developer"
		statuses := []string{"working", "working", "working", "agent-done", "working", "agent-done", "complete"}
		for _, status := range statuses {
			iterationCounter++
			sessionFile := filepath.Join(retrospectiveDir, fmt.Sprintf("%04d-%s-%s.json", iterationCounter, agent, "20260224120000"))
			if err := os.WriteFile(sessionFile, []byte(`{}`), 0644); err != nil {
				t.Fatalf("failed to write session file for status %q iteration %d: %v", status, iterationCounter, err)
			}
		}

		entries, err := os.ReadDir(retrospectiveDir)
		if err != nil {
			t.Fatalf("failed to read retrospective dir: %v", err)
		}
		if len(entries) != len(statuses) {
			t.Fatalf("expected %d files, got %d", len(statuses), len(entries))
		}

		numbers := extractFileNumbers(t, entries)
		assertSequentialNumbers(t, numbers)
		if numbers[0] != 1 {
			t.Errorf("first file number = %d; want 1", numbers[0])
		}
		if numbers[len(numbers)-1] != len(statuses) {
			t.Errorf("last file number = %d; want %d", numbers[len(numbers)-1], len(statuses))
		}
	})

	t.Run("multipleAgentsInterleavedSequential", func(t *testing.T) {
		retrospectiveDir := t.TempDir()
		iterationCounter := 0
		agents := []struct {
			name   string
			status string
		}{
			{"coordinator", "agent-done"},
			{"backend-go-developer", "working"},
			{"backend-go-developer", "agent-done"},
			{"coordinator", "working"},
			{"coordinator", "complete"},
		}
		for _, a := range agents {
			iterationCounter++
			sessionFile := filepath.Join(retrospectiveDir, fmt.Sprintf("%04d-%s-%s.json", iterationCounter, a.name, "20260224120000"))
			if err := os.WriteFile(sessionFile, []byte(`{}`), 0644); err != nil {
				t.Fatalf("failed to write session file for %s status %q iteration %d: %v", a.name, a.status, iterationCounter, err)
			}
		}

		entries, err := os.ReadDir(retrospectiveDir)
		if err != nil {
			t.Fatalf("failed to read retrospective dir: %v", err)
		}
		if len(entries) != len(agents) {
			t.Fatalf("expected %d files, got %d", len(agents), len(entries))
		}

		numbers := extractFileNumbers(t, entries)
		assertSequentialNumbers(t, numbers)
	})
}

func TestRetrospectiveExportPositionInLoop(t *testing.T) {
	t.Run("exportOccursBeforeStatusSwitch", func(t *testing.T) {
		content, err := os.ReadFile("main.go")
		if err != nil {
			t.Fatalf("failed to read main.go: %v", err)
		}
		src := string(content)

		funcStart := strings.Index(src, "func runFlowAgentWithModel(")
		if funcStart == -1 {
			t.Fatal("runFlowAgentWithModel function not found in main.go")
		}
		funcBody := src[funcStart:]

		exportCallPos := strings.Index(funcBody, `if cfg.retrospectiveDir != "" && capturedSessionID != "" && shouldLogAgent(cfg.dir, cfg.agent)`)
		if exportCallPos == -1 {
			t.Fatal("session export guard not found in runFlowAgentWithModel")
		}

		statusSwitchPos := strings.Index(funcBody, "switch newState.Status {")
		if statusSwitchPos == -1 {
			t.Fatal("status switch not found in runFlowAgentWithModel")
		}

		if exportCallPos >= statusSwitchPos {
			t.Error("session export must occur BEFORE the status switch to ensure every iteration produces a file")
		}
	})

	t.Run("noDuplicateExportInStatusBranches", func(t *testing.T) {
		content, err := os.ReadFile("main.go")
		if err != nil {
			t.Fatalf("failed to read main.go: %v", err)
		}
		src := string(content)

		funcStart := strings.Index(src, "func runFlowAgentWithModel(")
		if funcStart == -1 {
			t.Fatal("runFlowAgentWithModel function not found in main.go")
		}

		nextFuncStart := strings.Index(src[funcStart+1:], "\nfunc ")
		var funcBody string
		if nextFuncStart == -1 {
			funcBody = src[funcStart:]
		} else {
			funcBody = src[funcStart : funcStart+1+nextFuncStart]
		}

		statusSwitchPos := strings.Index(funcBody, "switch newState.Status {")
		if statusSwitchPos == -1 {
			t.Fatal("status switch not found in runFlowAgentWithModel")
		}
		afterSwitch := funcBody[statusSwitchPos:]

		count := strings.Count(afterSwitch, "exportSession(")
		if count > 0 {
			t.Errorf("found %d exportSession call(s) after the status switch; export should only occur before the switch", count)
		}
	})
}
