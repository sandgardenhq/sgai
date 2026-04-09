# Claude Code Backend Adapter Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Support both opencode and Claude Code as agent backends, selected per-project via `sgai.json`.

**Architecture:** Extract a Go `Backend` interface from the current opencode-specific code. Two implementations — `opencodeBackend` and `claudeCodeBackend` — encapsulate CLI args, environment variables, JSON event parsing, model validation, and session export. The `sgai.json` config gains a `"backend"` field (`"opencode"` default, `"claude-code"` option). At startup, SGAI instantiates the chosen backend once and threads it through to all call sites.

**Tech Stack:** Go, Claude Code CLI (`claude`), opencode CLI (`opencode`)

---

### Task 1: Define the Backend interface

**Files:**
- Create: `cmd/sgai/backend.go`
- Test: `cmd/sgai/backend_test.go`

**Step 1: Write the interface and types**

```go
package main

// Backend abstracts the CLI agent runner (opencode vs Claude Code).
type Backend interface {
	// Name returns the backend identifier ("opencode" or "claude-code").
	Name() string

	// BinaryName returns the CLI binary to exec (e.g. "opencode" or "claude").
	BinaryName() string

	// BuildAgentArgs builds CLI arguments for a workflow agent run.
	BuildAgentArgs(p AgentRunParams) []string

	// BuildAdhocArgs builds CLI arguments for an ad-hoc prompt run.
	BuildAdhocArgs(modelSpec string) []string

	// BuildEnv builds the environment variable slice for the agent process.
	BuildEnv(p AgentEnvParams) []string

	// BuildContinuousArgs builds CLI arguments for continuous mode.
	BuildContinuousArgs() []string

	// ParseEvent normalizes a JSON line from the agent's stdout into a
	// streamEvent. Returns false if the line is not a recognized event.
	ParseEvent(line []byte) (streamEvent, bool)

	// ValidateModels checks that all model specs are valid for this backend.
	// Returns nil if validation is not supported (e.g. Claude Code has no
	// models command).
	ValidateModels(models map[string]any) error

	// ExportSession exports a session to the given output path.
	// Returns nil if export is not supported.
	ExportSession(dir, sessionID, outputPath string) error

	// StripProviderPrefix transforms a model spec for this backend.
	// opencode expects "anthropic/claude-opus-4-6", Claude Code expects "claude-opus-4-6".
	StripProviderPrefix(model string) string
}

// AgentRunParams contains the parameters for BuildAgentArgs.
type AgentRunParams struct {
	Agent     string // display name (may include alias resolution)
	BaseAgent string // agent identity passed to CLI --agent flag
	ModelSpec string // e.g. "anthropic/claude-opus-4-6 (max)"
	SessionID string // resume session
}

// AgentEnvParams contains the parameters for BuildEnv.
type AgentEnvParams struct {
	Dir             string
	McpURL          string
	AgentIdentity   string
	InteractiveMode string // "yes" or "auto"
}
```

**Step 2: Commit**

```bash
git add cmd/sgai/backend.go
git commit -m "feat: add Backend interface for multi-backend support"
```

---

### Task 2: Implement opencodeBackend

**Files:**
- Create: `cmd/sgai/backend_opencode.go`
- Test: `cmd/sgai/backend_opencode_test.go`

This extracts existing logic from `main.go` into the interface. No behavior changes.

**Step 1: Write tests for BuildAgentArgs**

```go
package main

import "testing"

func TestOpencodeBackendBuildAgentArgs(t *testing.T) {
	b := &opencodeBackend{}
	args := b.BuildAgentArgs(AgentRunParams{
		Agent:     "coordinator",
		BaseAgent: "coordinator",
		ModelSpec: "anthropic/claude-opus-4-6 (max)",
		SessionID: "ses_123",
	})
	// Verify: "run", "--format=json", "--agent", "coordinator",
	//         "--model", "anthropic/claude-opus-4-6", "--variant", "max",
	//         "--session", "ses_123", "--title", "coordinator [anthropic/claude-opus-4-6 (max)]"
	assertContains(t, args, "--format=json")
	assertContains(t, args, "--agent")
	assertContains(t, args, "--variant")
	assertContains(t, args, "--session")
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/smankowski/github/sgai && go test ./cmd/sgai/ -run TestOpencodeBackendBuildAgentArgs -v`
Expected: FAIL — `opencodeBackend` not defined

