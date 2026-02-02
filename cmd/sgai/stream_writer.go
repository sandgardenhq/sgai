package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

type prefixWriter struct {
	prefix string
	w      io.Writer
}

func (p *prefixWriter) Write(data []byte) (int, error) {
	lines := linesWithTrailingEmpty(string(data))
	for i, line := range lines {
		if i < len(lines)-1 || line != "" {
			if _, err := p.w.Write([]byte(p.prefix + line + "\n")); err != nil {
				return 0, err
			}
		}
	}
	return len(data), nil
}

type streamEvent struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
	SessionID string `json:"sessionID"`
	Part      part   `json:"part"`
}

type part struct {
	Type   string     `json:"type"`
	Text   string     `json:"text,omitempty"`
	Tool   string     `json:"tool,omitempty"`
	State  *toolState `json:"state,omitempty"`
	Cost   float64    `json:"cost,omitempty"`
	Tokens partTokens `json:"tokens"`
}

type partTokens struct {
	Input     int        `json:"input"`
	Output    int        `json:"output"`
	Reasoning int        `json:"reasoning"`
	Cache     cacheStats `json:"cache"`
}

type cacheStats struct {
	Read  int `json:"read"`
	Write int `json:"write"`
}

type toolState struct {
	Status string         `json:"status"`
	Input  map[string]any `json:"input"`
	Title  string         `json:"title,omitempty"`
	Output string         `json:"output,omitempty"`
	Error  string         `json:"error,omitempty"`
}

type jsonPrettyWriter struct {
	prefix       string
	w            io.Writer
	buf          []byte
	currentText  strings.Builder
	sessionID    string
	statePath    string
	currentAgent string
	stepCounter  int
}

func (j *jsonPrettyWriter) Write(data []byte) (int, error) {
	j.buf = append(j.buf, data...)
	j.processBuffer()
	return len(data), nil
}

func (j *jsonPrettyWriter) processBuffer() {
	for {
		idx := strings.Index(string(j.buf), "\n")
		if idx == -1 {
			return
		}

		line := j.buf[:idx]
		j.buf = j.buf[idx+1:]

		if len(line) == 0 {
			continue
		}

		var event streamEvent
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}

		j.processEvent(event)
	}
}

func (j *jsonPrettyWriter) processEvent(event streamEvent) {
	if event.SessionID != "" {
		j.sessionID = event.SessionID
	}
	part := event.Part

	switch event.Type {
	case "text":
		if part.Text != "" {
			j.currentText.WriteString(part.Text)
		}

	case "tool", "tool_use":
		j.flushText()
		if part.State != nil {
			toolCall := formatToolCall(part.Tool, part.State.Input)
			switch part.State.Status {
			case "pending":
				if _, err := fmt.Fprintln(j.w, j.prefix+toolCall); err != nil {
					log.Println("write failed:", err)
				}
			case "running":
				if _, err := fmt.Fprintln(j.w, j.prefix+toolCall+" ..."); err != nil {
					log.Println("write failed:", err)
				}
			case "completed":
				if _, err := fmt.Fprintln(j.w, j.prefix+toolCall); err != nil {
					log.Println("write failed:", err)
				}
				if part.State.Output != "" {
					if isTodoTool(part.Tool) {
						j.formatTodoOutput(part.State.Output)
					} else {
						for line := range strings.SplitSeq(part.State.Output, "\n") {
							if _, err := fmt.Fprintln(j.w, j.prefix+"  → "+line); err != nil {
								log.Println("write failed:", err)
							}
						}
					}
				}
			case "error":
				if _, err := fmt.Fprintln(j.w, j.prefix+toolCall+" ERROR:", part.State.Error); err != nil {
					log.Println("write failed:", err)
				}
			}
		}

	case "step_start":
		j.flushText()
		j.stepCounter++

	case "step_finish":
		j.flushText()
		j.recordStepCost(part, event.Timestamp)

	case "reasoning":
		j.flushText()
		if part.Text != "" {
			if _, err := fmt.Fprintln(j.w, j.prefix+"[thinking] ..."); err != nil {
				log.Println("write failed:", err)
			}
		}

	default:
		if event.Type != "" {
			if _, err := fmt.Fprintln(j.w, j.prefix+"["+event.Type+"]"); err != nil {
				log.Println("write failed:", err)
			}
		}
	}
}

func (j *jsonPrettyWriter) flushText() {
	if j.currentText.Len() > 0 {
		text := j.currentText.String()
		for line := range strings.SplitSeq(text, "\n") {
			if _, err := fmt.Fprintln(j.w, j.prefix+line); err != nil {
				log.Println("write failed:", err)
			}
		}
		j.currentText.Reset()
	}
}

