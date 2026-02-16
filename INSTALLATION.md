# SGAI Installation Instructions

Execute the following steps in order. Verify each checkpoint before proceeding.

## Automated Setup with opencode

Run this command to have opencode follow this guide end-to-end:

```sh
opencode run "install SGAI using the instructions from https://github.com/sandgardenhq/sgai/blob/main/INSTALLATION.md"
```

See [README.md](./README.md) for a short overview of what this automated flow does.

## Step 1: Check and Install Dependencies

The following tools are required:

- `go` (Go 1.25+)
- `node` (Node.js)
- `bun`
- `opencode`
- `jj` (Jujutsu)
- `dot` (Graphviz)
- `gh` (GitHub CLI) — optional, for PR creation
- `tmux`
- `rg` (ripgrep)

Check which of these commands are already available on PATH. For any that are missing, install them using whatever package manager or method is appropriate for the current platform.

### ✅ Checkpoint

Verify ALL of the following commands are available on PATH:

```sh
go version
node --version
bun --version
opencode --version
jj --version
dot -V
gh --version
tmux -V
rg --version
```

If any command fails, troubleshoot before proceeding. Common fixes:
- Ensure `~/.bun/bin` is in PATH (for bun).
- Restart the shell or source the profile if needed.

---

## Step 2: Build and Install SGAI from Source

The SGAI binary embeds the React web dashboard at build time. You must build from source so the webapp is compiled into the binary.

```sh
git clone https://github.com/sandgardenhq/sgai.git
cd sgai
make build
cp ./bin/sgai "$(go env GOPATH)/bin/sgai"
```

This runs the full build pipeline: frontend build (`bun install` + `bun run build`) → Go lint → Go binary.

### ✅ Checkpoint

Verify `sgai` is available:

```sh
sgai --help
```

If not found, ensure `$(go env GOPATH)/bin` is in your PATH:

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
sgai --help
```

---

## Step 3: Create Workspace and Demo Project

Create the following directory structure relative to the current working directory:

```
./SGAI/
  TodoApp/
    GOAL.md
    .sgai/
  INSTALLATIONS_NOTES.md
```

**Create directories:**

```sh
mkdir -p ./SGAI/TodoApp/.sgai
```

**Create `./SGAI/TodoApp/GOAL.md`** with this content:

```markdown
---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
models:
  "coordinator": "opencode/big-pickle"
  "backend-go-developer": "opencode/big-pickle"
  "go-readability-reviewer": "opencode/big-pickle"
  "general-purpose": "opencode/big-pickle"
  "stpa-analyst": "opencode/big-pickle"
interactive: yes
---

# Todo App

Build a simple command-line todo application.

## Requirements

- Add, list, complete, and delete todos
- Persist todos to a local file
- Simple and clean CLI interface

## Tasks

- [ ] Create todo data model
- [ ] Implement add/list/complete/delete commands
- [ ] Add file-based persistence
- [ ] Add basic error handling
```

**Create `./SGAI/INSTALLATIONS_NOTES.md`** — write a file with these sections:

1. A title: "SGAI Installation Notes"
2. A blockquote constraint: "when INSTALLATIONS_NOTES.md is present, read that file to understand how to start sgai"
3. A "How to Start SGAI" section explaining to navigate to a project directory containing `GOAL.md` and `.sgai/`, then run `sgai serve`. Include that the dashboard is at http://localhost:8080.
4. A "How to Stop SGAI" section explaining how to find and kill the process (`ps aux | grep 'sgai serve'` then `kill <PID>`).
5. A "How to Restart SGAI" section with: `cd SGAI/TodoApp && sgai serve`
6. A "Troubleshooting" section noting that if `sgai` is not found, add Go binaries to PATH: `export PATH="$PATH:$(go env GOPATH)/bin"`

### ✅ Checkpoint

Verify the directory structure:

```sh
ls -la ./SGAI/
ls -la ./SGAI/TodoApp/
ls -la ./SGAI/TodoApp/.sgai/
cat ./SGAI/TodoApp/GOAL.md
cat ./SGAI/INSTALLATIONS_NOTES.md
```

---

## Step 4: Start SGAI

Start `sgai serve` in the TodoApp directory using a method that survives opencode exiting.

**Determine the best persistence method for the current platform and start sgai:**

- Use `tmux` if available (preferred):
  ```sh
  tmux new-session -d -s sgai -c ./SGAI/TodoApp 'sgai serve'
  ```
- Use `nohup` as a portable fallback:
  ```sh
  cd ./SGAI/TodoApp && nohup sgai serve > sgai.log 2>&1 &
  ```
- If `screen` is available:
  ```sh
  screen -dmS sgai bash -c 'cd ./SGAI/TodoApp && sgai serve'
  ```

Use whichever method is available, preferring `tmux` > `nohup` > `screen`.

### ✅ Checkpoint

Verify sgai is running:

```sh
# Check if the process is running
ps aux | grep 'sgai serve'

# Try reaching the dashboard
curl -s http://localhost:8080 > /dev/null && echo "SGAI is running" || echo "SGAI is not responding"
```

Update `./SGAI/INSTALLATIONS_NOTES.md` to reflect the actual persistence method used, so future sessions know how to restart sgai.

---

## Done

SGAI is installed and running. Open [http://localhost:8080](http://localhost:8080) in your browser to access the web dashboard and start working on the TodoApp project.


---

# Appendix

## Download sources

https://go.dev/dl
https://nodejs.org/en/download
https://bun.sh/
https://opencode.ai/docs/#install
https://github.com/jj-vcs/jj/releases
https://graphviz.org/download/ (for dot)
https://github.com/cli/cli#installation (for gh)
https://github.com/tmux/tmux/wiki
https://github.com/BurntSushi/ripgrep/releases
