package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func badgeStatus(wfState state.Workflow, running bool) (class, text string) {
	if wfState.NeedsHumanInput() {
		return "badge-needs-input", "Needs Input"
	}
	if running || wfState.Status == state.StatusWorking || wfState.Status == state.StatusAgentDone {
		return "badge-running", "Running"
	}
	if !running && wfState.Status == state.StatusComplete {
		return "badge-complete", "Complete"
	}
	return "badge-stopped", "Stopped"
}

type sessionStateData struct {
	BadgeClass      string
	BadgeText       string
	NeedsInput      bool
	Running         bool
	InteractiveAuto bool
	Status          string
	Message         string
	Task            string
}

type sessionAgentData struct {
	CurrentAgent         string
	CurrentModel         string
	ModelStatuses        map[string]string
	HumanMessage         string
	RenderedHumanMessage template.HTML
	AgentSequence        []agentSequenceDisplay
}

type sessionProgressData struct {
	Progress       []state.ProgressEntry
	ProgressOpen   bool
	LatestProgress string
	Messages       []messageDisplay
	Todos          []state.TodoItem
	ProjectTodos   []state.TodoItem
	Cost           state.SessionCost
}

type sessionContentData struct {
	GoalContent        template.HTML
	PMContent          template.HTML
	ProjectMgmtContent template.HTML
	HasProjectMgmt     bool
	CodeAvailable      bool
	SVGHash            string
}

type sessionData struct {
	Directory string
	DirName   string
	ActiveTab string
	sessionStateData
	sessionAgentData
	sessionProgressData
	sessionContentData
}

func (s *Server) prepareSessionData(dir string, wfState state.Workflow, r *http.Request) sessionData {
	goalContent := ""
	if data, err := os.ReadFile(filepath.Join(dir, "GOAL.md")); err == nil {
		stripped := stripFrontmatter(string(data))
		if rendered, err := renderMarkdown([]byte(stripped)); err == nil {
			goalContent = rendered
		} else {
			goalContent = stripped
		}
	}

	pmContent := ""
	projectMgmtExists := false
	if data, err := os.ReadFile(filepath.Join(dir, ".sgai", "PROJECT_MANAGEMENT.md")); err == nil {
		projectMgmtExists = true
		stripped := stripFrontmatter(string(data))
		if rendered, err := renderMarkdown([]byte(stripped)); err == nil {
			pmContent = rendered
		} else {
			pmContent = stripped
		}
	}

	var running bool
	var interactiveAuto bool
	s.mu.Lock()
	sess := s.sessions[dir]
	s.mu.Unlock()
	if sess != nil {
		sess.mu.Lock()
		running = sess.running
		interactiveAuto = sess.interactiveAuto
		sess.mu.Unlock()
	}

	badgeClass, badgeText := badgeStatus(wfState, running)
	needsInput := wfState.NeedsHumanInput()

	status := wfState.Status
	if status == "" {
		status = "-"
	}
	message := wfState.Task
	if message == "" {
		message = "-"
	}
	task := wfState.Task
	if task == "" {
		task = "-"
	}
	currentAgent := wfState.CurrentAgent
	if currentAgent == "" {
		currentAgent = "Unknown"
	}

	reversedMessages := reverseMessages(wfState.Messages)

	todos := wfState.Todos
	if currentAgent == "coordinator" {
		todos = wfState.ProjectTodos
	}

	reversedProgress := slices.Clone(wfState.Progress)
	slices.Reverse(reversedProgress)

	progressOpen := r.URL.Query().Get("progress_open") == "true"

	renderedHumanMessage := renderHumanMessage(wfState.HumanMessage)

	codeAvailable := s.codeAvailable && isLocalRequest(r)

	return sessionData{
		Directory: dir,
		DirName:   filepath.Base(dir),
		ActiveTab: "goal",
		sessionStateData: sessionStateData{
			BadgeClass:      badgeClass,
			BadgeText:       badgeText,
			NeedsInput:      needsInput,
			Running:         running,
			InteractiveAuto: interactiveAuto,
			Status:          status,
			Message:         message,
			Task:            task,
		},
		sessionAgentData: sessionAgentData{
			CurrentAgent:         currentAgent,
			CurrentModel:         wfState.CurrentModel,
			ModelStatuses:        wfState.ModelStatuses,
			HumanMessage:         wfState.HumanMessage,
			RenderedHumanMessage: renderedHumanMessage,
			AgentSequence:        prepareAgentSequenceDisplay(wfState.AgentSequence, running, getLastActivityTime(wfState.Progress)),
		},
		sessionProgressData: sessionProgressData{
			Progress:       reversedProgress,
			ProgressOpen:   progressOpen,
			LatestProgress: getLatestProgress(wfState.Progress),
			Messages:       reversedMessages,
			Todos:          todos,
			ProjectTodos:   wfState.ProjectTodos,
			Cost:           wfState.Cost,
		},
		sessionContentData: sessionContentData{
			GoalContent:        template.HTML(goalContent),
			PMContent:          template.HTML(pmContent),
			ProjectMgmtContent: template.HTML(pmContent),
			HasProjectMgmt:     projectMgmtExists,
			CodeAvailable:      codeAvailable,
			SVGHash:            getWorkflowSVGHash(dir, currentAgent),
		},
	}
}

