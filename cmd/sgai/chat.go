package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sandgardenhq/sgai/pkg/state"
)

// chatContext contains contextual information about the current page and workspace.
type chatContext struct {
	Route        string `json:"route"`
	WorkspaceDir string `json:"workspaceDir"`
}

// apiChatRequest is the request body for the chat endpoint.
type apiChatRequest struct {
	Message             string        `json:"message"`
	ConversationHistory []chatMessage `json:"conversationHistory"`
	Context             chatContext   `json:"context"`
}

// apiChatConfigResponse returns the configuration status for the chat assistant.
type apiChatConfigResponse struct {
	Configured bool   `json:"configured"`
	Provider   string `json:"provider,omitempty"`
	Model      string `json:"model,omitempty"`
}

func (s *Server) handleAPIChatConfig(w http.ResponseWriter, _ *http.Request) {
	config := detectLLMConfig()
	writeJSON(w, apiChatConfigResponse{
		Configured: config.APIKey != "",
		Provider:   string(config.Provider),
		Model:      config.Model,
	})
}

func (s *Server) handleAPIChat(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	var req apiChatRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Message) == "" {
		http.Error(w, "message cannot be empty", http.StatusBadRequest)
		return
	}

	config := detectLLMConfig()
	if config.APIKey == "" {
		http.Error(w, "no LLM API key configured (set ANTHROPIC_API_KEY or OPENAI_API_KEY)", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	systemPrompt := buildChatSystemPrompt(req.Context, s.rootDir)
	relevantDocs := retrieveRelevantChunks(req.Message, 5)

	fullSystemPrompt := systemPrompt
	if docsContent := formatRetrievedDocs(relevantDocs); docsContent != "" {
		fullSystemPrompt += "\n\n## RELEVANT DOCUMENTATION\n\n" + docsContent
	}

	messages := buildChatMessages(req.ConversationHistory, req.Message)

	errStream := streamChatCompletion(r.Context(), messages, fullSystemPrompt, func(chunk string) {
		data, errMarshal := json.Marshal(map[string]string{"content": chunk})
		if errMarshal != nil {
			return
		}
		_, errWrite := fmt.Fprintf(w, "data: %s\n\n", data)
		if errWrite != nil {
			return
		}
		flusher.Flush()
	})

	if errStream != nil {
		log.Println("chat streaming error:", errStream)
		data, _ := json.Marshal(map[string]string{"error": errStream.Error()})
		if _, errWrite := fmt.Fprintf(w, "data: %s\n\n", data); errWrite != nil {
			return
		}
		flusher.Flush()
	}

	if _, errWrite := fmt.Fprintf(w, "data: [DONE]\n\n"); errWrite != nil {
		return
	}
	flusher.Flush()
}

func buildChatSystemPrompt(ctx chatContext, rootDir string) string {
	var sb strings.Builder

	sb.WriteString("You are an SGAI assistant helping users understand SGAI (Software Generation AI).\n")
	sb.WriteString("SGAI is an AI-powered software factory that uses specialized agents to accomplish development tasks.\n\n")
	sb.WriteString("Key concepts:\n")
	sb.WriteString("- GOAL.md: Defines the project goal and workflow configuration in YAML frontmatter\n")
	sb.WriteString("- Agents: Specialized AI agents (coordinator, backend-go-developer, etc.)\n")
	sb.WriteString("- Flow: Defines how agents connect and pass work to each other\n")
	sb.WriteString("- State: Workflow state is stored in .sgai/state.json\n")
	sb.WriteString("- Skills: Reusable prompt templates that guide agent behavior\n")
	sb.WriteString("- Snippets: Reusable code templates\n\n")
	sb.WriteString("Be helpful, concise, and accurate. If you don't know something, say so.\n")
	sb.WriteString("Reference the documentation when applicable.\n")

	if ctx.WorkspaceDir != "" {
		workspacePath := filepath.Join(rootDir, ctx.WorkspaceDir)
		sb.WriteString("\n## CURRENT CONTEXT\n\n")
		sb.WriteString(fmt.Sprintf("- Current Page: %s\n", ctx.Route))
		sb.WriteString(fmt.Sprintf("- Workspace: %s\n", ctx.WorkspaceDir))

		if goalContent := readWorkspaceFile(workspacePath, "GOAL.md"); goalContent != "" {
			sb.WriteString("\n### GOAL.md Content:\n")
			sb.WriteString("```\n")
			sb.WriteString(truncateContent(goalContent, 2000))
			sb.WriteString("\n```\n")
		}

		if wfState := readWorkspaceState(workspacePath); wfState != nil {
			sb.WriteString(fmt.Sprintf("\n### Workflow Status: %s\n", wfState.Status))
			if wfState.CurrentAgent != "" {
				sb.WriteString(fmt.Sprintf("### Current Agent: %s\n", wfState.CurrentAgent))
			}
			if wfState.Task != "" {
				sb.WriteString(fmt.Sprintf("### Current Task: %s\n", wfState.Task))
			}
		}
	}

	return sb.String()
}

func buildChatMessages(history []chatMessage, currentMessage string) []chatMessage {
	messages := make([]chatMessage, 0, len(history)+1)
	messages = append(messages, history...)
	messages = append(messages, chatMessage{
		Role:    "user",
		Content: currentMessage,
	})
	return messages
}

func readWorkspaceFile(workspacePath, filename string) string {
	filePath := filepath.Join(workspacePath, filename)
	content, errRead := os.ReadFile(filePath)
	if errRead != nil {
		return ""
	}
	return string(content)
}

func readWorkspaceState(workspacePath string) *state.Workflow {
	stateJSONPath := filepath.Join(workspacePath, ".sgai", "state.json")
	wfState, errLoad := state.Load(stateJSONPath)
	if errLoad != nil {
		return nil
	}
	return &wfState
}

func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "\n... (truncated)"
}
