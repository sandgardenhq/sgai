package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// treesRetroDetailsData holds data for rendering retrospective session details in tree view templates.
type treesRetroDetailsData struct {
	Directory           string
	DirName             string
	SessionName         string
	GoalSummary         string
	GoalContent         template.HTML
	ImprovementsContent template.HTML
	HasImprovements     bool
	IsAnalyzing         bool
	IsApplying          bool
}

// improvementSuggestion represents an individual suggestion parsed from IMPROVEMENTS.md.
// Each suggestion is identified by a ### heading followed by a - [ ] APPROVE line.
type improvementSuggestion struct {
	Index       int
	Name        string
	Section     string
	Content     string
	FullContent string
}

func parseImprovementSuggestions(content string) []improvementSuggestion {
	stripped := stripFrontmatter(content)
	lines := linesWithTrailingEmpty(stripped)

	var suggestions []improvementSuggestion
	var currentSection string
	var currentSuggestion *improvementSuggestion
	var contentLines []string

	skipSections := map[string]bool{"Instructions": true, "Summary": true}

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			if currentSuggestion != nil {
				currentSuggestion.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))
				currentSuggestion.FullContent = buildSuggestionFullContent(currentSuggestion.Name, contentLines)
				suggestions = append(suggestions, *currentSuggestion)
				currentSuggestion = nil
				contentLines = nil
			}

			sectionTitle := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			if skipSections[sectionTitle] {
				currentSection = ""
			} else {
				currentSection = sectionTitle
			}
			continue
		}

		if strings.HasPrefix(line, "### ") && currentSection != "" {
			if currentSuggestion != nil {
				currentSuggestion.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))
				currentSuggestion.FullContent = buildSuggestionFullContent(currentSuggestion.Name, contentLines)
				suggestions = append(suggestions, *currentSuggestion)
			}

			suggestionName := strings.TrimSpace(strings.TrimPrefix(line, "### "))
			currentSuggestion = &improvementSuggestion{
				Index:   len(suggestions),
				Name:    suggestionName,
				Section: currentSection,
			}
			contentLines = nil
			continue
		}

		if line == "---" && currentSuggestion != nil {
			currentSuggestion.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))
			currentSuggestion.FullContent = buildSuggestionFullContent(currentSuggestion.Name, contentLines)
			suggestions = append(suggestions, *currentSuggestion)
			currentSuggestion = nil
			contentLines = nil
			continue
		}

		if currentSuggestion != nil {
			contentLines = append(contentLines, line)
		}
	}

	if currentSuggestion != nil {
		currentSuggestion.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))
		currentSuggestion.FullContent = buildSuggestionFullContent(currentSuggestion.Name, contentLines)
		suggestions = append(suggestions, *currentSuggestion)
	}

	return suggestions
}