type eventsProgressDisplay struct {
	Timestamp       string
	FormattedTime   string
	Agent           string
	Description     string
	ShowDateDivider bool
	DateDivider     string
}

func formatProgressForDisplay(entries []state.ProgressEntry) []eventsProgressDisplay {
	result := make([]eventsProgressDisplay, 0, len(entries))
	var lastDateStr string

	for _, entry := range entries {
		parsedTime, err := time.Parse(time.RFC3339, entry.Timestamp)
		if err != nil {
			result = append(result, eventsProgressDisplay{
				Timestamp:     entry.Timestamp,
				FormattedTime: entry.Timestamp,
				Agent:         entry.Agent,
				Description:   entry.Description,
			})
			continue
		}

		formattedTime := parsedTime.Local().Format("3:04 PM")
		currentDateStr := parsedTime.Local().Format("Jan 2, 2006")

		showDateDivider := currentDateStr != lastDateStr
		if showDateDivider {
			lastDateStr = currentDateStr
		}

		result = append(result, eventsProgressDisplay{
			Timestamp:       entry.Timestamp,
			FormattedTime:   formattedTime,
			Agent:           entry.Agent,
			Description:     entry.Description,
			ShowDateDivider: showDateDivider,
			DateDivider:     currentDateStr,
		})
	}

	return result
}

func renderHumanMessage(message string) template.HTML {
	if message == "" {
		return ""
	}
	rendered, err := renderMarkdown([]byte(message))
	if err != nil {
		return template.HTML(template.HTMLEscapeString(message))
	}
	return template.HTML(rendered)
}

// messageDisplay is a view model for rendering messages in the web interface.
// It combines the data model (Message) with UI-specific transformations:
// - Subject: the first non-empty, non-markdown line extracted from the message body
// - RenderedBody: markdown-to-HTML conversion of the message body
type messageDisplay struct {
	ID           int
	FromAgent    string
	ToAgent      string
	Read         bool
	ReadAt       string
	ReadBy       string
	Body         string
	Subject      string
	RenderedBody template.HTML
}

// extractSubjectAndRemainder extracts the first non-empty line as subject
// and returns the remainder of the body for display.
// Strips leading markdown heading characters (e.g., "# Title" becomes "Title").
func extractSubjectAndRemainder(body string) (subject string, remainder string) {
	lines := linesWithTrailingEmpty(body)
	subjectIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			subject = strings.TrimLeft(trimmed, "# ")
			subjectIdx = i
			break
		}
	}
	if subjectIdx >= 0 && subjectIdx < len(lines)-1 {
		remainder = strings.TrimSpace(strings.Join(lines[subjectIdx+1:], "\n"))
	}
	return subject, remainder
}

// prepareMessageDisplay converts a Message to a messageDisplay view model,
// extracting the subject line and rendering markdown body to HTML.
func prepareMessageDisplay(msg state.Message) messageDisplay {
	subject, _ := extractSubjectAndRemainder(msg.Body)
	rendered, _ := renderMarkdown([]byte(msg.Body))

	return messageDisplay{
		ID:           msg.ID,
		FromAgent:    msg.FromAgent,
		ToAgent:      msg.ToAgent,
		Read:         msg.Read,
		ReadAt:       msg.ReadAt,
		ReadBy:       msg.ReadBy,
		Body:         msg.Body,
		Subject:      subject,
		RenderedBody: template.HTML(rendered),
	}
}

// agentSequenceDisplay is a view model for rendering agent sequence in templates.
// It combines the data model (AgentSequenceEntry) with computed elapsed time.
type agentSequenceDisplay struct {
	Agent       string
	ElapsedTime string
	IsCurrent   bool
}