**Step 3: Implement opencodeBackend**

Move the existing logic from `buildAgentArgs`, `buildAdhocArgs`, `buildAgentEnv`, `fetchValidModels`, `validateModels`, and `exportSession` in `main.go` into methods on `opencodeBackend`. The struct holds no state — it's a pure method dispatch.

```go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type opencodeBackend struct{}

func (b *opencodeBackend) Name() string       { return "opencode" }
func (b *opencodeBackend) BinaryName() string { return "opencode" }

func (b *opencodeBackend) BuildAgentArgs(p AgentRunParams) []string {
	args := []string{"run", "--format=json", "--agent", p.BaseAgent}
	if p.ModelSpec != "" {
		model, variant := parseModelAndVariant(p.ModelSpec)
		args = append(args, "--model", model)
		if variant != "" {
			args = append(args, "--variant", variant)
		}
	}
	if p.SessionID != "" {
		args = append(args, "--session", p.SessionID)
	}
	title := p.Agent
	if p.ModelSpec != "" {
		title = p.Agent + " [" + p.ModelSpec + "]"
	}
	args = append(args, "--title", title)
	return args
}

func (b *opencodeBackend) BuildAdhocArgs(modelSpec string) []string {
	baseModel, variant := parseModelAndVariant(modelSpec)
	args := []string{"run", "-m", baseModel, "--agent", "build", "--title", "adhoc [" + modelSpec + "]"}
	if variant != "" {
		args = append(args, "--variant", variant)
	}
	return args
}

func (b *opencodeBackend) BuildEnv(p AgentEnvParams) []string {
	return append(os.Environ(),
		"OPENCODE_CONFIG_DIR="+filepath.Join(p.Dir, ".sgai"),
		"SGAI_MCP_URL="+p.McpURL,
		"SGAI_AGENT_IDENTITY="+p.AgentIdentity,
		"SGAI_MCP_INTERACTIVE="+p.InteractiveMode)
}

func (b *opencodeBackend) BuildContinuousArgs() []string {
	return []string{"run", "--title", "continuous-mode-prompt"}
}

func (b *opencodeBackend) ParseEvent(line []byte) (streamEvent, bool) {
	// Use existing JSON unmarshal — opencode format is the native format
	var event streamEvent
	if err := json.Unmarshal(line, &event); err != nil {
		return streamEvent{}, false
	}
	return event, true
}

func (b *opencodeBackend) ValidateModels(models map[string]any) error {
	// Existing validateModels logic using `opencode models`
	return validateModelsWithCommand(models)
}

func (b *opencodeBackend) ExportSession(dir, sessionID, outputPath string) error {
	cmd := exec.Command("opencode", "export", sessionID)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "OPENCODE_CONFIG_DIR="+filepath.Join(dir, ".sgai"))
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("opencode export failed: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(outputPath, output, 0644)
}

func (b *opencodeBackend) StripProviderPrefix(model string) string {
	return model // opencode uses the full "anthropic/claude-opus-4-6" form
}
```

**Step 4: Run tests**

Run: `cd /Users/smankowski/github/sgai && go test ./cmd/sgai/ -run TestOpencode -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cmd/sgai/backend_opencode.go cmd/sgai/backend_opencode_test.go
git commit -m "feat: implement opencodeBackend extracting existing logic"
```

---

### Task 3: Implement claudeCodeBackend

**Files:**
- Create: `cmd/sgai/backend_claudecode.go`
- Test: `cmd/sgai/backend_claudecode_test.go`

**Step 1: Write tests for BuildAgentArgs**

