package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

type sseClient struct {
	events chan sseEvent
	done   chan struct{}
}

type sseEvent struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type sseBroker struct {
	mu      sync.Mutex
	clients map[*sseClient]struct{}
}

func newSSEBroker() *sseBroker {
	return &sseBroker{
		clients: make(map[*sseClient]struct{}),
	}
}

func (b *sseBroker) subscribe() *sseClient {
	c := &sseClient{
		events: make(chan sseEvent, 64),
		done:   make(chan struct{}),
	}
	b.mu.Lock()
	b.clients[c] = struct{}{}
	b.mu.Unlock()
	return c
}

func (b *sseBroker) unsubscribe(c *sseClient) {
	b.mu.Lock()
	delete(b.clients, c)
	b.mu.Unlock()
	close(c.done)
}

func (b *sseBroker) publish(evt sseEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for c := range b.clients {
		select {
		case c.events <- evt:
		default:
		}
	}
}

func (s *Server) registerAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/events/stream", s.handleSSEStream)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/events/stream", s.handleWorkspaceSSEStream)
	mux.HandleFunc("GET /api/v1/agents", s.handleAPIAgents)
	mux.HandleFunc("GET /api/v1/skills", s.handleAPISkills)
	mux.HandleFunc("GET /api/v1/skills/{name...}", s.handleAPISkillDetail)
	mux.HandleFunc("GET /api/v1/snippets", s.handleAPISnippets)
	mux.HandleFunc("GET /api/v1/snippets/{lang}", s.handleAPISnippetsByLanguage)
	mux.HandleFunc("GET /api/v1/snippets/{lang}/{fileName}", s.handleAPISnippetDetail)
	mux.HandleFunc("GET /api/v1/workspaces", s.handleAPIWorkspaces)
	mux.HandleFunc("GET /api/v1/workspaces/{name}", s.handleAPIWorkspaceDetail)
	mux.HandleFunc("POST /api/v1/workspaces", s.handleAPICreateWorkspace)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/session", s.handleAPIWorkspaceSession)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/messages", s.handleAPIWorkspaceMessages)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/todos", s.handleAPIWorkspaceTodos)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/log", s.handleAPIWorkspaceLog)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/changes", s.handleAPIWorkspaceChanges)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/events", s.handleAPIWorkspaceEvents)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/forks", s.handleAPIWorkspaceForks)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/retrospectives", s.handleAPIWorkspaceRetrospectives)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/pending-question", s.handleAPIPendingQuestion)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/respond", s.handleAPIRespond)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/start", s.handleAPIStartSession)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/stop", s.handleAPIStopSession)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/reset", s.handleAPIResetSession)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/fork", s.handleAPIForkWorkspace)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/merge", s.handleAPIMergeWorkspace)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/delete-fork", s.handleAPIDeleteFork)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/rename", s.handleAPIRenameWorkspace)
	mux.HandleFunc("PUT /api/v1/workspaces/{name}/goal", s.handleAPIUpdateGoal)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/adhoc", s.handleAPIAdhocStatus)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/adhoc", s.handleAPIAdhoc)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/retrospective/analyze", s.handleAPIRetroAnalyze)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/retrospective/apply", s.handleAPIRetroApply)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/retrospective/delete", s.handleAPIRetroDelete)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/workflow.svg", s.handleAPIWorkflowSVG)
	mux.HandleFunc("GET /api/v1/workspaces/{name}/commits", s.handleAPIWorkspaceCommits)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/steer", s.handleAPISteer)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/description", s.handleAPIUpdateDescription)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/selfdrive", s.handleAPISelfDrive)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/pin", s.handleAPITogglePin)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/open-editor", s.handleAPIOpenEditor)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/open-opencode", s.handleAPIOpenInOpenCode)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/open-editor/goal", s.handleAPIOpenEditorGoal)
	mux.HandleFunc("POST /api/v1/workspaces/{name}/open-editor/project-management", s.handleAPIOpenEditorProjectManagement)
	mux.HandleFunc("GET /api/v1/models", s.handleAPIListModels)
	mux.HandleFunc("GET /api/v1/compose", s.handleAPIComposeState)
	mux.HandleFunc("POST /api/v1/compose", s.handleAPIComposeSave)
	mux.HandleFunc("GET /api/v1/compose/templates", s.handleAPIComposeTemplates)
	mux.HandleFunc("GET /api/v1/compose/preview", s.handleAPIComposePreview)
	mux.HandleFunc("POST /api/v1/compose/draft", s.handleAPIComposeDraft)
}

func (s *Server) handleSSEStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	client := s.sseBroker.subscribe()
	defer s.sseBroker.unsubscribe(client)

	s.sendSSESnapshot(w, flusher)

	for {
		select {
		case <-r.Context().Done():
			return
		case <-client.done:
			return
		case evt := <-client.events:
			if errWrite := writeSSEEvent(w, flusher, evt); errWrite != nil {
				return
			}
		}
	}
}

func (s *Server) handleWorkspaceSSEStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	workspaceName := r.PathValue("name")
	if workspaceName == "" {
		http.Error(w, "workspace name is required", http.StatusBadRequest)
		return
	}

	workspacePath := s.resolveWorkspaceNameToPath(workspaceName)
	if workspacePath == "" {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	broker := s.workspaceBroker(workspacePath)
	client := broker.subscribe()
	defer broker.unsubscribe(client)

	s.sendWorkspaceSSESnapshot(w, flusher, workspacePath)

	for {
		select {
		case <-r.Context().Done():
			return
		case <-client.done:
			return
		case evt := <-client.events:
			if errWrite := writeSSEEvent(w, flusher, evt); errWrite != nil {
				return
			}
		}
	}
}

func (s *Server) sendWorkspaceSSESnapshot(w http.ResponseWriter, flusher http.Flusher, workspacePath string) {
	snapshot := s.buildWorkspaceSSESnapshot(workspacePath)
	evt := sseEvent{Type: "snapshot", Data: snapshot}
	if errWrite := writeSSEEvent(w, flusher, evt); errWrite != nil {
		log.Println("failed to send workspace snapshot:", errWrite)
	}
}

type sseWorkspaceDetailSnapshot struct {
	Name       string `json:"name"`
	Running    bool   `json:"running"`
	NeedsInput bool   `json:"needsInput"`
	Status     string `json:"status"`
}

func (s *Server) buildWorkspaceSSESnapshot(workspacePath string) sseWorkspaceDetailSnapshot {
	running, needsInput := s.getWorkspaceStatus(workspacePath)
	wfState, _ := state.Load(statePath(workspacePath))
	status := wfState.Status
	if status == "" {
		status = "-"
	}
	return sseWorkspaceDetailSnapshot{
		Name:       filepath.Base(workspacePath),
		Running:    running,
		NeedsInput: needsInput,
		Status:     status,
	}
}

func (s *Server) sendSSESnapshot(w http.ResponseWriter, flusher http.Flusher) {
	snapshot := s.buildSSESnapshot()
	evt := sseEvent{Type: "snapshot", Data: snapshot}
	if errWrite := writeSSEEvent(w, flusher, evt); errWrite != nil {
		log.Println("failed to send snapshot:", errWrite)
	}
}

type sseSnapshot struct {
	Workspaces []sseWorkspaceSnapshot `json:"workspaces"`
}

