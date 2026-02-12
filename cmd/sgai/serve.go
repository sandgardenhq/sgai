package main

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/adrg/xdg"
	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

//go:embed GOAL.example.md
var goalExampleContent string

var tmplFallbackSVG = template.Must(template.New("fallbackSVG").Parse(
	`<svg xmlns="http://www.w3.org/2000/svg" width="400" height="{{.Height}}" viewBox="0 0 400 {{.Height}}">
<rect width="100%" height="100%" fill="#f8fafc"/>
<text x="10" y="20" font-family="monospace" font-size="12" fill="#475569">{{range .Lines}}<tspan x="10" dy="{{.DY}}">{{.Text}}</tspan>{{end}}</text>
</svg>`))

type project struct {
	Directory    string
	DirName      string
	LastModified time.Time
	HasWorkspace bool
}

func scanForProjects(rootDir string) ([]project, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}

	var projects []project
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := filepath.Join(rootDir, entry.Name())
		sgaiDir := filepath.Join(dirPath, ".sgai")

		hasWorkspace := false
		var modTime time.Time
		sgaiInfo, errsgai := os.Stat(sgaiDir)
		if errsgai == nil && sgaiInfo.IsDir() {
			hasWorkspace = true
			modTime = sgaiInfo.ModTime()
		} else {
			entryInfo, errEntry := entry.Info()
			if errEntry == nil {
				modTime = entryInfo.ModTime()
			}
		}

		projects = append(projects, project{
			Directory:    dirPath,
			DirName:      entry.Name(),
			LastModified: modTime,
			HasWorkspace: hasWorkspace,
		})
	}

	slices.SortFunc(projects, func(a, b project) int {
		return strings.Compare(strings.ToLower(a.DirName), strings.ToLower(b.DirName))
	})

	return projects, nil
}

func stripFrontmatter(content string) string {
	delimiter := "---"

	if !strings.HasPrefix(content, delimiter) {
		return content
	}

	rest := content[len(delimiter):]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	_, after, ok := strings.Cut(rest, delimiter)
	if !ok {
		return content
	}

	afterClosing := after
	return strings.TrimLeft(afterClosing, "\n")
}

type session struct {
	mu              sync.Mutex
	cmd             *exec.Cmd
	running         bool
	interactiveAuto bool
	outputLog       *circularLogBuffer
	retroTempDir    string
}

type editorOpener interface {
	open(path string) error
}

const defaultEditorPreset = "code"

// editorPreset defines a preset editor configuration with its command template
// and whether it runs in a terminal.
type editorPreset struct {
	command    string
	isTerminal bool
}

var editorPresets = map[string]editorPreset{
	"code":   {command: "code", isTerminal: false},
	"cursor": {command: "cursor", isTerminal: false},
	"zed":    {command: "zed", isTerminal: false},
	"subl":   {command: "subl", isTerminal: false},
	"idea":   {command: "idea", isTerminal: false},
	"emacs":  {command: "emacsclient -n", isTerminal: false},
	"nvim":   {command: "nvim", isTerminal: true},
	"vim":    {command: "vim", isTerminal: true},
	"atom":   {command: "atom", isTerminal: false},
}

// configurableEditor implements editorOpener with configurable editor support.
type configurableEditor struct {
	name       string
	command    string
	isTerminal bool
}

func (e *configurableEditor) open(path string) error {
	cmd := exec.Command(e.command, path)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func resolveEditor(configEditor string) (name, command string, isTerminal bool) {
	editorSpec := configEditor
	if editorSpec == "" {
		editorSpec = os.Getenv("VISUAL")
	}
	if editorSpec == "" {
		editorSpec = os.Getenv("EDITOR")
	}
	if editorSpec == "" {
		editorSpec = defaultEditorPreset
	}

	if preset, ok := editorPresets[editorSpec]; ok {
		return editorSpec, preset.command, preset.isTerminal
	}

	return editorSpec, editorSpec, false
}

func newConfigurableEditor(configEditor string) *configurableEditor {
	name, command, isTerminal := resolveEditor(configEditor)
	return &configurableEditor{
		name:       name,
		command:    command,
		isTerminal: isTerminal,
	}
}

func isEditorAvailable(command string) bool {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}
	_, err := exec.LookPath(parts[0])
	return err == nil
}

// Server handles HTTP requests for the sgai serve command.
type Server struct {
	mu               sync.Mutex
	sessions         map[string]*session
	everStartedDirs  map[string]bool
	pinnedDirs       map[string]bool
	pinnedConfigDir  string
	rootDir          string
	editorAvailable  bool
	isTerminalEditor bool
	editorName       string
	editor           editorOpener

	sseBroker *sseBroker

	adhocStates map[string]*adhocPromptState
}

