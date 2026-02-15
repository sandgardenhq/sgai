# Sandgarden AI Software Factory

Define your goals in `GOAL.md`, launch the web dashboard, and watch AI agents work together to build your software. Monitor progress in real-time, provide guidance when needed, and iterate until your goals are achieved.

## Features

- **Web dashboard** — Monitor and control agent execution via React SPA with real-time SSE updates, start/stop controls, and human-in-the-loop response interface
- **Multi-agent orchestration** — DOT-format directed acyclic graphs, inter-agent messaging, coordinator pattern for delegation
- **GOAL.md-driven development** — Define what you want to build, not how; the AI agents figure out the implementation
- **Human-in-the-loop** — Interactive mode for when agents need clarification (web UI)
- **MCP server** — Exposes workflow management tools (state updates, messaging, skills, snippets) to AI agents
- **Retrospective system** — Analyze completed sessions, extract reusable skills and snippets
- **Multi-model support** — Assign different AI models per agent role, run multiple models concurrently
- **Go-native** — Single binary, fast startup, minimal dependencies

## Prerequisites

| Dependency                                   | Purpose                                                                   |                           |
|----------------------------------------------|---------------------------------------------------------------------------|---------------------------|
| [Node.js](https://nodejs.org)                | JavaScript runtime — provides `npx` for MCP server auto-installation      |                           |
| [bun](https://bun.sh)                        | JavaScript runtime and bundler — builds the React frontend                |                           |
| [opencode](https://opencode.ai)              | AI inference engine — executes agents, validates models, exports sessions |                           |
| [jj](https://docs.jj-vcs.dev/) (Jujutsu)     | VCS integration in web UI (diffs, logs, workspace forking)                |                           |
| [dot](https://graphviz.org/) (Graphviz)      | Renders workflow DAG as proper SVG                                        | Plain-text SVG fallback   |
| [gh](https://cli.github.com/) (GitHub CLI)   | Creates draft PRs from fork merge flow                                    | Optional — merge works without PR creation |

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
cd cmd/sgai/webapp && bun install && cd ../../..
make build
```

## Quick Start (macOS)

1. **Install dependencies via Homebrew:**

   ```sh
   brew install node anomalyco/tap/opencode jj graphviz oven-sh/bun/bun
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
   sgai
   ```

   Open [http://localhost:8080](http://localhost:8080) in your browser to monitor and control the workflow.

## How It Works

```
GOAL.md → sgai → Monitor in Browser → Iterate
```

1. **Define your goals** — Write a `GOAL.md` file describing what you want to build
2. **Launch the dashboard** — Run `sgai` to start the web interface
3. **Monitor progress** — Watch agents work in real-time, see diffs, logs, and status updates
4. **Provide guidance** — When agents need clarification, respond through the web UI
5. **Iterate** — Review results, update goals, and continue until satisfied

The web dashboard shows:
- Real-time workflow status and agent activity
- SVG visualization of the agent DAG
- Start/Stop controls for the engine
- Session management and retrospective browsing
- Goal editing and agent/skill/snippet listing
- Human response interface for agent questions

## GOAL.md Reference

Create a `GOAL.md` file in your project directory to define your goals:

```markdown
---
completionGateScript: make test-go
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "go-readability-reviewer" -> "stpa-analyst"
models:
  "coordinator": "anthropic/claude-opus-4-6"
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
sgai                              # Start on localhost:8080
sgai --listen-addr 0.0.0.0:8080   # Start accessible externally
```

## MCP transport

SGAI exposes its Model Context Protocol (MCP) tools over HTTP.
For details on the HTTP endpoint and agent identity header, see [MCP remote HTTP transport](./docs/reference/mcp-remote-http.md).

## Frontend Development

The web dashboard is a React SPA in `cmd/sgai/webapp/`. Built artifacts are embedded in the Go binary via `//go:embed`.

### Frontend Stack

| Technology                                    | Purpose                                 |
|-----------------------------------------------|-----------------------------------------|
| [React 19](https://react.dev)                 | UI framework                            |
| [TypeScript](https://www.typescriptlang.org)  | Type-safe JavaScript                    |
| [Tailwind CSS v4](https://tailwindcss.com)    | Utility-first CSS                       |
| [shadcn/ui](https://ui.shadcn.com) + Radix UI | Accessible component library            |
| [React Router](https://reactrouter.com)       | Client-side routing                     |
| [Lucide React](https://lucide.dev)            | Icons                                   |

### Build Commands

```sh
cd cmd/sgai/webapp

bun install          # Install frontend dependencies
bun run build        # Production build → dist/
bun run dev.ts       # Dev server with file watching (proxies API to Go backend)
bun test src/        # Run unit/component tests
```

`make build` runs the full pipeline: frontend build (`bun install` + `bun run build`) → Go lint → Go binary.

After making frontend changes, always run:

```sh
bun run build && make build
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
