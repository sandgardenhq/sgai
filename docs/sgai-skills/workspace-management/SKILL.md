---
name: workspace-management
description: Create, fork, delete, and rename sgai workspaces via the HTTP API. Use when you need to set up new project workspaces, create parallel forks for concurrent development, clean up finished forks, or rename existing fork workspaces.
compatibility: Requires a running sgai server. Workspace names must use lowercase letters, numbers, and dashes only.
---

# Workspace Management

Workspaces are directories managed by sgai. There are three kinds:
- **Standalone** — independent workspace, not part of a fork tree
- **Root** — has one or more fork children (displayed in dashboard/fork mode)
- **Fork** — child of a root workspace, shares the jj VCS repository

## Create a Workspace

**Endpoint:** `POST /api/v1/workspaces`

```bash
curl -X POST $BASE_URL/api/v1/workspaces \
  -H "Content-Type: application/json" \
  -d '{"name": "my-project"}'
```

Request:
```json
{"name": "my-project"}
```

Response (201 Created):
```json
{
  "name": "my-project",
  "dir": "/path/to/workspaces/my-project"
}
```

Errors:
- `400` — invalid name (must be lowercase letters, numbers, dashes)
- `409` — directory already exists

### Name Validation Rules
- Only lowercase letters (a-z), numbers (0-9), and dashes (-)
- Cannot start or end with a dash
- No spaces or special characters

## Fork a Workspace

Create a jj workspace fork for parallel development.

**Endpoint:** `POST /api/v1/workspaces/{name}/fork`

```bash
curl -X POST $BASE_URL/api/v1/workspaces/my-project/fork \
  -H "Content-Type: application/json" \
  -d '{"name": "feature-branch"}'
```

Request:
```json
{"name": "feature-branch"}
```

Response (201 Created):
```json
{
  "name": "feature-branch",
  "dir": "/path/to/workspaces/feature-branch",
  "parent": "my-project",
  "createdAt": ""
}
```

Notes:
- Only standalone or root workspaces can be forked (forks cannot fork)
- Fork names are normalized (spaces become dashes, etc.)
- The fork shares the jj repository with the root via `jj workspace add`

## Delete a Fork

**Endpoint:** `POST /api/v1/workspaces/{name}/delete-fork`

Where `{name}` is the **root** workspace name.

```bash
curl -X POST $BASE_URL/api/v1/workspaces/my-project/delete-fork \
  -H "Content-Type: application/json" \
  -d '{"forkDir": "/full/path/to/fork", "confirm": true}'
```

Request:
```json
{
  "forkDir": "/full/path/to/workspaces/feature-branch",
  "confirm": true
}
```

Response:
```json
{
  "deleted": true,
  "message": "fork deleted successfully"
}
```

Notes:
- `confirm: true` is required (safety guard)
- The running session is stopped before deletion
- Uses `jj workspace forget` then removes the directory
- When a root workspace runs out of forks, it reverts from Fork Mode to Repository Mode

## Rename a Fork

Only fork workspaces can be renamed (not standalone or root).

**Endpoint:** `POST /api/v1/workspaces/{name}/rename`

```bash
curl -X POST $BASE_URL/api/v1/workspaces/feature-branch/rename \
  -H "Content-Type: application/json" \
  -d '{"name": "new-feature-name"}'
```

Request:
```json
{"name": "new-feature-name"}
```

Response:
```json
{
  "name": "new-feature-name",
  "oldName": "feature-branch",
  "dir": "/path/to/workspaces/new-feature-name"
}
```

Errors:
- `400` — workspace is not a fork
- `409` — session is running (stop first) or name conflict

## Toggle Pin

Pin workspaces to keep them prioritized at the top of the list.

**Endpoint:** `POST /api/v1/workspaces/{name}/pin`

```bash
curl -X POST $BASE_URL/api/v1/workspaces/my-project/pin
```

Response:
```json
{
  "pinned": true,
  "message": "pin toggled"
}
```

## Update Workspace Summary

Set a human-readable summary for a workspace (shown in the UI).

**Endpoint:** `PUT /api/v1/workspaces/{name}/summary`

```bash
curl -X PUT $BASE_URL/api/v1/workspaces/my-project/summary \
  -H "Content-Type: application/json" \
  -d '{"summary": "Implementing authentication module"}'
```

Response:
```json
{
  "updated": true,
  "summary": "Implementing authentication module",
  "workspace": "my-project"
}
```

## Update Commit Description

Update the jj commit description for the current working copy.

**Endpoint:** `POST /api/v1/workspaces/{name}/description`

```bash
curl -X POST $BASE_URL/api/v1/workspaces/my-project/description \
  -H "Content-Type: application/json" \
  -d '{"description": "feat: add authentication endpoints"}'
```

Response:
```json
{
  "updated": true,
  "description": "feat: add authentication endpoints"
}
```

## Get GOAL.md

**Endpoint:** `GET /api/v1/workspaces/{name}/goal`

```bash
curl -s $BASE_URL/api/v1/workspaces/my-project/goal
```

Response:
```json
{
  "content": "---\nflow: ...\n---\n\n- [ ] Task 1\n"
}
```

## Update GOAL.md

**Endpoint:** `PUT /api/v1/workspaces/{name}/goal`

```bash
curl -X PUT $BASE_URL/api/v1/workspaces/my-project/goal \
  -H "Content-Type: application/json" \
  -d '{"content": "- [ ] Build the auth system\n- [ ] Write tests\n"}'
```

Response:
```json
{
  "updated": true,
  "workspace": "my-project"
}
```

## Get Workspace Diff

Get the current jj diff for a workspace.

**Endpoint:** `GET /api/v1/workspaces/{name}/diff`

```bash
curl -s $BASE_URL/api/v1/workspaces/my-project/diff
```

Response:
```json
{
  "diff": "diff --git a/file.go b/file.go\n..."
}
```