func buildSuggestionFullContent(name string, contentLines []string) string {
	var b strings.Builder
	b.WriteString("### ")
	b.WriteString(name)
	b.WriteString("\n")
	for _, line := range contentLines {
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

type noteGetter func(suggestionIndex int) string

func filterSelectedSuggestions(suggestions []improvementSuggestion, selectedIndices []string) []improvementSuggestion {
	selectedMap := make(map[int]bool)
	for _, idxStr := range selectedIndices {
		if idx, err := strconv.Atoi(idxStr); err == nil {
			selectedMap[idx] = true
		}
	}

	var selected []improvementSuggestion
	for _, suggestion := range suggestions {
		if selectedMap[suggestion.Index] {
			selected = append(selected, suggestion)
		}
	}
	return selected
}

func buildSelectedImprovementsContent(suggestions []improvementSuggestion, getNotes noteGetter) string {
	var b strings.Builder
	b.WriteString("# Selected Improvements\n\n")
	currentSection := ""
	for _, suggestion := range suggestions {
		if suggestion.Section != currentSection {
			b.WriteString("## ")
			b.WriteString(suggestion.Section)
			b.WriteString("\n\n")
			currentSection = suggestion.Section
		}
		content := strings.Replace(suggestion.FullContent, "- [ ] APPROVE", "- [x] APPROVE", 1)
		b.WriteString(content)
		note := getNotes(suggestion.Index)
		if note != "" {
			b.WriteString("\n**User Notes:** ")
			b.WriteString(note)
			b.WriteString("\n")
		}
		b.WriteString("\n---\n\n")
	}
	return b.String()
}

func (s *Server) prepareTreesRetrospectiveDetails(dir, sessionName string) *treesRetroDetailsData {
	retrospectivesDir := filepath.Join(dir, ".sgai", "retrospectives")
	sessionDir := filepath.Join(retrospectivesDir, sessionName)

	goalPath := filepath.Join(sessionDir, "GOAL.md")
	improvementsPath := filepath.Join(sessionDir, "IMPROVEMENTS.md")

	goalSummary := stripMarkdownHeading(extractGoalSummary(goalPath))

	var goalContent template.HTML
	if data, err := os.ReadFile(goalPath); err == nil {
		normalized := normalizeEscapedNewlines(data)
		stripped := stripFrontmatter(string(normalized))
		if rendered, err := renderMarkdown([]byte(stripped)); err == nil {
			goalContent = template.HTML(rendered)
		}
	}

	var improvementsContent template.HTML
	hasImprovements := false
	if data, err := os.ReadFile(improvementsPath); err == nil {
		hasImprovements = true
		stripped := stripFrontmatter(string(data))
		if rendered, err := renderMarkdown([]byte(stripped)); err == nil {
			improvementsContent = template.HTML(rendered)
		}
	}

	sessionKey := "retro-analyze-" + dir + "-" + sessionName
	applyKey := "retro-apply-" + dir + "-" + sessionName

	s.mu.Lock()
	analyzeSession := s.sessions[sessionKey]
	applySession := s.sessions[applyKey]
	s.mu.Unlock()

	isAnalyzing := analyzeSession != nil && analyzeSession.running
	isApplying := applySession != nil && applySession.running

	return &treesRetroDetailsData{
		Directory:           dir,
		DirName:             filepath.Base(dir),
		SessionName:         sessionName,
		GoalSummary:         goalSummary,
		GoalContent:         goalContent,
		ImprovementsContent: improvementsContent,
		HasImprovements:     hasImprovements,
		IsAnalyzing:         isAnalyzing,
		IsApplying:          isApplying,
	}
}

func (s *Server) listRetrospectiveSessionsForProject(projectDir string) []retroSessionData {
	retrospectivesDir := filepath.Join(projectDir, ".sgai", "retrospectives")
	entries, err := os.ReadDir(retrospectivesDir)
	if err != nil {
		return nil
	}

	var sessions []retroSessionData
	for _, entry := range entries {
		if entry.IsDir() && retrospectiveDirPatternRE.MatchString(entry.Name()) {
			sessionDir := filepath.Join(retrospectivesDir, entry.Name())
			improvementsPath := filepath.Join(sessionDir, "IMPROVEMENTS.md")
			goalPath := filepath.Join(sessionDir, "GOAL.md")

			_, hasImprovements := os.Stat(improvementsPath)
			goalSummary := stripMarkdownHeading(extractGoalSummary(goalPath))

			sessions = append(sessions, retroSessionData{
				Name:            entry.Name(),
				HasImprovements: hasImprovements == nil,
				GoalSummary:     goalSummary,
			})
		}
	}

	slices.SortFunc(sessions, func(a, b retroSessionData) int {
		return strings.Compare(b.Name, a.Name)
	})

	return sessions
}

// retroSessionData represents a retrospective session with its metadata for UI rendering.
type retroSessionData struct {
	Name            string
	HasImprovements bool
	GoalSummary     string
}

// stripMarkdownHeading removes leading markdown heading characters from a string.
func stripMarkdownHeading(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#") {
		for len(s) > 0 && s[0] == '#' {
			s = s[1:]
		}
		s = strings.TrimSpace(s)
	}
	return s
}

func (s *Server) renderTreesRetrospectivesTab(w http.ResponseWriter, r *http.Request, dir string) {
	sessionParam := r.URL.Query().Get("session")
	sessions := s.listRetrospectiveSessionsForProject(dir)

	if sessionParam == "" && len(sessions) > 0 {
		sessionParam = sessions[0].Name
	}

	var detailsData *treesRetroDetailsData
	if sessionParam != "" {
		detailsData = s.prepareTreesRetrospectiveDetails(dir, sessionParam)
	}

	data := struct {
		Directory       string
		DirName         string
		Sessions        []retroSessionData
		SelectedSession string
		Details         *treesRetroDetailsData
	}{
		Directory:       dir,
		DirName:         filepath.Base(dir),
		Sessions:        sessions,
		SelectedSession: sessionParam,
		Details:         detailsData,
	}

	executeTemplate(w, templates.Lookup("trees_retrospectives_content.html"), data)
}

func (s *Server) renderTreesRetrospectivesTabToBuffer(buf *bytes.Buffer, _ *http.Request, dir, sessionParam string) {
	sessions := s.listRetrospectiveSessionsForProject(dir)

	if sessionParam == "" && len(sessions) > 0 {
		sessionParam = sessions[0].Name
	}

	var detailsData *treesRetroDetailsData
	if sessionParam != "" {
		detailsData = s.prepareTreesRetrospectiveDetails(dir, sessionParam)
	}

	data := struct {
		Directory       string
		DirName         string
		Sessions        []retroSessionData
		SelectedSession string
		Details         *treesRetroDetailsData
	}{
		Directory:       dir,
		DirName:         filepath.Base(dir),
		Sessions:        sessions,
		SelectedSession: sessionParam,
		Details:         detailsData,
	}

	if err := templates.Lookup("trees_retrospectives_content.html").Execute(buf, data); err != nil {
		log.Println("template execution failed:", err)
	}
}

func (s *Server) runWorkspaceRetrospectiveCommand(w http.ResponseWriter, r *http.Request, workspacePath, keyPrefix, subcommand, startErrorMsg string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	sessionID := r.FormValue("session")
	if sessionID == "" {
		http.Error(w, "Missing session", http.StatusBadRequest)
		return
	}

	redirectURL := workspaceURL(workspacePath, "retro") + "?session=" + sessionID
	sessionKey := keyPrefix + workspacePath + "-" + sessionID

	s.mu.Lock()
	sess := s.sessions[sessionKey]
	if sess != nil && sess.running {
		s.mu.Unlock()
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	sess = &session{running: true}
	s.sessions[sessionKey] = sess
	s.mu.Unlock()

	sgaiPath, err := os.Executable()
	if err != nil {
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
		http.Error(w, "Failed to find sgai executable", http.StatusInternalServerError)
		return
	}

	cmd := exec.Command(sgaiPath, "retrospective", subcommand, sessionID)
	cmd.Dir = workspacePath

	if err := cmd.Start(); err != nil {
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
		http.Error(w, startErrorMsg, http.StatusInternalServerError)
		return
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Println("retrospective", subcommand, "exited with error:", err)
		}
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
	}()

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (s *Server) handleWorkspaceRetroAnalyze(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	sessionID := r.FormValue("session")
	if sessionID == "" {
		http.Error(w, "Missing session", http.StatusBadRequest)
		return
	}

	sessionKey := "retro-analyze-" + workspacePath + "-" + sessionID

	s.mu.Lock()
	sess := s.sessions[sessionKey]
	if sess != nil && sess.running {
		s.mu.Unlock()
		http.Redirect(w, r, workspaceURL(workspacePath, "retro/"+sessionID+"/analyze"), http.StatusSeeOther)
		return
	}

	tempDir, err := os.MkdirTemp("", "sgai-retro-analyze-*")
	if err != nil {
		s.mu.Unlock()
		http.Error(w, "Failed to create temp directory", http.StatusInternalServerError)
		return
	}

	sess = &session{running: true, retroTempDir: tempDir}
	s.sessions[sessionKey] = sess
	s.mu.Unlock()

	sgaiPath, err := os.Executable()
	if err != nil {
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
		if err := os.RemoveAll(tempDir); err != nil {
			log.Println("cleanup failed:", err)
		}
		http.Error(w, "Failed to find sgai executable", http.StatusInternalServerError)
		return
	}

	cmd := exec.Command(sgaiPath, "retrospective", "analyze", "--temp-dir="+tempDir, sessionID)
	cmd.Dir = workspacePath
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
		if err := os.RemoveAll(tempDir); err != nil {
			log.Println("cleanup failed:", err)
		}
		http.Error(w, "Failed to create stdout pipe", http.StatusInternalServerError)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
		if err := os.RemoveAll(tempDir); err != nil {
			log.Println("cleanup failed:", err)
		}
		http.Error(w, "Failed to create stderr pipe", http.StatusInternalServerError)
		return
	}

	sess.cmd = cmd

	if err := cmd.Start(); err != nil {
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
		if err := os.RemoveAll(tempDir); err != nil {
			log.Println("cleanup failed:", err)
		}
		http.Error(w, "Failed to start analysis", http.StatusInternalServerError)
		return
	}

	go s.captureOutput(stdout, stderr, sessionKey, "[retro] ")

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Println("retrospective analyze exited with error:", err)
		}
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
	}()

	http.Redirect(w, r, workspaceURL(workspacePath, "retro/"+sessionID+"/analyze"), http.StatusSeeOther)
}

