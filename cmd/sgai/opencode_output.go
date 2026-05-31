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

func formatLogTimestamp(t time.Time) string {
	return "[" + t.UTC().Format(time.RFC3339) + "]"
}

type prefixWriter struct {
	prefix string
	w      io.Writer
}

func (p *prefixWriter) Write(data []byte) (int, error) {
	lines := linesWithTrailingEmpty(string(data))
	for i, line := range lines {
		if i < len(lines)-1 || line != "" {
			timestamp := formatLogTimestamp(time.Now())
			if _, errWrite := p.w.Write([]byte(timestamp + p.prefix + line + "\n")); errWrite != nil {
				return 0, errWrite
			}
		}
	}
	return len(data), nil
}

type streamEvent struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
	SessionID string `json:"sessionID"`
	Model     string `json:"model,omitempty"`
	Part      part   `json:"part"`
}

type part struct {
	Type   string     `json:"type"`
	Text   string     `json:"text,omitempty"`
	Tool   string     `json:"tool,omitempty"`
	Model  string     `json:"model,omitempty"`
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
	coord        *state.Coordinator
	currentAgent string
	stepCounter  int
}

func (j *jsonPrettyWriter) tsPrefix() string {
	return formatLogTimestamp(time.Now()) + j.prefix
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
		if errUnmarshal := json.Unmarshal(line, &event); errUnmarshal != nil {
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
			j.processToolPart(part)
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
			if _, errWrite := fmt.Fprintln(j.w, j.tsPrefix()+"[thinking] ..."); errWrite != nil {
				log.Println("write failed:", errWrite)
			}
		}

	default:
		if event.Type != "" {
			if _, errWrite := fmt.Fprintln(j.w, j.tsPrefix()+"["+event.Type+"]"); errWrite != nil {
				log.Println("write failed:", errWrite)
			}
		}
	}
}

func (j *jsonPrettyWriter) processToolPart(part part) {
	toolCall := formatToolCall(part.Tool, part.State.Input)
	switch part.State.Status {
	case "pending":
		if _, errWrite := fmt.Fprintln(j.w, j.tsPrefix()+toolCall); errWrite != nil {
			log.Println("write failed:", errWrite)
		}
	case "running":
		if _, errWrite := fmt.Fprintln(j.w, j.tsPrefix()+toolCall+" ..."); errWrite != nil {
			log.Println("write failed:", errWrite)
		}
	case "completed":
		j.processCompletedToolPart(part, toolCall)
	case "error":
		if _, errWrite := fmt.Fprintln(j.w, j.tsPrefix()+toolCall+" ERROR:", part.State.Error); errWrite != nil {
			log.Println("write failed:", errWrite)
		}
	}
}

func (j *jsonPrettyWriter) processCompletedToolPart(part part, toolCall string) {
	if _, errWrite := fmt.Fprintln(j.w, j.tsPrefix()+toolCall); errWrite != nil {
		log.Println("write failed:", errWrite)
	}
	if part.State.Output == "" {
		return
	}
	if isTodoTool(part.Tool) {
		j.handleTodoOutput(part.Tool, part.State.Output)
		return
	}
	for line := range strings.SplitSeq(part.State.Output, "\n") {
		if _, errWrite := fmt.Fprintln(j.w, j.tsPrefix()+"  → "+line); errWrite != nil {
			log.Println("write failed:", errWrite)
		}
	}
}

func (j *jsonPrettyWriter) flushText() {
	if j.currentText.Len() > 0 {
		text := j.currentText.String()
		for line := range strings.SplitSeq(text, "\n") {
			if _, errWrite := fmt.Fprintln(j.w, j.tsPrefix()+line); errWrite != nil {
				log.Println("write failed:", errWrite)
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
	if j.coord == nil || j.currentAgent == "" {
		return
	}
	if p.Cost == 0 && p.Tokens.Input == 0 && p.Tokens.Output == 0 {
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

	if errUpdate := j.coord.UpdateState(func(wf *state.Workflow) {
		wf.Cost.TotalCost += stepCost.Cost
		wf.Cost.TotalTokens.Add(stepCost.Tokens)

		agentIdx := slices.IndexFunc(wf.Cost.ByAgent, func(ac state.AgentCost) bool {
			return ac.Agent == j.currentAgent
		})
		if agentIdx == -1 {
			wf.Cost.ByAgent = append(wf.Cost.ByAgent, state.AgentCost{
				Agent:  j.currentAgent,
				Cost:   stepCost.Cost,
				Tokens: stepCost.Tokens,
				Steps:  []state.StepCost{stepCost},
			})
		} else {
			wf.Cost.ByAgent[agentIdx].Cost += stepCost.Cost
			wf.Cost.ByAgent[agentIdx].Tokens.Add(stepCost.Tokens)
			wf.Cost.ByAgent[agentIdx].Steps = append(wf.Cost.ByAgent[agentIdx].Steps, stepCost)
		}
	}); errUpdate != nil {
		log.Println("failed to save state:", errUpdate)
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

func isNativeTodoTool(tool string) bool {
	switch tool {
	case "todowrite", "todoread":
		return true
	default:
		return false
	}
}

func (j *jsonPrettyWriter) handleTodoOutput(tool, output string) {
	todos, errParse := j.formatTodoOutput(output)
	if errParse != nil {
		return
	}
	j.persistNativeTodoOutput(tool, todos)
}

func (j *jsonPrettyWriter) formatTodoOutput(output string) ([]state.TodoItem, error) {
	jsonOutput := stripMCPTodoPrefix(output)

	var todos []state.TodoItem
	if errUnmarshal := json.Unmarshal([]byte(jsonOutput), &todos); errUnmarshal != nil {
		for line := range strings.SplitSeq(output, "\n") {
			if _, errWrite := fmt.Fprintln(j.w, j.tsPrefix()+"  → "+line); errWrite != nil {
				log.Println("write failed:", errWrite)
			}
		}
		return nil, errUnmarshal
	}

	for _, t := range todos {
		symbol := todoStatusSymbol(t.Status)
		if _, errWrite := fmt.Fprintf(j.w, "%s  → %s %s (%s)\n", j.tsPrefix(), symbol, t.Content, t.Priority); errWrite != nil {
			log.Println("write failed:", errWrite)
		}
	}
	return todos, nil
}

func (j *jsonPrettyWriter) persistNativeTodoOutput(tool string, todos []state.TodoItem) {
	if !isNativeTodoTool(tool) || j.coord == nil || j.currentAgent == "" || j.currentAgent == "coordinator" {
		return
	}
	if errUpdate := j.coord.UpdateState(func(wf *state.Workflow) {
		wf.Todos = todos
	}); errUpdate != nil {
		log.Println("failed to save native todos:", errUpdate)
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