// NewServer creates a new Server instance with the given root directory.
// It converts rootDir to an absolute path to ensure consistent path comparisons
// between cookie values (set via validateDirectory) and template values
// (set via scanForProjects).
func NewServer(rootDir string) *Server {
	return NewServerWithConfig(rootDir, "")
}

// NewServerWithConfig creates a new Server with a specific editor configuration.
func NewServerWithConfig(rootDir, editorConfig string) *Server {
	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		absRootDir = rootDir
	}
	editor := newConfigurableEditor(editorConfig)
	editorAvail := isEditorAvailable(editor.command)
	if !editorAvail {
		fallback := newConfigurableEditor(defaultEditorPreset)
		if isEditorAvailable(fallback.command) {
			editor = fallback
			editorAvail = true
		}
	}
	return &Server{
		sessions:         make(map[string]*session),
		everStartedDirs:  make(map[string]bool),
		pinnedDirs:       make(map[string]bool),
		pinnedConfigDir:  filepath.Join(xdg.ConfigHome, "sgai"),
		adhocStates:      make(map[string]*adhocPromptState),
		sseBroker:        newSSEBroker(),
		rootDir:          absRootDir,
		editorAvailable:  editorAvail,
		isTerminalEditor: editor.isTerminal,
		editorName:       editor.name,
		editor:           editor,
	}
}

func (s *Server) validateDirectory(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("directory is required")
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("invalid directory path: %w", err)
	}

	absRoot, err := filepath.Abs(s.rootDir)
	if err != nil {
		return "", fmt.Errorf("invalid root path: %w", err)
	}

	cleanDir := filepath.Clean(absDir)
	cleanRoot := filepath.Clean(absRoot)

	realRoot, err := filepath.EvalSymlinks(cleanRoot)
	if err != nil {
		realRoot = cleanRoot
	}

	realDir, err := filepath.EvalSymlinks(cleanDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("invalid directory path: %w", err)
		}
		parent := cleanDir
		var nonExistentParts []string
		for {
			parentDir := filepath.Dir(parent)
			if parentDir == parent {
				break
			}
			realParent, err := filepath.EvalSymlinks(parentDir)
			if err == nil {
				realDir = realParent
				for i := len(nonExistentParts) - 1; i >= 0; i-- {
					realDir = filepath.Join(realDir, nonExistentParts[i])
				}
				realDir = filepath.Join(realDir, filepath.Base(parent))
				break
			}
			nonExistentParts = append(nonExistentParts, filepath.Base(parent))
			parent = parentDir
		}
		if realDir == "" {
			realDir = cleanDir
		}
	}

	relPath, err := filepath.Rel(realRoot, realDir)
	if err != nil {
		return "", fmt.Errorf("path traversal denied")
	}

	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("path traversal denied")
	}

	return cleanDir, nil
}

func statePath(dir string) string {
	return filepath.Join(dir, ".sgai", "state.json")
}

// isLocalRequest returns true if the HTTP request originates from localhost
// (127.0.0.1 for IPv4 or ::1 for IPv6).
func isLocalRequest(r *http.Request) bool {
	remoteAddr := r.RemoteAddr

	if strings.HasPrefix(remoteAddr, "[") {
		bracketEnd := strings.Index(remoteAddr, "]")
		if bracketEnd == -1 {
			return false
		}
		host := remoteAddr[1:bracketEnd]
		return host == "::1"
	}

	host, _, found := strings.Cut(remoteAddr, ":")
	if !found {
		return false
	}
	return host == "127.0.0.1"
}

type startSessionResult struct {
	alreadyRunning bool
	sess           *session
	startError     error
}

func (s *Server) startSession(workspacePath string, autoMode bool) startSessionResult {
	s.mu.Lock()
	sess := s.sessions[workspacePath]
	if sess != nil && sess.running {
		s.mu.Unlock()
		return startSessionResult{alreadyRunning: true, sess: sess}
	}

	sess = &session{running: true, interactiveAuto: autoMode}
	s.sessions[workspacePath] = sess
	s.everStartedDirs[workspacePath] = true
	s.mu.Unlock()

	resetHumanCommunication(workspacePath)

	sgaiPath, err := os.Executable()
	if err != nil {
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
		return startSessionResult{startError: fmt.Errorf("failed to find sgai executable")}
	}

	interactiveFlag := "--interactive=yes"
	if autoMode {
		interactiveFlag = "--interactive=auto"
	}
	cmd := exec.Command(sgaiPath, interactiveFlag, workspacePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
		return startSessionResult{startError: fmt.Errorf("failed to create stdout pipe")}
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
		return startSessionResult{startError: fmt.Errorf("failed to create stderr pipe")}
	}

	sess.cmd = cmd

	if err := cmd.Start(); err != nil {
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
		return startSessionResult{startError: fmt.Errorf("failed to start sgai")}
	}

	go s.captureOutput(stdout, stderr, workspacePath, fmt.Sprintf("[%s] ", filepath.Base(workspacePath)))

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("sgai process exited with error: %v", err)
		}
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
		s.clearEverStartedOnCompletion(workspacePath)
	}()

	return startSessionResult{sess: sess}
}

