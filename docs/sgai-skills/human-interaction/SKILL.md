---
name: human-interaction
description: Handle agent questions and work gates in sgai workspaces. Use when an agent is blocked waiting for human input, when you need to respond to multi-choice questions, approve work gates, or provide free-text answers to agent queries.
compatibility: Requires a running sgai session with a pending question. Check workspace state for pendingQuestion field before calling respond endpoint.
---

# Human Interaction

When sgai agents need human input, they set `needsInput: true` and populate `pendingQuestion` in the workspace state. Your harness must detect and respond to these to unblock the agent.

## Detecting Pending Questions

Poll `/api/v1/state` and check each workspace:

```bash
STATE=$(curl -s $BASE_URL/api/v1/state)

# Check all workspaces for pending questions
echo $STATE | jq '.workspaces[] | select(.needsInput == true) | {name, pendingQuestion}'
```

A workspace with a pending question looks like:

```json
{
  "name": "my-project",
  "needsInput": true,
  "pendingQuestion": {
    "questionId": "abc123def456ef78",
    "type": "free-text",
    "agentName": "coordinator",
    "message": "Which database should we use for the project?",
    "questions": []
  }
}
```

## Question Types

### `free-text`

Agent asks an open-ended question. Respond with `answer`.

```json
{
  "pendingQuestion": {
    "questionId": "abc123def456ef78",
    "type": "free-text",
    "agentName": "coordinator",
    "message": "What is the primary use case for this application?",
    "questions": []
  }
}
```

Response:
```bash
curl -X POST $BASE_URL/api/v1/workspaces/my-project/respond \
  -H "Content-Type: application/json" \
  -d '{
    "questionId": "abc123def456ef78",
    "answer": "This is a B2B SaaS platform for small businesses"
  }'
```

### `multi-choice`

Agent presents structured questions with predefined choices.

```json
{
  "pendingQuestion": {
    "questionId": "def456abc789ab12",
    "type": "multi-choice",
    "agentName": "coordinator",
    "message": "Please answer the following questions:",
    "questions": [
      {
        "question": "Which backend language?",
        "choices": ["Go", "Python", "Node.js", "Rust"],
        "multiSelect": false
      },
      {
        "question": "Which features are required?",
        "choices": ["Auth", "Payments", "Analytics", "Notifications"],
        "multiSelect": true
      }
    ]
  }
}
```

Response (single select one choice, multi-select multiple):
```bash
curl -X POST $BASE_URL/api/v1/workspaces/my-project/respond \
  -H "Content-Type: application/json" \
  -d '{
    "questionId": "def456abc789ab12",
    "selectedChoices": ["Go", "Auth", "Analytics"],
    "answer": "Also add OAuth2 integration"
  }'
```

### `work-gate`

A decision point requiring explicit approval to proceed. The agent stops until approved.

```json
{
  "pendingQuestion": {
    "questionId": "ghi789xyz123cd45",
    "type": "work-gate",
    "agentName": "coordinator",
    "message": "Ready to begin implementation. Please review the plan and approve.",
    "questions": [
      {
        "question": "Review complete?",
        "choices": ["Approve and proceed", "Request changes", "Cancel"],
        "multiSelect": false
      }
    ]
  }
}
```

To approve (select the approval choice):
```bash
curl -X POST $BASE_URL/api/v1/workspaces/my-project/respond \
  -H "Content-Type: application/json" \
  -d '{
    "questionId": "ghi789xyz123cd45",
    "selectedChoices": ["Approve and proceed"]
  }'
```

## Respond Endpoint

**Endpoint:** `POST /api/v1/workspaces/{name}/respond`

```bash
curl -X POST $BASE_URL/api/v1/workspaces/{name}/respond \
  -H "Content-Type: application/json" \
  -d '{
    "questionId": "QUESTION_ID_FROM_STATE",
    "answer": "optional free text",
    "selectedChoices": ["optional", "choice", "selections"]
  }'
```

Request fields:
| Field | Required | Description |
|-------|----------|-------------|
| `questionId` | Yes | Must match the current `pendingQuestion.questionId` |
| `answer` | No | Free-text answer (used for free-text and as additional context for multi-choice) |
| `selectedChoices` | No | Array of selected choice strings for multi-choice/work-gate |

Response:
```json
{
  "success": true,
  "message": "response submitted"
}
```

Errors:
- `409 Conflict` — no pending question, or question expired (stale questionId)
- `400 Bad Request` — empty response (must provide answer or choices)

## Question ID Handling

The `questionId` is a SHA256 hash of the question content. It changes when the question changes. Always:
1. Fetch fresh state before responding
2. Use the `questionId` from the current state
3. If you get a `409 "question expired"` error, re-fetch state and get the new ID

## Delete a Message

Remove a message from a workspace's message queue.

**Endpoint:** `DELETE /api/v1/workspaces/{name}/messages/{id}`

```bash
curl -X DELETE $BASE_URL/api/v1/workspaces/my-project/messages/42
```

Response:
```json
{
  "deleted": true,
  "id": 42,
  "message": "message deleted successfully"
}
```

## Complete Interaction Loop Example

```bash
#!/bin/bash
BASE_URL="http://127.0.0.1:PORT"
WORKSPACE="my-project"

while true; do
  STATE=$(curl -s $BASE_URL/api/v1/state)
  WS=$(echo $STATE | jq --arg name "$WORKSPACE" '.workspaces[] | select(.name == $name)')
  
  NEEDS_INPUT=$(echo $WS | jq '.needsInput')
  RUNNING=$(echo $WS | jq '.running')
  
  if [ "$NEEDS_INPUT" = "true" ]; then
    QUESTION_ID=$(echo $WS | jq -r '.pendingQuestion.questionId')
    TYPE=$(echo $WS | jq -r '.pendingQuestion.type')
    MESSAGE=$(echo $WS | jq -r '.pendingQuestion.message')
    
    echo "Question ($TYPE): $MESSAGE"
    
    # Your harness logic to determine the answer...
    ANSWER="My response to this question"
    
    curl -X POST $BASE_URL/api/v1/workspaces/$WORKSPACE/respond \
      -H "Content-Type: application/json" \
      -d "{\"questionId\": \"$QUESTION_ID\", \"answer\": \"$ANSWER\"}"
  fi
  
  sleep 5
done
```
