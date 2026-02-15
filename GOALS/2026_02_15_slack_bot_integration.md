---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "react-developer" -> "react-reviewer"
  "react-reviewer" -> "stpa-analyst"
  "project-critic-council"
  "skill-writer"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
  "react-developer": "anthropic/claude-opus-4-6"
  "react-reviewer": "anthropic/claude-opus-4-6"
  "stpa-analyst": "anthropic/claude-opus-4-6"
  "project-critic-council": ["anthropic/claude-opus-4-6"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---

- [x] Add a way to plug the Factory as a Slack bot (`sgai slack-bot` subcommand).
  - [x] **Slack Socket Mode Client** (Go infrastructure)
    - [x] Connect to Slack via Socket Mode (WebSocket, no public URL needed)
    - [x] Enforce allowlist of permitted Slack user IDs (deny by default, silent ignore for unauthorized)
    - [x] Configuration via env vars: `SLACK_BOT_TOKEN`, `SLACK_APP_TOKEN`, `SLACK_ALLOWED_USERS`
    - [x] Global per-session lock (`sync.Mutex` map) to prevent concurrent `opencode run --session` calls
  - [x] **OpenCode Frontdesk Agent** (`slack-frontdesk` agent prompt)
    - [x] Purpose-specific AI agent served via OpenCode headless mode (`opencode run --session`)
    - [x] Persistent session per Slack thread (session ID tracked in JSON database)
    - [x] Slack MCP tools: `slack_reply`, `slack_reply_blocks` (agent writes to Go channel, Go posts to Slack)
    - [x] SGAI Runtime MCP tools: `sgai_list_workspaces`, `sgai_workspace_status`, `sgai_start_workspace`, `sgai_stop_workspace`, `sgai_respond_to_question`, `sgai_read_events`, `sgai_read_messages`, `sgai_connect_thread`, `sgai_disconnect_thread`, `sgai_toggle_event_updates`
  - [x] This agent allows me to do all the things that the WebUI allows me to do
    - [x] Including sending me messages regarding interview questions (proactive notification on `waiting-for-human`)
    - [x] Including sending me messages regarding clearing the work-gate
    - [x] Including sending me messages when the workspace starts and concludes
    - [x] Including allowing me to toggle receiving event updates (Progress tab, events box content) as the factories work
  - [x] **Proactive Notification Watcher** (Go code, no AI)
    - [x] Poll workspace state.json every 2-3s for connected workspaces
    - [x] Detect and notify on: `waiting-for-human`, `complete`, workspace start/stop transitions
    - [x] Stream progress entries to threads with event updates enabled
  - [x] **Thread-Workspace Mapping**
    - [x] Each workspace can be plugged to a Slack thread via @mention conversation
    - [x] Session database at `~/.config/sgai/slack-sessions.json` (keyed by `channelID:threadTS`)
    - [x] When Slack gets too busy, re-plug a workspace into another thread and resume from there
  - [x] **Tests**
    - [x] Unit tests for Go infrastructure (Socket Mode client, session database, lock map, notification watcher)
    - [x] Unit tests for bot infrastructure (config parsing, message splitting, allowlist, lock serialization, session persistence, watcher detection)
    - [x] Edge cases: concurrent messages (lock), re-plugging, unauthorized deny, session recovery
    - [x] `make test` passes
