# Sandgarden AI Software Factory

sgai is an AI software factory: a local service that orchestrates multiple specialized AI agents from a single goal description.
Define what you want in `GOAL.md`, start sgai, and use the web dashboard to monitor progress, answer questions, and iterate.

## Key Concepts

| Concept | What it is | Why it matters |
|---|---|---|
| The factory | sgai coordinating multiple agents (not a single “chatbot”) | Work gets split into roles (build, review, analysis) and coordinated in a repeatable way |
| `GOAL.md` | The intent/spec for a project (“what”, not “how”) | Gives agents a durable source of truth to follow and update against |
| Agents | Specialized roles (developer, reviewer, analyst, etc.) | Improves quality by separating implementation and review responsibilities |
| Flow | A directed graph describing who reviews whom (`->`) | Makes review and safety/quality checks explicit |
| Human-in-the-loop | You supervise, clarify, and unblock | Prevents agents from guessing when requirements are ambiguous |
| Coordinator | The “foreman” agent that reads `GOAL.md` and delegates | Keeps the work moving and assigns tasks to the right roles |

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

## Automated Setup with opencode

The fastest way to install SGAI is with [opencode](https://opencode.ai).

**Before you begin:**

1. **opencode must be up-to-date.** Update it with `opencode update` or reinstall from [opencode.ai/docs/#install](https://opencode.ai/docs/#install).
2. **You must be authenticated to Claude.** Run `opencode auth login`, select **Anthropic** → **Claude Pro/Max**, and complete the OAuth flow.

Then run:

```sh
opencode --model anthropic/claude-opus-4-6 run "install SGAI using the instructions from https://github.com/sandgardenhq/sgai/blob/main/INSTALLATION.md"
```

This reads [`INSTALLATION.md`](INSTALLATION.md) and automatically detects your platform, installs dependencies, sets up a demo workspace, and starts the dashboard. See [INSTALLATION.md](INSTALLATION.md) for the full step-by-step instructions.

### Environment Variables

| Variable        | Purpose                                                                         |
|-----------------|---------------------------------------------------------------------------------|
| `SGAI_NTFY`     | URL for [ntfy](https://ntfy.sh) push notifications (optional remote alerting)   |

## Quick Start (macOS)

1. **Install dependencies via Homebrew:**

   ```sh
   brew install node anomalyco/tap/opencode jj graphviz oven-sh/bun/bun tmux ripgrep
   ```

2. **Log in to your AI provider:**

   ```sh
   opencode auth login
   ```

   Select **Anthropic** → **Claude Pro/Max** and complete the OAuth flow in your browser.
   For other providers or advanced configuration, see the [providers documentation](https://opencode.ai/docs/providers/).

3. **Create a `GOAL.md` in your project directory:**

   ```markdown
   # My Project Goal

   Build a REST API with user authentication.

   ## Tasks

   - [ ] Create user registration endpoint
   - [ ] Create login endpoint with JWT
   - [ ] Add password hashing
   ```

   This minimal `GOAL.md` uses default settings. See [GOAL.md Reference](#goalmd-reference) for advanced options like custom flows and model assignments.

4. **Launch the web dashboard:**

   ```sh
   sgai
   ```

   Open [http://localhost:8080](http://localhost:8080) in your browser to monitor and control the workflow.

   Run `sgai` from the parent directory that contains your projects. sgai discovers `GOAL.md` files in subdirectories automatically.

   ```
   workspace/              ← Run `sgai` from here
   ├── project-1/
   │   └── GOAL.md
   ├── project-2/
   │   └── GOAL.md
   └── project-3/
       └── GOAL.md
   ```

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

## Features

- **Web dashboard** — Monitor and control agent execution via React SPA with real-time SSE updates, start/stop controls, and human-in-the-loop response interface
- **Multi-agent orchestration** — DOT-format directed acyclic graphs, inter-agent messaging, coordinator pattern for delegation
- **GOAL.md-driven development** — Define what you want to build, not how; the AI agents figure out the implementation
- **Human-in-the-loop** — Interactive mode for when agents need clarification (web UI or terminal)
- **MCP server** — Exposes workflow management tools (state updates, messaging, skills, snippets) to AI agents
- **Retrospective system** — Analyze completed sessions, extract reusable skills and snippets
- **Multi-model support** — Assign different AI models per agent role, run multiple models concurrently
- **Go-native** — Single binary, fast startup, minimal dependencies

## Prerequisites

| Dependency | Purpose | Required? |
|---|---|---|
| [Node.js](https://nodejs.org) | JavaScript runtime — provides `npx` for MCP server auto-installation | Required |
| [bun](https://bun.sh) | JavaScript runtime and bundler — builds the React frontend | Required (build only) |
| [opencode](https://opencode.ai) | AI inference engine — executes agents, validates models, exports sessions | Required |
| [jj](https://docs.jj-vcs.dev/) (Jujutsu) | VCS integration in web UI (diffs, logs, workspace forking) | Required |
| [dot](https://graphviz.org/) (Graphviz) | Renders workflow DAG as proper SVG | Optional — plain-text SVG fallback |
| [gh](https://cli.github.com/) (GitHub CLI) | Creates draft PRs from fork merge flow | Optional — merge works without PR creation |
| [tmux](https://github.com/tmux/tmux) | Terminal multiplexer — manages detached sessions for agent processes | Required |
| [rg](https://github.com/BurntSushi/ripgrep) (ripgrep) | Fast text search — used by completion verification and code search skills | Required |

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

## Common Flows

Use these as ready-to-copy starting points for the `flow` frontmatter field.

### Go Backend Only

```yaml
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
```

### Go Backend with Safety Analysis

```yaml
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "go-readability-reviewer" -> "stpa-analyst"
```

### Full-Stack Go + HTMX/PicoCSS

```yaml
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "go-readability-reviewer" -> "stpa-analyst"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "htmx-picocss-frontend-reviewer" -> "stpa-analyst"
```

### Full-Stack Go + React

```yaml
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "go-readability-reviewer" -> "stpa-analyst"
  "react-developer" -> "react-reviewer"
  "react-reviewer" -> "stpa-analyst"
```

### Research / Exploration

```yaml
flow: |
  "general-purpose"
```

## Usage

```sh
sgai                                # Start on localhost:8080
sgai --listen-addr 0.0.0.0:8080    # Start accessible externally
```

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
