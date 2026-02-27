package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

type signalSubscriber struct {
	ch   chan struct{}
	done chan struct{}
}

type signalBroker struct {
	mu          sync.Mutex
	subscribers map[*signalSubscriber]struct{}
}

func newSignalBroker() *signalBroker {
	return &signalBroker{
		subscribers: make(map[*signalSubscriber]struct{}),
	}
}

func (b *signalBroker) subscribe() *signalSubscriber {
	s := &signalSubscriber{
		ch:   make(chan struct{}, 1),
		done: make(chan struct{}),
	}
	b.mu.Lock()
	b.subscribers[s] = struct{}{}
	b.mu.Unlock()
	return s
}

func (b *signalBroker) unsubscribe(s *signalSubscriber) {
	b.mu.Lock()
	delete(b.subscribers, s)
	b.mu.Unlock()
	close(s.done)
}

func (b *signalBroker) notify() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for s := range b.subscribers {
		select {
		case s.ch <- struct{}{}:
		default:
		}
	}
}

func (s *Server) registerAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/state", s.handleAPIState)
	mux.HandleFunc("GET /api/v1/signal", s.handleSignalStream)
	mux.HandleFunc("GET /api/v1/agents", s.handleAPIAgents)
	mux.HandleFunc("GET /api/v1/skills", s.handleAPISkills)
	mux.HandleFunc("GET /api/v1/skills/{name...}", s.handleAPISkillDetail)
	mux.HandleFunc("GET /api/v1/snippets", s.handleAPISnippets)
	mux.HandleFunc("GET /api/v1/snippets/{lang}", s.handleAPISnippetsByLanguage)
	mux.HandleFunc("GET /api/v1/snippets/{lang}/{fileName}", s.handleAPISnippetDetail)
	mux.HandleFunc("POST /api/v1/workspaces", s.handleAPICreateWorkspace)

	mux.HandleFunc("POST /api/v1/workspaces/{name}/respond", s.handleAPIRespond)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/start", s.handleAPIStartSession)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/stop", s.handleAPIStopSession)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/fork", s.handleAPIForkWorkspace)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/delete-fork", s.handleAPIDeleteFork)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/rename", s.handleAPIRenameWorkspace)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/goal", s.handleAPIGetGoal)
	mux.HandleFunc("PUT /api/v1/workspaces/{name}/goal", s.handleAPIUpdateGoal)
	mux.HandleFunc("PUT /api/v1/workspaces/{name}/summary", s.handleAPIUpdateSummary)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/adhoc", s.handleAPIAdhocStatus)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/adhoc", s.handleAPIAdhoc)
	mux.HandleFunc("DELETE /api/v1/workspaces/{name}/adhoc", s.handleAPIAdhocStop)

	mux.HandleFunc("GET /api/v1/workspaces/{name}/workflow.svg", s.handleAPIWorkflowSVG)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/steer", s.handleAPISteer)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/description", s.handleAPIUpdateDescription)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/pin", s.handleAPITogglePin)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/open-editor", s.handleAPIOpenEditor)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/open-opencode", s.handleAPIOpenInOpenCode)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/open-editor/goal", s.handleAPIOpenEditorGoal)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/open-editor/project-management", s.handleAPIOpenEditorProjectManagement)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/diff", s.handleAPIWorkspaceDiff)
	mux.HandleFunc("DELETE /api/v1/workspaces/{name}/messages/{id}", s.handleAPIDeleteMessage)
	mux.HandleFunc("GET /api/v1/models", s.handleAPIListModels)
	mux.HandleFunc("GET /api/v1/compose", s.handleAPIComposeState)
	mux.HandleFunc("POST /api/v1/compose", s.handleAPIComposeSave)
	mux.HandleFunc("GET /api/v1/compose/templates", s.handleAPIComposeTemplates)
	mux.HandleFunc("GET /api/v1/compose/preview", s.handleAPIComposePreview)
	mux.HandleFunc("POST /api/v1/compose/draft", s.handleAPIComposeDraft)
}

func (s *Server) handleSignalStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	sub := s.signals.subscribe()
	defer s.signals.unsubscribe(sub)

	for {
		select {
		case <-r.Context().Done():
			return
		case <-sub.done:
			return
		case <-sub.ch:
			if _, errWrite := fmt.Fprintf(w, "event: reload\ndata: {}\n\n"); errWrite != nil {
				return
			}
			flusher.Flush()
		}
	}
}

type apiFactoryState struct {
	Workspaces []apiWorkspaceFullState `json:"workspaces"`
}

type apiWorkspaceFullState struct {
	Name            string                      `json:"name"`
	Dir             string                      `json:"dir"`
	Running         bool                        `json:"running"`
	NeedsInput      bool                        `json:"needsInput"`
	InProgress      bool                        `json:"inProgress"`
	Pinned          bool                        `json:"pinned"`
	IsRoot          bool                        `json:"isRoot"`
	IsFork          bool                        `json:"isFork"`
	HasSGAI         bool                        `json:"hasSgai"`
	Status          string                      `json:"status"`
	BadgeClass      string                      `json:"badgeClass"`
	BadgeText       string                      `json:"badgeText"`
	HasEditedGoal   bool                        `json:"hasEditedGoal"`
	InteractiveAuto bool                        `json:"interactiveAuto"`
	ContinuousMode  bool                        `json:"continuousMode"`
	CurrentAgent    string                      `json:"currentAgent"`
	CurrentModel    string                      `json:"currentModel"`
	Task            string                      `json:"task"`
	GoalContent     string                      `json:"goalContent"`
	RawGoalContent  string                      `json:"rawGoalContent"`
	FullGoalContent string                      `json:"fullGoalContent"`
	PMContent       string                      `json:"pmContent"`
	HasProjectMgmt  bool                        `json:"hasProjectMgmt"`
	SVGHash         string                      `json:"svgHash"`
	TotalExecTime   string                      `json:"totalExecTime"`
	LatestProgress  string                      `json:"latestProgress"`
	HumanMessage    string                      `json:"humanMessage"`
	Summary         string                      `json:"summary,omitempty"`
	SummaryManual   bool                        `json:"summaryManual,omitempty"`
	AgentSequence   []apiAgentSequenceEntry     `json:"agentSequence"`
	Cost            state.SessionCost           `json:"cost"`
	ModelStatuses   []apiModelStatusEntry       `json:"modelStatuses,omitempty"`
	AgentModels     []apiAgentModelEntry        `json:"agentModels,omitempty"`
	Events          []apiEventEntry             `json:"events"`
	Messages        []apiMessageEntry           `json:"messages"`
	ProjectTodos    []apiTodoEntry              `json:"projectTodos"`
	AgentTodos      []apiTodoEntry              `json:"agentTodos"`
	Changes         apiChangesResponse          `json:"changes"`
	Commits         []apiCommitEntry            `json:"commits"`
	Forks           []apiForkEntry              `json:"forks,omitempty"`
	Log             []apiLogEntry               `json:"log"`
	PendingQuestion *apiPendingQuestionResponse `json:"pendingQuestion,omitempty"`
	Actions         []apiActionEntry            `json:"actions,omitempty"`
}

func (s *Server) handleAPIState(w http.ResponseWriter, _ *http.Request) {
	if cached, ok := s.stateCache.get("state"); ok {
		writeJSON(w, cached)
		return
	}
	factoryState, _ := s.stateFlight.do("state", func() (apiFactoryState, error) {
		if cached, ok := s.stateCache.get("state"); ok {
			return cached, nil
		}
		s.mu.Lock()
		genBefore := s.stateGeneration
		s.mu.Unlock()
		result := s.buildFullFactoryState()
		s.mu.Lock()
		genAfter := s.stateGeneration
		s.mu.Unlock()
		if genBefore == genAfter {
			s.stateCache.set("state", result)
		}
		return result, nil
	})
	writeJSON(w, factoryState)
}

func (s *Server) warmStateCache() {
	s.mu.Lock()
	genBefore := s.stateGeneration
	s.mu.Unlock()
	result := s.buildFullFactoryState()
	s.mu.Lock()
	genAfter := s.stateGeneration
	s.mu.Unlock()
	if genBefore == genAfter {
		s.stateCache.set("state", result)
	}
}

func (s *Server) loadWorkspaceState(dir string) state.Workflow {
	stPath := statePath(dir)
	info, errStat := os.Stat(stPath)
	if errStat != nil {
		return state.Workflow{}
	}
	if info.Size() > maxStateSizeBytes {
		return state.Workflow{}
	}
	return s.workspaceCoordinator(dir).State()
}

func (s *Server) buildFullFactoryState() apiFactoryState {
	groups, errScan := s.scanWorkspaceGroups()
	if errScan != nil {
		return apiFactoryState{}
	}

	var allWorkspaces []workspaceInfo
	for _, grp := range groups {
		allWorkspaces = append(allWorkspaces, grp.Root)
		allWorkspaces = append(allWorkspaces, grp.Forks...)
	}

	workspaces := make([]apiWorkspaceFullState, len(allWorkspaces))
	var wg sync.WaitGroup
	for i, ws := range allWorkspaces {
		wg.Go(func() {
			workspaces[i] = s.buildWorkspaceFullState(ws, groups)
		})
	}
	wg.Wait()

	return apiFactoryState{Workspaces: workspaces}
}

const maxStateSizeBytes = 10 * 1024 * 1024

