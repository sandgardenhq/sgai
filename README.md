# Sandgarden AI Software Factory

## Features

- **Outer-loop driven development** — `GOAL.md` is the source of truth; define a DAG of agents to loop until the goal is achieved
- **Multi-agent orchestration** — DOT-format directed acyclic graphs, inter-agent messaging, coordinator pattern for delegation
- **MCP server** — Exposes workflow management tools (state updates, messaging, skills, snippets) to AI agents
- **Web dashboard** — Monitor and control agent execution via HTMX + PicoCSS web UI
- **Retrospective system** — Analyze completed sessions, extract reusable skills and snippets
- **Multi-model support** — Assign different AI models per agent role, run multiple models concurrently
- **Human-in-the-loop** — Interactive mode for when agents need clarification (terminal or web UI)
- **Go-native** — Single binary, fast startup, minimal dependencies

## Prerequisites

| Dependency                                   | Purpose                                                                   |                           |
|----------------------------------------------|---------------------------------------------------------------------------|---------------------------|
| [opencode](https://opencode.ai)              | AI inference engine — executes agents, validates models, exports sessions |                           |
| [jj](https://docs.jj-vcs.dev/) (Jujutsu) | VCS integration in web UI (diffs, logs, workspace forking)                |                           |
| [dot](https://graphviz.org/) (Graphviz)      | Renders workflow DAG as proper SVG                                        | Plain-text SVG fallback   |
| `$EDITOR` (e.g. `vim`, `nano`)               | Human response editing in interactive terminal mode                       | Raw terminal input        |

### Environment Variables

| Variable        | Purpose                                                                         |
|-----------------|---------------------------------------------------------------------------------|
| `EDITOR`        | When set, defaults interactive mode to `yes` (opens editor for human responses) |
| `sgai_NTFY` | URL for [ntfy](https://ntfy.sh) push notifications (optional remote alerting)   |

## Installation

```sh
go install github.com/sandgardenhq/sgai/cmd/sgai@latest
```

Or from source:

```sh
git clone https://github.com/sandgardenhq/sgai.git
cd sgai
make build
```

## Quick Start (macOS)

1. **Install dependencies via Homebrew:**

   ```sh
   brew install anomalyco/tap/opencode jj graphviz
   ```

2. **Log in to your AI provider:**

   ```sh
   opencode auth login
   ```

   Select **Anthropic** → **Claude Pro/Max** and complete the OAuth flow in your browser.
   For other providers or advanced configuration, see the [providers documentation](https://opencode.ai/docs/providers/).

3. **Run sgai in your project:**

   ```sh
   sgai .
   ```

   Create a `GOAL.md` file to define what you want to build (see [Usage](#usage) below).

## Usage

Create a `GOAL.md` in your project directory:

```markdown
---
completionGateScript: make test-go
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "go-readability-reviewer" -> "stpa-analyst"
models:
  "coordinator": "anthropic/claude-opus-4-5"
  "backend-go-developer": "anthropic/claude-sonnet-4-5"
interactive: yes
---

# My Project Goal

Describe what you want to build here. Be specific about behavior,
not implementation. Focus on outcomes.

## Requirements

- What should happen when a user does X?
- What constraints exist?
- What does success look like?

## Tasks

- [ ] Task 1
- [ ] Task 2
  - [ ] Task 2.1
- [ ] Task 3
- [ ] Task 4
```

Run sgai:

```sh
sgai .                    # Run workflow in current directory
sgai <PATH>               # Run workflow in specified directory
sgai --fresh .            # Start fresh, don't resume existing workflow
sgai --interactive=no .   # Non-interactive (exit on human-communication)
```

### GOAL.md Frontmatter

| Field         | Description                                              |
|---------------|----------------------------------------------------------|
| `flow`        | DOT-format DAG defining agent execution order            |
| `models`      | Per-agent AI model assignments (supports variant syntax) |
| `completionGateScript`   | Shell command that determines workflow completion        |
| `interactive` | `yes` (open $EDITOR), `no` (exit), `auto` (self-driving) |

# Sandgarden AI Software Factory

## Features

- **Outer-loop driven development** — `GOAL.md` is the source of truth; define a DAG of agents to loop until the goal is achieved
- **Multi-agent orchestration** — DOT-format directed acyclic graphs, inter-agent messaging, coordinator pattern for delegation
- **MCP server** — Exposes workflow management tools (state updates, messaging, skills, snippets) to AI agents
- **Web dashboard** — Monitor and control agent execution via HTMX + PicoCSS web UI
- **Retrospective system** — Analyze completed sessions, extract reusable skills and snippets
- **Multi-model support** — Assign different AI models per agent role, run multiple models concurrently
- **Human-in-the-loop** — Interactive mode for when agents need clarification (terminal or web UI)
- **Go-native** — Single binary, fast startup, minimal dependencies

## Prerequisites

| Dependency                                   | Purpose                                                                   |                           |
|----------------------------------------------|---------------------------------------------------------------------------|---------------------------|
| [opencode](https://opencode.ai)              | AI inference engine — executes agents, validates models, exports sessions |                           |
| [jj](https://docs.jj-vcs.dev/) (Jujutsu) | VCS integration in web UI (diffs, logs, workspace forking)                |                           |
| [dot](https://graphviz.org/) (Graphviz)      | Renders workflow DAG as proper SVG                                        | Plain-text SVG fallback   |
| `$EDITOR` (e.g. `vim`, `nano`)               | Human response editing in interactive terminal mode                       | Raw terminal input        |

### Environment Variables

| Variable        | Purpose                                                                         |
|-----------------|---------------------------------------------------------------------------------|
| `EDITOR`        | When set, defaults interactive mode to `yes` (opens editor for human responses) |
| `sgai_NTFY` | URL for [ntfy](https://ntfy.sh) push notifications (optional remote alerting)   |

## Installation

```sh
go install github.com/sandgardenhq/sgai/cmd/sgai@latest
```

Or from source:

```sh
git clone https://github.com/sandgardenhq/sgai.git
cd sgai
make build
```

## Quick Start (macOS)

1. **Install dependencies via Homebrew:**

   ```sh
   brew install anomalyco/tap/opencode jj graphviz
   ```

2. **Log in to your AI provider:**

   ```sh
   opencode auth login
   ```

   Select **Anthropic** → **Claude Pro/Max** and complete the OAuth flow in your browser.
   For other providers or advanced configuration, see the [providers documentation](https://opencode.ai/docs/providers/).

3. **Run sgai in your project:**

   ```sh
   sgai .
   ```

   Create a `GOAL.md` file to define what you want to build (see [Usage](#usage) below).

## Usage

Create a `GOAL.md` in your project directory:

```markdown
---
completionGateScript: make test-go
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "go-readability-reviewer" -> "stpa-analyst"
models:
  "coordinator": "anthropic/claude-opus-4-5"
  "backend-go-developer": "anthropic/claude-sonnet-4-5"
interactive: yes
---

# My Project Goal

Describe what you want to build here. Be specific about behavior,
not implementation. Focus on outcomes.

## Requirements

- What should happen when a user does X?
- What constraints exist?
- What does success look like?

## Tasks

- [ ] Task 1
- [ ] Task 2
  - [ ] Task 2.1
- [ ] Task 3
- [ ] Task 4
```

Run sgai:

```sh
sgai .                    # Run workflow in current directory
sgai <PATH>               # Run workflow in specified directory
sgai --fresh .            # Start fresh, don't resume existing workflow
sgai --interactive=no .   # Non-interactive (exit on human-communication)
```

### GOAL.md Frontmatter

| Field         | Description                                              |
|---------------|----------------------------------------------------------|
| `flow`        | DOT-format DAG defining agent execution order            |
| `models`      | Per-agent AI model assignments (supports variant syntax) |
| `completionGateScript`   | Shell command that determines workflow completion        |
| `interactive` | `yes` (open $EDITOR), `no` (exit), `auto` (self-driving) |

### Workflow status values

Ready to drive workflow state from tools (for example, via the MCP workflow state update tool)? Here’s the key rule:

- Only the `coordinator` agent can set workflow `status` to `complete` or `human-communication`.
- Other agents can set workflow `status` to `working` or `agent-done`.

This restriction is enforced by the workflow state update schema, so non-coordinator agents do not get `complete` or `human-communication` as valid `status` enum values.

### Web Dashboard

sgai includes a web dashboard for monitoring and controlling workflow execution:

```sh
sgai serve                              # Start on localhost:8080
sgai serve --listen-addr 0.0.0.0:8080   # Start accessible externally
```

The dashboard provides:
- Real-time workflow status visualization
- Start/Stop controls for the engine
- SVG visualization of the agent DAG
- Session management and retrospective browsing
- Goal editing and agent/skill/snippet listing
- Human-communication response interface

### Other Commands

```sh
sgai sessions                             # List all sessions
sgai status [target_directory]            # Show workflow status summary
sgai retrospective analyze [session-id]   # Analyze a session
sgai retrospective apply <session-id>     # Apply improvements from a session
sgai list-agents [target_directory]       # List available agents
```

## Contributing

Contributions happen through specifications, not code.

**Why specification files instead of code?**

sgai uses configurable AI engines under the hood, but it's the opinionated experience layer. Specifications are translated into implementation by AI. Source code is generated output, not the source of truth. Contributing specs means:

- We discuss *what* to build, not *how* to build it
- Conversations lead to better outcomes than isolated code changes
- Maintainers can validate proposals against the current implementation

**How to contribute:**

1. Create a spec file in `GOALS/` following the naming convention:
   `YYYY_MM_DD_summarized_name.md` (e.g., `2025_12_23_add_parallel_execution.md`)

2. Submit a PR with your spec proposal

3. Maintainers will discuss the proposal and, if accepted, run the specification against the current implementation to validate

All are welcome. Questions? Open an issue.


sgai includes a web dashboard for monitoring and controlling workflow execution:

```sh
sgai serve                              # Start on localhost:8080
sgai serve --listen-addr 0.0.0.0:8080   # Start accessible externally
```

The dashboard provides:
- Real-time workflow status visualization
- Start/Stop controls for the engine
- SVG visualization of the agent DAG
- Session management and retrospective browsing
- Goal editing and agent/skill/snippet listing
- Human-communication response interface

### Other Commands

```sh
sgai sessions                             # List all sessions
sgai status [target_directory]            # Show workflow status summary
sgai retrospective analyze [session-id]   # Analyze a session
sgai retrospective apply <session-id>     # Apply improvements from a session
sgai list-agents [target_directory]       # List available agents
```

## Contributing

Contributions happen through specifications, not code.

**Why specification files instead of code?**

sgai uses configurable AI engines under the hood, but it's the opinionated experience layer. Specifications are translated into implementation by AI. Source code is generated output, not the source of truth. Contributing specs means:

- We discuss *what* to build, not *how* to build it
- Conversations lead to better outcomes than isolated code changes
- Maintainers can validate proposals against the current implementation

**How to contribute:**

1. Create a spec file in `GOALS/` following the naming convention:
   `YYYY_MM_DD_summarized_name.md` (e.g., `2025_12_23_add_parallel_execution.md`)

2. Submit a PR with your spec proposal

3. Maintainers will discuss the proposal and, if accepted, run the specification against the current implementation to validate

All are welcome. Questions? Open an issue.