```go
func TestClaudeCodeBackendBuildAgentArgs(t *testing.T) {
	b := &claudeCodeBackend{}
	args := b.BuildAgentArgs(AgentRunParams{
		Agent:     "coordinator",
		BaseAgent: "coordinator",
		ModelSpec: "anthropic/claude-opus-4-6 (max)",
		SessionID: "abc-123-def",
	})
	// Should produce: "-p", "--output-format", "stream-json", "--verbose",
	//   "--model", "claude-opus-4-6", "--effort", "max",
	//   "--session-id", "abc-123-def", "--name", "coordinator [anthropic/claude-opus-4-6 (max)]"
	assertContains(t, args, "-p")
	assertContains(t, args, "--output-format")
	assertNotContains(t, args, "--format=json") // opencode flag
	assertNotContains(t, args, "anthropic/")    // provider prefix stripped
}

func TestClaudeCodeStripProviderPrefix(t *testing.T) {
	b := &claudeCodeBackend{}
	tests := []struct{ input, want string }{
		{"anthropic/claude-opus-4-6", "claude-opus-4-6"},
		{"openai/gpt-4o", "gpt-4o"},
		{"claude-opus-4-6", "claude-opus-4-6"}, // no prefix
	}
	for _, tt := range tests {
		got := b.StripProviderPrefix(tt.input)
		if got != tt.want {
			t.Errorf("StripProviderPrefix(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/smankowski/github/sgai && go test ./cmd/sgai/ -run TestClaudeCode -v`
Expected: FAIL

**Step 3: Implement claudeCodeBackend**

Key differences from opencode:
- Binary: `claude` not `opencode`
- Uses `-p` (print mode) + `--output-format stream-json --verbose`
- `--variant` maps to `--effort`
- `--session` maps to `--session-id`
- `--title` maps to `--name`
- No `--agent` flag for `.sgai/` agents — reads the agent `.md` file and passes via `--system-prompt-file`
- Model format: strips `anthropic/` prefix
- No `OPENCODE_CONFIG_DIR` — uses `--mcp-config` and `--settings` flags
- Needs `--bare` to avoid loading user's own CLAUDE.md and plugins, BUT needs OAuth — so use `--bare` + explicit flags, or don't use `--bare` and accept the user's config bleeds through. Decision: use `--bare` is NOT an option since it disables OAuth. Instead, use `--system-prompt-file` to override the system prompt and `--permission-mode bypassPermissions` for non-interactive runs. The SGAI MCP server is injected via `--mcp-config`.
- `--dangerously-skip-permissions` for fully automated runs (SGAI already controls permissions via its own MCP)

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type claudeCodeBackend struct{}

func (b *claudeCodeBackend) Name() string       { return "claude-code" }
func (b *claudeCodeBackend) BinaryName() string { return "claude" }

func (b *claudeCodeBackend) BuildAgentArgs(p AgentRunParams) []string {
	args := []string{"-p", "--output-format", "stream-json", "--verbose"}

	if p.ModelSpec != "" {
		model, variant := parseModelAndVariant(p.ModelSpec)
		args = append(args, "--model", b.StripProviderPrefix(model))
		if variant != "" {
			args = append(args, "--effort", variant)
		}
	}
	if p.SessionID != "" {
		args = append(args, "--session-id", p.SessionID)
	}
	title := p.Agent
	if p.ModelSpec != "" {
		title = p.Agent + " [" + p.ModelSpec + "]"
	}
	args = append(args, "--name", title)
	// No --agent flag — system prompt is injected via stdin message preamble
	// or --append-system-prompt. The agent .md content is read by SGAI and
	// prepended to the prompt message by buildAgentMessage.
	return args
}

func (b *claudeCodeBackend) BuildAdhocArgs(modelSpec string) []string {
	baseModel, variant := parseModelAndVariant(modelSpec)
	args := []string{"-p", "--output-format", "stream-json", "--verbose",
		"--model", b.StripProviderPrefix(baseModel),
		"--name", "adhoc [" + modelSpec + "]"}
	if variant != "" {
		args = append(args, "--effort", variant)
	}
	return args
}

func (b *claudeCodeBackend) BuildEnv(p AgentEnvParams) []string {
	return append(os.Environ(),
		"SGAI_MCP_URL="+p.McpURL,
		"SGAI_AGENT_IDENTITY="+p.AgentIdentity,
		"SGAI_MCP_INTERACTIVE="+p.InteractiveMode)
}

func (b *claudeCodeBackend) BuildContinuousArgs() []string {
	return []string{"-p", "--output-format", "stream-json", "--verbose",
		"--name", "continuous-mode-prompt"}
}

func (b *claudeCodeBackend) StripProviderPrefix(model string) string {
	if idx := strings.Index(model, "/"); idx >= 0 {
		return model[idx+1:]
	}
	return model
}

func (b *claudeCodeBackend) ValidateModels(models map[string]any) error {
	return nil // Claude Code has no `models` command; skip validation
}

