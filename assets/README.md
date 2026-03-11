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

**Agent Aliases:**

You can create agent aliases that reuse an existing agent's prompt and tools with a different model. This lets you run the same agent role at different cost/capability tiers:

```markdown
---
flow: |
  "backend-go-developer-lite" -> "go-readability-reviewer"
alias:
  "backend-go-developer-lite": "backend-go-developer"
models:
  "backend-go-developer-lite": "anthropic/claude-haiku-4-5"
---
```

An aliased agent inherits everything from its base agent (prompt, tools, snippets) but uses its own model configuration. Aliased agents appear like any other agent in the workflow.

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

## What Happens to Your Code?

* Agents operate inside your local repository
* Changes go through your version control (we recommend jj, but Git works)
* Sgai does not automatically push to remote repositories

You stay in control.

---

## Contributing

Sgai accepts improvements as specifications inside `GOALS/`.

1. Create `GOALS/YYYY_MM_DD_feature_name.md`
2. Describe desired behavior and success criteria
3. Submit a PR

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
