# SGAI Installation Instructions

Execute the following steps in order. Verify each checkpoint before proceeding.

## Step 0: Platform & Permissions Pre-flight

Detect your operating system and verify that automated installation can proceed.

```sh
OS=$(uname -s)
echo "Detected OS: $OS"
```

### Linux: Sudo Check

On Linux, package installation requires `sudo`. If `sudo` is not available or requires a password, the automated installation **cannot proceed**. Run the following check:

```sh
if [ "$OS" = "Linux" ]; then
  if ! command -v sudo &>/dev/null; then
    echo "=============================================="
    echo "STOP: sudo is not installed."
    echo "Cannot proceed with automated installation."
    echo "=============================================="
    echo ""
    echo "Please install sudo manually as root, then re-run this installation."
    exit 1
  elif ! sudo -n true 2>/dev/null; then
    echo "=============================================="
    echo "STOP: sudo requires a password."
    echo "Cannot proceed with automated installation."
    echo "=============================================="
    echo ""
    echo "You must install the required packages manually."
    echo "Detecting your package manager..."
    echo ""

    if command -v apt &>/dev/null; then
      echo "Detected: apt (Debian/Ubuntu)"
      echo "Run the following commands:"
      echo ""
      echo "  sudo apt update"
      echo "  sudo apt install -y golang-go nodejs npm graphviz gh tmux ripgrep"
      echo "  # Install bun: curl -fsSL https://bun.sh/install | bash"
      echo "  # Install jj:  cargo install --locked jj-cli (or download from https://github.com/jj-vcs/jj/releases)"
    elif command -v dnf &>/dev/null; then
      echo "Detected: dnf (Fedora/RHEL)"
      echo "Run the following commands:"
      echo ""
      echo "  sudo dnf install -y golang nodejs npm graphviz gh tmux ripgrep"
      echo "  # Install bun: curl -fsSL https://bun.sh/install | bash"
      echo "  # Install jj:  cargo install --locked jj-cli (or download from https://github.com/jj-vcs/jj/releases)"
    elif command -v pacman &>/dev/null; then
      echo "Detected: pacman (Arch Linux)"
      echo "Run the following commands:"
      echo ""
      echo "  sudo pacman -S --noconfirm go nodejs npm graphviz github-cli tmux ripgrep"
      echo "  # Install bun: curl -fsSL https://bun.sh/install | bash"
      echo "  # Install jj:  cargo install --locked jj-cli (or download from https://github.com/jj-vcs/jj/releases)"
    elif command -v zypper &>/dev/null; then
      echo "Detected: zypper (openSUSE)"
      echo "Run the following commands:"
      echo ""
      echo "  sudo zypper install -y go nodejs npm graphviz gh tmux ripgrep"
      echo "  # Install bun: curl -fsSL https://bun.sh/install | bash"
      echo "  # Install jj:  cargo install --locked jj-cli (or download from https://github.com/jj-vcs/jj/releases)"
    elif command -v apk &>/dev/null; then
      echo "Detected: apk (Alpine)"
      echo "Run the following commands:"
      echo ""
      echo "  sudo apk add go nodejs npm graphviz github-cli tmux ripgrep"
      echo "  # Install bun: curl -fsSL https://bun.sh/install | bash"
      echo "  # Install jj:  cargo install --locked jj-cli (or download from https://github.com/jj-vcs/jj/releases)"
    else
      echo "Could not detect package manager. Install these packages manually:"
      echo "  go, node, bun, jj, graphviz (dot), gh, tmux, ripgrep"
    fi

    echo ""
    echo "After installing the packages above, re-run this installation."
    exit 1
  else
    echo "sudo is available and passwordless. Proceeding."
  fi
elif [ "$OS" = "Darwin" ]; then
  echo "macOS detected. Skipping sudo check (Homebrew does not require sudo)."
else
  echo "WARNING: Unrecognized OS '$OS'. Proceeding, but some steps may need manual adjustment."
fi
```

### Linux: Package Manager Detection

On Linux, detect the available package manager for use in subsequent steps:

```sh
if [ "$OS" = "Linux" ]; then
  if command -v apt &>/dev/null; then
    PM="apt"
  elif command -v dnf &>/dev/null; then
    PM="dnf"
  elif command -v pacman &>/dev/null; then
    PM="pacman"
  elif command -v zypper &>/dev/null; then
    PM="zypper"
  elif command -v apk &>/dev/null; then
    PM="apk"
  else
    PM=""
    echo "WARNING: No supported package manager found. You will need to install packages manually."
  fi
  echo "Detected package manager: ${PM:-none}"
fi
```

### ✅ Checkpoint

Verify the pre-flight check passed:

- On Linux: the script did **not** exit with "STOP". `sudo -n true` succeeded.
- On macOS: the script printed "macOS detected."

---

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

Check which of these commands are already available on PATH. For any that are missing, install them using the platform-specific instructions below.

### macOS (Homebrew)