func (s *Server) handleWorkspaceRetroApply(w http.ResponseWriter, r *http.Request, workspacePath string) {
	s.runWorkspaceRetrospectiveCommand(w, r, workspacePath, "retro-apply-", "apply", "Failed to start apply")
}

func (s *Server) handleWorkspaceRetroApplySelect(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method == http.MethodPost {
		s.handleWorkspaceRetroApplySelectPost(w, r, workspacePath)
		return
	}

	sessionParam := r.URL.Query().Get("session")
	if sessionParam == "" {
		http.Error(w, "Missing session parameter", http.StatusBadRequest)
		return
	}

	retrospectivesDir := filepath.Join(workspacePath, ".sgai", "retrospectives")
	sessionDir := filepath.Join(retrospectivesDir, sessionParam)
	improvementsPath := filepath.Join(sessionDir, "IMPROVEMENTS.md")

	content, err := os.ReadFile(improvementsPath)
	if err != nil {
		http.Error(w, "IMPROVEMENTS.md not found", http.StatusNotFound)
		return
	}

	suggestions := parseImprovementSuggestions(string(content))

	data := struct {
		Directory   string
		DirName     string
		SessionName string
		Suggestions []improvementSuggestion
	}{
		Directory:   workspacePath,
		DirName:     filepath.Base(workspacePath),
		SessionName: sessionParam,
		Suggestions: suggestions,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("trees_retrospectives_apply_select.html"), data)
}

