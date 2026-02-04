package main

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"maps"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed GOAL.example.md
var goalExampleContent string

var (
	templates         *template.Template
	tmplNoWorkflowSVG *template.Template
	tmplFallbackSVG   *template.Template
)

func init() {
	funcs := template.FuncMap{
		"add":            func(a, b int) int { return a + b },
		"shortModelName": extractModelShortName,
	}
	var err error
	templates, err = template.New("").Funcs(funcs).ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatalf("failed to parse templates: %v", err)
	}

	tmplNoWorkflowSVG = template.Must(template.New("noWorkflowSVG").Parse(
		`<svg xmlns="http://www.w3.org/2000/svg" width="200" height="40" viewBox="0 0 200 40">
<rect width="100%" height="100%" fill="#f8fafc"/>
<text x="100" y="24" font-family="system-ui, sans-serif" font-size="12" fill="#64748b" text-anchor="middle">No workflow available</text>
</svg>`))

	tmplFallbackSVG = template.Must(template.New("fallbackSVG").Parse(
		`<svg xmlns="http://www.w3.org/2000/svg" width="400" height="{{.Height}}" viewBox="0 0 400 {{.Height}}">
<rect width="100%" height="100%" fill="#f8fafc"/>
<text x="10" y="20" font-family="monospace" font-size="12" fill="#475569">{{range .Lines}}<tspan x="10" dy="{{.DY}}">{{.Text}}</tspan>{{end}}</text>
</svg>`))
}

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

func readGoalAndProjectMgmt(dir string) (goalContent, projectMgmtContent string) {
	if data, err := os.ReadFile(filepath.Join(dir, "GOAL.md")); err == nil {
		stripped := stripFrontmatter(string(data))
		if rendered, err := renderMarkdown([]byte(stripped)); err == nil {
			goalContent = rendered
		} else {
			goalContent = stripped
		}
	}

	if data, err := os.ReadFile(filepath.Join(dir, ".sgai", "PROJECT_MANAGEMENT.md")); err == nil {
		stripped := stripFrontmatter(string(data))
		if rendered, err := renderMarkdown([]byte(stripped)); err == nil {
			projectMgmtContent = rendered
		} else {
			projectMgmtContent = stripped
		}
	}

	return goalContent, projectMgmtContent
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
	"code":   {command: "code {path}", isTerminal: false},
	"cursor": {command: "cursor {path}", isTerminal: false},
	"zed":    {command: "zed {path}", isTerminal: false},
	"subl":   {command: "subl {path}", isTerminal: false},
	"idea":   {command: "idea {path}", isTerminal: false},
	"emacs":  {command: "emacsclient -n {path}", isTerminal: false},
	"nvim":   {command: "nvim {path}", isTerminal: true},
	"vim":    {command: "vim {path}", isTerminal: true},
	"atom":   {command: "atom {path}", isTerminal: false},
}

// configurableEditor implements editorOpener with configurable editor support.
// It supports preset editors and custom commands with {path} placeholders.
type configurableEditor struct {
	name       string
	command    string
	isTerminal bool
}

func (e *configurableEditor) open(path string) error {
	cmdLine := e.command
	if strings.Contains(cmdLine, "{path}") {
		cmdLine = strings.ReplaceAll(cmdLine, "{path}", path)
	} else {
		cmdLine = cmdLine + " " + path
	}

	parts := strings.Fields(cmdLine)
	if len(parts) == 0 {
		return fmt.Errorf("empty editor command")
	}

	return exec.Command(parts[0], parts[1:]...).Run()
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
	rootDir          string
	editorAvailable  bool
	isTerminalEditor bool
	editorName       string
	editor           editorOpener

	adhocModelsMu sync.Mutex
	cachedModels  []string
	adhocStates   map[string]*adhocPromptState
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
		adhocStates:      make(map[string]*adhocPromptState),
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

// isRetrospectiveDisabled returns true if retrospective functionality is disabled
// for the project at the given directory path via sgai.json configuration.
func isRetrospectiveDisabled(dir string) bool {
	config, err := loadProjectConfig(dir)
	if err != nil || config == nil {
		return false
	}
	return config.DisableRetrospective
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

	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.redirectToTrees)
	mux.HandleFunc("/respond", srv.pageRespond)
	mux.HandleFunc("/workflow.svg", srv.serveWorkflowSVG)
	mux.HandleFunc("/trees", srv.pageTrees)
	mux.HandleFunc("/trees/refresh", srv.pageTreesRefresh)
	mux.HandleFunc("GET /workspaces/new", srv.handleNewWorkspaceGet)
	mux.HandleFunc("POST /workspaces/new", srv.handleNewWorkspacePost)
	mux.HandleFunc("/workspaces/", srv.routeWorkspace)
	mux.HandleFunc("/compose", srv.routeCompose)
	mux.HandleFunc("/compose/", srv.routeCompose)

	log.Printf("sgai serve listening on http://%s", *listenAddr)
	if err := http.ListenAndServe(*listenAddr, mux); err != nil {
		log.Fatalln("server error:", err)
	}
}

func (s *Server) handleWorkspaceAgents(w http.ResponseWriter, _ *http.Request, workspacePath string) {
	type agentData struct {
		Name        string
		Description string
	}

	agentsDir := filepath.Join(workspacePath, ".sgai", "agent")
	agentsFS := os.DirFS(agentsDir)

	var agents []agentData
	err := fs.WalkDir(agentsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		name := strings.TrimSuffix(path, ".md")
		content, errRead := fs.ReadFile(agentsFS, path)
		if errRead != nil {
			return nil
		}
		desc := extractFrontmatterDescription(string(content))
		agents = append(agents, agentData{
			Name:        name,
			Description: desc,
		})
		return nil
	})
	if err != nil {
		agents = nil
	}

	slices.SortFunc(agents, func(a, b agentData) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	dirName := filepath.Base(workspacePath)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("agents.html"), struct {
		Agents  []agentData
		DirName string
	}{agents, dirName})
}