```sh
brew install go node graphviz gh tmux ripgrep
# Install bun:
curl -fsSL https://bun.sh/install | bash
# Install jj:
brew install jj
# Install opencode:
# See https://opencode.ai/docs/#install
```

### Linux

Use the package manager detected in Step 0. The commands below install all required dependencies at once.

> **Note:** These commands require passwordless `sudo` (verified in Step 0). If Step 0 told you to install packages manually, follow those instructions instead.

**apt (Debian/Ubuntu):**
```sh
sudo apt update
sudo apt install -y golang-go nodejs npm graphviz gh tmux ripgrep
```

**dnf (Fedora/RHEL):**
```sh
sudo dnf install -y golang nodejs npm graphviz gh tmux ripgrep
```

**pacman (Arch Linux):**
```sh
sudo pacman -S --noconfirm go nodejs npm graphviz github-cli tmux ripgrep
```

**zypper (openSUSE):**
```sh
sudo zypper install -y go nodejs npm graphviz gh tmux ripgrep
```

**apk (Alpine):**
```sh
sudo apk add go nodejs npm graphviz github-cli tmux ripgrep
```

**All Linux distros — install bun, jj, and opencode separately:**

```sh
# Install bun
curl -fsSL https://bun.sh/install | bash
export PATH="$HOME/.bun/bin:$PATH"

# Install jj (Jujutsu) — download from releases or use cargo:
# Option A: Download binary from https://github.com/jj-vcs/jj/releases
# Option B: cargo install --locked jj-cli

# Install opencode
# See https://opencode.ai/docs/#install
```

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

> **Why `bash -c`?** The `git clone` and `cp` commands below access directories outside the current working directory. Tools like [opencode](https://opencode.ai/docs/permissions#external-directories) restrict external directory access by default. Wrapping these operations in `bash -c` (or `tmux`) avoids triggering permission prompts.

```sh
bash -c 'git clone https://github.com/sandgardenhq/sgai.git ./sgai-source && cd ./sgai-source && make build && cp ./bin/sgai "$(go env GOPATH)/bin/sgai"'
```

Alternatively, use tmux if you prefer a detached build:

```sh
tmux new-session -d -s build-sgai 'git clone https://github.com/sandgardenhq/sgai.git ./sgai-source && cd ./sgai-source && make build && cp ./bin/sgai "$(go env GOPATH)/bin/sgai"'
# Wait for the build to finish:
tmux wait-for build-sgai 2>/dev/null || sleep 120
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

Create the following directory structure relative to the current working directory. The `sgai` directory is the root from which `sgai serve` is started. Each subdirectory (like `TodoApp`) is a workspace that appears in the dashboard.

```
./sgai/
  TodoApp/
    GOAL.md
    .sgai/
  INSTALLATIONS_NOTES.md
```

**Create directories:**

```sh
mkdir -p ./sgai/TodoApp/.sgai
```

**Create `./sgai/TodoApp/GOAL.md`** with this content:

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

**Create `./sgai/INSTALLATIONS_NOTES.md`** — write a file with these sections:

1. A title: "SGAI Installation Notes"
2. A blockquote constraint: "when INSTALLATIONS_NOTES.md is present, read that file to understand how to start sgai"
3. A "How to Start SGAI" section explaining to navigate to the `sgai/` directory (the root that contains workspaces like `TodoApp/`) and run `sgai serve`. Workspaces inside `sgai/` will appear in the dashboard at http://localhost:8080.
4. A "How to Stop SGAI" section explaining how to find and kill the process (`ps aux | grep 'sgai serve'` then `kill <PID>`).
5. A "How to Restart SGAI" section with: `cd sgai && sgai serve`
6. A "Troubleshooting" section noting that if `sgai` is not found, add Go binaries to PATH: `export PATH="$PATH:$(go env GOPATH)/bin"`

### ✅ Checkpoint

Verify the directory structure:

```sh
ls -la ./sgai/
ls -la ./sgai/TodoApp/
ls -la ./sgai/TodoApp/.sgai/
cat ./sgai/TodoApp/GOAL.md
cat ./sgai/INSTALLATIONS_NOTES.md
```

---

## Step 4: Start SGAI

Start `sgai serve` from the `sgai/` directory (not from a workspace subdirectory). This is the root directory that contains workspaces — `sgai serve` discovers them automatically and shows them in the dashboard.

**Determine the best persistence method for the current platform and start sgai:**

- Try `nohup` first (most portable):
  ```sh
  cd ./sgai && nohup sgai serve > sgai.log 2>&1 &
  ```
- If `tmux` is available, prefer it:
  ```sh
  tmux new-session -d -s sgai -c ./sgai 'sgai serve'
  ```
- If `screen` is available:
  ```sh
  screen -dmS sgai bash -c 'cd ./sgai && sgai serve'
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

Update `./sgai/INSTALLATIONS_NOTES.md` to reflect the actual persistence method used, so future sessions know how to restart sgai.

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
