# Sgai (pronounced “Sky”)

![Sgai Dashboard](https://github.com/sandgardenhq/sgai/blob/main/assets/screenshots/08-Workspace.png?raw=true)

Define your goal. Launch the dashboard.
Watch AI agents plan, execute, and validate your software — with you in control.

**📺 [Watch the 4-minute demo →](https://youtu.be/NYmjhwLUg8Q)**

---

## What Is Sgai?

Sgai turns software development into a **goal-driven, multi-agent workflow**.

Instead of prompting step-by-step, you:

1. Define the outcome.
2. Agents decompose it into a directed acyclic graph (DAG) of work.
3. You supervise execution and answer questions.
4. Completion gates (tests, linting, etc.) determine success.

Not autocomplete. Not a chat window.
A local AI software factory.

---

## Why Try It?

* See AI work as a visible execution graph — not hidden reasoning
* Coordinate multiple agent roles (developer, reviewer, coordinator)
* Keep humans in the loop
* Enforce correctness before marking work complete
* Run entirely inside your local repository

---

## Quick Start

### Recommended: Automated Setup via opencode

```bash
opencode update
opencode auth login
opencode --model anthropic/claude-opus-4-6 run "install Sgai using the instructions from https://github.com/sandgardenhq/sgai/blob/main/INSTALLATION.md"
```

This runs the official installation guide automatically and launches a demo workspace.

---

### Manual Installation

**Required:** Go, Node.js, bun, opencode
**Recommended:** jj, tmux, ripgrep, Graphviz

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

---

### 2. Agents Plan the Work

![Choose a Template](https://github.com/sandgardenhq/sgai/blob/main/assets/screenshots/02-ChooseATemplate.png?raw=true)

Sgai decomposes your goal into a DAG of coordinated agents with defined roles.

Dependencies are explicit. Execution is visible.

---

### 3. Supervise Execution

![Agent Questions](https://github.com/sandgardenhq/sgai/blob/main/assets/screenshots/09-Questions.png?raw=true)

Agents pause when they need clarification — making assumptions explicit instead of hidden.

You can:

* Watch real-time progress
* Answer agent questions
* Start, stop, or fork sessions
* Review diffs before accepting changes

Completion gates determine when work is actually done.

---

## What Happens to Your Code?

* Agents operate inside your local repository
* Changes go through your VCS (jj recommended)
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

## Discussions

Active discussions:
[https://github.com/sandgardenhq/sgai/discussions](https://github.com/sandgardenhq/sgai/discussions)

---

## Development

Developer documentation lives in `docs/`.

---

## Project Status

Sgai is actively evolving. Expect iteration and breaking changes.

Feedback and issues are welcome.

---

## License

[https://github.com/sandgardenhq/sgai/blob/main/LICENSE](https://github.com/sandgardenhq/sgai/blob/main/LICENSE)
