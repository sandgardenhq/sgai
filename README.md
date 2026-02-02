# Sandgarden AI Software Factory

Define your goals in `GOAL.md`, launch the web dashboard, and watch AI agents work together to build your software. Monitor progress in real-time, provide guidance when needed, and iterate until your goals are achieved.

## Features

- **Web dashboard** — Monitor and control agent execution via HTMX + PicoCSS web UI with real-time status visualization, start/stop controls, and an interactive interface for agent questions
- **Multi-agent orchestration** — DOT-format directed acyclic graphs, inter-agent messaging, coordinator pattern for delegation
- **GOAL.md-driven development** — Define what you want to build, not how; the AI agents figure out the implementation
- **Human-in-the-loop** — Interactive mode for when agents need clarification (web UI or terminal)
- **MCP server** — Exposes workflow management tools (state updates, messaging, skills, snippets) to AI agents
- **Retrospective system** — Analyze completed sessions, extract reusable skills and snippets
- **Multi-model support** — Assign different AI models per agent role, run multiple models concurrently
- **Go-native** — Single binary, fast startup, minimal dependencies

## Prerequisites

| Dependency                                   | Purpose                                                                   |                           |
|----------------------------------------------|---------------------------------------------------------------------------|---------------------------|
| [opencode](https://opencode.ai)              | AI inference engine — executes agents, validates models, exports sessions |                           |
| [jj](https://docs.jj-vcs.dev/) (Jujutsu)     | VCS integration in web UI (diffs, logs, workspace forking)                |                           |
| [dot](https://graphviz.org/) (Graphviz)      | Renders workflow DAG as proper SVG                                        | Plain-text SVG fallback   |

### Environment Variables

| Variable        | Purpose                                                                         |
|-----------------|---------------------------------------------------------------------------------|
| `SGAI_NTFY`     | URL for [ntfy](https://ntfy.sh) push notifications (optional remote alerting)   |

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

3. **Create a `GOAL.md` in your project directory:**

   ```markdown
   ---
   completionGateScript: make test
   flow: |
     "general-purpose"
     "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
   interactive: yes
   ---

   # My Project Goal

   Build a REST API with user authentication.

   ## Tasks

   - [ ] Create user registration endpoint
   - [ ] Create login endpoint with JWT
   - [ ] Add password hashing
   ```

4. **Launch the web dashboard:**

   ```sh
   sgai serve
   ```

   Open [http://localhost:8080](http://localhost:8080) in your browser to monitor and control the workflow.

## How It Works

```
GOAL.md → sgai serve → Monitor in Browser → Iterate
```

1. **Define your goals** — Write a `GOAL.md` file describing what you want to build
2. **Launch the dashboard** — Run `sgai serve` to start the web interface
3. **Monitor progress** — Watch agents work in real-time, see diffs, logs, and status updates
4. **Provide guidance** — When agents need clarification, respond to their questions through the web UI
5. **Iterate** — Review results, update goals, and continue until satisfied

The web dashboard shows:
- Real-time workflow status and agent activity
- SVG visualization of the agent DAG
- Start/Stop controls for the engine
- Session management and retrospective browsing
- Goal editing and agent/skill/snippet listing
- Interface for answering agent questions (multiple-choice prompts)

## GOAL.md Reference

Create a `GOAL.md` file in your project directory to define your goals:

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

### Frontmatter Options

| Field         | Description                                              |
|---------------|----------------------------------------------------------|
| `flow`        | DOT-format DAG defining agent execution order            |
| `models`      | Per-agent AI model assignments (supports variant syntax) |
| `completionGateScript`   | Shell command that determines workflow completion        |
| `interactive` | `yes` (respond via web UI), `no` (exit when agent asks a question), `auto` (self-driving) |

## Usage

```sh
sgai serve                              # Start on localhost:8080
sgai serve --listen-addr 0.0.0.0:8080   # Start accessible externally
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