type sseWorkspaceSnapshot struct {
	Name    string `json:"name"`
	Dir     string `json:"dir"`
	Running bool   `json:"running"`
	Status  string `json:"status"`
}

func (s *Server) buildSSESnapshot() sseSnapshot {
	groups, errScan := s.scanWorkspaceGroups()
	if errScan != nil {
		return sseSnapshot{}
	}

	var workspaces []sseWorkspaceSnapshot
	for _, grp := range groups {
		workspaces = append(workspaces, sseWorkspaceSnapshot{
			Name:    grp.Root.DirName,
			Dir:     grp.Root.Directory,
			Running: grp.Root.Running,
			Status:  workspaceStatusText(grp.Root),
		})
		for _, fork := range grp.Forks {
			workspaces = append(workspaces, sseWorkspaceSnapshot{
				Name:    fork.DirName,
				Dir:     fork.Directory,
				Running: fork.Running,
				Status:  workspaceStatusText(fork),
			})
		}
	}

	return sseSnapshot{Workspaces: workspaces}
}

func workspaceStatusText(w workspaceInfo) string {
	wfState, _ := state.Load(statePath(w.Directory))
	class, _ := badgeStatus(wfState, w.Running)
	return class
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, evt sseEvent) error {
	data, errMarshal := json.Marshal(evt.Data)
	if errMarshal != nil {
		return errMarshal
	}
	_, errWrite := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, data)
	if errWrite != nil {
		return errWrite
	}
	flusher.Flush()
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

type apiWorkspaceEntry struct {
	Name       string              `json:"name"`
	Dir        string              `json:"dir"`
	Running    bool                `json:"running"`
	NeedsInput bool                `json:"needsInput"`
	InProgress bool                `json:"inProgress"`
	Pinned     bool                `json:"pinned"`
	IsRoot     bool                `json:"isRoot"`
	Status     string              `json:"status"`
	HasSGAI    bool                `json:"hasSgai"`
	Forks      []apiWorkspaceEntry `json:"forks,omitempty"`
}

type apiWorkspacesResponse struct {
	Workspaces []apiWorkspaceEntry `json:"workspaces"`
}

func (s *Server) handleAPIWorkspaces(w http.ResponseWriter, _ *http.Request) {
	groups, errScan := s.scanWorkspaceGroups()
	if errScan != nil {
		http.Error(w, "failed to scan workspaces", http.StatusInternalServerError)
		return
	}

	workspaces := convertWorkspaceGroups(groups)
	writeJSON(w, apiWorkspacesResponse{Workspaces: workspaces})
}

func convertWorkspaceGroups(groups []workspaceGroup) []apiWorkspaceEntry {
	result := make([]apiWorkspaceEntry, 0, len(groups))
	for _, grp := range groups {
		entry := convertWorkspaceInfo(grp.Root)
		if len(grp.Forks) > 0 {
			entry.Forks = make([]apiWorkspaceEntry, 0, len(grp.Forks))
			for _, fork := range grp.Forks {
				entry.Forks = append(entry.Forks, convertWorkspaceInfo(fork))
			}
		}
		result = append(result, entry)
	}
	return result
}

func convertWorkspaceInfo(w workspaceInfo) apiWorkspaceEntry {
	wfState, _ := state.Load(statePath(w.Directory))
	_, statusText := badgeStatus(wfState, w.Running)
	return apiWorkspaceEntry{
		Name:       w.DirName,
		Dir:        w.Directory,
		Running:    w.Running,
		NeedsInput: w.NeedsInput,
		InProgress: w.InProgress,
		Pinned:     w.Pinned,
		IsRoot:     w.IsRoot,
		Status:     statusText,
		HasSGAI:    w.HasWorkspace,
	}
}

type apiWorkspaceDetailResponse struct {
	Name            string                    `json:"name"`
	Dir             string                    `json:"dir"`
	Running         bool                      `json:"running"`
	NeedsInput      bool                      `json:"needsInput"`
	Status          string                    `json:"status"`
	BadgeClass      string                    `json:"badgeClass"`
	BadgeText       string                    `json:"badgeText"`
	IsRoot          bool                      `json:"isRoot"`
	IsFork          bool                      `json:"isFork"`
	Pinned          bool                      `json:"pinned"`
	HasSGAI         bool                      `json:"hasSgai"`
	HasEditedGoal   bool                      `json:"hasEditedGoal"`
	InteractiveAuto bool                      `json:"interactiveAuto"`
	ContinuousMode  bool                      `json:"continuousMode"`
	CurrentAgent    string                    `json:"currentAgent"`
	CurrentModel    string                    `json:"currentModel"`
	Task            string                    `json:"task"`
	GoalContent     string                    `json:"goalContent"`
	RawGoalContent  string                    `json:"rawGoalContent"`
	FullGoalContent string                    `json:"fullGoalContent"`
	PMContent       string                    `json:"pmContent"`
	HasProjectMgmt  bool                      `json:"hasProjectMgmt"`
	SVGHash         string                    `json:"svgHash"`
	TotalExecTime   string                    `json:"totalExecTime"`
	LatestProgress  string                    `json:"latestProgress"`
	AgentSequence   []apiAgentSequenceEntry   `json:"agentSequence"`
	Cost            state.SessionCost         `json:"cost"`
	Forks           []apiWorkspaceForkSummary `json:"forks,omitempty"`
}

type apiAgentSequenceEntry struct {
	Agent       string `json:"agent"`
	ElapsedTime string `json:"elapsedTime"`
	IsCurrent   bool   `json:"isCurrent"`
}

type apiWorkspaceForkSummary struct {
	Name        string `json:"name"`
	Dir         string `json:"dir"`
	Running     bool   `json:"running"`
	CommitAhead int    `json:"commitAhead"`
}