func (b *claudeCodeBackend) ExportSession(dir, sessionID, outputPath string) error {
	return nil // Claude Code sessions are stored in ~/.claude/; no export needed
}
```

**Step 4: Write tests for ParseEvent (Claude Code stream-json format)**

Claude Code's stream-json format differs from opencode's. Key event types to handle:

- `{"type":"assistant","message":{"content":[{"type":"text","text":"..."}]}, "session_id":"..."}` → `streamEvent{Type:"text", Part:{Text:"..."}, SessionID:"..."}`
- `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":{...}}]}}` → `streamEvent{Type:"tool_use", Part:{Tool:"Edit", State:{...}}}`
- `{"type":"result","session_id":"...","result":"..."}` → passed through as-is
- `{"type":"system","subtype":"init",...}` → extract session_id, ignore otherwise

```go
func TestClaudeCodeParseEventText(t *testing.T) {
	b := &claudeCodeBackend{}
	line := []byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]},"session_id":"abc-123"}`)
	event, ok := b.ParseEvent(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if event.Type != "text" {
		t.Errorf("got type %q, want text", event.Type)
	}
	if event.Part.Text != "hello" {
		t.Errorf("got text %q, want hello", event.Part.Text)
	}
	if event.SessionID != "abc-123" {
		t.Errorf("got session %q, want abc-123", event.SessionID)
	}
}
```

**Step 5: Implement ParseEvent**

This is the most complex part — translating Claude Code's richer JSON event structure into the existing `streamEvent` format that `jsonPrettyWriter.processEvent` already handles.

```go
// claudeCodeRawEvent represents the raw JSON from Claude Code's stream-json output.
type claudeCodeRawEvent struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Message   *struct {
		Content []struct {
			Type     string         `json:"type"`
			Text     string         `json:"text,omitempty"`
			Thinking string         `json:"thinking,omitempty"`
			Name     string         `json:"name,omitempty"`
			ID       string         `json:"id,omitempty"`
			Input    map[string]any `json:"input,omitempty"`
		} `json:"content,omitempty"`
		Usage *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage,omitempty"`
	} `json:"message,omitempty"`
	// Result event fields (type=="result")
	Result       string  `json:"result,omitempty"`
	TotalCostUSD float64 `json:"total_cost_usd,omitempty"`
	Usage        *struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	} `json:"usage,omitempty"`
}

func (b *claudeCodeBackend) ParseEvent(line []byte) (streamEvent, bool) {
	var raw claudeCodeRawEvent
	if err := json.Unmarshal(line, &raw); err != nil {
		return streamEvent{}, false
	}

	base := streamEvent{SessionID: raw.SessionID}

	switch raw.Type {
	case "system":
		// Extract session_id from init events, skip otherwise
		if raw.SessionID != "" {
			base.Type = "system"
			return base, true
		}
		return streamEvent{}, false

	case "assistant":
		if raw.Message == nil || len(raw.Message.Content) == 0 {
			return streamEvent{}, false
		}
		content := raw.Message.Content[0]
		switch content.Type {
		case "text":
			base.Type = "text"
			base.Part.Text = content.Text
		case "thinking":
			base.Type = "reasoning"
			base.Part.Text = content.Thinking
		case "tool_use":
			base.Type = "tool_use"
			base.Part.Tool = content.Name
			base.Part.State = &toolState{
				Status: "running",
				Input:  content.Input,
			}
		default:
			return streamEvent{}, false
		}
		return base, true

	case "result":
		base.Type = "result"
		if raw.Usage != nil {
			base.Part.Cost = raw.TotalCostUSD
			base.Part.Tokens.Input = raw.Usage.InputTokens
			base.Part.Tokens.Output = raw.Usage.OutputTokens
			base.Part.Tokens.Cache.Read = raw.Usage.CacheReadInputTokens
			base.Part.Tokens.Cache.Write = raw.Usage.CacheCreationInputTokens
		}
		return base, true

	default:
		return streamEvent{}, false
	}
}
```

**Step 6: Run all Claude Code backend tests**

Run: `cd /Users/smankowski/github/sgai && go test ./cmd/sgai/ -run TestClaudeCode -v`
Expected: PASS

**Step 7: Commit**

```bash
git add cmd/sgai/backend_claudecode.go cmd/sgai/backend_claudecode_test.go
git commit -m "feat: implement claudeCodeBackend for Claude Code CLI"
```

---

### Task 4: Add backend config to sgai.json

**Files:**
- Modify: `cmd/sgai/config.go`
- Modify: `cmd/sgai/config_test.go`

**Step 1: Write test for backend field parsing**

```go
func TestProjectConfigBackend(t *testing.T) {
	t.Run("defaultsToOpencode", func(t *testing.T) {
		config := &projectConfig{}
		b := resolveBackend(config)
		if b.Name() != "opencode" {
			t.Errorf("got %q, want opencode", b.Name())
		}
	})
	t.Run("claudeCode", func(t *testing.T) {
		config := &projectConfig{Backend: "claude-code"}
		b := resolveBackend(config)
		if b.Name() != "claude-code" {
			t.Errorf("got %q, want claude-code", b.Name())
		}
	})
	t.Run("invalidBackend", func(t *testing.T) {
		config := &projectConfig{Backend: "invalid"}
		_, err := resolveBackendStrict(config)
		if err == nil {
			t.Error("expected error for invalid backend")
		}
	})
}
```

**Step 2: Add `Backend` field to `projectConfig`**

In `cmd/sgai/config.go`:

```go
type projectConfig struct {
	DefaultModel string                     `json:"defaultModel,omitempty"`
	Backend      string                     `json:"backend,omitempty"` // "opencode" (default) or "claude-code"
	MCP          map[string]json.RawMessage `json:"mcp,omitempty"`
	Editor       string                     `json:"editor,omitempty"`
	Actions      []actionConfig             `json:"actions,omitempty"`
}