func (s *Server) stopSession(workspacePath string) {
	s.mu.Lock()
	sess := s.sessions[workspacePath]
	s.mu.Unlock()

	if sess != nil {
		sess.mu.Lock()
		if sess.running && sess.cmd != nil && sess.cmd.Process != nil {
			if err := syscall.Kill(-sess.cmd.Process.Pid, syscall.SIGTERM); err != nil {
				log.Println("signal failed:", err)
			}
		}
		sess.running = false
		sess.mu.Unlock()
	}

	resetHumanCommunication(workspacePath)
}

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

func cmdServe(args []string) {
	serveFlags := flag.NewFlagSet("serve", flag.ExitOnError)
	listenAddr := serveFlags.String("listen-addr", "127.0.0.1:8080", "HTTP server listen address")
	serveFlags.Parse(args) //nolint:errcheck // ExitOnError FlagSet exits on error, never returns non-nil

	var rootDir string
	remainingArgs := serveFlags.Args()
	if len(remainingArgs) > 0 {
		rootDir = remainingArgs[0]
	} else {
		var err error
		rootDir, err = os.Getwd()
		if err != nil {
			log.Fatalf("failed to get working directory: %v", err)
		}
	}

	srv := NewServer(rootDir)
	if err := srv.loadPinnedProjects(); err != nil {
		log.Println("warning: failed to load pinned projects:", err)
	}
	srv.startStateWatcher()

	mux := http.NewServeMux()
	srv.registerAPIRoutes(mux)
	handler := srv.spaMiddleware(mux)

	baseURL := "http://" + *listenAddr
	log.Println("sgai serve listening on", baseURL)

	go func() {
		if err := http.ListenAndServe(*listenAddr, handler); err != nil {
			log.Fatalln("server error:", err)
		}
	}()

	startMenuBar(baseURL, srv)
}

func renderDotToSVG(dotContent string) string {
	dotPath, err := exec.LookPath("dot")
	if err != nil {
		return renderDotAsFallbackSVG(dotContent)
	}

	cmd := exec.Command(dotPath, "-Tsvg")
	cmd.Stdin = strings.NewReader(dotContent)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return renderDotAsFallbackSVG(dotContent)
	}
	return out.String()
}

func renderDotAsFallbackSVG(dotContent string) string {
	lines := linesWithTrailingEmpty(dotContent)
	height := max(20+len(lines)*16, 100)

	type lineData struct {
		DY   int
		Text string
	}
	var lineItems []lineData
	for i, line := range lines {
		dy := 16
		if i == 0 {
			dy = 0
		}
		lineItems = append(lineItems, lineData{DY: dy, Text: line})
	}

	var buf bytes.Buffer
	data := struct {
		Height int
		Lines  []lineData
	}{
		Height: height,
		Lines:  lineItems,
	}
	if err := tmplFallbackSVG.Execute(&buf, data); err != nil {
		return ""
	}
	return buf.String()
}

func linesWithTrailingEmpty(content string) []string {
	var lines []string
	for line := range strings.Lines(content) {
		lines = append(lines, strings.TrimSuffix(line, "\n"))
	}
	if content == "" || strings.HasSuffix(content, "\n") {
		lines = append(lines, "")
	}
	return lines
}

func getWorkflowSVGHash(dir string, currentAgent string) string {
	svg := getWorkflowSVG(dir, currentAgent)
	if svg == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(svg))
	return hex.EncodeToString(hash[:8])
}

