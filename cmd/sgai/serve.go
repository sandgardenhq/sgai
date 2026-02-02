package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
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

var templates *template.Template

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

	initGraphTemplates()
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

type vscodeOpener struct{}

func (v *vscodeOpener) open(path string) error {
	return exec.Command("code", path).Run()
}

// Server handles HTTP requests for the sgai serve command.
type Server struct {
	mu              sync.Mutex
	sessions        map[string]*session
	everStartedDirs map[string]bool
	rootDir         string
	codeAvailable   bool
	editor          editorOpener
}

// NewServer creates a new Server instance with the given root directory.
// It converts rootDir to an absolute path to ensure consistent path comparisons
// between cookie values (set via validateDirectory) and template values
// (set via scanForProjects).
func NewServer(rootDir string) *Server {
	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		absRootDir = rootDir
	}
	_, errCode := exec.LookPath("code")
	return &Server{
		sessions:        make(map[string]*session),
		everStartedDirs: make(map[string]bool),
		rootDir:         absRootDir,
		codeAvailable:   errCode == nil,
		editor:          &vscodeOpener{},
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

	mcData := struct {
		AgentName          string
		Questions          []state.QuestionItem
		Directory          string
		DirName            string
		ReturnTo           string
		GoalContent        template.HTML
		ProjectMgmtContent template.HTML
	}{
		AgentName:          agentName,
		Questions:          wfState.MultiChoiceQuestion.Questions,
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

type workspaceHandler func(w http.ResponseWriter, r *http.Request, workspacePath string)

type workspaceRoute struct {
	action  string
	handler workspaceHandler
}

type workspacePrefixRoute struct {
	prefix  string
	handler func(w http.ResponseWriter, r *http.Request, workspacePath, subpath string)
}

func (s *Server) workspaceRoutes() []workspaceRoute {
	return []workspaceRoute{
		{"progress", s.tabHandler("progress")},
		{"spec", s.redirectHandler("progress")},
		{"log", s.tabHandler("log")},
		{"internals", s.tabHandler("internals")},
		{"changes", s.tabHandler("changes")},
		{"commits", s.tabHandler("commits")},
		{"messages", s.tabHandler("messages")},
		{"retro", s.retroGuard(s.tabHandler("retrospectives"))},
		{"init", s.handleWorkspaceInit},
		{"goal", s.handleWorkspaceGoal},
		{"start", s.handleWorkspaceStart},
		{"stop", s.handleWorkspaceStop},
		{"reset-state", s.handleWorkspaceResetState},
		{"fork", s.handleWorkspaceFork},
		{"update-description", s.handleWorkspaceUpdateDescription},
		{"retro/analyze", s.retroGuard(s.handleWorkspaceRetroAnalyze)},
		{"retro/apply", s.retroGuard(s.handleWorkspaceRetroApply)},
		{"retro/apply-select", s.retroGuard(s.handleWorkspaceRetroApplySelect)},
		{"retro/delete", s.retroGuard(s.handleWorkspaceRetroDelete)},
		{"open-vscode", s.handleWorkspaceOpenVSCode},
		{"skills", s.handleWorkspaceSkills},
		{"snippets", s.handleWorkspaceSnippets},
		{"agents", s.handleWorkspaceAgents},
	}
}

func (s *Server) workspacePrefixRoutes() []workspacePrefixRoute {
	return []workspacePrefixRoute{
		{"retro/", s.routeWorkspaceRetroPrefix},
		{"skills/", s.routeWorkspaceSkillPrefix},
		{"snippets/", s.routeWorkspaceSnippetPrefix},
	}
}

func (s *Server) tabHandler(tabName string) workspaceHandler {
	return func(w http.ResponseWriter, r *http.Request, workspacePath string) {
		s.handleWorkspaceTab(w, r, workspacePath, tabName)
	}
}

func (s *Server) redirectHandler(target string) workspaceHandler {
	return func(w http.ResponseWriter, r *http.Request, workspacePath string) {
		http.Redirect(w, r, workspaceURL(workspacePath, target), http.StatusSeeOther)
	}
}

func (s *Server) retroGuard(handler workspaceHandler) workspaceHandler {
	return func(w http.ResponseWriter, r *http.Request, workspacePath string) {
		if isRetrospectiveDisabled(workspacePath) {
			http.Redirect(w, r, workspaceURL(workspacePath, "progress"), http.StatusSeeOther)
			return
		}
		handler(w, r, workspacePath)
	}
}

func (s *Server) routeWorkspaceRetroPrefix(w http.ResponseWriter, r *http.Request, workspacePath, subpath string) {
	s.routeWorkspaceRetro(w, r, workspacePath, subpath)
}

func (s *Server) routeWorkspaceSkillPrefix(w http.ResponseWriter, r *http.Request, workspacePath, subpath string) {
	s.handleWorkspaceSkillDetail(w, r, workspacePath, subpath)
}

func (s *Server) routeWorkspaceSnippetPrefix(w http.ResponseWriter, r *http.Request, workspacePath, subpath string) {
	s.handleWorkspaceSnippetDetail(w, r, workspacePath, subpath)
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

	for _, route := range s.workspaceRoutes() {
		if action == route.action {
			route.handler(w, r, workspacePath)
			return
		}
	}

	for _, prefix := range s.workspacePrefixRoutes() {
		if after, ok := strings.CutPrefix(action, prefix.prefix); ok {
			prefix.handler(w, r, workspacePath, after)
			return
		}
	}

	http.NotFound(w, r)
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
	codeAvailable := s.codeAvailable && isLocalRequest(r)
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
		DisableRetrospective bool
		CurrentAgent         string
		Model                string
		FormattedModel       string
		CurrentModel         string
		ModelStatuses        map[string]string
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
		CodeAvailable:        codeAvailable,
		DisableRetrospective: disableRetrospective,
		CurrentAgent:         agentInfo.agent,
		Model:                agentInfo.model,
		FormattedModel:       agentInfo.formattedModel,
		CurrentModel:         wfState.CurrentModel,
		ModelStatuses:        wfState.ModelStatuses,
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

func renderTabToBuffer(buf *bytes.Buffer, templateName string, data any) {
	if err := templates.Lookup(templateName).Execute(buf, data); err != nil {
		log.Println("template execution failed:", err)
	}
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
		s.renderTreesSpecificationTab(&buf, r, dir)
	case "log":
		s.renderTreesLogTab(&buf, dir)
	case "progress":
		s.renderTreesEventsTab(&buf, r, dir)
	case "changes":
		s.renderTreesChangesTab(&buf, dir)
	case "commits":
		s.renderTreesCommitsTab(&buf, dir)
	case "messages":
		s.renderTreesMessagesTab(&buf, dir)
	case "retrospectives":
		s.renderTreesRetrospectivesTabToBuffer(&buf, r, dir, sessionParam)
	default:
		buf.WriteString("<p>Unknown tab</p>")
	}

	result.Content = template.HTML(buf.String())
	return result
}

func (s *Server) renderTreesSpecificationTab(buf *bytes.Buffer, r *http.Request, dir string) {
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

	renderTabToBuffer(buf, "trees_specification_content.html", struct {
		Directory          string
		DirName            string
		GoalContent        template.HTML
		ProjectMgmtContent template.HTML
		HasProjectMgmt     bool
		CodeAvailable      bool
	}{
		Directory:          dir,
		DirName:            filepath.Base(dir),
		GoalContent:        goalContent,
		ProjectMgmtContent: projectMgmtContent,
		HasProjectMgmt:     projectMgmtExists,
		CodeAvailable:      s.codeAvailable && isLocalRequest(r),
	})
}

func (s *Server) renderTreesLogTab(buf *bytes.Buffer, dir string) {
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

	renderTabToBuffer(buf, "trees_log_content.html", struct {
		Directory string
		DirName   string
		Logs      []logEntry
	}{
		Directory: dir,
		DirName:   filepath.Base(dir),
		Logs:      logs,
	})
}

func (s *Server) renderTreesEventsTab(buf *bytes.Buffer, r *http.Request, dir string) {
	wfState, _ := state.Load(statePath(dir))

	reversedProgress := slices.Clone(wfState.Progress)
	slices.Reverse(reversedProgress)

	currentAgent := wfState.CurrentAgent
	if currentAgent == "" {
		currentAgent = "Unknown"
	}

	var goalContent template.HTML
	if data, err := os.ReadFile(filepath.Join(dir, "GOAL.md")); err == nil {
		stripped := stripFrontmatter(string(data))
		if rendered, err := renderMarkdown([]byte(stripped)); err == nil {
			goalContent = template.HTML(rendered)
		}
	}

	renderTabToBuffer(buf, "trees_events_content.html", struct {
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
	}{
		Directory:            dir,
		DirName:              filepath.Base(dir),
		Progress:             formatProgressForDisplay(reversedProgress),
		SVGHash:              getWorkflowSVGHash(dir, currentAgent),
		CurrentAgent:         currentAgent,
		CurrentModel:         wfState.CurrentModel,
		ModelStatuses:        wfState.ModelStatuses,
		NeedsInput:           wfState.NeedsHumanInput(),
		RenderedHumanMessage: renderHumanMessage(wfState.HumanMessage),
		GoalContent:          goalContent,
		CodeAvailable:        s.codeAvailable && isLocalRequest(r),
	})
}

func (s *Server) renderTreesChangesTab(buf *bytes.Buffer, dir string) {
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

	renderTabToBuffer(buf, "trees_changes_content.html", struct {
		Directory   string
		DirName     string
		DiffOutput  template.HTML
		Description string
	}{
		Directory:   dir,
		DirName:     filepath.Base(dir),
		DiffOutput:  formatDiffHTML(diffOutput),
		Description: strings.TrimSpace(string(descOutput)),
	})
}

func (s *Server) renderTreesCommitsTab(buf *bytes.Buffer, dir string) {
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

func (s *Server) renderTreesMessagesTab(buf *bytes.Buffer, dir string) {
	wfState, _ := state.Load(statePath(dir))

	renderTabToBuffer(buf, "trees_messages_content.html", struct {
		Directory string
		DirName   string
		Messages  []messageDisplay
	}{
		Directory: dir,
		DirName:   filepath.Base(dir),
		Messages:  reverseMessages(wfState.Messages),
	})
}