func resolveBackend(config *projectConfig) Backend {
	if config != nil && config.Backend == "claude-code" {
		return &claudeCodeBackend{}
	}
	return &opencodeBackend{}
}
```

**Step 3: Run tests**

Run: `cd /Users/smankowski/github/sgai && go test ./cmd/sgai/ -run TestProjectConfig -v`
Expected: PASS

**Step 4: Commit**

```bash
git add cmd/sgai/config.go cmd/sgai/config_test.go
git commit -m "feat: add backend field to sgai.json config"
```

---

### Task 5: Wire backend through to main.go call sites

**Files:**
- Modify: `cmd/sgai/main.go`
- Modify: `cmd/sgai/serve_api.go`
- Modify: `cmd/sgai/service_adhoc.go`
- Modify: `cmd/sgai/continuous.go`

This is the integration task. Thread the `Backend` through `multiModelConfig` and replace all hardcoded `"opencode"` exec calls.

**Step 1: Add Backend to multiModelConfig**

In `cmd/sgai/main.go`, add `backend Backend` to the `multiModelConfig` struct. Initialize it from `resolveBackend(config)` during startup.

**Step 2: Replace `buildAgentArgs` call site**

In `runFlowAgentWithModel` (main.go:377):

```go
// Before:
agentArgs := buildAgentArgs(cfg.agent, baseAgent, modelSpec, capturedSessionID)

// After:
agentArgs := cfg.backend.BuildAgentArgs(AgentRunParams{
	Agent:     cfg.agent,
	BaseAgent: baseAgent,
	ModelSpec: modelSpec,
	SessionID: capturedSessionID,
})
```

**Step 3: Replace `executeAgentProcess` exec call**

In `executeAgentProcess` (main.go:535):

```go
// Before:
cmd := exec.CommandContext(agentCtx, "opencode", agentArgs...)

// After:
cmd := exec.CommandContext(agentCtx, cfg.backend.BinaryName(), agentArgs...)
```

**Step 4: Replace `buildAgentEnv` call site**

In `buildAgentEnv` (main.go:506):

```go
// Before:
return append(os.Environ(),
	"OPENCODE_CONFIG_DIR="+filepath.Join(cfg.dir, ".sgai"),
	...)

// After:
return cfg.backend.BuildEnv(AgentEnvParams{
	Dir:             cfg.dir,
	McpURL:          cfg.mcpURL,
	AgentIdentity:   agentIdentity,
	InteractiveMode: interactiveEnv,
})
```

**Step 5: Replace model validation**

In `main.go` where `validateModels` is called:

```go
// Before:
if err := validateModels(models); err != nil { ... }