func (s *Server) handleAPIWorkspaceDetail(w http.ResponseWriter, r *http.Request) {
	workspaceName := r.PathValue("name")
	if workspaceName == "" {
		http.Error(w, "workspace name is required", http.StatusBadRequest)
		return
	}

	workspacePath := s.resolveWorkspaceNameToPath(workspaceName)
	if workspacePath == "" {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	detail := s.buildWorkspaceDetail(workspacePath)
	writeJSON(w, detail)
}

func (s *Server) buildWorkspaceDetail(workspacePath string) apiWorkspaceDetailResponse {
	wfState, _ := state.Load(statePath(workspacePath))

	interactiveAuto := wfState.InteractiveAutoLock
	var running bool
	s.mu.Lock()
	sess := s.sessions[workspacePath]
	s.mu.Unlock()
	if sess != nil {
		sess.mu.Lock()
		running = sess.running
		interactiveAuto = interactiveAuto || sess.interactiveAuto
		sess.mu.Unlock()
	}

	badgeClass, badgeText := badgeStatus(wfState, running)
	needsInput := wfState.NeedsHumanInput()
	kind := classifyWorkspace(workspacePath)

	currentAgent := wfState.CurrentAgent
	if currentAgent == "" {
		currentAgent = "Unknown"
	}

	task := wfState.Task
	status := wfState.Status
	if status == "" {
		status = "-"
	}

	goalContent, rawGoalContent, fullGoalContent, pmContent, hasProjectMgmt := readGoalAndPMForAPI(workspacePath)

	hasEditedGoal := false
	if data, errRead := os.ReadFile(filepath.Join(workspacePath, "GOAL.md")); errRead == nil {
		body := extractBody(data)
		hasEditedGoal = len(strings.TrimSpace(string(body))) > 0
	}

	agentSeq := convertAgentSequence(
		prepareAgentSequenceDisplay(wfState.AgentSequence, running, getLastActivityTime(wfState.Progress)),
	)

	totalExecTime := calculateTotalExecutionTime(wfState.AgentSequence, running, getLastActivityTime(wfState.Progress))

	detail := apiWorkspaceDetailResponse{
		Name:            filepath.Base(workspacePath),
		Dir:             workspacePath,
		Running:         running,
		NeedsInput:      needsInput,
		Status:          status,
		BadgeClass:      badgeClass,
		BadgeText:       badgeText,
		IsRoot:          kind == workspaceRoot,
		IsFork:          kind == workspaceFork,
		Pinned:          s.isPinned(workspacePath),
		HasSGAI:         hassgaiDirectory(workspacePath),
		HasEditedGoal:   hasEditedGoal,
		InteractiveAuto: interactiveAuto,
		ContinuousMode:  readContinuousModePrompt(workspacePath) != "",
		CurrentAgent:    currentAgent,
		CurrentModel:    resolveCurrentModel(workspacePath, wfState),
		Task:            task,
		GoalContent:     goalContent,
		RawGoalContent:  rawGoalContent,
		FullGoalContent: fullGoalContent,
		PMContent:       pmContent,
		HasProjectMgmt:  hasProjectMgmt,
		SVGHash:         getWorkflowSVGHash(workspacePath, currentAgent),
		TotalExecTime:   totalExecTime,
		LatestProgress:  getLatestProgress(wfState.Progress),
		AgentSequence:   agentSeq,
		Cost:            wfState.Cost,
	}

	if kind == workspaceRoot {
		detail.Forks = s.collectForkSummaries(workspacePath)
	}

	return detail
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

func (s *Server) collectForkSummaries(rootDir string) []apiWorkspaceForkSummary {
	groups, errScan := s.scanWorkspaceGroups()
	if errScan != nil {
		return nil
	}

	for _, grp := range groups {
		if grp.Root.Directory != rootDir {
			continue
		}
		bookmark := resolveBaseBookmark(rootDir)
		summaries := make([]apiWorkspaceForkSummary, 0, len(grp.Forks))
		for _, fork := range grp.Forks {
			summaries = append(summaries, apiWorkspaceForkSummary{
				Name:        fork.DirName,
				Dir:         fork.Directory,
				Running:     fork.Running,
				CommitAhead: countForkCommitsAhead(bookmark, fork.Directory),
			})
		}
		return summaries
	}

	return nil
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

type apiSessionResponse struct {
	Name            string                  `json:"name"`
	Status          string                  `json:"status"`
	Running         bool                    `json:"running"`
	NeedsInput      bool                    `json:"needsInput"`
	InteractiveAuto bool                    `json:"interactiveAuto"`
	BadgeClass      string                  `json:"badgeClass"`
	BadgeText       string                  `json:"badgeText"`
	CurrentAgent    string                  `json:"currentAgent"`
	CurrentModel    string                  `json:"currentModel"`
	Task            string                  `json:"task"`
	HumanMessage    string                  `json:"humanMessage"`
	LatestProgress  string                  `json:"latestProgress"`
	TotalExecTime   string                  `json:"totalExecTime"`
	SVGHash         string                  `json:"svgHash"`
	AgentSequence   []apiAgentSequenceEntry `json:"agentSequence"`
	Cost            state.SessionCost       `json:"cost"`
	ModelStatuses   []apiModelStatusEntry   `json:"modelStatuses,omitempty"`
}

type apiModelStatusEntry struct {
	ModelID string `json:"modelId"`
	Status  string `json:"status"`
}

func (s *Server) handleAPIWorkspaceSession(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	wfState, _ := state.Load(statePath(workspacePath))

	interactiveAuto := wfState.InteractiveAutoLock
	var running bool
	s.mu.Lock()
	sess := s.sessions[workspacePath]
	s.mu.Unlock()
	if sess != nil {
		sess.mu.Lock()
		running = sess.running
		interactiveAuto = interactiveAuto || sess.interactiveAuto
		sess.mu.Unlock()
	}

	badgeClass, badgeText := badgeStatus(wfState, running)
	currentAgent := wfState.CurrentAgent
	if currentAgent == "" {
		currentAgent = "Unknown"
	}

	status := wfState.Status
	if status == "" {
		status = "-"
	}

	agentSeq := convertAgentSequence(
		prepareAgentSequenceDisplay(wfState.AgentSequence, running, getLastActivityTime(wfState.Progress)),
	)

	modelStatuses := convertModelStatuses(orderedModelStatuses(workspacePath, wfState.ModelStatuses))

	writeJSON(w, apiSessionResponse{
		Name:            filepath.Base(workspacePath),
		Status:          status,
		Running:         running,
		NeedsInput:      wfState.NeedsHumanInput(),
		InteractiveAuto: interactiveAuto,
		BadgeClass:      badgeClass,
		BadgeText:       badgeText,
		CurrentAgent:    currentAgent,
		CurrentModel:    resolveCurrentModel(workspacePath, wfState),
		Task:            wfState.Task,
		HumanMessage:    wfState.HumanMessage,
		LatestProgress:  getLatestProgress(wfState.Progress),
		TotalExecTime:   calculateTotalExecutionTime(wfState.AgentSequence, running, getLastActivityTime(wfState.Progress)),
		SVGHash:         getWorkflowSVGHash(workspacePath, currentAgent),
		AgentSequence:   agentSeq,
		Cost:            wfState.Cost,
		ModelStatuses:   modelStatuses,
	})
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

type apiMessagesResponse struct {
	Messages []apiMessageEntry `json:"messages"`
}

func (s *Server) handleAPIWorkspaceMessages(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	wfState, _ := state.Load(statePath(workspacePath))
	messages := convertMessagesForAPI(wfState.Messages)
	writeJSON(w, apiMessagesResponse{Messages: messages})
}

func convertMessagesForAPI(messages []state.Message) []apiMessageEntry {
	result := make([]apiMessageEntry, 0, len(messages))
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		subject := extractSubject(msg.Body)
		result = append(result, apiMessageEntry{
			ID:        msg.ID,
			FromAgent: msg.FromAgent,
			ToAgent:   msg.ToAgent,
			Body:      msg.Body,
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

type apiTodosResponse struct {
	ProjectTodos []apiTodoEntry `json:"projectTodos"`
	AgentTodos   []apiTodoEntry `json:"agentTodos"`
	CurrentAgent string         `json:"currentAgent"`
}

func (s *Server) handleAPIWorkspaceTodos(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	wfState, _ := state.Load(statePath(workspacePath))
	currentAgent := wfState.CurrentAgent
	if currentAgent == "" {
		currentAgent = "Unknown"
	}

	writeJSON(w, apiTodosResponse{
		ProjectTodos: convertTodosForAPI(wfState.ProjectTodos),
		AgentTodos:   convertTodosForAPI(wfState.Todos),
		CurrentAgent: currentAgent,
	})
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

type apiLogResponse struct {
	Lines []apiLogEntry `json:"lines"`
}

func (s *Server) handleAPIWorkspaceLog(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	var lines []apiLogEntry

	s.mu.Lock()
	sess := s.sessions[workspacePath]
	s.mu.Unlock()

	if sess != nil && sess.outputLog != nil {
		logLines := sess.outputLog.lines()
		for _, line := range logLines {
			lines = append(lines, apiLogEntry{Prefix: line.prefix, Text: line.text})
		}
	}

	maxLines := parseOptionalIntParam(r, "lines", 0)
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	writeJSON(w, apiLogResponse{Lines: lines})
}

func parseOptionalIntParam(r *http.Request, name string, defaultVal int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultVal
	}
	parsed, errParse := strconv.Atoi(val)
	if errParse != nil || parsed < 0 {
		return defaultVal
	}
	return parsed
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

func (s *Server) handleAPIWorkspaceChanges(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	diffOutput, description := collectJJChanges(workspacePath)

	writeJSON(w, apiChangesResponse{
		Description: description,
		DiffLines:   diffOutput,
	})
}

func collectJJChanges(dir string) ([]apiDiffLine, string) {
	diffCmd := exec.Command("jj", "diff", "--git")
	diffCmd.Dir = dir
	rawDiff, errDiff := diffCmd.Output()
	if errDiff != nil {
		return nil, ""
	}

	descCmd := exec.Command("jj", "log", "--no-graph", "-T", "description", "-r", "@")
	descCmd.Dir = dir
	rawDesc, errDesc := descCmd.Output()
	if errDesc != nil {
		rawDesc = nil
	}

	var diffLines []apiDiffLine
	for line := range strings.SplitSeq(string(rawDiff), "\n") {
		if line == "" && len(diffLines) == 0 {
			continue
		}
		diffLines = append(diffLines, apiDiffLine{
			LineNumber: len(diffLines) + 1,
			Text:       line,
			Class:      classifyDiffLine(line),
		})
	}

	return diffLines, strings.TrimSpace(string(rawDesc))
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

type apiEventsResponse struct {
	Events        []apiEventEntry       `json:"events"`
	CurrentAgent  string                `json:"currentAgent"`
	CurrentModel  string                `json:"currentModel"`
	SVGHash       string                `json:"svgHash"`
	NeedsInput    bool                  `json:"needsInput"`
	HumanMessage  string                `json:"humanMessage"`
	GoalContent   string                `json:"goalContent"`
	ModelStatuses []apiModelStatusEntry `json:"modelStatuses,omitempty"`
	AgentModels   []apiAgentModelEntry  `json:"agentModels,omitempty"`
}

func (s *Server) handleAPIWorkspaceEvents(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	wfState, _ := state.Load(statePath(workspacePath))

	reversedProgress := slices.Clone(wfState.Progress)
	slices.Reverse(reversedProgress)

	events := convertEventsForAPI(formatProgressForDisplay(reversedProgress))

	currentAgent := wfState.CurrentAgent
	if currentAgent == "" {
		currentAgent = "Unknown"
	}

	goalContent, _, _, _, _ := readGoalAndPMForAPI(workspacePath)

	writeJSON(w, apiEventsResponse{
		Events:        events,
		CurrentAgent:  currentAgent,
		CurrentModel:  resolveCurrentModel(workspacePath, wfState),
		SVGHash:       getWorkflowSVGHash(workspacePath, currentAgent),
		NeedsInput:    wfState.NeedsHumanInput(),
		HumanMessage:  wfState.HumanMessage,
		GoalContent:   goalContent,
		ModelStatuses: convertModelStatuses(orderedModelStatuses(workspacePath, wfState.ModelStatuses)),
		AgentModels:   collectAgentModels(workspacePath),
	})
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
	CommitAhead int             `json:"commitAhead"`
	Commits     []apiForkCommit `json:"commits"`
}

type apiForksResponse struct {
	Forks []apiForkEntry `json:"forks"`
}

func (s *Server) handleAPIWorkspaceForks(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	forks := s.collectForksForAPI(workspacePath)
	writeJSON(w, apiForksResponse{Forks: forks})
}

func (s *Server) collectForksForAPI(rootDir string) []apiForkEntry {
	groups, errScan := s.scanWorkspaceGroups()
	if errScan != nil {
		return nil
	}

	for _, grp := range groups {
		if grp.Root.Directory != rootDir {
			continue
		}
		bookmark := resolveBaseBookmark(rootDir)
		forks := make([]apiForkEntry, 0, len(grp.Forks))
		for _, fork := range grp.Forks {
			commits := convertJJCommitsForAPI(runJJLogForFork(bookmark, fork.Directory))
			forks = append(forks, apiForkEntry{
				Name:        fork.DirName,
				Dir:         fork.Directory,
				Running:     fork.Running,
				CommitAhead: countForkCommitsAhead(bookmark, fork.Directory),
				Commits:     commits,
			})
		}
		return forks
	}

	return nil
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

type apiRetroSession struct {
	Name            string `json:"name"`
	HasImprovements bool   `json:"hasImprovements"`
	GoalSummary     string `json:"goalSummary"`
}

type apiRetroDetail struct {
	SessionName     string `json:"sessionName"`
	GoalSummary     string `json:"goalSummary"`
	GoalContent     string `json:"goalContent"`
	Improvements    string `json:"improvements"`
	ImprovementsRaw string `json:"improvementsRaw"`
	HasImprovements bool   `json:"hasImprovements"`
	IsAnalyzing     bool   `json:"isAnalyzing"`
	IsApplying      bool   `json:"isApplying"`
}

type apiRetrospectivesResponse struct {
	Sessions        []apiRetroSession `json:"sessions"`
	SelectedSession string            `json:"selectedSession"`
	Details         *apiRetroDetail   `json:"details,omitempty"`
}

func (s *Server) handleAPIWorkspaceRetrospectives(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	sessions := convertRetroSessionsForAPI(s.listRetrospectiveSessionsForProject(workspacePath))

	sessionParam := r.URL.Query().Get("session")
	if sessionParam == "" && len(sessions) > 0 {
		sessionParam = sessions[0].Name
	}

	var detail *apiRetroDetail
	if sessionParam != "" {
		detail = s.buildRetroDetailForAPI(workspacePath, sessionParam)
	}

	writeJSON(w, apiRetrospectivesResponse{
		Sessions:        sessions,
		SelectedSession: sessionParam,
		Details:         detail,
	})
}

func convertRetroSessionsForAPI(sessions []retroSessionData) []apiRetroSession {
	result := make([]apiRetroSession, 0, len(sessions))
	for _, s := range sessions {
		result = append(result, apiRetroSession(s))
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

func (s *Server) handleAPIPendingQuestion(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	wfState, _ := state.Load(statePath(workspacePath))

	if !wfState.NeedsHumanInput() {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	agentName := wfState.CurrentAgent
	if agentName == "" {
		agentName = "Unknown"
	}

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

	writeJSON(w, apiPendingQuestionResponse{
		QuestionID: generateQuestionID(wfState),
		Type:       questionType(wfState),
		AgentName:  agentName,
		Message:    wfState.HumanMessage,
		Questions:  questions,
	})
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

	wfState, errLoad := state.Load(statePath(workspacePath))
	if errLoad != nil {
		http.Error(w, "failed to load workspace state", http.StatusInternalServerError)
		return
	}

	if !wfState.NeedsHumanInput() {
		writeJSON(w, apiRespondResponse{Success: true, Message: "no pending question"})
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

	responsePath := filepath.Join(workspacePath, ".sgai", "response.txt")
	if errWrite := os.WriteFile(responsePath, []byte(responseText), 0644); errWrite != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
		return
	}

	wfState.Status = state.StatusWorking
	wfState.HumanMessage = ""
	wfState.MultiChoiceQuestion = nil
	wfState.Task = ""
	if errSave := state.Save(statePath(workspacePath), wfState); errSave != nil {
		http.Error(w, "failed to save state", http.StatusInternalServerError)
		return
	}

	s.publishGlobalAndWorkspace(filepath.Base(workspacePath), workspacePath, sseEvent{Type: "session:update", Data: map[string]string{
		"workspace": filepath.Base(workspacePath),
	}})

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

	if classifyWorkspace(workspacePath) == workspaceRoot {
		http.Error(w, "root workspace cannot start agentic work", http.StatusBadRequest)
		return
	}

	var req apiStartSessionRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	wfState, errLoadState := state.Load(statePath(workspacePath))
	if errLoadState != nil && !os.IsNotExist(errLoadState) {
		http.Error(w, "failed to load workflow state", http.StatusInternalServerError)
		return
	}
	req.Auto = wfState.InteractiveAutoLock || req.Auto

	result := s.startSession(workspacePath, req.Auto)

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

	s.publishGlobalAndWorkspace(filepath.Base(workspacePath), workspacePath, sseEvent{Type: "session:update", Data: map[string]string{
		"workspace": filepath.Base(workspacePath),
	}})

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

	s.publishGlobalAndWorkspace(filepath.Base(workspacePath), workspacePath, sseEvent{Type: "session:update", Data: map[string]string{
		"workspace": filepath.Base(workspacePath),
	}})

	writeJSON(w, apiSessionActionResponse{
		Name:    filepath.Base(workspacePath),
		Status:  "stopped",
		Running: false,
		Message: message,
	})
}

func (s *Server) handleAPIResetSession(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	if errRemove := os.Remove(statePath(workspacePath)); errRemove != nil && !os.IsNotExist(errRemove) {
		http.Error(w, "failed to reset state", http.StatusInternalServerError)
		return
	}

	s.publishGlobalAndWorkspace(filepath.Base(workspacePath), workspacePath, sseEvent{Type: "session:update", Data: map[string]string{
		"workspace": filepath.Base(workspacePath),
	}})

	writeJSON(w, apiSessionActionResponse{
		Name:    filepath.Base(workspacePath),
		Status:  "reset",
		Running: false,
		Message: "session state reset",
	})
}

func (s *Server) buildRetroDetailForAPI(dir, sessionName string) *apiRetroDetail {
	retrospectivesDir := filepath.Join(dir, ".sgai", "retrospectives")
	sessionDir := filepath.Join(retrospectivesDir, sessionName)

	goalPath := filepath.Join(sessionDir, "GOAL.md")
	improvementsPath := filepath.Join(sessionDir, "IMPROVEMENTS.md")

	goalSummary := stripMarkdownHeading(extractGoalSummary(goalPath))

	var goalContent string
	if data, errRead := os.ReadFile(goalPath); errRead == nil {
		normalized := normalizeEscapedNewlines(data)
		stripped := stripFrontmatter(string(normalized))
		if rendered, errRender := renderMarkdown([]byte(stripped)); errRender == nil {
			goalContent = rendered
		}
	}

	var improvementsContent string
	var improvementsRaw string
	hasImprovements := false
	if data, errRead := os.ReadFile(improvementsPath); errRead == nil {
		hasImprovements = true
		stripped := stripFrontmatter(string(data))
		improvementsRaw = stripped
		if rendered, errRender := renderMarkdown([]byte(stripped)); errRender == nil {
			improvementsContent = rendered
		}
	}

	sessionKey := "retro-analyze-" + dir + "-" + sessionName
	applyKey := "retro-apply-" + dir + "-" + sessionName

	s.mu.Lock()
	analyzeSession := s.sessions[sessionKey]
	applySession := s.sessions[applyKey]
	s.mu.Unlock()

	var isAnalyzing, isApplying bool
	if analyzeSession != nil {
		analyzeSession.mu.Lock()
		isAnalyzing = analyzeSession.running
		analyzeSession.mu.Unlock()
	}
	if applySession != nil {
		applySession.mu.Lock()
		isApplying = applySession.running
		applySession.mu.Unlock()
	}

	return &apiRetroDetail{
		SessionName:     sessionName,
		GoalSummary:     goalSummary,
		GoalContent:     goalContent,
		Improvements:    improvementsContent,
		ImprovementsRaw: improvementsRaw,
		HasImprovements: hasImprovements,
		IsAnalyzing:     isAnalyzing,
		IsApplying:      isApplying,
	}
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

	cs := getComposerSession(workspacePath)
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

	cs := getComposerSession(workspacePath)
	cs.mu.Lock()
	currentState := cs.state
	cs.mu.Unlock()

	goalContent := buildGOALContent(currentState)

	if errWrite := os.WriteFile(goalPath, []byte(goalContent), 0644); errWrite != nil {
		http.Error(w, "failed to save GOAL.md: "+errWrite.Error(), http.StatusInternalServerError)
		return
	}

	s.publishToWorkspace(workspacePath, sseEvent{Type: "compose:update", Data: map[string]string{
		"workspace": filepath.Base(workspacePath),
		"action":    "saved",
	}})

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

	cs := getComposerSession(workspacePath)
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

	cs := getComposerSession(workspacePath)
	cs.mu.Lock()
	cs.state = req.State
	cs.wizard = wizardState(req.Wizard)
	cs.mu.Unlock()

	s.publishToWorkspace(workspacePath, sseEvent{Type: "compose:update", Data: map[string]string{
		"workspace": filepath.Base(workspacePath),
		"action":    "draft-saved",
	}})

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

	if classifyWorkspace(workspacePath) == workspaceFork {
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

	s.sseBroker.publish(sseEvent{Type: "workspace:update", Data: map[string]string{
		"workspace": filepath.Base(workspacePath),
		"action":    "forked",
		"fork":      name,
	}})

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

type apiMergeRequest struct {
	ForkDir string `json:"forkDir"`
	Confirm bool   `json:"confirm"`
}

type apiMergeResponse struct {
	Merged  bool   `json:"merged"`
	Message string `json:"message"`
}

func (s *Server) handleAPIMergeWorkspace(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	if classifyWorkspace(workspacePath) != workspaceRoot {
		http.Error(w, "workspace is not a root", http.StatusBadRequest)
		return
	}

	var req apiMergeRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if !req.Confirm {
		http.Error(w, "confirmation required to merge and delete fork", http.StatusBadRequest)
		return
	}

	forkDir, errValidate := s.validateDirectory(req.ForkDir)
	if errValidate != nil {
		http.Error(w, "invalid fork directory", http.StatusBadRequest)
		return
	}

	if classifyWorkspace(forkDir) != workspaceFork {
		http.Error(w, "fork workspace not found", http.StatusBadRequest)
		return
	}

	if getRootWorkspacePath(forkDir) != workspacePath {
		http.Error(w, "fork does not belong to root", http.StatusBadRequest)
		return
	}

	diffCmd := exec.Command("jj", "diff", "--stat")
	diffCmd.Dir = forkDir
	diffOutput, errDiff := diffCmd.CombinedOutput()
	if errDiff != nil {
		http.Error(w, "failed to collect diff summary", http.StatusInternalServerError)
		return
	}
	if strings.TrimSpace(string(diffOutput)) != "" {
		http.Error(w, "fork has uncommitted changes", http.StatusConflict)
		return
	}

	logCmd := exec.Command("jj", "log", "-r", "::@", "--no-graph")
	logCmd.Dir = forkDir
	logOutput, errLog := logCmd.CombinedOutput()
	if errLog != nil {
		http.Error(w, "failed to collect commit history", http.StatusInternalServerError)
		return
	}

	goalContent := ""
	if data, errRead := os.ReadFile(filepath.Join(forkDir, "GOAL.md")); errRead == nil {
		goalContent = string(data)
	}

	forkName := filepath.Base(forkDir)
	branchName := normalizeForkName(forkName)
	if branchName == "" {
		branchName = "fork-merge"
	}
	prompt := strings.Join([]string{
		"Return a short branch name based on the fork work.",
		"Use only letters, numbers, dashes, and underscores.",
		"Fork name: " + forkName,
		"Commit history:\n" + string(logOutput),
		"Diff summary:\n" + string(diffOutput),
		"GOAL:\n" + goalContent,
	}, "\n")
	if result, errPrompt := invokeLLMForAssist(prompt); errPrompt == nil {
		normalized := normalizeForkName(result)
		if validateWorkspaceName(normalized) == "" && normalized != "" {
			branchName = normalized
		}
	}

	baseBookmark := "main"
	for _, candidate := range []string{"main", "trunk"} {
		baseCmd := exec.Command("jj", "log", "-r", candidate, "--no-graph", "-T", "change_id")
		baseCmd.Dir = workspacePath
		if errBase := baseCmd.Run(); errBase == nil {
			baseBookmark = candidate
			break
		}
	}

	forkRevset := fmt.Sprintf("ancestors(@) ~ ancestors(%s@)", baseBookmark)
	rebaseCmd := exec.Command("jj", "rebase", "-s", forkRevset, "-d", baseBookmark)
	rebaseCmd.Dir = forkDir
	rebaseOutput, errRebase := rebaseCmd.CombinedOutput()
	if errRebase != nil {
		if !strings.Contains(strings.ToLower(string(rebaseOutput)), "conflict") {
			http.Error(w, "failed to rebase fork", http.StatusInternalServerError)
			return
		}
		listCmd := exec.Command("jj", "resolve", "--list")
		listCmd.Dir = forkDir
		listOutput, errList := listCmd.CombinedOutput()
		if errList != nil {
			http.Error(w, "failed to list conflicts", http.StatusInternalServerError)
			return
		}
		conflictFiles := strings.Fields(string(listOutput))
		if len(conflictFiles) == 0 {
			http.Error(w, "failed to detect conflicts", http.StatusInternalServerError)
			return
		}
		for _, file := range conflictFiles {
			targetPath := file
			if !filepath.IsAbs(targetPath) {
				targetPath = filepath.Join(forkDir, targetPath)
			}
			content, errRead := os.ReadFile(targetPath)
			if errRead != nil {
				http.Error(w, "failed to read conflict file", http.StatusInternalServerError)
				return
			}
			prompt := strings.Join([]string{
				"Resolve the merge conflicts in the following file.",
				"Return only the resolved file content.",
				"File: " + file,
				"Content:\n" + string(content),
			}, "\n")
			resolved, errResolve := invokeLLMForAssist(prompt)
			if errResolve != nil || strings.TrimSpace(resolved) == "" {
				http.Error(w, "failed to resolve conflicts", http.StatusInternalServerError)
				return
			}
			if errWrite := os.WriteFile(targetPath, []byte(resolved), 0644); errWrite != nil {
				http.Error(w, "failed to write resolved file", http.StatusInternalServerError)
				return
			}
		}
		resolveCmd := exec.Command("jj", "resolve", "--accept=working")
		resolveCmd.Dir = forkDir
		if _, errResolve := resolveCmd.CombinedOutput(); errResolve != nil {
			http.Error(w, "failed to record conflict resolution", http.StatusInternalServerError)
			return
		}
	}

	squashFrom := fmt.Sprintf("(%s) ~ @", forkRevset)
	squashCandidatesCmd := exec.Command("jj", "log", "-r", squashFrom, "--no-graph", "-T", "change_id ++ \"\\n\"")
	squashCandidatesCmd.Dir = forkDir
	squashCandidatesOutput, errCandidates := squashCandidatesCmd.CombinedOutput()
	if errCandidates != nil {
		http.Error(w, "failed to identify squash candidates", http.StatusInternalServerError)
		return
	}
	if strings.TrimSpace(string(squashCandidatesOutput)) != "" {
		squashCmd := exec.Command("jj", "squash", "--from", squashFrom, "--into", "@")
		squashCmd.Dir = forkDir
		if _, errSquash := squashCmd.CombinedOutput(); errSquash != nil {
			http.Error(w, "failed to squash fork changes", http.StatusInternalServerError)
			return
		}
	}

	commitMessage := ""
	defaultDescCmd := exec.Command("jj", "log", "-r", "::@", "--no-graph", "-T", "description.first_line() ++ \"\\n\"")
	defaultDescCmd.Dir = forkDir
	defaultDescOutput, errDefaultDesc := defaultDescCmd.CombinedOutput()
	if errDefaultDesc == nil {
		var parts []string
		for line := range strings.SplitSeq(string(defaultDescOutput), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				parts = append(parts, trimmed)
			}
		}
		if len(parts) > 0 {
			commitMessage = strings.Join(parts, "; ")
		}
	}
	if commitMessage == "" {
		commitMessage = "merge fork work"
	}
	commitPrompt := strings.Join([]string{
		"Write a concise commit message summarizing the fork work.",
		"Commit history:\n" + string(logOutput),
		"Diff summary:\n" + string(diffOutput),
		"GOAL:\n" + goalContent,
	}, "\n")
	if result, errPrompt := invokeLLMForAssist(commitPrompt); errPrompt == nil && strings.TrimSpace(result) != "" {
		commitMessage = strings.TrimSpace(result)
	}

	descCmd := exec.Command("jj", "desc", "-m", commitMessage)
	descCmd.Dir = forkDir
	if _, errDesc := descCmd.CombinedOutput(); errDesc != nil {
		http.Error(w, "failed to update commit message", http.StatusInternalServerError)
		return
	}

	bookmarkCmd := exec.Command("jj", "bookmark", "create", branchName, "-r", "@")
	bookmarkCmd.Dir = forkDir
	if _, errBookmark := bookmarkCmd.CombinedOutput(); errBookmark != nil {
		http.Error(w, "failed to create bookmark", http.StatusInternalServerError)
		return
	}

	pushCmd := exec.Command("jj", "git", "push", "--bookmark", branchName)
	pushCmd.Dir = forkDir
	if _, errPush := pushCmd.CombinedOutput(); errPush != nil {
		http.Error(w, "failed to push bookmark", http.StatusInternalServerError)
		return
	}

	if _, errGH := exec.LookPath("gh"); errGH == nil {
		prTitle := commitMessage
		prBody := ""
		prPrompt := strings.Join([]string{
			"Write a GitHub pull request title and body.",
			"Return the title on a line starting with TITLE: and the body after BODY:.",
			"Commit history:\n" + string(logOutput),
			"Diff summary:\n" + string(diffOutput),
			"GOAL:\n" + goalContent,
		}, "\n")
		if result, errPrompt := invokeLLMForAssist(prPrompt); errPrompt == nil {
			lines := strings.Split(result, "\n")
			for i, line := range lines {
				if after, found := strings.CutPrefix(line, "TITLE:"); found {
					prTitle = strings.TrimSpace(after)
				}
				if strings.HasPrefix(line, "BODY:") {
					prBody = strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
					break
				}
			}
		}
		if prTitle == "" {
			prTitle = commitMessage
		}
		prCmd := exec.Command("gh", "pr", "create", "--draft", "--head", branchName, "--title", prTitle, "--body", prBody)
		prCmd.Dir = forkDir
		if _, errPR := prCmd.CombinedOutput(); errPR != nil {
			http.Error(w, "failed to create pull request", http.StatusInternalServerError)
			return
		}
	}

	forgetCmd := exec.Command("jj", "workspace", "forget", filepath.Base(forkDir))
	forgetCmd.Dir = workspacePath
	if _, errForget := forgetCmd.CombinedOutput(); errForget != nil {
		http.Error(w, "failed to forget fork workspace", http.StatusInternalServerError)
		return
	}
	if errRemove := os.RemoveAll(forkDir); errRemove != nil {
		http.Error(w, "failed to remove fork directory", http.StatusInternalServerError)
		return
	}

	s.sseBroker.publish(sseEvent{Type: "workspace:update", Data: map[string]string{
		"workspace": filepath.Base(workspacePath),
		"action":    "merged",
		"fork":      forkName,
	}})

	writeJSON(w, apiMergeResponse{
		Merged:  true,
		Message: "fork merged successfully",
	})
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

	if classifyWorkspace(workspacePath) != workspaceRoot {
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

	if classifyWorkspace(forkDir) != workspaceFork {
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

	s.sseBroker.publish(sseEvent{Type: "workspace:update", Data: map[string]string{
		"workspace": filepath.Base(workspacePath),
		"action":    "deleted",
		"fork":      forkName,
	}})

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

	if classifyWorkspace(workspacePath) != workspaceFork {
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

	s.sseBroker.publish(sseEvent{Type: "workspace:update", Data: map[string]string{
		"workspace": oldName,
		"action":    "renamed",
		"newName":   newName,
	}})

	writeJSON(w, apiRenameResponse{
		Name:    newName,
		OldName: oldName,
		Dir:     newPath,
	})
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

	s.sseBroker.publish(sseEvent{Type: "workspace:update", Data: map[string]string{
		"workspace": filepath.Base(workspacePath),
		"action":    "goal-updated",
	}})

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

	cmd := exec.Command("opencode", "run", "-m", st.selectedModel, "--agent", "build", "--title", "adhoc ["+st.selectedModel+"]")
	cmd.Dir = workspacePath
	cmd.Env = append(os.Environ(), "OPENCODE_CONFIG_DIR="+filepath.Join(workspacePath, ".sgai"))
	cmd.Stdin = strings.NewReader(st.promptText)
	writer := &lockedWriter{mu: &st.mu, buf: &st.output}
	prefix := fmt.Sprintf("[%s][adhoc:0000]", filepath.Base(workspacePath))
	stdoutPW := &prefixWriter{prefix: prefix + " ", w: os.Stdout}
	stderrPW := &prefixWriter{prefix: prefix + " ", w: os.Stderr}
	cmd.Stdout = io.MultiWriter(stdoutPW, writer)
	cmd.Stderr = io.MultiWriter(stderrPW, writer)

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

	s.sseBroker.publish(sseEvent{Type: "workspace:update", Data: map[string]string{
		"workspace": filepath.Base(workspacePath),
		"action":    "adhoc-started",
	}})

	writeJSON(w, apiAdhocResponse{
		Running: true,
		Message: "ad-hoc prompt started",
	})
}

type apiRetroAnalyzeRequest struct {
	Session string `json:"session"`
}

type apiRetroActionResponse struct {
	Running bool   `json:"running"`
	Session string `json:"session"`
	Message string `json:"message"`
}

func (s *Server) handleAPIRetroAnalyze(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	var req apiRetroAnalyzeRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Session == "" {
		http.Error(w, "session is required", http.StatusBadRequest)
		return
	}

	sessionKey := "retro-analyze-" + workspacePath + "-" + req.Session

	s.mu.Lock()
	sess := s.sessions[sessionKey]
	if sess != nil {
		sess.mu.Lock()
		running := sess.running
		sess.mu.Unlock()
		if running {
			s.mu.Unlock()
			writeJSON(w, apiRetroActionResponse{
				Running: true,
				Session: req.Session,
				Message: "analysis already running",
			})
			return
		}
	}

	tempDir, errTemp := os.MkdirTemp("", "sgai-retro-analyze-*")
	if errTemp != nil {
		s.mu.Unlock()
		http.Error(w, "failed to create temp directory", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	sess = &session{running: true, retroTempDir: tempDir, cancel: cancel}
	s.sessions[sessionKey] = sess
	s.mu.Unlock()

	logWriter := newSessionLogWriter(sess, workspacePath, s, filepath.Base(workspacePath))

	go func() {
		defer cancel()
		errAnalyze := runRetrospectiveAnalysis(ctx, workspacePath, req.Session, tempDir, logWriter)
		if errAnalyze != nil {
			log.Println("retrospective analyze failed:", errAnalyze)
		}
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
	}()

	s.sseBroker.publish(sseEvent{Type: "workspace:update", Data: map[string]string{
		"workspace": filepath.Base(workspacePath),
		"action":    "retro-analyze-started",
		"session":   req.Session,
	}})

	writeJSON(w, apiRetroActionResponse{
		Running: true,
		Session: req.Session,
		Message: "analysis started",
	})
}

type apiRetroApplyRequest struct {
	Session             string            `json:"session"`
	SelectedSuggestions []string          `json:"selectedSuggestions"`
	Notes               map[string]string `json:"notes,omitempty"`
}

func (s *Server) handleAPIRetroApply(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	var req apiRetroApplyRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Session == "" {
		http.Error(w, "session is required", http.StatusBadRequest)
		return
	}

	sessionKey := "retro-apply-" + workspacePath + "-" + req.Session

	s.mu.Lock()
	sess := s.sessions[sessionKey]
	if sess != nil {
		sess.mu.Lock()
		running := sess.running
		sess.mu.Unlock()
		if running {
			s.mu.Unlock()
			writeJSON(w, apiRetroActionResponse{
				Running: true,
				Session: req.Session,
				Message: "apply already running",
			})
			return
		}
	}

	sess = &session{running: true}
	s.sessions[sessionKey] = sess
	s.mu.Unlock()

	retrospectivesDir := filepath.Join(workspacePath, ".sgai", "retrospectives")
	sessionDir := filepath.Join(retrospectivesDir, req.Session)
	improvementsPath := filepath.Join(sessionDir, "IMPROVEMENTS.md")

	content, errRead := os.ReadFile(improvementsPath)
	if errRead != nil {
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()
		http.Error(w, "IMPROVEMENTS.md not found", http.StatusNotFound)
		return
	}

	suggestions := parseImprovementSuggestions(string(content))
	selectedContent := buildSelectedImprovementsContent(
		filterSelectedSuggestions(suggestions, req.SelectedSuggestions),
		func(idx int) string {
			if req.Notes == nil {
				return ""
			}
			return strings.TrimSpace(req.Notes[strconv.Itoa(idx)])
		},
	)

	logWriter := newSessionLogWriter(sess, workspacePath, s, filepath.Base(workspacePath))

	go func() {
		errApply := runRetrospectiveApply(workspacePath, req.Session, selectedContent, logWriter, logWriter)
		sess.mu.Lock()
		sess.running = false
		sess.mu.Unlock()

		if errApply != nil {
			log.Println("retrospective apply failed:", errApply)
			return
		}

		if errDelete := deleteRetrospectiveSession(workspacePath, req.Session); errDelete != nil {
			log.Println("failed to auto-delete retrospective session:", req.Session, errDelete)
		}
	}()

	s.sseBroker.publish(sseEvent{Type: "workspace:update", Data: map[string]string{
		"workspace": filepath.Base(workspacePath),
		"action":    "retro-apply-started",
		"session":   req.Session,
	}})

	writeJSON(w, apiRetroActionResponse{
		Running: true,
		Session: req.Session,
		Message: "apply started",
	})
}

func (s *Server) handleAPIRetroDelete(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	var req apiRetroAnalyzeRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Session == "" {
		http.Error(w, "session is required", http.StatusBadRequest)
		return
	}

	if errDelete := deleteRetrospectiveSession(workspacePath, req.Session); errDelete != nil {
		if errors.Is(errDelete, os.ErrNotExist) {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to delete session", http.StatusInternalServerError)
		return
	}

	s.sseBroker.publish(sseEvent{Type: "workspace:update", Data: map[string]string{
		"workspace": filepath.Base(workspacePath),
		"action":    "retro-session-deleted",
		"session":   req.Session,
	}})

	writeJSON(w, struct {
		OK bool `json:"ok"`
	}{OK: true})
}

func (s *Server) handleAPIWorkflowSVG(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	wfState, _ := state.Load(statePath(workspacePath))
	currentAgent := wfState.CurrentAgent
	if currentAgent == "" {
		currentAgent = "Unknown"
	}

	svg := getWorkflowSVG(workspacePath, currentAgent)
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

type apiCommitsResponse struct {
	Commits []apiCommitEntry `json:"commits"`
}

func (s *Server) handleAPIWorkspaceCommits(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	commits := runJJLogForWorkspace(workspacePath)
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

	writeJSON(w, apiCommitsResponse{Commits: entries})
}

func runJJLogForWorkspace(dir string) []jjCommit {
	cmd := exec.Command("jj", "log", "-T", jjLogTemplate)
	cmd.Dir = dir
	output, errCmd := cmd.Output()
	if errCmd != nil {
		return nil
	}
	return parseJJLogOutput(string(output))
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

	wfState, errLoad := state.Load(statePath(workspacePath))
	if errLoad != nil {
		http.Error(w, "failed to load workspace state", http.StatusInternalServerError)
		return
	}

	newMsg := state.Message{
		FromAgent: "Human Partner",
		ToAgent:   "coordinator",
		Body:      "Re-steering instruction: " + strings.TrimSpace(req.Message),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	insertIdx := findSteerInsertPosition(wfState.Messages)
	newMsg.ID = nextMessageID(wfState.Messages)
	wfState.Messages = slices.Insert(wfState.Messages, insertIdx, newMsg)

	if errSave := state.Save(statePath(workspacePath), wfState); errSave != nil {
		http.Error(w, "failed to save state", http.StatusInternalServerError)
		return
	}

	s.publishGlobalAndWorkspace(filepath.Base(workspacePath), workspacePath, sseEvent{Type: "messages:new", Data: map[string]string{
		"workspace": filepath.Base(workspacePath),
	}})

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

	s.publishToWorkspace(workspacePath, sseEvent{Type: "changes:update", Data: map[string]string{
		"workspace": filepath.Base(workspacePath),
	}})

	writeJSON(w, apiUpdateDescriptionResponse{
		Updated:     true,
		Description: req.Description,
	})
}

type apiSelfDriveResponse struct {
	Running  bool   `json:"running"`
	AutoMode bool   `json:"autoMode"`
	Message  string `json:"message"`
}

func (s *Server) handleAPISelfDrive(w http.ResponseWriter, r *http.Request) {
	workspacePath, ok := s.resolveWorkspaceFromPath(w, r)
	if !ok {
		return
	}

	if classifyWorkspace(workspacePath) == workspaceRoot {
		http.Error(w, "root workspace cannot start agentic work", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	sess := s.sessions[workspacePath]
	s.mu.Unlock()

	var wasRunning, wasAuto bool
	if sess != nil {
		sess.mu.Lock()
		wasRunning = sess.running
		wasAuto = sess.interactiveAuto
		sess.mu.Unlock()
	}

	wfState, errLoadState := state.Load(statePath(workspacePath))
	if errLoadState != nil && !os.IsNotExist(errLoadState) {
		http.Error(w, "failed to load workflow state", http.StatusInternalServerError)
		return
	}

	if wasRunning {
		var oldPid int
		sess.mu.Lock()
		if sess.cmd != nil && sess.cmd.Process != nil {
			oldPid = sess.cmd.Process.Pid
		}
		sess.mu.Unlock()
		s.stopSession(workspacePath)
		if oldPid > 0 {
			waitForProcessExit(oldPid, 5*time.Second)
		}
	}

	newAutoMode := !wasAuto
	if wfState.InteractiveAutoLock {
		newAutoMode = true
	}

	wfState.InteractiveAutoLock = newAutoMode
	if errSave := state.Save(statePath(workspacePath), wfState); errSave != nil {
		http.Error(w, "failed to save workflow state", http.StatusInternalServerError)
		return
	}

	result := s.startSession(workspacePath, newAutoMode)
	if result.startError != nil {
		http.Error(w, result.startError.Error(), http.StatusInternalServerError)
		return
	}

	s.publishGlobalAndWorkspace(filepath.Base(workspacePath), workspacePath, sseEvent{Type: "session:update", Data: map[string]string{
		"workspace": filepath.Base(workspacePath),
	}})

	writeJSON(w, apiSelfDriveResponse{
		Running:  true,
		AutoMode: newAutoMode,
		Message:  "self-drive mode toggled",
	})
}

func waitForProcessExit(pid int, timeout time.Duration) {
	deadline := time.After(timeout)
	for {
		errProbe := syscall.Kill(pid, 0)
		if errProbe != nil {
			return
		}
		select {
		case <-deadline:
			return
		case <-time.After(50 * time.Millisecond):
		}
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

	pinned := s.isPinned(workspacePath)

	s.sseBroker.publish(sseEvent{Type: "workspace:update", Data: map[string]string{
		"workspace": filepath.Base(workspacePath),
		"action":    "pin-toggled",
	}})

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

	wfState, errState := state.Load(statePath(workspacePath))
	if errState != nil {
		http.Error(w, "failed to load workflow state", http.StatusInternalServerError)
		return
	}
	currentAgent := wfState.CurrentAgent
	sessionID := wfState.SessionID

	models := modelsForAgentFromGoal(workspacePath, currentAgent)
	var model string
	if len(models) > 0 {
		model, _ = parseModelAndVariant(models[0])
	}

	interactive := "yes"
	autoMode := wfState.InteractiveAutoLock
	if sess != nil {
		sess.mu.Lock()
		autoMode = autoMode || sess.interactiveAuto
		sess.mu.Unlock()
	}
	if autoMode {
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