func (s *Server) buildWorkspaceFullState(ws workspaceInfo, groups []workspaceGroup) apiWorkspaceFullState {
	wfState := s.loadWorkspaceState(ws.Directory)
	kind := s.classifyWorkspaceCached(ws.Directory)

	interactiveAuto := wfState.InteractionMode == state.ModeSelfDrive || wfState.InteractionMode == state.ModeContinuous
	badgeClass, badgeText := badgeStatus(wfState, ws.Running)
	needsInput := wfState.NeedsHumanInput()

	currentAgent := wfState.CurrentAgent
	if currentAgent == "" {
		currentAgent = "Unknown"
	}

	status := wfState.Status
	if status == "" {
		status = "-"
	}

	goalContent, rawGoalContent, fullGoalContent, pmContent, hasProjectMgmt := readGoalAndPMForAPI(ws.Directory)

	hasEditedGoal := false
	if data, errRead := os.ReadFile(filepath.Join(ws.Directory, "GOAL.md")); errRead == nil {
		body := extractBody(data)
		hasEditedGoal = len(strings.TrimSpace(string(body))) > 0
	}

	agentSeq := convertAgentSequence(
		prepareAgentSequenceDisplay(wfState.AgentSequence, ws.Running, getLastActivityTime(wfState.Progress), ws.Directory),
	)
	totalExecTime := calculateTotalExecutionTime(wfState.AgentSequence, ws.Running, getLastActivityTime(wfState.Progress))

	modelStatuses := convertModelStatuses(orderedModelStatuses(ws.Directory, wfState.ModelStatuses))

	reversedProgress := slices.Clone(wfState.Progress)
	slices.Reverse(reversedProgress)
	events := convertEventsForAPI(formatProgressForDisplay(reversedProgress))

	messages := convertMessagesForAPI(wfState.Messages)

	var logLines []apiLogEntry
	s.mu.Lock()
	sess := s.sessions[ws.Directory]
	s.mu.Unlock()
	if sess != nil && sess.outputLog != nil {
		for _, line := range sess.outputLog.lines() {
			logLines = append(logLines, apiLogEntry{Prefix: line.prefix, Text: line.text})
		}
	}

	var changesResult jjChangesResult
	if hasJJRepo(ws.Directory) {
		changesResult = s.collectJJChangesCached(ws.Directory)
	}
	changes := apiChangesResponse{
		Description: changesResult.description,
		DiffLines:   changesResult.diffLines,
	}

	var commits []apiCommitEntry
	if hasJJRepo(ws.Directory) {
		commits = buildCommitEntries(s.filteredCommitsForWorkspace(ws.Directory))
	}

	var pendingQuestion *apiPendingQuestionResponse
	if wfState.NeedsHumanInput() {
		agentName := currentAgent
		var questions []apiQuestionItem
		if wfState.MultiChoiceQuestion != nil {
			questions = make([]apiQuestionItem, 0, len(wfState.MultiChoiceQuestion.Questions))
			for _, q := range wfState.MultiChoiceQuestion.Questions {
				questions = append(questions, apiQuestionItem{
					Question:    q.Question,
					Choices:     q.Choices,
					MultiSelect: q.MultiSelect,
				})
			}
		}
		pendingQuestion = &apiPendingQuestionResponse{
			QuestionID: generateQuestionID(wfState),
			Type:       questionType(wfState),
			AgentName:  agentName,
			Message:    wfState.HumanMessage,
			Questions:  questions,
		}
	}

	full := apiWorkspaceFullState{
		Name:            ws.DirName,
		Dir:             ws.Directory,
		Running:         ws.Running,
		NeedsInput:      needsInput,
		InProgress:      ws.InProgress,
		Pinned:          ws.Pinned,
		IsRoot:          kind == workspaceRoot,
		IsFork:          kind == workspaceFork,
		HasSGAI:         ws.HasWorkspace,
		Status:          status,
		BadgeClass:      badgeClass,
		BadgeText:       badgeText,
		HasEditedGoal:   hasEditedGoal,
		InteractiveAuto: interactiveAuto,
		ContinuousMode:  readContinuousModePrompt(ws.Directory) != "",
		CurrentAgent:    currentAgent,
		CurrentModel:    resolveCurrentModel(ws.Directory, wfState),
		Task:            wfState.Task,
		GoalContent:     goalContent,
		RawGoalContent:  rawGoalContent,
		FullGoalContent: fullGoalContent,
		PMContent:       pmContent,
		HasProjectMgmt:  hasProjectMgmt,
		SVGHash:         s.getWorkflowSVGHashCached(ws.Directory, currentAgent),
		TotalExecTime:   totalExecTime,
		LatestProgress:  getLatestProgress(wfState.Progress),
		HumanMessage:    wfState.HumanMessage,
		Summary:         wfState.Summary,
		SummaryManual:   wfState.SummaryManual,
		AgentSequence:   agentSeq,
		Cost:            wfState.Cost,
		ModelStatuses:   modelStatuses,
		AgentModels:     collectAgentModels(ws.Directory),
		Events:          events,
		Messages:        messages,
		ProjectTodos:    convertTodosForAPI(wfState.ProjectTodos),
		AgentTodos:      convertTodosForAPI(wfState.Todos),
		Changes:         changes,
		Commits:         commits,
		Log:             logLines,
		PendingQuestion: pendingQuestion,
		Actions:         loadActionsForAPI(ws.Directory),
	}

	if kind == workspaceRoot {
		full.Forks = s.collectForksForAPIFromGroups(ws.Directory, groups)
	}

	return full
}

func buildCommitEntries(commits []jjCommit) []apiCommitEntry {
	entries := make([]apiCommitEntry, 0, len(commits))
	for _, c := range commits {
		entries = append(entries, apiCommitEntry{
			ChangeID:    c.ChangeID,
			CommitID:    c.CommitID,
			Workspaces:  c.Workspaces,
			Timestamp:   c.Timestamp,
			Bookmarks:   c.Bookmarks,
			Description: c.Description,
			GraphChar:   c.GraphChar,
		})
	}
	return entries
}

func (s *Server) collectForksForAPIFromGroups(rootDir string, groups []workspaceGroup) []apiForkEntry {
	for _, grp := range groups {
		if grp.Root.Directory != rootDir {
			continue
		}
		bookmark := s.resolveBaseBookmarkCached(rootDir)
		forks := make([]apiForkEntry, len(grp.Forks))
		var wg sync.WaitGroup
		for i, fork := range grp.Forks {
			wg.Go(func() {
				commits := convertJJCommitsForAPI(s.runJJLogForForkCached(bookmark, fork.Directory))
				wfState := s.loadWorkspaceState(fork.Directory)
				forks[i] = apiForkEntry{
					Name:        fork.DirName,
					Dir:         fork.Directory,
					Running:     fork.Running,
					NeedsInput:  wfState.NeedsHumanInput(),
					InProgress:  fork.InProgress,
					Pinned:      fork.Pinned,
					CommitAhead: s.countForkCommitsAheadCached(bookmark, fork.Directory),
					Commits:     commits,
					Summary:     wfState.Summary,
				}
			})
		}
		wg.Wait()
		return forks
	}
	return nil
}

func (s *Server) spaMiddleware(mux *http.ServeMux) http.Handler {
	webappFS, errSub := fs.Sub(webappDist, "webapp/dist")
	if errSub != nil {
		log.Println("failed to create webapp sub-filesystem:", errSub)
	}

	var staticHandler http.Handler
	if webappFS != nil {
		staticHandler = http.FileServerFS(webappFS)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAPIRoute(r.URL.Path) {
			mux.ServeHTTP(w, r)
			return
		}

		if webappFS == nil {
			http.Error(w, "react app not available", http.StatusInternalServerError)
			return
		}

		if isStaticAsset(r.URL.Path) {
			staticHandler.ServeHTTP(w, r)
			return
		}

		serveReactIndex(w, webappFS)
	})
}

func isAPIRoute(urlPath string) bool {
	return strings.HasPrefix(urlPath, "/api/")
}

func isStaticAsset(urlPath string) bool {
	ext := path.Ext(urlPath)
	switch ext {
	case ".js", ".css", ".map", ".png", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".json":
		return true
	}
	return false
}

