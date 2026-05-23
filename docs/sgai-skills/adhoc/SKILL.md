---
name: adhoc
description: Run and manage ad-hoc AI prompts in sgai workspaces without starting a full agentic session. Use when you need to run a one-off AI prompt against a workspace, check the status of a running ad-hoc prompt, or stop a running ad-hoc prompt.
compatibility: Requires a running sgai server and opencode installed. The prompt runs as a non-interactive opencode command in the workspace directory.
---

# Ad-hoc Prompts

Ad-hoc prompts let you run a single AI prompt against a workspace without starting a full agentic session. Useful for quick tasks, code reviews, or one-off questions.

## Start an Ad-hoc Prompt

**Endpoint:** `POST /api/v1/workspaces/{name}/adhoc`

```bash
curl -X POST $BASE_URL/api/v1/workspaces/my-project/adhoc \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Review the authentication code and identify any security issues",
    "model": "openai/gpt-5.5 (low)"
  }'
```

Request:
```json
{
  "prompt": "Review the authentication code and identify any security issues",
  "model": "openai/gpt-5.5 (low)"
}
```

Response:
```json
{
  "running": true,
  "output": "",
  "message": "ad-hoc prompt started"
}
```

If already running:
```json
{
  "running": true,
  "output": "$ opencode run -m openai/gpt-5.5 --agent build --title adhoc [openai/gpt-5.5 (low)] --variant low...\nprompt: Review the...",
  "message": "ad-hoc prompt already running"
}
```

### Model Format

The model parameter accepts the same format as GOAL.md models:
- `"openai/gpt-5.5 (low)"` — cost-conscious GPT-5.5 baseline
- `"openai/gpt-5.5 (xhigh)"` — high-reasoning GPT-5.5 variant
- Use `GET /api/v1/models` to list available models

### How Ad-hoc Works

Ad-hoc prompts parse `provider/model (variant)` into `opencode run -m provider/model --agent build --title "adhoc [provider/model (variant)]" --variant variant` with your prompt piped as stdin. Without a parenthesized variant, SGAI omits `--variant`. The workspace directory is used as the working directory.

## Get Ad-hoc Status

Check if an ad-hoc prompt is running and get its current output.

**Endpoint:** `GET /api/v1/workspaces/{name}/adhoc`

```bash
curl -s $BASE_URL/api/v1/workspaces/my-project/adhoc
```

Response (running):
```json
{
  "running": true,
  "output": "$ opencode run -m openai/gpt-5.5 --agent build --title adhoc [openai/gpt-5.5 (low)] --variant low\nprompt: Review the authentication code...\n\nAnalyzing the codebase...\n\nFound 3 potential issues:\n1. JWT tokens lack expiration...",
  "message": "adhoc status"
}
```

Response (not running):
```json
{
  "running": false,
  "output": "$ opencode run...\n...\n[completed successfully]",
  "message": "adhoc status"
}
```

### Polling for Completion

```bash
WORKSPACE="my-project"

# Start the prompt
curl -X POST $BASE_URL/api/v1/workspaces/$WORKSPACE/adhoc \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Summarize the GOAL.md", "model": "openai/gpt-5.5 (low)"}'

# Poll until done
while true; do
  STATUS=$(curl -s $BASE_URL/api/v1/workspaces/$WORKSPACE/adhoc)
  RUNNING=$(echo $STATUS | jq '.running')

  if [ "$RUNNING" = "false" ]; then
    echo "Ad-hoc complete!"
    echo $STATUS | jq -r '.output'
    break
  fi

  echo "Still running..."
  sleep 3
done
```

## Stop an Ad-hoc Prompt

Stop a running ad-hoc prompt.

**Endpoint:** `DELETE /api/v1/workspaces/{name}/adhoc`

```bash
curl -X DELETE $BASE_URL/api/v1/workspaces/my-project/adhoc
```

Response:
```json
{
  "running": false,
  "output": "$ opencode run...\n...\n[process terminated]",
  "message": "ad-hoc stopped"
}
```

## Ad-hoc vs Full Session

| Feature | Ad-hoc | Full Session |
|---------|--------|--------------|
| Single prompt | ✓ | ✗ (multi-agent flow) |
| Multi-agent workflow | ✗ | ✓ |
| Human interaction | ✗ | ✓ |
| Progress tracking | Basic | Full (todos, events) |
| Cost tracking | ✗ | ✓ |
| Concurrent with session | No | N/A |
| GOAL.md required | ✗ | ✓ |

## Common Ad-hoc Use Cases

```bash
# Code review
curl -X POST $BASE_URL/api/v1/workspaces/my-project/adhoc \
  -d '{"prompt": "Review all Go files for potential race conditions", "model": "openai/gpt-5.5 (low)"}'

# Documentation generation
curl -X POST $BASE_URL/api/v1/workspaces/my-project/adhoc \
  -d '{"prompt": "Generate a README.md for this project", "model": "openai/gpt-5.5 (low)"}'

# Quick fix
curl -X POST $BASE_URL/api/v1/workspaces/my-project/adhoc \
  -d '{"prompt": "Fix the failing test in auth_test.go", "model": "openai/gpt-5.5 (low)"}'

# Analysis
curl -X POST $BASE_URL/api/v1/workspaces/my-project/adhoc \
  -d '{"prompt": "List all TODO comments in the codebase", "model": "openai/gpt-5.5 (low)"}'
```
