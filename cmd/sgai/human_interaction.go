package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
	"golang.org/x/term"
)

func waitForStateTransition(dir, statePath string) string {
	responsePath := filepath.Join(dir, ".sgai", "response.txt")
	for {
		st, err := state.Load(statePath)
		if err == nil && st.Status == state.StatusWorking {
			data, err := os.ReadFile(responsePath)
			if err != nil {
				return ""
			}
			if err := os.Remove(responsePath); err != nil {
				log.Println("cleanup failed:", err)
			}
			return string(data)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func writeResponseAndTransition(dir, statePath, response string) {
	responsePath := filepath.Join(dir, ".sgai", "response.txt")
	if err := os.WriteFile(responsePath, []byte(response), 0644); err != nil {
		log.Fatalln("failed to write response file:", err)
	}
	st, err := state.Load(statePath)
	if err != nil {
		log.Fatalln("failed to load state:", err)
	}
	st.Status = state.StatusWorking
	if err := state.Save(statePath, st); err != nil {
		log.Fatalln("failed to save state:", err)
	}
}

func launchEditorForResponse(dir, humanMessage, statePath string) {
	response, err := openEditorForResponse(humanMessage)
	if err != nil {
		log.Fatalln("failed to get human response:", err)
	}
	writeResponseAndTransition(dir, statePath, response)
}

func handleMultiChoiceQuestion(dir, statePath string, mcq *state.MultiChoiceQuestion) {
	response, err := collectMultiChoiceResponse(mcq)
	if err != nil {
		log.Fatalln("failed to collect multi-choice response:", err)
	}

	wfState, err := state.Load(statePath)
	if err != nil {
		log.Fatalln("failed to load state:", err)
	}
	wfState.MultiChoiceQuestion = nil
	if err := state.Save(statePath, wfState); err != nil {
		log.Fatalln("failed to clear multi-choice question:", err)
	}

	writeResponseAndTransition(dir, statePath, response)
}

func collectMultiChoiceResponse(mcq *state.MultiChoiceQuestion) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	var allResponses []string

	for qIdx, q := range mcq.Questions {
		fmt.Println()
		fmt.Printf("# Question %d of %d\n", qIdx+1, len(mcq.Questions))
		fmt.Println(q.Question)
		fmt.Println()

		if q.MultiSelect {
			fmt.Println("(Select one or more options by entering numbers separated by commas, e.g., 1,3)")
		} else {
			fmt.Println("(Select one option by entering its number)")
		}
		fmt.Println()

		for i, choice := range q.Choices {
			fmt.Printf("  [%d] %s\n", i+1, choice)
		}
		fmt.Println()
		fmt.Println("  [O] Other (provide custom input)")
		fmt.Println()

		var selectedChoices []string
		for {
			fmt.Print("Your selection: ")
			input, err := reader.ReadString('\n')
			if err != nil {
				return "", fmt.Errorf("failed to read input: %w", err)
			}
			input = strings.TrimSpace(input)

			if input == "" {
				fmt.Println("Please enter a selection.")
				continue
			}

			selectedChoices, err = parseChoiceSelection(input, q.Choices, q.MultiSelect)
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}
			break
		}

		fmt.Println()
		fmt.Print("Other (optional, press Enter to skip): ")
		otherInput, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read other input: %w", err)
		}
		otherInput = strings.TrimSpace(otherInput)

		if !q.MultiSelect && len(selectedChoices) == 0 && otherInput == "" {
			return "", fmt.Errorf("must select at least one option or provide custom input")
		}

		response := formatMultiChoiceResponse(selectedChoices, otherInput)
		if len(mcq.Questions) > 1 {
			allResponses = append(allResponses, fmt.Sprintf("Q%d: %s\n%s", qIdx+1, q.Question, response))
		} else {
			allResponses = append(allResponses, response)
		}
	}

	return strings.Join(allResponses, "\n\n"), nil
}

func parseChoiceSelection(input string, choices []string, multiSelect bool) ([]string, error) {
	input = strings.ToUpper(strings.TrimSpace(input))

	if input == "O" {
		return nil, nil
	}

	parts := strings.Split(input, ",")
	var selectedIndices []int

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.ToUpper(part) == "O" {
			continue
		}

		idx, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid selection '%s': must be a number or 'O'", part)
		}
		if idx < 1 || idx > len(choices) {
			return nil, fmt.Errorf("invalid selection %d: must be between 1 and %d", idx, len(choices))
		}
		selectedIndices = append(selectedIndices, idx-1)
	}

	if !multiSelect && len(selectedIndices) > 1 {
		return nil, fmt.Errorf("single-select mode: please select only one option")
	}

	var selected []string
	for _, idx := range selectedIndices {
		selected = append(selected, choices[idx])
	}

	return selected, nil
}

func formatMultiChoiceResponse(selectedChoices []string, otherInput string) string {
	var parts []string

	if len(selectedChoices) > 0 {
		parts = append(parts, "Selected: "+strings.Join(selectedChoices, ", "))
	}

	if otherInput != "" {
		parts = append(parts, "Other: "+otherInput)
	}

	return strings.Join(parts, "\n")
}

func openEditorForResponse(humanMessage string) (string, error) {
	editor := os.Getenv("EDITOR")

	if editor == "" {
		fmt.Println("# Agent Message")
		fmt.Println()
		fmt.Println(humanMessage)
		fmt.Println()

		fd := int(os.Stdin.Fd())
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			fmt.Println("# Your Response (end with Ctrl+D):")
			fmt.Println()
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
		defer func() {
			if err := term.Restore(fd, oldState); err != nil {
				log.Println("close failed:", err)
			}
		}()

		t := term.NewTerminal(os.Stdin, "> ")
		if _, err := fmt.Fprintln(t, "# Your Response (empty line to finish):"); err != nil {
			log.Println("write failed:", err)
		}
		var lines []string
		for {
			line, err := t.ReadLine()
			if err != nil {
				break
			}
			if line == "" {
				break
			}
			lines = append(lines, line)
		}
		return strings.Join(lines, "\n"), nil
	}

	tmpFile, err := os.CreateTemp("", "sgai-*.md")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	defer func() {
		if err := os.Remove(tmpPath); err != nil {
			log.Println("cleanup failed:", err)
		}
	}()

	content := "# Agent Message\n\n" + humanMessage + "\n\n# Your Response\n\n"
	if _, err := tmpFile.WriteString(content); err != nil {
		if errClose := tmpFile.Close(); errClose != nil {
			log.Println("close failed:", errClose)
		}
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		log.Println("close failed:", err)
	}

	editorParts := strings.Fields(editor)
	cmd := exec.Command(editorParts[0], append(editorParts[1:], tmpPath)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