func (s *Server) serveWorkflowSVG(w http.ResponseWriter, r *http.Request) {
	dirParam := r.URL.Query().Get("dir")
	if dirParam == "" {
		http.Error(w, "No directory specified", http.StatusBadRequest)
		return
	}

	dir, err := s.validateDirectory(dirParam)
	if err != nil {
		http.Error(w, "Invalid directory", http.StatusForbidden)
		return
	}

	wfState, _ := state.Load(statePath(dir))

	svgContent := getWorkflowSVG(dir, wfState.CurrentAgent)
	w.Header().Set("Content-Type", "image/svg+xml")
	if svgContent == "" {
		if err := tmplNoWorkflowSVG.Execute(w, nil); err != nil {
			log.Println("template execution failed:", err)
		}
		return
	}

	if _, err := w.Write([]byte(svgContent)); err != nil {
		log.Println("write failed:", err)
	}
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

type sessionData struct {
	Directory            string
	DirName              string
	BadgeClass           string
	BadgeText            string
	NeedsInput           bool
	Running              bool
	InteractiveAuto      bool
	Status               string
	StatusText           string
	Message              string
	Task                 string
	CurrentAgent         string
	CurrentModel         string
	ModelStatuses        map[string]string
	HumanMessage         string
	RenderedHumanMessage template.HTML
	Progress             []state.ProgressEntry
	ProgressOpen         bool
	LatestProgress       string
	Messages             []messageDisplay
	Todos                []state.TodoItem
	ProjectTodos         []state.TodoItem
	GoalContent          template.HTML
	PMContent            template.HTML
	ProjectMgmtContent   template.HTML
	HasProjectMgmt       bool
	HasEditedGoal        bool
	CodeAvailable        bool
	EditorAvailable      bool
	IsTerminalEditor     bool
	EditorName           string
	ActiveTab            string
	SVGHash              string
	AgentSequence        []agentSequenceDisplay
	Cost                 state.SessionCost
}

func (s *Server) prepareSessionData(dir string, wfState state.Workflow, r *http.Request) sessionData {
	goalContent := ""
	hasEditedGoal := false
	if data, err := os.ReadFile(filepath.Join(dir, "GOAL.md")); err == nil {
		stripped := stripFrontmatter(string(data))
		if rendered, errRender := renderMarkdown([]byte(stripped)); errRender == nil {
			goalContent = rendered
		} else {
			goalContent = stripped
		}
		body := extractBody(data)
		hasEditedGoal = len(strings.TrimSpace(string(body))) > 0
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

	editorAvailable := s.editorAvailable && isLocalRequest(r)

	return sessionData{
		Directory:            dir,
		DirName:              filepath.Base(dir),
		BadgeClass:           badgeClass,
		BadgeText:            badgeText,
		NeedsInput:           needsInput,
		Running:              running,
		InteractiveAuto:      interactiveAuto,
		Status:               status,
		Message:              message,
		Task:                 task,
		CurrentAgent:         currentAgent,
		CurrentModel:         wfState.CurrentModel,
		ModelStatuses:        wfState.ModelStatuses,
		HumanMessage:         wfState.HumanMessage,
		RenderedHumanMessage: renderedHumanMessage,
		Progress:             reversedProgress,
		ProgressOpen:         progressOpen,
		LatestProgress:       getLatestProgress(wfState.Progress),
		Messages:             reversedMessages,
		Todos:                todos,
		ProjectTodos:         wfState.ProjectTodos,
		GoalContent:          template.HTML(goalContent),
		PMContent:            template.HTML(pmContent),
		ProjectMgmtContent:   template.HTML(pmContent),
		HasProjectMgmt:       projectMgmtExists,
		HasEditedGoal:        hasEditedGoal,
		CodeAvailable:        editorAvailable,
		EditorAvailable:      editorAvailable,
		IsTerminalEditor:     s.isTerminalEditor,
		EditorName:           s.editorName,
		ActiveTab:            "goal",
		SVGHash:              getWorkflowSVGHash(dir, currentAgent),
		AgentSequence:        prepareAgentSequenceDisplay(wfState.AgentSequence, running, getLastActivityTime(wfState.Progress)),
		Cost:                 wfState.Cost,
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

func (s *Server) pageRespond(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		s.pageRespondPost(w, r)
		return
	}

	dirParam := r.URL.Query().Get("dir")
	if dirParam == "" {
		http.Error(w, "No directory specified", http.StatusBadRequest)
		return
	}

	dir, err := s.validateDirectory(dirParam)
	if err != nil {
		http.Error(w, "Invalid directory", http.StatusForbidden)
		return
	}

	wfState, err := state.Load(statePath(dir))
	if err != nil {
		http.Error(w, "Failed to load state", http.StatusInternalServerError)
		return
	}

	if wfState.Status != state.StatusWaitingForHuman || wfState.HumanMessage == "" {
		http.Redirect(w, r, treesRedirectURL(dir), http.StatusSeeOther)
		return
	}

	agentName := wfState.CurrentAgent
	if agentName == "" {
		agentName = "Unknown"
	}

	returnTo := r.URL.Query().Get("returnTo")
	if returnTo == "" {
		returnTo = treesRedirectURL(dir)
	}

	goalContent, projectMgmtContent := readGoalAndProjectMgmt(dir)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	isModal := r.URL.Query().Get("modal") == "true"

	if wfState.MultiChoiceQuestion == nil || len(wfState.MultiChoiceQuestion.Questions) == 0 {
		http.Redirect(w, r, returnTo, http.StatusSeeOther)
		return
	}

	type renderedQuestion struct {
		Question    template.HTML
		Choices     []string
		MultiSelect bool
	}
	renderedQuestions := make([]renderedQuestion, len(wfState.MultiChoiceQuestion.Questions))
	for i, q := range wfState.MultiChoiceQuestion.Questions {
		renderedQuestions[i] = renderedQuestion{
			Question:    renderHumanMessage(q.Question),
			Choices:     q.Choices,
			MultiSelect: q.MultiSelect,
		}
	}

	mcData := struct {
		AgentName          string
		Questions          []renderedQuestion
		Directory          string
		DirName            string
		ReturnTo           string
		GoalContent        template.HTML
		ProjectMgmtContent template.HTML
	}{
		AgentName:          agentName,
		Questions:          renderedQuestions,
		Directory:          dir,
		DirName:            filepath.Base(dir),
		ReturnTo:           returnTo,
		GoalContent:        template.HTML(goalContent),
		ProjectMgmtContent: template.HTML(projectMgmtContent),
	}
	if isModal {
		executeTemplate(w, templates.Lookup("response_multichoice_modal.html"), mcData)
	} else {
		executeTemplate(w, templates.Lookup("response_multichoice.html"), mcData)
	}
}

func (s *Server) pageRespondPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	dir := r.FormValue("directory")
	returnTo := r.FormValue("returnTo")

	if dir == "" {
		http.Error(w, "Missing directory", http.StatusBadRequest)
		return
	}

	validDir, err := s.validateDirectory(dir)
	if err != nil {
		http.Error(w, "Invalid directory", http.StatusForbidden)
		return
	}

	response := buildMultichoiceResponse(r)

	if response == "" {
		http.Error(w, "Missing response", http.StatusBadRequest)
		return
	}

	responsePath := filepath.Join(validDir, ".sgai", "response.txt")
	if err := os.WriteFile(responsePath, []byte(response), 0644); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}

	wfState, err := state.Load(statePath(validDir))
	if err == nil {
		wfState.Status = state.StatusWorking
		wfState.HumanMessage = ""
		wfState.MultiChoiceQuestion = nil
		if err := state.Save(statePath(validDir), wfState); err != nil {
			log.Println("failed to save state:", err)
		}
	}

	if returnTo == "" {
		returnTo = treesRedirectURL(validDir)
	}
	http.Redirect(w, r, returnTo, http.StatusSeeOther)
}

func buildMultichoiceResponse(r *http.Request) string {
	questionCountStr := r.FormValue("questionCount")
	questionCount := 1
	if n, err := strconv.Atoi(questionCountStr); err == nil && n > 0 {
		questionCount = n
	}

	var allResponses []string
	for qIdx := 0; qIdx < questionCount; qIdx++ {
		choicesKey := fmt.Sprintf("choices_%d", qIdx)
		choices := r.Form[choicesKey]

		if len(choices) > 0 {
			if questionCount > 1 {
				allResponses = append(allResponses, fmt.Sprintf("Q%d: Selected: %s", qIdx+1, strings.Join(choices, ", ")))
			} else {
				allResponses = append(allResponses, "Selected: "+strings.Join(choices, ", "))
			}
		}
	}

	other := strings.TrimSpace(r.FormValue("other"))
	if other != "" {
		allResponses = append(allResponses, "Other: "+other)
	}

	return strings.Join(allResponses, "\n")
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

func executeTemplate(w http.ResponseWriter, tmpl *template.Template, data any) {
	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("template execution error: %v", err)
	}
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
	HasWorkspace bool
}

type workspaceGroup struct {
	Root  workspaceInfo
	Forks []workspaceInfo
}

func hasJJDirectory(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".jj"))
	return err == nil && info.IsDir()
}