// After:
if err := backend.ValidateModels(models); err != nil { ... }
```

**Step 6: Replace session export**

In `exportSession` (main.go:1700):

```go
// Before:
cmd := exec.Command("opencode", "export", sessionID)
cmd.Env = append(os.Environ(), "OPENCODE_CONFIG_DIR="+filepath.Join(dir, ".sgai"))

// After:
return cfg.backend.ExportSession(dir, sessionID, outputPath)
```

**Step 7: Replace startup binary check**

In `main.go:54`:

```go
// Before:
if _, err := exec.LookPath("opencode"); err != nil {
	log.Fatalln("opencode is required but not found in PATH")
}

// After:
if _, err := exec.LookPath(backend.BinaryName()); err != nil {
	log.Fatalf("%s is required but not found in PATH", backend.BinaryName())
}
```

**Step 8: Replace adhoc call sites**

In `service_adhoc.go:56` and `serve_api.go:2086`:

```go
// Before:
args := buildAdhocArgs(st.selectedModel)
cmd := exec.Command("opencode", args...)
cmd.Env = append(os.Environ(), "OPENCODE_CONFIG_DIR="+filepath.Join(workspacePath, ".sgai"))

// After:
args := backend.BuildAdhocArgs(st.selectedModel)
cmd := exec.Command(backend.BinaryName(), args...)
cmd.Env = backend.BuildEnv(AgentEnvParams{Dir: workspacePath, ...})
```

**Step 9: Replace continuous mode**

In `continuous.go:52`:

```go
// Before:
cmd := exec.CommandContext(ctx, "opencode", "run", "--title", "continuous-mode-prompt")
cmd.Env = append(os.Environ(),
	"OPENCODE_CONFIG_DIR="+filepath.Join(dir, ".sgai"),
	...)

// After:
args := backend.BuildContinuousArgs()
cmd := exec.CommandContext(ctx, backend.BinaryName(), args...)
cmd.Env = backend.BuildEnv(AgentEnvParams{Dir: dir, McpURL: mcpURL, InteractiveMode: "auto"})
```

**Step 10: Update jsonPrettyWriter to use backend parser**

In `jsonPrettyWriter.processBuffer` (main.go:1321), the current code does direct `json.Unmarshal` into `streamEvent`. For the Claude Code backend, we need to route through `ParseEvent`:

```go
// Add backend field to jsonPrettyWriter
type jsonPrettyWriter struct {
	// ... existing fields ...
	backend Backend
}