func serveReactIndex(w http.ResponseWriter, webappFS fs.FS) {
	indexHTML, errRead := fs.ReadFile(webappFS, "index.html")
	if errRead != nil {
		http.Error(w, "react app not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, errWrite := w.Write(indexHTML); errWrite != nil {
		log.Println("failed to serve react index:", errWrite)
	}
}

func (s *Server) resolveAPIWorkspace(r *http.Request) string {
	if name := r.URL.Query().Get("workspace"); name != "" {
		return s.resolveWorkspaceNameToPath(name)
	}
	groups, errScan := s.scanWorkspaceGroups()
	if errScan != nil || len(groups) == 0 {
		return ""
	}
	return groups[0].Root.Directory
}

type apiAgentEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type apiAgentsResponse struct {
	Agents []apiAgentEntry `json:"agents"`
}

func (s *Server) handleAPIAgents(w http.ResponseWriter, r *http.Request) {
	workspacePath := s.resolveAPIWorkspace(r)
	if workspacePath == "" {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	agents := collectAgents(workspacePath)
	writeJSON(w, apiAgentsResponse{Agents: agents})
}

func collectAgents(workspacePath string) []apiAgentEntry {
	agentsDir := filepath.Join(workspacePath, ".sgai", "agent")
	agentsFS := os.DirFS(agentsDir)

	var agents []apiAgentEntry
	errWalk := fs.WalkDir(agentsFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		name := strings.TrimSuffix(p, ".md")
		content, errRead := fs.ReadFile(agentsFS, p)
		if errRead != nil {
			return nil
		}
		desc := extractFrontmatterDescription(string(content))
		agents = append(agents, apiAgentEntry{
			Name:        name,
			Description: desc,
		})
		return nil
	})
	if errWalk != nil {
		return nil
	}

	slices.SortFunc(agents, func(a, b apiAgentEntry) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	return agents
}

type apiSkillEntry struct {
	Name        string `json:"name"`
	FullPath    string `json:"fullPath"`
	Description string `json:"description"`
}

type apiSkillCategory struct {
	Name   string          `json:"name"`
	Skills []apiSkillEntry `json:"skills"`
}

type apiSkillsResponse struct {
	Categories []apiSkillCategory `json:"categories"`
}

func (s *Server) handleAPISkills(w http.ResponseWriter, r *http.Request) {
	workspacePath := s.resolveAPIWorkspace(r)
	if workspacePath == "" {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	categories := collectSkillCategories(workspacePath)
	writeJSON(w, apiSkillsResponse{Categories: categories})
}

func collectSkillCategories(workspacePath string) []apiSkillCategory {
	skillsDir := filepath.Join(workspacePath, ".sgai", "skills")
	skillsFS := os.DirFS(skillsDir)

	grouped := make(map[string][]apiSkillEntry)

	errWalk := fs.WalkDir(skillsFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "SKILL.md" {
			return nil
		}
		content, errRead := fs.ReadFile(skillsFS, p)
		if errRead != nil {
			return nil
		}
		skillPath := strings.TrimSuffix(p, "/SKILL.md")
		parts := strings.Split(skillPath, "/")
		var category, name string
		if len(parts) > 1 {
			category = parts[0]
			name = strings.Join(parts[1:], "/")
		} else {
			name = skillPath
		}
		desc := extractFrontmatterDescription(string(content))
		grouped[category] = append(grouped[category], apiSkillEntry{
			Name:        name,
			FullPath:    skillPath,
			Description: desc,
		})
		return nil
	})
	if errWalk != nil {
		return nil
	}

	var categories []apiSkillCategory
	for _, categoryName := range slices.Sorted(maps.Keys(grouped)) {
		skills := grouped[categoryName]
		slices.SortFunc(skills, func(a, b apiSkillEntry) int {
			return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		})
		displayName := categoryName
		if displayName == "" {
			displayName = "General"
		}
		categories = append(categories, apiSkillCategory{
			Name:   displayName,
			Skills: skills,
		})
	}

	return categories
}

type apiSkillDetailResponse struct {
	Name       string `json:"name"`
	FullPath   string `json:"fullPath"`
	Content    string `json:"content"`
	RawContent string `json:"rawContent"`
}

func (s *Server) handleAPISkillDetail(w http.ResponseWriter, r *http.Request) {
	workspacePath := s.resolveAPIWorkspace(r)
	if workspacePath == "" {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	skillName := r.PathValue("name")
	if skillName == "" {
		http.Error(w, "skill name is required", http.StatusBadRequest)
		return
	}

	skillsDir := filepath.Join(workspacePath, ".sgai", "skills")
	skillsFS := os.DirFS(skillsDir)

	skillFilePath := skillName + "/SKILL.md"
	content, errRead := fs.ReadFile(skillsFS, skillFilePath)
	if errRead != nil {
		http.Error(w, "skill not found", http.StatusNotFound)
		return
	}

	stripped := stripFrontmatter(string(content))
	rendered, errRender := renderMarkdown([]byte(stripped))
	if errRender != nil {
		rendered = stripped
	}

	writeJSON(w, apiSkillDetailResponse{
		Name:       path.Base(skillName),
		FullPath:   skillName,
		Content:    rendered,
		RawContent: stripped,
	})
}

type apiSnippetEntry struct {
	Name        string `json:"name"`
	FileName    string `json:"fileName"`
	FullPath    string `json:"fullPath"`
	Description string `json:"description"`
	Language    string `json:"language"`
}

type apiLanguageCategory struct {
	Name     string            `json:"name"`
	Snippets []apiSnippetEntry `json:"snippets"`
}

type apiSnippetsResponse struct {
	Languages []apiLanguageCategory `json:"languages"`
}

func (s *Server) handleAPISnippets(w http.ResponseWriter, r *http.Request) {
	workspacePath := s.resolveAPIWorkspace(r)
	if workspacePath == "" {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	languages := convertSnippetLanguages(gatherSnippetsByLanguage(workspacePath))
	writeJSON(w, apiSnippetsResponse{Languages: languages})
}

func convertSnippetLanguages(categories []languageCategory) []apiLanguageCategory {
	result := make([]apiLanguageCategory, 0, len(categories))
	for _, lc := range categories {
		snippets := make([]apiSnippetEntry, 0, len(lc.Snippets))
		for _, s := range lc.Snippets {
			snippets = append(snippets, apiSnippetEntry(s))
		}
		result = append(result, apiLanguageCategory{
			Name:     lc.Name,
			Snippets: snippets,
		})
	}
	return result
}

type apiSnippetsByLanguageResponse struct {
	Language string            `json:"language"`
	Snippets []apiSnippetEntry `json:"snippets"`
}

func (s *Server) handleAPISnippetsByLanguage(w http.ResponseWriter, r *http.Request) {
	workspacePath := s.resolveAPIWorkspace(r)
	if workspacePath == "" {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	lang := r.PathValue("lang")
	if lang == "" {
		http.Error(w, "language is required", http.StatusBadRequest)
		return
	}

	allLanguages := convertSnippetLanguages(gatherSnippetsByLanguage(workspacePath))
	for _, lc := range allLanguages {
		if lc.Name == lang {
			writeJSON(w, apiSnippetsByLanguageResponse{
				Language: lc.Name,
				Snippets: lc.Snippets,
			})
			return
		}
	}

	http.Error(w, "language not found", http.StatusNotFound)
}

type apiSnippetDetailResponse struct {
	Name        string `json:"name"`
	FileName    string `json:"fileName"`
	Language    string `json:"language"`
	Description string `json:"description"`
	WhenToUse   string `json:"whenToUse"`
	Content     string `json:"content"`
}

func (s *Server) handleAPISnippetDetail(w http.ResponseWriter, r *http.Request) {
	workspacePath := s.resolveAPIWorkspace(r)
	if workspacePath == "" {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	lang := r.PathValue("lang")
	fileName := r.PathValue("fileName")
	if lang == "" || fileName == "" {
		http.Error(w, "language and fileName are required", http.StatusBadRequest)
		return
	}

	snippetsDir := filepath.Join(workspacePath, ".sgai", "snippets")
	snippetsFS := os.DirFS(snippetsDir)

	var content []byte
	extensions := []string{".go", ".html", ".css", ".js", ".ts", ".py", ".sh", ".yaml", ".yml", ".json", ".md", ".sql", ".txt", ""}
	for _, ext := range extensions {
		filePath := lang + "/" + fileName + ext
		var errRead error
		content, errRead = fs.ReadFile(snippetsFS, filePath)
		if errRead == nil {
			break
		}
	}

	if content == nil {
		http.Error(w, "snippet not found", http.StatusNotFound)
		return
	}

	fm := parseFrontmatterMap(content)
	name := fm["name"]
	if name == "" {
		name = fileName
	}

	writeJSON(w, apiSnippetDetailResponse{
		Name:        name,
		FileName:    fileName,
		Language:    lang,
		Description: fm["description"],
		WhenToUse:   fm["when_to_use"],
		Content:     stripFrontmatter(string(content)),
	})
}

type apiActionEntry struct {
	Name        string `json:"name"`
	Model       string `json:"model"`
	Prompt      string `json:"prompt"`
	Description string `json:"description,omitempty"`
}

type apiAgentSequenceEntry struct {
	Agent       string `json:"agent"`
	Model       string `json:"model"`
	ElapsedTime string `json:"elapsedTime"`
	IsCurrent   bool   `json:"isCurrent"`
}

func loadActionsForAPI(workspacePath string) []apiActionEntry {
	config, errLoad := loadProjectConfig(workspacePath)
	if errLoad != nil || config == nil || len(config.Actions) == 0 {
		return convertActionsForAPI(defaultActionConfigs())
	}
	return convertActionsForAPI(config.Actions)
}

func convertActionsForAPI(configs []actionConfig) []apiActionEntry {
	actions := make([]apiActionEntry, 0, len(configs))
	for _, a := range configs {
		actions = append(actions, apiActionEntry(a))
	}
	return actions
}

func readGoalAndPMForAPI(dir string) (goalContent, rawGoalContent, fullGoalContent, pmContent string, hasProjectMgmt bool) {
	if data, errRead := os.ReadFile(filepath.Join(dir, "GOAL.md")); errRead == nil {
		fullGoalContent = string(data)
		stripped := stripFrontmatter(fullGoalContent)
		rawGoalContent = stripped
		if rendered, errRender := renderMarkdown([]byte(stripped)); errRender == nil {
			goalContent = rendered
		} else {
			goalContent = stripped
		}
	}

	if data, errRead := os.ReadFile(filepath.Join(dir, ".sgai", "PROJECT_MANAGEMENT.md")); errRead == nil {
		hasProjectMgmt = true
		stripped := stripFrontmatter(string(data))
		if rendered, errRender := renderMarkdown([]byte(stripped)); errRender == nil {
			pmContent = rendered
		} else {
			pmContent = stripped
		}
	}

	return goalContent, rawGoalContent, fullGoalContent, pmContent, hasProjectMgmt
}

func convertAgentSequence(displays []agentSequenceDisplay) []apiAgentSequenceEntry {
	result := make([]apiAgentSequenceEntry, 0, len(displays))
	for _, d := range displays {
		result = append(result, apiAgentSequenceEntry(d))
	}
	return result
}

type apiCreateWorkspaceRequest struct {
	Name string `json:"name"`
}

type apiCreateWorkspaceResponse struct {
	Name string `json:"name"`
	Dir  string `json:"dir"`
}

func (s *Server) handleAPICreateWorkspace(w http.ResponseWriter, r *http.Request) {
	var req apiCreateWorkspaceRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if errMsg := validateWorkspaceName(req.Name); errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	workspacePath := filepath.Join(s.rootDir, req.Name)
	if _, errStat := os.Stat(workspacePath); errStat == nil {
		http.Error(w, "a directory with this name already exists", http.StatusConflict)
		return
	} else if !os.IsNotExist(errStat) {
		http.Error(w, "failed to check workspace path", http.StatusInternalServerError)
		return
	}

	if errMkdir := os.MkdirAll(workspacePath, 0755); errMkdir != nil {
		http.Error(w, "failed to create workspace directory", http.StatusInternalServerError)
		return
	}

	if errInit := initializeWorkspace(workspacePath); errInit != nil {
		http.Error(w, "failed to initialize workspace", http.StatusInternalServerError)
		return
	}

	s.invalidateWorkspaceScanCache()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(apiCreateWorkspaceResponse{
		Name: req.Name,
		Dir:  workspacePath,
	}); err != nil {
		log.Println("failed to encode json response:", err)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Println("failed to encode json response:", err)
	}
}

func (s *Server) resolveWorkspaceFromPath(w http.ResponseWriter, r *http.Request) (string, bool) {
	workspaceName := r.PathValue("name")
	if workspaceName == "" {
		http.Error(w, "workspace name is required", http.StatusBadRequest)
		return "", false
	}
	workspacePath := s.resolveWorkspaceNameToPath(workspaceName)
	if workspacePath == "" {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return "", false
	}
	return workspacePath, true
}

type apiModelStatusEntry struct {
	ModelID string `json:"modelId"`
	Status  string `json:"status"`
}

func convertModelStatuses(displays []modelStatusDisplay) []apiModelStatusEntry {
	if len(displays) == 0 {
		return nil
	}
	result := make([]apiModelStatusEntry, 0, len(displays))
	for _, d := range displays {
		result = append(result, apiModelStatusEntry(d))
	}
	return result
}

type apiMessageEntry struct {
	ID        int    `json:"id"`
	FromAgent string `json:"fromAgent"`
	ToAgent   string `json:"toAgent"`
	Body      string `json:"body"`
	Subject   string `json:"subject"`
	Read      bool   `json:"read"`
	ReadAt    string `json:"readAt,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

const maxMessageBodyBytes = 10 * 1024

func convertMessagesForAPI(messages []state.Message) []apiMessageEntry {
	result := make([]apiMessageEntry, 0, len(messages))
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		subject := extractSubject(msg.Body)
		body := msg.Body
		if len(body) > maxMessageBodyBytes {
			body = body[:maxMessageBodyBytes] + "...[truncated]"
		}
		result = append(result, apiMessageEntry{
			ID:        msg.ID,
			FromAgent: msg.FromAgent,
			ToAgent:   msg.ToAgent,
			Body:      body,
			Subject:   subject,
			Read:      msg.Read,
			ReadAt:    msg.ReadAt,
			CreatedAt: msg.CreatedAt,
		})
	}
	return result
}

type apiTodoEntry struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

func convertTodosForAPI(todos []state.TodoItem) []apiTodoEntry {
	result := make([]apiTodoEntry, 0, len(todos))
	for _, t := range todos {
		result = append(result, apiTodoEntry{
			ID:       t.ID,
			Content:  t.Content,
			Status:   t.Status,
			Priority: t.Priority,
		})
	}
	return result
}

type apiLogEntry struct {
	Prefix string `json:"prefix"`
	Text   string `json:"text"`
}

type apiChangesResponse struct {
	Description string        `json:"description"`
	DiffLines   []apiDiffLine `json:"diffLines"`
}

type apiDiffLine struct {
	LineNumber int    `json:"lineNumber"`
	Text       string `json:"text"`
	Class      string `json:"class"`
}

func (s *Server) collectJJChangesCached(dir string) jjChangesResult {
	result, _ := s.jjChangesFlight.do(dir, func() (jjChangesResult, error) {
		diffLines, description := collectJJChanges(dir)
		return jjChangesResult{diffLines: diffLines, description: description}, nil
	})
	return result
}

func collectJJChanges(dir string) ([]apiDiffLine, string) {
	statCmd := exec.Command("jj", "diff", "--stat")
	statCmd.Dir = dir
	rawStat, errStat := statCmd.Output()
	if errStat != nil {
		return nil, ""
	}

	descCmd := exec.Command("jj", "log", "--no-graph", "-T", "description", "-r", "@")
	descCmd.Dir = dir
	rawDesc, errDesc := descCmd.Output()
	if errDesc != nil {
		rawDesc = nil
	}

	var diffLines []apiDiffLine
	for line := range strings.SplitSeq(string(rawStat), "\n") {
		if line == "" && len(diffLines) == 0 {
			continue
		}
		diffLines = append(diffLines, apiDiffLine{
			LineNumber: len(diffLines) + 1,
			Text:       line,
			Class:      "diff-stat-line",
		})
	}

	return diffLines, strings.TrimSpace(string(rawDesc))
}

func collectJJFullDiff(dir string) string {
	diffCmd := exec.Command("jj", "diff", "--git")
	diffCmd.Dir = dir
	rawDiff, errDiff := diffCmd.Output()
	if errDiff != nil {
		return ""
	}
	return string(rawDiff)
}

type apiFullDiffResponse struct {
	Diff string `json:"diff"`
}

func (s *Server) handleAPIWorkspaceDiff(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	if !hasJJRepo(workspacePath) {
		writeJSON(w, apiFullDiffResponse{})
		return
	}

	diff := collectJJFullDiff(workspacePath)
	writeJSON(w, apiFullDiffResponse{Diff: diff})
}

type apiEventEntry struct {
	Timestamp       string `json:"timestamp"`
	FormattedTime   string `json:"formattedTime"`
	Agent           string `json:"agent"`
	Description     string `json:"description"`
	ShowDateDivider bool   `json:"showDateDivider"`
	DateDivider     string `json:"dateDivider"`
}

type apiAgentModelEntry struct {
	Agent  string   `json:"agent"`
	Models []string `json:"models"`
}

func convertEventsForAPI(displays []eventsProgressDisplay) []apiEventEntry {
	result := make([]apiEventEntry, 0, len(displays))
	for _, d := range displays {
		result = append(result, apiEventEntry(d))
	}
	return result
}

type apiForkCommit struct {
	ChangeID    string   `json:"changeId"`
	CommitID    string   `json:"commitId"`
	Timestamp   string   `json:"timestamp"`
	Bookmarks   []string `json:"bookmarks,omitempty"`
	Description string   `json:"description"`
}

type apiForkEntry struct {
	Name        string          `json:"name"`
	Dir         string          `json:"dir"`
	Running     bool            `json:"running"`
	NeedsInput  bool            `json:"needsInput"`
	InProgress  bool            `json:"inProgress"`
	Pinned      bool            `json:"pinned"`
	CommitAhead int             `json:"commitAhead"`
	Commits     []apiForkCommit `json:"commits"`
	Summary     string          `json:"summary,omitempty"`
}

func convertJJCommitsForAPI(commits []jjCommit) []apiForkCommit {
	result := make([]apiForkCommit, 0, len(commits))
	for _, c := range commits {
		result = append(result, apiForkCommit{
			ChangeID:    c.ChangeID,
			CommitID:    c.CommitID,
			Timestamp:   c.Timestamp,
			Bookmarks:   c.Bookmarks,
			Description: c.Description,
		})
	}
	return result
}

func generateQuestionID(wfState state.Workflow) string {
	if !wfState.NeedsHumanInput() {
		return ""
	}
	h := sha256.New()
	h.Write([]byte(wfState.HumanMessage))
	if wfState.MultiChoiceQuestion != nil {
		data, _ := json.Marshal(wfState.MultiChoiceQuestion)
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func questionType(wfState state.Workflow) string {
	if wfState.MultiChoiceQuestion != nil {
		if wfState.MultiChoiceQuestion.IsWorkGate {
			return "work-gate"
		}
		return "multi-choice"
	}
	if wfState.HumanMessage != "" {
		return "free-text"
	}
	return ""
}

type apiQuestionItem struct {
	Question    string   `json:"question"`
	Choices     []string `json:"choices"`
	MultiSelect bool     `json:"multiSelect"`
}

type apiPendingQuestionResponse struct {
	QuestionID string            `json:"questionId"`
	Type       string            `json:"type"`
	AgentName  string            `json:"agentName"`
	Message    string            `json:"message"`
	Questions  []apiQuestionItem `json:"questions,omitempty"`
}

type apiRespondRequest struct {
	QuestionID      string   `json:"questionId"`
	Answer          string   `json:"answer"`
	SelectedChoices []string `json:"selectedChoices"`
}

type apiRespondResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (s *Server) handleAPIRespond(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	var req apiRespondRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	coord := s.sessionCoordinator(workspacePath)
	if coord != nil {
		s.handleRespondViaCoordinator(w, coord, req)
		return
	}

	s.handleRespondLegacy(w, workspacePath, req)
}

func (s *Server) sessionCoordinator(workspacePath string) *state.Coordinator {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.sessions[workspacePath]
	if sess == nil {
		return nil
	}
	sess.mu.Lock()
	defer sess.mu.Unlock()
	return sess.coord
}

func (s *Server) handleRespondViaCoordinator(w http.ResponseWriter, coord *state.Coordinator, req apiRespondRequest) {
	wfState := coord.State()

	if !wfState.NeedsHumanInput() {
		http.Error(w, "no pending question", http.StatusConflict)
		return
	}

	currentID := generateQuestionID(wfState)
	if req.QuestionID != currentID {
		http.Error(w, "question expired", http.StatusConflict)
		return
	}

	responseText := buildAPIResponseText(req, wfState.MultiChoiceQuestion)
	if responseText == "" {
		http.Error(w, "response cannot be empty", http.StatusBadRequest)
		return
	}

	if wfState.MultiChoiceQuestion != nil && wfState.MultiChoiceQuestion.IsWorkGate {
		approvedViaSelection := slices.Contains(req.SelectedChoices, workGateApprovalText)
		if approvedViaSelection {
			if err := coord.UpdateState(func(wf *state.Workflow) {
				if wf.InteractionMode == state.ModeBrainstorming {
					wf.InteractionMode = state.ModeBuilding
				}
			}); err != nil {
				http.Error(w, "failed to save work gate approval", http.StatusInternalServerError)
				return
			}
		}
	}

	coord.Respond(responseText)

	s.notifyStateChange()

	writeJSON(w, apiRespondResponse{Success: true, Message: "response submitted"})
}

func (s *Server) handleRespondLegacy(w http.ResponseWriter, workspacePath string, req apiRespondRequest) {
	coord := s.workspaceCoordinator(workspacePath)
	wfState := coord.State()

	if !wfState.NeedsHumanInput() {
		http.Error(w, "no pending question in legacy path", http.StatusConflict)
		return
	}

	currentID := generateQuestionID(wfState)
	if req.QuestionID != currentID {
		http.Error(w, "question expired", http.StatusConflict)
		return
	}

	responseText := buildAPIResponseText(req, wfState.MultiChoiceQuestion)
	if responseText == "" {
		http.Error(w, "response cannot be empty", http.StatusBadRequest)
		return
	}

	if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
		wf.Status = state.StatusWorking
		wf.HumanMessage = ""
		wf.MultiChoiceQuestion = nil
		wf.Task = ""
	}); errUpdate != nil {
		http.Error(w, "failed to save state", http.StatusInternalServerError)
		return
	}

	s.notifyStateChange()

	writeJSON(w, apiRespondResponse{Success: true, Message: "response submitted"})
}

func buildAPIResponseText(req apiRespondRequest, multiChoice *state.MultiChoiceQuestion) string {
	var parts []string
	if len(req.SelectedChoices) > 0 {
		parts = append(parts, "Selected: "+strings.Join(req.SelectedChoices, ", "))
	}
	if req.Answer != "" && (multiChoice == nil || !multiChoice.IsWorkGate) {
		parts = append(parts, req.Answer)
	} else if req.Answer != "" && multiChoice != nil && multiChoice.IsWorkGate {
		parts = append(parts, req.Answer)
	}
	return strings.Join(parts, "\n")
}

type apiStartSessionRequest struct {
	Auto bool `json:"auto"`
}

type apiSessionActionResponse struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Running bool   `json:"running"`
	Message string `json:"message"`
}

func (s *Server) handleAPIStartSession(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	if s.classifyWorkspaceCached(workspacePath) == workspaceRoot {
		http.Error(w, "root workspace cannot start agentic work", http.StatusBadRequest)
		return
	}

	var req apiStartSessionRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	coord := s.workspaceCoordinator(workspacePath)
	continuousPrompt := readContinuousModePrompt(workspacePath)

	var interactionMode string
	switch {
	case continuousPrompt != "":
		interactionMode = state.ModeContinuous
	case req.Auto:
		interactionMode = state.ModeSelfDrive
	default:
		interactionMode = state.ModeBrainstorming
	}

	if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
		wf.InteractionMode = interactionMode
	}); errUpdate != nil {
		http.Error(w, "failed to save workflow state", http.StatusInternalServerError)
		return
	}

	result := s.startSession(workspacePath)

	if result.alreadyRunning {
		writeJSON(w, apiSessionActionResponse{
			Name:    filepath.Base(workspacePath),
			Status:  "running",
			Running: true,
			Message: "session already running",
		})
		return
	}

	if result.startError != nil {
		http.Error(w, result.startError.Error(), http.StatusInternalServerError)
		return
	}

	s.notifyStateChange()

	writeJSON(w, apiSessionActionResponse{
		Name:    filepath.Base(workspacePath),
		Status:  "running",
		Running: true,
		Message: "session started",
	})
}

func (s *Server) handleAPIStopSession(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	s.mu.Lock()
	sess := s.sessions[workspacePath]
	s.mu.Unlock()

	var alreadyStopped bool
	if sess == nil {
		alreadyStopped = true
	} else {
		sess.mu.Lock()
		alreadyStopped = !sess.running
		sess.mu.Unlock()
	}

	s.stopSession(workspacePath)

	message := "session stopped"
	if alreadyStopped {
		message = "session already stopped"
	}

	s.notifyStateChange()

	writeJSON(w, apiSessionActionResponse{
		Name:    filepath.Base(workspacePath),
		Status:  "stopped",
		Running: false,
		Message: message,
	})
}

type apiComposeStateResponse struct {
	Workspace      string             `json:"workspace"`
	State          composerState      `json:"state"`
	Wizard         apiWizardState     `json:"wizard"`
	TechStackItems []apiTechStackItem `json:"techStackItems"`
	FlowError      string             `json:"flowError,omitempty"`
}

type apiWizardState struct {
	CurrentStep    int      `json:"currentStep"`
	FromTemplate   string   `json:"fromTemplate,omitempty"`
	Description    string   `json:"description,omitempty"`
	TechStack      []string `json:"techStack"`
	SafetyAnalysis bool     `json:"safetyAnalysis"`
	CompletionGate string   `json:"completionGate,omitempty"`
}

type apiTechStackItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
}

func (s *Server) handleAPIComposeState(w http.ResponseWriter, r *http.Request) {
	workspacePath := s.resolveAPIWorkspace(r)
	if workspacePath == "" {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	cs := s.getComposerSession(workspacePath)
	cs.mu.Lock()
	currentState := cs.state
	wizard := syncWizardState(cs.wizard, currentState)
	cs.mu.Unlock()

	var flowErr string
	if currentState.Flow != "" {
		if _, errParse := parseFlow(currentState.Flow, workspacePath); errParse != nil {
			flowErr = errParse.Error()
		}
	}

	techStack := buildAPITechStackItems(wizard.TechStack)

	writeJSON(w, apiComposeStateResponse{
		Workspace:      filepath.Base(workspacePath),
		State:          currentState,
		Wizard:         apiWizardState(wizard),
		TechStackItems: techStack,
		FlowError:      flowErr,
	})
}

func buildAPITechStackItems(selectedTech []string) []apiTechStackItem {
	selectedMap := make(map[string]bool)
	for _, ts := range selectedTech {
		selectedMap[ts] = true
	}

	items := make([]apiTechStackItem, len(defaultTechStackItems))
	for i, item := range defaultTechStackItems {
		items[i] = apiTechStackItem{
			ID:       item.ID,
			Name:     item.Name,
			Selected: selectedMap[item.ID],
		}
	}
	return items
}

type apiComposeSaveResponse struct {
	Saved     bool   `json:"saved"`
	Workspace string `json:"workspace"`
}

func (s *Server) handleAPIComposeSave(w http.ResponseWriter, r *http.Request) {
	workspacePath := s.resolveAPIWorkspace(r)
	if workspacePath == "" {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	goalPath := filepath.Join(workspacePath, "GOAL.md")

	ifMatch := r.Header.Get("If-Match")
	if ifMatch != "" {
		currentContent, errRead := os.ReadFile(goalPath)
		if errRead != nil && !os.IsNotExist(errRead) {
			http.Error(w, "failed to read current GOAL.md", http.StatusInternalServerError)
			return
		}
		currentEtag := computeEtag(currentContent)
		if ifMatch != currentEtag {
			http.Error(w, "GOAL.md has been modified by another session", http.StatusPreconditionFailed)
			return
		}
	}

	cs := s.getComposerSession(workspacePath)
	cs.mu.Lock()
	currentState := cs.state
	cs.mu.Unlock()

	goalContent := buildGOALContent(currentState)

	if errWrite := os.WriteFile(goalPath, []byte(goalContent), 0644); errWrite != nil {
		http.Error(w, "failed to save GOAL.md: "+errWrite.Error(), http.StatusInternalServerError)
		return
	}

	s.notifyStateChange()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(apiComposeSaveResponse{
		Saved:     true,
		Workspace: filepath.Base(workspacePath),
	}); err != nil {
		log.Println("failed to encode json response:", err)
	}
}

func computeEtag(content []byte) string {
	h := sha256.Sum256(content)
	return `"` + hex.EncodeToString(h[:8]) + `"`
}

type apiComposeTemplateEntry struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Icon        string              `json:"icon"`
	Agents      []composerAgentConf `json:"agents"`
	Flow        string              `json:"flow"`
}

type apiComposeTemplatesResponse struct {
	Templates []apiComposeTemplateEntry `json:"templates"`
}

func (s *Server) handleAPIComposeTemplates(w http.ResponseWriter, _ *http.Request) {
	entries := make([]apiComposeTemplateEntry, len(workflowTemplates))
	for i, tmpl := range workflowTemplates {
		entries[i] = apiComposeTemplateEntry(tmpl)
	}

	writeJSON(w, apiComposeTemplatesResponse{Templates: entries})
}

type apiComposePreviewResponse struct {
	Content   string `json:"content"`
	FlowError string `json:"flowError,omitempty"`
	Etag      string `json:"etag"`
}

func (s *Server) handleAPIComposePreview(w http.ResponseWriter, r *http.Request) {
	workspacePath := s.resolveAPIWorkspace(r)
	if workspacePath == "" {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	cs := s.getComposerSession(workspacePath)
	cs.mu.Lock()
	currentState := cs.state
	cs.mu.Unlock()

	preview := buildGOALContent(currentState)

	var flowErr string
	if currentState.Flow != "" {
		if _, errParse := parseFlow(currentState.Flow, workspacePath); errParse != nil {
			flowErr = errParse.Error()
		}
	}

	goalPath := filepath.Join(workspacePath, "GOAL.md")
	existingContent, errRead := os.ReadFile(goalPath)
	if errRead != nil && !os.IsNotExist(errRead) {
		http.Error(w, "failed to read current GOAL.md", http.StatusInternalServerError)
		return
	}
	etag := computeEtag(existingContent)

	writeJSON(w, apiComposePreviewResponse{
		Content:   preview,
		FlowError: flowErr,
		Etag:      etag,
	})
}

type apiComposeDraftRequest struct {
	State  composerState  `json:"state"`
	Wizard apiWizardState `json:"wizard"`
}

type apiComposeDraftResponse struct {
	Saved bool `json:"saved"`
}

func (s *Server) handleAPIComposeDraft(w http.ResponseWriter, r *http.Request) {
	workspacePath := s.resolveAPIWorkspace(r)
	if workspacePath == "" {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	var req apiComposeDraftRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	cs := s.getComposerSession(workspacePath)
	cs.mu.Lock()
	cs.state = req.State
	cs.wizard = wizardState(req.Wizard)
	cs.mu.Unlock()

	s.notifyStateChange()

	writeJSON(w, apiComposeDraftResponse{Saved: true})
}

type apiForkRequest struct {
	Name string `json:"name"`
}

type apiForkResponse struct {
	Name      string `json:"name"`
	Dir       string `json:"dir"`
	Parent    string `json:"parent"`
	CreatedAt string `json:"createdAt"`
}

func (s *Server) handleAPIForkWorkspace(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	if s.classifyWorkspaceCached(workspacePath) == workspaceFork {
		http.Error(w, "forks cannot create new forks", http.StatusBadRequest)
		return
	}

	var req apiForkRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	name := normalizeForkName(req.Name)
	if errMsg := validateWorkspaceName(name); errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	parentDir := filepath.Dir(workspacePath)
	forkPath := filepath.Join(parentDir, name)
	if _, errStat := os.Stat(forkPath); errStat == nil {
		http.Error(w, "a directory with this name already exists", http.StatusConflict)
		return
	} else if !os.IsNotExist(errStat) {
		http.Error(w, "failed to check workspace path", http.StatusInternalServerError)
		return
	}

	cmd := exec.Command("jj", "workspace", "add", forkPath)
	cmd.Dir = workspacePath
	output, errFork := cmd.CombinedOutput()
	if errFork != nil {
		http.Error(w, fmt.Sprintf("failed to fork workspace: %v: %s", errFork, output), http.StatusInternalServerError)
		return
	}

	if errSkel := unpackSkeleton(forkPath); errSkel != nil {
		log.Println("failed to unpack skeleton for fork:", errSkel)
	}
	if errExclude := addGitExclude(forkPath); errExclude != nil {
		log.Println("failed to add git exclude for fork:", errExclude)
	}
	if errGoal := writeGoalExample(forkPath); errGoal != nil {
		log.Println("failed to create GOAL.md for fork:", errGoal)
	}

	s.invalidateWorkspaceScanCache()
	s.classifyCache.delete(workspacePath)
	s.notifyStateChange()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(apiForkResponse{
		Name:   name,
		Dir:    forkPath,
		Parent: filepath.Base(workspacePath),
	}); err != nil {
		log.Println("failed to encode json response:", err)
	}
}

type apiDeleteForkRequest struct {
	ForkDir string `json:"forkDir"`
	Confirm bool   `json:"confirm"`
}

type apiDeleteForkResponse struct {
	Deleted bool   `json:"deleted"`
	Message string `json:"message"`
}

func (s *Server) handleAPIDeleteFork(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	if s.classifyWorkspaceCached(workspacePath) != workspaceRoot {
		http.Error(w, "workspace is not a root", http.StatusBadRequest)
		return
	}

	var req apiDeleteForkRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if !req.Confirm {
		http.Error(w, "confirmation required to delete fork", http.StatusBadRequest)
		return
	}

	forkDir, errValidate := s.validateDirectory(req.ForkDir)
	if errValidate != nil {
		http.Error(w, "invalid fork directory", http.StatusBadRequest)
		return
	}

	if s.classifyWorkspaceCached(forkDir) != workspaceFork {
		http.Error(w, "fork workspace not found", http.StatusBadRequest)
		return
	}

	if getRootWorkspacePath(forkDir) != workspacePath {
		http.Error(w, "fork does not belong to root", http.StatusBadRequest)
		return
	}

	forkName := filepath.Base(forkDir)

	s.stopSession(forkDir)

	forgetCmd := exec.Command("jj", "workspace", "forget", forkName)
	forgetCmd.Dir = workspacePath
	if _, errForget := forgetCmd.CombinedOutput(); errForget != nil {
		http.Error(w, "failed to forget fork workspace", http.StatusInternalServerError)
		return
	}

	if errRemove := os.RemoveAll(forkDir); errRemove != nil {
		http.Error(w, "failed to remove fork directory", http.StatusInternalServerError)
		return
	}

	s.invalidateWorkspaceScanCache()
	s.classifyCache.delete(workspacePath)
	s.classifyCache.delete(forkDir)
	s.notifyStateChange()

	writeJSON(w, apiDeleteForkResponse{
		Deleted: true,
		Message: "fork deleted successfully",
	})
}

type apiRenameRequest struct {
	Name string `json:"name"`
}

type apiRenameResponse struct {
	Name    string `json:"name"`
	OldName string `json:"oldName"`
	Dir     string `json:"dir"`
}

func (s *Server) handleAPIRenameWorkspace(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	if s.classifyWorkspaceCached(workspacePath) != workspaceFork {
		http.Error(w, "only forks can be renamed", http.StatusBadRequest)
		return
	}

	var req apiRenameRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	newName := normalizeForkName(req.Name)
	if errMsg := validateWorkspaceName(newName); errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	sess := s.sessions[workspacePath]
	s.mu.Unlock()
	if sess != nil {
		sess.mu.Lock()
		running := sess.running
		sess.mu.Unlock()
		if running {
			http.Error(w, "cannot rename: session is running", http.StatusConflict)
			return
		}
	}

	oldName := filepath.Base(workspacePath)
	parentDir := filepath.Dir(workspacePath)
	newPath := filepath.Join(parentDir, newName)
	if _, errStat := os.Stat(newPath); errStat == nil {
		http.Error(w, "a directory with this name already exists", http.StatusConflict)
		return
	} else if !os.IsNotExist(errStat) {
		http.Error(w, "failed to check target path", http.StatusInternalServerError)
		return
	}

	if errRename := os.Rename(workspacePath, newPath); errRename != nil {
		http.Error(w, fmt.Sprintf("failed to rename directory: %v", errRename), http.StatusInternalServerError)
		return
	}

	cmd := exec.Command("jj", "workspace", "rename", newName)
	cmd.Dir = newPath
	if output, errJJ := cmd.CombinedOutput(); errJJ != nil {
		log.Println("jj workspace rename failed:", errJJ, string(output))
	}

	s.mu.Lock()
	if sess, ok := s.sessions[workspacePath]; ok {
		delete(s.sessions, workspacePath)
		s.sessions[newPath] = sess
	}
	if s.everStartedDirs[workspacePath] {
		delete(s.everStartedDirs, workspacePath)
		s.everStartedDirs[newPath] = true
	}
	s.mu.Unlock()

	s.invalidateWorkspaceScanCache()
	s.notifyStateChange()

	writeJSON(w, apiRenameResponse{
		Name:    newName,
		OldName: oldName,
		Dir:     newPath,
	})
}

type apiGoalResponse struct {
	Content string `json:"content"`
}

func (s *Server) handleAPIGetGoal(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	data, errRead := os.ReadFile(filepath.Join(workspacePath, "GOAL.md"))
	if errRead != nil {
		http.Error(w, "failed to read GOAL.md", http.StatusInternalServerError)
		return
	}

	writeJSON(w, apiGoalResponse{Content: string(data)})
}

type apiUpdateGoalRequest struct {
	Content string `json:"content"`
}

type apiUpdateGoalResponse struct {
	Updated   bool   `json:"updated"`
	Workspace string `json:"workspace"`
}

func (s *Server) handleAPIUpdateGoal(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	var req apiUpdateGoalRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "content cannot be empty", http.StatusBadRequest)
		return
	}

	goalPath := filepath.Join(workspacePath, "GOAL.md")
	if errWrite := os.WriteFile(goalPath, []byte(req.Content), 0644); errWrite != nil {
		http.Error(w, "failed to write GOAL.md", http.StatusInternalServerError)
		return
	}

	prefix := workspacePath + "|"
	s.svgCache.deleteFunc(func(k string) bool {
		return strings.HasPrefix(k, prefix)
	})
	s.notifyStateChange()

	writeJSON(w, apiUpdateGoalResponse{
		Updated:   true,
		Workspace: filepath.Base(workspacePath),
	})
}

type apiUpdateSummaryRequest struct {
	Summary string `json:"summary"`
}

type apiUpdateSummaryResponse struct {
	Updated   bool   `json:"updated"`
	Summary   string `json:"summary"`
	Workspace string `json:"workspace"`
}

func (s *Server) handleAPIUpdateSummary(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	var req apiUpdateSummaryRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	coord := s.workspaceCoordinator(workspacePath)
	summary := strings.TrimSpace(req.Summary)

	if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
		wf.Summary = summary
		wf.SummaryManual = true
	}); errUpdate != nil {
		http.Error(w, "failed to save workspace state", http.StatusInternalServerError)
		return
	}

	workspaceName := filepath.Base(workspacePath)
	s.notifyStateChange()

	writeJSON(w, apiUpdateSummaryResponse{
		Updated:   true,
		Summary:   summary,
		Workspace: workspaceName,
	})
}

type apiAdhocRequest struct {
	Prompt string `json:"prompt"`
	Model  string `json:"model"`
}

type apiAdhocResponse struct {
	Running bool   `json:"running"`
	Output  string `json:"output"`
	Message string `json:"message"`
}

func (s *Server) handleAPIAdhocStatus(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	st := s.getAdhocState(workspacePath)
	st.mu.Lock()
	running := st.running
	output := st.output.String()
	st.mu.Unlock()

	writeJSON(w, apiAdhocResponse{
		Running: running,
		Output:  output,
		Message: "adhoc status",
	})
}

func (s *Server) handleAPIAdhoc(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	var req apiAdhocRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Prompt) == "" || strings.TrimSpace(req.Model) == "" {
		http.Error(w, "prompt and model are required", http.StatusBadRequest)
		return
	}

	st := s.getAdhocState(workspacePath)
	st.mu.Lock()
	if st.running {
		output := st.output.String()
		st.mu.Unlock()
		writeJSON(w, apiAdhocResponse{
			Running: true,
			Output:  output,
			Message: "ad-hoc prompt already running",
		})
		return
	}

	st.running = true
	st.output.Reset()
	st.selectedModel = strings.TrimSpace(req.Model)
	st.promptText = strings.TrimSpace(req.Prompt)

	args := buildAdhocArgs(st.selectedModel)
	cmd := exec.Command("opencode", args...)
	cmd.Dir = workspacePath
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = append(os.Environ(), "OPENCODE_CONFIG_DIR="+filepath.Join(workspacePath, ".sgai"))
	cmd.Stdin = strings.NewReader(st.promptText)
	writer := &lockedWriter{mu: &st.mu, buf: &st.output}
	prefix := fmt.Sprintf("[%s][adhoc:0000]", filepath.Base(workspacePath))
	stdoutPW := &prefixWriter{prefix: prefix + " ", w: os.Stdout, startTime: time.Now()}
	stderrPW := &prefixWriter{prefix: prefix + " ", w: os.Stderr, startTime: time.Now()}
	cmd.Stdout = io.MultiWriter(stdoutPW, writer)
	cmd.Stderr = io.MultiWriter(stderrPW, writer)
	commandLine := "$ opencode " + strings.Join(args, " ")
	promptLine := "prompt: " + st.promptText
	_, _ = fmt.Fprintln(stderrPW, commandLine)
	_, _ = fmt.Fprintln(stderrPW, promptLine)
	st.output.WriteString(commandLine + "\n")
	st.output.WriteString(promptLine + "\n")

	if errStart := cmd.Start(); errStart != nil {
		st.running = false
		st.mu.Unlock()
		http.Error(w, "failed to start command", http.StatusInternalServerError)
		return
	}

	st.cmd = cmd
	st.mu.Unlock()

	go func() {
		errWait := cmd.Wait()
		st.mu.Lock()
		if errWait != nil {
			st.output.WriteString("\n[command exited with error: " + errWait.Error() + "]\n")
		}
		st.running = false
		st.cmd = nil
		st.mu.Unlock()
	}()

	s.notifyStateChange()

	writeJSON(w, apiAdhocResponse{
		Running: true,
		Message: "ad-hoc prompt started",
	})
}

func buildAdhocArgs(modelSpec string) []string {
	baseModel, variant := parseModelAndVariant(modelSpec)
	args := []string{"run", "-m", baseModel, "--agent", "build", "--title", "adhoc [" + modelSpec + "]"}
	if variant != "" {
		args = append(args, "--variant", variant)
	}
	return args
}

func (s *Server) handleAPIAdhocStop(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	st := s.getAdhocState(workspacePath)
	st.stop()
	s.notifyStateChange()

	st.mu.Lock()
	output := st.output.String()
	st.mu.Unlock()

	writeJSON(w, apiAdhocResponse{
		Running: false,
		Output:  output,
		Message: "ad-hoc stopped",
	})
}

func (s *Server) handleAPIWorkflowSVG(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	wfState := s.workspaceCoordinator(workspacePath).State()
	currentAgent := wfState.CurrentAgent
	if currentAgent == "" {
		currentAgent = "Unknown"
	}

	svg := s.getWorkflowSVGCached(workspacePath, currentAgent)
	if svg == "" {
		http.Error(w, "workflow SVG not available", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-cache")
	if _, errWrite := w.Write([]byte(svg)); errWrite != nil {
		log.Println("failed to write workflow SVG:", errWrite)
	}
}

type apiCommitEntry struct {
	ChangeID    string   `json:"changeId"`
	CommitID    string   `json:"commitId"`
	Workspaces  []string `json:"workspaces,omitempty"`
	Timestamp   string   `json:"timestamp"`
	Bookmarks   []string `json:"bookmarks,omitempty"`
	Description string   `json:"description"`
	GraphChar   string   `json:"graphChar"`
}

func runJJLogForWorkspace(dir string) []jjCommit {
	cmd := exec.Command("jj", "log", "-n", "50", "-T", jjLogTemplate)
	cmd.Dir = dir
	output, errCmd := cmd.Output()
	if errCmd != nil {
		return nil
	}
	return parseJJLogOutput(string(output))
}

func (s *Server) runJJLogForWorkspaceCached(dir string) []jjCommit {
	commits, _ := s.workspaceLogFlight.do(dir, func() ([]jjCommit, error) {
		return runJJLogForWorkspace(dir), nil
	})
	return commits
}

func runJJLogForStandalone(dir string) []jjCommit {
	cmd := exec.Command("jj", "log", "-r", "remote_bookmarks()..@", "-T", jjLogTemplate)
	cmd.Dir = dir
	output, errCmd := cmd.Output()
	if errCmd != nil {
		return nil
	}
	return parseJJLogOutput(string(output))
}

func (s *Server) runJJLogForStandaloneCached(dir string) []jjCommit {
	key := "standalone|" + dir
	commits, _ := s.workspaceLogFlight.do(key, func() ([]jjCommit, error) {
		return runJJLogForStandalone(dir), nil
	})
	return commits
}

func (s *Server) filteredCommitsForWorkspace(workspacePath string) []jjCommit {
	switch s.classifyWorkspaceCached(workspacePath) {
	case workspaceStandalone:
		commits := s.runJJLogForStandaloneCached(workspacePath)
		if len(commits) > 0 {
			return commits
		}
		return s.runJJLogForWorkspaceCached(workspacePath)
	case workspaceFork:
		rootDir := getRootWorkspacePath(workspacePath)
		if rootDir == "" {
			return s.runJJLogForWorkspaceCached(workspacePath)
		}
		bookmark := s.resolveBaseBookmarkCached(rootDir)
		commits := s.runJJLogForForkCached(bookmark, workspacePath)
		if len(commits) > 0 {
			return commits
		}
		return s.runJJLogForWorkspaceCached(workspacePath)
	default:
		return s.runJJLogForWorkspaceCached(workspacePath)
	}
}

type apiSteerRequest struct {
	Message string `json:"message"`
}

type apiSteerResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (s *Server) handleAPISteer(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	var req apiSteerRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Message) == "" {
		http.Error(w, "message cannot be empty", http.StatusBadRequest)
		return
	}

	coord := s.workspaceCoordinator(workspacePath)
	steerBody := "Re-steering instruction: " + strings.TrimSpace(req.Message)
	steerCreatedAt := time.Now().UTC().Format(time.RFC3339)

	if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
		newMsg := state.Message{
			ID:        nextMessageID(wf.Messages),
			FromAgent: "Human Partner",
			ToAgent:   "coordinator",
			Body:      steerBody,
			CreatedAt: steerCreatedAt,
		}
		insertIdx := findSteerInsertPosition(wf.Messages)
		wf.Messages = slices.Insert(wf.Messages, insertIdx, newMsg)
	}); errUpdate != nil {
		http.Error(w, "failed to save state", http.StatusInternalServerError)
		return
	}

	s.notifyStateChange()

	writeJSON(w, apiSteerResponse{Success: true, Message: "steering instruction added"})
}

func findSteerInsertPosition(messages []state.Message) int {
	for i, msg := range messages {
		if !msg.Read {
			return i
		}
	}
	return 0
}

type apiUpdateDescriptionRequest struct {
	Description string `json:"description"`
}

type apiUpdateDescriptionResponse struct {
	Updated     bool   `json:"updated"`
	Description string `json:"description"`
}

func (s *Server) handleAPIUpdateDescription(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	var req apiUpdateDescriptionRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	cmd := exec.Command("jj", "desc", "-m", req.Description)
	cmd.Dir = workspacePath
	if output, errCmd := cmd.CombinedOutput(); errCmd != nil {
		http.Error(w, fmt.Sprintf("failed to update description: %s", output), http.StatusInternalServerError)
		return
	}

	s.notifyStateChange()

	writeJSON(w, apiUpdateDescriptionResponse{
		Updated:     true,
		Description: req.Description,
	})
}

type apiTogglePinResponse struct {
	Pinned  bool   `json:"pinned"`
	Message string `json:"message"`
}

func (s *Server) handleAPITogglePin(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	if errToggle := s.togglePin(workspacePath); errToggle != nil {
		http.Error(w, "failed to toggle pin", http.StatusInternalServerError)
		return
	}

	s.invalidateWorkspaceScanCache()
	s.notifyStateChange()

	pinned := s.isPinned(workspacePath)

	writeJSON(w, apiTogglePinResponse{
		Pinned:  pinned,
		Message: "pin toggled",
	})
}

type apiOpenEditorResponse struct {
	Opened  bool   `json:"opened"`
	Editor  string `json:"editor"`
	Message string `json:"message"`
}

func (s *Server) handleAPIOpenEditor(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	if !s.editorAvailable {
		http.Error(w, "no editor available", http.StatusServiceUnavailable)
		return
	}

	if errOpen := s.editor.open(workspacePath); errOpen != nil {
		http.Error(w, fmt.Sprintf("failed to open editor: %v", errOpen), http.StatusInternalServerError)
		return
	}

	writeJSON(w, apiOpenEditorResponse{
		Opened:  true,
		Editor:  s.editorName,
		Message: "opened in editor",
	})
}

type apiOpenInOpenCodeResponse struct {
	Opened  bool   `json:"opened"`
	Message string `json:"message"`
}

func (s *Server) handleAPIOpenInOpenCode(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	if !isLocalRequest(r) {
		http.Error(w, "opencode can only be opened from localhost", http.StatusForbidden)
		return
	}

	s.mu.Lock()
	sess := s.sessions[workspacePath]
	s.mu.Unlock()
	if sess == nil {
		http.Error(w, "factory is not running", http.StatusConflict)
		return
	}
	sess.mu.Lock()
	running := sess.running
	sess.mu.Unlock()
	if !running {
		http.Error(w, "factory is not running", http.StatusConflict)
		return
	}

	wfState := s.workspaceCoordinator(workspacePath).State()
	currentAgent := wfState.CurrentAgent
	sessionID := wfState.SessionID

	models := modelsForAgentFromGoal(workspacePath, currentAgent)
	var model string
	if len(models) > 0 {
		model, _ = parseModelAndVariant(models[0])
	}

	interactive := "yes"
	if wfState.InteractionMode == state.ModeSelfDrive {
		interactive = "auto"
	}

	execPath, errExec := os.Executable()
	if errExec != nil {
		http.Error(w, "failed to resolve executable path", http.StatusInternalServerError)
		return
	}

	opencodeCmd := fmt.Sprintf("opencode --session %q --agent %q", sessionID, currentAgent)
	if model != "" {
		opencodeCmd += fmt.Sprintf(" --model %q", model)
	}
	scriptContent := fmt.Sprintf("#!/bin/bash\ntrap 'rm -f \"$0\"' EXIT\ncd %q\nexport OPENCODE_CONFIG_DIR=.sgai\nexport SGAI_MCP_EXECUTABLE=%q\nexport SGAI_MCP_INTERACTIVE=%q\n%s\n",
		workspacePath, execPath, interactive, opencodeCmd)

	scriptPath, errScript := writeOpenCodeScript(scriptContent)
	if errScript != nil {
		http.Error(w, "failed to prepare opencode script", http.StatusInternalServerError)
		return
	}

	if errOpen := openInTerminal(scriptPath); errOpen != nil {
		_ = os.Remove(scriptPath)
		http.Error(w, "failed to open terminal", http.StatusInternalServerError)
		return
	}

	writeJSON(w, apiOpenInOpenCodeResponse{
		Opened:  true,
		Message: "opened in opencode",
	})
}

func (s *Server) handleAPIOpenEditorGoal(w http.ResponseWriter, r *http.Request) {
	s.openEditorForFile(w, r, "GOAL.md")
}

func (s *Server) handleAPIOpenEditorProjectManagement(w http.ResponseWriter, r *http.Request) {
	s.openEditorForFile(w, r, filepath.Join(".sgai", "PROJECT_MANAGEMENT.md"))
}

func (s *Server) openEditorForFile(w http.ResponseWriter, r *http.Request, relPath string) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	if !s.editorAvailable {
		http.Error(w, "no editor available", http.StatusServiceUnavailable)
		return
	}

	targetPath := filepath.Join(workspacePath, relPath)
	if _, errStat := os.Stat(targetPath); errStat != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	if errOpen := s.editor.open(targetPath); errOpen != nil {
		http.Error(w, fmt.Sprintf("failed to open editor: %v", errOpen), http.StatusInternalServerError)
		return
	}

	writeJSON(w, apiOpenEditorResponse{
		Opened:  true,
		Editor:  s.editorName,
		Message: "opened in editor",
	})
}

type apiModelEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type apiModelsResponse struct {
	Models       []apiModelEntry `json:"models"`
	DefaultModel string          `json:"defaultModel,omitempty"`
}

func (s *Server) handleAPIListModels(w http.ResponseWriter, r *http.Request) {
	validModels, errFetch := fetchValidModels()
	if errFetch != nil {
		log.Println("cannot fetch models:", errFetch)
		writeJSON(w, apiModelsResponse{Models: []apiModelEntry{}})
		return
	}

	modelNames := slices.Sorted(maps.Keys(validModels))
	entries := make([]apiModelEntry, 0, len(modelNames))
	for _, name := range modelNames {
		entries = append(entries, apiModelEntry{
			ID:   name,
			Name: name,
		})
	}

	defaultModel := s.coordinatorModelFromWorkspace(r.URL.Query().Get("workspace"))
	writeJSON(w, apiModelsResponse{Models: entries, DefaultModel: defaultModel})
}

func (s *Server) coordinatorModelFromWorkspace(workspace string) string {
	if workspace == "" {
		return ""
	}
	workspacePath := s.resolveWorkspaceNameToPath(workspace)
	if workspacePath == "" {
		return ""
	}
	models := modelsForAgentFromGoal(workspacePath, "coordinator")
	if len(models) == 0 {
		return ""
	}
	baseModel, _ := parseModelAndVariant(models[0])
	return baseModel
}

func resolveCurrentModel(workspacePath string, wfState state.Workflow) string {
	if wfState.CurrentModel != "" {
		return wfState.CurrentModel
	}
	agent := wfState.CurrentAgent
	if agent == "" {
		return ""
	}
	models := modelsForAgentFromGoal(workspacePath, agent)
	if len(models) == 0 {
		return ""
	}
	return models[0]
}

func collectAgentModels(workspacePath string) []apiAgentModelEntry {
	goalPath := filepath.Join(workspacePath, "GOAL.md")
	goalData, errRead := os.ReadFile(goalPath)
	if errRead != nil {
		return nil
	}
	metadata, errParse := parseYAMLFrontmatter(goalData)
	if errParse != nil || len(metadata.Models) == 0 {
		return nil
	}
	agents := slices.Sorted(maps.Keys(metadata.Models))
	result := make([]apiAgentModelEntry, 0, len(agents))
	for _, agent := range agents {
		models := getModelsForAgent(metadata.Models, agent)
		if len(models) == 0 {
			continue
		}
		result = append(result, apiAgentModelEntry{
			Agent:  agent,
			Models: models,
		})
	}
	return result
}

// apiDeleteMessageResponse represents the response from deleting a message.
type apiDeleteMessageResponse struct {
	Deleted bool   `json:"deleted"`
	ID      int    `json:"id"`
	Message string `json:"message"`
}

// handleAPIDeleteMessage handles DELETE requests to remove a message from a workspace.
func (s *Server) handleAPIDeleteMessage(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "message id is required", http.StatusBadRequest)
		return
	}

	var messageID int
	if _, errScan := fmt.Sscanf(idStr, "%d", &messageID); errScan != nil {
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}

	coord := s.workspaceCoordinator(workspacePath)
	wfState := coord.State()

	found := false
	for _, msg := range wfState.Messages {
		if msg.ID == messageID {
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "message not found", http.StatusNotFound)
		return
	}

	if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
		for i, msg := range wf.Messages {
			if msg.ID == messageID {
				wf.Messages = slices.Delete(wf.Messages, i, i+1)
				break
			}
		}
	}); errUpdate != nil {
		http.Error(w, "failed to save workspace state", http.StatusInternalServerError)
		return
	}

	s.notifyStateChange()

	writeJSON(w, apiDeleteMessageResponse{
		Deleted: true,
		ID:      messageID,
		Message: "message deleted successfully",
	})
}