func hassgaiDirectory(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".sgai"))
	return err == nil && info.IsDir()
}

func isRootWorkspace(dir string) bool {
	repoPath := filepath.Join(dir, ".jj", "repo")
	info, err := os.Stat(repoPath)
	return err == nil && info.IsDir()
}

func isForkWorkspace(dir string) bool {
	repoPath := filepath.Join(dir, ".jj", "repo")
	info, err := os.Stat(repoPath)
	return err == nil && !info.IsDir()
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

func runJJLogForRoot(dir string) []jjCommit {
	revset := `::@ | working_copies()`
	cmd := exec.Command("jj", "log", "-r", revset, "-T", jjLogTemplate)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil
	}
	return parseJJLogOutput(string(output))
}

func runJJLogForFork(dir string) []jjCommit {
	revset := `::@`
	cmd := exec.Command("jj", "log", "-r", revset, "-T", jjLogTemplate)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil
	}
	return parseJJLogOutput(string(output))
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

func renderJJLogHTML(commits []jjCommit, currentWorkspace string) string {
	if len(commits) == 0 {
		return `<article class="jj-empty"><p>No commits found</p></article>`
	}

	var buf strings.Builder
	buf.WriteString(`<article class="jj-log-article"><div class="jj-log">`)

	for i, commit := range commits {
		isLast := i == len(commits)-1
		buf.WriteString(renderCommitHTML(commit, isLast, currentWorkspace))
	}

	buf.WriteString(`</div></article>`)
	return buf.String()
}

func renderCommitHTML(commit jjCommit, _ bool, currentWorkspace string) string {
	var buf strings.Builder

	commitClass := "jj-commit"
	if commit.GraphChar == "@" || slices.Contains(commit.Workspaces, currentWorkspace) {
		commitClass += " current"
	}
	buf.WriteString(fmt.Sprintf(`<div class="%s">`, commitClass))

	buf.WriteString(`<div class="jj-graph-tree">`)
	buf.WriteString(`<pre class="jj-graph-pre">`)
	var graphContent strings.Builder
	if len(commit.GraphLines) > 0 {
		graphContent.WriteString(commit.GraphLines[0])
	}
	if len(commit.TrailingGraph) > 0 {
		for _, tg := range commit.TrailingGraph {
			graphContent.WriteString("\n")
			graphContent.WriteString(tg)
		}
	}
	buf.WriteString(template.HTMLEscapeString(graphContent.String()))
	buf.WriteString(`</pre>`)
	buf.WriteString(`</div>`)

	buf.WriteString(`<div class="jj-content">`)

	buf.WriteString(`<div class="jj-meta">`)
	buf.WriteString(`<code class="jj-change-id">`)
	buf.WriteString(template.HTMLEscapeString(commit.ChangeID))
	buf.WriteString(`</code>`)
	buf.WriteString(`<code class="jj-commit-id">`)
	buf.WriteString(template.HTMLEscapeString(commit.CommitID))
	buf.WriteString(`</code>`)

	if len(commit.Workspaces) > 0 {
		for _, ws := range commit.Workspaces {
			wsClass := "jj-workspace-badge"
			if ws == currentWorkspace {
				wsClass += " current"
				buf.WriteString(fmt.Sprintf(`<mark class="%s">%s</mark>`, wsClass, template.HTMLEscapeString(ws)))
			} else {
				buf.WriteString(fmt.Sprintf(`<a href="/trees?workspace=%s&tab=commits" class="%s" title="Navigate to %s fork">%s</a>`,
					url.QueryEscape(ws),
					wsClass,
					template.HTMLEscapeString(ws),
					template.HTMLEscapeString(ws)))
			}
		}
	}

	if len(commit.Bookmarks) > 0 {
		for _, bm := range commit.Bookmarks {
			buf.WriteString(fmt.Sprintf(`<kbd class="jj-bookmark-badge">%s</kbd>`, template.HTMLEscapeString(bm)))
		}
	}

	if commit.Timestamp != "" {
		buf.WriteString(fmt.Sprintf(`<small class="jj-timestamp">%s</small>`, template.HTMLEscapeString(commit.Timestamp)))
	}
	buf.WriteString(`</div>`)

	description := commit.Description
	if description == "" || description == "(no description)" {
		buf.WriteString(`<p class="jj-description empty">(no description)</p>`)
	} else {
		buf.WriteString(fmt.Sprintf(`<p class="jj-description">%s</p>`, template.HTMLEscapeString(description)))
	}

	buf.WriteString(`</div>`)
	buf.WriteString(`</div>`)

	return buf.String()
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
	inProgress := running || needsInput || s.wasEverStarted(dir)

	return workspaceInfo{
		Directory:    dir,
		DirName:      dirName,
		IsRoot:       isRoot,
		Running:      running,
		NeedsInput:   needsInput,
		InProgress:   inProgress,
		HasWorkspace: hasWorkspace,
	}
}

func (s *Server) wasEverStarted(dir string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.everStartedDirs[dir]
}

