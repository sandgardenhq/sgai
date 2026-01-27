---
name: Run Long Running Processes in Tmux
description: Guide for using tmux to manage detached sessions for long-running processes, including lifecycle management, cleanup, startup, verification, and output capture
when_to_use: When you need to run servers or long-running commands in the background, send commands to them, capture output, handle TTY requirements, or manage session lifecycle with error handling
version: 2.0.0
languages: bash
---

# Run Long Running Processes in Tmux

## Overview

Tmux allows running processes in detached sessions, providing pseudo-terminals for processes requiring TTY, and tools to interact with them without attaching. Enhanced with complete session lifecycle management including cleanup, startup verification, and output capture.

## When to Use

Use when:
- Running servers or daemons in background
- Processes need TTY but you want detached execution
- Need to send commands or signals to running processes
- Capturing output from detached sessions
- Managing long-running tasks in automation scripts
- Testing server processes or managing background services
- Need session cleanup and conflict resolution

## Core Pattern

1. Cleanup existing sessions: tmux kill-session -t name 2>/dev/null || true
2. Start detached session: tmux new-session -d -s name "command"
3. Verify startup: tmux capture-pane -t name -S - -p | grep -q "expected_output"
4. Interact: tmux send-keys -t name "command" C-m
5. Capture: tmux capture-pane -t name -S - -p > file
6. Monitor: tmux attach -t name (when needed)

## Quick Reference

| Command | Purpose | Example |
|---------|---------|---------|
| kill-session -t name 2>/dev/null || true | Cleanup existing | tmux kill-session -t server 2>/dev/null || true |
| new-session -d -s name "cmd" | Start detached | tmux new-session -d -s server "python -m http.server" |
| capture-pane -t name -S - -p | Verify startup | tmux capture-pane -t server -S - -p | grep -q "Serving" |
| send-keys -t name "cmd" C-m | Send command | tmux send-keys -t server "echo test" C-m |
| capture-pane -t name -S - -p > file | Capture output | tmux capture-pane -t server -S - -p > log.txt |
| attach -t name | Reattach | tmux attach -t server |
| kill-session -t name | Stop session | tmux kill-session -t server |

## Implementation

### Session Lifecycle Management

```bash
# Complete lifecycle with error handling
SESSION_NAME="myserver"
COMMAND="python -m http.server 8000"

# 1. Cleanup existing session
tmux kill-session -t "$SESSION_NAME" 2>/dev/null || true

# 2. Start new session
tmux new-session -d -s "$SESSION_NAME" "$COMMAND"

# 3. Wait for startup and verify
sleep 2
if ! tmux capture-pane -t "$SESSION_NAME" -S - -p | grep -q "Serving HTTP"; then
    echo "Server failed to start properly"
    tmux kill-session -t "$SESSION_NAME"
    exit 1
fi

# 4. Capture initial output
tmux capture-pane -t "$SESSION_NAME" -S - -p > startup.log

# 5. Session is ready for interaction
```

### Starting Detached Sessions

```bash
# Basic detached session
tmux new-session -d -s mysession "my_long_running_command"

# With working directory and logging
tmux new-session -d -s appsession -c /path/to/dir "./run_app.sh >app.log 2>&1"

# With session conflict handling
if tmux has-session -t appsession 2>/dev/null; then
    echo "Session already exists, cleaning up..."
    tmux kill-session -t appsession
fi
tmux new-session -d -s appsession "./run_app.sh"
```

For processes requiring TTY, tmux provides PTYs automatically.

### Session Verification

```bash
# Check if session exists
if tmux has-session -t mysession 2>/dev/null; then
    echo "Session is running"
else
    echo "Session not found"
fi

# Verify process output
tmux capture-pane -t mysession -S - -p | grep -q "expected_pattern"
echo $?

# Check session status
tmux display-message -t mysession -p "#{session_name}: #{window_status}"
```

### Sending Commands

```bash
# Send command with enter
tmux send-keys -t mysess:0.0 "cd /path/to/dir" C-m
tmux send-keys -t mysess:0.0 "./run_my_task.sh --option" C-m

# Send with delay for process readiness
tmux send-keys -t server "status" C-m
sleep 1
tmux capture-pane -t server -S - -p | tail -5
```

Note: Process must accept stdin as commands. May need delays for shell readiness.

### Capturing Output

```bash
# Capture full history
tmux capture-pane -t mysess:0.0 -S - -p > output.txt

# Capture with verification
OUTPUT=$(tmux capture-pane -t mysess:0.0 -S - -p)
if echo "$OUTPUT" | grep -q "ERROR"; then
    echo "Error detected in output"
fi

# Alternative with buffer
tmux capture-pane -t mysess:0.0 -S -
tmux save-buffer -b 0 output.txt

# Real-time monitoring
watch -n 5 'tmux capture-pane -t server -S - -p | tail -10'
```

Text only, may lose colors unless -e flag used.

## Common Mistakes

- Forgetting -d flag (attaches instead of detaching)
- Wrong pane syntax (session:window.pane)
- Assuming processes can receive commands if not shell-based
- Not handling TTY requirements (tmux usually covers)
- Losing sessions on reboot (use supervisors)
- Not cleaning up existing sessions before starting new ones
- Forgetting to verify session startup before proceeding
- Not handling session conflicts in automation scripts

## Real-World Impact

Enables background execution of servers, automated testing, and remote process management without keeping terminals open. Enhanced lifecycle management reduces repetitive session management tasks and provides reliable server testing workflows with proper error handling and cleanup.