func prepareAgentSequenceDisplay(sequence []state.AgentSequenceEntry, running bool, lastActivityTime string) []agentSequenceDisplay {
	now := time.Now().UTC()
	result := make([]agentSequenceDisplay, 0, len(sequence))

	var endTime time.Time
	if !running && lastActivityTime != "" {
		if parsed, err := time.Parse(time.RFC3339, lastActivityTime); err == nil {
			endTime = parsed
		}
	}

	for i, entry := range sequence {
		startTime, err := time.Parse(time.RFC3339, entry.StartTime)
		if err != nil {
			log.Println("prepareAgentSequenceDisplay: skipping entry with invalid timestamp:", entry.StartTime, err)
			continue
		}
		var elapsed time.Duration
		isLastEntry := i+1 >= len(sequence)
		switch {
		case entry.IsCurrent && running:
			elapsed = now.Sub(startTime)
		case !isLastEntry:
			nextStartTime, err := time.Parse(time.RFC3339, sequence[i+1].StartTime)
			if err != nil {
				elapsed = now.Sub(startTime)
			} else {
				elapsed = nextStartTime.Sub(startTime)
			}
		case running:
			elapsed = now.Sub(startTime)
		case !endTime.IsZero():
			elapsed = endTime.Sub(startTime)
		}
		elapsedStr := formatDuration(elapsed)
		result = append(result, agentSequenceDisplay{
			Agent:       entry.Agent,
			ElapsedTime: elapsedStr,
			IsCurrent:   entry.IsCurrent,
		})
	}
	slices.Reverse(result)
	return result
}

func formatDiffHTML(diffOutput []byte) template.HTML {
	if len(diffOutput) == 0 {
		return ""
	}

	var buf bytes.Buffer
	buf.WriteString("<pre>")

	lines := strings.SplitSeq(string(diffOutput), "\n")
	for line := range lines {
		escapedLine := template.HTMLEscapeString(line)
		lineClass := classifyDiffLine(line)
		buf.WriteString(fmt.Sprintf(`<span class="diff-line %s">%s</span>`, lineClass, escapedLine))
		buf.WriteString("\n")
	}

	buf.WriteString("</pre>")
	return template.HTML(buf.String())
}

func classifyDiffLine(line string) string {
	switch {
	case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
		return "diff-line-add"
	case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
		return "diff-line-del"
	case strings.HasPrefix(line, "@@"):
		return "diff-line-hunk"
	case strings.HasPrefix(line, "diff ") || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++"):
		return "diff-line-file"
	default:
		return "diff-line-context"
	}
}

func calculateTotalExecutionTime(sequence []state.AgentSequenceEntry, running bool, lastActivityTime string) string {
	if len(sequence) == 0 {
		return ""
	}

	startTime, err := time.Parse(time.RFC3339, sequence[0].StartTime)
	if err != nil {
		return ""
	}

	var endTime time.Time
	switch {
	case running:
		endTime = time.Now().UTC()
	case lastActivityTime != "":
		parsed, err := time.Parse(time.RFC3339, lastActivityTime)
		if err != nil {
			return ""
		}
		endTime = parsed
	default:
		return ""
	}

	elapsed := endTime.Sub(startTime)
	return formatDuration(elapsed)
}

// reverseMessages reverses the order of messages for display (newest first).
func reverseMessages(messages []state.Message) []messageDisplay {
	reversed := make([]messageDisplay, len(messages))
	for i := range messages {
		reversed[len(messages)-1-i] = prepareMessageDisplay(messages[i])
	}
	return reversed
}

func getLatestProgress(progress []state.ProgressEntry) string {
	if len(progress) == 0 {
		return "-"
	}
	return progress[len(progress)-1].Description
}

func getLastActivityTime(progress []state.ProgressEntry) string {
	if len(progress) == 0 {
		return ""
	}
	return progress[len(progress)-1].Timestamp
}

func lookupModelForAgent(dir, agentName string) string {
	goalPath := filepath.Join(dir, "GOAL.md")
	goalData, err := os.ReadFile(goalPath)
	if err != nil {
		return ""
	}
	metadata, err := parseYAMLFrontmatter(goalData)
	if err != nil {
		return ""
	}
	return selectModelForAgent(metadata.Models, agentName)
}

func formatModelForDisplay(modelSpec string) string {
	if idx := strings.LastIndex(modelSpec, "/"); idx >= 0 {
		modelSpec = modelSpec[idx+1:]
	}
	if idx := strings.Index(modelSpec, " "); idx >= 0 {
		modelSpec = modelSpec[:idx]
	}
	return modelSpec
}

type agentModelInfo struct {
	agent          string
	model          string
	formattedModel string
}

func resolveAgentModelInfo(wfState state.Workflow, dir string) agentModelInfo {
	agent := wfState.CurrentAgent
	if agent == "" {
		return agentModelInfo{}
	}
	model := lookupModelForAgent(dir, agent)
	formatted := ""
	if model != "" {
		formatted = formatModelForDisplay(model)
	}
	return agentModelInfo{agent: agent, model: model, formattedModel: formatted}
}

// workspacePageData holds data for rendering the workspace trees page.
type workspacePageData struct {
	Groups                 []workspaceGroup
	SelectedDir            string
	JJLog                  string
	SelectedPath           string
	SelectedTab            string
	SelectedSession        string
	WorkspaceContent       template.HTML
	InProgressWorkspaces   []workspaceInfo
	HasNeedsInputWorkspace bool
}

type workspaceContentResult struct {
	Content template.HTML
}

type tabContentResult struct {
	Content template.HTML
}