func (s *Server) scanWorkspaceGroups() ([]workspaceGroup, error) {
	projects, err := scanForProjects(s.rootDir)
	if err != nil {
		return nil, err
	}

	rootMap := make(map[string]*workspaceGroup)
	var standaloneGroups []workspaceGroup

	for _, proj := range projects {
		if !hasJJDirectory(proj.Directory) {
			standaloneGroups = append(standaloneGroups, workspaceGroup{
				Root: s.createWorkspaceInfo(proj.Directory, proj.DirName, false, proj.HasWorkspace),
			})
			continue
		}

		if isRootWorkspace(proj.Directory) {
			if _, exists := rootMap[proj.Directory]; !exists {
				rootMap[proj.Directory] = &workspaceGroup{
					Root: s.createWorkspaceInfo(proj.Directory, proj.DirName, true, proj.HasWorkspace),
				}
			}
			continue
		}

		if isForkWorkspace(proj.Directory) {
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
			continue
		}

		standaloneGroups = append(standaloneGroups, workspaceGroup{
			Root: s.createWorkspaceInfo(proj.Directory, proj.DirName, false, proj.HasWorkspace),
		})
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

func collectInProgressWorkspaces(groups []workspaceGroup) []workspaceInfo {
	var result []workspaceInfo
	for _, grp := range groups {
		if grp.Root.InProgress {
			result = append(result, grp.Root)
		}
		for _, fork := range grp.Forks {
			if fork.InProgress {
				result = append(result, fork)
			}
		}
	}
	return result
}

func hasAnyNeedsInput(workspaces []workspaceInfo) bool {
	for _, w := range workspaces {
		if w.NeedsInput {
			return true
		}
	}
	return false
}

func buildWorkspacePageData(groups []workspaceGroup, selectedPath, tabName, sessionParam string, content template.HTML) workspacePageData {
	inProgressWorkspaces := collectInProgressWorkspaces(groups)
	return workspacePageData{
		Groups:                 groups,
		SelectedDir:            baseDirName(selectedPath),
		JJLog:                  "",
		SelectedPath:           selectedPath,
		SelectedTab:            tabName,
		SelectedSession:        sessionParam,
		WorkspaceContent:       content,
		InProgressWorkspaces:   inProgressWorkspaces,
		HasNeedsInputWorkspace: hasAnyNeedsInput(inProgressWorkspaces),
	}
}

func (s *Server) redirectToTrees(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/trees", http.StatusSeeOther)
}

func treesRedirectURL(dir string) string {
	return "/trees?workspace=" + url.QueryEscape(filepath.Base(dir)) + "&tab=internals"
}

func buildReturnToURL(dir, tab, session string) string {
	returnTo := "/trees?workspace=" + url.QueryEscape(filepath.Base(dir)) + "&tab=" + url.QueryEscape(tab)
	if session != "" {
		returnTo += "&session=" + url.QueryEscape(session)
	}
	return returnTo
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

func (s *Server) pageTrees(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/trees" {
		http.NotFound(w, r)
		return
	}

	groups, err := s.scanWorkspaceGroups()
	if err != nil {
		http.Error(w, "Failed to scan workspaces", http.StatusInternalServerError)
		return
	}

	workspaceParam := r.URL.Query().Get("workspace")
	tabParam := r.URL.Query().Get("tab")
	sessionParam := r.URL.Query().Get("session")
	if tabParam == "" {
		tabParam = "internals"
	}

	var selectedPath string
	if workspaceParam != "" {
		selectedPath = s.resolveWorkspaceNameToPath(workspaceParam)
	}
	if selectedPath == "" {
		if cookie, err := r.Cookie("selected_workspace"); err == nil {
			selectedPath = cookie.Value
		}
	}

	if selectedPath != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "selected_workspace",
			Value:    selectedPath,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	var workspaceResult workspaceContentResult
	if selectedPath != "" {
		workspaceResult = s.renderWorkspaceContent(selectedPath, tabParam, sessionParam, r)
	}

	data := buildWorkspacePageData(groups, selectedPath, tabParam, sessionParam, workspaceResult.Content)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("trees.html"), data)
}

func (s *Server) pageTreesRefresh(w http.ResponseWriter, r *http.Request) {
	groups, err := s.scanWorkspaceGroups()
	if err != nil {
		http.Error(w, "Failed to scan workspaces", http.StatusInternalServerError)
		return
	}

	workspaceParam := r.URL.Query().Get("workspace")
	tabParam := r.URL.Query().Get("tab")
	sessionParam := r.URL.Query().Get("session")
	if tabParam == "" {
		tabParam = "internals"
	}

	var selectedPath string
	if workspaceParam != "" {
		selectedPath = s.resolveWorkspaceNameToPath(workspaceParam)
	}
	if selectedPath == "" {
		if cookie, err := r.Cookie("selected_workspace"); err == nil {
			selectedPath = cookie.Value
		}
	}

	var workspaceResult workspaceContentResult
	if selectedPath != "" {
		workspaceResult = s.renderWorkspaceContent(selectedPath, tabParam, sessionParam, r)
	}

	data := buildWorkspacePageData(groups, selectedPath, tabParam, sessionParam, workspaceResult.Content)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("trees_content.html"), data)
}

func buildTreesStatusText(wfState state.Workflow, _ string) string {
	return buildBaseStatusText(wfState)
}