func (j *jsonPrettyWriter) Flush() {
	j.processBuffer()
	j.flushText()
}

func (j *jsonPrettyWriter) recordStepCost(p part, timestamp int64) {
	if j.statePath == "" || j.currentAgent == "" {
		return
	}
	if p.Cost == 0 && p.Tokens.Input == 0 && p.Tokens.Output == 0 {
		return
	}

	wfState, err := state.Load(j.statePath)
	if err != nil {
		return
	}

	stepCost := state.StepCost{
		StepID: fmt.Sprintf("%s-step-%d", j.currentAgent, j.stepCounter),
		Agent:  j.currentAgent,
		Cost:   p.Cost,
		Tokens: state.TokenUsage{
			Input:      p.Tokens.Input,
			Output:     p.Tokens.Output,
			Reasoning:  p.Tokens.Reasoning,
			CacheRead:  p.Tokens.Cache.Read,
			CacheWrite: p.Tokens.Cache.Write,
		},
		Timestamp: time.Unix(0, timestamp*int64(time.Millisecond)).UTC().Format(time.RFC3339),
	}

	wfState.Cost.TotalCost += stepCost.Cost
	wfState.Cost.TotalTokens.Add(stepCost.Tokens)

	agentIdx := slices.IndexFunc(wfState.Cost.ByAgent, func(ac state.AgentCost) bool {
		return ac.Agent == j.currentAgent
	})
	if agentIdx == -1 {
		wfState.Cost.ByAgent = append(wfState.Cost.ByAgent, state.AgentCost{
			Agent:  j.currentAgent,
			Cost:   stepCost.Cost,
			Tokens: stepCost.Tokens,
			Steps:  []state.StepCost{stepCost},
		})
	} else {
		wfState.Cost.ByAgent[agentIdx].Cost += stepCost.Cost
		wfState.Cost.ByAgent[agentIdx].Tokens.Add(stepCost.Tokens)
		wfState.Cost.ByAgent[agentIdx].Steps = append(wfState.Cost.ByAgent[agentIdx].Steps, stepCost)
	}

	if err := state.Save(j.statePath, wfState); err != nil {
		log.Println("failed to save state:", err)
	}
}

func isTodoTool(tool string) bool {
	switch tool {
	case "todowrite", "todoread", "sgai_project_todowrite", "sgai_project_todoread":
		return true
	default:
		return false
	}
}

func (j *jsonPrettyWriter) formatTodoOutput(output string) {
	type todo struct {
		Content  string `json:"content"`
		Status   string `json:"status"`
		Priority string `json:"priority"`
	}

	jsonOutput := stripMCPTodoPrefix(output)

	var todos []todo
	if err := json.Unmarshal([]byte(jsonOutput), &todos); err != nil {
		for line := range strings.SplitSeq(output, "\n") {
			if _, err := fmt.Fprintln(j.w, j.prefix+"  → "+line); err != nil {
				log.Println("write failed:", err)
			}
		}
		return
	}

	for _, t := range todos {
		symbol := todoStatusSymbol(t.Status)
		if _, err := fmt.Fprintf(j.w, "%s  → %s %s (%s)\n", j.prefix, symbol, t.Content, t.Priority); err != nil {
			log.Println("write failed:", err)
		}
	}
}

func stripMCPTodoPrefix(output string) string {
	idx := strings.Index(output, "\n[")
	if idx == -1 {
		return output
	}
	prefix := strings.TrimSpace(output[:idx])
	if strings.HasSuffix(prefix, "todos") || strings.HasSuffix(prefix, "todo") {
		return output[idx+1:]
	}
	return output
}

func todoStatusSymbol(status string) string {
	switch status {
	case "pending":
		return "○"
	case "in_progress":
		return "◐"
	case "completed":
		return "●"
	case "cancelled":
		return "✕"
	default:
		return "○"
	}
}

func formatToolCall(tool string, input map[string]any) string {
	if len(input) == 0 {
		return tool
	}
	escapeReplacer := strings.NewReplacer(
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
	)
	var parts []string
	for k, v := range input {
		switch val := v.(type) {
		case string:
			val = escapeReplacer.Replace(val)
			if k != "filePath" && len(val) > 50 {
				val = val[:47] + "..."
			}
			parts = append(parts, k+": '"+val+"'")
		case bool:
			parts = append(parts, k+": "+strconv.FormatBool(val))
		case float64:
			parts = append(parts, k+": "+strconv.FormatFloat(val, 'f', -1, 64))
		default:
			parts = append(parts, k+": "+fmt.Sprint(val))
		}
	}
	return tool + "(" + strings.Join(parts, ", ") + ")"
}
