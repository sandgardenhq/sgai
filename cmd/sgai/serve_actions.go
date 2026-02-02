package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func (s *Server) handleWorkspaceInit(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	sgaiDir := filepath.Join(workspacePath, ".sgai")
	if err := os.MkdirAll(sgaiDir, 0755); err != nil {
		http.Error(w, "failed to create workspace directory", http.StatusInternalServerError)
		return
	}

	goalPath := filepath.Join(workspacePath, "GOAL.md")
	if err := os.WriteFile(goalPath, []byte(goalExampleContent), 0644); err != nil {
		http.Error(w, "failed to create GOAL.md", http.StatusInternalServerError)
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

	sgaiDir := filepath.Join(workspacePath, ".sgai")
	if err := os.MkdirAll(sgaiDir, 0755); err != nil {
		s.renderNewWorkspaceWithError(w, "failed to create .sgai directory")
		return
	}

	goalPath := filepath.Join(workspacePath, "GOAL.md")
	if err := os.WriteFile(goalPath, []byte(goalExampleContent), 0644); err != nil {
		s.renderNewWorkspaceWithError(w, "failed to create GOAL.md")
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

// handleWorkspaceOpenVSCode opens VSCode for a workspace or specific file.
// Security: Only allows localhost requests and a whitelist of specific files.
func (s *Server) handleWorkspaceOpenVSCode(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	if !s.codeAvailable {
		http.Error(w, "VSCode CLI not available", http.StatusServiceUnavailable)
		return
	}

	if !isLocalRequest(r) {
		http.Error(w, "VSCode can only be opened from localhost", http.StatusForbidden)
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
		http.Error(w, "Failed to open VSCode", http.StatusInternalServerError)
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

	sgaiDir := filepath.Join(targetPath, ".sgai")
	if errMkdir := os.MkdirAll(sgaiDir, 0755); errMkdir != nil {
		log.Printf("Warning: failed to create .sgai directory: %v", errMkdir)
	}

	goalPath := filepath.Join(targetPath, "GOAL.md")
	if errWrite := os.WriteFile(goalPath, []byte(goalExampleContent), 0644); errWrite != nil {
		log.Println("warning: failed to create GOAL.md:", errWrite)
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