func buildBaseStatusText(wfState state.Workflow) string {
	if wfState.Task != "" {
		return wfState.Task
	}
	if len(wfState.Progress) > 0 {
		return wfState.Progress[len(wfState.Progress)-1].Description
	}
	return ""
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

func (s *Server) routeWorkspace(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/workspaces/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	workspaceName := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	workspacePath := s.resolveWorkspaceNameToPath(workspaceName)
	if workspacePath == "" {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}

	switch action {
	case "progress":
		s.handleWorkspaceTab(w, r, workspacePath, "progress")
	case "spec":
		http.Redirect(w, r, workspaceURL(workspacePath, "progress"), http.StatusSeeOther)
		return
	case "log":
		s.handleWorkspaceTab(w, r, workspacePath, "log")
	case "internals":
		s.handleWorkspaceTab(w, r, workspacePath, "internals")
	case "changes":
		s.handleWorkspaceTab(w, r, workspacePath, "changes")
	case "commits":
		s.handleWorkspaceTab(w, r, workspacePath, "commits")
	case "messages":
		s.handleWorkspaceTab(w, r, workspacePath, "messages")
	case "run":
		s.handleWorkspaceTab(w, r, workspacePath, "run")
	case "retro":
		if isRetrospectiveDisabled(workspacePath) {
			http.Redirect(w, r, workspaceURL(workspacePath, "progress"), http.StatusSeeOther)
			return
		}
		s.handleWorkspaceTab(w, r, workspacePath, "retrospectives")
	case "init":
		s.handleWorkspaceInit(w, r, workspacePath)
	case "goal":
		s.handleWorkspaceGoal(w, r, workspacePath)
	case "start":
		s.handleWorkspaceStart(w, r, workspacePath)
	case "stop":
		s.handleWorkspaceStop(w, r, workspacePath)
	case "reset-state":
		s.handleWorkspaceResetState(w, r, workspacePath)
	case "fork":
		s.handleWorkspaceFork(w, r, workspacePath)
	case "update-description":
		s.handleWorkspaceUpdateDescription(w, r, workspacePath)
	case "retro/analyze":
		if isRetrospectiveDisabled(workspacePath) {
			http.Redirect(w, r, workspaceURL(workspacePath, "progress"), http.StatusSeeOther)
			return
		}
		s.handleWorkspaceRetroAnalyze(w, r, workspacePath)
	case "retro/apply":
		if isRetrospectiveDisabled(workspacePath) {
			http.Redirect(w, r, workspaceURL(workspacePath, "progress"), http.StatusSeeOther)
			return
		}
		s.handleWorkspaceRetroApply(w, r, workspacePath)
	case "retro/apply-select":
		if isRetrospectiveDisabled(workspacePath) {
			http.Redirect(w, r, workspaceURL(workspacePath, "progress"), http.StatusSeeOther)
			return
		}
		s.handleWorkspaceRetroApplySelect(w, r, workspacePath)
	case "retro/delete":
		if isRetrospectiveDisabled(workspacePath) {
			http.Redirect(w, r, workspaceURL(workspacePath, "progress"), http.StatusSeeOther)
			return
		}
		s.handleWorkspaceRetroDelete(w, r, workspacePath)
	case "open-vscode":
		s.handleWorkspaceOpenVSCode(w, r, workspacePath)
	case "skills":
		s.handleWorkspaceSkills(w, r, workspacePath)
	case "snippets":
		s.handleWorkspaceSnippets(w, r, workspacePath)
	case "agents":
		s.handleWorkspaceAgents(w, r, workspacePath)
	case "adhoc/submit":
		s.handleAdhocSubmit(w, r, workspacePath)
	case "adhoc/output":
		s.handleAdhocOutput(w, r, workspacePath)
	case "adhoc/save-state":
		s.handleAdhocSaveState(w, r, workspacePath)
	default:
		if after, ok := strings.CutPrefix(action, "retro/"); ok {
			s.routeWorkspaceRetro(w, r, workspacePath, after)
			return
		}
		if after, ok := strings.CutPrefix(action, "skills/"); ok {
			s.handleWorkspaceSkillDetail(w, r, workspacePath, after)
			return
		}
		if after, ok := strings.CutPrefix(action, "snippets/"); ok {
			s.handleWorkspaceSnippetDetail(w, r, workspacePath, after)
			return
		}
		http.NotFound(w, r)
	}
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

func workspaceURL(workspacePath, action string) string {
	return "/workspaces/" + url.PathEscape(filepath.Base(workspacePath)) + "/" + action
}

func baseDirName(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Base(path)
}

func (s *Server) handleWorkspaceTab(w http.ResponseWriter, r *http.Request, workspacePath, tabName string) {
	groups, err := s.scanWorkspaceGroups()
	if err != nil {
		http.Error(w, "Failed to scan workspaces", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "selected_workspace",
		Value:    workspacePath,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	sessionParam := r.URL.Query().Get("session")
	workspaceResult := s.renderWorkspaceContent(workspacePath, tabName, sessionParam, r)

	data := buildWorkspacePageData(groups, workspacePath, tabName, sessionParam, workspaceResult.Content)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("trees.html"), data)
}

func (s *Server) renderWorkspaceContent(dir, tabName, sessionParam string, r *http.Request) workspaceContentResult {
	if !hassgaiDirectory(dir) {
		return s.renderNoWorkspacePlaceholder(dir)
	}

	wfState, _ := state.Load(statePath(dir))
	statusText := buildTreesStatusText(wfState, dir)

	var running bool
	s.mu.Lock()
	sess := s.sessions[dir]
	s.mu.Unlock()
	if sess != nil {
		sess.mu.Lock()
		running = sess.running
		sess.mu.Unlock()
	}

	needsInput := wfState.NeedsHumanInput()
	isFork := hasJJDirectory(dir) && !isRootWorkspace(dir)
	returnToURL := buildReturnToURL(dir, tabName, sessionParam)

	totalExecTime := calculateTotalExecutionTime(wfState.AgentSequence, running, getLastActivityTime(wfState.Progress))
	editorAvailable := s.editorAvailable && isLocalRequest(r)
	disableRetrospective := isRetrospectiveDisabled(dir)

	agentInfo := resolveAgentModelInfo(wfState, dir)

	tabResult := s.renderTabContent(dir, tabName, sessionParam, r)

	if isStaleWorkingState(running, wfState) {
		bannerData := struct{ DirName string }{DirName: filepath.Base(dir)}
		var bannerBuf bytes.Buffer
		if err := templates.Lookup("trees_reset_banner.html").Execute(&bannerBuf, bannerData); err == nil {
			tabResult.Content = template.HTML(bannerBuf.String()) + tabResult.Content
		}
	}

	hasEditedGoal := false
	if data, err := os.ReadFile(filepath.Join(dir, "GOAL.md")); err == nil {
		body := extractBody(data)
		hasEditedGoal = len(strings.TrimSpace(string(body))) > 0
	}

	workspaceData := struct {
		Directory            string
		DirName              string
		StatusText           string
		ActiveTab            string
		Running              bool
		Session              string
		TabContent           template.HTML
		NeedsInput           bool
		IsFork               bool
		ReturnTo             string
		TotalExecutionTime   string
		CodeAvailable        bool
		EditorAvailable      bool
		IsTerminalEditor     bool
		EditorName           string
		DisableRetrospective bool
		CurrentAgent         string
		Model                string
		FormattedModel       string
		CurrentModel         string
		ModelStatuses        map[string]string
		HasEditedGoal        bool
	}{
		Directory:            dir,
		DirName:              filepath.Base(dir),
		StatusText:           statusText,
		ActiveTab:            tabName,
		Running:              running,
		Session:              sessionParam,
		TabContent:           tabResult.Content,
		NeedsInput:           needsInput,
		IsFork:               isFork,
		ReturnTo:             returnToURL,
		TotalExecutionTime:   totalExecTime,
		CodeAvailable:        editorAvailable,
		EditorAvailable:      editorAvailable,
		IsTerminalEditor:     s.isTerminalEditor,
		EditorName:           s.editorName,
		DisableRetrospective: disableRetrospective,
		CurrentAgent:         agentInfo.agent,
		Model:                agentInfo.model,
		FormattedModel:       agentInfo.formattedModel,
		CurrentModel:         wfState.CurrentModel,
		ModelStatuses:        wfState.ModelStatuses,
		HasEditedGoal:        hasEditedGoal,
	}

	var buf bytes.Buffer
	if err := templates.Lookup("trees_workspace.html").Execute(&buf, workspaceData); err != nil {
		log.Printf("Error rendering workspace template: %v", err)
		return workspaceContentResult{
			Content: template.HTML(fmt.Sprintf("<p>Error rendering workspace: %s</p>", err.Error())),
		}
	}
	return workspaceContentResult{
		Content: template.HTML(buf.String()),
	}
}

func (s *Server) renderNoWorkspacePlaceholder(dir string) workspaceContentResult {
	data := struct {
		Directory string
		DirName   string
	}{
		Directory: dir,
		DirName:   filepath.Base(dir),
	}
	var buf bytes.Buffer
	if err := templates.Lookup("trees_no_workspace.html").Execute(&buf, data); err != nil {
		log.Println("error rendering no-workspace template:", err)
		return workspaceContentResult{
			Content: template.HTML("<p>Error rendering placeholder</p>"),
		}
	}
	return workspaceContentResult{
		Content: template.HTML(buf.String()),
	}
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
	if factoraLinePresent(existingContent) {
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

func (s *Server) handleWorkspaceInit(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	if err := initializeWorkspace(workspacePath); err != nil {
		http.Error(w, "failed to initialize workspace", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, workspaceURL(workspacePath, "goal"), http.StatusSeeOther)
}

func (s *Server) handleNewWorkspaceGet(w http.ResponseWriter, _ *http.Request) {
	data := struct {
		RootDir      string
		ErrorMessage string
	}{
		RootDir: s.rootDir,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("new_workspace.html"), data)
}

func (s *Server) handleNewWorkspacePost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderNewWorkspaceWithError(w, "failed to parse form")
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if errMsg := validateWorkspaceName(name); errMsg != "" {
		s.renderNewWorkspaceWithError(w, errMsg)
		return
	}

	workspacePath := filepath.Join(s.rootDir, name)
	if _, errStat := os.Stat(workspacePath); errStat == nil {
		s.renderNewWorkspaceWithError(w, "a directory with this name already exists")
		return
	} else if !os.IsNotExist(errStat) {
		s.renderNewWorkspaceWithError(w, "failed to check workspace path")
		return
	}

	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		s.renderNewWorkspaceWithError(w, "failed to create workspace directory")
		return
	}

	if err := initializeWorkspace(workspacePath); err != nil {
		s.renderNewWorkspaceWithError(w, "failed to initialize workspace")
		return
	}

	http.Redirect(w, r, workspaceURL(workspacePath, "goal"), http.StatusSeeOther)
}

func (s *Server) renderNewWorkspaceWithError(w http.ResponseWriter, errMsg string) {
	data := struct {
		RootDir      string
		ErrorMessage string
	}{
		RootDir:      s.rootDir,
		ErrorMessage: errMsg,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("new_workspace.html"), data)
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
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '_'
		if !isValid {
			return "workspace name can only contain letters, numbers, dashes, and underscores"
		}
	}
	return ""
}

func (s *Server) renderTabContent(dir, tabName, sessionParam string, r *http.Request) tabContentResult {
	var buf bytes.Buffer
	var result tabContentResult

	switch tabName {
	case "internals":
		wfState, _ := state.Load(statePath(dir))
		data := s.prepareSessionData(dir, wfState, r)
		if err := templates.Lookup("trees_session_content.html").Execute(&buf, data); err != nil {
			result.Content = template.HTML("<p>Error rendering content</p>")
			return result
		}
	case "specification":
		s.renderTreesSpecificationTabToBuffer(&buf, r, dir)
	case "log":
		s.renderTreesLogTabToBuffer(&buf, dir)
	case "progress":
		s.renderTreesEventsTabToBuffer(&buf, r, dir)
	case "changes":
		s.renderTreesChangesTabToBuffer(&buf, dir)
	case "commits":
		s.renderTreesCommitsTabToBuffer(&buf, dir)
	case "messages":
		s.renderTreesMessagesTabToBuffer(&buf, dir)
	case "run":
		s.renderTreesRunTabToBuffer(&buf, dir)
	case "retrospectives":
		s.renderTreesRetrospectivesTabToBuffer(&buf, r, dir, sessionParam)
	default:
		buf.WriteString("<p>Unknown tab</p>")
	}

	result.Content = template.HTML(buf.String())
	return result
}

func (s *Server) renderTreesSpecificationTabToBuffer(buf *bytes.Buffer, r *http.Request, dir string) {
	var goalContent template.HTML
	if data, err := os.ReadFile(filepath.Join(dir, "GOAL.md")); err == nil {
		stripped := stripFrontmatter(string(data))
		if rendered, err := renderMarkdown([]byte(stripped)); err == nil {
			goalContent = template.HTML(rendered)
		}
	}

	var projectMgmtContent template.HTML
	projectMgmtExists := false
	if data, err := os.ReadFile(filepath.Join(dir, ".sgai", "PROJECT_MANAGEMENT.md")); err == nil {
		projectMgmtExists = true
		stripped := stripFrontmatter(string(data))
		if rendered, err := renderMarkdown([]byte(stripped)); err == nil {
			projectMgmtContent = template.HTML(rendered)
		}
	}

	editorAvailable := s.editorAvailable && isLocalRequest(r)

	data := struct {
		Directory          string
		DirName            string
		GoalContent        template.HTML
		ProjectMgmtContent template.HTML
		HasProjectMgmt     bool
		CodeAvailable      bool
		EditorAvailable    bool
		IsTerminalEditor   bool
		EditorName         string
	}{
		Directory:          dir,
		DirName:            filepath.Base(dir),
		GoalContent:        goalContent,
		ProjectMgmtContent: projectMgmtContent,
		HasProjectMgmt:     projectMgmtExists,
		CodeAvailable:      editorAvailable,
		EditorAvailable:    editorAvailable,
		IsTerminalEditor:   s.isTerminalEditor,
		EditorName:         s.editorName,
	}

	if err := templates.Lookup("trees_specification_content.html").Execute(buf, data); err != nil {
		log.Println("template execution failed:", err)
	}
}

func (s *Server) renderTreesLogTabToBuffer(buf *bytes.Buffer, dir string) {
	type logEntry struct {
		Prefix string
		Text   string
	}

	var logs []logEntry

	s.mu.Lock()
	sess := s.sessions[dir]
	s.mu.Unlock()

	if sess != nil && sess.outputLog != nil {
		lines := sess.outputLog.lines()
		for _, line := range lines {
			logs = append(logs, logEntry{Prefix: line.prefix, Text: line.text})
		}
	}

	data := struct {
		Directory string
		DirName   string
		Logs      []logEntry
	}{
		Directory: dir,
		DirName:   filepath.Base(dir),
		Logs:      logs,
	}

	if err := templates.Lookup("trees_log_content.html").Execute(buf, data); err != nil {
		log.Println("template execution failed:", err)
	}
}

func (s *Server) renderTreesEventsTabToBuffer(buf *bytes.Buffer, r *http.Request, dir string) {
	wfState, _ := state.Load(statePath(dir))

	reversedProgress := slices.Clone(wfState.Progress)
	slices.Reverse(reversedProgress)

	progressDisplay := formatProgressForDisplay(reversedProgress)

	currentAgent := wfState.CurrentAgent
	if currentAgent == "" {
		currentAgent = "Unknown"
	}

	needsInput := wfState.NeedsHumanInput()
	renderedHumanMessage := renderHumanMessage(wfState.HumanMessage)

	var goalContent template.HTML
	if data, err := os.ReadFile(filepath.Join(dir, "GOAL.md")); err == nil {
		stripped := stripFrontmatter(string(data))
		if rendered, err := renderMarkdown([]byte(stripped)); err == nil {
			goalContent = template.HTML(rendered)
		}
	}

	editorAvailable := s.editorAvailable && isLocalRequest(r)

	data := struct {
		Directory            string
		DirName              string
		Progress             []eventsProgressDisplay
		SVGHash              string
		CurrentAgent         string
		CurrentModel         string
		ModelStatuses        map[string]string
		NeedsInput           bool
		RenderedHumanMessage template.HTML
		GoalContent          template.HTML
		CodeAvailable        bool
		EditorAvailable      bool
		IsTerminalEditor     bool
		EditorName           string
	}{
		Directory:            dir,
		DirName:              filepath.Base(dir),
		Progress:             progressDisplay,
		SVGHash:              getWorkflowSVGHash(dir, currentAgent),
		CurrentAgent:         currentAgent,
		CurrentModel:         wfState.CurrentModel,
		ModelStatuses:        wfState.ModelStatuses,
		NeedsInput:           needsInput,
		RenderedHumanMessage: renderedHumanMessage,
		GoalContent:          goalContent,
		CodeAvailable:        editorAvailable,
		EditorAvailable:      editorAvailable,
		IsTerminalEditor:     s.isTerminalEditor,
		EditorName:           s.editorName,
	}

	if err := templates.Lookup("trees_events_content.html").Execute(buf, data); err != nil {
		log.Println("template execution failed:", err)
	}
}

func (s *Server) renderTreesChangesTabToBuffer(buf *bytes.Buffer, dir string) {
	diffCmd := exec.Command("jj", "diff", "--git")
	diffCmd.Dir = dir
	diffOutput, err := diffCmd.Output()
	if err != nil {
		buf.WriteString(`<p><em>Unable to get diff (not a jj repository or jj not available)</em></p>`)
		return
	}

	descCmd := exec.Command("jj", "log", "--no-graph", "-T", "description", "-r", "@")
	descCmd.Dir = dir
	descOutput, err := descCmd.Output()
	if err != nil {
		descOutput = []byte("")
	}

	data := struct {
		Directory   string
		DirName     string
		DiffOutput  template.HTML
		Description string
	}{
		Directory:   dir,
		DirName:     filepath.Base(dir),
		DiffOutput:  formatDiffHTML(diffOutput),
		Description: strings.TrimSpace(string(descOutput)),
	}

	if err := templates.Lookup("trees_changes_content.html").Execute(buf, data); err != nil {
		log.Println("template execution failed:", err)
	}
}

func (s *Server) renderTreesCommitsTabToBuffer(buf *bytes.Buffer, dir string) {
	if !hasJJDirectory(dir) {
		buf.WriteString("<p><em>Not a jj repository</em></p>")
		return
	}

	currentWorkspace := filepath.Base(dir)
	var commits []jjCommit
	if isRootWorkspace(dir) {
		commits = runJJLogForRoot(dir)
	} else {
		commits = runJJLogForFork(dir)
	}

	if len(commits) == 0 {
		buf.WriteString("<p><em>No log available</em></p>")
		return
	}

	buf.WriteString(renderJJLogHTML(commits, currentWorkspace))
}

func (s *Server) renderTreesMessagesTabToBuffer(buf *bytes.Buffer, dir string) {
	wfState, _ := state.Load(statePath(dir))
	reversedMessages := reverseMessages(wfState.Messages)

	data := struct {
		Directory string
		DirName   string
		Messages  []messageDisplay
	}{
		Directory: dir,
		DirName:   filepath.Base(dir),
		Messages:  reversedMessages,
	}

	if err := templates.Lookup("trees_messages_content.html").Execute(buf, data); err != nil {
		log.Println("template execution failed:", err)
	}
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

func (s *Server) handleWorkspaceGoal(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method == http.MethodPost {
		s.handleWorkspaceGoalPost(w, r, workspacePath)
		return
	}

	goalPath := filepath.Join(workspacePath, "GOAL.md")
	content, err := os.ReadFile(goalPath)
	if os.IsNotExist(err) {
		content = []byte(goalExampleContent)
		if err := os.WriteFile(goalPath, content, 0644); err != nil {
			http.Error(w, "Failed to create GOAL.md", http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		http.Error(w, "Failed to read GOAL.md", http.StatusInternalServerError)
		return
	}

	data := struct {
		Content   string
		Directory string
		DirName   string
	}{
		Content:   string(content),
		Directory: workspacePath,
		DirName:   filepath.Base(workspacePath),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("edit_goal.html"), data)
}

func (s *Server) handleWorkspaceGoalPost(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")
	if content == "" {
		http.Error(w, "Missing content", http.StatusBadRequest)
		return
	}

	goalPath := filepath.Join(workspacePath, "GOAL.md")
	if err := os.WriteFile(goalPath, []byte(content), 0644); err != nil {
		http.Error(w, "Failed to write GOAL.md", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, workspaceURL(workspacePath, "progress"), http.StatusSeeOther)
}

func (s *Server) handleWorkspaceStart(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	autoMode := r.FormValue("auto") == "true"

	result := s.startSession(workspacePath, autoMode)
	if result.alreadyRunning {
		http.Redirect(w, r, workspaceURL(workspacePath, "progress"), http.StatusSeeOther)
		return
	}
	if result.startError != nil {
		http.Error(w, result.startError.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, workspaceURL(workspacePath, "progress"), http.StatusSeeOther)
}

func (s *Server) handleWorkspaceStop(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	s.stopSession(workspacePath)

	http.Redirect(w, r, workspaceURL(workspacePath, "internals"), http.StatusSeeOther)
}

func isStaleWorkingState(running bool, wfState state.Workflow) bool {
	return !running && (wfState.Status == state.StatusWorking || wfState.Status == state.StatusAgentDone)
}

func (s *Server) handleWorkspaceResetState(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	if err := os.Remove(statePath(workspacePath)); err != nil && !os.IsNotExist(err) {
		http.Error(w, "Failed to reset state", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, workspaceURL(workspacePath, "progress"), http.StatusSeeOther)
}

// handleWorkspaceOpenVSCode opens the configured editor for a workspace or specific file.
// Security: Only allows localhost requests and a whitelist of specific files.
func (s *Server) handleWorkspaceOpenVSCode(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	if !s.editorAvailable {
		http.Error(w, "editor not available", http.StatusServiceUnavailable)
		return
	}

	if !isLocalRequest(r) {
		http.Error(w, "editor can only be opened from localhost", http.StatusForbidden)
		return
	}

	if s.isTerminalEditor {
		http.Error(w, "terminal editors cannot be opened from the web interface", http.StatusBadRequest)
		return
	}

	fileParam := r.URL.Query().Get("file")
	targetPath := workspacePath

	if fileParam != "" {
		allowedFiles := map[string]string{
			"GOAL.md":               filepath.Join(workspacePath, "GOAL.md"),
			"PROJECT_MANAGEMENT.md": filepath.Join(workspacePath, ".sgai", "PROJECT_MANAGEMENT.md"),
		}
		resolvedPath, allowed := allowedFiles[fileParam]
		if !allowed {
			http.Error(w, "Invalid file parameter", http.StatusBadRequest)
			return
		}
		targetPath = resolvedPath
	}

	if err := s.editor.open(targetPath); err != nil {
		http.Error(w, "failed to open editor", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, workspaceURL(workspacePath, "spec"), http.StatusSeeOther)
}

func (s *Server) handleWorkspaceFork(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	targetPath := filepath.Join(
		filepath.Dir(workspacePath),
		filepath.Base(workspacePath)+"-"+time.Now().Format("2006-01-02-150405"),
	)

	cmd := exec.Command("jj", "workspace", "add", targetPath)
	cmd.Dir = workspacePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fork workspace: %v: %s", err, output), http.StatusInternalServerError)
		return
	}

	if err := unpackSkeleton(targetPath); err != nil {
		log.Printf("failed to unpack skeleton for fork: %v", err)
	}
	if err := addGitExclude(targetPath); err != nil {
		log.Printf("failed to add git exclude for fork: %v", err)
	}
	if err := writeGoalExample(targetPath); err != nil {
		log.Printf("failed to create GOAL.md for fork: %v", err)
	}

	http.Redirect(w, r, workspaceURL(targetPath, "spec"), http.StatusSeeOther)
}

func (s *Server) handleWorkspaceUpdateDescription(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	description := r.FormValue("description")
	if description == "" {
		http.Error(w, "Missing description", http.StatusBadRequest)
		return
	}

	cmd := exec.Command("jj", "desc", "-m", description)
	cmd.Dir = workspacePath
	if output, err := cmd.CombinedOutput(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update description: %v: %s", err, output), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, workspaceURL(workspacePath, "changes"), http.StatusSeeOther)
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
			log.Printf("retrospective %s exited with error: %v", subcommand, err)
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
			log.Printf("retrospective analyze exited with error: %v", err)
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
			log.Printf("retrospective apply exited with error: %v", errApply)
			return
		}

		if err := deleteRetrospectiveSession(workspacePath, sessionID); err != nil {
			log.Printf("failed to auto-delete retrospective session %s: %v", sessionID, err)
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

type skillData struct {
	Name        string
	FullPath    string
	Description string
}

type categoryData struct {
	Name   string
	Skills []skillData
}

func (s *Server) handleWorkspaceSkills(w http.ResponseWriter, _ *http.Request, workspacePath string) {
	skillsDir := filepath.Join(workspacePath, ".sgai", "skills")
	skillsFS := os.DirFS(skillsDir)

	categories := make(map[string][]skillData)

	err := fs.WalkDir(skillsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "SKILL.md" {
			return nil
		}
		content, errRead := fs.ReadFile(skillsFS, path)
		if errRead != nil {
			return nil
		}
		skillPath := strings.TrimSuffix(path, "/SKILL.md")
		parts := strings.Split(skillPath, "/")
		var category string
		var name string
		if len(parts) > 1 {
			category = parts[0]
			name = strings.Join(parts[1:], "/")
		} else {
			category = ""
			name = skillPath
		}
		desc := extractFrontmatterDescription(string(content))
		categories[category] = append(categories[category], skillData{
			Name:        name,
			FullPath:    skillPath,
			Description: desc,
		})
		return nil
	})
	if err != nil {
		categories = nil
	}

	var categoryList []categoryData
	categoryNames := slices.Sorted(maps.Keys(categories))

	for _, categoryName := range categoryNames {
		skills := categories[categoryName]
		slices.SortFunc(skills, func(a, b skillData) int {
			return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		})
		displayName := categoryName
		if displayName == "" {
			displayName = "General"
		}
		categoryList = append(categoryList, categoryData{
			Name:   displayName,
			Skills: skills,
		})
	}

	dirName := filepath.Base(workspacePath)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("skills.html"), struct {
		Categories []categoryData
		DirName    string
	}{categoryList, dirName})
}

func (s *Server) handleWorkspaceSkillDetail(w http.ResponseWriter, r *http.Request, workspacePath, skillPath string) {
	if skillPath == "" {
		dirName := filepath.Base(workspacePath)
		http.Redirect(w, r, "/workspaces/"+dirName+"/skills", http.StatusSeeOther)
		return
	}

	skillsDir := filepath.Join(workspacePath, ".sgai", "skills")
	skillsFS := os.DirFS(skillsDir)

	skillFilePath := skillPath + "/SKILL.md"
	content, err := fs.ReadFile(skillsFS, skillFilePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	name := filepath.Base(skillPath)
	stripped := stripFrontmatter(string(content))
	rendered, errRender := renderMarkdown([]byte(stripped))
	if errRender != nil {
		rendered = stripped
	}

	dirName := filepath.Base(workspacePath)
	data := struct {
		Name       string
		FullPath   string
		Content    template.HTML
		RawContent string
		DirName    string
	}{
		Name:       name,
		FullPath:   skillPath,
		Content:    template.HTML(rendered),
		RawContent: stripped,
		DirName:    dirName,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("skill_detail.html"), data)
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

func (s *Server) handleWorkspaceSnippets(w http.ResponseWriter, _ *http.Request, workspacePath string) {
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
		languages = nil
	}

	var languageList []languageCategory
	languageNames := slices.Sorted(maps.Keys(languages))

	for _, languageName := range languageNames {
		snippets := languages[languageName]
		slices.SortFunc(snippets, func(a, b snippetData) int {
			return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		})
		languageList = append(languageList, languageCategory{
			Name:     languageName,
			Snippets: snippets,
		})
	}

	dirName := filepath.Base(workspacePath)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("snippets.html"), struct {
		Languages []languageCategory
		DirName   string
	}{languageList, dirName})
}

func (s *Server) handleWorkspaceSnippetDetail(w http.ResponseWriter, r *http.Request, workspacePath, snippetPath string) {
	if snippetPath == "" {
		dirName := filepath.Base(workspacePath)
		http.Redirect(w, r, "/workspaces/"+dirName+"/snippets", http.StatusSeeOther)
		return
	}

	parts := strings.Split(snippetPath, "/")
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	language := parts[0]
	filename := parts[1]

	snippetsDir := filepath.Join(workspacePath, ".sgai", "snippets")
	snippetsFS := os.DirFS(snippetsDir)

	var content []byte
	var foundExt string
	extensions := []string{".go", ".html", ".css", ".js", ".ts", ".py", ".sh", ".yaml", ".yml", ".json", ".md", ".sql", ".txt", ""}

	for _, ext := range extensions {
		filePath := language + "/" + filename + ext
		var errRead error
		content, errRead = fs.ReadFile(snippetsFS, filePath)
		if errRead == nil {
			foundExt = ext
			break
		}
	}

	if content == nil {
		http.NotFound(w, r)
		return
	}

	fm := parseFrontmatterMap(content)
	name := fm["name"]
	if name == "" {
		name = filename
	}
	description := fm["description"]
	whenToUse := fm["when_to_use"]
	codeContent := stripFrontmatter(string(content))

	dirName := filepath.Base(workspacePath)
	data := struct {
		Name        string
		FileName    string
		Language    string
		Description string
		WhenToUse   string
		Content     string
		Extension   string
		DirName     string
	}{
		Name:        name,
		FileName:    filename,
		Language:    language,
		Description: description,
		WhenToUse:   whenToUse,
		Content:     codeContent,
		Extension:   strings.TrimPrefix(foundExt, "."),
		DirName:     dirName,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("snippet_detail.html"), data)
}
