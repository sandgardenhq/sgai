<img src="https://raw.githubusercontent.com/sandgardenhq/sgai/refs/heads/main/assets/images/logo.png" alt="Sgai Logo" width="200">

# Sgai (pronounced "Sky") - Goal-Driven AI Software Factory

Define your goal. Launch the dashboard.
Watch AI agents plan, execute, and validate your software — with you in control.

**Example:** "Build a drag-and-drop image compressor" → 3 agents (developer, reviewer, designer) → Working app with tests passing → 45 minutes.

**📺 [Watch the 4-minute demo →](https://youtu.be/NYmjhwLUg8Q)**

<img style="margin:20px 0;border:1px solid #999;" src="https://github.com/sandgardenhq/sgai/blob/main/assets/screenshots/08-Workspace.png?raw=true" alt="Sgai Dashboard" width="800">

---

## What Is Sgai?

Sgai turns software development into a **goal-driven, multi-agent workflow**.

Instead of prompting step-by-step, you:

1. **Define the outcome** — "Build a music sequencer web app"
2. **Agents plan the work** — Breaking it into a visual workflow diagram of tasks
3. **You supervise** — Watch progress, answer questions when agents need guidance
4. **Success checks** — Tests, linting, or other validation determines "done"

Not autocomplete. Not a chat window.
A local AI software factory.

---

## Why Try It?

* **See what's happening** — Visual workflow diagram instead of hidden AI reasoning
* **Multiple specialists** — Developer writes code, reviewer checks it, safety analyst validates
* **Approve before execution** — Review the plan and answer questions, then agents work autonomously
* **Proof of completion** — Tests must pass before work is marked done
* **Works locally** — Runs in your repository, nothing leaves your machine

---

## Quick Start

### Recommended: Automated Setup via opencode

```bash
opencode upgrade
opencode auth login
opencode --model anthropic/claude-opus-4-6 run "install Sgai using the instructions from https://github.com/sandgardenhq/sgai/blob/main/INSTALLATION.md"
```

This runs the official installation guide automatically and launches a demo workspace.

---

### Manual Installation

**Required:** Go, Node.js, bun, opencode

**Recommended:** jj (version control), tmux (session management), ripgrep (code search), Graphviz (diagram rendering)

```bash
go install github.com/sandgardenhq/sgai/cmd/sgai@latest
```

Or build from source:

```bash
git clone https://github.com/sandgardenhq/sgai.git
cd sgai
cd cmd/sgai/webapp && bun install && cd ../../..
make build
```

See [INSTALLATION.md](https://github.com/sandgardenhq/sgai/blob/main/INSTALLATION.md) for details.

---

## Run It

```bash
sgai serve
```

Open: [http://localhost:8080](http://localhost:8080)

---

## How It Works

**📺 Prefer watching? See the demo → [https://youtu.be/NYmjhwLUg8Q](https://youtu.be/NYmjhwLUg8Q)**

### 1. Create a Goal

Most users create goals using the built-in wizard.

Goals are stored in `GOAL.md` and describe outcomes — not implementation steps.

**Example GOAL.md:**

```markdown
---
flow: |
  "backend-developer" -> "code-reviewer"
completionGateScript: make test
interactive: yes
---

# Build a REST API

Create endpoints for user registration and login with JWT auth.

- [ ] POST /register validates email, hashes password
- [ ] POST /login returns JWT token
- [ ] Tests pass before completion
```

See [GOAL.example.md](GOAL.example.md) for full reference.

### 2. Agents Plan the Work

<img style="margin:20px 0;border:1px solid #999;" src="https://github.com/sandgardenhq/sgai/blob/main/assets/screenshots/02-ChooseATemplate.png?raw=true" alt="Choose a Template" width="600">

Sgai breaks your goal into a workflow diagram of coordinated agents with defined roles.

Dependencies are explicit. Execution is visible.

### 3. Approve the Plan & Monitor

<img style="margin:20px 0;border:1px solid #999;" src="https://github.com/sandgardenhq/sgai/blob/main/assets/screenshots/09-Questions.png?raw=true" alt="Agent Questions" width="600">

Before execution begins, agents ask clarifying questions about your goal.

Once you approve the plan, agents work autonomously — executing tasks, running tests, and validating completion.

You can:

* Monitor real-time progress (optional)
* Interrupt execution if needed
* Review diffs and session history
* Fork sessions to try different approaches

Most of the time, you approve the plan and come back when it's done.

### 4. Learn from Past Sessions with _Skills_

<img style="margin:20px 0;border:1px solid #999;" src="https://github.com/sandgardenhq/sgai/blob/main/assets/screenshots/11-Skills.png?raw=true" alt="Skills Library" width="500">

Sgai extracts reusable skills and code snippets from completed sessions — your agents get smarter over time.

---

## Drive Sgai from Your AI Harness

Sgai exposes two integration paths for AI agents and harnesses — MCP tools and HTTP skills — so Claude Code, Codex, or any MCP-capable assistant can orchestrate Sgai programmatically.

### MCP Interface

When you run `sgai serve`, the MCP endpoint is available on the same port as the web UI:

```
sgai serve listening on http://127.0.0.1:8080
```

The MCP endpoint is at `/mcp/external` on the main server. Connect any MCP-capable harness to it:

```bash
npx mcporter list --http-url http://127.0.0.1:8080/mcp/external --allow-http
```

**Configure in [OpenCode](https://opencode.ai/docs/mcp-servers/):**

```jsonc
// opencode.jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "sgai": {
      "type": "remote",
      "url": "http://127.0.0.1:8080/mcp/external"
    }
  }
}
```

Replace `8080` with the actual port if you use a custom `--listen-addr`.

35+ tools mirror the full web UI — workspace lifecycle, session control, human interaction, monitoring, knowledge, compose, and adhoc. Key tools:

| Tool | What it does |
|------|-------------|
| `list_workspaces` | Discover all workspaces and their status |
| `start_session` | Launch an agent session (with optional auto-drive mode) |
| `respond_to_question` | Answer a pending agent question |
| `wait_for_question` | Block until an agent needs human input (MCP elicitation) |

### Skills / HTTP API

Sgai also ships a set of [agentskills.io](https://agentskills.io/specification)-conformant skills for harnesses that prefer plain HTTP.

**Entrypoint:** [`docs/sgai-skills/using-sgai/SKILL.md`](docs/sgai-skills/using-sgai/SKILL.md)

The core pattern is a cyclical probe/poll/act loop:

```
LOOP:
  1. PROBE  → GET /api/v1/state          # Discover workspaces + status
  2. CHECK  → pendingQuestion != null?   # Does any workspace need input?
  3. ACT    → start, steer, or respond   # Take action based on state
  4. WAIT   → poll again after delay
```

Example probe:

```bash
curl -s http://127.0.0.1:8080/api/v1/state | jq '.workspaces[0].pendingQuestion'
```

Full reference in [`docs/sgai-skills/`](docs/sgai-skills/) — seven sub-skills covering workspace-management, session-control, human-interaction, monitoring, knowledge, compose, and adhoc.

---

## What Happens to Your Code?

* Agents operate inside your local repository
* Changes go through your version control (we recommend jj, but Git works)
* Sgai does not automatically push to remote repositories

You stay in control.

---

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

See the [GOALS directory](https://github.com/sandgardenhq/sgai/tree/main/GOALS) for examples.

---

## Questions?

**Found a bug or have a feature request?** [Open an issue →](https://github.com/sandgardenhq/sgai/issues)

**Want to discuss ideas or share what you built?** [Start a discussion →](https://github.com/sandgardenhq/sgai/discussions)

---

## Development

Developer documentation lives in `docs/`, produced by [Doc Holiday](https://doc.holiday), of course!

---

## License

[https://github.com/sandgardenhq/sgai/blob/main/LICENSE](https://github.com/sandgardenhq/sgai/blob/main/LICENSE)

---

## About

Sgai was created by [Ulderico Cirello](https://cirello.org/), and is maintained by [Sandgarden](https://www.sandgarden.com/).