func (s *Server) handleWorkspaceRetroApplySelectPost(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	sessionID := r.FormValue("session")
	selectedIndices := r.Form["suggestions"]

	if sessionID == "" {
		http.Error(w, "Missing session", http.StatusBadRequest)
		return
	}

	redirectURL := workspaceURL(workspacePath, "retro") + "?session=" + sessionID

	if len(selectedIndices) == 0 {
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	retrospectivesDir := filepath.Join(workspacePath, ".sgai", "retrospectives")
	sessionDir := filepath.Join(retrospectivesDir, sessionID)
	improvementsPath := filepath.Join(sessionDir, "IMPROVEMENTS.md")

	content, err := os.ReadFile(improvementsPath)
	if err != nil {
		http.Error(w, "IMPROVEMENTS.md not found", http.StatusNotFound)
		return
	}

	suggestions := parseImprovementSuggestions(string(content))
	selectedSuggestions := filterSelectedSuggestions(suggestions, selectedIndices)
	selectedContent := buildSelectedImprovementsContent(selectedSuggestions, func(idx int) string {
		return strings.TrimSpace(r.FormValue(fmt.Sprintf("notes-%d", idx)))
	})

	sessionKey := "retro-apply-" + workspacePath + "-" + sessionID

	s.mu.Lock()
	sess := s.sessions[sessionKey]
	if sess != nil && sess.running {
		s.mu.Unlock()
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	sess = &session{running: true}
	s.sessions[sessionKey] = sess
	s.mu.Unlock()

	sgaiPath, err := os.Executable()
	if err != nil {
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
		http.Error(w, "Failed to find sgai executable", http.StatusInternalServerError)
		return
	}

	cmd := exec.Command(sgaiPath, "retrospective", "apply", "--selected", sessionID)
	cmd.Dir = workspacePath
	cmd.Stdin = strings.NewReader(selectedContent)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
		http.Error(w, "Failed to create stdout pipe", http.StatusInternalServerError)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
		http.Error(w, "Failed to create stderr pipe", http.StatusInternalServerError)
		return
	}

	sess.cmd = cmd

	if err := cmd.Start(); err != nil {
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
		http.Error(w, "Failed to start apply", http.StatusInternalServerError)
		return
	}

	go s.captureOutput(stdout, stderr, sessionKey, "[retro] ")

	go func() {
		errApply := cmd.Wait()
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()

		if errApply != nil {
			log.Println("retrospective apply exited with error:", errApply)
			return
		}

		if err := deleteRetrospectiveSession(workspacePath, sessionID); err != nil {
			log.Println("failed to auto-delete retrospective session:", sessionID, err)
		}
	}()

	http.Redirect(w, r, workspaceURL(workspacePath, "retro/"+sessionID+"/apply"), http.StatusSeeOther)
}

func (s *Server) handleWorkspaceRetroDelete(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	sessionID := r.FormValue("session")
	if sessionID == "" {
		http.Error(w, "Missing session parameter", http.StatusBadRequest)
		return
	}

	if err := deleteRetrospectiveSession(workspacePath, sessionID); err != nil {
		http.Error(w, "Failed to delete retrospective: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.renderTreesRetrospectivesTab(w, r, workspacePath)
}

// deleteRetrospectiveSession removes the retrospective session directory.
func deleteRetrospectiveSession(workspacePath, sessionID string) error {
	retrospectivesDir := filepath.Join(workspacePath, ".sgai", "retrospectives")
	sessionDir := filepath.Join(retrospectivesDir, sessionID)

	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	return os.RemoveAll(sessionDir)
}

func (s *Server) routeWorkspaceRetro(w http.ResponseWriter, r *http.Request, workspacePath, subPath string) {
	parts := strings.SplitN(subPath, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	sessionID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch action {
	case "analyze":
		s.handleWorkspaceRetroAnalyzePage(w, r, workspacePath, sessionID)
	case "analyze/status":
		s.handleWorkspaceRetroAnalyzeStatus(w, r, workspacePath, sessionID)
	case "analyze/stop":
		s.handleWorkspaceRetroAnalyzeStop(w, r, workspacePath, sessionID)
	case "apply":
		s.handleWorkspaceRetroApplyPage(w, r, workspacePath, sessionID)
	case "apply/status":
		s.handleWorkspaceRetroApplyStatus(w, r, workspacePath, sessionID)
	case "apply/stop":
		s.handleWorkspaceRetroApplyStop(w, r, workspacePath, sessionID)
	default:
		http.NotFound(w, r)
	}
}

// retroAnalyzeData holds data for rendering the retrospective analyze page.
type retroAnalyzeData struct {
	Directory           string
	DirName             string
	SessionID           string
	Running             bool
	Progress            []eventsProgressDisplay
	Logs                []retroLogEntry
	WorkspaceID         string
	ImprovementsPreview template.HTML
}

// retroLogEntry represents a single log entry with prefix and text.
type retroLogEntry struct {
	Prefix string
	Text   string
}

func (s *Server) renderImprovementsPreview(improvementsPath string) template.HTML {
	content, err := os.ReadFile(improvementsPath)
	if err != nil {
		return ""
	}

	stripped := stripFrontmatter(string(content))
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, emoji.Emoji),
		goldmark.WithRendererOptions(html.WithHardWraps()),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(stripped), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(stripped))
	}
	return template.HTML(buf.String())
}

func (s *Server) prepareRetroAnalyzeData(workspacePath, sessionID string) retroAnalyzeData {
	sessionKey := "retro-analyze-" + workspacePath + "-" + sessionID

	s.mu.Lock()
	sess := s.sessions[sessionKey]
	s.mu.Unlock()

	var running bool
	var tempDir string
	if sess != nil {
		sess.mu.Lock()
		running = sess.running
		tempDir = sess.retroTempDir
		sess.mu.Unlock()
	}

	var progress []eventsProgressDisplay
	var improvementsPreview template.HTML

	sessionImprovementsPath := filepath.Join(workspacePath, ".sgai", "retrospectives", sessionID, "IMPROVEMENTS.md")

	if running && tempDir != "" {
		wfState, _ := state.Load(filepath.Join(tempDir, ".sgai", "state.json"))
		reversedProgress := slices.Clone(wfState.Progress)
		slices.Reverse(reversedProgress)
		progress = formatProgressForDisplay(reversedProgress)
		improvementsPreview = s.renderImprovementsPreview(filepath.Join(tempDir, "IMPROVEMENTS.md"))
	} else {
		improvementsPreview = s.renderImprovementsPreview(sessionImprovementsPath)
	}

	var logs []retroLogEntry
	if sess != nil && sess.outputLog != nil {
		lines := sess.outputLog.lines()
		for _, line := range lines {
			logs = append(logs, retroLogEntry{Prefix: line.prefix, Text: line.text})
		}
	}

	return retroAnalyzeData{
		Directory:           workspacePath,
		DirName:             filepath.Base(workspacePath),
		SessionID:           sessionID,
		Running:             running,
		Progress:            progress,
		Logs:                logs,
		WorkspaceID:         filepath.Base(workspacePath),
		ImprovementsPreview: improvementsPreview,
	}
}

func (s *Server) handleWorkspaceRetroAnalyzePage(w http.ResponseWriter, _ *http.Request, workspacePath, sessionID string) {
	data := s.prepareRetroAnalyzeData(workspacePath, sessionID)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("retro_analyze_page.html"), data)
}

