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