func getWorkflowSVG(dir string, currentAgent string) string {
	goalPath := filepath.Join(dir, "GOAL.md")
	goalData, err := os.ReadFile(goalPath)
	if err != nil {
		return ""
	}

	metadata, err := parseYAMLFrontmatter(goalData)
	if err != nil {
		return ""
	}

	d, err := parseFlow(metadata.Flow, dir)
	if err != nil {
		return ""
	}

	dotContent := d.toDOT()

	if currentAgent != "" {
		dotContent = injectCurrentAgentStyle(dotContent, currentAgent)
	}
	dotContent = injectLightTheme(dotContent)

	return renderDotToSVG(dotContent)
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

func renderMarkdown(content []byte) (string, error) {
	var buf bytes.Buffer
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, emoji.Emoji),
		goldmark.WithRendererOptions(html.WithHardWraps()),
	)
	if err := md.Convert(content, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

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
			log.Printf("prepareAgentSequenceDisplay: skipping entry with invalid timestamp %q: %v", entry.StartTime, err)
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

func resetHumanCommunication(dir string) {
	wfState, err := state.Load(statePath(dir))
	if err != nil {
		return
	}
	wfState.HumanMessage = ""
	if state.IsHumanPending(wfState.Status) {
		wfState.Status = state.StatusWorking
	}
	if err := state.Save(statePath(dir), wfState); err != nil {
		log.Println("failed to save state:", err)
	}
}

func extractSubject(body string) string {
	for _, line := range linesWithTrailingEmpty(body) {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return strings.TrimLeft(trimmed, "# ")
		}
	}
	return ""
}

func injectCurrentAgentStyle(dot, currentAgent string) string {
	agentLine := fmt.Sprintf(`    "%s"`, currentAgent)
	styledLine := fmt.Sprintf(`    "%s" [style=filled, fillcolor="#10b981", fontcolor=white]`, currentAgent)

	if !strings.Contains(dot, agentLine) {
		return dot
	}

	return strings.Replace(dot, agentLine, styledLine, 1)
}

func injectLightTheme(dot string) string {
	lightTheme := `    bgcolor="transparent"
    node [style=filled, fillcolor="#e2e8f0", fontcolor="#1e293b", color="#94a3b8"]
    edge [color="#64748b", fontcolor="#475569"]`

	braceIdx := strings.Index(dot, "{")
	if braceIdx == -1 {
		return dot
	}

	return dot[:braceIdx+1] + "\n" + lightTheme + dot[braceIdx+1:]
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

type workspaceInfo struct {
	Directory    string
	DirName      string
	IsRoot       bool
	Running      bool
	NeedsInput   bool
	InProgress   bool
	Pinned       bool
	HasWorkspace bool
}

type workspaceGroup struct {
	Root  workspaceInfo
	Forks []workspaceInfo
}

type workspaceKind string

const (
	workspaceStandalone workspaceKind = "standalone"
	workspaceRoot       workspaceKind = "root"
	workspaceFork       workspaceKind = "fork"
)

func classifyWorkspace(dir string) workspaceKind {
	repoPath := filepath.Join(dir, ".jj", "repo")
	info, errStat := os.Stat(repoPath)
	if errStat != nil {
		return workspaceStandalone
	}
	if !info.IsDir() {
		return workspaceFork
	}
	cmd := exec.Command("jj", "workspace", "list")
	cmd.Dir = dir
	output, errExec := cmd.Output()
	if errExec != nil {
		return workspaceStandalone
	}
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return workspaceStandalone
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) > 1 {
		return workspaceRoot
	}
	return workspaceStandalone
}

func hassgaiDirectory(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".sgai"))
	return err == nil && info.IsDir()
}

func getRootWorkspacePath(forkDir string) string {
	repoPath := filepath.Join(forkDir, ".jj", "repo")
	content, err := os.ReadFile(repoPath)
	if err != nil {
		return ""
	}
	rootPath := strings.TrimSpace(string(content))
	if rootPath == "" {
		return ""
	}
	jjDir := filepath.Dir(rootPath)
	return filepath.Dir(jjDir)
}

type jjCommit struct {
	ChangeID      string
	CommitID      string
	Workspaces    []string
	Timestamp     string
	Bookmarks     []string
	Description   string
	GraphChar     string
	HasLine       bool
	GraphLines    []string
	TrailingGraph []string
}

const jjLogTemplate = `change_id.short(8) ++ " " ++ commit_id.short(8) ++ " " ++ if(working_copies, working_copies.map(|wc| wc.name()).join(" ") ++ " ", "") ++ author.timestamp().ago() ++ if(bookmarks, " " ++ bookmarks.join(" "), "") ++ "\n  " ++ coalesce(description.first_line(), "(no description)") ++ "\n"`

var timestampUnits = []string{"second", "seconds", "minute", "minutes", "hour", "hours", "day", "days", "week", "weeks", "month", "months", "year", "years", "ago"}

func resolveBaseBookmark(rootDir string) string {
	for _, candidate := range []string{"main", "trunk"} {
		cmd := exec.Command("jj", "log", "-r", candidate, "--no-graph", "-T", "change_id")
		cmd.Dir = rootDir
		if errRun := cmd.Run(); errRun == nil {
			return candidate
		}
	}
	return "main"
}