func (s *Server) handleWorkspaceRetroAnalyzeStatus(w http.ResponseWriter, _ *http.Request, workspacePath, sessionID string) {
	data := s.prepareRetroAnalyzeData(workspacePath, sessionID)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("retro_analyze_status.html"), data)
}

func (s *Server) handleWorkspaceRetroAnalyzeStop(w http.ResponseWriter, r *http.Request, workspacePath, sessionID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	sessionKey := "retro-analyze-" + workspacePath + "-" + sessionID

	s.mu.Lock()
	sess := s.sessions[sessionKey]
	s.mu.Unlock()

	if sess == nil {
		http.Redirect(w, r, workspaceURL(workspacePath, "retro/"+sessionID+"/analyze"), http.StatusSeeOther)
		return
	}

	sess.mu.Lock()
	running := sess.running
	cmd := sess.cmd
	tempDir := sess.retroTempDir
	sess.mu.Unlock()

	if !running {
		http.Redirect(w, r, workspaceURL(workspacePath, "retro?session="+sessionID), http.StatusSeeOther)
		return
	}

	if cmd != nil && cmd.Process != nil {
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
			log.Println("signal failed:", err)
		}
	}

	if tempDir != "" {
		if err := os.RemoveAll(tempDir); err != nil {
			log.Println("cleanup failed:", err)
		}
	}

	sess.mu.Lock()
	sess.running = false
	sess.cmd = nil
	sess.retroTempDir = ""
	sess.mu.Unlock()

	http.Redirect(w, r, workspaceURL(workspacePath, "retro?session="+sessionID), http.StatusSeeOther)
}

