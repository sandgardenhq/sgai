package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	mux.HandleFunc("GET /api/v1/usage", s.handleAPIUsage)
	mux.HandleFunc("POST /api/v1/usage/refresh", s.handleAPIUsageRefresh)
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
	mux.HandleFunc("POST /api/v1/workspaces/{name}/reset", s.handleAPIResetWorkspace)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/fork", s.handleAPIForkWorkspace)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/delete-fork", s.handleAPIDeleteFork)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/delete", s.handleAPIDeleteWorkspace)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/goal", s.handleAPIGetGoal)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/fork-template", s.handleAPIForkTemplate)
	mux.HandleFunc("PUT /api/v1/workspaces/{name}/goal", s.handleAPIUpdateGoal)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/adhoc", s.handleAPIAdhocStatus)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/adhoc", s.handleAPIAdhoc)
	mux.HandleFunc("DELETE /api/v1/workspaces/{name}/adhoc", s.handleAPIAdhocStop)

	mux.HandleFunc("POST /api/v1/workspaces/{name}/pin", s.handleAPITogglePin)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/open-editor", s.handleAPIOpenEditor)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/open-editor/goal", s.handleAPIOpenEditorGoal)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/open-editor/project-management", s.handleAPIOpenEditorProjectManagement)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/diff", s.handleAPIWorkspaceDiff)
	mux.HandleFunc("GET /api/v1/models", s.handleAPIListModels)
	mux.HandleFunc("GET /api/v1/compose", s.handleAPIComposeState)
	mux.HandleFunc("POST /api/v1/compose", s.handleAPIComposeSave)
	mux.HandleFunc("GET /api/v1/compose/templates", s.handleAPIComposeTemplates)
	mux.HandleFunc("GET /api/v1/compose/preview", s.handleAPIComposePreview)
	mux.HandleFunc("POST /api/v1/compose/draft", s.handleAPIComposeDraft)

	mux.HandleFunc("GET /api/v1/browse-directories", s.handleAPIBrowseDirectories)
	mux.HandleFunc("POST /api/v1/workspaces/attach", s.handleAPIAttachWorkspace)
	mux.HandleFunc("POST /api/v1/workspaces/detach", s.handleAPIDetachWorkspace)
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
	IsExternal      bool                        `json:"isExternal"`
	HasSGAI         bool                        `json:"hasSgai"`
	Status          string                      `json:"status"`
	BadgeClass      string                      `json:"badgeClass"`
	BadgeText       string                      `json:"badgeText"`
	HasEditedGoal   bool                        `json:"hasEditedGoal"`
	InteractiveAuto bool                        `json:"interactiveAuto"`
	ContinuousMode  bool                        `json:"continuousMode"`
	Task            string                      `json:"task"`
	GoalContent     string                      `json:"goalContent"`
	Description     string                      `json:"description"`
	RawGoalContent  string                      `json:"rawGoalContent"`
	FullGoalContent string                      `json:"fullGoalContent"`
	PMContent       string                      `json:"pmContent"`
	HasProjectMgmt  bool                        `json:"hasProjectMgmt"`
	TotalExecTime   string                      `json:"totalExecTime"`
	LatestProgress  string                      `json:"latestProgress"`
	HumanMessage    string                      `json:"humanMessage"`
	Cost            state.SessionCost           `json:"cost"`
	Events          []apiEventEntry             `json:"events"`
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

	reversedProgress := slices.Clone(wfState.Progress)
	slices.Reverse(reversedProgress)
	events := convertEventsForAPI(formatProgressForDisplay(reversedProgress))

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
			Message:    wfState.HumanMessage,
			Questions:  questions,
		}
	}

	description := extractGoalDescription(fullGoalContent)
	if description == "" {
		description = ws.DirName
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
		IsExternal:      ws.External,
		HasSGAI:         ws.HasWorkspace,
		Status:          status,
		BadgeClass:      badgeClass,
		BadgeText:       badgeText,
		HasEditedGoal:   hasEditedGoal,
		InteractiveAuto: interactiveAuto,
		ContinuousMode:  readContinuousModePrompt(ws.Directory) != "",
		Task:            wfState.Task,
		GoalContent:     goalContent,
		Description:     description,
		RawGoalContent:  rawGoalContent,
		FullGoalContent: fullGoalContent,
		PMContent:       pmContent,
		HasProjectMgmt:  hasProjectMgmt,
		TotalExecTime:   calculateTotalExecutionTime(wfState.Progress, ws.Running),
		LatestProgress:  getLatestProgress(wfState.Progress),
		HumanMessage:    wfState.HumanMessage,
		Cost:            wfState.Cost,
		Events:          events,
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
				description := fork.DirName
				if goalData, errGoal := os.ReadFile(filepath.Join(fork.Directory, "GOAL.md")); errGoal == nil {
					if extracted := extractGoalDescription(string(goalData)); extracted != "" {
						description = extracted
					}
				}
				forks[i] = apiForkEntry{
					Name:        fork.DirName,
					Dir:         fork.Directory,
					Running:     fork.Running,
					NeedsInput:  wfState.NeedsHumanInput(),
					InProgress:  fork.InProgress,
					Pinned:      fork.Pinned,
					CommitAhead: s.countForkCommitsAheadCached(bookmark, fork.Directory),
					Commits:     commits,
					Description: description,
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
	return strings.HasPrefix(urlPath, "/api/") || strings.HasPrefix(urlPath, "/mcp/")
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

func loadActionsForAPI(workspacePath string) []apiActionEntry {
	config, errLoad := loadProjectConfig(workspacePath)
	if errLoad != nil || config == nil || config.Actions == nil {
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

	result, errCreate := s.createWorkspaceService(req.Name)
	if errCreate != nil {
		statusCode := http.StatusInternalServerError
		switch {
		case errors.Is(errCreate, errDirectoryExists):
			statusCode = http.StatusConflict
		case errors.Is(errCreate, errWorkspaceNameInvalid):
			statusCode = http.StatusBadRequest
		}
		http.Error(w, errCreate.Error(), statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(apiCreateWorkspaceResponse(result)); err != nil {
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
	statCmd := exec.Command("jj", "diff", "--from", "default@", "--stat")
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
	diffCmd := exec.Command("jj", "diff", "--from", "default@", "--git")
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
	Description string          `json:"description"`
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

	http.Error(w, "no active session coordinator", http.StatusConflict)
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

	responseText := buildAPIResponseText(req)
	if responseText == "" {
		http.Error(w, "response cannot be empty", http.StatusBadRequest)
		return
	}

	if !coord.Respond(responseText) {
		http.Error(w, "no active question receiver", http.StatusConflict)
		return
	}

	s.notifyStateChange()

	writeJSON(w, apiRespondResponse{Success: true, Message: "response submitted"})
}

func buildAPIResponseText(req apiRespondRequest) string {
	var parts []string
	if len(req.SelectedChoices) > 0 {
		parts = append(parts, "Selected: "+strings.Join(req.SelectedChoices, ", "))
	}
	if req.Answer != "" {
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

	interactionMode := startInteractionMode(req.Auto, continuousPrompt)

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

func (s *Server) handleAPIResetWorkspace(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	result, errReset := s.resetWorkspaceService(workspacePath)
	if errReset != nil {
		if errors.Is(errReset, errWorkspaceRunning) {
			http.Error(w, errReset.Error(), http.StatusConflict)
			return
		}
		http.Error(w, errReset.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, result)
}

type apiComposeStateResponse struct {
	Workspace      string             `json:"workspace"`
	State          composerState      `json:"state"`
	Wizard         apiWizardState     `json:"wizard"`
	TechStackItems []apiTechStackItem `json:"techStackItems"`
}

type apiWizardState struct {
	CurrentStep    int      `json:"currentStep"`
	FromTemplate   string   `json:"fromTemplate,omitempty"`
	Description    string   `json:"description,omitempty"`
	TechStack      []string `json:"techStack"`
	SafetyAnalysis bool     `json:"safetyAnalysis"`
	Retrospective  bool     `json:"retrospective"`
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

	techStack := buildAPITechStackItems(wizard.TechStack)

	writeJSON(w, apiComposeStateResponse{
		Workspace:      filepath.Base(workspacePath),
		State:          currentState,
		Wizard:         apiWizardState(wizard),
		TechStackItems: techStack,
	})
}

func buildAPITechStackItems(selectedTech []string) []apiTechStackItem {
	selectedMap := make(map[string]bool)
	for _, ts := range selectedTech {
		selectedMap[ts] = true
	}

	defaults := defaultTechStackItems()
	items := make([]apiTechStackItem, len(defaults))
	for i, item := range defaults {
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
}

type apiComposeTemplatesResponse struct {
	Templates []apiComposeTemplateEntry `json:"templates"`
}

func (s *Server) handleAPIComposeTemplates(w http.ResponseWriter, _ *http.Request) {
	templates := workflowTemplates()
	entries := make([]apiComposeTemplateEntry, len(templates))
	for i, tmpl := range templates {
		entries[i] = apiComposeTemplateEntry(tmpl)
	}

	writeJSON(w, apiComposeTemplatesResponse{Templates: entries})
}

type apiComposePreviewResponse struct {
	Content string `json:"content"`
	Etag    string `json:"etag"`
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

	goalPath := filepath.Join(workspacePath, "GOAL.md")
	existingContent, errRead := os.ReadFile(goalPath)
	if errRead != nil && !os.IsNotExist(errRead) {
		http.Error(w, "failed to read current GOAL.md", http.StatusInternalServerError)
		return
	}
	etag := computeEtag(existingContent)

	writeJSON(w, apiComposePreviewResponse{
		Content: preview,
		Etag:    etag,
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
	GoalContent string `json:"goalContent"`
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

	var req apiForkRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, errFork := s.forkWorkspaceService(workspacePath, req.GoalContent)
	if errFork != nil {
		statusCode := http.StatusInternalServerError
		switch {
		case errors.Is(errFork, errForkOfFork):
			statusCode = http.StatusBadRequest
		case errors.Is(errFork, errGoalContentEmpty):
			statusCode = http.StatusBadRequest
		case errors.Is(errFork, errDirectoryExists):
			statusCode = http.StatusConflict
		}
		http.Error(w, errFork.Error(), statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(apiForkResponse(result)); err != nil {
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

	rootPath := s.resolveRootForDeleteFork(workspacePath)
	if rootPath == "" {
		http.Error(w, "workspace is not a root or fork", http.StatusBadRequest)
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

	forkDir := s.resolveForkDir(req.ForkDir, workspacePath, rootPath)
	if forkDir == "" {
		http.Error(w, "invalid fork directory", http.StatusBadRequest)
		return
	}

	if s.classifyWorkspaceCached(forkDir) != workspaceFork {
		http.Error(w, "fork workspace not found", http.StatusBadRequest)
		return
	}

	if resolveSymlinks(getRootWorkspacePath(forkDir)) != rootPath {
		http.Error(w, "fork does not belong to root", http.StatusBadRequest)
		return
	}

	forkName := filepath.Base(forkDir)

	s.stopSession(forkDir)
	s.backfillWorkspaceUsageBeforeRemoval(forkDir)

	forgetCmd := exec.Command("jj", "workspace", "forget", forkName)
	forgetCmd.Dir = rootPath
	if _, errForget := forgetCmd.CombinedOutput(); errForget != nil {
		http.Error(w, "failed to forget fork workspace", http.StatusInternalServerError)
		return
	}

	if errRemove := os.RemoveAll(forkDir); errRemove != nil {
		http.Error(w, "failed to remove fork directory", http.StatusInternalServerError)
		return
	}

	s.invalidateWorkspaceScanCache()
	s.classifyCache.delete(rootPath)
	s.classifyCache.delete(forkDir)
	s.notifyStateChange()

	writeJSON(w, apiDeleteForkResponse{
		Deleted: true,
		Message: "fork deleted successfully",
	})
}

func (s *Server) resolveRootForDeleteFork(workspacePath string) string {
	classification := s.classifyWorkspaceCached(workspacePath)
	switch classification {
	case workspaceRoot:
		return resolveSymlinks(workspacePath)
	case workspaceFork:
		return resolveSymlinks(getRootWorkspacePath(workspacePath))
	default:
		return ""
	}
}

func (s *Server) resolveForkDir(requestForkDir, workspacePath, rootPath string) string {
	if requestForkDir != "" {
		validated, errValidate := s.validateDirectory(requestForkDir)
		if errValidate != nil {
			return ""
		}
		return validated
	}
	if workspacePath != rootPath {
		return workspacePath
	}
	return ""
}

type apiDeleteWorkspaceRequest struct {
	Confirm bool `json:"confirm"`
}

type apiDeleteWorkspaceResponse struct {
	Deleted bool   `json:"deleted"`
	Message string `json:"message"`
}

func (s *Server) handleAPIDeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	var req apiDeleteWorkspaceRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if !req.Confirm {
		http.Error(w, "confirmation required to delete workspace", http.StatusBadRequest)
		return
	}

	if _, errStat := os.Stat(workspacePath); errStat != nil {
		http.Error(w, "workspace directory not found", http.StatusNotFound)
		return
	}

	kind := s.classifyWorkspaceCached(workspacePath)
	external := s.isExternalWorkspace(workspacePath)

	if external {
		switch kind {
		case workspaceFork:
			result, errDelete := s.deleteExternalForkService(workspacePath)
			if errDelete != nil {
				http.Error(w, errDelete.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, apiDeleteWorkspaceResponse(result))
		default:
			s.stopSession(workspacePath)
			result, errDetach := s.detachExternalWorkspaceService(workspacePath)
			if errDetach != nil {
				http.Error(w, errDetach.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, apiDeleteWorkspaceResponse{Deleted: result.Detached, Message: result.Message})
		}
		return
	}

	switch kind {
	case workspaceRoot:
		http.Error(w, "cannot delete a root workspace that has forks", http.StatusBadRequest)
		return
	case workspaceFork:
		result, errDelete := s.deleteForkByPathService(workspacePath)
		if errDelete != nil {
			http.Error(w, errDelete.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, apiDeleteWorkspaceResponse(result))
	default:
		result, errDelete := s.deleteWorkspaceService(workspacePath)
		if errDelete != nil {
			http.Error(w, errDelete.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, apiDeleteWorkspaceResponse(result))
	}
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

type apiForkTemplateResponse struct {
	Content string `json:"content"`
}

func (s *Server) handleAPIForkTemplate(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	if s.classifyWorkspaceCached(workspacePath) != workspaceRoot {
		http.Error(w, "workspace is not a root workspace", http.StatusBadRequest)
		return
	}

	content := s.resolveForkTemplateContent(workspacePath)
	writeJSON(w, apiForkTemplateResponse{Content: content})
}

func (s *Server) resolveForkTemplateContent(rootDir string) string {
	groups, errScan := s.scanWorkspaceGroups()
	if errScan != nil {
		return goalExampleContent
	}

	for _, grp := range groups {
		if grp.Root.Directory != rootDir {
			continue
		}
		if len(grp.Forks) == 0 {
			return goalExampleContent
		}
		content := readNewestForkGoal(grp.Forks)
		if content != "" {
			return content
		}
		return goalExampleContent
	}
	return goalExampleContent
}

func readNewestForkGoal(forks []workspaceInfo) string {
	type forkWithTime struct {
		dir     string
		modTime time.Time
	}
	candidates := make([]forkWithTime, 0, len(forks))
	for _, fork := range forks {
		goalPath := filepath.Join(fork.Directory, "GOAL.md")
		info, errStat := os.Stat(goalPath)
		if errStat != nil {
			continue
		}
		candidates = append(candidates, forkWithTime{dir: fork.Directory, modTime: info.ModTime()})
	}
	slices.SortFunc(candidates, func(a, b forkWithTime) int {
		return b.modTime.Compare(a.modTime)
	})
	for _, c := range candidates {
		data, errRead := os.ReadFile(filepath.Join(c.dir, "GOAL.md"))
		if errRead != nil {
			continue
		}
		content := string(data)
		if strings.TrimSpace(content) != "" {
			return content
		}
	}
	return ""
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

	s.notifyStateChange()

	writeJSON(w, apiUpdateGoalResponse{
		Updated:   true,
		Workspace: filepath.Base(workspacePath),
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

	result := s.adhocStatusService(workspacePath)

	writeJSON(w, apiAdhocResponse(result))
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

	result := s.adhocStartService(workspacePath, req.Prompt, req.Model)
	if result.Error != nil {
		status := http.StatusInternalServerError
		if result.BadRequest {
			status = http.StatusBadRequest
		}
		http.Error(w, result.Error.Error(), status)
		return
	}
	if result.Message == "ad-hoc prompt already running" {
		writeJSON(w, apiAdhocResponse{
			Running: result.Running,
			Output:  result.Output,
			Message: result.Message,
		})
		return
	}

	writeJSON(w, apiAdhocResponse{
		Running: result.Running,
		Message: result.Message,
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

	result := s.adhocStopService(workspacePath)

	writeJSON(w, apiAdhocResponse(result))
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
		rootDir := resolveSymlinks(getRootWorkspacePath(workspacePath))
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
	models, errModels := s.listModelsService(r.URL.Query().Get("workspace"))
	if errModels != nil {
		http.Error(w, errModels.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, apiModelsResponse(models))
}

func (s *Server) coordinatorModelFromWorkspace(workspace string) string {
	if workspace == "" {
		return ""
	}
	workspacePath := s.resolveWorkspaceNameToPath(workspace)
	if workspacePath == "" {
		return ""
	}
	model := modelFromGoal(workspacePath)
	if model == "" {
		return ""
	}
	baseModel, _ := parseModelAndVariant(model)
	return baseModel
}

type apiBrowseDirectoriesResponse struct {
	Path    string           `json:"path"`
	Entries []directoryEntry `json:"entries"`
}

func (s *Server) handleAPIBrowseDirectories(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	entries, errBrowse := browseDirectoriesService(path)
	if errBrowse != nil {
		http.Error(w, errBrowse.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, apiBrowseDirectoriesResponse{Path: path, Entries: entries})
}

type apiAttachWorkspaceRequest struct {
	Path string `json:"path"`
}

type apiAttachWorkspaceResponse struct {
	Name    string `json:"name"`
	Dir     string `json:"dir"`
	HasGoal bool   `json:"hasGoal"`
}

func (s *Server) handleAPIAttachWorkspace(w http.ResponseWriter, r *http.Request) {
	var req apiAttachWorkspaceRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, errAttach := s.attachExternalWorkspaceService(req.Path)
	if errAttach != nil {
		statusCode := http.StatusInternalServerError
		switch {
		case errors.Is(errAttach, errPathNotAbsolute):
			statusCode = http.StatusBadRequest
		case errors.Is(errAttach, errNotADirectory):
			statusCode = http.StatusBadRequest
		case errors.Is(errAttach, errUnderRootDir):
			statusCode = http.StatusBadRequest
		case errors.Is(errAttach, errAlreadyAttached):
			statusCode = http.StatusConflict
		}
		http.Error(w, errAttach.Error(), statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(apiAttachWorkspaceResponse(result)); err != nil {
		log.Println("failed to encode json response:", err)
	}
}

type apiDetachWorkspaceRequest struct {
	Path string `json:"path"`
}

type apiDetachWorkspaceResponse struct {
	Detached bool   `json:"detached"`
	Message  string `json:"message"`
}

func (s *Server) handleAPIDetachWorkspace(w http.ResponseWriter, r *http.Request) {
	var req apiDetachWorkspaceRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, errDetach := s.detachExternalWorkspaceService(req.Path)
	if errDetach != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(errDetach, errNotAttached) {
			statusCode = http.StatusNotFound
		}
		http.Error(w, errDetach.Error(), statusCode)
		return
	}

	writeJSON(w, apiDetachWorkspaceResponse(result))
}