func runJJLogForFork(bookmark, forkDir string) []jjCommit {
	revset := fmt.Sprintf("%s..@", bookmark)
	cmd := exec.Command("jj", "log", "-r", revset, "-T", jjLogTemplate)
	cmd.Dir = forkDir
	output, errCmd := cmd.Output()
	if errCmd != nil {
		return nil
	}
	return parseJJLogOutput(string(output))
}

func countForkCommitsAhead(bookmark, forkDir string) int {
	revset := fmt.Sprintf("ancestors(@, 2) ~ ancestors(%s@, 2)", bookmark)
	cmd := exec.Command("jj", "log", "-r", revset, "--no-graph", "-T", "change_id ++ \"\\n\"")
	cmd.Dir = forkDir
	output, errCmd := cmd.Output()
	if errCmd != nil {
		return 0
	}
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return 0
	}
	return len(strings.Split(trimmed, "\n"))
}

func parseJJLogOutput(output string) []jjCommit {
	var commits []jjCommit
	lines := linesWithTrailingEmpty(output)

	var currentCommit *jjCommit
	for i, line := range lines {
		if line == "" {
			continue
		}

		if isCommitHeaderLine(line) {
			if currentCommit != nil {
				commits = append(commits, *currentCommit)
			}
			currentCommit = parseCommitHeader(line)
			currentCommit.HasLine = hasNextCommit(lines, i)
			currentCommit.GraphLines = []string{extractGraphPrefix(line)}
		} else if currentCommit != nil {
			strippedContent := stripGraphPrefix(line)
			graphPrefix := extractGraphPrefix(line)

			if currentCommit.Description == "" && strippedContent != "" {
				currentCommit.Description = strings.TrimSpace(strippedContent)
			}

			if graphPrefix != "" {
				currentCommit.TrailingGraph = append(currentCommit.TrailingGraph, graphPrefix)
			}
		}
	}

	if currentCommit != nil {
		commits = append(commits, *currentCommit)
	}

	return commits
}

func isCommitMarker(r rune) bool {
	return r == '○' || r == '×' || r == '@' || r == '◆' || r == '~'
}

func isCommitHeaderLine(line string) bool {
	if len(line) < 3 {
		return false
	}
	for _, r := range line {
		if isCommitMarker(r) {
			return true
		}
		if !isGraphChar(r) {
			return false
		}
	}
	return false
}

func isGraphChar(r rune) bool {
	return r == '│' || r == '├' || r == '─' || r == '┘' || r == ' '
}

func stripGraphPrefix(line string) string {
	runes := []rune(line)
	for i, r := range runes {
		if !isGraphChar(r) {
			return string(runes[i:])
		}
	}
	return ""
}

func extractGraphPrefix(line string) string {
	runes := []rune(line)
	for i, r := range runes {
		if !isGraphChar(r) && !isCommitMarker(r) {
			return strings.TrimRight(string(runes[:i]), " ")
		}
	}
	return strings.TrimRight(line, " ")
}

func hasNextCommit(lines []string, currentIdx int) bool {
	return slices.ContainsFunc(lines[currentIdx+1:], isCommitHeaderLine)
}

func findCommitMarker(line string) (marker rune, restOfLine string) {
	runes := []rune(line)
	for i, r := range runes {
		if isCommitMarker(r) {
			return r, strings.TrimSpace(string(runes[i+1:]))
		}
	}
	return 0, line
}

func parseCommitHeader(line string) *jjCommit {
	commit := &jjCommit{}

	marker, rest := findCommitMarker(line)
	if marker == 0 {
		return commit
	}
	commit.GraphChar = string(marker)

	parts := strings.Fields(rest)
	if len(parts) < 2 {
		return commit
	}

	commit.ChangeID = parts[0]
	commit.CommitID = parts[1]

	remaining := parts[2:]

	for i := 0; i < len(remaining); i++ {
		part := remaining[i]

		if isTimestamp(part) {
			commit.Timestamp = part
			for i+1 < len(remaining) && isTimestampUnit(remaining[i+1]) {
				commit.Timestamp += " " + remaining[i+1]
				i++
			}
			continue
		}

		if strings.HasSuffix(part, "*") || isBookmark(part) {
			commit.Bookmarks = append(commit.Bookmarks, part)
			continue
		}

		if !isTimestamp(part) && !isTimestampUnit(part) && len(commit.Workspaces) == 0 {
			commit.Workspaces = append(commit.Workspaces, part)
		}
	}

	return commit
}

func isTimestamp(s string) bool {
	if len(s) == 0 {
		return false
	}
	first := s[0]
	return first >= '0' && first <= '9'
}