type retroApplyData struct {
	Directory   string
	DirName     string
	SessionID   string
	Running     bool
	Logs        []retroLogEntry
	WorkspaceID string
}

func (s *Server) prepareRetroApplyData(workspacePath, sessionID string) retroApplyData {
	sessionKey := "retro-apply-" + workspacePath + "-" + sessionID

	s.mu.Lock()
	sess := s.sessions[sessionKey]
	s.mu.Unlock()

	var running bool
	if sess != nil {
		sess.mu.Lock()
		running = sess.running
		sess.mu.Unlock()
	}

	var logs []retroLogEntry
	if sess != nil && sess.outputLog != nil {
		lines := sess.outputLog.lines()
		for _, line := range lines {
			logs = append(logs, retroLogEntry{Prefix: line.prefix, Text: line.text})
		}
	}

	return retroApplyData{
		Directory:   workspacePath,
		DirName:     filepath.Base(workspacePath),
		SessionID:   sessionID,
		Running:     running,
		Logs:        logs,
		WorkspaceID: filepath.Base(workspacePath),
	}
}

func (s *Server) handleWorkspaceRetroApplyPage(w http.ResponseWriter, _ *http.Request, workspacePath, sessionID string) {
	data := s.prepareRetroApplyData(workspacePath, sessionID)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("retro_apply_page.html"), data)
}

func (s *Server) handleWorkspaceRetroApplyStatus(w http.ResponseWriter, _ *http.Request, workspacePath, sessionID string) {
	data := s.prepareRetroApplyData(workspacePath, sessionID)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("retro_apply_status.html"), data)
}

func (s *Server) handleWorkspaceRetroApplyStop(w http.ResponseWriter, r *http.Request, workspacePath, sessionID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	sessionKey := "retro-apply-" + workspacePath + "-" + sessionID

	s.mu.Lock()
	sess := s.sessions[sessionKey]
	s.mu.Unlock()

	if sess == nil {
		http.Redirect(w, r, workspaceURL(workspacePath, "retro/"+sessionID+"/apply"), http.StatusSeeOther)
		return
	}

	sess.mu.Lock()
	running := sess.running
	cmd := sess.cmd
	sess.mu.Unlock()

	if !running {
		http.Redirect(w, r, workspaceURL(workspacePath, "retro?session="+sessionID), http.StatusSeeOther)
		return
	}

	if cmd != nil && cmd.Process != nil {
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
			log.Println("signal failed:", err)
		}
	}

	sess.mu.Lock()
	sess.running = false
	sess.cmd = nil
	sess.mu.Unlock()

	http.Redirect(w, r, workspaceURL(workspacePath, "retro?session="+sessionID), http.StatusSeeOther)
}