func (j *jsonPrettyWriter) processBuffer() {
	for {
		idx := strings.Index(string(j.buf), "\n")
		if idx == -1 {
			return
		}
		line := j.buf[:idx]
		j.buf = j.buf[idx+1:]
		if len(line) == 0 {
			continue
		}

		event, ok := j.backend.ParseEvent(line)
		if !ok {
			continue
		}
		j.processEvent(event)
	}
}
```

**Step 11: Run full test suite**

Run: `cd /Users/smankowski/github/sgai && go test ./cmd/sgai/ -v -count=1`
Expected: PASS (all existing tests should still pass since default backend is opencode)

**Step 12: Commit**

```bash
git add cmd/sgai/main.go cmd/sgai/serve_api.go cmd/sgai/service_adhoc.go cmd/sgai/continuous.go
git commit -m "feat: wire Backend interface through all agent execution paths"
```

---

### Task 6: Handle Claude Code agent identity via system prompt

**Files:**
- Modify: `cmd/sgai/backend_claudecode.go`
- Modify: `cmd/sgai/main.go` (buildAgentMessage or executeAgentProcess)

opencode resolves `--agent coordinator` by reading `.sgai/agent/coordinator.md` itself. Claude Code's `--agent` flag refers to its own built-in agents (Explore, Plan, etc.), not SGAI's.

**Approach:** For Claude Code, SGAI reads the agent `.md` file content and passes it via `--append-system-prompt` flag. The frontmatter (permissions, etc.) is stripped — only the body is passed.

**Step 1: Add BuildSystemPromptArgs to claudeCodeBackend**

```go
// In BuildAgentArgs, after the base args, add the system prompt from the agent file:
func (b *claudeCodeBackend) BuildAgentArgs(p AgentRunParams) []string {
	args := []string{"-p", "--output-format", "stream-json", "--verbose"}
	// ... model, session, name flags ...

	// Agent system prompt: read from .sgai/agent/<name>.md, extract body
	if p.AgentDir != "" {
		agentPath := filepath.Join(p.AgentDir, ".sgai", "agent", p.BaseAgent+".md")
		if content, err := os.ReadFile(agentPath); err == nil {
			body := string(extractBody(content))
			if body != "" {
				args = append(args, "--append-system-prompt", body)
			}
		}
	}

	return args
}
```

Add `AgentDir string` to `AgentRunParams`.

**Step 2: Handle Claude Code permissions**

The `.sgai/agent/*.md` frontmatter contains permission rules (edit deny/allow, doom_loop, etc.). For Claude Code:
- `--permission-mode bypassPermissions` for automated runs (SGAI controls permissions through its own MCP layer)
- Or `--dangerously-skip-permissions` if SGAI is running in a sandboxed context

Add to `BuildAgentArgs`:
```go
args = append(args, "--permission-mode", "bypassPermissions")
```

**Step 3: Handle MCP config for Claude Code**

Claude Code needs to know about the SGAI MCP server. Generate a temporary MCP config JSON and pass via `--mcp-config`:

```go
func (b *claudeCodeBackend) BuildEnv(p AgentEnvParams) []string {
	env := append(os.Environ(),
		"SGAI_MCP_URL="+p.McpURL,
		"SGAI_AGENT_IDENTITY="+p.AgentIdentity,
		"SGAI_MCP_INTERACTIVE="+p.InteractiveMode)
	return env
}

// In BuildAgentArgs, add MCP config:
// The SGAI MCP server URL is passed via env var, and the agent's
// opencode.jsonc MCP section needs to be translated to Claude Code format.
// For now, pass --mcp-config with the SGAI MCP server.
```

The MCP config translation (opencode.jsonc format to Claude Code settings.json format) is a separate concern. For the initial implementation, SGAI's own MCP server (which provides sgai_update_workflow_state, sgai_send_message, etc.) is the critical one — project MCPs (playwright, context7) from opencode.jsonc can be translated in a follow-up.

**Step 4: Test end-to-end with a simple prompt**

Write an integration test that:
1. Creates a temp dir with `.sgai/agent/test.md`
2. Builds args via `claudeCodeBackend`
3. Verifies `--append-system-prompt` contains the agent body
4. Verifies `--permission-mode` is set

**Step 5: Commit**

```bash
git add cmd/sgai/backend_claudecode.go cmd/sgai/main.go cmd/sgai/backend_claudecode_test.go
git commit -m "feat: handle agent identity and permissions for Claude Code backend"
```

---

### Task 7: Generate MCP config for Claude Code

**Files:**
- Modify: `cmd/sgai/backend_claudecode.go`
- Create: `cmd/sgai/backend_claudecode_mcp.go` (if large enough)
- Test: `cmd/sgai/backend_claudecode_mcp_test.go`

Claude Code uses `--mcp-config` to load MCP servers. SGAI needs to:
1. Always inject the SGAI MCP server (the one serving workflow tools)
2. Optionally translate project MCPs from `opencode.jsonc`

**Step 1: Write the SGAI MCP config generator**

```go
func buildClaudeCodeMCPConfig(sgaiMCPURL string, projectMCPs map[string]json.RawMessage) (string, error) {
	config := map[string]any{
		"mcpServers": map[string]any{
			"sgai": map[string]any{
				"type": "sse",
				"url":  sgaiMCPURL,
			},
		},
	}
	// Add project MCPs (translate opencode format to Claude Code format)
	if servers, ok := config["mcpServers"].(map[string]any); ok {
		for name, rawConfig := range projectMCPs {
			var ocMCP struct {
				Type    string   `json:"type"`
				Command []string `json:"command"`
				Enabled *bool    `json:"enabled,omitempty"`
			}
			if err := json.Unmarshal(rawConfig, &ocMCP); err != nil {
				continue
			}
			if ocMCP.Enabled != nil && !*ocMCP.Enabled {
				continue
			}
			if ocMCP.Type == "local" && len(ocMCP.Command) > 0 {
				servers[name] = map[string]any{
					"command": ocMCP.Command[0],
					"args":    ocMCP.Command[1:],
				}
			}
		}
	}

	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
```

Pass the generated JSON string to `--mcp-config` in `BuildAgentArgs`:
```go
args = append(args, "--mcp-config", mcpConfigJSON)
```

**Step 2: Test MCP config generation**

**Step 3: Commit**

```bash
git add cmd/sgai/backend_claudecode_mcp.go cmd/sgai/backend_claudecode_mcp_test.go
git commit -m "feat: generate MCP config for Claude Code backend"
```

---

### Task 8: Update applyCustomMCPs for dual-backend support

**Files:**
- Modify: `cmd/sgai/config.go`

The current `applyCustomMCPs` reads and writes `opencode.jsonc`. For Claude Code, this isn't needed — MCP config is passed via `--mcp-config` flag at runtime.

**Step 1: Guard applyCustomMCPs behind backend check**

```go
func applyCustomMCPs(dir string, config *projectConfig, backend Backend) error {
	if backend.Name() != "opencode" {
		return nil // Claude Code handles MCPs via --mcp-config flag
	}
	// ... existing opencode.jsonc logic ...
}
```

**Step 2: Run tests**

Run: `cd /Users/smankowski/github/sgai && go test ./cmd/sgai/ -run TestApplyCustomMCPs -v`

**Step 3: Commit**

```bash
git add cmd/sgai/config.go
git commit -m "feat: guard applyCustomMCPs behind opencode backend check"
```

---

### Task 9: Integration testing

**Files:**
- Modify: `cmd/sgai/backend_test.go`

**Step 1: Write integration test that verifies backend selection**

```go
func TestBackendSelection(t *testing.T) {
	t.Run("defaultBackend", func(t *testing.T) {
		b := resolveBackend(nil)
		if b.Name() != "opencode" {
			t.Errorf("nil config should default to opencode, got %s", b.Name())
		}
	})
	t.Run("explicitOpencode", func(t *testing.T) {
		b := resolveBackend(&projectConfig{Backend: "opencode"})
		if b.Name() != "opencode" {
			t.Errorf("expected opencode, got %s", b.Name())
		}
	})
	t.Run("claudeCode", func(t *testing.T) {
		b := resolveBackend(&projectConfig{Backend: "claude-code"})
		if b.Name() != "claude-code" {
			t.Errorf("expected claude-code, got %s", b.Name())
		}
	})
}
```

**Step 2: Write test verifying both backends produce valid args**

Test that args from both backends have no nil/empty required elements, that session IDs are correctly placed, etc.

**Step 3: Run full test suite**

Run: `cd /Users/smankowski/github/sgai && go test ./... -v -count=1`
Expected: ALL PASS

**Step 4: Commit**

```bash
git add cmd/sgai/backend_test.go
git commit -m "test: add integration tests for backend selection"
```

---

## Summary of files changed

| File | Action | Purpose |
|------|--------|---------|
| `cmd/sgai/backend.go` | Create | `Backend` interface + types |
| `cmd/sgai/backend_opencode.go` | Create | opencode implementation (extracted from main.go) |
| `cmd/sgai/backend_claudecode.go` | Create | Claude Code implementation |
| `cmd/sgai/backend_claudecode_mcp.go` | Create | MCP config generation for Claude Code |
| `cmd/sgai/backend_test.go` | Create | Interface + integration tests |
| `cmd/sgai/backend_opencode_test.go` | Create | opencode backend tests |
| `cmd/sgai/backend_claudecode_test.go` | Create | Claude Code backend tests |
| `cmd/sgai/backend_claudecode_mcp_test.go` | Create | MCP config tests |
| `cmd/sgai/config.go` | Modify | Add `Backend` field, `resolveBackend()`, guard `applyCustomMCPs` |
| `cmd/sgai/config_test.go` | Modify | Test backend config |
| `cmd/sgai/main.go` | Modify | Thread backend through multiModelConfig, replace hardcoded exec calls |
| `cmd/sgai/serve_api.go` | Modify | Replace adhoc opencode exec |
| `cmd/sgai/service_adhoc.go` | Modify | Replace adhoc opencode exec |
| `cmd/sgai/continuous.go` | Modify | Replace continuous mode opencode exec |

## Usage

After implementation, users configure their project's `sgai.json`:

```json
{
  "backend": "claude-code",
  "defaultModel": "anthropic/claude-opus-4-6 (max)"
}
```

Or stick with the default:

```json
{
  "defaultModel": "anthropic/claude-opus-4-6 (max)"
}
```

(Defaults to opencode when `"backend"` is omitted.)