func isTimestampUnit(s string) bool {
	for _, u := range timestampUnits {
		if strings.HasPrefix(s, u) {
			return true
		}
	}
	return false
}

func isBookmark(s string) bool {
	return strings.Contains(s, "@") || strings.Contains(s, "/")
}

func (s *Server) getWorkspaceStatus(dir string) (running bool, needsInput bool) {
	s.mu.Lock()
	sess := s.sessions[dir]
	s.mu.Unlock()
	if sess != nil {
		sess.mu.Lock()
		running = sess.running
		sess.mu.Unlock()
	}

	wfState, _ := state.Load(statePath(dir))
	needsInput = wfState.NeedsHumanInput()
	return running, needsInput
}

func (s *Server) createWorkspaceInfo(dir, dirName string, isRoot, hasWorkspace bool) workspaceInfo {
	running, needsInput := s.getWorkspaceStatus(dir)
	pinned := s.isPinned(dir)
	inProgress := running || needsInput || s.wasEverStarted(dir) || pinned

	return workspaceInfo{
		Directory:    dir,
		DirName:      dirName,
		IsRoot:       isRoot,
		Running:      running,
		NeedsInput:   needsInput,
		InProgress:   inProgress,
		Pinned:       pinned,
		HasWorkspace: hasWorkspace,
	}
}

func (s *Server) wasEverStarted(dir string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.everStartedDirs[dir]
}

func (s *Server) clearEverStartedOnCompletion(dir string) {
	wfState, err := state.Load(statePath(dir))
	if err != nil {
		return
	}
	if wfState.Status != state.StatusComplete {
		return
	}
	s.mu.Lock()
	delete(s.everStartedDirs, dir)
	s.mu.Unlock()
}

func (s *Server) pinnedFilePath() string {
	return filepath.Join(s.pinnedConfigDir, "pinned.json")
}

