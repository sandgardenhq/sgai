# Sandgarden AI Software Factory

Define a goal in `GOAL.md`, launch the web dashboard, and watch AI agents build software while you monitor progress and iterate until the goal is achieved.

## Features

- **Web dashboard** — Visualize real-time status, start/stop execution, and respond to agent questions through a human-in-the-loop interface.
- **GOAL.md-driven development** — Specify *what* to build in `GOAL.md`; agents decide *how* to implement it.
- **Multi-agent orchestration** — Use a DAG to coordinate multiple agents.
- **MCP server** — Expose workflow management tools (state updates, messaging, skills, snippets) to AI agents.
- **Retrospective system** — Analyze completed sessions and extract reusable skills and snippets.
- **Multi-model support** — Assign different AI models per agent role.
- **Human-in-the-loop** — Provide interactive clarification through the web UI or terminal.
- **Go-native** — Run a single Go binary with minimal dependencies.

## Prerequisites

| Dependency                               | Purpose                                                          | Notes               |
|------------------------------------------|------------------------------------------------------------------|---------------------|
| [opencode](https://opencode.ai)          | AI inference engine                                              |                     |
| [jj](https://docs.jj-vcs.dev/) (Jujutsu) | VCS integration in the web UI (diffs, logs, workspace forking)   |                     |
| [dot](https://graphviz.org/) (Graphviz)  | Render the workflow DAG as SVG                                   | Plain-text fallback |

### Environment Variables

| Variable    | Purpose                                                       |
|------------|---------------------------------------------------------------|
| `SGAI_NTFY` | URL for [ntfy](https://ntfy.sh) push notifications (optional) |

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

3. **Create a `GOAL.md` in your project directory:**

   ```markdown
   ---
   completionGateScript: make test
   flow: |
     "general" -> "frontend-developer"
     "frontend-developer" -> "frontend-reviewer"
   interactive: yes
   ---

   # My Project Goal

   Build a REST API with user authentication.

   ## Tasks

   - [ ] Create endpoints
   - [ ] Implement password hashing
   ```

4. **Launch the web dashboard:**

   ```sh
   sgai serve
   ```

   Open `http://localhost:8080` in your browser to monitor and control the workflow.

## How It Works

`GOAL.md` → `sgai serve` → Monitor in Browser → Iterate

1. Define goals in `GOAL.md`.
2. Launch the dashboard with `sgai serve`.
3. Monitor progress (diffs, logs, status updates).
4. Provide guidance through the web UI.
5. Iterate by updating your goals.

The dashboard shows:

- Real-time workflow status and agent activity
- SVG DAG visualization
- Start/stop controls
- Session management and retrospective browsing
- Goal editing and agent/skill/snippet listing
- A human-communication response interface

## GOAL.md Reference

Create a `GOAL.md` file in your project directory to define your goals.

### Frontmatter Options

| Field                  | Description                                                        |
|------------------------|--------------------------------------------------------------------|
| `flow`                 | DOT-format DAG defining agent execution order                      |
| `models`               | Per-agent AI model assignments                                     |
| `completionGateScript` | Shell command that determines workflow completion                  |
| `interactive`          | `yes` (respond via web UI), `no` (exit), `auto` (self-driving)     |

## Contributing

Contributions happen through specifications, not code.

1. Create a spec file in `GOALS/` following the naming convention `YYYY_MM_DD_summarized_name.md`.
2. Submit a PR with the spec proposal.
3. Maintainers will discuss the proposal and validate against the current implementation.

- Maintainers can validate proposals against the current implementation

**How to contribute:**

1. Create a spec file in `GOALS/` following the naming convention:
   `YYYY_MM_DD_summarized_name.md` (e.g., `2025_12_23_add_parallel_execution.md`)

2. Submit a PR with your spec proposal

3. Maintainers will discuss the proposal and, if accepted, run the specification against the current implementation to validate

All are welcome. Questions? Open an issue.
