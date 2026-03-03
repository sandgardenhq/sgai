package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// chatMessage represents a message in a chat conversation.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type llmProvider string

const (
	llmProviderClaude llmProvider = "claude"
	llmProviderOpenAI llmProvider = "openai"
)

// llmConfig holds configuration for LLM API calls.
type llmConfig struct {
	Provider llmProvider
	Model    string
	APIKey   string
}

func detectLLMConfig() llmConfig {
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		model := os.Getenv("CHAT_MODEL")
		if model == "" {
			model = "claude-sonnet-4-20250514"
		}
		return llmConfig{
			Provider: llmProviderClaude,
			Model:    model,
			APIKey:   key,
		}
	}

	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		model := os.Getenv("CHAT_MODEL")
		if model == "" {
			model = "gpt-4o"
		}
		return llmConfig{
			Provider: llmProviderOpenAI,
			Model:    model,
			APIKey:   key,
		}
	}

	return llmConfig{}
}

// streamChatCompletion streams a chat completion response from the configured LLM provider.
// The onChunk callback receives each text chunk as it arrives.
// Returns an error if the API call fails.
func streamChatCompletion(ctx context.Context, messages []chatMessage, systemPrompt string, onChunk func(string)) error {
	config := detectLLMConfig()
	if config.APIKey == "" {
		return fmt.Errorf("no LLM API key configured (set ANTHROPIC_API_KEY or OPENAI_API_KEY)")
	}

	switch config.Provider {
	case llmProviderClaude:
		return streamClaudeCompletion(ctx, config, messages, systemPrompt, onChunk)
	case llmProviderOpenAI:
		return streamOpenAICompletion(ctx, config, messages, systemPrompt, onChunk)
	default:
		return fmt.Errorf("unknown LLM provider")
	}
}

func streamClaudeCompletion(ctx context.Context, config llmConfig, messages []chatMessage, systemPrompt string, onChunk func(string)) error {
	reqBody := map[string]any{
		"model":      config.Model,
		"max_tokens": 4096,
		"stream":     true,
		"messages":   messages,
	}
	if systemPrompt != "" {
		reqBody["system"] = systemPrompt
	}

	body, errMarshal := json.Marshal(reqBody)
	if errMarshal != nil {
		return fmt.Errorf("marshaling request: %w", errMarshal)
	}

	req, errReq := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if errReq != nil {
		return fmt.Errorf("creating request: %w", errReq)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", config.APIKey)
	req.Header.Set("Anthropic-Version", "2023-06-01")

	resp, errDo := http.DefaultClient.Do(req)
	if errDo != nil {
		return fmt.Errorf("sending request: %w", errDo)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("claude API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return parseClaudeSSE(resp.Body, onChunk)
}

func parseClaudeSSE(body io.Reader, onChunk func(string)) error {
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}

		if errUnmarshal := json.Unmarshal([]byte(data), &event); errUnmarshal != nil {
			continue
		}

		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
			onChunk(event.Delta.Text)
		}
	}

	return scanner.Err()
}

func streamOpenAICompletion(ctx context.Context, config llmConfig, messages []chatMessage, systemPrompt string, onChunk func(string)) error {
	allMessages := messages
	if systemPrompt != "" {
		allMessages = append([]chatMessage{{Role: "system", Content: systemPrompt}}, messages...)
	}

	reqBody := map[string]any{
		"model":    config.Model,
		"stream":   true,
		"messages": allMessages,
	}

	body, errMarshal := json.Marshal(reqBody)
	if errMarshal != nil {
		return fmt.Errorf("marshaling request: %w", errMarshal)
	}

	req, errReq := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if errReq != nil {
		return fmt.Errorf("creating request: %w", errReq)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	resp, errDo := http.DefaultClient.Do(req)
	if errDo != nil {
		return fmt.Errorf("sending request: %w", errDo)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openai API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return parseOpenAISSE(resp.Body, onChunk)
}

func parseOpenAISSE(body io.Reader, onChunk func(string)) error {
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if errUnmarshal := json.Unmarshal([]byte(data), &event); errUnmarshal != nil {
			continue
		}

		if len(event.Choices) > 0 && event.Choices[0].Delta.Content != "" {
			onChunk(event.Choices[0].Delta.Content)
		}
	}

	return scanner.Err()
}