func (s *Server) loadPinnedProjects() error {
	data, errRead := os.ReadFile(s.pinnedFilePath())
	if errRead != nil {
		if os.IsNotExist(errRead) {
			return nil
		}
		return fmt.Errorf("reading pinned projects: %w", errRead)
	}
	var dirs []string
	if errJSON := json.Unmarshal(data, &dirs); errJSON != nil {
		return fmt.Errorf("parsing pinned projects: %w", errJSON)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pinnedDirs = make(map[string]bool, len(dirs))
	for _, d := range dirs {
		s.pinnedDirs[d] = true
	}
	return nil
}

func (s *Server) savePinnedProjects() error {
	s.mu.Lock()
	dirs := slices.Collect(maps.Keys(s.pinnedDirs))
	s.mu.Unlock()
	slices.Sort(dirs)
	if errDir := os.MkdirAll(s.pinnedConfigDir, 0o755); errDir != nil {
		return fmt.Errorf("creating pin config directory: %w", errDir)
	}
	data, errJSON := json.Marshal(dirs)
	if errJSON != nil {
		return fmt.Errorf("encoding pinned projects: %w", errJSON)
	}
	if errWrite := os.WriteFile(s.pinnedFilePath(), data, 0o644); errWrite != nil {
		return fmt.Errorf("writing pinned projects: %w", errWrite)
	}
	return nil
}

func (s *Server) isPinned(dir string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pinnedDirs[dir]
}

func (s *Server) togglePin(dir string) error {
	s.mu.Lock()
	if s.pinnedDirs[dir] {
		delete(s.pinnedDirs, dir)
	} else {
		s.pinnedDirs[dir] = true
	}
	s.mu.Unlock()
	return s.savePinnedProjects()
}

func (s *Server) scanWorkspaceGroups() ([]workspaceGroup, error) {
	projects, err := scanForProjects(s.rootDir)
	if err != nil {
		return nil, err
	}

	rootMap := make(map[string]*workspaceGroup)
	var standaloneGroups []workspaceGroup

	for _, proj := range projects {
		switch classifyWorkspace(proj.Directory) {
		case workspaceRoot:
			if _, exists := rootMap[proj.Directory]; !exists {
				rootMap[proj.Directory] = &workspaceGroup{
					Root: s.createWorkspaceInfo(proj.Directory, proj.DirName, true, proj.HasWorkspace),
				}
			}
		case workspaceFork:
			rootPath := getRootWorkspacePath(proj.Directory)
			if rootPath == "" {
				standaloneGroups = append(standaloneGroups, workspaceGroup{
					Root: s.createWorkspaceInfo(proj.Directory, proj.DirName, false, proj.HasWorkspace),
				})
				continue
			}
			if existing, exists := rootMap[rootPath]; exists {
				existing.Forks = append(existing.Forks, s.createWorkspaceInfo(proj.Directory, proj.DirName, false, proj.HasWorkspace))
			} else {
				rootMap[rootPath] = &workspaceGroup{
					Root:  s.createWorkspaceInfo(rootPath, filepath.Base(rootPath), true, hassgaiDirectory(rootPath)),
					Forks: []workspaceInfo{s.createWorkspaceInfo(proj.Directory, proj.DirName, false, proj.HasWorkspace)},
				}
			}
		default:
			standaloneGroups = append(standaloneGroups, workspaceGroup{
				Root: s.createWorkspaceInfo(proj.Directory, proj.DirName, false, proj.HasWorkspace),
			})
		}
	}

	var groups []workspaceGroup
	for _, grp := range rootMap {
		groups = append(groups, *grp)
	}
	groups = append(groups, standaloneGroups...)

	slices.SortFunc(groups, func(a, b workspaceGroup) int {
		return strings.Compare(strings.ToLower(a.Root.DirName), strings.ToLower(b.Root.DirName))
	})

	return groups, nil
}

func (s *Server) resolveWorkspaceNameToPath(workspaceName string) string {
	if workspaceName == "" {
		return ""
	}

	groups, err := s.scanWorkspaceGroups()
	if err != nil {
		return ""
	}

	for _, grp := range groups {
		if grp.Root.DirName == workspaceName {
			return grp.Root.Directory
		}
		for _, fork := range grp.Forks {
			if fork.DirName == workspaceName {
				return fork.Directory
			}
		}
	}

	return ""
}

type modelStatusDisplay struct {
	ModelID string
	Status  string
}

func orderedModelStatuses(dir string, modelStatuses map[string]string) []modelStatusDisplay {
	if len(modelStatuses) == 0 {
		return nil
	}

	modelOrder := modelsForAgentFromGoal(dir, "project-critic-council")
	ordered := make([]modelStatusDisplay, 0, len(modelStatuses))
	used := make(map[string]bool)

	for _, modelSpec := range modelOrder {
		modelID := formatModelID("project-critic-council", modelSpec)
		status, ok := modelStatuses[modelID]
		if !ok {
			continue
		}
		ordered = append(ordered, modelStatusDisplay{ModelID: modelID, Status: status})
		used[modelID] = true
	}

	remaining := make([]string, 0, len(modelStatuses))
	for modelID := range modelStatuses {
		if used[modelID] {
			continue
		}
		remaining = append(remaining, modelID)
	}
	if len(remaining) == 0 {
		return ordered
	}
	if len(ordered) == 0 {
		ordered = make([]modelStatusDisplay, 0, len(modelStatuses))
	}
	if len(remaining) > 1 {
		slices.Sort(remaining)
	}
	for _, modelID := range remaining {
		ordered = append(ordered, modelStatusDisplay{ModelID: modelID, Status: modelStatuses[modelID]})
	}
	return ordered
}

func modelsForAgentFromGoal(dir, agent string) []string {
	goalPath := filepath.Join(dir, "GOAL.md")
	goalData, errRead := os.ReadFile(goalPath)
	if errRead != nil {
		return nil
	}
	metadata, errParse := parseYAMLFrontmatter(goalData)
	if errParse != nil {
		return nil
	}
	return getModelsForAgent(metadata.Models, agent)
}

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

func initializeWorkspace(workspacePath string) error {
	if err := unpackSkeleton(workspacePath); err != nil {
		return fmt.Errorf("unpacking skeleton: %w", err)
	}
	if err := initJJ(workspacePath); err != nil {
		return fmt.Errorf("initializing jj: %w", err)
	}
	if err := addGitExclude(workspacePath); err != nil {
		return fmt.Errorf("adding git exclude: %w", err)
	}
	if err := writeGoalExample(workspacePath); err != nil {
		return fmt.Errorf("writing GOAL.md: %w", err)
	}
	return nil
}

func unpackSkeleton(workspacePath string) error {
	subFS, err := fs.Sub(skelFS, "skel")
	if err != nil {
		return fmt.Errorf("accessing skeleton subdirectory: %w", err)
	}
	return fs.WalkDir(subFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		outPath := filepath.Join(workspacePath, path)
		if d.IsDir() {
			return os.MkdirAll(outPath, 0755)
		}
		data, errRead := fs.ReadFile(subFS, path)
		if errRead != nil {
			return errRead
		}
		return os.WriteFile(outPath, data, 0644)
	})
}

func initJJ(dir string) error {
	cmd := exec.Command("jj", "status")
	cmd.Dir = dir
	if err := cmd.Run(); err == nil {
		return nil
	} else if isExecNotFound(err) {
		return nil
	}
	initCmd := exec.Command("jj", "git", "init", "--colocate")
	initCmd.Dir = dir
	if errInit := initCmd.Run(); errInit != nil {
		return fmt.Errorf("running jj git init: %w", errInit)
	}
	return nil
}

func addGitExclude(dir string) error {
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return nil
	}
	gitInfoDir := filepath.Join(gitDir, "info")
	if err := os.MkdirAll(gitInfoDir, 0755); err != nil {
		return fmt.Errorf("creating .git/info directory: %w", err)
	}
	excludePath := filepath.Join(gitInfoDir, "exclude")
	existingContent, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading .git/info/exclude: %w", err)
	}
	if dotSGAILinePresent(existingContent) {
		return nil
	}
	existingContent = append(existingContent, []byte("/.sgai\n")...)
	if err := os.WriteFile(excludePath, existingContent, 0644); err != nil {
		return fmt.Errorf("writing .git/info/exclude: %w", err)
	}
	return nil
}

func writeGoalExample(dir string) error {
	goalPath := filepath.Join(dir, "GOAL.md")
	return os.WriteFile(goalPath, []byte(goalExampleContent), 0644)
}

func validateWorkspaceName(name string) string {
	if name == "" {
		return "workspace name is required"
	}
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "workspace name cannot contain path separators or '..'"
	}
	for _, ch := range name {
		isValid := (ch >= 'a' && ch <= 'z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-'
		if !isValid {
			return "workspace name can only contain lowercase letters, numbers, and dashes"
		}
	}
	return ""
}

func normalizeForkName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	normalized := strings.ReplaceAll(trimmed, "_", " ")
	parts := strings.Fields(normalized)
	joined := strings.Join(parts, "-")
	return strings.ToLower(joined)
}

func invokeLLMForAssist(prompt string) (string, error) {
	cmd := exec.Command("opencode", "run", "--title", "llm-assist")
	cmd.Stdin = strings.NewReader(prompt)
	output, errRun := cmd.Output()
	if errRun != nil {
		return "", fmt.Errorf("opencode failed: %w", errRun)
	}
	return strings.TrimSpace(string(output)), nil
}

func writeOpenCodeScript(content string) (string, error) {
	tmpFile, errTmp := os.CreateTemp("", "sgai-opencode-*.sh")
	if errTmp != nil {
		return "", errTmp
	}

	if _, errWrite := tmpFile.WriteString(content); errWrite != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", errWrite
	}

	if errClose := tmpFile.Close(); errClose != nil {
		_ = os.Remove(tmpFile.Name())
		return "", errClose
	}

	if errChmod := os.Chmod(tmpFile.Name(), 0755); errChmod != nil {
		_ = os.Remove(tmpFile.Name())
		return "", errChmod
	}

	return tmpFile.Name(), nil
}

// deleteRetrospectiveSession removes the retrospective session directory.
func deleteRetrospectiveSession(workspacePath, sessionID string) error {
	retrospectivesDir := filepath.Join(workspacePath, ".sgai", "retrospectives")
	sessionDir := filepath.Join(retrospectivesDir, sessionID)

	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return fmt.Errorf("session %s: %w", sessionID, os.ErrNotExist)
	}

	return os.RemoveAll(sessionDir)
}

type snippetData struct {
	Name        string
	FileName    string
	FullPath    string
	Description string
	Language    string
}

type languageCategory struct {
	Name     string
	Snippets []snippetData
}

func gatherSnippetsByLanguage(workspacePath string) []languageCategory {
	snippetsDir := filepath.Join(workspacePath, ".sgai", "snippets")
	snippetsFS := os.DirFS(snippetsDir)

	languages := make(map[string][]snippetData)

	err := fs.WalkDir(snippetsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		content, errRead := fs.ReadFile(snippetsFS, path)
		if errRead != nil {
			return nil
		}

		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			return nil
		}

		language := parts[0]
		filename := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))

		fm := parseFrontmatterMap(content)
		name := fm["name"]
		if name == "" {
			name = filename
		}
		description := fm["description"]

		languages[language] = append(languages[language], snippetData{
			Name:        name,
			FileName:    filename,
			FullPath:    language + "/" + filename,
			Description: description,
			Language:    language,
		})

		return nil
	})
	if err != nil {
		return nil
	}

	var result []languageCategory
	languageNames := slices.Sorted(maps.Keys(languages))

	for _, languageName := range languageNames {
		snippets := languages[languageName]
		slices.SortFunc(snippets, func(a, b snippetData) int {
			return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		})
		result = append(result, languageCategory{
			Name:     languageName,
			Snippets: snippets,
		})
	}

	return result
}
